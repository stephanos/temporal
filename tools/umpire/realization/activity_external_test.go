package realization

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

type externalAdmitter struct{ rejectionAdmitter }

func (*externalAdmitter) ClassKey(c *umpirespb.ActionClass) string { return c.GetAction() }

func externalRealization(method string) *umpirespb.Realization {
	text := func(s string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Text{Text: s}}}}
	}
	identity := []*umpirespb.Assignment{{Target: "namespace", Value: text("namespace")}, {Target: "activity_id", Value: text("activity")}}
	withRun := append(cloneExternalAssignments(identity), &umpirespb.Assignment{Target: "run_id", Value: &umpirespb.Operand{Kind: &umpirespb.Operand_LearnedValue{LearnedValue: "execution-run"}}})
	rpc := func(id, method string, assign []*umpirespb.Assignment) *umpirespb.Command {
		return &umpirespb.Command{Id: id, Instruction: &umpirespb.Command_Rpc{Rpc: &umpirespb.Rpc{Role: "frontend", Method: activityService + method, Assign: assign}}}
	}
	carrier := rpc("start", "StartActivityExecution", identity)
	carrier.GetRpc().Reads = []*umpirespb.ResponseRead{{Path: "run_id", Cardinality: umpirespb.ResponseRead_CARDINALITY_ONE, Targets: []*umpirespb.Target{{Target: &umpirespb.Target_Bind{Bind: "execution-run"}}}}}
	pub := &umpirespb.Command{Id: "await-pending", Instruction: &umpirespb.Command_AwaitActivityPublication{AwaitActivityPublication: "pending"}}
	path := func(name string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: name, Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}}}}
	}
	equal := func(name string, value *umpirespb.Operand) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: path(name), Right: value}}}
	}
	enum := func(name string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: name}}}}
	}
	poll := func(id string) *umpirespb.Command {
		status, state := "ACTIVITY_EXECUTION_STATUS_RUNNING", "PENDING_ACTIVITY_STATE_STARTED"
		if method == "RespondActivityTaskCanceledById" {
			state = "PENDING_ACTIVITY_STATE_CANCEL_REQUESTED"
		}
		var conditions []*umpirespb.Operand
		if id == "settled" {
			status, state = "ACTIVITY_EXECUTION_STATUS_FAILED", "PENDING_ACTIVITY_STATE_UNSPECIFIED"
			if method == "RespondActivityTaskCanceledById" {
				status = "ACTIVITY_EXECUTION_STATUS_CANCELED"
			}
			if method == "RespondActivityTaskCompletedById" {
				status = "ACTIVITY_EXECUTION_STATUS_COMPLETED"
			}
			conditions = append(conditions, &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: path("close_time")}}})
		}
		conditions = append(conditions, equal("status", enum(status)), equal("run_state", enum(state)))
		assign := cloneExternalAssignments(withRun)
		assign = append(assign, &umpirespb.Assignment{Target: "include_outcome", Value: &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Flag{Flag: true}}}}})
		return &umpirespb.Command{Id: id, Instruction: &umpirespb.Command_Poll{Poll: &umpirespb.Poll{Evidence: id, Assign: assign, Until: &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{Operands: conditions}}}}}}
	}
	answer := rpc("answer", method, withRun)
	cleanup := rpc("terminate", "TerminateActivityExecution", cloneExternalAssignments(withRun))
	cleanup.Regardless = true
	commands := []*umpirespb.Command{carrier, pub}
	e := &umpirespb.ActivityExternalSettlement{Carrier: "start", Activity: "attempts", Attempt: 1, Pending: "pending", Held: "held", Answer: "answer", Settlement: "settled", Cleanup: cleanup}
	if method == "RespondActivityTaskCanceledById" {
		commands = append(commands, rpc("request-cancel", "RequestCancelActivityExecution", cloneExternalAssignments(withRun)))
		e.RequestCancel = "request-cancel"
	}
	commands = append(commands, poll("held"), answer, poll("settled"))
	if method == "RespondActivityTaskCompletedById" {
		commands = []*umpirespb.Command{carrier, answer, poll("settled")}
		e.Activity, e.Pending, e.Held, e.Attempt = "", "", "", 0
	}
	controller := &umpirespb.Script{Id: "controller", Activation: &umpirespb.Script_Controller{Controller: &umpirespb.Empty{}}}
	for _, c := range commands {
		item := &umpirespb.Item{Command: c}
		if c == answer {
			item.Command = nil
			item.Performs = []*umpirespb.Performance{{Step: &umpirespb.ActionClass{Action: "service.answer"}, Command: c}}
		}
		controller.Items = append(controller.Items, item)
	}
	activity := &umpirespb.Script{Id: "attempts", Activation: &umpirespb.Script_Activity{Activity: &umpirespb.ActivityActivation{}}, Items: []*umpirespb.Item{{When: []*umpirespb.ActionClass{{Action: "service.answer"}}, Command: &umpirespb.Command{Id: "publish", Instruction: &umpirespb.Command_AttemptWithheld{AttemptWithheld: &umpirespb.AttemptWithheld{Mode: umpirespb.WITHHOLDING_MODE_SDK_PENDING, ExternalSettlement: "answer"}}}}}}
	scripts := []*umpirespb.Script{controller, activity}
	if method == "RespondActivityTaskCompletedById" {
		scripts = []*umpirespb.Script{controller}
	}
	return &umpirespb.Realization{Cleanup: "cleanup", Scripts: scripts, Evidence: []*umpirespb.Evidence{{Id: "held", From: &umpirespb.Evidence_Single{Single: &umpirespb.ReadSource{Method: activityService + "DescribeActivityExecution", Path: "info"}}}, {Id: "settled", From: &umpirespb.Evidence_Single{Single: &umpirespb.ReadSource{Method: activityService + "DescribeActivityExecution", Path: "info"}}}}, ExternalSettlements: []*umpirespb.ActivityExternalSettlement{e}, Behavior: &umpirespb.ApiBehavior{InstructionDefaults: &umpirespb.InstructionLimit{TimeoutMs: 1000, Attempts: 1}}}
}

