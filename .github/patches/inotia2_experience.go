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
var inotia2ExperiencePatches = []inotia2OfflineShopPatch{
    {address: 0x2aea00, originalSHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", replacement: inotia2PatchBytes("5246292a00db8900004b184751551300")},
    {address: 0x154cc8, originalSHA: "b4f54e330b24378a6d500873904a15b184f7411caa744b01c080cc389c10bec1", replacement: inotia2PatchBytes("59f19afe")},
}

func installInotia2Experience(client []byte, backend cpu.Backend) error {
    if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
        return nil
    }
    if err := installInotia2OfflineShopPatches(backend, inotia2ExperiencePatches); err != nil {
        return fmt.Errorf("Inotia2 level 41+ monster experience: %w", err)
    }
    return nil
}
