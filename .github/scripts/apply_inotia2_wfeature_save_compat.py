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
TESTS.write_text(tests)

print(f"patched {KERNEL} with title-scoped Inotia2 W-Feature PHONENUMBER compatibility")
print(f"updated {TESTS} with phone-number compatibility regression coverage")
