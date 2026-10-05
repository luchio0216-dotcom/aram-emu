package application

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
	"testing"
)

const swapTestInventory = uint32(0x240000)
const swapTestHeld = uint32(0x230100)
const swapTestGrid = uint32(0x230120)

func swapTestCode(t *testing.T, b cpu.Backend) {
	skillBooksTestCode(t, b)
	for _, p := range inotia2InventorySwapPatches {
		discardWrite(t, b, p.address, p.replacement)
	}
	// Authored stand-ins for native quantity, inventory lookup and transaction.
	// The transaction captures its arguments and returns a configured outcome;
	// real native movement/stacking is covered by private game integration runs.
	discardWrite(t, b, 0x147eb4, inotia2PatchBytes("c068400e7047"))
	discardWrite(t, b, 0x146dd4, inotia2PatchBytes("30b504460d460a4b1b6800221968a14205d004330132602af8d1002030bd10460f2108401209520110432870012030bd6c2d1900"))
	discardWrite(t, b, 0x147464, inotia2PatchBytes("10b4044c6060a160e2602361206810bc7047c04600102300"))
	discardWrite(t, b, 0x1675a0, inotia2PatchBytes("024b0020186058609860704700012300"))
	for a, v := range map[uint32]uint32{
		0x192d6c: swapTestInventory, 0x192fac: swapTestHeld,
		0x192a6c: 0x230160, 0x192f94: 0x230164,
		0x1929b0: 0x230168, 0x192f7c: 0x23016c,
		0x230160: swapTestGrid, 0x230168: 1, 0x23016c: 1,
	} {
		discardWord(t, b, a, v)
	}
}

func swapTestItem(t *testing.T, b cpu.Backend, ordinal, quantity uint32) uint32 {
	p := uint32(0x250000) + ordinal*32
	data := make([]byte, 24)
	for i := range data {
		data[i] = byte(ordinal*19 + uint32(i)*7 + 3)
	}
	binary.LittleEndian.PutUint16(data[8:], uint16((ordinal+400)<<6|11))
	binary.LittleEndian.PutUint32(data[12:], quantity<<25|0x1234567)
	// The option-list pointer is deliberately distinct from its item/quantity.
	binary.LittleEndian.PutUint32(data[20:], 0x260000+ordinal*8)
	discardWrite(t, b, p, data)
	discardWrite(t, b, 0x260000+ordinal*8, []byte{29, 5, byte(ordinal), 1, 0, 0, 0, 0})
	return p
}

func swapTestSelect(t *testing.T, b cpu.Backend, source, quantity, bag, slot uint32) {
	discardWord(t, b, swapTestHeld, 2|quantity<<16)
	discardWord(t, b, swapTestHeld+4, source)
	discardWord(t, b, 0x230164, bag)
	discardWrite(t, b, swapTestGrid+11, []byte{byte(slot), 16})
	discardWord(t, b, swapTestGrid+16, swapTestInventory+bag*64)
}

func swapTestMove(t *testing.T, b cpu.Backend, item, quantity, bag, slot uint32) uint32 {
	discardReg(t, b, cpu.RegisterLR, 0x200101)
	for r, v := range []uint32{item, quantity, bag, slot} {
		discardReg(t, b, uint32(r), v)
	}
	discardRun(t, b, inotia2InventorySwapMove)
	value, err := b.ReadRegister(cpu.RegisterR0)
	if err != nil {
		t.Fatal(err)
	}
	skillBooksPreserved(t, b)
	return value
}

