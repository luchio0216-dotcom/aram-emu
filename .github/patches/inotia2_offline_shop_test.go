package application

import (
 "bytes"
 "crypto/sha256"
 "fmt"
 "testing"
 "github.com/mirusu400/aram-core/cpu"
)

func shopTestCode(t *testing.T, b cpu.Backend) {
 t.Helper()
 for _, p := range inotia2OfflineShopPatches { discardWrite(t, b, p.address, p.replacement) }
 for r, v := range map[uint32]uint32{
  cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555,
  cpu.RegisterR6: 0x66666666, cpu.RegisterR8: 0x88888888,
  cpu.RegisterR10: 0xaaaaaaaa, cpu.RegisterLR: 0x200101,
 } { discardReg(t, b, r, v) }
}

func shopTestPreserved(t *testing.T, b cpu.Backend) {
 t.Helper()
 for r, v := range map[uint32]uint32{
  cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555,
  cpu.RegisterR6: 0x66666666, cpu.RegisterR8: 0x88888888,
  cpu.RegisterR10: 0xaaaaaaaa, cpu.RegisterSP: 0x2ff000,
 } { discardCheckReg(t, b, r, v) }
}

func TestInotia2OfflineShopUnknownTitle(t *testing.T) {
 // A nil backend also proves an unrelated image never touches guest memory.
 if err := installInotia2OfflineShop([]byte("unrelated title"), nil); err != nil { t.Fatal(err) }
}

func TestInotia2OfflineShopInstallationGuards(t *testing.T) {
 inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
  original := []byte("abcdefgh")
  p := inotia2OfflineShopPatch{address: 0x210000,
   originalSHA: fmt.Sprintf("%x", sha256.Sum256(original)), replacement: []byte("ABCDEFGH")}
  q := p; q.address += 16
  reset := func() {
   discardWrite(t, b, p.address, original); discardWrite(t, b, q.address, original)
  }
  reset()
  bad := q; bad.originalSHA = "wrong"
  if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, bad}); err == nil {
   t.Fatal("late checksum mismatch accepted")
  }
  got := make([]byte, 8); _ = b.ReadMemory(p.address, got)
  if !bytes.Equal(got, original) { t.Fatal("earlier patch changed before validation completed") }
  overlap := q; overlap.address = p.address+4
  if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, overlap}); err == nil {
   t.Fatal("overlapping patches accepted")
  }
  if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil { t.Fatal(err) }
  if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil { t.Fatal("not idempotent", err) }
  discardWrite(t, b, q.address, original)
  if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err == nil {
   t.Fatal("mixed partial installation accepted")
  }
 })
}

func TestInotia2OfflineShopPricesAndMerchantFallback(t *testing.T) {
 for _, tc := range []struct{name string; active, item, want uint32}{
  {"blessed seal", 1, 857, 10000}, {"gem", 1, 835, 5000},
  {"elixir", 1, 943, 20000}, {"last product", 1, 949, 20000},
  {"ordinary merchant", 0, 857, 0x210007},
  {"noncatalog merchant stock", 1, 32, 0x210007},
 } {
  t.Run(tc.name, func(t *testing.T) {
   inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
    shopTestCode(t, b)
    discardWord(t, b, inotia2OfflineShopFlag, tc.active)
    discardWord(t, b, 0x210008, tc.item<<6)
    // Synthetic original price routine checks fallback receives the intact item pointer.
    discardWrite(t, b, 0x149f00, []byte{7, 0x30, 0x70, 0x47})
    discardWrite(t, b, 0x14aed0, []byte{0x70, 0xbd})
    discardReg(t, b, cpu.RegisterR0, 0x210000)
    discardRun(t, b, 0x14aec8)
    discardCheckReg(t, b, cpu.RegisterR0, tc.want)
    shopTestPreserved(t, b)
   })
  })
 }
}

func TestInotia2OfflineShopEntryCleanup(t *testing.T) {
 inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
  shopTestCode(t, b)
  // Synthetic state dispatcher records the requested state; no network stubs are called.
  discardWrite(t, b, 0x176fd8, inotia2PatchBytes("014b18607047c04600002100"))
  discardRun(t, b, 0x120890)
  if discardRead(t, b, inotia2OfflineShopFlag) != 1 || discardRead(t, b, 0x210000) != 16 {
   t.Fatal("cash entry did not open the local merchant state")
  }
  shopTestPreserved(t, b)
  discardWord(t, b, 0x1924c4+0x1048, 0x215010)
  discardWord(t, b, 0x215010, 1)
  discardWord(t, b, inotia2OfflineShopFlag+12, 3)
  discardWrite(t, b, 0x13f8a4, []byte{0x70, 0x47})
  discardReg(t, b, cpu.RegisterLR, 0x200101)
  discardRun(t, b, inotia2OfflineShopCleanup)
  if discardRead(t, b, inotia2OfflineShopFlag) != 0 || discardRead(t, b, 0x215010) != 3 {
   t.Fatal("local shop scope or previous merchant presentation leaked after exit")
  }
  shopTestPreserved(t, b)
 })
}

func TestInotia2OfflineShopStockInitialization(t *testing.T) {
 for _, active := range []uint32{0,1} {
  t.Run(fmt.Sprintf("active=%d", active), func(t *testing.T) {
   inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
    shopTestCode(t, b)
    discardWord(t, b, inotia2OfflineShopFlag, active)
    // Every synthetic native call has the same argument/return contract as the
    // observed client. Constructor and insertion log IDs, never proprietary items.
    for _, address := range []uint32{0x167e10,0x13f8a4,0x14a0bc,0x11fc90,0x164b24,0x168ec8} {
     discardWrite(t, b, address, []byte{0x70,0x47})
    }
    discardWrite(t, b, 0x16816c, inotia2PatchBytes("0048704700002100"))
    discardWrite(t, b, 0x13f6b0, inotia2PatchBytes("8900024b585001207047c04600002200"))
    discardWrite(t, b, 0x167ef0, []byte{0,0x20,0x70,0x47})
    discardWrite(t, b, 0x1205b0, inotia2PatchBytes("0cbc90469a4670bd"))
    discardWord(t, b, 0x1924c4+0x598, 0x215000)
    discardWord(t, b, 0x1924c4+0x1048, 0x215010)
    discardWord(t, b, 0x215010, 3)
    discardRun(t, b, 0x1205a8)
    if active == 0 {
     if discardRead(t, b, 0x215000) != 0 { t.Fatal("ordinary merchant initialization intercepted") }
    } else {
     if discardRead(t, b, 0x215000) != 0x210000 { t.Fatal("shop widget not installed") }
     for index, id := range []uint32{857,665,835,836,839,943,944,945,946,947,949} {
      if got := discardRead(t, b, 0x220000+uint32(index)*4); got != id { t.Fatalf("stock %d = %d, want %d", index, got, id) }
     }
     var widget [40]byte; if err := b.ReadMemory(0x210000, widget[:]); err != nil { t.Fatal(err) }
     if widget[1] != 6 || widget[3] != 6 || widget[4] != 4 || widget[11] != 0 || widget[12] != 11 {
      t.Fatalf("wrong stock widget geometry: %v", widget[:16])
     }
     if discardRead(t, b, inotia2OfflineShopFlag+12) != 3 || discardRead(t, b, 0x215010) != 1 {
      t.Fatal("shop presentation not activated or previous setting lost")
     }
    }
    shopTestPreserved(t, b)
   })
  })
 }
}
