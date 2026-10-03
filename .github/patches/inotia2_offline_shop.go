package application

import (
 "bytes"
 "crypto/sha256"
 "fmt"
 "github.com/mirusu400/aram-core/cpu"
)

const inotia2OfflineShopFlag = uint32(0x2ad400)
const inotia2OfflineShopEntry = uint32(0x2ae200)
const inotia2OfflineShopInit = uint32(0x2ae240)
const inotia2OfflineShopPrice = uint32(0x2ae500)
const inotia2OfflineShopCleanup = uint32(0x2ae580)
const inotia2OfflineShopTable = uint32(0x2ae680)

type inotia2OfflineShopPatch struct {
 address uint32
 originalSHA string
 legacySHA string
 replacement []byte
}

// New code/data share the already mapped executable padding page. Mutable
// title flags stay on its separate data page, avoiding per-frame JIT eviction.
// Only cash-menu entry, merchant initialization, purchase-price lookup and
// merchant cleanup are hooked. The native buying transaction retains its
// cloning, capacity check, successful insertion and gold-debit ordering.
// Other merchant initialization/pricing falls through to the original body.
var inotia2OfflineShopPatches = []inotia2OfflineShopPatch{
 {address: 0x2ae200, originalSHA: "de47c9b27eb8d300dbb5f2c353e632c393262cf06340c4fa7f1b40c4cbd36f90", legacySHA: "873a4ad42c341429dc413496a64be261d999ae781c48b29aef5b4dc794506910", replacement: inotia2PatchBytes("10b5034c012020601020c8f6e5fe10bd00d42a00")},
 {address: 0x2ae240, originalSHA: "c4fcd50d9f0c893c46288b57d8e62b18523145956b249b6ecd6c21718be49065", legacySHA: "34bcde65fd05bfbb7101b7b2ae728d5e7331748292f20ca1c3307e1f78f8c288", replacement: inotia2PatchBytes("70b55646454660b42b4b1b68002b01d12a4b1847b9f6dcfdb9f688ff0446002c46d091f61ffb264d002628889bf626ff002802d0314691f61bfa08350136182ef3d3204d204b1b681878204a1060012018700020206106206070182020731420e07114202072042020710420a0700620e0700020e072164b1b68e361154b1b682362154b1b686362144b1b681c60002071f6defc20460021b9f60afe0146104b1a680020b6f61efc00200e49baf6ecfd0cbc90469a4670bd00d42a00b105120080e62a00c42419000c3519000cd42a00782a19007c2a1900283519005c2a190078aa2a0080e72a00")},
 {address: 0x2ae500, originalSHA: "1751ac12e70e15b4f76c16775cd329ae55973b612521dab2de828a5cdb6c8ab3", legacySHA: "04173062399ac6ade7c623b0ea9b1c0b3a5209f40e7a9ac0b6bd74bb28c5a5e5", replacement: inotia2PatchBytes("11b50d4b1b68002b0dd0028992090b4b18211888904203d008330139f9d102e0586801b010bd11bc08bc9e4670b505469bf6e6fc024b184700d42a0080e62a00d1ae1400")},
 {address: 0x2ae580, originalSHA: "6db65fd59fd356f6729140571b5bcd6bb3b83492a16e1bf0a3884442fc3c8a0e", legacySHA: "a0086b1ddf141d6e3ffdcf78353f1e190ebb5b57d59823f8fe523ecf19fda4ba", replacement: inotia2PatchBytes("10b5064c2068002805d0054b1b68e06818700020206091f685f910bd00d42a000c351900")},
 {address: 0x2ae680, originalSHA: "5d89f056865052bcb89c910d2d62872e029fb273c3db03f8968a52a41593c1b5", legacySHA: "be31de2980b74b9a5c0cc2c1aade7ca73beb6dae1f19592b081f563409f7ce6a", replacement: inotia2PatchBytes("59030000102700009902000010270000430300008813000044030000102700004703000010270000af030000204e0000b0030000204e0000b1030000204e0000b2030000204e0000b3030000204e0000b5030000204e0000ad03000050c30000ac03000050c30000b603000030750000b7030000307500008702000088130000450300001027000046030000102700000400000030750000a503000050c30000a6030000a0860100a703000030750000a8030000307500007003000010270000")},
 {address: 0x2ae780, originalSHA: "5322fecfc92a5e3248a297a3df3eddfb9bd9049504272e4f572b87fa36d4b3bd", legacySHA: "5322fecfc92a5e3248a297a3df3eddfb9bd9049504272e4f572b87fa36d4b3bd", replacement: inotia2PatchBytes("b7cec4c320b0f1b5e520bbf3c1a100")},
 {address: 0x120890, originalSHA: "c26fbefa97ddecbc7bdff67dfeb4021ce6c213256be0c909c8ab2bcf22959881", legacySHA: "47d0a4b5c5147e4b360d558c6a6473e5335a9bff6b59f608c313d4e666fd49dc", replacement: inotia2PatchBytes("004b184701e22a00")},
 {address: 0x1205a8, originalSHA: "d440ac40c5248ac74e645983fad8b8ddeaa920da77e77f151468db0cb8d6178a", legacySHA: "10a5bd418c183a937e381b3b8ff561d049a1bf10fc1671d4a037093c14797dfe", replacement: inotia2PatchBytes("004b184741e22a00")},
 {address: 0x14aec8, originalSHA: "6d2c56da87762ababfdd04e2c1d374784eb4ba3c43111517bd1344d2777aabfe", legacySHA: "c7b84bb11ede0f2062814d98652157a9d3ec25ebc1ba271391384dc60224d0d7", replacement: inotia2PatchBytes("004b184701e52a00")},
 {address: 0x11fec8, originalSHA: "89a28ca5bfe82a53418cd53957380216b65b694a38d206603d5753aef8866674", legacySHA: "93edb1a3cc9fc5cc6faa894bad1e2b7363acfa40ebc32f923e740f8e1c3b92f6", replacement: inotia2PatchBytes("8ef15afb")},
}

