package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
)

func TestInotia2AutoLootPatchGuardsAndAtomicity(t *testing.T) {
	original := []byte("0123456789abcdef")
	imageSHA := fmt.Sprintf("%x", sha256.Sum256(original))
	patch := func(offset, size int, replacement string) inotia2ImagePatch {
		return inotia2ImagePatch{offset: offset, expectedSHA: fmt.Sprintf("%x", sha256.Sum256(original[offset:offset+size])), replacement: []byte(replacement)}
	}
	patches := []inotia2ImagePatch{patch(2, 2, "AB"), patch(8, 2, "CD")}
	got, err := applyInotia2ImagePatches(original, imageSHA, patches)
	if err != nil || string(got) != "01AB4567CDabcdef" || string(original) != "0123456789abcdef" {
		t.Fatalf("guarded patch: %q, %v; source %q", got, err, original)
	}
	unknown := bytes.Clone(original)
	unknown[0] = 'x'
	got, err = applyInotia2ImagePatches(unknown, imageSHA, patches)
	if err != nil || !bytes.Equal(got, unknown) {
		t.Fatalf("unknown image was changed: %q, %v", got, err)
	}
	for _, tc := range []struct {
		name    string
		patches []inotia2ImagePatch
	}{
		{"late checksum mismatch", []inotia2ImagePatch{patches[0], {offset: 8, expectedSHA: "wrong", replacement: []byte("CD")}}},
		{"out of bounds", []inotia2ImagePatch{patches[0], {offset: 16, replacement: []byte("CD")}}},
		{"negative offset", []inotia2ImagePatch{{offset: -1, replacement: []byte("CD")}}},
		{"overlap", []inotia2ImagePatch{patches[0], patch(3, 2, "CD")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := applyInotia2ImagePatches(original, imageSHA, tc.patches)
			if err == nil || got != nil || string(original) != "0123456789abcdef" {
				t.Fatalf("invalid patch was not rejected atomically: %q, %v; %q", got, err, original)
			}
		})
	}
}

// Run the new Thumb helper on ARAM's portable CPU with entirely synthetic
// actor and pickup stubs. No game, save data, or extracted assets are fixtures.
func TestInotia2AutoLootSelectedHeroAndRegisterPreservation(t *testing.T) {
	for _, tc := range []struct {
		name            string
		actor, selected uint32
		tag, pickups    uint32
	}{
		{"selected hero", 0x200000, 0x200000, 4, 1},
		{"empty field", 0x200000, 0x200000, 4, 0},
		{"companion or NPC", 0x210000, 0x200000, 4, 0},
		{"inactive actor", 0x200000, 0x200000, 0, 0},
		{"no selected hero", 0x210000, 0, 4, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := inotia2AutoLootTestCPU(t)
			if tc.name != "empty field" {
				inotia2AutoLootTestWrite(t, b, 0x194c6c, []byte{1, 0, 0, 0})
			}
			inotia2AutoLootTestWrite(t, b, 0x12c580, []byte{byte(tc.tag), 0x20, 0x70, 0x47}) // movs r0, tag; bx lr
			// Increment one synthetic pickup counter and return.
			inotia2AutoLootTestWrite(t, b, 0x12ccf0, inotia2PatchBytes("0249086801300860704700bf7caa2a00"))
			selected := make([]byte, 4)
			binary.LittleEndian.PutUint32(selected, tc.selected)
			inotia2AutoLootTestWrite(t, b, 0x2aaa78, selected)
			inotia2AutoLootTestRegister(t, b, cpu.RegisterR0, tc.actor)
			// Start at the actual replaced BL, then trap after its return.
			inotia2AutoLootTestWrite(t, b, 0x137c2a, append(bytes.Clone(inotia2AutoLootPatches[0].replacement), 0, 0xbe))
			inotia2AutoLootTestRun(t, b, 0x137c2a)
			counter := make([]byte, 4)
			if err := b.ReadMemory(0x2aaa7c, counter); err != nil {
				t.Fatal(err)
			}
			if got := binary.LittleEndian.Uint32(counter); got != tc.pickups {
				t.Fatalf("pickups = %d, want %d", got, tc.pickups)
			}
			if got := inotia2AutoLootTestReadRegister(t, b, cpu.RegisterR0); got != tc.tag {
				t.Fatalf("actor tag = %d, want %d", got, tc.tag)
			}
			inotia2AutoLootTestPreserved(t, b)
			flag := make([]byte, 4)
			if err := b.ReadMemory(0x2ad348, flag); err != nil {
				t.Fatal(err)
			}
			if binary.LittleEndian.Uint32(flag) != 0 {
				t.Fatal("automatic pickup flag leaked beyond synchronous call")
			}
		})
	}
}

