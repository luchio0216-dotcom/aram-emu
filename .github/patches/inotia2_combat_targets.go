package application

import (
	"crypto/sha256"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
)

const inotia2CombatTargetsHelper = uint32(0x2ae980)
const inotia2CombatTargetsHook = uint32(0x13180a)

// Exact-client native helpers leave save identity and v1008 clock/audio intact.
var inotia2CombatTargetsPatches = []inotia2OfflineShopPatch{
	{address: 0x2ae980, originalSHA: "6db65fd59fd356f6729140571b5bcd6bb3b83492a16e1bf0a3884442fc3c8a0e", replacement: inotia2PatchBytes("002800da2046844204d003496b581b68024a1047024a1047f8140000151813008f181300")},
	{address: 0x13180a, originalSHA: "9cef0bc358b96063a6b92291355ec2237881e08b5b2132f62710cdd7594d4a12", replacement: inotia2PatchBytes("014b1847c04681e92a00")},
}

func installInotia2CombatTargets(client []byte, backend cpu.Backend) error {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	if err := installInotia2OfflineShopPatches(backend, inotia2CombatTargetsPatches); err != nil {
		return fmt.Errorf("Inotia2 combat targets: %w", err)
	}
	return nil
}
