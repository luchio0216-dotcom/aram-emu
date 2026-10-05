package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
	"testing"
)

func shopTestCode(t *testing.T, b cpu.Backend) {
	t.Helper()
	for _, p := range inotia2OfflineShopPatches {
		discardWrite(t, b, p.address, p.replacement)
	}
	for r, v := range map[uint32]uint32{
		cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555,
		cpu.RegisterR6: 0x66666666, cpu.RegisterR8: 0x88888888,
		cpu.RegisterR10: 0xaaaaaaaa, cpu.RegisterLR: 0x200101,
	} {
		discardReg(t, b, r, v)
	}
}

func shopTestPreserved(t *testing.T, b cpu.Backend) {
	t.Helper()
	for r, v := range map[uint32]uint32{
		cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555,
		cpu.RegisterR6: 0x66666666, cpu.RegisterR8: 0x88888888,
		cpu.RegisterR10: 0xaaaaaaaa, cpu.RegisterSP: 0x2ff000,
	} {
		discardCheckReg(t, b, r, v)
	}
}

func TestInotia2OfflineShopUnknownTitle(t *testing.T) {
	// A nil backend also proves an unrelated image never touches guest memory.
	if err := installInotia2OfflineShop([]byte("unrelated title"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestInotia2OfflineShopInstallationGuards(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		original := []byte("abcdefgh")
		p := inotia2OfflineShopPatch{address: 0x210000,
			originalSHA: fmt.Sprintf("%x", sha256.Sum256(original)), replacement: []byte("ABCDEFGH")}
		q := p
		q.address += 16
		reset := func() {
			discardWrite(t, b, p.address, original)
			discardWrite(t, b, q.address, original)
		}
		reset()
		bad := q
		bad.originalSHA = "wrong"
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, bad}); err == nil {
			t.Fatal("late checksum mismatch accepted")
		}
		got := make([]byte, 8)
		_ = b.ReadMemory(p.address, got)
		if !bytes.Equal(got, original) {
			t.Fatal("earlier patch changed before validation completed")
		}
		overlap := q
		overlap.address = p.address + 4
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, overlap}); err == nil {
			t.Fatal("overlapping patches accepted")
		}
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil {
			t.Fatal(err)
		}
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil {
			t.Fatal("not idempotent", err)
		}
		discardWrite(t, b, q.address, original)
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err == nil {
			t.Fatal("mixed partial installation accepted")
		}
		// A complete v1013-v1016 state also upgrades; a mixed state is refused.
		previous := []byte("last24v1")
		p.previousSHA = fmt.Sprintf("%x", sha256.Sum256(previous))
		q.previousSHA = p.previousSHA
		discardWrite(t, b, p.address, previous)
		discardWrite(t, b, q.address, previous)
		p.replacement = []byte("last30v1")
		q.replacement = p.replacement
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil {
			t.Fatal("previous catalog rejected", err)
		}
		discardWrite(t, b, p.address, previous)
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err == nil {
			t.Fatal("partial previous upgrade accepted")
		}
		p.replacement = []byte("ABCDEFGH")
		q.replacement = p.replacement
		// Restoring a complete previous catalog upgrades it atomically. A partly
		// upgraded state must not change any earlier span.
		legacy := []byte("oldbuild")
		p.legacySHA = fmt.Sprintf("%x", sha256.Sum256(legacy))
		q.legacySHA = p.legacySHA
		discardWrite(t, b, p.address, legacy)
		discardWrite(t, b, q.address, legacy)
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil {
			t.Fatal("complete legacy catalog rejected", err)
		}
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err != nil {
			t.Fatal("upgraded catalog not idempotent", err)
		}
		discardWrite(t, b, p.address, legacy)
		if err := installInotia2OfflineShopPatches(b, []inotia2OfflineShopPatch{p, q}); err == nil {
			t.Fatal("partial legacy upgrade accepted")
		}
		_ = b.ReadMemory(p.address, got)
		if !bytes.Equal(got, legacy) {
			t.Fatal("legacy span changed before complete validation")
		}
	})
}

