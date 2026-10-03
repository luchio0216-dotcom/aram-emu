package application

import (
	"bytes"
	"encoding/binary"
	"github.com/mirusu400/aram-core/cpu"
	"math/rand"
	"testing"
)

func TestInotia2CombatGeometryUnknownTitle(t *testing.T) {
	if err := installInotia2CombatGeometry([]byte("unrelated title"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestInotia2CombatGeometryInstallation(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		for _, p := range inotia2CombatGeometryPatches {
			discardWrite(t, b, p.address, p.replacement)
		}
		if err := installInotia2OfflineShopPatches(b, inotia2CombatGeometryPatches); err != nil {
			t.Fatal(err)
		}
		zero := make([]byte, len(inotia2CombatGeometryPatches[0].replacement))
		discardWrite(t, b, inotia2CombatGeometryHelper, zero)
		if err := installInotia2OfflineShopPatches(b, inotia2CombatGeometryPatches); err == nil {
			t.Fatal("mixed image accepted")
		}
		discardWrite(t, b, inotia2CombatGeometryHook, []byte("bad hook"))
		if err := installInotia2OfflineShopPatches(b, inotia2CombatGeometryPatches); err == nil {
			t.Fatal("corrupt leaf accepted")
		}
		got := make([]byte, len(zero))
		if err := b.ReadMemory(inotia2CombatGeometryHelper, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, zero) {
			t.Fatal("helper written before entry validation")
		}
	})
}

func combatGeometryReference(a, b [4]int16) uint32 {
	if a[2] < b[0] || a[0] > b[2] || a[3] < b[1] || a[1] > b[3] {
		return 0
	}
	width := int32(min(a[2], b[2])) - int32(max(a[0], b[0]))
	height := int32(min(a[3], b[3])) - int32(max(a[1], b[1]))
	return uint32(width * height)
}

func TestInotia2CombatGeometryResultAndABI(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		for _, p := range inotia2CombatGeometryPatches {
			discardWrite(t, b, p.address, p.replacement)
		}
		rng := rand.New(rand.NewSource(815))
		cases := [][2][4]int16{
			{{0, 0, 10, 10}, {20, 0, 30, 10}}, {{0, 0, 10, 10}, {10, 0, 20, 10}},
			{{0, 0, 10, 10}, {-5, -5, 5, 5}}, {{0, 0, 10, 10}, {2, 2, 4, 4}},
			{{0, 0, 0, 0}, {0, 0, 0, 0}}, {{10, 0, 5, 10}, {0, 0, 20, 10}},
			{{-32768, -32768, 32767, 32767}, {-32768, -32768, 32767, 32767}},
			{{32767, 32767, -32768, -32768}, {0, 0, 10, 10}},
		}
		for range 2000 {
			var pair [2][4]int16
			for i := range pair {
				for j := range pair[i] {
					pair[i][j] = int16(rng.Uint32())
				}
			}
			cases = append(cases, pair)
		}
		for index, pair := range cases {
			for i, r := range pair {
				var encoded [8]byte
				for j, v := range r {
					binary.LittleEndian.PutUint16(encoded[j*2:], uint16(v))
				}
				discardWrite(t, b, 0x210000+uint32(i)*8, encoded[:])
			}
			discardReg(t, b, cpu.RegisterR0, 0x210000)
			discardReg(t, b, cpu.RegisterR1, 0x210008)
			for r := uint32(cpu.RegisterR4); r <= cpu.RegisterR12; r++ {
				discardReg(t, b, r, 0x11110000+r)
			}
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardRun(t, b, inotia2CombatGeometryHook)
			got, err := b.ReadRegister(cpu.RegisterR0)
			if err != nil {
				t.Fatal(err)
			}
			if want := combatGeometryReference(pair[0], pair[1]); got != want {
				t.Fatalf("case %d %v: got %d want %d", index, pair, got, want)
			}
			for r := uint32(cpu.RegisterR4); r <= cpu.RegisterR12; r++ {
				discardCheckReg(t, b, r, 0x11110000+r)
			}
			discardCheckReg(t, b, cpu.RegisterSP, 0x2ff000)
		}
	})
}
