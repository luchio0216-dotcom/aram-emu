#!/usr/bin/env python3
from pathlib import Path
import sys

if len(sys.argv) != 2:
    raise SystemExit("usage: apply_inotia_smooth_pacing.py <aram-frontend-dir>")

root = Path(sys.argv[1])
pacing = root / "frontend" / "pacing.go"
text = pacing.read_text()

old = "\tframePacingQuantaPerTick = 8\n"
new = "\tframePacingQuantaPerTick = 1\n"
if old not in text:
    if new not in text:
        raise SystemExit("framePacingQuantaPerTick baseline not found")
else:
    text = text.replace(old, new, 1)

# This specialized Inotia build must never add the optional 3ms sleep after a
# completed guest quantum.  The user's goal is smooth 60 Hz presentation, and
# UI priority's deliberate rest turns a borderline 15-16 ms quantum into a
# missed display deadline.
old_req = "\t\tuiPriority: s.settings.UIPriority,\n"
new_req = "\t\tuiPriority: false, // Inotia smooth build: never inject 3ms guest sleep\n"
if old_req not in text:
    if new_req not in text:
        raise SystemExit("frame request uiPriority baseline not found")
else:
    text = text.replace(old_req, new_req, 1)

pacing.write_text(text)
print("patched pacing: one guest quantum per UI tick, no deliberate worker sleep")