func TestInotia2OfflineShopPricesAndMerchantFallback(t *testing.T) {
	for _, tc := range []struct {
		name               string
		active, item, want uint32
	}{
		{"blessed seal", 1, 857, 10000}, {"gem", 1, 835, 5000},
		{"elixir", 1, 943, 20000}, {"original last product", 1, 949, 20000},
		{"equipment", 1, 941, 10000}, {"revival scroll", 1, 647, 5000},
		{"bag", 1, 4, 30000}, {"native pack", 1, 933, 50000},
		{"epic pack", 1, 934, 100000}, {"last product", 1, 880, 10000},
		{"chaos dice", 1, 812, 10000},
		{"class seal 970", 1, 970, 10000}, {"class seal 971", 1, 971, 10000}, {"class seal 972", 1, 972, 10000}, {"class seal 973", 1, 973, 10000}, {"class seal 974", 1, 974, 10000}, {"class seal 975", 1, 975, 10000},
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
				shopRun(t, b, 0x14aec8)
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
		shopRun(t, b, 0x120890)
		if discardRead(t, b, inotia2OfflineShopFlag) != 1 || discardRead(t, b, 0x210000) != 16 {
			t.Fatal("cash entry did not open the local merchant state")
		}
		shopTestPreserved(t, b)
		discardWord(t, b, 0x1924c4+0x1048, 0x215010)
		discardWord(t, b, 0x215010, 1)
		discardWord(t, b, inotia2OfflineShopFlag+12, 3)
		discardWrite(t, b, 0x13f8a4, []byte{0x70, 0x47})
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		shopRun(t, b, inotia2OfflineShopCleanup)
		if discardRead(t, b, inotia2OfflineShopFlag) != 0 || discardRead(t, b, 0x215010) != 3 {
			t.Fatal("local shop scope or previous merchant presentation leaked after exit")
		}
		shopTestPreserved(t, b)
	})
}

func TestInotia2OfflineShopStockInitialization(t *testing.T) {
	for category, count := range []uint32{27, 68, 132, 21} {
		t.Run(fmt.Sprintf("category=%d", category), func(t *testing.T) {
			inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
				shopTestCode(t, b)
				discardWord(t, b, inotia2OfflineShopFlag, 1)
				discardWord(t, b, inotia2OfflineShopFlag+4, uint32(category))
				for _, address := range []uint32{0x167e10, 0x13f8a4, 0x14a0bc, 0x11fc90, 0x164b24, 0x168ec8, 0x147988} {
					discardWrite(t, b, address, []byte{0x70, 0x47})
				}
				discardWrite(t, b, 0x16816c, inotia2PatchBytes("0048704700002100"))
				discardWrite(t, b, 0x125c54, inotia2PatchBytes("0048704700002300"))
				discardWrite(t, b, 0x167ef0, []byte{0, 0x20, 0x70, 0x47})
				discardWrite(t, b, 0x13f7a8, []byte{0, 0x20, 0x70, 0x47})
				discardWord(t, b, 0x192a5c, 0x215000)
				discardWord(t, b, 0x19350c, 0x215010)
				discardWord(t, b, 0x215010, 3)
				discardWord(t, b, 0x1934f4, 0x220000)
				shopRun(t, b, 0x1205a8)
				if discardRead(t, b, 0x215000) != 0x210000 {
					t.Fatal("widget not installed")
				}
				table := discardRead(t, b, 0x2add10+uint32(category)*12)
				for index := uint32(0); index < count; index++ {
					var id [2]byte
					check(t, b.ReadMemory(table+index*4, id[:]))
					if got := discardRead(t, b, 0x230004+index*8); got != uint32(binary.LittleEndian.Uint16(id[:])) {
						t.Fatalf("stock %d = %d", index, got)
					}
					if discardRead(t, b, 0x230000+index*8) != 0 {
						t.Fatal("occupied stock marked empty")
					}
				}
				var widget [40]byte
				check(t, b.ReadMemory(0x210000, widget[:]))
				if widget[1] != 6 || widget[2] != byte((count+5)/6) || widget[3] != 6 || widget[4] != 4 || widget[11] != 0 || widget[12] != byte(count) {
					t.Fatalf("geometry %v", widget[:16])
				}
				if discardRead(t, b, 0x1934f4) != 0x230000 || discardRead(t, b, inotia2OfflineShopFlag+24) != 0x210000 {
					t.Fatal("expanded stock not scoped")
				}
				shopTestPreserved(t, b)
				discardReg(t, b, cpu.RegisterLR, 0x200101)
				shopRun(t, b, inotia2OfflineShopCleanup)
				if discardRead(t, b, 0x1934f4) != 0x220000 || discardRead(t, b, inotia2OfflineShopFlag) != 0 || discardRead(t, b, 0x215010) != 3 {
					t.Fatal("stock/presentation not restored")
				}
				for index := uint32(0); index < count; index++ {
					if discardRead(t, b, 0x230004+index*8) != 0 {
						t.Fatal("stock retained after cleanup")
					}
				}
				shopTestPreserved(t, b)
			})
		})
	}
	// An ordinary merchant still enters its original body.
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		shopTestCode(t, b)
		discardWrite(t, b, 0x1205b0, inotia2PatchBytes("0cbc90469a4670bd"))
		shopRun(t, b, 0x1205a8)
		shopTestPreserved(t, b)
	})
}

func shopRun(t *testing.T, b cpu.Backend, pc uint32) {
	t.Helper()
	r := b.Run(context.Background(), pc, cpu.ModeThumb, 20000)
	if r.Err != nil || r.Reason != cpu.StopBreakpoint {
		t.Fatalf("shop helper failed: %+v", r)
	}
}

