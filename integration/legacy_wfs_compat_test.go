package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirusu400/aram-core/application"
	aramcore "github.com/mirusu400/aram-core/core"
	"github.com/mirusu400/aram-core/debugkit"
	"github.com/mirusu400/aram-frontend/frontend"
)

func testCertificatePackage(t *testing.T, names ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := z.Create(name)
		if err != nil { t.Fatal(err) }
		if _, err := w.Write([]byte("destination certificate")); err != nil { t.Fatal(err) }
	}
	if err := z.Close(); err != nil { t.Fatal(err) }
	return buf.Bytes()
}

func testMigrationBackend(t *testing.T, current string, pkg []byte) (*Backend, *legacySaveTestMachine) {
	t.Helper()
	baseline := encodeCoreSaveDataForTest(t, map[string][]byte{
		"/save0.dat": []byte("old guest slot"),
		"/cert.c2s": []byte("stale guest certificate"),
		"/char.dat": []byte("destination support data"),
	})
	machine := &legacySaveTestMachine{data: baseline, state: aramcore.StatePaused}
	backend := &Backend{stateRoot: t.TempDir(), machine: machine}
	backend.input = frontend.InputInfo{SHA256: current}
	backend.source = aramcore.Source{ReaderAt: bytes.NewReader(pkg), Size: int64(len(pkg))}
	return backend, machine
}

func TestLegacyWFSKnownInotia2AliasesAreDirectional(t *testing.T) {
	for _, tc := range []struct{ current, source string; want bool }{
		{inotia2ARAMInputHash, inotia2RareInputHash, true},
		{inotia2ARAMInputHash, inotia2OriginalInputHash, true},
		{strings.ToUpper(inotia2ARAMInputHash), strings.ToUpper(inotia2RareInputHash), true},
		{saveHashA, saveHashA, true},
		{inotia2RareInputHash, inotia2ARAMInputHash, false},
		{saveHashA, inotia2RareInputHash, false},
		{inotia2ARAMInputHash, saveHashA, false},
	} {
		if got := compatibleLegacyWFSIdentity(tc.current, tc.source); got != tc.want {
			t.Fatalf("compatible(%s, %s) = %t, want %t", tc.current, tc.source, got, tc.want)
		}
	}
}

func TestLegacyWFSInotia2MigrationPreservesSlotsAndPackageAuthentication(t *testing.T) {
	for _, identity := range []string{inotia2RareInputHash, inotia2OriginalInputHash, inotia2ARAMInputHash} {
		t.Run(shortSaveHash(identity), func(t *testing.T) {
			backend, machine := testMigrationBackend(t, inotia2ARAMInputHash, testCertificatePackage(t, "title/p/cert.c2s"))
			baseline := append([]byte(nil), machine.data...)
			legacy := encodeLegacyWFSForTest(t, identity, map[string][]byte{
				"db/cert.c2s": []byte("foreign W-Feature certificate"),
				"db/save0.dat": []byte{0, 255, 17, 0, 21},
				"db/save1.dat": []byte("second slot"),
				"db/envinfo.dat": []byte("imported options"),
			})
			if err := backend.ImportSaveData(legacy); err != nil { t.Fatal(err) }
			if !bytes.Equal(machine.data, baseline) { t.Fatal("live guest was modified before restart") }
			if err := backend.Close(); err != nil { t.Fatal(err) }
			data, err := os.ReadFile(filepath.Join(backend.stateRoot, inotia2ARAMInputHash, "savedata.bin"))
			if err != nil { t.Fatal(err) }
			files := decodeCoreSaveDataForTest(t, data)
			if !bytes.Equal(files["/save0.dat"], []byte{0, 255, 17, 0, 21}) { t.Fatal("save slot bytes changed") }
			if string(files["/save1.dat"]) != "second slot" { t.Fatal("second save slot was not imported") }
			if string(files["/cert.c2s"]) != "destination certificate" { t.Fatal("destination authentication was lost") }
			if string(files["/char.dat"]) != "destination support data" { t.Fatal("destination support file was lost") }
			if string(files["/envinfo.dat"]) != "imported options" { t.Fatal("W-Feature options were lost") }
		})
	}
}

func TestLegacyWFSOrdinaryTitleStillImportsOwnCertificate(t *testing.T) {
	backend, _ := testMigrationBackend(t, saveHashA, nil)
	legacy := encodeLegacyWFSForTest(t, saveHashA, map[string][]byte{"db/cert.c2s": []byte("ordinary certificate"), "db/save0.dat": []byte("slot")})
	if err := backend.ImportSaveData(legacy); err != nil { t.Fatal(err) }
	data, err := os.ReadFile(filepath.Join(backend.stateRoot, saveHashA, "savedata.bin"))
	if err != nil { t.Fatal(err) }
	if got := string(decodeCoreSaveDataForTest(t, data)["/cert.c2s"]); got != "ordinary certificate" { t.Fatalf("certificate = %q", got) }
}