func cloneExternalAssignments(in []*umpirespb.Assignment) []*umpirespb.Assignment {
	out := make([]*umpirespb.Assignment, len(in))
	for i, a := range in {
		out[i] = proto.CloneOf(a)
	}
	return out
}

func externalProblems(r *umpirespb.Realization) []string {
	d := &externalAdmitter{}
	a := &realizing{d: d, r: r, roles: map[string]*umpirespb.Role{"frontend": {Kind: umpirespb.Role_KIND_ENDPOINT}}, learned: map[string]*umpirespb.Learned{"execution-run": {Kind: umpirespb.Learned_KIND_TEXT}}, read: map[string]bool{}, bound: map[string]string{}, observations: map[string]bool{}, publications: map[string]bool{}}
	a.evidence = map[string]*umpirespb.Evidence{}
	for _, e := range r.GetEvidence() {
		a.evidence[e.GetId()] = e
	}
	a.externalSettlements()
	a.resetSettlements()
	a.publicationAwaits()
	return d.problems
}

func TestActivityExternalSettlementAdmitsTimerFreePendingAndRejectsIncompleteBindings(t *testing.T) {
	for _, method := range []string{"RespondActivityTaskCompletedById", "RespondActivityTaskFailedById", "RespondActivityTaskCanceledById"} {
		t.Run(method, func(t *testing.T) { require.Empty(t, externalProblems(externalRealization(method))) })
	}
	for name, mutate := range map[string]func(*umpirespb.Realization){
		"missing": func(r *umpirespb.Realization) { r.ExternalSettlements[0].Answer = "absent" },
		"duplicate": func(r *umpirespb.Realization) {
			r.ExternalSettlements = append(r.ExternalSettlements, proto.CloneOf(r.ExternalSettlements[0]))
		},
		"unknown-publication": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[1].Command.Instruction = &umpirespb.Command_AwaitActivityPublication{AwaitActivityPublication: "unknown"}
		},
		"crossed": func(r *umpirespb.Realization) { r.ExternalSettlements[0].Activity = "other" },
		"wrong-method": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[3].Performs[0].Command.GetRpc().Method = activityService + "RespondActivityTaskFailed"
		},
		"wrong-request": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[3].Performs[0].Command.GetRpc().Assign[1].Value.GetLiteral().Kind = &umpirespb.ProtoValue_Text{Text: "other"}
		},
		"early-publication": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[0], r.Scripts[0].Items[1] = r.Scripts[0].Items[1], r.Scripts[0].Items[0]
		},
		"late-publication": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[1], r.Scripts[0].Items[3] = r.Scripts[0].Items[3], r.Scripts[0].Items[1]
		},
		"unpinned-held-run": func(r *umpirespb.Realization) {
			held := r.Scripts[0].Items[2].Command.GetPoll()
			held.Assign = slices.DeleteFunc(held.Assign, func(a *umpirespb.Assignment) bool { return a.GetTarget() == "run_id" })
		},
		"crossed-settlement-run": func(r *umpirespb.Realization) {
			r.Scripts[0].Items[4].Command.GetPoll().Assign[2].Value = &umpirespb.Operand{Kind: &umpirespb.Operand_LearnedValue{LearnedValue: "other-run"}}
		},
		"incomplete-cleanup": func(r *umpirespb.Realization) { r.ExternalSettlements[0].Cleanup.Regardless = false },
		"wrong-cleanup-target": func(r *umpirespb.Realization) {
			r.ExternalSettlements[0].Cleanup.GetRpc().Assign[1].Value.GetLiteral().Kind = &umpirespb.ProtoValue_Text{Text: "other"}
		},
		"nonpositive-attempt": func(r *umpirespb.Realization) { r.ExternalSettlements[0].Attempt = 0 },
		"duplicate-disposition": func(r *umpirespb.Realization) {
			r.Scripts[1].Items = append(r.Scripts[1].Items, proto.CloneOf(r.Scripts[1].Items[0]))
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := externalRealization("RespondActivityTaskFailedById")
			mutate(r)
			require.NotEmpty(t, externalProblems(r))
		})
	}
}

func TestActivityExternalSettlementRequiresHeldAndTerminalEvidence(t *testing.T) {
	r := externalRealization("RespondActivityTaskFailedById")
	r.Scripts[0].Items[2].Command.GetPoll().Until = nil
	require.NotEmpty(t, externalProblems(r), "a bare Describe without a held state and terminal condition is no settlement proof")
}
