package integration

import (
	"bytes"
	"encoding/gob"
	"testing"

	shared "github.com/mirusu400/aram-core/runtime"
)

func TestARAMSaveExportsAsLegacyWFSWhenRepresentable(t *testing.T) {
	payload := encodeCoreSaveDataForTest(t, map[string][]byte{
		"/char.dat":  []byte("aram support"),
		"/map.dat":   []byte("aram map"),
		"/prefs":     []byte("prefs"),
		"/save0.dat": []byte("level 16 rogue"),
		"/save1.dat": []byte("slot two"),
	})
	blob, compatible, err := encodeLegacyWFSFromARAMSave(saveHashA, payload)
	if err != nil {
		t.Fatalf("encode legacy WFS: %v", err)
	}
	if !compatible {
		t.Fatal("private-file ARAM save was not considered WFS-compatible")
	}
	if !bytes.HasPrefix(blob, []byte(legacyWFSMagic)) {
		t.Fatalf("backup magic = %q, want %q", blob[:8], legacyWFSMagic)
	}

	identity, files, err := decodeLegacyWFS(blob)
	if err != nil {
		t.Fatalf("decode exported WFS: %v", err)
	}
	if identity != saveHashA {
		t.Fatalf("identity = %q, want %q", identity, saveHashA)
	}
	for path, want := range map[string]string{
		"/char.dat":  "aram support",
		"/map.dat":   "aram map",
		"/prefs":     "prefs",
		"/save0.dat": "level 16 rogue",
		"/save1.dat": "slot two",
	} {
		if got := string(files[path]); got != want {
			t.Fatalf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestARAMSaveKeepsNativeFormatWhenWFSWouldLoseRecordStores(t *testing.T) {
	envelope := aramSaveDataEnvelope{Magic: saveBackupMagic}
	envelope.Storage.Schema = 1
	envelope.Storage.Files = []shared.FileState{{
		Namespace: shared.NamespacePrivate,
		Path:      "/save0.dat",
		Data:      []byte("file save"),
	}}
	envelope.Storage.RecordStores = []shared.PersistentRecordStoreState{{
		Owner:  1,
		Name:   "SAVE",
		NextID: 2,
		Records: []shared.RecordState{{
			ID:   1,
			Data: []byte("record save"),
		}},
	}}
	var payload bytes.Buffer
	if err := gob.NewEncoder(&payload).Encode(envelope); err != nil {
		t.Fatal(err)
	}

	blob, compatible, err := encodeLegacyWFSFromARAMSave(saveHashA, payload.Bytes())
	if err != nil {
		t.Fatalf("probe WFS compatibility: %v", err)
	}
	if compatible || blob != nil {
		t.Fatal("record-store save was converted to lossy WFS")
	}
}
