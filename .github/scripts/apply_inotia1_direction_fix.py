from pathlib import Path

TARGET = Path("integration/backend.go")
INOTIA1_SHA256 = "7a1bc46d6421af4f310f040be55d2e1e2eac4b70f41c413f0f2a9e6737bd0d91"

old = '''func (backend *Backend) QueueInput(event frontend.InputEvent) error {
\tmachine := backend.currentMachine()
\tif machine == nil {
\t\treturn backendError(
\t\t\tfrontend.FailureBackendUnavailable,
\t\t\terrors.New("no aram-core machine is loaded"),
\t\t)
\t}
\treturn machine.QueueInput(aramcore.InputEvent{
\t\tControl: event.Control,
\t\tPressed: event.Pressed,
\t\tAt:      event.At,
\t})
}
'''

new = f'''func (backend *Backend) QueueInput(event frontend.InputEvent) error {{
\tmachine := backend.currentMachine()
\tif machine == nil {{
\t\treturn backendError(
\t\t\tfrontend.FailureBackendUnavailable,
\t\t\terrors.New("no aram-core machine is loaded"),
\t\t)
\t}}
\t// Inotia 1 (KT/WIPI 1.2, AID 010100D3) ships a Clet$CletCard whose
\t// keyNotify ABI is the regular Card ABI: 1=press, 2=release. aram-core's
\t// generic Clet compatibility path reverses those edges for Clets built with
\t// the other KTF SDK ABI (2=press, 1=release). For this exact title that
\t// turns a physical release into another native key press, so movement stays
\t// latched after the player lifts a direction. Reverse only the four
\t// directional host edges here; the core then emits the regular keyNotify
\t// pair this title expects while every other title/control is unchanged.
\tif backend.inotia1RegularDirectionEdges(event.Control) {{
\t\tevent.Pressed = !event.Pressed
\t}}
\treturn machine.QueueInput(aramcore.InputEvent{{
\t\tControl: event.Control,
\t\tPressed: event.Pressed,
\t\tAt:      event.At,
\t}})
}}

func (backend *Backend) inotia1RegularDirectionEdges(control string) bool {{
\tswitch control {{
\tcase "up", "down", "left", "right":
\tdefault:
\t\treturn false
\t}}
\tbackend.mu.RLock()
\thash := backend.input.SHA256
\tbackend.mu.RUnlock()
\treturn strings.EqualFold(hash, "{INOTIA1_SHA256}")
}}
'''

text = TARGET.read_text()
if new in text:
    raise SystemExit("Inotia1 input compatibility patch is already applied")
if old not in text:
    raise SystemExit("QueueInput source did not match the expected baseline")
TARGET.write_text(text.replace(old, new, 1))
print(f"patched {TARGET} for Inotia1 directional Clet edge compatibility")
