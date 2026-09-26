#!/usr/bin/env python3
"""Patch only ARAM's experimental guest-width choices for Inotia1 testing.

IMPORTANT: This script must not touch input, save-data, graphics-core, frame
scheduling, or Android controls. The product branch itself is the known-working
Inotia1 direction + WFS base; this patch only changes the list/labels of guest
framebuffer widths exposed by the existing widescreen experiment.
"""

from pathlib import Path
import sys

if len(sys.argv) != 2:
    raise SystemExit("usage: apply_inotia_resolution_choices.py <aram-frontend-dir>")

root = Path(sys.argv[1]).resolve()
path = root / "frontend" / "shell_settings_display.go"
text = path.read_text()

old_choices = '''func widescreenChoices() []int {
\treturn []int{0, 320, 384, 480, 640}
}
'''
new_choices = '''func widescreenChoices() []int {
\t// Inotia1 field-of-view test widths. Keep native 240x320 available as Off,
\t// then increase width gradually so we can find the WIPI-Emulator-like view
\t// without jumping straight to the much heavier 320+ modes.
\treturn []int{0, 260, 280, 300, 320}
}
'''
if old_choices not in text:
    raise SystemExit("widescreenChoices source did not match pinned frontend")
text = text.replace(old_choices, new_choices, 1)

old_label = '''func (s *Shell) widescreenLabel(width int) string {
\tif width <= 0 {
\t\treturn s.tr("Off (native width)")
\t}
\treturn s.trf("%d px wide", width)
}
'''
new_label = '''func (s *Shell) widescreenLabel(width int) string {
\tif width <= 0 {
\t\treturn s.tr("Off (native 240 x 320)")
\t}
\treturn s.trf("%d x 320 test", width)
}
'''
if old_label not in text:
    raise SystemExit("widescreenLabel source did not match pinned frontend")
text = text.replace(old_label, new_label, 1)

path.write_text(text)
print(f"patched only resolution choices in {path}")
