package integration

import (
	"time"

	aramcore "github.com/mirusu400/aram-core/core"
	"github.com/mirusu400/aram-frontend/frontend"
)

// coreHapticsSource is the optional aram-core contract for reading the guest's
// current vibration request: motor strength (0-100) and the time remaining
// before it stops.
type coreHapticsSource interface {
	Vibration() (uint8, time.Duration)
}

// Haptics reports the guest's vibration so the frontend can actuate a real
// gamepad rumble motor or phone vibrator. A core that does not expose vibration
// yields the zero state, so the frontend drives no haptics.
func (backend *Backend) Haptics() frontend.HapticsState {
	backend.mu.RLock()
	defer backend.mu.RUnlock()
	if backend.frameRunning {
		// A stalled guest must not prolong a real motor pulse. Only the host
		// snapshot expires with wall time; the core's virtual clock is intact.
		return remainingHaptics(backend.frameHaptics, time.Since(backend.frameStarted))
	}
	return machineHaptics(backend.machine)
}

func machineHaptics(machine aramcore.Machine) frontend.HapticsState {
	if machine == nil {
		return frontend.HapticsState{}
	}
	source, ok := unwrapMachine(machine).(coreHapticsSource)
	if !ok {
		return frontend.HapticsState{}
	}
	level, remaining := source.Vibration()
	return frontend.HapticsState{Level: level, Duration: remaining}
}

func remainingHaptics(state frontend.HapticsState, elapsed time.Duration) frontend.HapticsState {
	if elapsed < 0 {
		elapsed = 0
	}
	if state.Level == 0 || state.Duration <= elapsed {
		return frontend.HapticsState{}
	}
	state.Duration -= elapsed
	return state
}
