from pathlib import Path

RESULTS = Path("../aram-frontend/frontend/shell_results.go")
BACKEND = Path("integration/backend.go")
LEGACY_WFS = Path("integration/legacy_wfs.go")
LEGACY_TEST = Path("integration/legacy_wfs_test.go")


def replace_once(path: Path, old: str, new: str, marker: str) -> None:
    source = path.read_text()
    if marker in source:
        print(f"{path}: patch already present")
        return
    if old not in source:
        raise SystemExit(f"{path}: expected baseline block not found")
    path.write_text(source.replace(old, new, 1))
    print(f"patched {path}")


# The frontend must reopen the title after a successful restore. A running game
# can have the old save cached in guest memory, so a fresh Open is the only safe
# point at which every title observes the restored persistent storage.
results_source = RESULTS.read_text()
results_old = '''\t\tcase result := <-s.saveRestoreResults:
\t\t\tif result.err != nil {
\t\t\t\ts.setStatus(s.tr("Restore save: ") + result.err.Error())
\t\t\t\tcontinue
\t\t\t}
\t\t\ts.setStatus(s.trf("Save restored from %s", result.name))
'''
results_new = '''\t\tcase result := <-s.saveRestoreResults:
\t\t\tif result.err != nil {
\t\t\t\ts.setStatus(s.tr("Restore save: ") + result.err.Error())
\t\t\t\tcontinue
\t\t\t}
\t\t\ts.setStatus(s.trf("Save restored from %s", result.name))
\t\t\t// ImportSaveData replaces the title's persistent storage, but a
\t\t\t// running game can already have the old save cached in guest memory.
\t\t\t// Reopen the exact same input immediately after a successful restore
\t\t\t// so its first save read observes the imported WFS/ARAM data instead
\t\t\t// of continuing with stale in-memory state and later writing it back.
\t\t\tif s.input != nil && !s.loading {
\t\t\t\ts.restartCurrentTitle()
\t\t\t}
'''
if results_new not in results_source:
    if results_old not in results_source:
        raise SystemExit("shell_results.go save restore result block did not match expected baseline")
    RESULTS.write_text(results_source.replace(results_old, results_new, 1))
    print(f"patched {RESULTS} to restart the loaded title after a successful save restore")
else:
    print(f"{RESULTS}: restart patch already present")


# A WFS restore is committed to savedata.bin before the frontend restarts the
# title. The normal Backend.Close() path exports the still-running machine one
# more time. For games such as Inotia 2 that cache save0.dat, that final export
# can overwrite the just-restored file with the pre-restore contents. Remember
# the title whose WFS was staged so exactly the next close skips that stale
# persistence pass; all ordinary closes keep their existing behavior.
backend_field_old = '''\trunRequested  bool
\tlastFrameHash uint64
'''
backend_field_new = '''\trunRequested  bool
\t// skipNextCloseSaveHash is set only after a legacy WFS restore has already
\t// been durably staged for the imminent frontend restart. The old live guest
\t// must not overwrite that restored blob during the restart's Close().
\tskipNextCloseSaveHash string
\tlastFrameHash         uint64
'''
replace_once(BACKEND, backend_field_old, backend_field_new, "skipNextCloseSaveHash string")

backend_close_old = '''\tbackend.mu.Lock()
\tmachine := backend.machine
\tsourceFile := backend.sourceFile
\tclosingHash := backend.input.SHA256
\tbackend.machine = nil
\tbackend.sourceFile = nil
\tbackend.source = aramcore.Source{}
\tbackend.input = frontend.InputInfo{}
\tbackend.runRequested = false
'''
backend_close_new = '''\tbackend.mu.Lock()
\tmachine := backend.machine
\tsourceFile := backend.sourceFile
\tclosingHash := backend.input.SHA256
\tskipSavePersist := closingHash != "" &&
\t\tstrings.EqualFold(backend.skipNextCloseSaveHash, closingHash)
\tbackend.skipNextCloseSaveHash = ""
\tbackend.machine = nil
\tbackend.sourceFile = nil
\tbackend.source = aramcore.Source{}
\tbackend.input = frontend.InputInfo{}
\tbackend.runRequested = false
'''
replace_once(BACKEND, backend_close_old, backend_close_new, "skipSavePersist := closingHash != \"\"")

