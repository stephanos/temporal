package lower

import (
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func (a *adapter) resetSettlement(e *umpirespb.ActivityResetSettlement) (*testpilotspb.ActivityResetSettlement, error) {
	ref := func(id string) *testpilotspb.InstructionReference {
		return &testpilotspb.InstructionReference{EntrypointId: a.controllerID(), InstructionId: id}
	}
	if e.GetAttempt() < 1 || e.GetFreshAttempt() != e.GetAttempt()+1 {
		return nil, errorAt(e.GetPosition(), "reset settlement %s declares attempt %d and fresh attempt %d; the fresh attempt is the held one's next", e.GetResetRequest(), e.GetAttempt(), e.GetFreshAttempt())
	}
	return &testpilotspb.ActivityResetSettlement{
		Carrier: ref(e.GetCarrier()), ActivityEntrypointId: e.GetActivity(), ReservationOrdinal: e.GetAttempt() - 1,
		PendingSlotId: e.GetPending(), Held: ref(e.GetHeld()), ResetRequest: ref(e.GetResetRequest()), FreshReservationOrdinal: e.GetFreshAttempt() - 1,
		Settlement: ref(e.GetSettlement()), Cleanup: &testpilotspb.InstructionReference{EntrypointId: a.r.GetCleanup(), InstructionId: e.GetCleanup().GetId()},
	}, nil
}

func (a *accounting) resetSettlements() error {
	declared := a.l.a.r.GetResetSettlements()
	carried := a.c.GetProgram().GetActivityResetSettlements()
	if len(declared) != len(carried) {
		return errorAt(a.l.a.r.GetPosition(), "reset settlements differ from Program")
	}
	for i, e := range declared {
		want, err := a.l.adapter.resetSettlement(e)
		if err != nil {
			return err
		}
		if !proto.Equal(want, carried[i]) {
			return errorAt(e.GetPosition(), "reset settlement %s differs from Program", e.GetResetRequest())
		}
		if !slices.ContainsFunc(a.c.GetProgram().GetCleanup().GetInstructions(), func(n *testpilotspb.InstructionNode) bool { return n.GetInstructionId() == e.GetCleanup().GetId() }) {
			return errorAt(e.GetPosition(), "reset settlement %s lacks cleanup", e.GetResetRequest())
		}
		a.own("reset_settlements", e.GetResetRequest(), e.GetPosition(), "program.activity_reset_settlements["+e.GetResetRequest()+"]",
			"program.cleanup.instructions["+e.GetCleanup().GetId()+"]", "program.slots["+e.GetPending()+"]")
	}
	return nil
}

// resetOccurrences holds a reset settlement to the path it lowers: the held group publishes before
// the controller's one reset, the selected timer ends that group exactly once after the reset, and
// the next delivery is the fresh group the declaration names.
func (l *lowering) resetOccurrences() (problems []error) {
	for _, e := range l.a.r.GetResetSettlements() {
		var script *umpirespb.Script
		for _, s := range l.a.r.GetScripts() {
			if s.GetId() == e.GetActivity() && s.GetActivity() != nil {
				script = s
			}
		}
		if script == nil {
			problems = append(problems, errorAt(e.GetPosition(), "reset settlement %s names no activity script %s", e.GetResetRequest(), e.GetActivity()))
			continue
		}
		var resetKeys []string
		for _, s := range l.a.r.GetScripts() {
			for _, item := range s.GetItems() {
				for _, p := range item.GetPerforms() {
					if p.GetCommand().GetId() == e.GetResetRequest() {
						resetKeys = append(resetKeys, l.adapter.classKey(p.GetStep()))
					}
				}
			}
		}
		timer := l.adapter.classKey(e.GetTimer())
		starts, _ := l.attemptClasses(script)
		resetAt, timerAt, delivered, resetGroup, timerGroup := -1, -1, int64(0), int64(0), int64(0)
		resets, timers := 0, 0
		for at, key := range l.keys {
			if starts[key] {
				delivered++
			}
			if slices.Contains(resetKeys, key) {
				resets++
				resetAt, resetGroup = at, delivered
			}
			if key == timer {
				timers++
				timerAt, timerGroup = at, delivered
			}
		}
		switch {
		case resets != 1 || timers != 1:
			problems = append(problems, errorAt(e.GetPosition(), "reset settlement %s requires exactly one reset and one selected timer; got %d and %d", e.GetResetRequest(), resets, timers))
		case resetAt > timerAt:
			problems = append(problems, errorAt(e.GetPosition(), "reset settlement %s: the reset must precede the timer that applies it", e.GetResetRequest()))
		case timerGroup != e.GetAttempt() || resetGroup != e.GetAttempt():
			problems = append(problems, errorAt(e.GetPosition(), "reset settlement %s declares attempt %d; the reset holds attempt %d and the selected timer ends attempt %d", e.GetResetRequest(), e.GetAttempt(), resetGroup, timerGroup))
		case delivered < e.GetFreshAttempt():
			problems = append(problems, errorAt(e.GetPosition(), "reset settlement %s declares fresh attempt %d; the path delivers %d", e.GetResetRequest(), e.GetFreshAttempt(), delivered))
		}
	}
	return problems
}
