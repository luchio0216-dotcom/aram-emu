package application

import (
	shared "github.com/mirusu400/aram-core/runtime"
	"testing"
)

func TestInotia2ContinuousMusicUsesExactTitleAndTrackIdentity(t *testing.T) {
	if len(inotia2ContinuousMusicDigests) != 14 {
		t.Fatal("music allowlist changed")
	}
	seen := map[[32]byte]bool{}
	for _, digest := range inotia2ContinuousMusicDigests {
		if seen[digest] {
			t.Fatal("duplicate music resource")
		}
		seen[digest] = true
	}
	media, err := shared.NewMedia(shared.NewRegistry(32), shared.DefaultMediaLimits())
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range []bool{false, true, false} {
		configureInotia2ContinuousMusic(media, match)
		if media.MusicVoiceActive() {
			t.Fatal("configuration alone started music")
		}
	}
	if inotia2FramePacingMatches("010100D3", "Clet", inotia2FramePacingPatchedHash) ||
		inotia2FramePacingMatches("010100D5", "Clet", [32]byte{}) {
		t.Fatal("unverified title got the playback experiment")
	}
}
