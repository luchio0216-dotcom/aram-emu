from pathlib import Path

SAVEDATA = Path("../aram-frontend/frontend/shell_savedata.go")
PICKER = Path("../aram-frontend/frontend/picker.go")

savedata = SAVEDATA.read_text()

if '\t"bytes"\n' not in savedata:
    marker = 'import (\n\t"errors"\n'
    if marker not in savedata:
        raise SystemExit("shell_savedata.go import block did not match expected baseline")
    savedata = savedata.replace(marker, 'import (\n\t"bytes"\n\t"errors"\n', 1)

old_export = '''\tgo func() {
\t\tvar path string
\t\tdata, err := transfer.ExportSaveData()
\t\tif err == nil {
\t\t\tpath, err = writeTextArtifact("save-backups", prefix, ".aramsave", data)
\t\t}
'''
new_export = '''\tgo func() {
\t\tvar path string
\t\tdata, err := transfer.ExportSaveData()
\t\tif err == nil {
\t\t\textension := ".aramsave"
\t\t\tif bytes.HasPrefix(data, []byte("WFSAVEBK")) {
\t\t\t\textension = ".wfs"
\t\t\t}
\t\t\tpath, err = writeTextArtifact("save-backups", prefix, extension, data)
\t\t}
'''
if old_export in savedata:
    savedata = savedata.replace(old_export, new_export, 1)
elif new_export not in savedata:
    raise SystemExit("shell_savedata.go export block did not match expected baseline")

old_newest = '''// newestSaveBackup returns the most recently written .aramsave file in
// directory, or an empty path when the folder holds none.
func newestSaveBackup(directory string) (string, error) {
\treturn newestArtifact(directory, ".aramsave")
}
'''
new_newest = '''// newestSaveBackup returns the most recently written save backup in directory.
// New WIPI-compatible exports use .wfs while native ARAM-only saves keep the
// .aramsave extension, so both generations remain shareable from mobile.
func newestSaveBackup(directory string) (string, error) {
\tentries, err := os.ReadDir(directory)
\tif err != nil {
\t\treturn "", err
\t}
\tnames := make([]string, 0, len(entries))
\tfor _, entry := range entries {
\t\tif entry.IsDir() {
\t\t\tcontinue
\t\t}
\t\textension := filepath.Ext(entry.Name())
\t\tif !strings.EqualFold(extension, ".wfs") &&
\t\t\t!strings.EqualFold(extension, ".aramsave") {
\t\t\tcontinue
\t\t}
\t\tnames = append(names, entry.Name())
\t}
\tif len(names) == 0 {
\t\treturn "", nil
\t}
\tsort.Strings(names)
\treturn filepath.Join(directory, names[len(names)-1]), nil
}
'''
if old_newest in savedata:
    savedata = savedata.replace(old_newest, new_newest, 1)
elif new_newest not in savedata:
    raise SystemExit("shell_savedata.go newest backup block did not match expected baseline")

SAVEDATA.write_text(savedata)

picker = PICKER.read_text()
old_patterns = '''func saveBackupPatterns() []string {
\treturn []string{"*.aramsave"}
}
'''
new_patterns = '''func saveBackupPatterns() []string {
\treturn []string{"*.wfs", "*.aramsave"}
}
'''
if old_patterns in picker:
    picker = picker.replace(old_patterns, new_patterns, 1)
elif new_patterns not in picker:
    raise SystemExit("picker.go save backup patterns did not match expected baseline")
PICKER.write_text(picker)

print(f"patched {SAVEDATA} to write WFSAVEBK exports with a .wfs extension")
print(f"patched {PICKER} to accept both .wfs and .aramsave backups")
