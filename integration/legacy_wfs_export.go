package integration

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"sort"
	"strings"

	shared "github.com/mirusu400/aram-core/runtime"
)

// encodeLegacyWFSFromARAMSave converts an ARAM storage snapshot to the legacy
// WFSAVEBK container used by older WIPI emulators when that conversion is
// lossless. WFS only carries db/* files, so saves containing shared-namespace
// data or record stores stay in ARAM's native .aramsave format instead of
// silently dropping persistent state.
func encodeLegacyWFSFromARAMSave(hashHex string, data []byte) ([]byte, bool, error) {
	var envelope aramSaveDataEnvelope
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&envelope); err != nil {
		// Optional capability implementations used by tests and future backends
		// are allowed to return their own payload. Such data is simply not WFS-
		// representable and falls back to the native ARAM container.
		return nil, false, nil
	}
	if envelope.Magic != saveBackupMagic {
		return nil, false, nil
	}
	if len(envelope.Storage.RecordStores) != 0 {
		return nil, false, nil
	}
	for _, directory := range envelope.Storage.Directories {
		if directory.Namespace != shared.NamespacePrivate {
			return nil, false, nil
		}
	}

	entries := make(map[string][]byte, len(envelope.Storage.Files)+1)
	// Real legacy WIPI backups carry this zero-length bookkeeping entry. The
	// importer ignores it, but emitting it keeps ARAM-produced WFS files shaped
	// like the old emulator's files rather than merely using the same extension.
	entries["db/.removed"] = nil
	for _, file := range envelope.Storage.Files {
		if file.Namespace != shared.NamespacePrivate || file.ReadOnly {
			return nil, false, nil
		}
		name, err := legacyWFSNameForGuestPath(file.Path)
		if err != nil {
			return nil, false, err
		}
		if _, duplicate := entries[name]; duplicate {
			return nil, false, fmt.Errorf("legacy WFS save: duplicate export file %q", name)
		}
		entries[name] = append([]byte(nil), file.Data...)
	}
	if len(entries) == 1 {
		return nil, false, nil
	}

	identity, err := hex.DecodeString(hashHex)
	if err != nil {
		return nil, false, fmt.Errorf("legacy WFS save: title identity is not hex: %w", err)
	}
	if len(identity) != saveBackupHashBytes {
		return nil, false, fmt.Errorf(
			"legacy WFS save: title identity is %d bytes, want %d",
			len(identity), saveBackupHashBytes,
		)
	}
	if len(entries) > legacyWFSMaxEntries {
		return nil, false, fmt.Errorf("legacy WFS save: too many file entries: %d", len(entries))
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	var payload bytes.Buffer
	for _, name := range names {
		if len(name) == 0 || len(name) > legacyWFSMaxName || len(name) > 0xffff {
			return nil, false, fmt.Errorf("legacy WFS save: invalid export filename %q", name)
		}
		contents := entries[name]
		if uint64(len(contents)) > uint64(^uint32(0)) {
			return nil, false, fmt.Errorf("legacy WFS save: file %q is too large", name)
		}
		var short [2]byte
		binary.LittleEndian.PutUint16(short[:], uint16(len(name)))
		payload.Write(short[:])
		payload.WriteString(name)
		var word [4]byte
		binary.LittleEndian.PutUint32(word[:], uint32(len(contents)))
		payload.Write(word[:])
		payload.Write(contents)
	}
	body := payload.Bytes()
	if len(body) > legacyWFSMaxBytes-legacyWFSHeaderBytes {
		return nil, false, fmt.Errorf("legacy WFS save: exported payload exceeds size limit")
	}

	out := make([]byte, 0, legacyWFSHeaderBytes+len(body))
	out = append(out, legacyWFSMagic...)
	out = append(out, legacyWFSVersion, 0)
	out = append(out, identity...)
	var word [4]byte
	binary.LittleEndian.PutUint32(word[:], uint32(len(body)))
	out = append(out, word[:]...)
	binary.LittleEndian.PutUint32(word[:], crc32.ChecksumIEEE(body))
	out = append(out, word[:]...)
	out = append(out, body...)
	return out, true, nil
}

func legacyWFSNameForGuestPath(guestPath string) (string, error) {
	if !strings.HasPrefix(guestPath, "/") || guestPath == "/" {
		return "", fmt.Errorf("legacy WFS save: invalid private path %q", guestPath)
	}
	name := "db/" + strings.TrimPrefix(guestPath, "/")
	decoded, err := legacyWFSGuestPath(name)
	if err != nil {
		return "", err
	}
	if decoded != guestPath {
		return "", fmt.Errorf("legacy WFS save: private path %q is not round-trippable", guestPath)
	}
	return name, nil
}
