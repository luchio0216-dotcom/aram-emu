//go:build ((linux || android || darwin) && arm64) || (windows && amd64)

package application

import (
	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
	"testing"
)

func TestInotia2SkillBooksNativeCPU(t *testing.T) {
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
	t.Run("class selection and capacity", TestInotia2SkillBooksClassSelectionAndCapacity)
	t.Run("inventory unseal menu", TestInotia2SkillBooksInventoryUnsealMenu)
	t.Run("names and fallback", TestInotia2SkillBooksNamesAndOriginalFallback)
	t.Run("table extension", TestInotia2SkillBooksExtendNativeDefinitions)
	t.Run("constructor and bootstrap", TestInotia2SkillBooksConstructorBootstrapAndTextABI)
}
