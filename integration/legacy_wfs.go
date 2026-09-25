package integration

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"path"
	"sort"
	"strings"

	aramcore "github.com/mirusu400/aram-core/core"
	shared "github.com/mirusu400/aram-core/runtime"
)

// Legacy WIPI emulator save backups use the same outer identity/checksum idea
// as ARAM, but carry a small table of db/* files instead of ARAM's gob-encoded
// StoragePersistenceState. ARAM must merge those files into the already-loaded
// handset storage: replacing the storage outright would discard title-created
// support files such as char.dat/map.dat that Inotia 1 needs to boot.
const (
	legacyWFSMagic       = "WFSAVEBK"
	legacyWFSVersion     = 1
	legacyWFSHeaderBytes = len(legacyWFSMagic) + 1 + 1 + saveBackupHashBytes + 4 + 4
	legacyWFSMaxBytes    = 256 << 20
	legacyWFSMaxEntries  = 16384
	legacyWFSMaxName     = 1024
)

type aramSaveDataEnvelope struct {
	Magic   string
	Storage shared.StoragePersistenceState
}

func isLegacyWFSSave(data []byte) bool {
	return len(data) >= len(legacyWFSMagic) &&
		string(data[:len(legacyWFSMagic)]) == legacyWFSMagic
}

// decodeLegacyWFS validates the legacy container and maps its db/* members to
// ARAM private-storage paths. db/.removed is metadata from the old emulator,
// not a guest file, and is intentionally ignored.
func decodeLegacyWFS(data []byte) (string, map[string][]byte, error) {
	if len(data) < legacyWFSHeaderBytes {
		return "", nil, errors.New("legacy WFS save: file is too small or truncated")
	}
	if len(data) > legacyWFSMaxBytes {
		return "", nil, errors.New("legacy WFS save: file exceeds size limit")
	}
	if !isLegacyWFSSave(data) {
		return "", nil, errors.New("legacy WFS save: wrong magic")
	}
	offset := len(legacyWFSMagic)
	if version := data[offset]; version != legacyWFSVersion {
		return "", nil, fmt.Errorf("legacy WFS save: unsupported version %d", version)
	}
	offset += 2 // version + reserved
	identity := append([]byte(nil), data[offset:offset+saveBackupHashBytes]...)
	offset += saveBackupHashBytes
	length := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	wantCRC := binary.LittleEndian.Uint32(data[offset : offset+4])
	offset += 4
	payload := data[offset:]
	if uint32(len(payload)) != length {
		return "", nil, fmt.Errorf(
			"legacy WFS save: payload is %d bytes, header declares %d",
			len(payload), length,
		)
	}
	if crc32.ChecksumIEEE(payload) != wantCRC {
		return "", nil, errors.New("legacy WFS save: file is corrupt (checksum mismatch)")
	}

	files := make(map[string][]byte)
	for cursor, entries := 0, 0; cursor < len(payload); entries++ {
		if entries >= legacyWFSMaxEntries {
			return "", nil, errors.New("legacy WFS save: too many file entries")
		}
		if len(payload)-cursor < 2 {
			return "", nil, errors.New("legacy WFS save: truncated filename length")
		}
		nameLength := int(binary.LittleEndian.Uint16(payload[cursor : cursor+2]))
		cursor += 2
		if nameLength == 0 || nameLength > legacyWFSMaxName || len(payload)-cursor < nameLength+4 {
			return "", nil, errors.New("legacy WFS save: invalid filename")
		}
		name := string(payload[cursor : cursor+nameLength])
		cursor += nameLength
		dataLength := binary.LittleEndian.Uint32(payload[cursor : cursor+4])
		cursor += 4
		if uint64(dataLength) > uint64(len(payload)-cursor) {
			return "", nil, errors.New("legacy WFS save: truncated file data")
		}
		contents := append([]byte(nil), payload[cursor:cursor+int(dataLength)]...)
		cursor += int(dataLength)

		if name == "db/.removed" {
			continue
		}
		guestPath, err := legacyWFSGuestPath(name)
		if err != nil {
			return "", nil, err
		}
		if _, duplicate := files[guestPath]; duplicate {
			return "", nil, fmt.Errorf("legacy WFS save: duplicate file %q", guestPath)
		}
		files[guestPath] = contents
	}
	if len(files) == 0 {
		return "", nil, errors.New("legacy WFS save: contains no restorable files")
	}
	return hex.EncodeToString(identity), files, nil
}

