from pathlib import Path
import json

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core"
PATCHES = ROOT / ".github/patches"
prepared = []
for change in json.loads((PATCHES / "inotia2_continuous_music.json").read_text()):
    path = CORE / change["path"]
    source = path.read_text()
    for hunk in change["hunks"]:
        if source.count(hunk["after"]) == 1:
            continue
        if source.count(hunk["before"]) != 1:
            raise SystemExit(f"unexpected continuous music baseline in {path}")
        source = source.replace(hunk["before"], hunk["after"], 1)
    prepared.append((path, source))
for folder, names in (
    ("runtime", ("media_continuous_music.go", "media_continuous_music_test.go")),
    ("application", ("inotia2_continuous_music.go", "inotia2_continuous_music_test.go")),
):
    for name in names:
        prepared.append((CORE / folder / name, (PATCHES / name).read_text()))
for path, source in prepared:
    path.write_text(source)
print("Verified Inotia2 music continues independently of effects and menu clips")
