package application

import (
 "crypto/sha256"
 "fmt"
 "github.com/mirusu400/aram-core/cpu"
)

const inotia2AutoIdentifyHelper = uint32(0x2ae800)
const inotia2AutoIdentifyHook = uint32(0x1476cc)

// The native acquisition routine has validated capacity before reaching this
// hook. Its live incoming item has not yet been inserted/merged/freed. Calling
// the scroll's native identification action preserves generated options, item
// identity and non-equipment quantities. Stock templates remain unidentified;
// cancellation, insufficient gold and full bags never reach this hook.
// There is no timer, frame poll, input injection or audio change.
var inotia2AutoIdentifyPatches = []inotia2OfflineShopPatch{
 {address: 0x2ae800, originalSHA: "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925", replacement: inotia2PatchBytes("02b530469bf62efc3589ad09304699f651fb0abc9e463a78004b1847d5761400")},
 {address: 0x1476cc, originalSHA: "d88fb7f026605d1fd0aa59921b997f94c0db0e1d635a14205e3e2af0d9e4477f", replacement: inotia2PatchBytes("004b184701e82a00")},
}

func installInotia2AutoIdentify(client []byte, backend cpu.Backend) error {
 if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA { return nil }
 // Reuse complete-image validation: v1008/v1012/v1013 have original spans;
 // current states have both replacements. Partial or corrupt states fail
 // before any writes. Text size, save hashes and mapped geometry stay fixed.
 if err := installInotia2OfflineShopPatches(backend, inotia2AutoIdentifyPatches); err != nil {
  return fmt.Errorf("Inotia2 automatic identification: %w", err)
 }
 return nil
}
