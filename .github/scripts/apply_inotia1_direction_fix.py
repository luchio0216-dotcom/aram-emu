from pathlib import Path

SCHEDULER = Path("../aram-core/application/internal/ktf/ktf_scheduler.go")
TESTS = Path("../aram-core/application/internal/ktf/ktf_clet_input_test.go")
INOTIA1_AID = "010100D3"

scheduler = SCHEDULER.read_text()

# Inotia1 on the reference WIPI emulator receives ordinary WIPI Card edges:
# keyNotify(1, key) on press and keyNotify(2, key) on release.  Keep the host's
# real Pressed state intact and translate at the guest ABI boundary only.
old_call = "eventType, err := r.keyEventType(card, pressed)"
new_call = "eventType, err := r.keyEventType(card, key, pressed)"
if old_call not in scheduler and new_call not in scheduler:
    raise SystemExit("QueueKeyEvent call site did not match the expected baseline")
scheduler = scheduler.replace(old_call, new_call, 1)

old_func = '''func (r *Runtime) keyEventType(card uint32, pressed bool) (uint32, error) {
\teventType := KeyReleased
\tif pressed {
\t\teventType = KeyPressed
\t}
\tcardWords, err := r.ReadWords(card, 2)
\tif err != nil {
\t\treturn 0, err
\t}
\tclass, err := r.InspectJavaClass(cardWords[1])
\tif err != nil {
\t\treturn 0, err
\t}
\tif class.Name == "Clet$CletCard" {
\t\tif pressed {
\t\t\treturn KeyReleased, nil
\t\t}
\t\treturn KeyPressed, nil
\t}
\treturn eventType, nil
}
'''

new_func = f'''func (r *Runtime) keyEventType(card uint32, key int32, pressed bool) (uint32, error) {{
\teventType := KeyReleased
\tif pressed {{
\t\teventType = KeyPressed
\t}}
\tcardWords, err := r.ReadWords(card, 2)
\tif err != nil {{
\t\treturn 0, err
\t}}
\tclass, err := r.InspectJavaClass(cardWords[1])
\tif err != nil {{
\t\treturn 0, err
\t}}
\tif class.Name == "Clet$CletCard" {{
\t\t// Inotia1 (AID {INOTIA1_AID}) follows the reference WIPI emulator's
\t\t// normal Card ABI for directional edges: press=1, release=2.  The
\t\t// generic KTF Clet path is inverted for other titles (issue #237), so
\t\t// keep that inversion everywhere except these four directions.
\t\tif r.inotiaDirectionUsesCardInput(key) {{
\t\t\treturn eventType, nil
\t\t}}
\t\tif pressed {{
\t\t\treturn KeyReleased, nil
\t\t}}
\t\treturn KeyPressed, nil
\t}}
\treturn eventType, nil
}}
'''

if new_func not in scheduler:
    if old_func not in scheduler:
        raise SystemExit("keyEventType source did not match the expected baseline")
    scheduler = scheduler.replace(old_func, new_func, 1)

# At the 240x320 geometry Inotia can register the direction keys through
# Display.grabKey. The reference WIPI emulator's grabKey implementation is a
# no-op, so those physical directions still travel through the displayed Card.
# ARAM's implemented grab path would instead divert them to JletEventListener,
# bypassing the Clet/Card edge fix and producing the observed stuck movement.
# Force only Inotia1's four directions back through the Card path; every other
# title/key keeps stock ARAM behavior.
helper_marker = "func (r *Runtime) inotiaDirectionUsesCardInput(key int32) bool"
if helper_marker not in scheduler:
    anchor = '''func (r *Runtime) CanQueueKeyEventFor(key int32) bool {
'''
    helper = f'''func (r *Runtime) inotiaDirectionUsesCardInput(key int32) bool {{
\tif r.Pkg.Descriptor.AID != "{INOTIA1_AID}" {{
\t\treturn false
\t}}
\tswitch key {{
\tcase -1, -2, -3, -4: // up, down, left, right
\t\treturn true
\tdefault:
\t\treturn false
\t}}
}}

func (r *Runtime) CanQueueKeyEventFor(key int32) bool {{
'''
    if anchor not in scheduler:
        raise SystemExit("CanQueueKeyEventFor anchor did not match the expected baseline")
    scheduler = scheduler.replace(anchor, helper, 1)

