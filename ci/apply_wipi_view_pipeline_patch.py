#!/usr/bin/env python3
"""Apply the Inotia WIPI-view + latest-frame mailbox experiment.

The build keeps ARAM's normal code pinned, then makes three narrow test changes:

1. The existing 320-wide experiment becomes 320x426.  This preserves the
   handset's 3:4-ish portrait aspect, matching the visible scale/FOV of the
   WIPI Emulator screenshot instead of producing ARAM's square 320x320 view.
2. That WIPI-view path bypasses LCD post-processing and uses nearest sampling,
   matching WIPI Emulator's simple Fit + nearest presentation path.
3. The aram-core backend publishes an immutable latest-frame mailbox after
   every completed guest quantum.  The frontend may consume that mailbox while
   a multi-quantum worker batch is still running instead of holding the old
   picture until the whole batch completes.
"""

from __future__ import annotations

import pathlib
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected source block once, found {count}")
    return text.replace(old, new, 1)


def patch_product(root: pathlib.Path) -> None:
    backend = root / "integration" / "backend.go"
    text = backend.read_text()

    text = replace_once(
        text,
        "\tframeSequence    uint64\n\n\timageSHA256",
        "\tframeSequence    uint64\n\t// latestVideo is an immutable frame mailbox. RunFrame refreshes it after\n"
        "\t// every completed guest quantum so the UI never has to touch a machine\n"
        "\t// that is currently executing the next quantum.\n"
        "\tlatestVideo frontend.VideoFrame\n\n\timageSHA256",
        "backend mailbox field",
    )

    text = replace_once(
        text,
        "\tbackend.lastPresentation = 0\n\tbackend.frameSequence = 0\n\tbackend.cheats = library",
        "\tbackend.lastPresentation = 0\n\tbackend.frameSequence = 0\n\tbackend.latestVideo = frontend.VideoFrame{}\n\tbackend.cheats = library",
        "open mailbox reset",
    )

    text = replace_once(
        text,
        "\tif err := machine.StepFrame(ctx); err != nil {\n"
        "\t\tbackend.setRunRequested(false)\n"
        "\t\treturn backendError(classifyMachineError(machine, err), err)\n"
        "\t}\n"
        "\tif !machineCanContinue(machine.State()) {",
        "\tif err := machine.StepFrame(ctx); err != nil {\n"
        "\t\tbackend.setRunRequested(false)\n"
        "\t\treturn backendError(classifyMachineError(machine, err), err)\n"
        "\t}\n"
        "\t// Publish immediately after each completed quantum. aram-core's\n"
        "\t// VideoPresentation image is immutable, so the UI can safely keep it\n"
        "\t// while this worker advances the following quantum.\n"
        "\tbackend.publishLatestVideo(machine)\n"
        "\tif !machineCanContinue(machine.State()) {",
        "RunFrame mailbox publish",
    )

    old_video = '''func (backend *Backend) VideoFrame() frontend.VideoFrame {
\tmachine := backend.currentMachine()
\tif machine == nil {
\t\treturn frontend.VideoFrame{}
\t}
\tunwrapped := unwrapMachine(machine)
\tif presenter, ok := unwrapped.(coreVideoPresenter); ok {
\t\treturn backend.presentedTimedVideoFrame(presenter)
\t}
\tif presenter, ok := unwrapped.(coreFramePresenter); ok {
\t\treturn backend.presentedVideoFrame(presenter)
\t}
\tframe := machine.Framebuffer()
\tif frame == nil || frame.Bounds().Dx() <= 0 || frame.Bounds().Dy() <= 0 {
\t\treturn frontend.VideoFrame{}
\t}
\tbackend.mu.Lock()
\t// frameChanged always runs, even for the first frame, so the comparison
\t// baseline is recorded before the next tick reads it.
\tchanged := backend.frameChanged(frame)
\tif backend.frameSequence == 0 || changed {
\t\tbackend.frameSequence++
\t}
\tsequence := backend.frameSequence
\tbackend.mu.Unlock()
\treturn frontend.VideoFrame{Image: frame, Sequence: sequence}
}
'''
    new_video = '''// captureVideoFrame reads one completed machine presentation. Callers only use
// this on a machine that is not inside StepFrame; the returned core presentation
// is immutable and can therefore be handed to the UI without another copy.
func (backend *Backend) captureVideoFrame(machine aramcore.Machine) frontend.VideoFrame {
\tif machine == nil {
\t\treturn frontend.VideoFrame{}
\t}
\tunwrapped := unwrapMachine(machine)
\tif presenter, ok := unwrapped.(coreVideoPresenter); ok {
\t\treturn backend.presentedTimedVideoFrame(presenter)
\t}
\tif presenter, ok := unwrapped.(coreFramePresenter); ok {
\t\treturn backend.presentedVideoFrame(presenter)
\t}
\tframe := machine.Framebuffer()
\tif frame == nil || frame.Bounds().Dx() <= 0 || frame.Bounds().Dy() <= 0 {
\t\treturn frontend.VideoFrame{}
\t}
\tbackend.mu.Lock()
\tchanged := backend.frameChanged(frame)
\tif backend.frameSequence == 0 || changed {
\t\tbackend.frameSequence++
\t}
\tsequence := backend.frameSequence
\tbackend.mu.Unlock()
\treturn frontend.VideoFrame{Image: frame, Sequence: sequence}
}

func (backend *Backend) publishLatestVideo(machine aramcore.Machine) {
\tframe := backend.captureVideoFrame(machine)
\tif frame.Image == nil {
\t\treturn
\t}
\tbackend.mu.Lock()
\tbackend.latestVideo = frame
\tbackend.mu.Unlock()
}

func (backend *Backend) VideoFrame() frontend.VideoFrame {
\tbackend.mu.RLock()
\tlatest := backend.latestVideo
\tbackend.mu.RUnlock()
\tif latest.Image != nil {
\t\treturn latest
\t}
\t// Before the first quantum has completed there is no mailbox entry yet.
\t// Read the initial frame directly once; subsequent UI polls are lock-free
\t// with respect to guest execution.
\treturn backend.captureVideoFrame(backend.currentMachine())
}
'''
    text = replace_once(text, old_video, new_video, "VideoFrame mailbox")

    text = replace_once(
        text,
        "\tbackend.lastPresentation = 0\n\tbackend.frameSequence = 0\n\tbackend.cheats = nil",
        "\tbackend.lastPresentation = 0\n\tbackend.frameSequence = 0\n\tbackend.latestVideo = frontend.VideoFrame{}\n\tbackend.cheats = nil",
        "close mailbox reset",
    )

    backend.write_text(text)


