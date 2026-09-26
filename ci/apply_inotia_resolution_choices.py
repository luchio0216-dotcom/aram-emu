#!/usr/bin/env python3
"""Make ARAM mimic the WIPI Emulator geometry for this Inotia1 KTF build.

Permanent base (direction fix + WFS save/restore) is applied by the workflow
before this script and must remain byte-for-byte unchanged.

The previous frontend-only Y test could never work: aram-emu only forwarded
GuestWidthOverride into aram-core, while the KTF loader always restored the
package descriptor's native height. This patch fixes the geometry where KTF
actually chooses its framebuffer size.
"""

from pathlib import Path
import sys

if len(sys.argv) != 3:
    raise SystemExit("usage: apply_inotia_resolution_choices.py <aram-core-dir> <aram-frontend-dir>")

core_root = Path(sys.argv[1]).resolve()
frontend_root = Path(sys.argv[2]).resolve()

# The user's Inotia1 KTF package is AID 010100D3 and declares 176x220 in
# __adf__. The reference WIPI Emulator ignores that host-screen geometry and
# always provides a 240x320 platform. Reproduce that exact behavior for this
# title before NewRuntimeForProfile receives the frame.
quirks = core_root / "application" / "internal" / "ktf" / "ktf_title_quirks.go"
text = quirks.read_text()
needle = '''func resolveKTFDisplaySize(
\tdescriptor ktf.Descriptor,
\tclientHash [sha256.Size]byte,
) (int, int) {
'''
replacement = '''func resolveKTFDisplaySize(
\tdescriptor ktf.Descriptor,
\tclientHash [sha256.Size]byte,
) (int, int) {
\t// Inotia1 (AID 010100D3) declares a 176x220 handset display, but the
\t// WIPI Emulator mobile host deliberately exposes a fixed 240x320 screen.
\t// Match that host geometry exactly so the game gets the same vertical FOV.
\tif descriptor.AID == "010100D3" &&
\t\tdescriptor.DisplayWidth == 176 && descriptor.DisplayHeight == 220 {
\t\treturn 240, 320
\t}
'''
if needle not in text:
    raise SystemExit("resolveKTFDisplaySize source did not match pinned core")
text = text.replace(needle, replacement, 1)
quirks.write_text(text)

# Disable the old width experiment for this test build. This also neutralizes
# any 260/320/426 value persisted by earlier APKs with the same package id.
settings = frontend_root / "frontend" / "shell_settings_display.go"
text = settings.read_text()
old_choices = '''func widescreenChoices() []int {
\treturn []int{0, 320, 384, 480, 640}
}
'''
new_choices = '''func widescreenChoices() []int {
\treturn []int{0}
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
\treturn s.tr("WIPI fixed 240 x 320")
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
\t// The KTF core applies the Inotia-specific WIPI 240x320 geometry. Ignore
\t// stale width presets from previous experimental APKs.
\treturn DisplaySettings{}
}
'''
if old_display not in text:
    raise SystemExit("currentDisplaySettings source did not match pinned frontend")
text = text.replace(old_display, new_display, 1)
settings.write_text(text)

print(f"patched WIPI 240x320 KTF geometry in {quirks}")
print(f"disabled stale width overrides in {settings}")
