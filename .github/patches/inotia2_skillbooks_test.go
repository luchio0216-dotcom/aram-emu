package application

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/mirusu400/aram-core/cpu"
	"testing"
)

func skillBooksTestCode(t *testing.T, b cpu.Backend) {
	for _, p := range inotia2SkillBooksPatches {
		discardWrite(t, b, p.address, p.replacement)
	}
	for r, v := range map[uint32]uint32{cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555, cpu.RegisterR6: 0x66666666, cpu.RegisterR7: 0x77777777, cpu.RegisterR8: 0x88888888, cpu.RegisterR10: 0xaaaaaaaa, cpu.RegisterLR: 0x200101} {
		discardReg(t, b, r, v)
	}
}
func skillBooksPreserved(t *testing.T, b cpu.Backend) {
	for r, v := range map[uint32]uint32{cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555, cpu.RegisterR6: 0x66666666, cpu.RegisterR7: 0x77777777, cpu.RegisterR8: 0x88888888, cpu.RegisterR10: 0xaaaaaaaa, cpu.RegisterSP: 0x2ff000} {
		discardCheckReg(t, b, r, v)
	}
}
func TestInotia2SkillBooksUnknownTitle(t *testing.T) {
	if err := installInotia2SkillBooks([]byte("another game"), nil); err != nil {
		t.Fatal(err)
	}
}
func TestInotia2SkillBooksClassSelectionAndCapacity(t *testing.T) {
	for class := uint32(0); class < 6; class++ {
		for ordinal := uint32(0); ordinal < 12; ordinal++ {
			for _, capacity := range []uint32{0, 1} {
				t.Run(fmt.Sprintf("class=%d/skill=%d/capacity=%d", class, ordinal, capacity), func(t *testing.T) {
					inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
						skillBooksTestCode(t, b)
						// Synthetic constructor, RNG, option setter, insertion, release and popup.
						discardWrite(t, b, 0x14a0bc, inotia2PatchBytes("014b1860014870470000230000002100"))
						discardWrite(t, b, 0x125bb4, []byte{byte(ordinal), 0x20, 0x70, 0x47})
						discardWrite(t, b, 0x147b74, inotia2PatchBytes("10b4044c20606160a260e360012010bc7047c04610002300"))
						discardWrite(t, b, 0x147688, []byte{byte(capacity), 0x20, 0x70, 0x47})
						discardWrite(t, b, 0x147988, inotia2PatchBytes("014b18607047c04630002300"))
						discardWrite(t, b, 0x16986c, []byte{0x70, 0x47})
						discardReg(t, b, cpu.RegisterR0, 970+class)
						discardRun(t, b, 0x14a528)
						if got := discardRead(t, b, 0x230000); got != 366 {
							t.Fatalf("constructed item %d", got)
						}
						want := class*16 + ordinal
						if ordinal >= 6 {
							want += 2
						}
						if got := discardRead(t, b, 0x23001c); got != want {
							t.Fatalf("skill %d, want %d", got, want)
						}
						for a, v := range map[uint32]uint32{0x230010: 0x210000, 0x230014: 0, 0x230018: 29} {
							if discardRead(t, b, a) != v {
								t.Fatalf("bad native option contract at %x", a)
							}
						}
						discardCheckReg(t, b, cpu.RegisterR0, capacity)
						wantRelease := uint32(0)
						if capacity == 0 {
							wantRelease = 0x210000
						}
						if discardRead(t, b, 0x230030) != wantRelease {
							t.Fatal("full bag leaked item or successful insertion released it")
						}
						skillBooksPreserved(t, b)
					})
				})
			}
		}
	}
}
func TestInotia2SkillBooksNamesAndOriginalFallback(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		skillBooksTestCode(t, b)
		for class := uint32(0); class < 6; class++ {
			for _, description := range []uint32{0, 1} {
				discardReg(t, b, cpu.RegisterLR, 0x200101)
				discardReg(t, b, cpu.RegisterR0, 970+class)
				discardReg(t, b, cpu.RegisterR1, description)
				discardRun(t, b, 0x148c9c)
				want := discardRead(t, b, 0x2aec80+4*(class+description*6))
				discardCheckReg(t, b, cpu.RegisterR0, want)
				skillBooksPreserved(t, b)
			}
		}
		// Authored stand-ins resume each preserved original prologue and return.
		discardWrite(t, b, 0x148ca4, []byte{0x70, 0xbd})
		discardWrite(t, b, 0x14a530, []byte{0x30, 0xbd})
		for _, id := range []uint32{0, 839, 969, 976, 1023} {
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, id)
			discardReg(t, b, cpu.RegisterR1, 0)
			discardRun(t, b, 0x148c9c)
			discardCheckReg(t, b, cpu.RegisterR0, id)
			skillBooksPreserved(t, b)
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, id)
			discardRun(t, b, 0x14a528)
			discardCheckReg(t, b, cpu.RegisterR0, id-813)
			skillBooksPreserved(t, b)
		}
	})
}
func TestInotia2SkillBooksExtendNativeDefinitions(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		skillBooksTestCode(t, b)
		original := make([]byte, 970*17)
		for i := range original {
			original[i] = byte(i*19 + 7)
		}
		discardWrite(t, b, 0x210000, original)
		discardWord(t, b, 0x194828, 0x210000)
		discardWrite(t, b, 0x194824, []byte{0xca, 3, 17, 0})
		// Host-free native stand-ins for allocate and memcpy; all copied bytes are synthetic.
		discardWrite(t, b, 0x125c54, inotia2PatchBytes("014b1860014870470000230000002200"))
		discardWrite(t, b, 0x183b64, inotia2PatchBytes("10b50446002a05d00b78037001300131013af9d1204610bd"))
		skillBooksRunEnsure(t, b)
		if discardRead(t, b, 0x230000) != 976*17 || discardRead(t, b, 0x194828) != 0x220000 {
			t.Fatal("incorrect extension allocation or pointer")
		}
		var meta [4]byte
		_ = b.ReadMemory(0x194824, meta[:])
		if binary.LittleEndian.Uint16(meta[:]) != 976 || meta[2] != 17 {
			t.Fatal("native bounds or stride changed incorrectly")
		}
		got := make([]byte, 976*17)
		_ = b.ReadMemory(0x220000, got)
		if !bytes.Equal(got[:len(original)], original) {
			t.Fatal("original definitions changed")
		}
		for i := 970; i < 976; i++ {
			if !bytes.Equal(got[i*17+2:(i+1)*17], original[839*17+2:840*17]) {
				t.Fatalf("class %d definition differs", i-970)
			}
			if binary.LittleEndian.Uint16(got[i*17:i*17+2]) != uint16(60000+i-970) {
				t.Fatal("class name ID missing")
			}
		}
		discardWord(t, b, 0x230000, 0)
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		skillBooksRunEnsure(t, b)
		if discardRead(t, b, 0x230000) != 0 {
			t.Fatal("extension allocated twice")
		}
		skillBooksPreserved(t, b)
		// Allocation failure leaves the existing table and count intact.
		discardWrite(t, b, 0x194824, []byte{0xca, 3, 17, 0})
		discardWord(t, b, 0x194828, 0x210000)
		discardWrite(t, b, 0x125c54, []byte{0, 0x20, 0x70, 0x47})
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		skillBooksRunEnsure(t, b)
		if discardRead(t, b, 0x194828) != 0x210000 {
			t.Fatal("allocation failure replaced original table")
		}
		_ = b.ReadMemory(0x194824, meta[:])
		if binary.LittleEndian.Uint16(meta[:]) != 970 {
			t.Fatal("allocation failure published new bounds")
		}
		skillBooksPreserved(t, b)
	})
}

