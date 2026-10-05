package application

import (
	"crypto/sha256"
	"fmt"

	"github.com/mirusu400/aram-core/cpu"
)

const inotia2ExperienceHelper = uint32(0x2aea00)
const inotia2ExperienceHook = uint32(0x154cc8)

// Scale only the exact client's final, positive monster reward. The native
// level-up routine, quest rewards, save identity and v1008 timing are retained.
// Levels 41/51/61/71/81/91 use 4/5/6/8/10/12x respectively. Complete
// previous 4x full states upgrade atomically, before execution resumes.
var inotia2ExperiencePatches = []inotia2OfflineShopPatch{
	{address: 0x2aea00, originalSHA: "17b0761f87b081d5cf10757ccc89f12be355c70e2e29df288b65b30710dcbcd1", recentSHA: "7f641ad40abf7ab6ae2566c7be0c99ab69307b891a4b5454adcb6a41c948dbce", replacement: inotia2PatchBytes("5246292a10db0423332a0cdb05233d2a09db0623472a06db0823512a03db0a235b2a00db0c235943004b184751551300")},
	{address: 0x154cc8, originalSHA: "b4f54e330b24378a6d500873904a15b184f7411caa744b01c080cc389c10bec1", recentSHA: "b40fdbf7196465949363f8a269470f58811b27970a2713c6721d19d756781300", replacement: inotia2PatchBytes("59f19afe")},
}

func installInotia2Experience(client []byte, backend cpu.Backend) error {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	if err := installInotia2OfflineShopPatches(backend, inotia2ExperiencePatches); err != nil {
		return fmt.Errorf("Inotia2 tiered monster experience: %w", err)
	}
	return nil
}
