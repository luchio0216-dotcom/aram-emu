#!/usr/bin/env python3
"""Patch only ARAM's experimental display geometry for Inotia1 Y-axis testing.

IMPORTANT: This script must not touch input, save-data, graphics-core, frame
scheduling, or Android controls. The branch is built from the user-confirmed
Inotia1 direction + WFS base. This patch only reuses the existing experimental
setting value as a height preset while holding guest width at 240.
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
\t// Inotia1 vertical field-of-view test. Zero preserves the exact native
\t// 240x320 device. Non-zero values are interpreted as HEIGHT presets while
\t// width stays fixed at 240, so this experiment expands Y only.
\treturn []int{0, 340, 360, 384, 400, 426}
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
new_label = '''func (s *Shell) widescreenLabel(height int) string {
\tif height <= 0 {
\t\treturn s.tr("Off (native 240 x 320)")
\t}
\treturn s.trf("240 x %d Y test", height)
}
'''
if old_label not in text:
    raise SystemExit("widescreenLabel source did not match pinned frontend")
text = text.replace(old_label, new_label, 1)

old_display = '''func (s *Shell) currentDisplaySettings() DisplaySettings {
\twidth := s.settings.GuestWidthOverride
\tif width <= 0 {
\t\treturn DisplaySettings{}
\t}
\treturn DisplaySettings{Width: width, Height: nativeGuestHeight}
}
'''
new_display = '''func (s *Shell) currentDisplaySettings() DisplaySettings {
\theight := s.settings.GuestWidthOverride
\tif height <= 0 {
\t\treturn DisplaySettings{}
\t}
\t// Y-axis experiment: keep the original 240-pixel guest width and only
\t// extend the logical height. The saved setting field keeps its historical
\t// name for compatibility; in this test build its value is a height preset.
\treturn DisplaySettings{Width: 240, Height: height}
}
'''
if old_display not in text:
    raise SystemExit("currentDisplaySettings source did not match pinned frontend")
text = text.replace(old_display, new_display, 1)

path.write_text(text)
print(f"patched only Y-axis resolution choices in {path}")
