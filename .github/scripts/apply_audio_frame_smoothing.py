from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CORE = ROOT.parent / "aram-core" / "runtime"
p = CORE / "media.go"
s = p.read_text()
old = """\tif target < current+2_048 {
\t\ttarget = current + 2_048
\t}"""
new = """\t// Keep lazy synthesis close to the samples the current frame needs.
\t// A 2048-frame lookahead synthesized almost three 60 Hz audio frames
\t// together at 44.1 kHz, creating periodic work bursts on the emulation
\t// thread. Smaller batches preserve the synthesis order and exact PCM
\t// while spreading that work across normal media advances.
\tif target < current+256 {
\t\ttarget = current + 256
\t}"""
if new not in s:
    if s.count(old) != 1: raise SystemExit("SMAF lazy-render baseline mismatch")
    s = s.replace(old, new, 1)
p.write_text(s)
(CORE / "media_smaf_frame_smoothing_test.go").write_text((ROOT / ".github/patches/media_smaf_frame_smoothing_test.go").read_text())
print("Lazy SMAF synthesis now uses 256-frame lookahead with identical PCM")
