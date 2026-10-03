from pathlib import Path
import importlib.util
import re

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core" / "application"
spec = importlib.util.spec_from_file_location(
    "autoidentify_assembler", ROOT / ".github/scripts/assemble_inotia2_autoidentify.py"
)
assembler = importlib.util.module_from_spec(spec)
spec.loader.exec_module(assembler)
definition = (ROOT / ".github/patches/inotia2_autoidentify.go").read_text()
emitted = [
    (int(address, 16), bytes.fromhex(data))
    for address, data in re.findall(
        r'address: (0x[0-9a-f]+), originalSHA: "[0-9a-f]+", replacement: inotia2PatchBytes\("([0-9a-f]+)"\)',
        definition,
    )
]
if emitted != assembler.build():
    raise SystemExit("Auto-identification bytecode differs from authored assembler")

owner = CORE / "inotia2_autoloot.go"
source = owner.read_text()
old = "\treturn installInotia2OfflineShop(client, backend)\n"
new = """\tif err := installInotia2OfflineShop(client, backend); err != nil {
\t\treturn err
\t}
\treturn installInotia2AutoIdentify(client, backend)
"""
if new not in source:
    if source.count(old) != 1:
        raise SystemExit("Local shop installation baseline does not match")
    owner.write_text(source.replace(old, new, 1))

state = CORE / "state.go"
source = state.read_text()
old = "\tcopy(m.frame.Pix, parsed.frame)\n"
new = """\t// v1008/v1012/v1013 states gain the guarded acquisition hook too.
\tif err := installInotia2AutoIdentify(m.initialText, m.cpu); err != nil {
\t\treturn err
\t}
""" + old
if new not in source:
    if source.count(old) != 1:
        raise SystemExit("Full-state restore baseline does not match")
    state.write_text(source.replace(old, new, 1))

for name in ("inotia2_autoidentify.go", "inotia2_autoidentify_test.go",
             "inotia2_autoidentify_native_test.go"):
    (CORE / name).write_text((ROOT / ".github/patches" / name).read_text())
print("Installed guarded equipment identification on accepted acquisitions")
