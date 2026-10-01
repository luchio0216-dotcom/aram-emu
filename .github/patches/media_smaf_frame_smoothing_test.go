package runtime

import "testing"

func frameSmoothingScore(t testing.TB) []byte {
	t.Helper()
	return smafScore([]byte{
		0x00, 0xb0, 0x07, 0x7f,
		0x00, 0xc0, 0x00,
		0x00, 0xb1, 0x07, 0x60,
		0x00, 0xc1, 0x18,
		0x00, 0x90, 60, 110, 120,
		0x0a, 0x91, 67, 90, 110,
		0x0a, 0x90, 64, 100, 90,
		0x28, 0x80, 60, 60,
		0x28, 0xff, 0x2f,
	})
}

// Render the same synthetic score eagerly and in the exact lazy batches used
// by the mixer. Every sample must match, including the natural release tail.
func TestSMAFFrameSmoothingPreservesPCM(t *testing.T) {
	score := frameSmoothingScore(t)
	want := decodeSMAFPCM16(score, 44_100)
	if want == nil { t.Fatal("synthetic score did not decode") }
	d := &smafDecoder{rate:44_100}
	if !d.parse(score) || !d.buildEvents() { t.Fatal("synthetic score rejected") }
	got := &decodedPCM{sampleRate:44_100, channels:2, smaf:newSMAFRenderStream(d)}
	for frame:=uint64(0); frame<uint64(len(want.samples)/2); frame++ {
		got.ensureFrame(frame)
		for ch:=uint64(0); ch<2; ch++ {
			i:=frame*2+ch
			if i>=uint64(len(got.samples)) || got.samples[i]!=want.samples[i] {
				t.Fatalf("PCM differs at frame %d channel %d",frame,ch)
			}
		}
		if frames:=uint64(len(got.samples)/2); frames>frame+256 { t.Fatalf("excessive lazy lookahead: %d after frame %d",frames,frame) }
	}
	got.ensureFrame(uint64(len(want.samples)/2)+256)
	if len(got.samples)!=len(want.samples) { t.Fatalf("release tail changed: got %d samples, want %d",len(got.samples),len(want.samples)) }
}