func swapTestRead(t *testing.T, b cpu.Backend, p uint32, n int) []byte {
	data := make([]byte, n)
	if err := b.ReadMemory(p, data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInotia2InventorySwapUnknownTitleAndLayout(t *testing.T) {
	if err := installInotia2InventorySwap([]byte("unrelated client"), nil); err != nil {
		t.Fatal(err)
	}
	var occupied []inotia2OfflineShopPatch
	for _, ps := range [][]inotia2OfflineShopPatch{inotia2OfflineShopPatches, inotia2AutoIdentifyPatches,
		inotia2CombatGeometryPatches, inotia2CombatTargetsPatches, inotia2ExperiencePatches,
		inotia2SkillBooksPatches, inotia2SkillBooksMenuPatches, inotia2InventorySwapPatches} {
		occupied = append(occupied, ps...)
	}
	occupied = append(occupied, inotia2OfflineShopPatch{address: inotia2DiscardHelperAddress, replacement: inotia2DiscardHelper})
	for i, p := range occupied {
		for _, q := range occupied[:i] {
			if uint64(p.address) < uint64(q.address)+uint64(len(q.replacement)) && uint64(q.address) < uint64(p.address)+uint64(len(p.replacement)) {
				t.Fatalf("patches overlap at %x and %x", p.address, q.address)
			}
		}
		if p.address >= 0x2ae000 && p.address+uint32(len(p.replacement)) > 0x2af000 {
			t.Fatal("helper exceeds existing executable page")
		}
	}
}

func TestInotia2InventorySwapChainPreservesItemsAndCancel(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		swapTestCode(t, b)
		quantities := []uint32{1, 86, 99, 3, 15}
		pointers := make([]uint32, len(quantities))
		data := make(map[uint32][]byte)
		options := make(map[uint32][]byte)
		for i, q := range quantities {
			p := swapTestItem(t, b, uint32(i), q)
			pointers[i] = p
			discardWord(t, b, swapTestInventory+uint32(i)*4, p)
			data[p] = swapTestRead(t, b, p, 24)
			options[p] = swapTestRead(t, b, 0x260000+uint32(i)*8, 8)
		}
		swapTestSelect(t, b, pointers[0], quantities[0], 0, 1)
		// Each displaced item remains at the original source slot. The expected
		// permutation is checked separately after every step, including a repeat.
		order := append([]uint32(nil), pointers...)
		held := pointers[0]
		for _, destination := range []uint32{1, 2, 4, 3, 1, 4, 2} {
			quantity := discardRead(t, b, swapTestHeld) >> 16 & 255
			swapTestSelect(t, b, held, quantity, 0, destination)
			displaced := order[destination]
			if got := swapTestMove(t, b, held, quantity, 0, destination); got != 2 {
				t.Fatalf("swap returned %d", got)
			}
			order[0], order[destination] = displaced, held
			held = displaced
			if discardRead(t, b, swapTestHeld+4) != held {
				t.Fatal("displaced item is not held")
			}
			for i, want := range order {
				if discardRead(t, b, swapTestInventory+uint32(i)*4) != want {
					t.Fatalf("step slot %d ownership changed", i)
				}
				if !bytes.Equal(data[want], swapTestRead(t, b, want, 24)) {
					t.Fatal("item seeds, flags, quantity or option pointer changed")
				}
			}
			for i, p := range pointers {
				if !bytes.Equal(options[p], swapTestRead(t, b, 0x260000+uint32(i)*8, 8)) {
					t.Fatal("equipment/skill options changed")
				}
			}
		}
		// Native cancel only clears the held overlay. No slot/quantity is lost.
		before := swapTestRead(t, b, swapTestInventory, 96*4)
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		discardRun(t, b, 0x1675a0)
		if discardRead(t, b, swapTestHeld) != 0 || !bytes.Equal(before, swapTestRead(t, b, swapTestInventory, 96*4)) {
			t.Fatal("cancel changed inventory ownership")
		}
	})
}

func TestInotia2InventorySwapBagBoundaries(t *testing.T) {
	for sourceBag := uint32(0); sourceBag < 6; sourceBag++ {
		for destinationBag := uint32(0); destinationBag < 6; destinationBag++ {
			t.Run(fmt.Sprintf("%d-to-%d", sourceBag, destinationBag), func(t *testing.T) {
				inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
					swapTestCode(t, b)
					a := swapTestItem(t, b, 0, 99)
					c := swapTestItem(t, b, 1, 1)
					source := swapTestInventory + sourceBag*64 + 15*4
					destination := swapTestInventory + destinationBag*64
					discardWord(t, b, source, a)
					discardWord(t, b, destination, c)
					swapTestSelect(t, b, a, 99, destinationBag, 0)
					got := swapTestMove(t, b, a, 99, destinationBag, 0)
					if (sourceBag == 5) != (destinationBag == 5) {
						if got != 0 || discardRead(t, b, source) != a || discardRead(t, b, destination) != c {
							t.Fatal("crossed the native special-bag boundary")
						}
					} else if got != 2 || discardRead(t, b, source) != c || discardRead(t, b, destination) != a || discardRead(t, b, swapTestHeld)>>16&255 != 1 {
						t.Fatal("whole-stack cross-bag swap failed")
					}
				})
			})
		}
	}
}

