from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core" / "application"
loader = CORE / "machine_load.go"
source = loader.read_text()
old = """	pkg ktf.Package,
) error {
	requiredMemory := uint64(len(pkg.Client)) +
"""
new = """	pkg ktf.Package,
) error {
	// Keep the user's ZIP digest and save identity unchanged. Only the exact
	// validated Inotia2 Rare12 native client receives automatic ground pickup.
	if pkg.Descriptor.AID == "010100D5" {
		client, err := inotia2AutoLootClient(pkg.Client)
		if err != nil { return err }
		pkg.Client = client
	}
	requiredMemory := uint64(len(pkg.Client)) +
"""
if new not in source:
    if source.count(old) != 1:
        raise SystemExit("KTF client loader baseline did not match")
    loader.write_text(source.replace(old, new, 1))
source = loader.read_text()
old_map = "\truntime.DeferThreads = true\n"
new_map = "\truntime.ImageSz, err = inotia2AutoLootMappedSize(pkg.Client, runtime.ImageSz)\n\tif err != nil { return err }\n\truntime.DeferThreads = true\n"
if new_map not in source:
    if source.count(old_map) != 1: raise SystemExit("KTF mapping baseline mismatch")
    loader.write_text(source.replace(old_map, new_map, 1))
source = loader.read_text()
old_install = "\tresult, executable, err := runtime.Bootstrap(ctx)\n"
new_install = "\tif err := installInotia2AutoLootHelpers(pkg.Client, runtime.CPU); err != nil { return err }\n" + old_install
if new_install not in source:
    if source.count(old_install) != 1: raise SystemExit("KTF helper installation baseline mismatch")
    loader.write_text(source.replace(old_install, new_install, 1))
reset = CORE / "machine.go"
source = reset.read_text()
old_reset_map = "\t\truntime.DeferThreads = true\n\t\tif err := runtime.ResetMappedMemory(); err != nil {"
new_reset_map = """\t\truntime.ImageSz, err = inotia2AutoLootMappedSize(pkg.Client, runtime.ImageSz)
\t\tif err != nil { m.state = machinecore.StateFaulted; return err }
\t\truntime.DeferThreads = true
\t\tif err := runtime.ResetMappedMemory(); err != nil {"""
if new_reset_map not in source:
    if source.count(old_reset_map) != 1: raise SystemExit("KTF reset mapping baseline mismatch")
    source = source.replace(old_reset_map, new_reset_map, 1)
old_reset_install = "\t\tresult, executable, err := runtime.Bootstrap(ctx)\n"
new_reset_install = "\t\tif err := installInotia2AutoLootHelpers(pkg.Client, runtime.CPU); err != nil { m.state = machinecore.StateFaulted; return err }\n" + old_reset_install
if new_reset_install not in source:
    if source.count(old_reset_install) != 1: raise SystemExit("KTF reset helper baseline mismatch")
    source = source.replace(old_reset_install, new_reset_install, 1)
reset.write_text(source)
for name in ["inotia2_autoloot.go", "inotia2_autoloot_test.go", "inotia2_discard_test.go", "inotia2_discard_native_test.go"]:
    (CORE / name).write_text((ROOT / ".github" / "patches" / name).read_text())
print("Installed hash-guarded Inotia2 automatic pickup with unchanged ZIP save identity")
