package application

import (
	"crypto/sha256"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
)

const inotia2InventorySwapCheck = uint32(0x2ae340)
const inotia2InventorySwapMove = uint32(0x2ae420)
const inotia2InventorySwapDraw = uint32(0x2aef20)
const inotia2InventorySwapFinish = uint32(0x2aef60)

// Keep native empty-slot moves and merges. Occupied targets can exchange a
// complete item/stack with its source slot; the displaced pointer remains owned
// there and becomes the held item. Cancel never has to recover an unowned item.
// These hooks belong only to the inventory UI, not equipment or merchants.
var inotia2InventorySwapPatches = []inotia2OfflineShopPatch{
	{address: 0x2ae340, originalSHA: "a5645e7a3fa0866cde8842c4dab96567507c3d1a3c028b816bc63f6966367b70", replacement: inotia2PatchBytes("f0b583b004460d46002c56d0002d54d0ac4252d02a4b1b681b78012b4dd1294b1b681b78012b48d1274b1f683b88022b43d17b68a34240d1204699f69bfd00283bd0bb78984238d1204b1b681a68052a33d801921e4b1b681b68002b2dd0d87a0f282ad8197b884227d21b69194e3668920192199a4220d1800012181368ab421bd10292204600a998f604fd002814d06b4618784109019a052902d1052a0cd101e0052a09d0890189191f23184080000918029a2b46012000e0002003b0f0bd7c2f1900b0291900ac2f1900942f19006c2a19006c2d1900")},
	{address: 0x2ae420, originalSHA: "cd00e292c5970d3c5e2f0ffa5171e555bc46bfc4faddfb4a418b6840b86e79a3", replacement: inotia2PatchBytes("ffb581b099f61ef8002825d1039a049b052a20d80f2b1ed892019b00d2180f4b1b68d21811680198fff77aff002812d00c4615461e46304699f62cfd00280ad0632808d8019b26602b60054b1b685e609870022000e0002005b0f0bd6c2d1900ac2f1900")},
	{address: 0x2aef20, originalSHA: "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925", replacement: inotia2PatchBytes("37b5099c002c06d038462146fff708fa002800d00024234637bc002b00bdc046")},
	{address: 0x2aef60, originalSHA: "374708fff7719dd5979ec875d56cd2286f6d3cf7ec317a3b25632aab28ec37bb", replacement: inotia2PatchBytes("022800d17047014b1847c046a1751600")},
	{address: 0x111bc8, originalSHA: "16c1b30def5f4fb37677f27066a93476859c11670fe1d553d5e0ad2e77f10b0e", replacement: inotia2PatchBytes("9cf12afc")},
	{address: 0x111bd2, originalSHA: "b3256ecb8345aef7488fbaa04b172fe0050a31ebc87b082e858e73ffb8e77cbd", replacement: inotia2PatchBytes("9df1c5f9")},
	{address: 0x1679e4, originalSHA: "d3a7084140ade7ff3f87eceadc46f40f78feb570e65c94eabdddcd4404adb0dd", replacement: inotia2PatchBytes("47f19cfa")},
}

func installInotia2InventorySwap(client []byte, backend cpu.Backend) error {
	if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA {
		return nil
	}
	if err := installInotia2OfflineShopPatches(backend, inotia2InventorySwapPatches); err != nil {
		return fmt.Errorf("Inotia2 inventory swap: %w", err)
	}
	return nil
}
