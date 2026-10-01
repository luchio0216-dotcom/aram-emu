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
for name in ["inotia2_autoloot.go", "inotia2_autoloot_test.go"]:
    (CORE / name).write_text((ROOT / ".github" / "patches" / name).read_text())
print("Installed hash-guarded Inotia2 automatic pickup with unchanged ZIP save identity")