backend_persist_old = '''\tif machine != nil {
\t\t// Flush the title's writable storage so saves survive a close/reopen.
\t\terrs = append(errs, backend.persistSaveData(machine, closingHash))
\t\terrs = append(errs, machine.Close())
\t}
'''
backend_persist_new = '''\tif machine != nil {
\t\t// Ordinary closes flush the live title. A successful WFS restore is the
\t\t// one exception: its merged blob is already durable and the live guest
\t\t// may still cache the old save, so persisting here would clobber it.
\t\tif !skipSavePersist {
\t\t\terrs = append(errs, backend.persistSaveData(machine, closingHash))
\t\t}
\t\terrs = append(errs, machine.Close())
\t}
'''
replace_once(BACKEND, backend_persist_old, backend_persist_new, "if !skipSavePersist {")


# Do not apply the WFS bytes to the live guest and then resume it. Instead take
# a consistent snapshot, overlay the WFS files, write that merged ARAM storage
# directly to savedata.bin, and leave execution stopped until the frontend's
# immediate restart. This removes the window in which Inotia 2 can rewrite its
# cached pre-restore save0.dat.
legacy_old = '''// importLegacyWFSSave pauses a running machine only long enough to take a
// consistent storage snapshot, overlays the legacy save files, persists the
// result, then resumes the game. This makes a WFS restore work on a fresh ARAM
// launch while preserving the title-generated private support files.
func (backend *Backend) importLegacyWFSSave(
\tmachine aramcore.Machine,
\tcapability saveDataMachine,
\tcurrent string,
\tdata []byte,
) error {
\tidentity, legacyFiles, err := decodeLegacyWFS(data)
\tif err != nil {
\t\treturn err
\t}
\tif !strings.EqualFold(identity, current) {
\t\treturn fmt.Errorf(
\t\t\t"this legacy WFS save belongs to a different title (%s…), not the loaded one",
\t\t\tshortSaveHash(identity),
\t\t)
\t}

\twasRunning := machine.State() == aramcore.StateRunning
\tif wasRunning {
\t\tif err := machine.Pause(); err != nil {
\t\t\treturn fmt.Errorf("legacy WFS save: pause title for restore: %w", err)
\t\t}
\t}
\tresumed := !wasRunning
\tdefer func() {
\t\tif !resumed {
\t\t\t_ = machine.Resume()
\t\t}
\t}()

\tbase, err := capability.ExportSaveData()
\tif err != nil {
\t\treturn fmt.Errorf("legacy WFS save: snapshot current storage: %w", err)
\t}
\tmerged, err := mergeLegacyWFSIntoSaveData(base, legacyFiles)
\tif err != nil {
\t\treturn err
\t}
\tif err := capability.ImportSaveData(merged); err != nil {
\t\treturn fmt.Errorf("legacy WFS save: import merged storage: %w", err)
\t}
\tif err := backend.persistSaveData(machine, current); err != nil {
\t\treturn err
\t}
\tif wasRunning {
\t\tresumed = true
\t\tif err := machine.Resume(); err != nil {
\t\t\treturn fmt.Errorf("legacy WFS save: resume title after restore: %w", err)
\t\t}
\t}
\treturn nil
}
'''
legacy_new = '''// importLegacyWFSSave takes a consistent snapshot of the live title, overlays
// the legacy db/* files, and stages the merged ARAM storage on disk for the
// frontend's immediate close/reopen. It intentionally does not resume or write
// the merged bytes back into the old guest: games such as Inotia 2 cache their
// save slot in memory and can otherwise overwrite the restored save0.dat before
// the restart gets a chance to reload it.
func (backend *Backend) importLegacyWFSSave(
\tmachine aramcore.Machine,
\tcapability saveDataMachine,
\tcurrent string,
\tdata []byte,
) error {
\tidentity, legacyFiles, err := decodeLegacyWFS(data)
\tif err != nil {
\t\treturn err
\t}
\tif !strings.EqualFold(identity, current) {
\t\treturn fmt.Errorf(
\t\t\t"this legacy WFS save belongs to a different title (%s…), not the loaded one",
\t\t\tshortSaveHash(identity),
\t\t)
\t}

\twasRunning := machine.State() == aramcore.StateRunning
\twasRequested := backend.runningRequested()
\tif wasRunning {
\t\tif err := machine.Pause(); err != nil {
\t\t\treturn fmt.Errorf("legacy WFS save: pause title for restore: %w", err)
\t\t}
\t}
\trestoreCommitted := false
\tdefer func() {
\t\tif restoreCommitted {
\t\t\treturn
\t\t}
\t\tif wasRunning {
\t\t\t_ = machine.Resume()
\t\t}
\t\tbackend.setRunRequested(wasRequested)
\t}()

\t// Stop product-level frame stepping while the successful restore waits for
\t// the frontend result handler to restart the title.
\tbackend.setRunRequested(false)
\tbase, err := capability.ExportSaveData()
\tif err != nil {
\t\treturn fmt.Errorf("legacy WFS save: snapshot current storage: %w", err)
\t}
\tmerged, err := mergeLegacyWFSIntoSaveData(base, legacyFiles)
\tif err != nil {
\t\treturn err
\t}
\tif err := backend.writeSaveData(current, merged); err != nil {
\t\treturn fmt.Errorf("legacy WFS save: stage restored storage: %w", err)
\t}

\tbackend.mu.Lock()
\tbackend.skipNextCloseSaveHash = current
\tbackend.runRequested = false
\tbackend.mu.Unlock()
\trestoreCommitted = true
\treturn nil
}
'''
replace_once(LEGACY_WFS, legacy_old, legacy_new, "restoreCommitted := false")


