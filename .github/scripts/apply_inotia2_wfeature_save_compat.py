from pathlib import Path

JLET = Path("../aram-core/application/internal/ktf/ktf_java_jlet.go")
TESTS = Path("../aram-core/application/internal/ktf/ktf_clet_input_test.go")
INOTIA2_AID = "010100D5"

source = JLET.read_text()
old = '''\tcase "getCurrentProgramID()I":
\t\treturn 1, nil
'''
new = f'''\tcase "getCurrentProgramID()I":
\t\t// W-Feature's KTF Java compatibility surface answers zero here. Inotia 2
\t\t// includes this value in the handset identity used while validating its
\t\t// persistent character data, so ARAM's generic value 1 makes a W-Feature
\t\t// save appear as ERROR(00). Keep the compatibility strictly title-scoped;
\t\t// every other KTF title retains ARAM's existing value.
\t\tif r.Pkg.Descriptor.AID == "{INOTIA2_AID}" {{
\t\t\treturn 0, nil
\t\t}}
\t\treturn 1, nil
'''
if new not in source:
    if old not in source:
        raise SystemExit("KTF Jlet getCurrentProgramID baseline did not match")
    source = source.replace(old, new, 1)
JLET.write_text(source)

tests = TESTS.read_text()
marker = "func TestKTFInotia2WFeatureProgramIDCompatibility"
if marker not in tests:
    tests += f'''

// TestKTFInotia2WFeatureProgramIDCompatibility pins the W-Feature handset
// identity that Inotia 2 persists alongside its character save. The exception
// is deliberately AID-scoped so unrelated KTF titles keep the generic ID 1.
func TestKTFInotia2WFeatureProgramIDCompatibility(t *testing.T) {{
\truntime := newTestRuntime(t)
\truntime.Pkg.Descriptor.AID = "{INOTIA2_AID}"
\tgot, err := runtime.handleJletMethod("getCurrentProgramID", "()I")
\tcheck(t, err)
\tif got != 0 {{
\t\tt.Fatalf("Inotia2 getCurrentProgramID = %d, want W-Feature value 0", got)
\t}}

\truntime.Pkg.Descriptor.AID = "010100D3"
\tgot, err = runtime.handleJletMethod("getCurrentProgramID", "()I")
\tcheck(t, err)
\tif got != 1 {{
\t\tt.Fatalf("non-Inotia2 getCurrentProgramID = %d, want generic ARAM value 1", got)
\t}}
}}
'''
TESTS.write_text(tests)

print(f"patched {JLET} with title-scoped Inotia2 W-Feature program ID compatibility")
print(f"updated {TESTS} with program-ID compatibility regression coverage")
