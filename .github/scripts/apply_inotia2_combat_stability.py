from pathlib import Path
import json

ROOT = Path(__file__).resolve().parents[2]
scopes = {"core": ROOT.parent / "aram-core", "frontend": ROOT.parent / "aram-frontend"}
changes = json.loads((ROOT / ".github/patches/inotia2_combat_stability.json").read_text())
# Prepare and validate every file before making any change. The native-only CI
# checkout has no frontend; the product jobs always check out both siblings.
prepared = []
for change in changes:
    scope = scopes[change["scope"]]
    if not scope.is_dir():
        continue
    path = scope / change["path"]
    source = path.read_text()
    for hunk in change["hunks"]:
        if hunk["after"] in source:
            continue
        if source.count(hunk["before"]) != 1:
            raise SystemExit(f"unexpected combat-stability baseline in {path}")
        source = source.replace(hunk["before"], hunk["after"], 1)
    prepared.append((path, source))
for scope, path, filename in (
    ("core", "cpu/interpreter/native_indirect_arm64_test.go", "native_indirect_arm64_test.go"),
    ("frontend", "frontend/frame_batch_test.go", "frame_batch_test.go"),
):
    if scopes[scope].is_dir():
        prepared.append((scopes[scope] / path, (ROOT / ".github/patches" / filename).read_text()))
for path, source in prepared:
    path.write_text(source)
print("Bounded ARM64 indirect gates and time-limited frame batches applied")
