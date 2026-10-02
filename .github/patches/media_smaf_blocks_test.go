package runtime

import (
	"math"
	"testing"
)

func lazyBlocksForTest(t testing.TB) (*decodedPCM, *decodedPCM) {
	t.Helper()
	score := frameSmoothingScore(t)
	eager := decodeSMAFPCM16(score, 44100)
	d := &smafDecoder{rate: 44100}
	if eager == nil || !d.parse(score) || !d.buildEvents() {
		t.Fatal("synthetic score rejected")
	}
	return &decodedPCM{sampleRate: 44100, channels: 2, smaf: newSMAFRenderStream(d)}, eager
}

func TestSMAFBlockGrowthKeepsPlayedPCMPrefixInPlace(t *testing.T) {
	lazy, eager := lazyBlocksForTest(t)
	lazy.ensureFrame(0)
	first := &lazy.samples[0]
	lazy.ensureFrame(uint64(len(eager.samples)/2) + 256)
	if &lazy.samples[0] != first {
		t.Fatal("extending music copied its played prefix")
	}
	if lazy.pcmFrames()*2 != uint64(len(eager.samples)) || len(lazy.smafBlocks) == 0 {
		t.Fatal("synthetic score did not cross a storage boundary")
	}
	for i, want := range eager.samples {
		if got := lazy.pcmSample(uint64(i)); got != want {
			t.Fatalf("sample %d: got %d want %d", i, got, want)
		}
	}
	// Loops and seeks must keep the exact earlier PCM, including block seams.
	for _, frame := range []uint64{0, 4095, 4096, 4097, 8191, 8192} {
		if frame < lazy.pcmFrames() {
			lazy.ensureFrame(frame)
			if lazy.pcmSample(frame*2) != eager.samples[frame*2] {
				t.Fatal("backward sample changed")
			}
		}
	}
	lazy.ensureFrame(math.MaxUint64)
	if lazy.pcmFrames()*2 != uint64(len(eager.samples)) {
		t.Fatal("extreme seek wrapped or changed the natural tail")
	}
}

func TestSMAFBlocksResampleIdenticallyAcrossStorageSeams(t *testing.T) {
	lazy, eager := lazyBlocksForTest(t)
	frames := uint64(len(eager.samples) / 2)
	lazy.ensureFrame(frames)
	for _, rate := range []uint32{22050, 48000} {
		kernel := resampleKernelFor(44100, rate)
		for _, frame := range []uint64{0, 4090, 4095, 4096, 4100, 8191, 8192, frames - 1} {
			if frame >= frames {
				continue
			}
			for _, fraction := range []float64{0, 0.125, 0.5, 0.875} {
				a, b := eager.resampleAt(kernel, frame, fraction, frames)
				c, d := lazy.resampleAt(kernel, frame, fraction, frames)
				if a != c || b != d {
					t.Fatalf("resampled PCM changed at frame %d phase %g", frame, fraction)
				}
			}
		}
	}
}

func BenchmarkSMAFBlockedPCMIncrementalMusic(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		lazy, eager := lazyBlocksForTest(b)
		for frame := uint64(0); frame < uint64(len(eager.samples)/2); frame += 256 {
			lazy.ensureFrame(frame)
		}
	}
}
