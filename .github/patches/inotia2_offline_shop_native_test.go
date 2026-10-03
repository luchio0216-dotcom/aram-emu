//go:build ((linux || android || darwin) && arm64) || (windows && amd64)

package application

import (
 "testing"
 "github.com/mirusu400/aram-core/cpu"
 "github.com/mirusu400/aram-core/cpu/interpreter"
)

func TestInotia2OfflineShopNativeCPU(t *testing.T) {
 cpuMu.Lock()
 old, exists := cpuBackends["native"]
 cpuBackends["native"] = func() cpu.Backend { return interpreter.NewNativeJIT() }
 cpuMu.Unlock()
 t.Cleanup(func() {
  cpuMu.Lock(); defer cpuMu.Unlock()
  if exists { cpuBackends["native"] = old } else { delete(cpuBackends,"native") }
 })
 t.Run("scope and prices", TestInotia2OfflineShopPricesAndMerchantFallback)
 t.Run("entry and cleanup", TestInotia2OfflineShopEntryCleanup)
 t.Run("stock", TestInotia2OfflineShopStockInitialization)
 t.Run("installation guards", TestInotia2OfflineShopInstallationGuards)
}
