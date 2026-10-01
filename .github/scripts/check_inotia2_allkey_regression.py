"""Prove the queued-key regression rejects the previous direction/OK-only fix."""
from pathlib import Path
import subprocess

scheduler = Path("../aram-core/application/internal/ktf/ktf_scheduler.go")
original = scheduler.read_text()
all_keys = '''\t\tif r.Pkg.Descriptor.AID == "010100D5" {
\t\t\treturn eventType, nil
\t\t}
'''
old_partial = '''\t\tif r.Pkg.Descriptor.AID == "010100D5" {
\t\t\tswitch key {
\t\t\tcase -1, -2, -3, -4, -5:
\t\t\t\treturn eventType, nil
\t\t\t}
\t\t}
'''
if original.count(all_keys) != 1:
    raise SystemExit("expected exactly one Inotia2 all-key edge override")
try:
    scheduler.write_text(original.replace(all_keys, old_partial, 1))
    result = subprocess.run(
        ["go", "test", "./application/internal/ktf",
         "-run", "^TestKTFInotia2QueuedDirectionReleaseThenAnyKey$", "-count=1"],
        cwd="../aram-core", text=True, capture_output=True, timeout=180,
    )
    if result.returncode == 0 or "queued type = 2, want 1" not in result.stdout:
        print(result.stdout)
        print(result.stderr)
        raise SystemExit("queued-key regression did not reject the old partial fix")
    print("PASS: queued-key regression rejects the previous direction/OK-only fix")
finally:
    scheduler.write_text(original)
