package realization

import (
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// ServerAttempt is the number the server gives the attempt that group `number` of activity script
// `script` runs. A reset settlement declared on the script restarts the numbering at its fresh
// group, which is the server's first attempt again; the groups after it follow it. Without one, the
// Nth group is the server's attempt N.
func ServerAttempt(r *umpirespb.Realization, script string, number int64) int64 {
	for _, e := range r.GetResetSettlements() {
		if e.GetActivity() == script && e.GetFreshAttempt() >= 1 && number >= e.GetFreshAttempt() {
			return number - e.GetFreshAttempt() + 1
		}
	}
	return number
}

// resetSettlements checks each declared reset settlement: one held attempt of a declared activity
// script publishes its pending record, which the controller awaits before a bounded held Describe of
// the learned execution and one typed ResetActivityExecution of it. The script withholds that
// attempt's answer until the selected per-attempt timer, which applies the reset; the next group is
// the server's first attempt again. A terminal Describe and bounded cleanup end it. Every request
// names the carrier's namespace and activity and the learned run, and no workflow.
func (a *realizing) resetSettlements() {
	controllers := map[string]*umpirespb.Command{}
	performing := map[string][]string{}
	activities := map[string]*umpirespb.Script{}
	var controller *umpirespb.Script
	for _, s := range a.r.GetScripts() {
		if s.GetActivity() != nil {
			activities[s.GetId()] = s
		}
		if s.GetController() == nil {
			continue
		}
		controller = s
		for _, item := range s.GetItems() {
			if c := item.GetCommand(); c != nil {
				controllers[c.GetId()] = c
			}
			for _, p := range item.GetPerforms() {
				controllers[p.GetCommand().GetId()] = p.GetCommand()
				performing[p.GetCommand().GetId()] = append(performing[p.GetCommand().GetId()], a.d.ClassKey(p.GetStep()))
			}
		}
	}
	seen := map[string]bool{}
	for _, e := range a.r.GetResetSettlements() {
		at := e.GetPosition()
		fail := func(format string, args ...any) {
			a.report(at, "reset settlement "+e.GetResetRequest()+": "+format, args...)
		}
		if e.GetResetRequest() == "" || seen[e.GetResetRequest()] || e.GetPending() == "" || a.publications[e.GetPending()] || a.learned[e.GetPending()] != nil {
			fail("missing or duplicate reset/publication identity")
			continue
		}
		seen[e.GetResetRequest()] = true
		a.publications[e.GetPending()] = true
		activity := activities[e.GetActivity()]
		if activity == nil || e.GetAttempt() < 1 || e.GetFreshAttempt() != e.GetAttempt()+1 {
			fail("needs a positive held attempt of a declared activity script and the fresh attempt after it")
			continue
		}
		var refs []*umpirespb.Command
		for _, id := range []string{e.GetCarrier(), e.GetHeld(), e.GetResetRequest(), e.GetSettlement()} {
			c := controllers[id]
			if id == "" || c == nil {
				fail("no controller command %s", id)
			}
			refs = append(refs, c)
		}
		if slices.Contains(refs, nil) {
			continue
		}
		carrier, held, reset, settlement := refs[0], refs[1], refs[2], refs[3]
		if carrier.GetRpc().GetMethod() != activityService+"StartActivityExecution" {
			fail("carrier is no StartActivityExecution")
		}
		if reset.GetRpc().GetMethod() != activityService+"ResetActivityExecution" || len(performing[reset.GetId()]) != 1 {
			fail("reset must be one typed ResetActivityExecution that performs one class")
		}
		if held.GetPoll() == nil || settlement.GetPoll() == nil {
			fail("held and settlement must be bounded Describe polls")
		}
		for _, c := range refs {
			if c.GetTimeoutMs() <= 0 && a.r.GetBehavior().GetInstructionDefaults().GetTimeoutMs() <= 0 {
				fail("command %s has no positive bound", c.GetId())
			}
			if c.GetRegardless() {
				fail("command %s must require successful predecessors", c.GetId())
			}
		}
		var publication *umpirespb.Command
		for _, c := range controllers {
			if c.GetAwaitActivityPublication() == e.GetPending() {
				if publication != nil {
					fail("duplicate publication await")
				}
				publication = c
			}
		}
		if publication == nil {
			fail("absent pending publication await")
			continue
		}
		if !controllerAfter(controller, publication.GetId(), carrier.GetId()) || !controllerAfter(controller, held.GetId(), publication.GetId()) || !controllerAfter(controller, reset.GetId(), held.GetId()) || !controllerAfter(controller, settlement.GetId(), reset.GetId()) {
			fail("publication, held, reset and settlement order is incomplete")
		}
		pending := 0
		timer := a.d.ClassKey(e.GetTimer())
		for _, item := range activity.GetItems() {
			withheld := item.GetCommand().GetAttemptWithheld()
			if withheld == nil || withheld.GetMode() != umpirespb.WITHHOLDING_MODE_SDK_PENDING {
				continue
			}
			if len(item.GetWhen()) == 1 && a.d.ClassKey(item.GetWhen()[0]) == timer && withheld.GetExternalSettlement() == "" {
				pending++
			}
		}
		if pending != 1 {
			fail("requires exactly one SDK_PENDING disposition of activity %s selected by the timer %s", e.GetActivity(), timer)
		}
		if slices.Contains(performing[reset.GetId()], timer) {
			fail("the timer that applies the reset is no class the reset performs")
		}
		cleanup := e.GetCleanup()
		if cleanup == nil || cleanup.GetId() == "" || !cleanup.GetRegardless() || cleanup.GetRpc().GetMethod() != activityService+"TerminateActivityExecution" || (cleanup.GetTimeoutMs() <= 0 && a.r.GetBehavior().GetInstructionDefaults().GetTimeoutMs() <= 0) {
			fail("cleanup requires a bounded always-run TerminateActivityExecution command")
		} else {
			guarded := proto.CloneOf(cleanup)
			guarded.Regardless = false
			a.rpc(commandOf{guarded, cleanup.GetId(), at, true, a.r.GetCleanup()}, guarded.GetRpc())
		}
		identity := assignmentOf(carrier.GetRpc().GetAssign(), "namespace")
		operation := assignmentOf(carrier.GetRpc().GetAssign(), "activity_id")
		var run string
		for _, read := range carrier.GetRpc().GetReads() {
			if read.GetPath() == "run_id" && read.GetCardinality() == umpirespb.ResponseRead_CARDINALITY_ONE {
				for _, target := range read.GetTargets() {
					if target.GetBind() != "" {
						run = target.GetBind()
					}
				}
			}
		}
		for _, rpc := range []*umpirespb.Rpc{reset.GetRpc(), cleanup.GetRpc()} {
			workflow := assignmentOf(rpc.GetAssign(), "workflow_id")
			if identity == nil || operation == nil || run == "" || !sameExternalOperand(identity, assignmentOf(rpc.GetAssign(), "namespace")) || !sameExternalOperand(operation, assignmentOf(rpc.GetAssign(), "activity_id")) || assignmentOf(rpc.GetAssign(), "run_id").GetLearnedValue() != run || (workflow != nil && (workflow.GetLiteral() == nil || workflow.GetLiteral().GetText() != "")) {
				fail("crossed or incomplete namespace/activity/learned execution run identity")
			}
		}
		for _, c := range []*umpirespb.Command{held, settlement} {
			poll := c.GetPoll()
			from := a.evidence[poll.GetEvidence()].GetSingle()
			if from.GetMethod() != activityService+"DescribeActivityExecution" || from.GetPath() != "info" || !sameExternalOperand(identity, assignmentOf(poll.GetAssign(), "namespace")) || !sameExternalOperand(operation, assignmentOf(poll.GetAssign(), "activity_id")) || assignmentOf(poll.GetAssign(), "run_id").GetLearnedValue() != run {
				fail("command %s lacks Describe info of the learned execution", c.GetId())
			}
		}
		enum := func(name string) *umpirespb.Operand {
			return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: name}}}}
		}
		if !externalEqual(held.GetPoll().GetUntil(), "status", enum("ACTIVITY_EXECUTION_STATUS_RUNNING")) || !externalEqual(held.GetPoll().GetUntil(), "run_state", enum("PENDING_ACTIVITY_STATE_STARTED")) {
			fail("held evidence lacks exact running started state")
		}
		terminal := false
		for _, status := range []string{"ACTIVITY_EXECUTION_STATUS_COMPLETED", "ACTIVITY_EXECUTION_STATUS_FAILED", "ACTIVITY_EXECUTION_STATUS_CANCELED", "ACTIVITY_EXECUTION_STATUS_TERMINATED", "ACTIVITY_EXECUTION_STATUS_TIMED_OUT"} {
			terminal = terminal || externalEqual(settlement.GetPoll().GetUntil(), "status", enum(status))
		}
		if !terminal || !externalEqual(settlement.GetPoll().GetUntil(), "run_state", enum("PENDING_ACTIVITY_STATE_UNSPECIFIED")) || !externalPresent(settlement.GetPoll().GetUntil(), "close_time") {
			fail("settlement lacks exact terminal status, cleared run state and close time")
		}
	}
}
