package application

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
)

// Exercise the emitted Thumb helpers against synthetic native ground entries.
// ARM64 also exercises them through the same native backend used on Android.
func inotia2DiscardBackends(t *testing.T, test func(*testing.T, cpu.Backend)) {
	t.Helper()
	backends := []struct {
		name   string
		newCPU func() cpu.Backend
	}{
		{"portable", func() cpu.Backend { return interpreter.New() }}, {"jit", func() cpu.Backend { return interpreter.NewJIT() }},
	}
	if native, err := ResolveCPUBackend("native"); err == nil {
		backends = append(backends, struct {
			name   string
			newCPU func() cpu.Backend
		}{"native", native})
	}
	for _, tc := range backends {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.newCPU()
			t.Cleanup(func() { _ = b.Close() })
			if err := b.Map(0x100000, 0x200000, cpu.PermissionRead|cpu.PermissionWrite|cpu.PermissionExecute); err != nil {
				t.Fatal(err)
			}
			discardWrite(t, b, inotia2DiscardHelperAddress, inotia2DiscardHelper)
			discardWrite(t, b, 0x200100, []byte{0, 0xbe})
			discardReg(t, b, cpu.RegisterSP, 0x2ff000)
			test(t, b)
		})
	}
}
func discardWrite(t *testing.T, b cpu.Backend, a uint32, data []byte) {
	t.Helper()
	if err := b.WriteMemory(a, data); err != nil {
		t.Fatal(err)
	}
}
func discardWord(t *testing.T, b cpu.Backend, a, v uint32) {
	t.Helper()
	var data [4]byte
	binary.LittleEndian.PutUint32(data[:], v)
	discardWrite(t, b, a, data[:])
}
func discardRead(t *testing.T, b cpu.Backend, a uint32) uint32 {
	t.Helper()
	var data [4]byte
	if err := b.ReadMemory(a, data[:]); err != nil {
		t.Fatal(err)
	}
	return binary.LittleEndian.Uint32(data[:])
}
func discardReg(t *testing.T, b cpu.Backend, r, v uint32) {
	t.Helper()
	if err := b.WriteRegister(r, v); err != nil {
		t.Fatal(err)
	}
}
func discardRun(t *testing.T, b cpu.Backend, pc uint32) {
	t.Helper()
	r := b.Run(context.Background(), pc, cpu.ModeThumb, 1000)
	if r.Err != nil || r.Reason != cpu.StopBreakpoint {
		t.Fatalf("helper failed: %+v", r)
	}
}
func discardCheckReg(t *testing.T, b cpu.Backend, r, want uint32) {
	t.Helper()
	got, e := b.ReadRegister(r)
	if e != nil || got != want {
		t.Fatalf("register %d = %#x, %v; want %#x", r, got, e, want)
	}
}

func TestInotia2AutoLootDiscardMarksOnlyInventoryDrop(t *testing.T) {
	for _, caller := range []uint32{0x110e3f, 0x134251, 0x13d4a3, 0x200101} {
		for _, slot := range []uint32{0, 5, 15} {
			t.Run(fmt.Sprintf("caller=%x/slot=%d", caller, slot), func(t *testing.T) {
				inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
					// Model the native add-entry frame and its saved caller return address.
					discardWord(t, b, 0x2ff010, caller)
					discardWord(t, b, inotia2DiscardFlagsAddress+4*slot, 1)
					for r, v := range map[uint32]uint32{cpu.RegisterR0: slot, cpu.RegisterR1: 0x11111111, cpu.RegisterR2: 12345, cpu.RegisterR3: 0x210000, cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x220000, cpu.RegisterR6: 100, cpu.RegisterR7: 200, cpu.RegisterLR: 0x200101} {
						discardReg(t, b, r, v)
					}
					discardRun(t, b, inotia2DiscardHelperAddress)
					want := uint32(0)
					if caller == 0x110e3f {
						want = 1
					}
					if got := discardRead(t, b, inotia2DiscardFlagsAddress+4*slot); got != want {
						t.Fatalf("blocked=%d want %d", got, want)
					}
					if discardRead(t, b, 0x210000) != 0x220000 || discardRead(t, b, 0x210004) != 200<<16|100 || discardRead(t, b, 0x210008) != 12345 {
						t.Fatal("native ground entry stores changed")
					}
					for r, v := range map[uint32]uint32{cpu.RegisterR0: slot, cpu.RegisterR1: 0x11111111, cpu.RegisterR2: 12345, cpu.RegisterR3: 0x210000, cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x220000, cpu.RegisterR6: 100, cpu.RegisterR7: 200, cpu.RegisterSP: 0x2ff000} {
						discardCheckReg(t, b, r, v)
					}
				})
			})
		}
	}
}

