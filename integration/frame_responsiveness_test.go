package integration

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	aramcore "github.com/mirusu400/aram-core/core"
	"github.com/mirusu400/aram-frontend/frontend"
)

// Metadata and input deliberately share the guest frame's lock, as they do in
// the application machine. Host polling must remain responsive anyway.
type lockedFrameMachine struct {
	aramcore.Machine
	mu      sync.Mutex
	state   aramcore.State
	entered chan struct{}
	release chan struct{}
	inputs  []aramcore.InputEvent
	fault   bool
}

func (m *lockedFrameMachine) State() aramcore.State { m.mu.Lock(); defer m.mu.Unlock(); return m.state }
func (m *lockedFrameMachine) FrameQuantum() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return (time.Second + 30) / 60
}
func (m *lockedFrameMachine) Vibration() (uint8, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return 37, time.Minute
}
func (m *lockedFrameMachine) QueueInput(e aramcore.InputEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs = append(m.inputs, e)
	return nil
}
func (m *lockedFrameMachine) StepFrame(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.entered)
	<-m.release
	if m.fault {
		m.state = aramcore.StateFaulted
		return errors.New("synthetic guest fault")
	}
	m.state = aramcore.StatePaused
	return nil
}

func TestFrameStateAndInputDoNotBlockHostDuringGuestWork(t *testing.T) {
	for _, fault := range []bool{false, true} {
		m := &lockedFrameMachine{state: aramcore.StatePaused, entered: make(chan struct{}), release: make(chan struct{}), fault: fault}
		b := &Backend{machine: m, runRequested: true}
		finished := make(chan error, 1)
		go func() { finished <- b.RunFrame(context.Background()) }()
		<-m.entered
		var once sync.Once
		unblock := func() { once.Do(func() { close(m.release) }) }
		defer unblock()
		want := []aramcore.InputEvent{{Control: "right", Pressed: false}, {Control: "num4", Pressed: true}, {Control: "num4", Pressed: false}, {Control: "left", Pressed: true}}
		polled := make(chan error, 1)
		go func() {
			if s := b.State(); s != frontend.StateRunning {
				polled <- errors.New("running state was lost")
				return
			}
			if !b.Capability(frontend.CommandPauseResume).Supported {
				polled <- errors.New("pause unavailable during frame")
				return
			}
			if b.FrameQuantum() != (time.Second+30)/60 {
				polled <- errors.New("frame quantum changed during guest work")
				return
			}
			if pulse := b.Haptics(); pulse.Level != 37 || pulse.Duration <= 0 || pulse.Duration > time.Minute {
				polled <- errors.New("haptic snapshot changed during guest work")
				return
			}
			for _, e := range want {
				if err := b.QueueInput(frontend.InputEvent{Control: e.Control, Pressed: e.Pressed}); err != nil {
					polled <- err
					return
				}
			}
			polled <- nil
		}()
		select {
		case err := <-polled:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			unblock()
			t.Fatal("host waited for a locked guest frame")
		}
		unblock()
		err := <-finished
		if fault {
			if err == nil || b.State() != frontend.StateFaulted {
				t.Fatal("completed fault was hidden")
			}
		} else {
			if err != nil || b.State() != frontend.StateRunning {
				t.Fatalf("completed state: %v", err)
			}
		}
		if !reflect.DeepEqual(m.inputs, want) {
			t.Fatalf("input edges reordered or lost: got %+v want %+v", m.inputs, want)
		}
		if b.frameRunning || len(b.pendingFrameInput) != 0 {
			t.Fatal("frame completion left host work pending")
		}
	}
}

func TestFrameHapticsExpireWithoutWaitingForGuest(t *testing.T) {
	pulse := frontend.HapticsState{Level: 70, Duration: 100 * time.Millisecond}
	for _, elapsed := range []time.Duration{100 * time.Millisecond, time.Second} {
		if got := remainingHaptics(pulse, elapsed); got != (frontend.HapticsState{}) {
			t.Fatalf("expired pulse remained active: %+v", got)
		}
	}
	if got := remainingHaptics(pulse, 25*time.Millisecond); got.Level != 70 || got.Duration != 75*time.Millisecond {
		t.Fatalf("snapshot duration did not count down: %+v", got)
	}
	if got := remainingHaptics(pulse, -time.Second); got != pulse {
		t.Fatalf("negative elapsed time extended the pulse: %+v", got)
	}
}

func TestFrameInputBoundRejectsWithoutDroppingAcceptedEdges(t *testing.T) {
	b := &Backend{machine: &lockedFrameMachine{}, frameRunning: true}
	for i := 0; i < 1024; i++ {
		if err := b.QueueInput(frontend.InputEvent{Control: "right", Pressed: i%2 == 0}); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.QueueInput(frontend.InputEvent{Control: "right"}); err == nil {
		t.Fatal("unbounded host input queue")
	}
	if len(b.pendingFrameInput) != 1024 {
		t.Fatal("accepted input was lost")
	}
	if err := b.QueueInput(frontend.InputEvent{}); err == nil {
		t.Fatal("malformed input accepted")
	}
}
