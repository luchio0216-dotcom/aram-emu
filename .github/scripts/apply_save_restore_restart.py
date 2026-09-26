from pathlib import Path

RESULTS = Path("../aram-frontend/frontend/shell_results.go")

source = RESULTS.read_text()
old = '''\t\tcase result := <-s.saveRestoreResults:
\t\t\tif result.err != nil {
\t\t\t\ts.setStatus(s.tr("Restore save: ") + result.err.Error())
\t\t\t\tcontinue
\t\t\t}
\t\t\ts.setStatus(s.trf("Save restored from %s", result.name))
'''
new = '''\t\tcase result := <-s.saveRestoreResults:
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

if new not in source:
    if old not in source:
        raise SystemExit("shell_results.go save restore result block did not match expected baseline")
    source = source.replace(old, new, 1)

RESULTS.write_text(source)
print(f"patched {RESULTS} to restart the loaded title after a successful save restore")
