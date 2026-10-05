from pathlib import Path
import importlib.util
import re

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core" / "application"
spec = importlib.util.spec_from_file_location(
    "local_shop_assembler", ROOT / ".github/scripts/assemble_inotia2_offline_shop.py"
)
assembler = importlib.util.module_from_spec(spec)
spec.loader.exec_module(assembler)
definition = (ROOT / ".github/patches/inotia2_offline_shop.go").read_text()
emitted = [
    (int(address, 16), bytes.fromhex(data))
    for address, data in re.findall(
        r'address: (0x[0-9a-f]+), originalSHA: "[0-9a-f]+", legacySHA: "[0-9a-f]+", previousSHA: "[0-9a-f]+", recentSHA: "[0-9a-f]+", replacement: inotia2PatchBytes\("([0-9a-f]+)"\)',
        definition,
    )
]
if emitted != assembler.build():
    raise SystemExit("Local shop bytecode differs from the authored assembler")

owner = CORE / "inotia2_autoloot.go"
source = owner.read_text()
old = "\treturn backend.WriteMemory(inotia2DiscardHelperAddress, inotia2DiscardHelper)\n"
new = """\tif err := backend.WriteMemory(inotia2DiscardHelperAddress, inotia2DiscardHelper); err != nil {
\t\treturn err
\t}
\treturn installInotia2OfflineShop(client, backend)
"""
if "installInotia2OfflineShop(client, backend)" not in source:
    if source.count(old) != 1:
        raise SystemExit("v1008 helper installation baseline does not match")
    owner.write_text(source.replace(old, new, 1))
for name in (
    "inotia2_offline_shop.go",
    "inotia2_offline_shop_test.go",
    "inotia2_offline_shop_native_test.go",
):
    (CORE / name).write_text((ROOT / ".github/patches" / name).read_text())
print("Installed guarded four-category 248-product local gold shop; native transactions and v1019 state compatibility retained")

state = CORE / "state.go"
source = state.read_text()
old_state = "\tcopy(m.frame.Pix, parsed.frame)\n"
new_state = """\t// Full states from v1008 and v1012 receive the current title-scoped shop.
\tif err := installInotia2OfflineShop(m.initialText, m.cpu); err != nil {
\t\treturn err
\t}
""" + old_state
if "installInotia2OfflineShop(m.initialText, m.cpu)" not in source:
    if source.count(old_state) != 1:
        raise SystemExit("Full-state restore baseline does not match")
    state.write_text(source.replace(old_state, new_state, 1))

