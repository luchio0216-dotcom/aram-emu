from pathlib import Path
import importlib
import re

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / 'aram-core' / 'application'
for feature in ('geometry', 'targets'):
    assembler = importlib.import_module('assemble_inotia2_combat_' + feature)
    definition = (ROOT / '.github/patches' / ('inotia2_combat_' + feature + '.go')).read_text()
    emitted = [(int(a,16),bytes.fromhex(b)) for a,b in re.findall(
        r'address: (0x[0-9a-f]+), originalSHA: "[0-9a-f]+", replacement: inotia2PatchBytes\("([0-9a-f]+)"\)', definition)]
    if emitted != assembler.build():
        raise SystemExit('Combat ' + feature + ' differs from authored assembly')

owner = CORE / 'inotia2_autoloot.go'
source = owner.read_text()
old = '\treturn installInotia2AutoIdentify(client, backend)\n'
new = '''\tif err := installInotia2AutoIdentify(client, backend); err != nil {
\t\treturn err
\t}
\tif err := installInotia2CombatGeometry(client, backend); err != nil {
\t\treturn err
\t}
\treturn installInotia2CombatTargets(client, backend)
'''
if new not in source:
    if source.count(old) != 1:
        raise SystemExit('Combat helper install baseline mismatch')
    owner.write_text(source.replace(old,new,1))
state = CORE / 'state.go'
source = state.read_text()
old = '\tcopy(m.frame.Pix, parsed.frame)\n'
new = '''\t// Older full states gain the two exact-client combat fixes.
\tif err := installInotia2CombatGeometry(m.initialText, m.cpu); err != nil { return err }
\tif err := installInotia2CombatTargets(m.initialText, m.cpu); err != nil { return err }
''' + old
if new not in source:
    if source.count(old) != 1:
        raise SystemExit('Combat full-state restore baseline mismatch')
    state.write_text(source.replace(old,new,1))
for feature in ('geometry','targets'):
    for suffix in ('.go','_test.go','_native_test.go'):
        name = 'inotia2_combat_' + feature + suffix
        (CORE / name).write_text((ROOT / '.github/patches' / name).read_text())
print('Installed guarded missing-target fallback and equivalent combat geometry')