func TestInotia2OfflineShopUnsignedSelectionAndScrolling(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		shopTestCode(t, b)
		discardWord(t, b, inotia2OfflineShopFlag+24, 0x210000)
		discardWrite(t, b, 0x210000+12, []byte{132})
		// Synthetic original resolver continuations expose the exact index ABI.
		discardWrite(t, b, 0x167efa, inotia2PatchBytes("084600bd"))

		for _, index := range []uint32{0, 126, 127, 128, 131} {
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, 0x210000)
			signed := index
			if index >= 128 {
				signed |= 0xffffff00
			}
			discardReg(t, b, cpu.RegisterR1, signed)
			shopRun(t, b, 0x167ef0)
			discardCheckReg(t, b, cpu.RegisterR0, index)
			shopTestPreserved(t, b)
		}
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		discardReg(t, b, cpu.RegisterR0, 0x210100)
		discardReg(t, b, cpu.RegisterR1, 0xffffffff)
		shopRun(t, b, 0x167ef0)
		discardCheckReg(t, b, cpu.RegisterR0, 0)
		// The last armor row crosses int8's boundary. Native movement works with
		// full integers; its authored stand-in moves 125 down to 131.
		discardWrite(t, b, 0x14b35c, inotia2PatchBytes("83221a6001207047"))
		// Synthetic unsigned division, shared by the scrolling helper.
		discardWrite(t, b, 0x183db4, inotia2PatchBytes("0022884202d3401a0132fae710467047"))
		discardWrite(t, b, 0x210000, []byte{0, 6, 22, 6, 4, 0, 0, 20, 20, 2, 2, 125, 132})
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		discardReg(t, b, cpu.RegisterR0, 0x210000)
		discardReg(t, b, cpu.RegisterR1, 0xabcd)
		shopRun(t, b, 0x16827c)
		var widget [13]byte
		check(t, b.ReadMemory(0x210000, widget[:]))
		if widget[11] != 131 || widget[6] != 18 || widget[4] != 4 {
			t.Fatalf("last row unreachable or outside viewport: %v", widget)
		}
		shopTestPreserved(t, b)
		for _, tc := range []struct {
			pc, widget uint32
			want       uint32
		}{{0x2adcb0, 0x210000, 131}, {0x2adcb0, 0x210100, 0xffffffff}} {
			if tc.widget == 0x210100 {
				discardWrite(t, b, tc.widget+11, []byte{255})
			}
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, tc.widget)
			shopRun(t, b, tc.pc)
			discardCheckReg(t, b, cpu.RegisterR3, tc.want)
			shopTestPreserved(t, b)
		}
	})
}

// A stock grid is rounded to whole rows. Its padding cells must never be
// interpreted as item pointers, in either our local shop or an NPC shop.
func TestInotia2OfflineShopEmptyCells(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		shopTestCode(t, b)
		// Authored stand-in for native array resolution. Its bounds branch uses
		// the displaced null-return address, reproducing the v1020 regression.
		discardWrite(t, b, 0x167efa, inotia2PatchBytes("037b9942fada02698b00985800bd"))
		for _, tc := range []struct {
			name  string
			count uint32
			local bool
		}{
			{"consumables", 27, true}, {"weapons", 68, true}, {"accessories", 21, true},
			{"NPC weapon shop", 17, false}, {"NPC armor shop", 33, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				widget := uint32(0x210000)
				scoped := uint32(0)
				if tc.local {
					scoped = widget
				}
				discardWord(t, b, inotia2OfflineShopFlag+24, scoped)
				discardWrite(t, b, widget+12, []byte{byte(tc.count)})
				discardWord(t, b, widget+16, 0x220000)
				discardWord(t, b, 0x220000+(tc.count-1)*4, 0x230000)
				for index := tc.count; index < ((tc.count+5)/6)*6; index++ {
					discardReg(t, b, cpu.RegisterLR, 0x200101)
					discardReg(t, b, cpu.RegisterR0, widget)
					discardReg(t, b, cpu.RegisterR1, index)
					shopRun(t, b, 0x167ef0)
					discardCheckReg(t, b, cpu.RegisterR0, 0)
					shopTestPreserved(t, b)
				}
				for _, index := range []uint32{tc.count, 255, 0xffffffff} {
					discardReg(t, b, cpu.RegisterLR, 0x200101)
					discardReg(t, b, cpu.RegisterR0, widget)
					discardReg(t, b, cpu.RegisterR1, index)
					shopRun(t, b, 0x167ef0)
					discardCheckReg(t, b, cpu.RegisterR0, 0)
					shopTestPreserved(t, b)
				}
				discardReg(t, b, cpu.RegisterLR, 0x200101)
				discardReg(t, b, cpu.RegisterR0, widget)
				discardReg(t, b, cpu.RegisterR1, tc.count-1)
				shopRun(t, b, 0x167ef0)
				discardCheckReg(t, b, cpu.RegisterR0, 0x230000)
				shopTestPreserved(t, b)
			})
		}
	})
}
