//go:build ((linux || android || darwin) && arm64) || (windows && amd64)

package application

import (
	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
	"testing"
)

func TestInotia2InventorySwapNativeCPU(t *testing.T) {
	cpuMu.Lock()
	old, exists := cpuBackends["native"]
	cpuBackends["native"] = func() cpu.Backend { return interpreter.NewNativeJIT() }
	cpuMu.Unlock()
	t.Cleanup(func() {
		cpuMu.Lock()
		defer cpuMu.Unlock()
		if exists {
			cpuBackends["native"] = old
		} else {
			delete(cpuBackends, "native")
		}
	})
	t.Run("chains and cancel", TestInotia2InventorySwapChainPreservesItemsAndCancel)
	t.Run("bag boundaries", TestInotia2InventorySwapBagBoundaries)
	t.Run("guards and native delegation", TestInotia2InventorySwapGuardsAndNativeDelegation)
	t.Run("highlight and completion", TestInotia2InventorySwapHighlightAndCompletion)
}
