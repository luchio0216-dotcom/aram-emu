from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core" / "cpu" / "interpreter"
p = CORE / "native_mmap_arm64.go"
s = p.read_text()
old = """// arenaPageSize is the mprotect granularity assumed for the W^X flip. A wrong
// guess only costs precision - the range is still page-aligned by rounding down
// to a multiple of it, and arm64 Linux/Android pages are never smaller than 4
// KiB - so it does not have to be queried at run time.
const arenaPageSize = uintptr(4096)"""
new = """// mprotect needs the host's real page granularity. New Android devices may
// use 16 KiB pages: a 4 KiB-aligned subrange can fail with EINVAL or leave a
// neighboring cached block's page writable. Keep W^X changes host-aligned.
var arenaPageSize = uintptr(syscall.Getpagesize())"""
if new not in s:
    if s.count(old) != 1: raise SystemExit("ARM64 native page-size baseline mismatch")
    s = s.replace(old, new, 1)
p.write_text(s)
(CORE / "native_android_pages_test.go").write_text((ROOT / ".github/patches/native_android_pages_test.go").read_text())
print("ARM64 native W^X pages now follow the host's page size")
