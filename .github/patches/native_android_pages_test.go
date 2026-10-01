//go:build (android || linux) && arm64

package interpreter

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/mirusu400/aram-core/cpu"
)

// A 16 KiB protection granularity is valid on both 4 KiB and 16 KiB hosts.
// Emit and execute enough separate blocks to cross both boundaries repeatedly.
func TestNativeArenaExecutesAcross16KProtectionPages(t *testing.T) {
	previous := arenaPageSize
	arenaPageSize = 16384
	t.Cleanup(func() { arenaPageSize = previous })
	b := NewNativeJIT()
	defer b.Close()
	if b.nativeArena == nil { t.Skip("native executable arena unavailable") }
	if err := b.Map(0x1000, 0x10000, cpu.PermissionRead|cpu.PermissionWrite|cpu.PermissionExecute); err != nil { t.Fatal(err) }
	for i := 0; i < 1000; i++ {
		address := uint32(0x1000+i*16)
		data := make([]byte,4)
		binary.LittleEndian.PutUint16(data, 0x2000|uint16(i&255))
		binary.LittleEndian.PutUint16(data[2:], 0xbe00)
		if err := b.WriteMemory(address,data); err != nil { t.Fatal(err) }
		result := b.Run(context.Background(), address, cpu.ModeThumb, 16)
		if result.Err != nil || result.Reason != cpu.StopBreakpoint { t.Fatalf("block %d: %+v",i,result) }
		value, err := b.ReadRegister(cpu.RegisterR0)
		if err != nil || value != uint32(i&255) { t.Fatalf("block %d result=%d, %v",i,value,err) }
	}
	if b.nativeArena.off < 32768 { t.Fatalf("test did not cross protection pages: %d bytes",b.nativeArena.off) }
}