def patch_frontend(root: pathlib.Path) -> None:
    display = root / "frontend" / "shell_settings_display.go"
    text = display.read_text()
    text = replace_once(
        text,
        "const nativeGuestHeight = 320\n",
        "const nativeGuestHeight = 320\n\n"
        "// wipiViewHeight pairs 320 logical pixels across with the same portrait\n"
        "// aspect as the original 240x320 handset: 320 / 426 ~= 240 / 320.\n"
        "// This matches the WIPI Emulator field of view instead of making the\n"
        "// guest square like the old 320x320 widescreen experiment.\n"
        "const wipiViewHeight = 426\n",
        "WIPI view height",
    )
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
\t\treturn s.tr("320 x 426 (WIPI view)")
\t}
\treturn s.trf("%d px wide", width)
}
''',
        "WIPI view label",
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
\theight := nativeGuestHeight
\tif width == 320 {
\t\theight = wipiViewHeight
\t}
\treturn DisplaySettings{Width: width, Height: height}
}
''',
        "WIPI view geometry",
    )
    display.write_text(text)

    media = root / "frontend" / "shell_media.go"
    text = media.read_text()
    text = replace_once(
        text,
        "\tif s.frameRunPending || s.loading || len(s.busyCommands) != 0 {\n\t\treturn\n\t}\n",
        "\t// aram-core publishes an immutable latest-frame mailbox after every\n"
        "\t// completed guest quantum. Keep presenting that mailbox while the\n"
        "\t// worker is already computing the next quantum; waiting for the whole\n"
        "\t// batch is what made a late frame look like a visible freeze/jump.\n"
        "\tif s.loading || len(s.busyCommands) != 0 {\n\t\treturn\n\t}\n",
        "frontend pending-frame gate",
    )
    media.write_text(text)

    render = root / "frontend" / "render.go"
    text = render.read_text()
    text = replace_once(
        text,
        "\tdisplay := s.displayProfile()\n\teffect := display.DisplayEffect\n",
        "\tdisplay := s.displayProfile()\n\teffect := display.DisplayEffect\n"
        "\t// WIPI Emulator presents the guest with a single nearest-neighbour Fit\n"
        "\t// pass. Mirror that path for the 320x426 WIPI-view preset and avoid\n"
        "\t// temporal/LCD shaders competing with guest emulation for frame time.\n"
        "\tif s.settings.GuestWidthOverride == 320 {\n\t\teffect = displayEffectOff\n\t}\n",
        "WIPI view effect bypass",
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
        "WIPI view nearest filter",
    )
    render.write_text(text)


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit("usage: apply_wipi_view_pipeline_patch.py <aram-emu-dir> <aram-frontend-dir>")
    product = pathlib.Path(sys.argv[1]).resolve()
    frontend = pathlib.Path(sys.argv[2]).resolve()
    patch_product(product)
    patch_frontend(frontend)
    print("Applied WIPI-view geometry and latest-frame mailbox patches")


if __name__ == "__main__":
    main()
