//go:build ((linux || android || darwin) && arm64) || (windows && amd64)

package application

import (
	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
	"testing"
)

// Register the concrete native implementation on the headless Linux ARM64 CI
// runner too; application registration normally belongs to Android hosts.
func TestInotia2AutoLootDiscardOnNativeCPU(t *testing.T) {
	native := func() cpu.Backend { return interpreter.NewNativeJIT() }
	RegisterCPUBackend("discard-native-test", native)
	// The shared fixtures explicitly include the native factory once registered.
	cpuMu.Lock()
	old, exists := cpuBackends["native"]
	cpuBackends["native"] = native
	cpuMu.Unlock()
	t.Cleanup(func() {
		cpuMu.Lock()
		if exists {
			cpuBackends["native"] = old
		} else {
			delete(cpuBackends, "native")
		}
		delete(cpuBackends, "discard-native-test")
		cpuMu.Unlock()
	})
	t.Run("mark", TestInotia2AutoLootDiscardMarksOnlyInventoryDrop)
	t.Run("leave and return", TestInotia2AutoLootDiscardLeaveThenReturn)
	t.Run("swap and field reset", TestInotia2AutoLootDiscardFlagsFollowNativeSwapAndFieldReset)
}
