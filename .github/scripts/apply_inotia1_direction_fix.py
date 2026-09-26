from pathlib import Path

SCHEDULER = Path("../aram-core/application/internal/ktf/ktf_scheduler.go")
TESTS = Path("../aram-core/application/internal/ktf/ktf_clet_input_test.go")
INOTIA1_AID = "010100D3"
INOTIA2_AID = "010100D5"

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
\t\t// Inotia 1 (AID {INOTIA1_AID}) uses ordinary Card edge values for the
\t\t// four directional keys. Keep the original title-specific fix intact:
\t\t// press=1 and release=2, while its non-direction Clet keys retain the
\t\t// generic native ABI below.
\t\tif r.Pkg.Descriptor.AID == "{INOTIA1_AID}" {{
\t\t\tswitch key {{
\t\t\tcase -1, -2, -3, -4: // up, down, left, right
\t\t\t\treturn eventType, nil
\t\t\t}}
\t\t}}

\t\t// Inotia 2 (AID {INOTIA2_AID}) uses the same ordinary edge values for
\t\t// directions and the centre/OK key. Treating directions as ordinary but
\t\t// leaving OK on the inverted Clet edge makes an OK press look like a
\t\t// release immediately after a direction release; the title can then
\t\t// re-latch its last movement direction. Keep all five gameplay keys on
\t\t// one coherent edge convention so releasing a direction is final before
\t\t// the following attack/OK press is delivered.
\t\tif r.Pkg.Descriptor.AID == "{INOTIA2_AID}" {{
\t\t\tswitch key {{
\t\t\tcase -1, -2, -3, -4, -5: // up, down, left, right, OK/select
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

# Update the focused unit tests for the new key-aware helper and add title-
# specific regression coverage for both Inotia releases.
tests = TESTS.read_text()
tests = tests.replace(
    "runtime.keyEventType(ordinary, true)",
    "runtime.keyEventType(ordinary, -5, true)",
)
tests = tests.replace(
    "runtime.keyEventType(ordinary, false)",
    "runtime.keyEventType(ordinary, -5, false)",
)

marker1 = "func TestKTFInotia1DirectionsUseRegularHeldEdges"
if marker1 not in tests:
    tests += f'''

// TestKTFInotia1DirectionsUseRegularHeldEdges pins the original Inotia 1 fix.
// A held direction enters as a normal press and leaves as a normal release,
// while non-direction Clet keys retain the generic native ABI.
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
\t\tt.Fatalf("Inotia1 OK press event = %d, want native Clet value %d", pressed, KeyReleased)
\t}}
}}
'''

marker2 = "func TestKTFInotia2DirectionsAndOKUseRegularHeldEdges"
if marker2 not in tests:
    tests += f'''

// TestKTFInotia2DirectionsAndOKUseRegularHeldEdges pins the Inotia 2 gameplay
// edge convention. In particular, direction release followed immediately by
// OK press must remain release(2), press(1), never the mixed Clet convention
// that can re-latch the previous direction.
func TestKTFInotia2DirectionsAndOKUseRegularHeldEdges(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA2_AID}"
\truntime.JvmContext = allocWords(t, runtime, 3+128)

\tclass := inspectClass(t, runtime, ensureClass(t, runtime, "Clet$CletCard"))
\t_, err := runtime.addHostJavaMethod(class, "keyNotify", "(II)Z")
\tcheck(t, err)
\tcard, err := runtime.NewJavaInstanceForClass(class)
\tcheck(t, err)

\tfor _, key := range []int32{{-1, -2, -3, -4, -5}} {{
\t\tpressed, err := runtime.keyEventType(card, key, true)
\t\tcheck(t, err)
\t\tif pressed != KeyPressed {{
\t\t\tt.Fatalf("Inotia2 gameplay key %d press event = %d, want %d", key, pressed, KeyPressed)
\t\t}}
\t\treleased, err := runtime.keyEventType(card, key, false)
\t\tcheck(t, err)
\t\tif released != KeyReleased {{
\t\t\tt.Fatalf("Inotia2 gameplay key %d release event = %d, want %d", key, released, KeyReleased)
\t\t}}
\t}}

\tupReleased, err := runtime.keyEventType(card, -1, false)
\tcheck(t, err)
\tokPressed, err := runtime.keyEventType(card, -5, true)
\tcheck(t, err)
\tif upReleased != KeyReleased || okPressed != KeyPressed {{
\t\tt.Fatalf("Inotia2 up-release/OK-press sequence = (%d,%d), want (%d,%d)", upReleased, okPressed, KeyReleased, KeyPressed)
\t}}

\tsoftPressed, err := runtime.keyEventType(card, -6, true)
\tcheck(t, err)
\tif softPressed != KeyReleased {{
\t\tt.Fatalf("Inotia2 non-gameplay Clet press event = %d, want native value %d", softPressed, KeyReleased)
\t}}
}}
'''
TESTS.write_text(tests)

print(f"patched {SCHEDULER} for Inotia1 D3 directions and Inotia2 D5 directions+OK")
print(f"updated {TESTS} with D3 preservation and D5 release/OK regression coverage")

# Layer the W-Feature save-identity compatibility after the already-validated
# input fixes so the two changes remain independent and regression-tested.
compat_path = Path(".github/scripts/apply_inotia2_wfeature_save_compat.py")
compat_source = compat_path.read_text()
exec(compile(compat_source, str(compat_path), "exec"), {"__name__": "__main__"})