func TestInotia2AutoLootDiscardLeaveThenReturn(t *testing.T) {
	for _, tc := range []struct {
		name                              string
		x, y                              int16
		automatic, blocked, skip, cleared bool
	}{
		{"idle on discarded item", 100, 200, true, true, true, false},
		{"inside corner", 108, 192, true, true, true, false},
		{"walk left", 91, 200, true, true, false, true},
		{"walk right", 109, 200, true, true, false, true},
		{"walk up", 100, 191, true, true, false, true},
		{"walk down", 100, 209, true, true, false, true},
		{"ordinary monster loot", 100, 200, true, false, false, false},
		{"manual OK remains available", 100, 200, false, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
				// The native loop presents x bounds in r7/r10 and y bounds on its stack.
				discardWord(t, b, 0x210004, uint32(uint16(tc.y))<<16|uint32(uint16(tc.x)))
				discardWord(t, b, 0x2ff018, 192)
				discardWord(t, b, 0x2ff01c, 208)
				if tc.automatic {
					discardWord(t, b, 0x2ad348, 1)
				}
				if tc.blocked {
					discardWord(t, b, inotia2DiscardFlagsAddress+12, 1)
				}
				// Trap two paths with distinct return values rather than fake item pickup.
				discardWrite(t, b, 0x12ce34, []byte{1, 0x20, 0, 0xbe})
				discardWrite(t, b, 0x12cd54, []byte{0, 0x20, 0, 0xbe})
				setup := func() {
					for r, v := range map[uint32]uint32{cpu.RegisterR2: 0x210000, cpu.RegisterR3: 0, cpu.RegisterR5: 3, cpu.RegisterR7: 92, cpu.RegisterR10: 108, cpu.RegisterLR: 0x12cd55} {
						discardReg(t, b, r, v)
					}
				}
				setup()
				discardRun(t, b, 0x2ae020)
				want := uint32(0)
				if tc.skip {
					want = 1
				}
				discardCheckReg(t, b, cpu.RegisterR0, want)
				flag := uint32(0)
				if tc.blocked && !tc.cleared {
					flag = 1
				}
				if got := discardRead(t, b, inotia2DiscardFlagsAddress+12); got != flag {
					t.Fatalf("blocked=%d want %d", got, flag)
				}
				discardCheckReg(t, b, cpu.RegisterSP, 0x2ff000)
				if tc.cleared {
					// Once released, coming back over the item enters normal native pickup.
					discardWord(t, b, 0x210004, 200<<16|100)
					setup()
					discardRun(t, b, 0x2ae020)
					discardCheckReg(t, b, cpu.RegisterR0, 0)
				}
			})
		})
	}
}

func TestInotia2AutoLootDiscardFlagsFollowNativeSwapAndFieldReset(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		// The native copy stub returns without changing its inputs.
		discardWrite(t, b, 0x14f0f8, []byte{0x70, 0x47})
		discardWord(t, b, inotia2DiscardFlagsAddress+15*4, 1)
		discardWord(t, b, inotia2DiscardFlagsAddress+2*4, 0)
		for r, v := range map[uint32]uint32{cpu.RegisterR0: 0x210000, cpu.RegisterR1: 0x220000, cpu.RegisterR2: 22, cpu.RegisterR3: 33, cpu.RegisterR4: 15, cpu.RegisterR5: 2, cpu.RegisterR7: 77, cpu.RegisterLR: 0x200101} {
			discardReg(t, b, r, v)
		}
		discardRun(t, b, 0x2ae068)
		if discardRead(t, b, inotia2DiscardFlagsAddress+2*4) != 1 {
			t.Fatal("discard flag lost when native list swaps slots")
		}
		for r, v := range map[uint32]uint32{cpu.RegisterR0: 0x210000, cpu.RegisterR1: 0x220000, cpu.RegisterR2: 22, cpu.RegisterR3: 33, cpu.RegisterR4: 15, cpu.RegisterR5: 2, cpu.RegisterR7: 77, cpu.RegisterSP: 0x2ff000} {
			discardCheckReg(t, b, r, v)
		}
		// Reusing that last slot for ordinary loot must overwrite its old flag.
		discardWord(t, b, 0x2ff010, 0x13d4a3)
		for r, v := range map[uint32]uint32{cpu.RegisterR0: 15, cpu.RegisterR2: 1, cpu.RegisterR3: 0x210000, cpu.RegisterLR: 0x200101} {
			discardReg(t, b, r, v)
		}
		discardRun(t, b, 0x2ae000)
		if discardRead(t, b, inotia2DiscardFlagsAddress+15*4) != 0 {
			t.Fatal("new ordinary drop inherited an old discard flag")
		}
		discardReg(t, b, cpu.RegisterR2, 0x194c6c)
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		discardWord(t, b, 0x194c6c, 16)
		discardRun(t, b, 0x2ae080)
		var flags [64]byte
		if err := b.ReadMemory(inotia2DiscardFlagsAddress, flags[:]); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(flags[:], make([]byte, 64)) || discardRead(t, b, 0x194c6c) != 0 {
			t.Fatal("field reset did not clear provenance and native count")
		}
	})
}
