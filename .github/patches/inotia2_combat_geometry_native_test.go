//go:build ((linux || android || darwin) && arm64) || (windows && amd64)

package application

import (
	"github.com/mirusu400/aram-core/cpu"
	"github.com/mirusu400/aram-core/cpu/interpreter"
	"testing"
)

func TestInotia2CombatGeometryNativeCPU(t *testing.T) {
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
	t.Run("equivalent geometry", TestInotia2CombatGeometryResultAndABI)
	t.Run("installation guards", TestInotia2CombatGeometryInstallation)
}
