package clock

import (
	"context"
	"testing"
)

func TestBrokerClockRejectsInactiveParticipantWithoutAdvancing(t *testing.T) {
	clock := NewProcessClock(false)
	initial := clock.Current()
	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{"request", func() error { return clock.Accepted(nil, 1, 0) }},
		{"activation", func() error { _, err := clock.ActivateAt(nil, initial); return err }},
		{"quiescence", func() error {
			_, err := clock.Quiesce(context.Background(), nil, BrokerTimeRequest{Generation: 1, Current: initial, Deadline: initial + 1})
			return err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); err == nil || err.Error() != "simulation time participant is inactive" {
				t.Fatalf("operation error = %v", err)
			}
			if current := clock.Current(); current != initial {
				t.Fatalf("time advanced to %d, want %d", current, initial)
			}
		})
	}
}
