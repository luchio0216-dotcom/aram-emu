from pathlib import Path
import json

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core"
PATCHES = ROOT / ".github/patches"
prepared = []
for change in json.loads((PATCHES / "audio_responsiveness.json").read_text()):
    path = CORE / change["path"]
    source = path.read_text()
    for hunk in change["hunks"]:
        if hunk["after"] in source:
            continue
        if source.count(hunk["before"]) != 1:
            raise SystemExit(f"unexpected audio responsiveness baseline in {path}")
        source = source.replace(hunk["before"], hunk["after"], 1)
    prepared.append((path, source))
for name in ("media_smaf_blocks.go", "media_smaf_blocks_test.go"):
    prepared.append((CORE / "runtime" / name, (PATCHES / name).read_text()))
for path, source in prepared:
    path.write_text(source)
print("Incremental music uses fixed PCM blocks without moving the played prefix")