func legacyWFSGuestPath(name string) (string, error) {
	if strings.IndexByte(name, 0) >= 0 || strings.Contains(name, `\`) ||
		!strings.HasPrefix(name, "db/") {
		return "", fmt.Errorf("legacy WFS save: unsupported file path %q", name)
	}
	relative := strings.TrimPrefix(name, "db/")
	if relative == "" || relative == "." || strings.Contains(relative, ":") {
		return "", fmt.Errorf("legacy WFS save: invalid file path %q", name)
	}
	for _, segment := range strings.Split(relative, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("legacy WFS save: invalid file path %q", name)
		}
	}
	cleaned := path.Clean("/" + relative)
	if cleaned == "/" || len(cleaned) > legacyWFSMaxName {
		return "", fmt.Errorf("legacy WFS save: invalid file path %q", name)
	}
	return cleaned, nil
}

func mergeLegacyWFSIntoSaveData(base []byte, legacy map[string][]byte) ([]byte, error) {
	if len(base) == 0 {
		return nil, errors.New("legacy WFS save: loaded title has no initialized storage to merge into")
	}
	var envelope aramSaveDataEnvelope
	if err := gob.NewDecoder(bytes.NewReader(base)).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("legacy WFS save: decode current ARAM storage: %w", err)
	}
	if envelope.Magic != saveBackupMagic {
		return nil, fmt.Errorf("legacy WFS save: current ARAM storage has unexpected magic %q", envelope.Magic)
	}

	type fileKey struct {
		namespace shared.Namespace
		path      string
	}
	files := make(map[fileKey]shared.FileState, len(envelope.Storage.Files)+len(legacy))
	for _, file := range envelope.Storage.Files {
		files[fileKey{namespace: file.Namespace, path: file.Path}] = file
	}
	for guestPath, contents := range legacy {
		key := fileKey{namespace: shared.NamespacePrivate, path: guestPath}
		modified := int64(0)
		if current, ok := files[key]; ok {
			modified = current.Modified
		}
		files[key] = shared.FileState{
			Namespace: shared.NamespacePrivate,
			Path:      guestPath,
			Data:      append([]byte(nil), contents...),
			Modified:  modified,
			ReadOnly:  false,
		}
	}

	envelope.Storage.Files = make([]shared.FileState, 0, len(files))
	for _, file := range files {
		envelope.Storage.Files = append(envelope.Storage.Files, file)
	}
	sort.Slice(envelope.Storage.Files, func(i, j int) bool {
		left, right := envelope.Storage.Files[i], envelope.Storage.Files[j]
		if left.Namespace != right.Namespace {
			return left.Namespace < right.Namespace
		}
		return left.Path < right.Path
	})

	var out bytes.Buffer
	if err := gob.NewEncoder(&out).Encode(envelope); err != nil {
		return nil, fmt.Errorf("legacy WFS save: encode merged ARAM storage: %w", err)
	}
	return out.Bytes(), nil
}

// importLegacyWFSSave pauses a running machine only long enough to take a
// consistent storage snapshot, overlays the legacy save files, persists the
// result, then resumes the game. This makes a WFS restore work on a fresh ARAM
// launch while preserving the title-generated private support files.
func (backend *Backend) importLegacyWFSSave(
	machine aramcore.Machine,
	capability saveDataMachine,
	current string,
	data []byte,
) error {
	identity, legacyFiles, err := decodeLegacyWFS(data)
	if err != nil {
		return err
	}
	if !strings.EqualFold(identity, current) {
		return fmt.Errorf(
			"this legacy WFS save belongs to a different title (%s…), not the loaded one",
			shortSaveHash(identity),
		)
	}

	wasRunning := machine.State() == aramcore.StateRunning
	if wasRunning {
		if err := machine.Pause(); err != nil {
			return fmt.Errorf("legacy WFS save: pause title for restore: %w", err)
		}
	}
	resumed := !wasRunning
	defer func() {
		if !resumed {
			_ = machine.Resume()
		}
	}()

	base, err := capability.ExportSaveData()
	if err != nil {
		return fmt.Errorf("legacy WFS save: snapshot current storage: %w", err)
	}
	merged, err := mergeLegacyWFSIntoSaveData(base, legacyFiles)
	if err != nil {
		return err
	}
	if err := capability.ImportSaveData(merged); err != nil {
		return fmt.Errorf("legacy WFS save: import merged storage: %w", err)
	}
	if err := backend.persistSaveData(machine, current); err != nil {
		return err
	}
	if wasRunning {
		resumed = true
		if err := machine.Resume(); err != nil {
			return fmt.Errorf("legacy WFS save: resume title after restore: %w", err)
		}
	}
	return nil
}
