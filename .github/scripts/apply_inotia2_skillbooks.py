from pathlib import Path
import re
from assemble_inotia2_skillbooks import build

ROOT=Path(__file__).resolve().parents[2]
CORE=ROOT.parent/'aram-core'/'application'
definition=(ROOT/'.github/patches/inotia2_skillbooks.go').read_text()
emitted=[(int(a,16),bytes.fromhex(b)) for a,b in re.findall(
 r'address: (0x[0-9a-f]+), originalSHA: "[0-9a-f]+", recentSHA: "[0-9a-f]+", replacement: inotia2PatchBytes\("([0-9a-f]+)"\)',definition)]
if emitted!=build():raise SystemExit('Class skillbook helpers differ from authored assembly')

owner=CORE/'inotia2_autoloot.go';source=owner.read_text()
old='\treturn installInotia2Experience(client, backend)\n'
new='''\tif err := installInotia2Experience(client, backend); err != nil { return err }
\treturn installInotia2SkillBooks(client, backend)
'''
if "installInotia2SkillBooks(client, backend)" not in source:
 if source.count(old)!=1:raise SystemExit('Class skillbook installer baseline mismatch')
 owner.write_text(source.replace(old,new,1))

owner=CORE/'state.go';source=owner.read_text()
old='\tcopy(m.frame.Pix, parsed.frame)\n'
new='''\t// Existing full states receive the class book hooks; table extension is lazy.
\tif err := installInotia2SkillBooks(m.initialText, m.cpu); err != nil { return err }
'''+old
if "installInotia2SkillBooks(m.initialText, m.cpu)" not in source:
 if source.count(old)!=1:raise SystemExit('Class skillbook state baseline mismatch')
 owner.write_text(source.replace(old,new,1))
for suffix in ('.go','_test.go','_native_test.go'):
 name='inotia2_skillbooks'+suffix
 (CORE/name).write_text((ROOT/'.github/patches'/name).read_text())
print('Installed six class sealed skillbooks; native IDs, capacity and saves retained')

