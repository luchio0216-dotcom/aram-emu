package runtime

// A block holds about 93 ms of stereo PCM at 44.1 kHz (16 KiB).
// Synthesis still advances in 256-frame batches; the larger storage blocks
// avoid a growing-prefix allocation and copy after minutes of music.
const smafPCMBlockFrames = uint64(4096)

func (decoded *decodedPCM) pcmFrames() uint64 {
	if decoded.smaf != nil {
		return decoded.smafFrames
	}
	return uint64(len(decoded.samples)) / uint64(decoded.channels)
}

func (decoded *decodedPCM) pcmSample(index uint64) int16 {
	if decoded.smaf == nil || index < smafPCMBlockFrames*2 {
		return decoded.samples[index]
	}
	const samplesPerBlock = smafPCMBlockFrames * 2
	return decoded.smafBlocks[index/samplesPerBlock-1][index%samplesPerBlock]
}

func (decoded *decodedPCM) ensureFrame(frame uint64) {
	if decoded == nil || decoded.smaf == nil || frame < decoded.smafFrames || decoded.smaf.finished || decoded.smaf.end == 0 {
		return
	}
	// Clamp before adding one so malformed extreme seeks cannot wrap.
	if frame >= decoded.smaf.end {
		frame = decoded.smaf.end - 1
	}
	target := frame + 1
	if next := decoded.smafFrames + 256; target < next {
		target = next
	}
	if target > decoded.smaf.end {
		target = decoded.smaf.end
	}
	for decoded.smafFrames < target && !decoded.smaf.finished {
		blockIndex := decoded.smafFrames / smafPCMBlockFrames
		end := min(target, (blockIndex+1)*smafPCMBlockFrames)
		var block []int16
		if blockIndex == 0 {
			if decoded.samples == nil {
				decoded.samples = make([]int16, 0, smafPCMBlockFrames*2)
			}
			block = decoded.samples
		} else {
			if blockIndex-1 == uint64(len(decoded.smafBlocks)) {
				decoded.smafBlocks = append(decoded.smafBlocks, make([]int16, 0, smafPCMBlockFrames*2))
			}
			block = decoded.smafBlocks[blockIndex-1]
		}
		previous := len(block)
		block = decoded.smaf.renderUntil(block, end)
		decoded.smafFrames += uint64(len(block)-previous) / 2
		if blockIndex == 0 {
			decoded.samples = block
		} else {
			decoded.smafBlocks[blockIndex-1] = block
		}
	}
}
