#!/usr/bin/env python3
"""Apply an Inotia1 WIPI-Emulator-style presentation preset.

WIPI Emulator itself keeps the guest framebuffer native 240x320 and only Fits
that image into the available phone UI. This patch mirrors that presentation
without forcing ARAM into immersive chrome-hidden mode.

Permanent behavior of this preset:
- guest geometry stays native 240x320;
- normal ARAM top chrome/menu remains visible;
- only the guest image uses largest aspect-preserving Fit inside its normal
  viewport;
- LCD post-processing is bypassed and nearest-neighbour sampling is used.

No input or save-data code is touched here.
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
\t\treturn s.tr("WIPI view (native 240 x 320)")
\t}
\treturn s.trf("%d px wide", width)
}
''',
        "WIPI native label",
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
\t// 320 is only a presentation sentinel in this compatibility build. The
\t// title still sees its native 240x320 handset framebuffer.
\tif width == 320 {
\t\treturn DisplaySettings{}
\t}
\treturn DisplaySettings{Width: width, Height: nativeGuestHeight}
}
''',
        "native guest geometry",
    )
    display.write_text(text)

    render = root / "frontend" / "render.go"
    text = render.read_text()

    # Keep ARAM's ordinary chrome/menu path. Only make the guest image itself
    # use the same largest aspect-preserving Fit calculation as immersive mode.
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
\t\t// WIPI Emulator presents the native framebuffer directly.
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
    print("Applied native 240x320 WIPI-view presentation with normal ARAM chrome")


if __name__ == "__main__":
    main()
