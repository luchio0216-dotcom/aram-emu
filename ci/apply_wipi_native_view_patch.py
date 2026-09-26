#!/usr/bin/env python3
"""Apply an Inotia1 WIPI-Emulator-style presentation preset.

Important: WIPI Emulator itself renders a native 240x320 framebuffer and only
scales that image to the available phone display area.  It does NOT widen the
guest framebuffer.  ARAM's existing 320-wide experiment changes the guest's
reported geometry, so the game exposes more world horizontally.

This patch reuses the 320 experiment menu slot as a presentation-only sentinel:
- guest geometry stays native 240x320;
- the touch build uses the immersive/fill viewport above the keypad;
- aspect ratio is preserved;
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
\t// 320 is the WIPI-view presentation preset in the Android compatibility
\t// build. Keep the guest at its title-native 240x320 geometry and only make
\t// the host presentation fill the available display area.
\tif width == 320 {
\t\treturn DisplaySettings{}
\t}
\treturn DisplaySettings{Width: width, Height: nativeGuestHeight}
}
''',
        "native guest geometry",
    )
    display.write_text(text)

    shell = root / "frontend" / "shell.go"
    text = shell.read_text()
    text = replace_once(
        text,
        '''\tif s.touchChromeHiddenActive() {
\t\ts.drawImmersiveWorkspace(screen)
\t\ts.drawTouchControls(screen)
\t\ts.drawTouchChromeToggle(screen)
\t\treturn
\t}
''',
        '''\tif s.touchChromeHiddenActive() || s.settings.GuestWidthOverride == 320 {
\t\t// WIPI Emulator keeps a native 240x320 guest and Fits it into all
\t\t// available space above the keypad. Reuse ARAM's immersive viewport to
\t\t// reproduce that presentation without changing the guest FOV.
\t\ts.drawImmersiveWorkspace(screen)
\t\ts.drawTouchControls(screen)
\t\ts.drawTouchChromeToggle(screen)
\t\treturn
\t}
''',
        "immersive WIPI presentation",
    )
    shell.write_text(text)

    render = root / "frontend" / "render.go"
    text = render.read_text()
    text = replace_once(
        text,
        '''\tdisplay := s.displayProfile()
\teffect := display.DisplayEffect
''',
        '''\tdisplay := s.displayProfile()
\teffect := display.DisplayEffect
\tif s.settings.GuestWidthOverride == 320 {
\t\t// WIPI Emulator presents the framebuffer directly with nearest-neighbour
\t\t// Fit. Avoid extra LCD/temporal shader passes in this compatibility mode.
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
    print("Applied native 240x320 WIPI-view presentation patch")


if __name__ == "__main__":
    main()
