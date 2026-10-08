package realization

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// resetRealization is a deferred reset: the controller starts the activity, awaits the held
// attempt's publication, reads it held, resets it and reads it closed. The activity script withholds
// the held attempt's answer until the selected timer, and its next group completes.
func resetRealization() *umpirespb.Realization {
	r := externalRealization("RespondActivityTaskFailedById")
	controller, activity := r.Scripts[0], r.Scripts[1]
	held, answer, settled := controller.Items[2], controller.Items[3], controller.Items[4]
	reset := answer.Performs[0]
	reset.Step = &umpirespb.ActionClass{Action: "client.reset"}
	reset.Command.Id = "reset"
	reset.Command.GetRpc().Method = activityService + "ResetActivityExecution"
	for _, condition := range settled.Command.GetPoll().GetUntil().GetAll().GetOperands() {
		if eq := condition.GetEqual(); eq != nil && eq.GetLeft().GetPath().GetPath() == "status" {
			eq.Right.GetLiteral().Kind = &umpirespb.ProtoValue_EnumName{EnumName: "ACTIVITY_EXECUTION_STATUS_COMPLETED"}
		}
	}
	controller.Items = []*umpirespb.Item{controller.Items[0], controller.Items[1], held, answer, settled}
	activity.Items = []*umpirespb.Item{
		{When: []*umpirespb.ActionClass{{Action: "deadline.heartbeat"}}, Command: &umpirespb.Command{Id: "pending", Instruction: &umpirespb.Command_AttemptWithheld{AttemptWithheld: &umpirespb.AttemptWithheld{Mode: umpirespb.WITHHOLDING_MODE_SDK_PENDING}}}},
	}
	e := r.ExternalSettlements[0]
	r.ExternalSettlements = nil
	r.ResetSettlements = []*umpirespb.ActivityResetSettlement{{
		Carrier: e.GetCarrier(), Activity: e.GetActivity(), Attempt: 1, Pending: e.GetPending(), Held: e.GetHeld(), ResetRequest: "reset",
		Timer: &umpirespb.ActionClass{Action: "deadline.heartbeat"}, FreshAttempt: 2, Settlement: e.GetSettlement(), Cleanup: e.GetCleanup(),
	}}
	return r
}

func TestActivityResetSettlementAdmitsADeferredResetAndRefusesIncompleteBindings(t *testing.T) {
	require.Empty(t, externalProblems(resetRealization()))
	for name, mutate := range map[string]func(*umpirespb.Realization){
		"fresh attempt is not the next": func(r *umpirespb.Realization) { r.ResetSettlements[0].FreshAttempt = 3 },
		"nonpositive attempt":           func(r *umpirespb.Realization) { r.ResetSettlements[0].Attempt = 0 },
		"unknown script":                func(r *umpirespb.Realization) { r.ResetSettlements[0].Activity = "other" },
		"undeclared publication":        func(r *umpirespb.Realization) { r.ResetSettlements[0].Pending = "other" },
		"duplicate": func(r *umpirespb.Realization) {
			r.ResetSettlements = append(r.ResetSettlements, proto.CloneOf(r.ResetSettlements[0]))
		},
		"reset is no reset": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[3].Performs[0].Command.GetRpc().Method = activityService + "PauseActivityExecution"
		},
		"reset before held": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[2], r.Scripts[0].Items[3] = r.Scripts[0].Items[3], r.Scripts[0].Items[2]
		},
		"held before publication": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[1], r.Scripts[0].Items[2] = r.Scripts[0].Items[2], r.Scripts[0].Items[1]
		},
		"no timer basis": func(r *umpirespb.Realization) {
			r.Scripts[1].Items[0].When = []*umpirespb.ActionClass{{Action: "deadline.startToClose"}}
		},
		"context withholding": func(r *umpirespb.Realization) {
			r.Scripts[1].Items[0].Command.GetAttemptWithheld().Mode = umpirespb.WITHHOLDING_MODE_CONTEXT
		},
		"external answer basis": func(r *umpirespb.Realization) {
			r.Scripts[1].Items[0].Command.GetAttemptWithheld().ExternalSettlement = "reset"
		},
		"crossed reset run": func(r *umpirespb.Realization) {
			assign := r.Scripts[0].Items[3].Performs[0].Command.GetRpc().Assign
			assign[2].Value = &umpirespb.Operand{Kind: &umpirespb.Operand_LearnedValue{LearnedValue: "other-run"}}
		},
		"unpinned held run": func(r *umpirespb.Realization) {
			held := r.Scripts[0].Items[2].Command.GetPoll()
			held.Assign = slices.DeleteFunc(held.Assign, func(a *umpirespb.Assignment) bool { return a.GetTarget() == "run_id" })
		},
		"open settlement": func(r *umpirespb.Realization) {
			poll := r.Scripts[0].Items[4].Command.GetPoll()
			poll.Until = poll.GetUntil().GetAll().GetOperands()[1]
		},
		"incomplete cleanup": func(r *umpirespb.Realization) { r.ResetSettlements[0].Cleanup.Regardless = false },
	} {
		t.Run(name, func(t *testing.T) {
			r := resetRealization()
			mutate(r)
			require.NotEmpty(t, externalProblems(r))
		})
	}
}

// A declared reset restarts the server's numbering at its fresh group, and only on its own script.
func TestServerAttemptRestartsAtTheDeclaredFreshGroup(t *testing.T) {
	r := resetRealization()
	for number, want := range map[int64]int64{1: 1, 2: 1, 3: 2} {
		require.Equal(t, want, ServerAttempt(r, "attempts", number))
	}
	require.Equal(t, int64(2), ServerAttempt(r, "other", 2))
	require.Equal(t, int64(2), ServerAttempt(externalRealization("RespondActivityTaskFailedById"), "attempts", 2))
}
