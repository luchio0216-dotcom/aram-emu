package bootstrap

import (
	"os"
	"time"
)

// activateRuntime only publishes an extracted staging directory. It neither
// removes a target nor changes the selected-runtime marker.
func activateRuntime(staging, target string) error {
	return retryActivation(staging, target, os.Rename, time.Sleep, activationRetryable)
}

// retryActivation allows six attempts and five requested sleeps totalling
// 310ms (10/20/40/80/160ms), not a hard elapsed-time deadline. Dependencies
// are local arguments, never mutable global hooks. Exhaustion preserves the
// final rename error. This mitigates transient denial, not its unknown cause.
func retryActivation(staging, target string, rename func(string, string) error, sleep func(time.Duration), retryable func(error) bool) error {
	for attempt := 0; ; attempt++ {
		err := rename(staging, target)
		if err == nil || attempt == 5 || !retryable(err) {
			return err
		}
		sleep((10 * time.Millisecond) << attempt)
	}
}
