package runtime

import (
	"crypto/sha256"
	"reflect"
	"testing"
	"time"
)

func TestContinuousMusicSurvivesStopReleaseEffectAndReplay(t *testing.T) {
	m, bus, bgm := newRampMedia(t)
	source, _ := m.Source(1, bgm)
	m.SetContinuousMusicSources([][sha256.Size]byte{sha256.Sum256(source)})
	check(t, m.Play(1, bgm, 1))
	step := 125 * time.Microsecond
	advance := func(start, end time.Duration, want []int16) {
		t.Helper()
		check(t, m.Advance(start, end, bus))
		if got := m.Drain().PCM16; !reflect.DeepEqual(got, want) {
			t.Fatalf("PCM %v..%v: got %v want %v", start, end, got, want)
		}
	}
	advance(0, step, []int16{10}) // exactly one music voice
	revision := m.OutputRevision()
	voice := m.bgmVoice
	check(t, m.Stop(1, bgm))
	check(t, m.DestroyClip(1, bgm, bus))
	fx, err := m.CreateClip(1, "audio/wav", 0)
	check(t, err)
	_, err = m.Append(1, fx, pcmWave(8000, 1, []int16{1000, 2000}))
	check(t, err)
	check(t, m.Play(1, fx, 1))
	advance(step, 2*step, []int16{1020})
	check(t, m.Pause(1, fx))
	check(t, m.Seek(1, fx, 0))
	check(t, m.Resume(1, fx))
	check(t, m.Stop(1, fx))
	check(t, m.Clear(1, fx))
	check(t, m.DestroyClip(1, fx, bus))
	advance(2*step, 3*step, []int16{30})
	if m.OutputRevision() != revision {
		t.Fatal("an effect or menu stop reset the music output timeline")
	}
	replay, err := m.CreateClip(1, "audio/wav", 0)
	check(t, err)
	_, err = m.Append(1, replay, source)
	check(t, err)
	check(t, m.Play(1, replay, 1))
	if m.bgmVoice != voice || m.OutputRevision() != revision {
		t.Fatal("the same music selection restarted its voice")
	}
	advance(3*step, 7*step, []int16{40, 10, 20, 30})
	info, err := m.Info(1, replay)
	check(t, err)
	if info.State != ClipStopped || info.RemainingPlays != 0 {
		t.Fatal("the guest one-shot clip no longer completes normally")
	}
	check(t, m.DestroyClip(1, replay, bus))
	advance(7*step, 8*step, []int16{40})
}

func TestContinuousMusicSelectionReplacesOnlyKnownTracks(t *testing.T) {
	m, bus, bgm := newRampMedia(t)
	first, _ := m.Source(1, bgm)
	second := pcmWave(8000, 1, []int16{8, 10})
	m.SetContinuousMusicSources([][sha256.Size]byte{sha256.Sum256(first), sha256.Sum256(second)})
	check(t, m.Play(1, bgm, 1))
	check(t, m.Advance(0, 125*time.Microsecond, bus))
	_ = m.Drain()
	check(t, m.Stop(1, bgm))
	check(t, m.Clear(1, bgm))
	_, err := m.Append(1, bgm, second)
	check(t, err)
	revision := m.OutputRevision()
	check(t, m.Play(1, bgm, 1))
	if m.OutputRevision() == revision {
		t.Fatal("a different map track kept the previous output generation")
	}
	check(t, m.Advance(125*time.Microsecond, 250*time.Microsecond, bus))
	if got := m.Drain().PCM16; !reflect.DeepEqual(got, []int16{8}) {
		t.Fatalf("two map tracks mixed or a track doubled: %v", got)
	}
	m.SetContinuousMusicSources(nil)
	check(t, m.Stop(1, bgm))
	check(t, m.Advance(250*time.Microsecond, 375*time.Microsecond, bus))
	if m.MusicVoiceActive() || len(m.Drain().PCM16) != 0 {
		t.Fatal("disabling the policy did not restore normal stop semantics")
	}
}

func TestContinuousMusicSurvivesServicesBinaryRestore(t *testing.T) {
	config := DefaultConfig()
	config.Limits.Media.OutputSampleRate = 8000
	config.Limits.Media.OutputChannels = 1
	s, err := NewServices(config)
	check(t, err)
	source := pcmWave(8000, 1, []int16{10, 20, 30, 40})
	digests := [][sha256.Size]byte{sha256.Sum256(source)}
	s.Media.SetContinuousMusicSources(digests)
	id, err := s.Media.CreateClip(1, "audio/wav", 0)
	check(t, err)
	_, err = s.Media.Append(1, id, source)
	check(t, err)
	check(t, s.Media.Play(1, id, 1))
	check(t, s.Media.Advance(0, 125*time.Microsecond, s.Events))
	_ = s.Media.Drain()
	check(t, s.Media.Stop(1, id))
	check(t, s.Media.DestroyClip(1, id, s.Events))
	encoded, err := s.MarshalBinary()
	check(t, err)
	restored, err := NewServices(config)
	check(t, err)
	check(t, restored.UnmarshalBinary(encoded))
	if restored.Media.MusicVoiceActive() {
		t.Fatal("a generic state parser enabled an unselected playback policy")
	}
	if restored.Media.Snapshot().BGMVoice == nil {
		t.Fatal("transactional state parsing dropped the saved music cursor")
	}
	restored.Media.SetContinuousMusicSources(digests)
	check(t, restored.Media.Advance(125*time.Microsecond, 250*time.Microsecond, restored.Events))
	if got := restored.Media.Drain().PCM16; !reflect.DeepEqual(got, []int16{20}) {
		t.Fatalf("restored music cursor changed: %v", got)
	}
}

func TestContinuousMusicAllowlistDoesNotRetainAnEffect(t *testing.T) {
	m, bus, fx := newRampMedia(t)
	m.SetContinuousMusicSources([][sha256.Size]byte{sha256.Sum256([]byte("different music"))})
	check(t, m.Play(1, fx, 1))
	check(t, m.Advance(0, time.Millisecond, bus))
	if m.MusicVoiceActive() {
		t.Fatal("an unlisted sound became endless music")
	}
	info, err := m.Info(1, fx)
	check(t, err)
	if info.State != ClipStopped {
		t.Fatal("effect did not end")
	}
}