func TestInotia2InventorySwapGuardsAndNativeDelegation(t *testing.T) {
	cases := []string{"equipment", "header", "use", "quantity", "partial", "same-slot", "empty", "missing-source", "bad-bag", "bad-slot", "small-bag", "wrong-grid", "different-held", "zero-target-quantity", "native-success"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
				swapTestCode(t, b)
				a := swapTestItem(t, b, 0, 86)
				c := swapTestItem(t, b, 1, 99)
				discardWord(t, b, swapTestInventory, a)
				discardWord(t, b, swapTestInventory+4, c)
				quantity, bag, slot := uint32(86), uint32(0), uint32(1)
				swapTestSelect(t, b, a, quantity, bag, slot)
				switch name {
				case "equipment":
					discardWord(t, b, 0x23016c, 0)
				case "header":
					discardWord(t, b, 0x230168, 0)
				case "use":
					discardWrite(t, b, swapTestHeld, []byte{1, 0})
				case "quantity":
					discardWrite(t, b, swapTestHeld, []byte{3, 0})
				case "partial":
					quantity = 20
					discardWrite(t, b, swapTestHeld+2, []byte{20})
				case "same-slot":
					slot = 0
					discardWrite(t, b, swapTestGrid+11, []byte{0})
				case "empty":
					discardWord(t, b, swapTestInventory+4, 0)
				case "missing-source":
					discardWord(t, b, swapTestInventory, 0)
				case "bad-bag":
					bag = 6
					discardWord(t, b, 0x230164, 6)
				case "bad-slot":
					slot = 16
					discardWrite(t, b, swapTestGrid+11, []byte{16})
				case "small-bag":
					discardWrite(t, b, swapTestGrid+12, []byte{1})
				case "wrong-grid":
					discardWord(t, b, swapTestGrid+16, swapTestInventory+64)
				case "different-held":
					discardWord(t, b, swapTestHeld+4, c)
				case "zero-target-quantity":
					discardWord(t, b, c+12, 0x1234567)
				case "native-success":
					discardWord(t, b, 0x231000, 1)
				}
				before := swapTestRead(t, b, swapTestInventory, 96*4)
				held := swapTestRead(t, b, swapTestHeld, 12)
				want := uint32(0)
				if name == "native-success" {
					want = 1
				}
				if got := swapTestMove(t, b, a, quantity, bag, slot); got != want {
					t.Fatalf("returned %d, want %d", got, want)
				}
				if !bytes.Equal(before, swapTestRead(t, b, swapTestInventory, 96*4)) || !bytes.Equal(held, swapTestRead(t, b, swapTestHeld, 12)) {
					t.Fatal("rejected/native transaction changed ownership or held state")
				}
				for i, value := range []uint32{a, quantity, bag, slot} {
					if discardRead(t, b, 0x231004+uint32(i)*4) != value {
						t.Fatal("native transaction did not receive original arguments first")
					}
				}
			})
		})
	}
}

func TestInotia2InventorySwapHighlightAndCompletion(t *testing.T) {
	for _, valid := range []bool{false, true} {
		inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
			swapTestCode(t, b)
			a := swapTestItem(t, b, 0, 1)
			c := swapTestItem(t, b, 1, 86)
			discardWord(t, b, swapTestInventory, a)
			discardWord(t, b, swapTestInventory+4, c)
			swapTestSelect(t, b, a, 1, 0, 1)
			if !valid {
				discardWord(t, b, 0x230168, 0)
			}
			before := swapTestRead(t, b, swapTestInventory, 96*4)
			held := swapTestRead(t, b, swapTestHeld, 12)
			// Recreate the renderer's existing stack argument, execute the replaced
			// callsite, and inspect the flags used by its original conditional branch.
			discardWord(t, b, 0x2ff00c, c)
			discardReg(t, b, cpu.RegisterR7, a)
			discardWrite(t, b, 0x1679e8, []byte{0, 0xbe})
			discardRun(t, b, 0x1679e4)
			cpsr, err := b.ReadRegister(cpu.RegisterCPSR)
			if err != nil {
				t.Fatal(err)
			}
			if (cpsr&(1<<30) != 0) != valid {
				t.Fatal("highlight flags do not match swap acceptance")
			}
			if !bytes.Equal(before, swapTestRead(t, b, swapTestInventory, 96*4)) || !bytes.Equal(held, swapTestRead(t, b, swapTestHeld, 12)) {
				t.Fatal("renderer mutated inventory")
			}
			discardReg(t, b, cpu.RegisterR7, 0x77777777)
			for _, result := range []uint32{2, 1} {
				swapTestSelect(t, b, a, 1, 0, 1)
				discardReg(t, b, cpu.RegisterR0, result)
				discardReg(t, b, cpu.RegisterLR, 0x200101)
				discardRun(t, b, inotia2InventorySwapFinish)
				want := uint32(0)
				if result == 2 {
					want = a
				}
				if discardRead(t, b, swapTestHeld+4) != want {
					t.Fatal("completion cleared a chained item or retained a finished move")
				}
				skillBooksPreserved(t, b)
			}
		})
	}
}
