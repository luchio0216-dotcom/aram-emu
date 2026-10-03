package application

import (
	"bytes"
	"encoding/binary"
	"github.com/mirusu400/aram-core/cpu"
	"testing"
)

func TestInotia2CombatTargetsUnknownTitle(t *testing.T) {
	if err := installInotia2CombatTargets([]byte("unrelated title"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestInotia2CombatTargetsInstallation(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		for _, p := range inotia2CombatTargetsPatches {
			discardWrite(t, b, p.address, p.replacement)
		}
		if err := installInotia2OfflineShopPatches(b, inotia2CombatTargetsPatches); err != nil {
			t.Fatal(err)
		}
		zero := make([]byte, len(inotia2CombatTargetsPatches[0].replacement))
		discardWrite(t, b, inotia2CombatTargetsHelper, zero)
		if err := installInotia2OfflineShopPatches(b, inotia2CombatTargetsPatches); err == nil {
			t.Fatal("partial image accepted")
		}
		discardWrite(t, b, inotia2CombatTargetsHook, []byte("bad hook"))
		if err := installInotia2OfflineShopPatches(b, inotia2CombatTargetsPatches); err == nil {
			t.Fatal("corrupt hook accepted")
		}
		got := make([]byte, len(zero))
		if err := b.ReadMemory(inotia2CombatTargetsHelper, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, zero) {
			t.Fatal("helper written before entry validation")
		}
	})
}

func TestInotia2CombatTargetsMissingIndexAndLiveRegisters(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		for _, p := range inotia2CombatTargetsPatches {
			discardWrite(t, b, p.address, p.replacement)
		}
		// Authored continuations: stop before any threat-table access. This verifies
		// selection of the native best-index path when the current target is absent.
		discardWrite(t, b, 0x131814, []byte{1, 0x27, 0, 0xbe})
		discardWrite(t, b, 0x13188e, []byte{0x06, 0x01, 2, 0x27, 0, 0xbe}) // lsl r6,r0,#4
		var pointer [4]byte
		binary.LittleEndian.PutUint32(pointer[:], 0x220000)
		discardWrite(t, b, 0x1924c4+0x14f8, pointer[:])
		binary.LittleEndian.PutUint32(pointer[:], 0x230000)
		discardWrite(t, b, 0x220000, pointer[:])
		cases := [][2]int32{{-1, 0}, {-1, 3}, {-2147483648, 31}, {0, 0}, {0, 3}, {3, 0}, {31, 31}, {31, 4}}
		for _, pair := range cases {
			current, best := pair[0], pair[1]
			discardReg(t, b, cpu.RegisterR0, uint32(current))
			discardReg(t, b, cpu.RegisterR1, 0x12345678)
			discardReg(t, b, cpu.RegisterR4, uint32(best))
			discardReg(t, b, cpu.RegisterR5, 0x1924c4)
			for r := uint32(cpu.RegisterR6); r <= cpu.RegisterR12; r++ {
				discardReg(t, b, r, 0x11110000+r)
			}
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardRun(t, b, inotia2CombatTargetsHook)
			want := current
			if current < 0 {
				want = best
			}
			discardCheckReg(t, b, cpu.RegisterR0, uint32(want))
			discardCheckReg(t, b, cpu.RegisterR4, uint32(best))
			discardCheckReg(t, b, cpu.RegisterR5, 0x1924c4)
			if want == best {
				discardCheckReg(t, b, cpu.RegisterR6, uint32(best)*16)
				discardCheckReg(t, b, cpu.RegisterR7, 2)
			} else {
				discardCheckReg(t, b, cpu.RegisterR1, 0x14f8)
				discardCheckReg(t, b, cpu.RegisterR3, 0x230000)
				discardCheckReg(t, b, cpu.RegisterR6, 0x11110006)
				discardCheckReg(t, b, cpu.RegisterR7, 1)
			}
			for r := uint32(cpu.RegisterR8); r <= cpu.RegisterR12; r++ {
				discardCheckReg(t, b, r, 0x11110000+r)
			}
			discardCheckReg(t, b, cpu.RegisterLR, 0x200101)
			discardCheckReg(t, b, cpu.RegisterSP, 0x2ff000)
		}
	})
}
