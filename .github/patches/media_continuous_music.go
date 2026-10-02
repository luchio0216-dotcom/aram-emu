package runtime

import (
	"crypto/sha256"
	"slices"
)

// SetContinuousMusicSources is an opt-in playback policy selected by a
// verified title profile. Only these exact encoded tracks get a host music
// voice. The guest clip still completes, stops and releases normally.
func (m *Media) SetContinuousMusicSources(digests [][sha256.Size]byte) {
	if slices.Equal(m.continuousMusicSources, digests) {
		m.reconcileContinuousMusic()
		return
	}
	wasEnabled := len(m.continuousMusicSources) != 0
	m.continuousMusicSources = slices.Clone(digests)
	if len(digests) == 0 {
		for _, clip := range m.clips {
			clip.continuousMusic = false
		}
		if wasEnabled && m.bgmVoice != nil {
			m.bgmVoice = nil
			m.invalidateOutput()
		}
		return
	}
	m.reconcileContinuousMusic()
}

func (m *Media) reconcileContinuousMusic() {
	if len(m.continuousMusicSources) == 0 {
		return
	}
	if m.pendingMusicVoice != nil && m.isContinuousMusicSource(m.pendingMusicVoice.source) {
		m.bgmVoice = m.pendingMusicVoice
		m.pendingMusicVoice = nil
	}
	// Rebuild this derived flag after a state load. Older states may have a
	// sounding BGM clip but no detached voice yet.
	for _, id := range m.sortedClipIDs() {
		clip := m.clips[id]
		clip.continuousMusic = m.isContinuousMusicSource(clip.source)
		if clip.continuousMusic && clip.state == ClipPlaying && m.bgmVoice == nil {
			m.selectContinuousMusic(clip)
		}
	}
}

func (m *Media) isContinuousMusicSource(source []byte) bool {
	if len(m.continuousMusicSources) == 0 {
		return false
	}
	return slices.Contains(m.continuousMusicSources, sha256.Sum256(source))
}

func (m *Media) selectContinuousMusic(clip *mediaClip) {
	if m.bgmVoice != nil && slices.Equal(m.bgmVoice.source, clip.source) {
		// A guest replay after an effect is the same selection. Keep the live
		// cursor and decoded PCM instead of rewinding or adding a second voice.
		return
	}
	m.bgmVoice = &mediaClip{
		mediaType: clip.mediaType, source: cloneBytes(clip.source),
		decoded: clip.decoded, position: clip.position, state: ClipPlaying,
		volume: clip.volume, muted: clip.muted, pan: clip.pan, remainingPlays: -1,
	}
	m.pendingMusicVoice = nil
	m.invalidateOutput()
}

func (m *Media) invalidateClipOutput() {
	// The title stops, clears and releases its one-shot clips around effects.
	// Keep the already produced music timeline through those operations.
	if len(m.continuousMusicSources) != 0 && m.bgmVoice != nil {
		return
	}
	m.invalidateOutput()
}

func (m *Media) savedMusicVoice() *mediaClip {
	if m.bgmVoice != nil {
		return m.bgmVoice
	}
	// Services parsing constructs a fresh Media before the title policy is
	// re-applied. Carry the validated voice through that transaction silently.
	return m.pendingMusicVoice
}
