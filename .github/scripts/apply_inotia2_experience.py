from pathlib import Path
import re
from assemble_inotia2_experience import build

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / 'aram-core' / 'application'
definition = (ROOT / '.github/patches/inotia2_experience.go').read_text()
emitted = [(int(a,16),bytes.fromhex(b)) for a,b in re.findall(
    r'address: (0x[0-9a-f]+), originalSHA: "[0-9a-f]+", recentSHA: "[0-9a-f]+", replacement: inotia2PatchBytes\("([0-9a-f]+)"\)', definition)]
if emitted != build():
    raise SystemExit('Experience wrapper differs from authored assembly')

owner = CORE / 'inotia2_autoloot.go'
source = owner.read_text()
old = '\treturn installInotia2CombatTargets(client, backend)\n'
new = '''\tif err := installInotia2CombatTargets(client, backend); err != nil {
\t\treturn err
\t}
\treturn installInotia2Experience(client, backend)
'''
if new not in source:
    if source.count(old) != 1:
        raise SystemExit('Experience helper install baseline mismatch')
    owner.write_text(source.replace(old,new,1))

state = CORE / 'state.go'
source = state.read_text()
old = '\tcopy(m.frame.Pix, parsed.frame)\n'
new = '''\t// Existing full states receive the same final monster-reward hook.
\tif err := installInotia2Experience(m.initialText, m.cpu); err != nil { return err }
''' + old
if new not in source:
    if source.count(old) != 1:
        raise SystemExit('Experience full-state restore baseline mismatch')
    state.write_text(source.replace(old,new,1))

for suffix in ('.go','_test.go','_native_test.go'):
    name = 'inotia2_experience' + suffix
    (CORE / name).write_text((ROOT / '.github/patches' / name).read_text())
print('Installed final monster experience tiers 41:4x,51:5x,61:6x,71:8x,81:10x,91:12x; quests unchanged')