func skillBooksRunEnsure(t *testing.T, b cpu.Backend) {
	t.Helper()
	r := b.Run(context.Background(), inotia2SkillBooksEnsure, cpu.ModeThumb, 200000)
	if r.Err != nil || r.Reason != cpu.StopBreakpoint {
		t.Fatalf("native table extension failed: %+v", r)
	}
}

func TestInotia2SkillBooksConstructorBootstrapAndTextABI(t *testing.T) {
	inotia2DiscardBackends(t, func(t *testing.T, b cpu.Backend) {
		skillBooksTestCode(t, b)
		discardWrite(t, b, 0x194824, []byte{0xd0, 3, 17, 0})
		// Synthetic continuation restores the high-register constructor frame.
		discardWrite(t, b, 0x14a0c4, inotia2PatchBytes("0cbc90469a46f0bd"))
		for _, id := range []uint32{839, 970, 972, 975} {
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, id)
			discardRun(t, b, 0x14a0bc)
			discardCheckReg(t, b, cpu.RegisterR0, id)
			skillBooksPreserved(t, b)
		}
		discardWrite(t, b, 0x101950, []byte{77, 0x20, 0x70, 0x47})
		discardReg(t, b, cpu.RegisterLR, 0x200101)
		discardRun(t, b, 0x2aeb00)
		discardCheckReg(t, b, cpu.RegisterR0, 77)
		skillBooksPreserved(t, b)
		for class := uint32(0); class < 6; class++ {
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, 60000+class)
			discardRun(t, b, 0x14fd0c)
			discardCheckReg(t, b, cpu.RegisterR0, discardRead(t, b, 0x2aec80+class*4))
			skillBooksPreserved(t, b)
		}
		discardWrite(t, b, 0x14fd14, []byte{0, 0xbd})
		for _, id := range []uint32{0, 921, 59999, 60006} {
			discardReg(t, b, cpu.RegisterLR, 0x200101)
			discardReg(t, b, cpu.RegisterR0, id)
			discardRun(t, b, 0x14fd0c)
			discardCheckReg(t, b, cpu.RegisterR0, id)
			skillBooksPreserved(t, b)
		}
	})
}