old_gate = '''func (r *Runtime) CanQueueKeyEventFor(key int32) bool {
\tif r.grabbedKeys[key] != 0 {
\t\treturn r.HasJavaTaskCapacity()
\t}
\treturn r.canQueueCardKeyEvent()
}
'''
new_gate = '''func (r *Runtime) CanQueueKeyEventFor(key int32) bool {
\tif r.grabbedKeys[key] != 0 && !r.inotiaDirectionUsesCardInput(key) {
\t\treturn r.HasJavaTaskCapacity()
\t}
\treturn r.canQueueCardKeyEvent()
}
'''
if old_gate in scheduler:
    scheduler = scheduler.replace(old_gate, new_gate, 1)
elif new_gate not in scheduler:
    raise SystemExit("CanQueueKeyEventFor body did not match the expected baseline")

old_queue = '''func (r *Runtime) QueueKeyEvent(pressed bool, key int32) (bool, error) {
\tif listener := r.grabbedKeys[key]; listener != 0 {
'''
new_queue = '''func (r *Runtime) QueueKeyEvent(pressed bool, key int32) (bool, error) {
\tif listener := r.grabbedKeys[key]; listener != 0 && !r.inotiaDirectionUsesCardInput(key) {
'''
if old_queue in scheduler:
    scheduler = scheduler.replace(old_queue, new_queue, 1)
elif new_queue not in scheduler:
    raise SystemExit("QueueKeyEvent grabbed-key branch did not match the expected baseline")

old_await = '''func (r *Runtime) CanAwaitKeyEvent(key int32) bool {
\treturn !r.terminationRequested &&
\t\t(r.grabbedKeys[key] != 0 || r.CanAwaitEvents())
}
'''
new_await = '''func (r *Runtime) CanAwaitKeyEvent(key int32) bool {
\tgrabbed := r.grabbedKeys[key] != 0 && !r.inotiaDirectionUsesCardInput(key)
\treturn !r.terminationRequested && (grabbed || r.CanAwaitEvents())
}
'''
if old_await in scheduler:
    scheduler = scheduler.replace(old_await, new_await, 1)
elif new_await not in scheduler:
    raise SystemExit("CanAwaitKeyEvent body did not match the expected baseline")

SCHEDULER.write_text(scheduler)

# Update focused unit tests for the key-aware helper.
tests = TESTS.read_text()
tests = tests.replace(
    "runtime.keyEventType(ordinary, true)",
    "runtime.keyEventType(ordinary, -5, true)",
)
tests = tests.replace(
    "runtime.keyEventType(ordinary, false)",
    "runtime.keyEventType(ordinary, -5, false)",
)

marker = "func TestKTFInotia1DirectionsUseRegularHeldEdges"
if marker not in tests:
    tests += f'''

// TestKTFInotia1DirectionsUseRegularHeldEdges pins both physical edges to the
// normal WIPI Card ABI used by the reference emulator.
func TestKTFInotia1DirectionsUseRegularHeldEdges(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA1_AID}"
\truntime.JvmContext = allocWords(t, runtime, 3+128)

\tclass := inspectClass(t, runtime, ensureClass(t, runtime, "Clet$CletCard"))
\t_, err := runtime.addHostJavaMethod(class, "keyNotify", "(II)Z")
\tcheck(t, err)
\tcard, err := runtime.NewJavaInstanceForClass(class)
\tcheck(t, err)

\tfor _, key := range []int32{{-1, -2, -3, -4}} {{
\t\tpressed, err := runtime.keyEventType(card, key, true)
\t\tcheck(t, err)
\t\tif pressed != KeyPressed {{
\t\t\tt.Fatalf("Inotia1 direction %d press event = %d, want %d", key, pressed, KeyPressed)
\t\t}}
\t\treleased, err := runtime.keyEventType(card, key, false)
\t\tcheck(t, err)
\t\tif released != KeyReleased {{
\t\t\tt.Fatalf("Inotia1 direction %d release event = %d, want %d", key, released, KeyReleased)
\t\t}}
\t}}

\tpressed, err := runtime.keyEventType(card, -5, true)
\tcheck(t, err)
\tif pressed != KeyReleased {{
\t\tt.Fatalf("Inotia1 non-direction Clet press event = %d, want native value %d", pressed, KeyReleased)
\t}}
}}
'''

