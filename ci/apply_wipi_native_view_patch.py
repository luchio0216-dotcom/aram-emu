#!/usr/bin/env python3
"""Apply the Inotia1 WIPI-Emulator-style portrait field-of-view preset.

Important ARAM/KTF quirk found from the previous test build:
- asking the frontend for 320x426 produced a *landscape* guest surface;
- the KTF presentation path effectively reaches the title with the axes swapped.

For the WIPI-like portrait target we therefore request 426x320 from ARAM.  The
resulting KTF title-visible geometry is the 320x426-class portrait view the user
is after: the same ~3:4 aspect as the original 240x320 handset, but 4/3 more
logical pixels in each axis, so HUD/world objects become ~25% smaller and more
world is visible without the square 320x320 over-wide look.

Permanent behavior of this preset:
- the saved experimental choice remains the existing 320 sentinel so old test
  installs migrate without resetting settings;
- that sentinel now requests DisplaySettings 426x320;
- normal ARAM top chrome/menu remains visible;
- LCD post-processing is bypassed and nearest-neighbour sampling is used;
- input and save-data code are deliberately untouched here.
"""

from __future__ import annotations

import pathlib
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected one source block, found {count}")
    return text.replace(old, new, 1)


def patch_frontend(root: pathlib.Path) -> None:
    display = root / "frontend" / "shell_settings_display.go"
    text = display.read_text()

    text = replace_once(
        text,
        '''func (s *Shell) widescreenLabel(width int) string {
\tif width <= 0 {
\t\treturn s.tr("Off (native width)")
\t}
\treturn s.trf("%d px wide", width)
}
''',
        '''func (s *Shell) widescreenLabel(width int) string {
\tif width <= 0 {
\t\treturn s.tr("Off (native width)")
\t}
\tif width == 320 {
\t\treturn s.tr("WIPI view (426 x 320)")
\t}
\treturn s.trf("%d px wide", width)
}
''',
        "WIPI portrait label",
    )

    text = replace_once(
        text,
        '''func (s *Shell) currentDisplaySettings() DisplaySettings {
\twidth := s.settings.GuestWidthOverride
\tif width <= 0 {
\t\treturn DisplaySettings{}
\t}
\treturn DisplaySettings{Width: width, Height: nativeGuestHeight}
}
''',
        '''func (s *Shell) currentDisplaySettings() DisplaySettings {
\twidth := s.settings.GuestWidthOverride
\tif width <= 0 {
\t\treturn DisplaySettings{}
\t}
\t// KTF's handset/presentation orientation reaches this title with the
\t// requested axes swapped.  426x320 here therefore gives the game the
\t// portrait 320x426-class field of view seen in WIPI Emulator, while the
\t// previous 320x426 request produced an unintended landscape view.
\tif width == 320 {
\t\treturn DisplaySettings{Width: 426, Height: nativeGuestHeight}
\t}
\treturn DisplaySettings{Width: width, Height: nativeGuestHeight}
}
''',
        "WIPI portrait guest geometry",
    )
    display.write_text(text)

    render = root / "frontend" / "render.go"
    text = render.read_text()

    # Keep ARAM's ordinary chrome/menu path. Only make the guest image use the
    # largest aspect-preserving Fit calculation in its normal workspace.
    text = replace_once(
        text,
        '''func (s *Shell) drawGuestViewport(screen *ebiten.Image, viewport image.Rectangle) {
\tpalette := defaultARAMPalette()
''',
        '''func (s *Shell) drawGuestViewport(screen *ebiten.Image, viewport image.Rectangle) {
\trestoreFill := s.fillGuestViewport
\tif s.settings.GuestWidthOverride == 320 {
\t\t// WIPI-view keeps the normal ARAM app bar/menu visible and changes only
\t\t// guest presentation. Do not route through drawImmersiveWorkspace.
\t\ts.fillGuestViewport = true
\t}
\tdefer func() { s.fillGuestViewport = restoreFill }()

\tpalette := defaultARAMPalette()
''',
        "WIPI fit inside normal chrome",
    )

    text = replace_once(
        text,
        '''\tdisplay := s.displayProfile()
\teffect := display.DisplayEffect
''',
        '''\tdisplay := s.displayProfile()
\teffect := display.DisplayEffect
\tif s.settings.GuestWidthOverride == 320 {
\t\t// Match WIPI Emulator's cheap direct presentation path.
\t\teffect = displayEffectOff
\t}
''',
        "WIPI display effect bypass",
    )

    text = replace_once(
        text,
        '''\tif display.Filter == "linear" {
\t\toptions.Filter = ebiten.FilterLinear
\t} else {
\t\toptions.Filter = ebiten.FilterNearest
\t}
''',
        '''\tif s.settings.GuestWidthOverride == 320 {
\t\toptions.Filter = ebiten.FilterNearest
\t} else if display.Filter == "linear" {
\t\toptions.Filter = ebiten.FilterLinear
\t} else {
\t\toptions.Filter = ebiten.FilterNearest
\t}
''',
        "WIPI nearest filter",
    )
    render.write_text(text)


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: apply_wipi_native_view_patch.py <aram-frontend-dir>")
    patch_frontend(pathlib.Path(sys.argv[1]).resolve())
    print("Applied WIPI portrait view: ARAM request 426x320, normal chrome, nearest presentation")


if __name__ == "__main__":
    main()
