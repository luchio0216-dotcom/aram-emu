from pathlib import Path

KERNEL = Path("../aram-core/application/internal/ktf/ktf_wipic_kernel.go")
SCHEDULER = Path("../aram-core/application/internal/ktf/ktf_scheduler.go")
TESTS = Path("../aram-core/application/internal/ktf/ktf_clet_input_test.go")
INOTIA2_AID = "010100D5"

source = KERNEL.read_text()
old = '''func (r *Runtime) systemPropertyValue(key string) (string, bool) {
\tkey = strings.ToUpper(strings.TrimSpace(key))
\tif value, ok := r.wipicSystemProperties[key]; ok {
'''
new = f'''func (r *Runtime) systemPropertyValue(key string) (string, bool) {{
\tkey = strings.ToUpper(strings.TrimSpace(key))
\t// W-Feature supplies a deterministic offline KTF subscriber number. Inotia 2
\t// reads PHONENUMBER while constructing the handset identity stored alongside
\t// save0.dat; ARAM's generic KTF profile normally reports an unconfigured
\t// number as unsupported. Match W-Feature only for Inotia 2 so its portable
\t// .wfs saves see the same subscriber identity without changing other titles.
\tif r.Pkg.Descriptor.AID == "{INOTIA2_AID}" && key == "PHONENUMBER" {{
\t\treturn "01000000000", true
\t}}
\tif value, ok := r.wipicSystemProperties[key]; ok {{
'''
if new not in source:
    if old not in source:
        raise SystemExit("KTF systemPropertyValue baseline did not match")
    source = source.replace(old, new, 1)
KERNEL.write_text(source)

# Inotia 2 needs one edge convention for the entire keypad, not only the
# directions and OK/select. Mixing normal direction edges with inverted Clet
# edges for digits/CLR/soft keys lets the title briefly re-latch the last held
# direction when the next non-OK button is pressed. Keep every Inotia 2 Clet
# key on ordinary press=KeyPressed/release=KeyReleased semantics. This remains
# strictly title-scoped, so the generic Clet workaround and Inotia 1 behavior
# are unchanged.
scheduler = SCHEDULER.read_text()
old_edges = f'''\t\t// Inotia 2 (AID {INOTIA2_AID}) uses the same ordinary edge values for
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
'''
new_edges = f'''\t\t// Inotia 2 (AID {INOTIA2_AID}) expects ordinary edge values for the
\t\t// whole Clet keypad. If directions use ordinary edges but digits, CLR or
\t\t// soft keys stay on the generic inverted Clet convention, pressing one of
\t\t// those keys immediately after a direction release can re-latch the last
\t\t// direction for one guest tick. Use one coherent press/release convention
\t\t// for every Inotia 2 key so a direction release is final before any next
\t\t// keypad event is delivered.
\t\tif r.Pkg.Descriptor.AID == "{INOTIA2_AID}" {{
\t\t\treturn eventType, nil
\t\t}}
'''
if new_edges not in scheduler:
    if old_edges not in scheduler:
        raise SystemExit("Inotia2 direction/OK edge block did not match expected patched baseline")
    scheduler = scheduler.replace(old_edges, new_edges, 1)
SCHEDULER.write_text(scheduler)

tests = TESTS.read_text()
marker = "func TestKTFInotia2WFeaturePhoneNumberCompatibility"
if marker not in tests:
    tests += f'''

// TestKTFInotia2WFeaturePhoneNumberCompatibility pins the W-Feature subscriber
// identity only for Inotia 2 while preserving ARAM's ordinary KTF behavior.
func TestKTFInotia2WFeaturePhoneNumberCompatibility(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA2_AID}"
\tgot, ok := runtime.systemPropertyValue("PHONENUMBER")
\tif !ok || got != "01000000000" {{
\t\tt.Fatalf("Inotia2 PHONENUMBER = %q, %t; want W-Feature 01000000000, true", got, ok)
\t}}

\truntime.Pkg.Descriptor.AID = "010100D3"
\tgot, ok = runtime.systemPropertyValue("PHONENUMBER")
\tif ok || got != "" {{
\t\tt.Fatalf("non-Inotia2 PHONENUMBER changed from generic ARAM behavior: %q, %t", got, ok)
\t}}
}}
'''

