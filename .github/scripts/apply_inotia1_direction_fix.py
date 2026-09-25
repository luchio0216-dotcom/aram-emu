from pathlib import Path

SCHEDULER = Path("../aram-core/application/internal/ktf/ktf_scheduler.go")
TESTS = Path("../aram-core/application/internal/ktf/ktf_clet_input_test.go")
INOTIA1_AID = "010100D3"

scheduler = SCHEDULER.read_text()

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
\t\t// Inotia 1 (KT/WIPI 1.2, AID {INOTIA1_AID}) uses the regular Card
\t\t// edge values for the four directional keys even though its display
\t\t// object is Clet$CletCard. Preserve the host's real held state and
\t\t// translate only the guest ABI edge here: press=1, release=2. The
\t\t// previous product workaround inverted event.Pressed before QueueInput,
\t\t// which fixed the stuck-release symptom but corrupted hold semantics.
\t\tif r.Pkg.Descriptor.AID == "{INOTIA1_AID}" {{
\t\t\tswitch key {{
\t\t\tcase -1, -2, -3, -4: // up, down, left, right
\t\t\t\treturn eventType, nil
\t\t\t}}
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
SCHEDULER.write_text(scheduler)

# Update the focused unit tests for the new key-aware helper and add a title-
# specific regression test that pins both directions of the edge while keeping
# non-direction Clet behavior unchanged.
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

// TestKTFInotia1DirectionsUseRegularHeldEdges pins the title-specific edge
// conversion without mutating the host Pressed state. A held direction enters
// as a normal press and leaves as a normal release, while non-direction Clet
// keys retain the generic native ABI.
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
TESTS.write_text(tests)

print(f"patched {SCHEDULER} at the Clet guest edge layer; host Pressed state is untouched")
print(f"updated {TESTS} with Inotia1 directional held-edge regression coverage")
