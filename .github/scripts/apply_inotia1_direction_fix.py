from pathlib import Path
import subprocess

# Keep the already-validated Inotia 1/2 input patch byte-for-byte from the
# compatibility baseline, then layer the W-Feature save-identity compatibility
# experiment on top. Using git show avoids duplicating the established input
# patch while this branch is being validated against real W-Feature backups.
baseline_commit = "5bcdce4d868f6c32b1d243bbbd63f5ecf6ec80f1"
baseline = subprocess.check_output(
    ["git", "show", f"{baseline_commit}:.github/scripts/apply_inotia1_direction_fix.py"],
    text=True,
)
exec(compile(baseline, "apply_inotia1_direction_fix.baseline.py", "exec"), {"__name__": "__main__"})

compat = Path(".github/scripts/apply_inotia2_wfeature_save_compat.py").read_text()
exec(compile(compat, "apply_inotia2_wfeature_save_compat.py", "exec"), {"__name__": "__main__"})
