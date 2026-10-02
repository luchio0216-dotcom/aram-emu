package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
)

const inotia2AutoLootOriginalSHA = "e9dc70fa04194aed2e5051a3da92ff6b7999506fd85d69f432b6abba7f5d8551"
const inotia2AutoLootPatchedSHA = "0447cbaad06b2757f79536ae50c94e7a6cebf1b2b8739977aa9f7bfa67173a73"
const inotia2AutoLootPreviousSHA = "36b0e69210923dee4d622d239dc9ff900f07780e921c860cb6aa9d541b312aff"

type inotia2ImagePatch struct {
	offset      int
	expectedSHA string
	replacement []byte
}

func inotia2PatchBytes(value string) []byte {
	data, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return data
}

// These are newly assembled Thumb helpers, not game assets. The known Rare12
// client already returns from its offline-ready check at 0x12af8c; reclaim
// only that unreachable body and keep the original text and BSS metadata.
// Empty ground lists skip pickup entirely. The automatic-call flag lives in
// page-aligned image padding, away from translated code and store barriers.
// The actor update hook preserves the original actor tag and callee registers,
// then calls the game's proximity pickup only for the currently selected hero.
// Native pickup retains its slot/stack checks and removes drops only after
// insertion succeeds. Its full-inventory message is suppressed only while
// this synchronous automatic call is running; manual OK retains its behavior.
var inotia2AutoLootPatches = []inotia2ImagePatch{
	{offset: 0x37c2a, expectedSHA: "16126326ddc04c123c19ae8a51ad876978108d3e82402d382c07624dafb1a8ce", replacement: inotia2PatchBytes("f3f7b1f9")},
	{offset: 0x2ce38, expectedSHA: "b0392ee1e89ed4f94b06aac8a28859b9f0e6524d6e31dba44b415d7e070a1487", replacement: inotia2PatchBytes("fef7ccf8")},
	{offset: 0x2af90, expectedSHA: "d2732bb9ad6321ccc1ff83b7de06db67497faa81b39420ed4b4439b721ca1aca", replacement: inotia2PatchBytes("70b5044601f0f4fa0546002d0fd0094909688c420bd108490968002907d0074e01233360204601f09bfe00233360284670bd00bf78aa2a006c4c190048d32a00c046c04608b5054b1b68002b02d0044b019308bd01200ba908bd00bf48d32a001bce1200c046c046c046c046c046c046c046c046c046c046c046c046c046c04600000000")},
}

// Parallel flags follow the native 16-entry ground list. The discard caller
// sets only its own entry; native swaps carry its flag, and field reset clears
// it. The native proximity loop releases a flag after leaving its bounds.
var inotia2DiscardPatches = []inotia2ImagePatch{
	{offset: 0x4f25e, expectedSHA: "3604ce7653856988f4b6b3e2db4a4f761a3acd9f0e648ea69d6449a51272b078", replacement: inotia2PatchBytes("5ef1cffec046c046")},
	{offset: 0x2cd50, expectedSHA: "fb0cf922ad90b876b159150f96c8e07bf70f71a92e8d171ad49fa4948bdeabf0", replacement: inotia2PatchBytes("81f166f9")},
	{offset: 0x4f188, expectedSHA: "0f13d339f562b633dab25cd031209f85ba68f0b36faa6cdb5fdaa5526e5fadf7", replacement: inotia2PatchBytes("5ef16eff")},
	{offset: 0x4f2a0, expectedSHA: "6979ccd2a7f18991d0afa89621d11d60dc6ce4b85ab5fbb4245b709560d66a4d", replacement: inotia2PatchBytes("5ef1eefe")},
}

func inotia2AutoLootClient(client []byte) ([]byte, error) {
	digest := fmt.Sprintf("%x", sha256.Sum256(client))
	if digest == inotia2AutoLootPatchedSHA {
		return client, nil
	}
	if digest == inotia2AutoLootPreviousSHA {
		return applyInotia2ImagePatches(client, inotia2AutoLootPreviousSHA, inotia2DiscardPatches)
	}
	return applyInotia2ImagePatches(client, inotia2AutoLootOriginalSHA,
		append(append([]inotia2ImagePatch(nil), inotia2AutoLootPatches...), inotia2DiscardPatches...))
}

// Check every expected span before changing a clone. Unknown clients pass
// through unchanged; corrupt or overlapping patch definitions fail atomically.
func applyInotia2ImagePatches(client []byte, imageSHA string, patches []inotia2ImagePatch) ([]byte, error) {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != imageSHA {
		return client, nil
	}
	for i, p := range patches {
		if p.offset < 0 || p.offset > len(client) || len(p.replacement) > len(client)-p.offset {
			return nil, fmt.Errorf("Inotia2 automatic pickup patch %d is outside client", i)
		}
		span := client[p.offset : p.offset+len(p.replacement)]
		if fmt.Sprintf("%x", sha256.Sum256(span)) != p.expectedSHA {
			return nil, fmt.Errorf("Inotia2 automatic pickup patch %d original checksum mismatch", i)
		}
		for _, q := range patches[:i] {
			if p.offset < q.offset+len(q.replacement) && q.offset < p.offset+len(p.replacement) {
				return nil, fmt.Errorf("Inotia2 automatic pickup patch %d overlaps an earlier patch", i)
			}
		}
	}
	out := bytes.Clone(client)
	for _, p := range patches {
		copy(out[p.offset:], p.replacement)
	}
	return out, nil
}

// Reserve mapped padding after the original BSS without changing the BSS size
// passed into bootstrap: the native module checks that argument against its
// relocation header. Padding is zeroed on reset and saved with mapped memory.
func inotia2AutoLootMappedSize(client []byte, size uint32) (uint32, error) {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return size, nil
	}
	const original = uint32(608192 + 1149832)
	// Keep mutable flags on the existing padding page and executable helpers
	// on a separate page. Flag writes therefore cannot invalidate native code.
	const padded = ((original + 4095) &^ uint32(4095)) + 4096
	if size == padded {
		return size, nil
	}
	if size != original {
		return 0, fmt.Errorf("Inotia2 automatic pickup image mapping baseline mismatch: %d", size)
	}
	return padded, nil
}

const inotia2DiscardHelperAddress = uint32(0x2ae000)
const inotia2DiscardFlagsAddress = uint32(0x2ad34c)

// New Thumb code occupies mapped padding, leaving text length, original BSS
// size, bootstrap arguments and the user's persistent-save identity intact.
// The code is installed before execution and captured in full machine states.
var inotia2DiscardHelper = inotia2PatchBytes("1d609a609e80df800fb50999244a0023914200d1012380001f4a13500fbdc046d0180fb51d4b1b68002b1ad0a9001a4b5a58002a15d00421425eba420ddb52450bdc0621425e0b998a4206db0c998a4203dc0fbc01b0134b1847a9000e4b00225a500fbc042300bd80b5a1f645f80fb4a000a900084b1a585a500fbc80bdc046002313700fb5044810210022026004300139fbd10fbdc0464cd32a0048d32a003f0e110035ce1200")

func installInotia2AutoLootHelpers(client []byte, backend cpu.Backend) error {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	return backend.WriteMemory(inotia2DiscardHelperAddress, inotia2DiscardHelper)
}
