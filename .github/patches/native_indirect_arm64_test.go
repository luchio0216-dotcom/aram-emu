//go:build (android || linux) && arm64

package interpreter

import (
	"context"
	"testing"
	"unsafe"

	"github.com/mirusu400/aram-core/cpu"
)

func indirectFixture(t *testing.T) (*Backend, *nativeBlock, uint32) {
	t.Helper()
	b := nativeBackend(t)
	mustMap(t, b, 0x1000, 0x6000, cpu.PermissionRead|cpu.PermissionWrite|cpu.PermissionExecute)
	check(t, b.WriteMemory(0x1000, []byte{0x20, 0x47}))             // bx r4
	check(t, b.WriteMemory(0x1100, []byte{0x03, 0x20, 0x00, 0xbe})) // movs r0,#3; bkpt
	check(t, b.WriteRegister(cpu.RegisterR4, 0x1101))
	check(t, b.WriteRegister(cpu.RegisterCPSR, cpu.StatusThumb))
	if b.nativeBlockAt(0x1100) == nil {
		t.Fatal("target was not translated")
	}
	source := b.nativeBlockAt(0x1000)
	if source == nil {
		t.Fatal("source was not translated")
	}
	return b, source, 0x1100
}

func TestNativeIndirectThumbGateExecutesWithoutDispatcher(t *testing.T) {
	b, source, target := indirectFixture(t)
	if unsafe.Sizeof(nativeIndirectEntry{}) != 16 || unsafe.Offsetof(nativeIndirectEntry{}.gate) != 8 {
		t.Fatal("indirect entry layout does not match emitted AArch64 loads")
	}
	b.nativeRemain = 8
	status := callNativeBlock(source.entry, &b.regs[0], &b.nativeRemain)
	if status != nativeStatusBKPT || b.regs[cpu.RegisterR0] != 3 || b.nativeRemain != 5 || b.regs[cpu.RegisterPC] != target+4 {
		t.Fatalf("indirect gate result status=%d r0=%d remain=%d pc=%#x", status, b.regs[0], b.nativeRemain, b.regs[15])
	}
	// The cached target is its budget gate, never its unchecked body.
	b.nativeRemain = 1
	b.regs[cpu.RegisterR0] = 0
	status = callNativeBlock(source.entry, &b.regs[0], &b.nativeRemain)
	if status != nativeStatusBudget || b.nativeRemain != 0 || b.regs[0] != 0 || b.regs[15] != target {
		t.Fatalf("indirect target bypassed budget: status=%d remain=%d pc=%#x", status, b.nativeRemain, b.regs[15])
	}
}

func TestNativeIndirectThumbCollisionAndInvalidationFallBack(t *testing.T) {
	b, source, target := indirectFixture(t)
	collision := target + nativeIndirectCacheSize*2
	check(t, b.WriteMemory(collision, []byte{0x09, 0x20, 0x00, 0xbe}))
	if b.nativeBlockAt(collision) == nil {
		t.Fatal("colliding target was not translated")
	}
	b.nativeRemain = 8
	status := callNativeBlock(source.entry, &b.regs[0], &b.nativeRemain)
	if status != nativeStatusMode || b.regs[15] != target || b.nativeRemain != 7 {
		t.Fatalf("cache collision reached another PC: status=%d pc=%#x", status, b.regs[15])
	}
	b.publishNativeLink(cpu.ModeThumb, target, b.nativeBlocks[target].gate)
	check(t, b.WriteMemory(target, []byte{0x07, 0x20})) // self-modifying target
	b.nativeRemain = 8
	status = callNativeBlock(source.entry, &b.regs[0], &b.nativeRemain)
	if status != nativeStatusMode || b.regs[15] != target {
		t.Fatal("invalidated indirect target remained executable")
	}
	result := b.Run(context.Background(), 0x1000, cpu.ModeThumb, 8)
	if result.Err != nil || result.Reason != cpu.StopBreakpoint || result.Instructions != 3 || b.regs[0] != 7 {
		t.Fatalf("modified target was not retranslated: %+v r0=%d", result, b.regs[0])
	}
	b.nativeInvalidate()
	for i := range b.nativeIndirect {
		entry := &b.nativeIndirect[i]
		if entry.tag.Load() != 0 || entry.gate.Load() != 0 {
			t.Fatal("full reset retained an indirect gate")
		}
	}
}

func TestNativeIndirectThumbARMTargetUsesArchitecturalDispatcher(t *testing.T) {
	b, source, _ := indirectFixture(t)
	check(t, b.WriteRegister(cpu.RegisterR4, 0x1200))
	b.nativeRemain = 8
	status := callNativeBlock(source.entry, &b.regs[0], &b.nativeRemain)
	if status != nativeStatusMode || b.regs[15] != 0x1200 || b.regs[cpu.RegisterCPSR]&cpu.StatusThumb != 0 {
		t.Fatal("ARM mode change bypassed the architectural dispatcher")
	}
}

func BenchmarkNativeIndirectThumbCalls(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		name := "dispatcher"
		if enabled {
			name = "cached-gates"
		}
		b.Run(name, func(b *testing.B) {
			backend := NewNativeJIT()
			defer backend.Close()
			if backend.nativeBlocks == nil {
				b.Skip("native JIT unavailable")
			}
			if !enabled {
				backend.nativeIndirect = nil
			}
			if err := backend.Map(0x1000, 0x2000, cpu.PermissionRead|cpu.PermissionWrite|cpu.PermissionExecute); err != nil {
				b.Fatal(err)
			}
			// Synthetic repeated tiny function calls, the dispatch pattern that
			// dominates byte-reader and renderer helper calls in combat.
			if err := backend.WriteMemory(0x1000, []byte{0xa0, 0x47, 0x01, 0x3a, 0xfc, 0xd1, 0x00, 0xbe}); err != nil {
				b.Fatal(err)
			}
			if err := backend.WriteMemory(0x1100, []byte{0x01, 0x30, 0x70, 0x47}); err != nil {
				b.Fatal(err)
			}
			for _, pc := range []uint32{0x1000, 0x1002, 0x1100} {
				backend.nativeBlockAt(pc)
			}
			const calls = 10_000
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				backend.regs[cpu.RegisterR0] = 0
				backend.regs[cpu.RegisterR2] = calls
				backend.regs[cpu.RegisterR4] = 0x1101
				result := backend.Run(context.Background(), 0x1000, cpu.ModeThumb, calls*5+1)
				if result.Err != nil || result.Reason != cpu.StopBreakpoint || result.Instructions != calls*5+1 || backend.regs[0] != calls {
					b.Fatalf("call loop: %+v r0=%d", result, backend.regs[0])
				}
			}
		})
	}
}