func installInotia2OfflineShop(client []byte, backend cpu.Backend) error {
 // Leave package bytes, title/save hashes, BSS bootstrap size and audio/frame
 // behavior at the v1008 baseline. Unknown games never reach a memory write.
 if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA { return nil }
 return installInotia2OfflineShopPatches(backend, inotia2OfflineShopPatches)
}

func installInotia2OfflineShopPatches(backend cpu.Backend, patches []inotia2OfflineShopPatch) error {
 originalState, legacyState, installedState := true, true, true
 for i, p := range patches {
  if len(p.replacement) == 0 || uint64(p.address)+uint64(len(p.replacement)) > 1<<32 {
   return fmt.Errorf("Inotia2 local shop patch %d has invalid bounds", i)
  }
  for _, q := range patches[:i] {
   if uint64(p.address) < uint64(q.address)+uint64(len(q.replacement)) && uint64(q.address) < uint64(p.address)+uint64(len(p.replacement)) {
    return fmt.Errorf("Inotia2 local shop patch %d overlaps", i)
   }
  }
  before := make([]byte, len(p.replacement))
  if err := backend.ReadMemory(p.address, before); err != nil { return err }
  digest := fmt.Sprintf("%x", sha256.Sum256(before))
  original := digest == p.originalSHA
  legacy := p.legacySHA != "" && digest == p.legacySHA
  installed := bytes.Equal(before, p.replacement)
  if !original && !legacy && !installed {
   return fmt.Errorf("Inotia2 local shop patch %d original checksum mismatch", i)
  }
  originalState = originalState && original
  legacyState = legacyState && legacy
  installedState = installedState && installed
 }
 if installedState { return nil }
 if !originalState && !legacyState { return fmt.Errorf("Inotia2 local shop patch installation is incomplete") }
 // Validate every span before changing code. Execution has not started here;
 // accept only a complete original or v1012 image, never a mixture. This also
 // upgrades the previous catalog when restoring a v1012 full machine state.
 // An unexpected mapped-memory write error aborts loading the title.
 for _, p := range patches {
  if err := backend.WriteMemory(p.address, p.replacement); err != nil { return err }
 }
 return nil
}