func TestInotia2AutoLootFullBagQuietOnlyDuringAutomaticCall(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		t.Run(fmt.Sprintf("automatic=%t", automatic), func(t *testing.T) {
			b := inotia2AutoLootTestCPU(t)
			if automatic {
				inotia2AutoLootTestWrite(t, b, 0x2ad348, []byte{1, 0, 0, 0})
			}
			// Synthetic success return at the native pickup epilogue target.
			inotia2AutoLootTestWrite(t, b, 0x12ce1a, []byte{0, 0x20, 0, 0xbe})
			inotia2AutoLootTestRegister(t, b, cpu.RegisterLR, 0x200101)
			inotia2AutoLootTestRun(t, b, 0x12afd4)
			want := uint32(1)
			if automatic {
				want = 0
			}
			if got := inotia2AutoLootTestReadRegister(t, b, cpu.RegisterR0); got != want {
				t.Fatalf("full-bag return = %d, want %d", got, want)
			}
			if !automatic {
				if got := inotia2AutoLootTestReadRegister(t, b, cpu.RegisterR1); got != 0x2ff024 {
					t.Fatalf("manual message stack argument = %#x", got)
				}
			}
			inotia2AutoLootTestPreserved(t, b)
		})
	}
}

func TestInotia2AutoLootAuthorizedClientFixture(t *testing.T) {
	path := os.Getenv("ARAM_INOTIA2_CLIENT_FIXTURE")
	if path == "" {
		t.Skip("private authorized client not supplied")
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(original)) != inotia2AutoLootOriginalSHA {
		t.Fatal("fixture is not the validated Rare12 client")
	}
	before := bytes.Clone(original)
	patched, err := inotia2AutoLootClient(original)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(patched)) != inotia2AutoLootPatchedSHA {
		t.Fatal("patched client digest differs")
	}
	if !bytes.Equal(original, before) {
		t.Fatal("package client mutated in place")
	}
	legacy, err := applyInotia2ImagePatches(original, inotia2AutoLootOriginalSHA, inotia2AutoLootPatches)
	if err != nil {
		t.Fatal(err)
	}
	upgraded, err := inotia2AutoLootClient(legacy)
	if err != nil || !bytes.Equal(upgraded, patched) {
		t.Fatal("previous automatic-pickup image did not upgrade losslessly")
	}
	size, err := inotia2AutoLootMappedSize(patched, 608192+1149832)
	if err != nil || size != 1765376 {
		t.Fatalf("scratch mapping = %d, %v", size, err)
	}
	if _, err := inotia2AutoLootMappedSize(patched, 1149000); err == nil {
		t.Fatal("unexpected image size accepted")
	}
	again, err := inotia2AutoLootClient(patched)
	if err != nil || !bytes.Equal(again, patched) {
		t.Fatal("automatic pickup patch is not idempotent")
	}
	for i := range original {
		covered := false
		for _, p := range append(append([]inotia2ImagePatch(nil), inotia2AutoLootPatches...), inotia2DiscardPatches...) {
			if i >= p.offset && i < p.offset+len(p.replacement) {
				covered = true
			}
		}
		if !covered && original[i] != patched[i] {
			t.Fatalf("unrelated byte changed at %#x", i)
		}
	}
}

func inotia2AutoLootTestCPU(t *testing.T) *interpreter.Backend {
	t.Helper()
	b := interpreter.New()
	t.Cleanup(func() { _ = b.Close() })
	if err := b.Map(0x100000, 0x200000, cpu.PermissionRead|cpu.PermissionWrite|cpu.PermissionExecute); err != nil {
		t.Fatal(err)
	}
	inotia2AutoLootTestWrite(t, b, 0x12af90, inotia2AutoLootPatches[2].replacement)
	inotia2AutoLootTestWrite(t, b, 0x200100, []byte{0, 0xbe})
	for r, v := range map[uint32]uint32{cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555, cpu.RegisterR6: 0x66666666, cpu.RegisterSP: 0x2ff000} {
		inotia2AutoLootTestRegister(t, b, r, v)
	}
	return b
}

func inotia2AutoLootTestWrite(t *testing.T, b *interpreter.Backend, address uint32, data []byte) {
	t.Helper()
	if err := b.WriteMemory(address, data); err != nil {
		t.Fatal(err)
	}
}

func inotia2AutoLootTestRegister(t *testing.T, b *interpreter.Backend, r, value uint32) {
	t.Helper()
	if err := b.WriteRegister(r, value); err != nil {
		t.Fatal(err)
	}
}

func inotia2AutoLootTestReadRegister(t *testing.T, b *interpreter.Backend, r uint32) uint32 {
	t.Helper()
	value, err := b.ReadRegister(r)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func inotia2AutoLootTestRun(t *testing.T, b *interpreter.Backend, address uint32) {
	t.Helper()
	result := b.Run(context.Background(), address, cpu.ModeThumb, 1000)
	if result.Err != nil || result.Reason != cpu.StopBreakpoint {
		t.Fatalf("Thumb helper did not return: %+v", result)
	}
}

func inotia2AutoLootTestPreserved(t *testing.T, b *interpreter.Backend) {
	t.Helper()
	for r, want := range map[uint32]uint32{cpu.RegisterR4: 0x44444444, cpu.RegisterR5: 0x55555555, cpu.RegisterR6: 0x66666666, cpu.RegisterSP: 0x2ff000} {
		if got := inotia2AutoLootTestReadRegister(t, b, r); got != want {
			t.Fatalf("register %d = %#x, want %#x", r, got, want)
		}
	}
}
