package application

import (
	"crypto/sha256"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
)

const inotia2CombatGeometryHelper = uint32(0x2ae900)
const inotia2CombatGeometryHook = uint32(0x1734bc)

// Exact-client native helpers leave save identity and v1008 clock/audio intact.
var inotia2CombatGeometryPatches = []inotia2OfflineShopPatch{
	{address: 0x2ae900, originalSHA: "5b6fb58e61fa475939767d68a446f97f1bff02c0e5935a3ea8bb51e6515783d8", replacement: inotia2PatchBytes("f0b50022865e8f5e0422835e8c5ebb421cdba6421adcbe4200da3e46a34200dd23469e1b0222875e8d5e0622835e8c5eab420bdba74209dcaf4200da2f46a34200dd2346df1b7e433046f0bd0020f0bd")},
	{address: 0x1734bc, originalSHA: "8af9f2d5cf390b08711473c72ecd6d3816cfff1d92e31a6b9c56c655874554aa", replacement: inotia2PatchBytes("004b184701e92a00")},
}

func installInotia2CombatGeometry(client []byte, backend cpu.Backend) error {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	if err := installInotia2OfflineShopPatches(backend, inotia2CombatGeometryPatches); err != nil {
		return fmt.Errorf("Inotia2 combat geometry: %w", err)
	}
	return nil
}
