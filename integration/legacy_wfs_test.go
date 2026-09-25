package integration

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"testing"

	aramcore "github.com/mirusu400/aram-core/core"
	shared "github.com/mirusu400/aram-core/runtime"
	"github.com/mirusu400/aram-frontend/frontend"
)

type legacySaveTestMachine struct {
	aramcore.Machine
	data        []byte
	state       aramcore.State
	pauseCount  int
	resumeCount int
}

func (m *legacySaveTestMachine) State() aramcore.State { return m.state }

func (m *legacySaveTestMachine) Pause() error {
	if m.state != aramcore.StateRunning {
		return errors.New("pause outside running state")
	}
	m.pauseCount++
	m.state = aramcore.StatePaused
	return nil
}

func (m *legacySaveTestMachine) Resume() error {
	if m.state != aramcore.StatePaused {
		return errors.New("resume outside paused state")
	}
	m.resumeCount++
	m.state = aramcore.StateRunning
	return nil
}

func (m *legacySaveTestMachine) ExportSaveData() ([]byte, error) {
	if m.state == aramcore.StateRunning {
		return nil, errors.New("cannot export while running")
	}
	return append([]byte(nil), m.data...), nil
}

func (m *legacySaveTestMachine) ImportSaveData(data []byte) error {
	m.data = append([]byte(nil), data...)
	return nil
}

func encodeCoreSaveDataForTest(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	envelope := aramSaveDataEnvelope{Magic: saveBackupMagic}
	envelope.Storage.Schema = 1
	for _, name := range names {
		envelope.Storage.Files = append(envelope.Storage.Files, shared.FileState{
			Namespace: shared.NamespacePrivate,
			Path:      name,
			Data:      append([]byte(nil), files[name]...),
		})
	}
	var out bytes.Buffer
	if err := gob.NewEncoder(&out).Encode(envelope); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func decodeCoreSaveDataForTest(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	var envelope aramSaveDataEnvelope
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Magic != saveBackupMagic {
		t.Fatalf("save magic = %q", envelope.Magic)
	}
	files := make(map[string][]byte)
	for _, file := range envelope.Storage.Files {
		if file.Namespace == shared.NamespacePrivate {
			files[file.Path] = append([]byte(nil), file.Data...)
		}
	}
	return files
}

func encodeLegacyWFSForTest(t *testing.T, hash string, files map[string][]byte) []byte {
	t.Helper()
	identity, err := hex.DecodeString(hash)
	if err != nil || len(identity) != saveBackupHashBytes {
		t.Fatalf("bad test identity: %v", err)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var payload bytes.Buffer
	for _, name := range names {
		if len(name) > 0xffff {
			t.Fatal("test filename too long")
		}
		var short [2]byte
		binary.LittleEndian.PutUint16(short[:], uint16(len(name)))
		payload.Write(short[:])
		payload.WriteString(name)
		var size [4]byte
		binary.LittleEndian.PutUint32(size[:], uint32(len(files[name])))
		payload.Write(size[:])
		payload.Write(files[name])
	}
	body := payload.Bytes()
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
	return out
}

func TestLegacyWFSImportMergesSaveFilesAndPreservesARAMSupportFiles(t *testing.T) {
	baseline := encodeCoreSaveDataForTest(t, map[string][]byte{
		"/char.dat":  []byte("aram generated character table"),
		"/map.dat":   []byte("aram generated map table"),
		"/prefs":     []byte("current prefs"),
		"/save0.dat": []byte("empty slot"),
	})
	machine := &legacySaveTestMachine{data: baseline, state: aramcore.StateRunning}
	backend := &Backend{stateRoot: t.TempDir(), machine: machine}
	backend.input = frontend.InputInfo{SHA256: saveHashA}

	legacy := encodeLegacyWFSForTest(t, saveHashA, map[string][]byte{
		"db/.removed":  nil,
		"db/prefs":     []byte("legacy prefs"),
		"db/save0.dat": []byte("level 16 rogue"),
		"db/save1.dat": []byte("blank slot two"),
	})
	if err := backend.ImportSaveData(legacy); err != nil {
		t.Fatalf("legacy import: %v", err)
	}
	if machine.state != aramcore.StateRunning || machine.pauseCount != 1 || machine.resumeCount != 1 {
		t.Fatalf(
			"machine state=%s pauses=%d resumes=%d",
			machine.state, machine.pauseCount, machine.resumeCount,
		)
	}

	files := decodeCoreSaveDataForTest(t, machine.data)
	if got := string(files["/char.dat"]); got != "aram generated character table" {
		t.Fatalf("char.dat was not preserved: %q", got)
	}
	if got := string(files["/map.dat"]); got != "aram generated map table" {
		t.Fatalf("map.dat was not preserved: %q", got)
	}
	if got := string(files["/prefs"]); got != "legacy prefs" {
		t.Fatalf("prefs = %q", got)
	}
	if got := string(files["/save0.dat"]); got != "level 16 rogue" {
		t.Fatalf("save0.dat = %q", got)
	}
	if got := string(files["/save1.dat"]); got != "blank slot two" {
		t.Fatalf("save1.dat = %q", got)
	}

	persisted, err := os.ReadFile(filepath.Join(backend.stateRoot, saveHashA, "savedata.bin"))
	if err != nil {
		t.Fatalf("read persisted migrated save: %v", err)
	}
	persistedFiles := decodeCoreSaveDataForTest(t, persisted)
	if got := string(persistedFiles["/save0.dat"]); got != "level 16 rogue" {
		t.Fatalf("persisted save0.dat = %q", got)
	}
}

func TestLegacyWFSImportRejectsWrongTitleWithoutMutation(t *testing.T) {
	baseline := encodeCoreSaveDataForTest(t, map[string][]byte{
		"/save0.dat": []byte("keep me"),
	})
	machine := &legacySaveTestMachine{data: baseline, state: aramcore.StatePaused}
	backend := &Backend{stateRoot: t.TempDir(), machine: machine}
	backend.input = frontend.InputInfo{SHA256: saveHashA}
	legacy := encodeLegacyWFSForTest(t, saveHashB, map[string][]byte{
		"db/save0.dat": []byte("wrong title"),
	})

	err := backend.ImportSaveData(legacy)
	if err == nil {
		t.Fatal("legacy WFS from a different title was accepted")
	}
	if !bytes.Equal(machine.data, baseline) {
		t.Fatal("rejected legacy WFS mutated the running save")
	}
}

func TestLegacyWFSDecodeRejectsCorruptionAndUnsafePath(t *testing.T) {
	good := encodeLegacyWFSForTest(t, saveHashA, map[string][]byte{
		"db/save0.dat": []byte("save"),
	})
	corrupt := append([]byte(nil), good...)
	corrupt[len(corrupt)-1] ^= 0xff
	if _, _, err := decodeLegacyWFS(corrupt); err == nil {
		t.Fatal("legacy WFS checksum corruption was accepted")
	}

	unsafe := encodeLegacyWFSForTest(t, saveHashA, map[string][]byte{
		"db/../save0.dat": []byte("save"),
	})
	if _, _, err := decodeLegacyWFS(unsafe); err == nil {
		t.Fatal("legacy WFS parent path was accepted")
	}
}