# The earlier input regression intentionally proved non-gameplay keys still
# used the generic Clet ABI. That is now exactly the bug being fixed, so update
# that expectation and add an explicit direction-release -> digit-4 sequence.
old_soft = '''\tsoftPressed, err := runtime.keyEventType(card, -6, true)
\tcheck(t, err)
\tif softPressed != KeyReleased {
\t\tt.Fatalf("Inotia2 non-gameplay Clet press event = %d, want native value %d", softPressed, KeyReleased)
\t}
'''
new_soft = '''\tsoftPressed, err := runtime.keyEventType(card, -6, true)
\tcheck(t, err)
\tif softPressed != KeyPressed {
\t\tt.Fatalf("Inotia2 non-gameplay Clet press event = %d, want regular value %d", softPressed, KeyPressed)
\t}
'''
if new_soft not in tests:
    if old_soft not in tests:
        raise SystemExit("Inotia2 non-gameplay edge regression block did not match expected baseline")
    tests = tests.replace(old_soft, new_soft, 1)

marker_edges = "func TestKTFInotia2DirectionReleaseThenAnyKeyDoesNotRelatch"
if marker_edges not in tests:
    tests += f'''

// TestKTFInotia2DirectionReleaseThenAnyKeyDoesNotRelatch reproduces the phone
// keypad sequence seen in-game: release right, then immediately press digit 4.
// The second event must be a fresh press, never an inverted release that lets
// the title consume one more tick of the previous movement direction.
func TestKTFInotia2DirectionReleaseThenAnyKeyDoesNotRelatch(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA2_AID}"
\truntime.JvmContext = allocWords(t, runtime, 3+128)

\tclass := inspectClass(t, runtime, ensureClass(t, runtime, "Clet$CletCard"))
\t_, err := runtime.addHostJavaMethod(class, "keyNotify", "(II)Z")
\tcheck(t, err)
\tcard, err := runtime.NewJavaInstanceForClass(class)
\tcheck(t, err)

\trightReleased, err := runtime.keyEventType(card, -4, false)
\tcheck(t, err)
\tdigit4Pressed, err := runtime.keyEventType(card, int32('4'), true)
\tcheck(t, err)
\tif rightReleased != KeyReleased || digit4Pressed != KeyPressed {{
\t\tt.Fatalf("Inotia2 right-release/digit4-press = (%d,%d), want (%d,%d)", rightReleased, digit4Pressed, KeyReleased, KeyPressed)
\t}}

\t// Representative non-direction keypad keys must all use the same ordinary
\t// edge convention: digits, star/hash, CLR and soft/menu-style keys.
\tfor _, key := range []int32{{'0', '1', '4', '9', '*', '#', -16, -6}} {{
\t\tpressed, err := runtime.keyEventType(card, key, true)
\t\tcheck(t, err)
\t\treleased, err := runtime.keyEventType(card, key, false)
\t\tcheck(t, err)
\t\tif pressed != KeyPressed || released != KeyReleased {{
\t\t\tt.Fatalf("Inotia2 key %d edges = (%d,%d), want (%d,%d)", key, pressed, released, KeyPressed, KeyReleased)
\t\t}}
\t}}
}}
'''
TESTS.write_text(tests)

print(f"patched {KERNEL} with title-scoped Inotia2 W-Feature PHONENUMBER compatibility")
print(f"patched {SCHEDULER} so every Inotia2 Clet key uses ordinary press/release edges")
print(f"updated {TESTS} with W-Feature identity and direction-release/any-key regressions")