route_marker = "func TestKTFInotia1DirectionsBypassGrabbedKeyPath"
if route_marker not in tests:
    tests += f'''

// TestKTFInotia1DirectionsBypassGrabbedKeyPath reproduces the 240x320 case:
// even if the game registered a direction with grabKey, the reference WIPI
// emulator keeps physical direction input on the visible Card path.
func TestKTFInotia1DirectionsBypassGrabbedKeyPath(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA1_AID}"
\truntime.JvmContext = allocWords(t, runtime, 3+128)

\tclass := inspectClass(t, runtime, ensureClass(t, runtime, "Clet$CletCard"))
\t_, err := runtime.addHostJavaMethod(class, "keyNotify", "(II)Z")
\tcheck(t, err)
\tcard, err := runtime.NewJavaInstanceForClass(class)
\tcheck(t, err)

\tconst display = uint32(0x10004000)
\truntime.DefaultDisplay = display
\truntime.DisplayCards[display] = card

\tfor _, key := range []int32{{-1, -2, -3, -4}} {{
\t\t// A deliberately invalid grabbed-listener address is safe only when
\t\t// the Inotia direction correctly bypasses the grabbed-key branch.
\t\truntime.grabbedKeys[key] = 0x12345678
\t\tif !runtime.CanQueueKeyEventFor(key) {{
\t\t\tt.Fatalf("Inotia1 direction %d was not accepted by Card gate", key)
\t\t}}

\t\tqueued, err := runtime.QueueKeyEvent(true, key)
\t\tcheck(t, err)
\t\tif !queued || len(runtime.Tasks) == 0 {{
\t\t\tt.Fatalf("Inotia1 direction %d press queue = %t, tasks=%d", key, queued, len(runtime.Tasks))
\t\t}}
\t\tpressTask := runtime.Tasks[len(runtime.Tasks)-1]
\t\tif pressTask.KeyCard != card {{
\t\t\tt.Fatalf("Inotia1 direction %d press used grabbed path: KeyCard=0x%08x", key, pressTask.KeyCard)
\t\t}}
\t\tcheck(t, runtime.CPU.RestoreContext(pressTask.Context))
\t\tpressed, err := runtime.CPU.ReadRegister(cpu.RegisterR2)
\t\tcheck(t, err)
\t\tif pressed != KeyPressed {{
\t\t\tt.Fatalf("Inotia1 direction %d queued press = %d, want %d", key, pressed, KeyPressed)
\t\t}}
\t\tpressTask.Done = true

\t\tqueued, err = runtime.QueueKeyEvent(false, key)
\t\tcheck(t, err)
\t\tif !queued {{
\t\t\tt.Fatalf("Inotia1 direction %d release did not queue", key)
\t\t}}
\t\treleaseTask := runtime.Tasks[len(runtime.Tasks)-1]
\t\tif releaseTask.KeyCard != card {{
\t\t\tt.Fatalf("Inotia1 direction %d release used grabbed path: KeyCard=0x%08x", key, releaseTask.KeyCard)
\t\t}}
\t\tcheck(t, runtime.CPU.RestoreContext(releaseTask.Context))
\t\treleased, err := runtime.CPU.ReadRegister(cpu.RegisterR2)
\t\tcheck(t, err)
\t\tif released != KeyReleased {{
\t\t\tt.Fatalf("Inotia1 direction %d queued release = %d, want %d", key, released, KeyReleased)
\t\t}}
\t\treleaseTask.Done = true
\t}}
}}
'''

TESTS.write_text(tests)

print(f"patched {SCHEDULER}: Inotia1 directions use reference WIPI Card press/release path")
print(f"updated {TESTS}: direct edge + grabbed-path regressions")