# Regression coverage: the restore must remain staged even if the old guest
# still contains stale save bytes when Backend.Close() runs for the restart.
test_close_old = '''func (m *legacySaveTestMachine) Resume() error {
\tif m.state != aramcore.StatePaused {
\t\treturn errors.New("resume outside paused state")
\t}
\tm.resumeCount++
\tm.state = aramcore.StateRunning
\treturn nil
}

func (m *legacySaveTestMachine) ExportSaveData() ([]byte, error) {
'''
test_close_new = '''func (m *legacySaveTestMachine) Resume() error {
\tif m.state != aramcore.StatePaused {
\t\treturn errors.New("resume outside paused state")
\t}
\tm.resumeCount++
\tm.state = aramcore.StateRunning
\treturn nil
}

func (m *legacySaveTestMachine) Close() error { return nil }

func (m *legacySaveTestMachine) ExportSaveData() ([]byte, error) {
'''
replace_once(LEGACY_TEST, test_close_old, test_close_new, "func (m *legacySaveTestMachine) Close() error")

test_backend_old = '''\tmachine := &legacySaveTestMachine{data: baseline, state: aramcore.StateRunning}
\tbackend := &Backend{stateRoot: t.TempDir(), machine: machine}
\tbackend.input = frontend.InputInfo{SHA256: saveHashA}
'''
test_backend_new = '''\tmachine := &legacySaveTestMachine{data: baseline, state: aramcore.StateRunning}
\tbackend := &Backend{stateRoot: t.TempDir(), machine: machine, runRequested: true}
\tbackend.input = frontend.InputInfo{SHA256: saveHashA}
'''
replace_once(LEGACY_TEST, test_backend_old, test_backend_new, "runRequested: true}")

