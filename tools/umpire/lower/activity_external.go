package lower

import (
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/cel"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/proto"
)

func externalSucceeded(ref *testpilotspb.InstructionReference) *testpilotspb.Expression {
	status := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: ref, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}})
	return cel.All(cel.Present(status), cp.Equal(status, cp.Literal(cp.Enum("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"))))
}

func (a *adapter) controllerID() string {
	for _, s := range a.r.GetScripts() {
		if s.GetController() != nil {
			return s.GetId()
		}
	}
	return ""
}

func (a *adapter) externalSettlement(e *umpirespb.ActivityExternalSettlement) (*testpilotspb.ActivityExternalSettlement, error) {
	ref := func(id string) *testpilotspb.InstructionReference {
		return &testpilotspb.InstructionReference{EntrypointId: a.controllerID(), InstructionId: id}
	}
	if e.GetActivity() != "" && e.GetAttempt() < 1 {
		return nil, errorAt(e.GetPosition(), "external settlement %s has no positive attempt", e.GetAnswer())
	}
	out := &testpilotspb.ActivityExternalSettlement{Carrier: ref(e.GetCarrier()), Answer: ref(e.GetAnswer()), Settlement: ref(e.GetSettlement()), Cleanup: &testpilotspb.InstructionReference{EntrypointId: a.r.GetCleanup(), InstructionId: e.GetCleanup().GetId()}}
	if e.GetActivity() != "" {
		out.ActivityEntrypointId, out.ReservationOrdinal, out.PendingSlotId, out.Held = e.GetActivity(), e.GetAttempt()-1, e.GetPending(), ref(e.GetHeld())
	}
	if e.GetRequestCancel() != "" {
		out.RequestCancel = ref(e.GetRequestCancel())
	}
	return out, nil
}

func (a *accounting) externalSettlements() error {
	declared := a.l.a.r.GetExternalSettlements()
	carried := a.c.GetProgram().GetActivityExternalSettlements()
	if len(declared) != len(carried) {
		return errorAt(a.l.a.r.GetPosition(), "external settlements differ from Program")
	}
	for i, e := range declared {
		want, err := a.l.adapter.externalSettlement(e)
		if err != nil {
			return err
		}
		if !proto.Equal(want, carried[i]) {
			return errorAt(e.GetPosition(), "external settlement %s differs from Program", e.GetAnswer())
		}
		if !slices.ContainsFunc(a.c.GetProgram().GetCleanup().GetInstructions(), func(n *testpilotspb.InstructionNode) bool { return n.GetInstructionId() == e.GetCleanup().GetId() }) {
			return errorAt(e.GetPosition(), "external settlement %s lacks cleanup", e.GetAnswer())
		}
		parts := []string{"program.activity_external_settlements[" + e.GetAnswer() + "]", "program.cleanup.instructions[" + e.GetCleanup().GetId() + "]"}
		if e.GetPending() != "" {
			parts = append(parts, "program.slots["+e.GetPending()+"]")
		}
		a.own("external_settlements", e.GetAnswer(), e.GetPosition(), parts...)
	}
	return nil
}