func TestLegacyWFSInotia2RejectsCorruptionAndMissingAuthenticationBeforeMutation(t *testing.T) {
	for _, tc := range []struct{ name string; pkg []byte; corrupt bool }{
		{"no package", nil, false},
		{"no certificate", testCertificatePackage(t, "title/p/other.dat"), false},
		{"duplicate certificates", testCertificatePackage(t, "a/p/cert.c2s", "b/P/CERT.C2S"), false},
		{"corrupt WFS", testCertificatePackage(t, "title/p/cert.c2s"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend, machine := testMigrationBackend(t, inotia2ARAMInputHash, tc.pkg)
			baseline := append([]byte(nil), machine.data...)
			legacy := encodeLegacyWFSForTest(t, inotia2RareInputHash, map[string][]byte{"db/save0.dat": []byte("new slot")})
			if tc.corrupt { legacy[len(legacy)-1] ^= 1 }
			if err := backend.ImportSaveData(legacy); err == nil { t.Fatal("invalid migration was accepted") }
			if !bytes.Equal(machine.data, baseline) || backend.skipNextCloseSaveHash != "" { t.Fatal("rejected migration changed storage") }
			if _, err := os.Stat(filepath.Join(backend.stateRoot, inotia2ARAMInputHash, "savedata.bin")); !os.IsNotExist(err) { t.Fatal("rejected migration staged a save") }
		})
	}
}

// Private files are optional local evidence. Hosted CI uses synthetic fixtures
// and never needs to upload a user's game, progress, or authentication data.
func TestPrivateWFeatureSaveImport(t *testing.T) {
	gamePath, savePath := os.Getenv("ARAM_TEST_GAME"), os.Getenv("ARAM_TEST_WFS")
	if gamePath == "" || savePath == "" { t.Skip("set ARAM_TEST_GAME and ARAM_TEST_WFS for private evidence") }
	game, err := os.ReadFile(gamePath)
	if err != nil { t.Fatal(err) }
	backup, err := os.ReadFile(savePath)
	if err != nil { t.Fatal(err) }
	identity, originalFiles, err := decodeLegacyWFS(backup)
	if err != nil { t.Fatal(err) }
	current := fmt.Sprintf("%x", sha256.Sum256(game))
	source := aramcore.Source{Name: filepath.Base(gamePath), ReaderAt: bytes.NewReader(game), Size: int64(len(game)), SHA256: current}
	factory := application.NewFactory()
	factory.FrameRunBudget = application.DefaultHandsetRunBudget
	factory.KTFRunBudget = application.DefaultKTFHandsetRunBudget
	machine, err := factory.Create(context.Background(), source)
	if err != nil { t.Fatal(err) }
	backend := &Backend{stateRoot: t.TempDir(), machine: machine, source: source, input: frontend.InputInfo{SHA256: current}}
	if err := backend.ImportSaveData(backup); err != nil { t.Fatal(err) }
	if err := backend.Close(); err != nil { t.Fatal(err) }
	reopened, err := factory.Create(context.Background(), source)
	if err != nil { t.Fatal(err) }
	defer reopened.Close()
	if err := backend.restoreSaveData(reopened, current); err != nil { t.Fatal(err) }
	capability, ok := saveDataFrom(reopened)
	if !ok { t.Fatal("reopened core has no save-data capability") }
	data, err := capability.ExportSaveData()
	if err != nil { t.Fatal(err) }
	files := decodeCoreSaveDataForTest(t, data)
	for name, original := range originalFiles {
		if name == "/cert.c2s" { continue }
		if !bytes.Equal(files[name], original) { t.Fatalf("restored file %s changed", name) }
	}
	// Close releases backend.source; compare against the same loaded package
	// through a fresh holder after exercising the real close/reopen path.
	holder := &Backend{source: source}
	prepared, err := holder.prepareLegacyWFSFiles(current, identity, originalFiles)
	if err != nil || !bytes.Equal(files["/cert.c2s"], prepared["/cert.c2s"]) { t.Fatalf("destination authentication mismatch: %v", err) }
	report := map[string]any{"source_sha256": identity, "destination_sha256": current, "restored_files": len(originalFiles), "slot_bytes_preserved": true, "certificate_from_loaded_game": true, "restart_preserved_import": true, "save0_sha256": fmt.Sprintf("%x", sha256.Sum256(files["/save0.dat"]))}
	output := os.Getenv("ARAM_TEST_OUTPUT")
	if output != "" {
		if err := os.MkdirAll(output, 0755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(filepath.Join(output, "restored-savedata.bin"), data, 0600); err != nil { t.Fatal(err) }
	}
	session, err := debugkit.New(reopened, debugkit.Options{Diagnostics: application.MachineDiagnostics(reopened)})
	if err != nil { t.Fatal(err) }
	if err := session.Start(context.Background()); err != nil { t.Fatal(err) }
	if err := session.Step(context.Background(), 240); err != nil { t.Fatal(err) }
	report["boot_frames"] = 240
	report["boot_state"] = reopened.State().String()
	if output != "" {
		frame := session.Snapshot()
		if frame == nil { t.Fatal("reopened title produced no framebuffer") }
		image, err := os.Create(filepath.Join(output, "restored-boot.png"))
		if err != nil { t.Fatal(err) }
		encodeErr := png.Encode(image, frame)
		closeErr := image.Close()
		if encodeErr != nil { t.Fatal(encodeErr) }
		if closeErr != nil { t.Fatal(closeErr) }
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil { t.Fatal(err) }
		if err := os.WriteFile(filepath.Join(output, "verification.json"), append(encoded, '\n'), 0600); err != nil { t.Fatal(err) }
	}
	t.Logf("private WFS restored and reopened: source=%s current=%s files=%d save0=%x", identity, current, len(originalFiles), sha256.Sum256(files["/save0.dat"]))
}