test_state_old = '''\tif machine.state != aramcore.StateRunning || machine.pauseCount != 1 || machine.resumeCount != 1 {
\t\tt.Fatalf(
\t\t\t"machine state=%s pauses=%d resumes=%d",
\t\t\tmachine.state, machine.pauseCount, machine.resumeCount,
\t\t)
\t}

\tfiles := decodeCoreSaveDataForTest(t, machine.data)
'''
test_state_new = '''\tif machine.state != aramcore.StatePaused || machine.pauseCount != 1 || machine.resumeCount != 0 {
\t\tt.Fatalf(
\t\t\t"machine state=%s pauses=%d resumes=%d",
\t\t\tmachine.state, machine.pauseCount, machine.resumeCount,
\t\t)
\t}
\tif backend.runningRequested() {
\t\tt.Fatal("frame stepping remained enabled while the restored save awaited restart")
\t}
\tif backend.skipNextCloseSaveHash != saveHashA {
\t\tt.Fatalf("skip-close hash = %q, want %q", backend.skipNextCloseSaveHash, saveHashA)
\t}

\t// The old guest itself is intentionally not mutated. The restored storage is
\t// staged on disk and becomes live only after the frontend restarts the title.
\tfiles := decodeCoreSaveDataForTest(t, machine.data)
'''
replace_once(LEGACY_TEST, test_state_old, test_state_new, "frame stepping remained enabled")

# The old guest should still contain its baseline slot; assert the staged file,
# then emulate a worst-case stale in-memory slot at close and prove Close does
# not overwrite the committed WFS restore.
test_files_old = '''\tif got := string(files["/prefs"]); got != "legacy prefs" {
\t\tt.Fatalf("prefs = %q", got)
\t}
\tif got := string(files["/save0.dat"]); got != "level 16 rogue" {
\t\tt.Fatalf("save0.dat = %q", got)
\t}
\tif got := string(files["/save1.dat"]); got != "blank slot two" {
\t\tt.Fatalf("save1.dat = %q", got)
\t}

\tpersisted, err := os.ReadFile(filepath.Join(backend.stateRoot, saveHashA, "savedata.bin"))
'''
test_files_new = '''\tif got := string(files["/prefs"]); got != "current prefs" {
\t\tt.Fatalf("live prefs changed before restart: %q", got)
\t}
\tif got := string(files["/save0.dat"]); got != "empty slot" {
\t\tt.Fatalf("live save0.dat changed before restart: %q", got)
\t}
\tif _, ok := files["/save1.dat"]; ok {
\t\tt.Fatal("live save1.dat appeared before restart")
\t}

\tpersisted, err := os.ReadFile(filepath.Join(backend.stateRoot, saveHashA, "savedata.bin"))
'''
replace_once(LEGACY_TEST, test_files_old, test_files_new, "live prefs changed before restart")

test_tail_old = '''\tpersistedFiles := decodeCoreSaveDataForTest(t, persisted)
\tif got := string(persistedFiles["/save0.dat"]); got != "level 16 rogue" {
\t\tt.Fatalf("persisted save0.dat = %q", got)
\t}
}
'''
test_tail_new = '''\tpersistedFiles := decodeCoreSaveDataForTest(t, persisted)
\tif got := string(persistedFiles["/save0.dat"]); got != "level 16 rogue" {
\t\tt.Fatalf("persisted save0.dat = %q", got)
\t}
\tif got := string(persistedFiles["/char.dat"]); got != "aram generated character table" {
\t\tt.Fatalf("staged char.dat was not preserved: %q", got)
\t}

\t// Simulate the exact failure mode seen in Inotia 2: the old guest still has
\t// pre-restore save bytes when the frontend closes it for the restart.
\tmachine.data = baseline
\tif err := backend.Close(); err != nil {
\t\tt.Fatalf("close after staged WFS restore: %v", err)
\t}
\tpersisted, err = os.ReadFile(filepath.Join(backend.stateRoot, saveHashA, "savedata.bin"))
\tif err != nil {
\t\tt.Fatalf("read staged save after close: %v", err)
\t}
\tpersistedFiles = decodeCoreSaveDataForTest(t, persisted)
\tif got := string(persistedFiles["/save0.dat"]); got != "level 16 rogue" {
\t\tt.Fatalf("close clobbered restored save0.dat: %q", got)
\t}
}
'''
replace_once(LEGACY_TEST, test_tail_old, test_tail_new, "close clobbered restored save0.dat")

print("WFS restore now stages durable data and protects it from restart-time stale persistence")
