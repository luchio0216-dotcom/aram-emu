package application

import (
    "bytes"
    "encoding/binary"
    "testing"

    "github.com/mirusu400/aram-core/cpu"
)

func TestInotia2ExperienceUnknownTitle(t *testing.T) {
    if err := installInotia2Experience([]byte("unrelated title"), nil); err != nil {
        t.Fatal(err)
    }
}

func TestInotia2ExperienceInstallation(t *testing.T) {
    inotia2DiscardBackends(t,func(t *testing.T,b cpu.Backend){
        // Assemble the original BL contract without using an extracted fixture.
        delta := int32(0x135550)-int32(inotia2ExperienceHook)-4
        original := make([]byte,4)
        binary.LittleEndian.PutUint16(original,0xf000|uint16((delta>>12)&0x7ff))
        binary.LittleEndian.PutUint16(original[2:],0xf800|uint16((delta>>1)&0x7ff))
        discardWrite(t,b,inotia2ExperienceHook,original)
        if err:=installInotia2OfflineShopPatches(b,inotia2ExperiencePatches);err!=nil {t.Fatal(err)}
        if err:=installInotia2OfflineShopPatches(b,inotia2ExperiencePatches);err!=nil {t.Fatal("not idempotent",err)}
        zero:=make([]byte,len(inotia2ExperiencePatches[0].replacement))
        discardWrite(t,b,inotia2ExperienceHelper,zero)
        if err:=installInotia2OfflineShopPatches(b,inotia2ExperiencePatches);err==nil {t.Fatal("partial image accepted")}
        discardWrite(t,b,inotia2ExperienceHook,[]byte("bad!"))
        if err:=installInotia2OfflineShopPatches(b,inotia2ExperiencePatches);err==nil {t.Fatal("corrupt hook accepted")}
        got:=make([]byte,len(zero));if err:=b.ReadMemory(inotia2ExperienceHelper,got);err!=nil {t.Fatal(err)}
        if !bytes.Equal(got,zero){t.Fatal("helper written before hook validation")}
    })
}

func TestInotia2ExperienceLevelBoundaryAndAwardABI(t *testing.T) {
    inotia2DiscardBackends(t,func(t *testing.T,b cpu.Backend){
        for _,p:=range inotia2ExperiencePatches{discardWrite(t,b,p.address,p.replacement)}
        // Authored award sink records actor and reward, then returns to the
        // original call's continuation. It represents the native level-up API.
        discardWrite(t,b,0x135550,inotia2PatchBytes("014a10605160704700502100"))
        discardWrite(t,b,0x154ccc,inotia2PatchBytes("00be"))
        for _,level:=range []uint32{1,39,40,41,42,46,50,99,255}{
            for _,reward:=range []uint32{0,1,137,358,382,10_000}{
                discardReg(t,b,cpu.RegisterR0,0x210000)
                discardReg(t,b,cpu.RegisterR1,reward)
                for r:=uint32(cpu.RegisterR4);r<=cpu.RegisterR12;r++{discardReg(t,b,r,0x11110000+r)}
                discardReg(t,b,cpu.RegisterR10,level)
                discardRun(t,b,inotia2ExperienceHook)
                want:=reward;if level>=41{want*=4}
                if got:=discardRead(t,b,0x215000);got!=0x210000{t.Fatalf("actor changed: %#x",got)}
                if got:=discardRead(t,b,0x215004);got!=want{t.Fatalf("level %d reward %d: got %d want %d",level,reward,got,want)}
                for r:=uint32(cpu.RegisterR4);r<=cpu.RegisterR12;r++{
                    v:=uint32(0x11110000)+r;if r==cpu.RegisterR10{v=level};discardCheckReg(t,b,r,v)
                }
                discardCheckReg(t,b,cpu.RegisterSP,0x2ff000)
                discardCheckReg(t,b,cpu.RegisterLR,0x154ccd)
            }
        }
    })
}
