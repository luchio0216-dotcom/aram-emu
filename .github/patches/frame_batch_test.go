package frontend

import (
	"context"
	"errors"
	"testing"
	"time"
)

type timedBatchBackend struct {
	step func() error
}

func (b timedBatchBackend) RunFrame(context.Context) error { return b.step() }

func TestFrameBatchPublishesSlowCombatBeforeCatchUp(t *testing.T) {
	for _, test := range []struct {
		name string
		cost time.Duration
		want int
	}{
		{"fast catch-up", time.Millisecond, 8},
		{"partial catch-up", 2 * time.Millisecond, 4},
		{"one slow combat frame", 150 * time.Millisecond, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			clock := time.Unix(0, 0)
			calls := 0
			backend := timedBatchBackend{step: func() error {
				calls++
				clock = clock.Add(test.cost)
				return nil
			}}
			request := frameRunRequest{backend: backend, owed: 8, quantum: 16 * time.Millisecond, generation: 7}
			result := executeFrameBatch(request, func() time.Time { return clock })
			if result.err != nil || calls != test.want || result.completedQuanta != test.want {
				t.Fatalf("completed calls=%d result=%+v", calls, result)
			}
			if result.generation != 7 || result.guestAdvanced+result.deferredGuest != 8*request.quantum {
				t.Fatalf("guest time was lost or extra time advanced: %+v", result)
			}
		})
	}
}

func TestFrameBatchStopsOnBackendFault(t *testing.T) {
	cause := errors.New("synthetic guest fault")
	calls := 0
	backend := timedBatchBackend{step: func() error {
		calls++
		if calls == 2 {
			return cause
		}
		return nil
	}}
	result := executeFrameBatch(frameRunRequest{backend: backend, owed: 8, quantum: time.Millisecond}, func() time.Time { return time.Unix(0, 0) })
	if !errors.Is(result.err, cause) || calls != 2 || result.completedQuanta != 1 || result.deferredGuest != 0 {
		t.Fatalf("fault did not stop the batch: %+v calls=%d", result, calls)
	}
}

func TestDeferredFrameTimeIsBoundedAndIgnoresOldSession(t *testing.T) {
	s := &Shell{frameGeneration: 3, frameRunPending: true, frameRunResults: make(chan frameRunResult, 2)}
	quantum := s.frameQuantum()
	s.frameAccumulator = quantum / 2
	s.frameRunResults <- frameRunResult{generation: 2, deferredGuest: 7 * quantum}
	s.consumeResults()
	if s.frameAccumulator != quantum/2 || !s.frameRunPending {
		t.Fatal("old session changed the current pacing budget")
	}
	s.frameRunResults <- frameRunResult{generation: 3, deferredGuest: 7 * quantum}
	s.consumeResults()
	if s.frameAccumulator != 7*quantum+quantum/2 || s.frameRunPending {
		t.Fatalf("unexecuted quanta were not restored: %v", s.frameAccumulator)
	}
	s.restoreDeferredGuest(8 * quantum)
	if s.frameAccumulator != framePacingQuantaPerTick*quantum {
		t.Fatal("slow combat grew an unbounded catch-up backlog")
	}
}
