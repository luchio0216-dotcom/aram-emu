from pathlib import Path

KERNEL = Path("../aram-core/application/internal/ktf/ktf_wipic_kernel.go")
TESTS = Path("../aram-core/application/internal/ktf/ktf_clet_input_test.go")
INOTIA2_AID = "010100D5"

source = KERNEL.read_text()
old = '''func (r *Runtime) systemPropertyValue(key string) (string, bool) {
\tkey = strings.ToUpper(strings.TrimSpace(key))
\tif value, ok := r.wipicSystemProperties[key]; ok {
'''
new = f'''func (r *Runtime) systemPropertyValue(key string) (string, bool) {{
\tkey = strings.ToUpper(strings.TrimSpace(key))
\t// Inotia 2 persists a handset identity beside save0.dat and rejects a slot
\t// when that identity differs. W-Feature, whose .wfs files are the portable
\t// reference for this title, reports PHONEMODEL=Emulator. Keep this override
\t// strictly scoped to Inotia 2; every other KTF title retains ARAM's existing
\t// generic KTF handset model.
\tif r.Pkg.Descriptor.AID == "{INOTIA2_AID}" && key == "PHONEMODEL" {{
\t\treturn "Emulator", true
\t}}
\tif value, ok := r.wipicSystemProperties[key]; ok {{
'''
if new not in source:
    if old not in source:
        raise SystemExit("KTF systemPropertyValue baseline did not match")
    source = source.replace(old, new, 1)
KERNEL.write_text(source)

tests = TESTS.read_text()
marker = "func TestKTFInotia2WFeaturePhoneModelCompatibility"
if marker not in tests:
    tests += f'''

// TestKTFInotia2WFeaturePhoneModelCompatibility pins the title-scoped handset
// model used by W-Feature while leaving ARAM's generic KTF model unchanged.
func TestKTFInotia2WFeaturePhoneModelCompatibility(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA2_AID}"
\tgot, ok := runtime.systemPropertyValue("PHONEMODEL")
\tif !ok || got != "Emulator" {{
\t\tt.Fatalf("Inotia2 PHONEMODEL = %q, %t; want W-Feature Emulator, true", got, ok)
\t}}

\truntime.Pkg.Descriptor.AID = "010100D3"
\tgot, ok = runtime.systemPropertyValue("PHONEMODEL")
\tif !ok || got == "Emulator" {{
\t\tt.Fatalf("non-Inotia2 PHONEMODEL unexpectedly uses W-Feature override: %q, %t", got, ok)
\t}}
}}
'''
TESTS.write_text(tests)

print(f"patched {KERNEL} with title-scoped Inotia2 W-Feature PHONEMODEL compatibility")
print(f"updated {TESTS} with phone-model compatibility regression coverage")
