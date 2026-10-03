package application

import (
 "bytes"
 "testing"
 "github.com/mirusu400/aram-core/cpu"
)

func TestInotia2AutoIdentifyUnknownTitle(t *testing.T) {
 if err := installInotia2AutoIdentify([]byte("unrelated title"), nil); err != nil { t.Fatal(err) }
}

func TestInotia2AutoIdentifyInstallation(t *testing.T) {
 inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
  for _, p := range inotia2AutoIdentifyPatches { discardWrite(t, b, p.address, p.replacement) }
  if err := installInotia2OfflineShopPatches(b, inotia2AutoIdentifyPatches); err != nil {
   t.Fatal("current acquisition hook not idempotent",err)
  }
  // A partly installed state is not accepted or repaired silently. Validate
  // both spans before writing either, including a late corrupt hook.
  zero := make([]byte,len(inotia2AutoIdentifyPatches[0].replacement))
  discardWrite(t,b,inotia2AutoIdentifyHelper,zero)
  if err := installInotia2OfflineShopPatches(b,inotia2AutoIdentifyPatches); err == nil {
   t.Fatal("mixed original helper/current hook accepted")
  }
  discardWrite(t,b,inotia2AutoIdentifyHook,[]byte("bad hook"))
  if err := installInotia2OfflineShopPatches(b,inotia2AutoIdentifyPatches); err == nil {
   t.Fatal("corrupt acquisition hook accepted")
  }
  got := make([]byte,len(zero)); if err := b.ReadMemory(inotia2AutoIdentifyHelper,got); err != nil {t.Fatal(err)}
  if !bytes.Equal(got,zero) {t.Fatal("helper written before hook validation")}
 })
}

func TestInotia2AutoIdentifyAcquisitionContract(t *testing.T) {
 // Synthetic native callees exercise call order, the live incoming pointer,
 // possible ID resolution, quantity return, and preserved native stack/frame.
 // Proprietary identification logic is verified only in private game replay.
 for _, tc := range []struct{name string; initial, resolved uint32}{
  {"equipment",941,0},{"native ID resolution",34,35},{"already identified",950,0},
  {"stackable consumable",720,0},{"bag",4,0},
 } {
  t.Run(tc.name,func(t *testing.T){
   inotia2DiscardBackends(t,func(t *testing.T,b cpu.Backend){
    for _, p := range inotia2AutoIdentifyPatches {discardWrite(t,b,p.address,p.replacement)}
    discardWrite(t,b,0x14a064,inotia2PatchBytes("10b50446064b186001229a601a69002a00d02281112122223323204610bdc04600502100"))
    discardWrite(t,b,0x147eb4,inotia2PatchBytes("034b58609a6801329a6025207047c04600502100"))
    discardWrite(t,b,0x1476d4,[]byte{0,0xbe})
    discardWord(t,b,0x210000,0x12345678)
    discardWord(t,b,0x210008,tc.initial<<6|8)
    discardWord(t,b,0x21000c,0x64543210)
    discardWord(t,b,0x210014,0x220000)
    discardWord(t,b,0x210100,0x43)
    if tc.resolved!=0 {discardWord(t,b,0x215010,tc.resolved<<6|8)}
    for r,v := range map[uint32]uint32{
     cpu.RegisterR0:tc.initial,cpu.RegisterR1:0x11111111,
     cpu.RegisterR4:0x44444444,cpu.RegisterR5:tc.initial,
     cpu.RegisterR6:0x210000,cpu.RegisterR7:0x210100,
     cpu.RegisterR8:0x88888888,cpu.RegisterR10:0xaaaaaaaa,
     cpu.RegisterLR:0x200101,
    } {discardReg(t,b,r,v)}
    discardRun(t,b,inotia2AutoIdentifyHook)
    if discardRead(t,b,0x215000)!=0x210000 || discardRead(t,b,0x215004)!=0x210000 || discardRead(t,b,0x215008)!=2 {
     t.Fatal("identify/quantity call order or incoming pointer changed")
    }
    want := tc.initial; if tc.resolved!=0 {want=tc.resolved}
    for r,v := range map[uint32]uint32{
     cpu.RegisterR0:37,cpu.RegisterR1:0x11111111,cpu.RegisterR2:0x43,
     cpu.RegisterR4:0x44444444,cpu.RegisterR5:want,
     cpu.RegisterR6:0x210000,cpu.RegisterR7:0x210100,
     cpu.RegisterR8:0x88888888,cpu.RegisterR10:0xaaaaaaaa,
     cpu.RegisterLR:0x200101,cpu.RegisterSP:0x2ff000,
    } {discardCheckReg(t,b,r,v)}
    if discardRead(t,b,0x210000)!=0x12345678 || discardRead(t,b,0x21000c)!=0x64543210 || discardRead(t,b,0x210014)!=0x220000 {
     t.Fatal("wrapper changed UID, metadata or option linkage")
    }
   })
  })
 }
}
