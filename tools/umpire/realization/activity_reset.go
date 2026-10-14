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

// resetCommands is what a realization's scripts declare that a reset settlement names: the
// controller's commands by id, the classes each performs, the activity scripts by id, and the
// controller.
type resetCommands struct {
	controllers map[string]*umpirespb.Command
	performing  map[string][]string
	activities  map[string]*umpirespb.Script
	controller  *umpirespb.Script
}

func (a *realizing) resetCommands() resetCommands {
	out := resetCommands{controllers: map[string]*umpirespb.Command{}, performing: map[string][]string{}, activities: map[string]*umpirespb.Script{}}
	for _, s := range a.r.GetScripts() {
		if s.GetActivity() != nil {
			out.activities[s.GetId()] = s
		}
		if s.GetController() == nil {
			continue
		}
		out.controller = s
		for _, item := range s.GetItems() {
			if c := item.GetCommand(); c != nil {
				out.controllers[c.GetId()] = c
			}
			for _, p := range item.GetPerforms() {
				out.controllers[p.GetCommand().GetId()] = p.GetCommand()
				out.performing[p.GetCommand().GetId()] = append(out.performing[p.GetCommand().GetId()], a.d.ClassKey(p.GetStep()))
			}
		}
	}
	return out
}

// resetSettlements checks each declared reset settlement: one held attempt of a declared activity
// script publishes its pending record, which the controller awaits before a bounded held Describe of
// the learned execution and one typed ResetActivityExecution of it. The script withholds that
// attempt's answer until the selected per-attempt timer, which applies the reset; the next group is
// the server's first attempt again. A terminal Describe and bounded cleanup end it. Every request
// names the carrier's namespace and activity and the learned run, and no workflow.
func (a *realizing) resetSettlements() {
	commands := a.resetCommands()
	seen := map[string]bool{}
	for _, e := range a.r.GetResetSettlements() {
		fail := func(format string, args ...any) {
			a.report(e.GetPosition(), "reset settlement "+e.GetResetRequest()+": "+format, args...)
		}
		if e.GetResetRequest() == "" || seen[e.GetResetRequest()] || e.GetPending() == "" || a.publications[e.GetPending()] || a.learned[e.GetPending()] != nil {
			fail("missing or duplicate reset/publication identity")
			continue
		}
		seen[e.GetResetRequest()] = true
		a.publications[e.GetPending()] = true
		activity := commands.activities[e.GetActivity()]
		if activity == nil || e.GetAttempt() < 1 || e.GetFreshAttempt() != e.GetAttempt()+1 {
			fail("needs a positive held attempt of a declared activity script and the fresh attempt after it")
			continue
		}
		var refs []*umpirespb.Command
		for _, id := range []string{e.GetCarrier(), e.GetHeld(), e.GetResetRequest(), e.GetSettlement()} {
			if commands.controllers[id] == nil {
				fail("no controller command %s", id)
			}
			refs = append(refs, commands.controllers[id])
		}
		if slices.Contains(refs, nil) {
			continue
		}
		a.resetCalls(commands, refs, fail)
		if !a.resetOrder(e, commands, refs, fail) {
			continue
		}
		a.resetDisposition(e, commands, activity, refs[2], fail)
		a.resetCleanup(e, fail)
		a.resetIdentity(e, refs, fail)
		a.resetReads(refs, fail)
	}
}

// resetCalls checks the methods and bounds of a reset settlement's controller commands: carrier,
// held read, reset and terminal read.
func (a *realizing) resetCalls(commands resetCommands, refs []*umpirespb.Command, fail func(string, ...any)) {
	carrier, held, reset, settlement := refs[0], refs[1], refs[2], refs[3]
	if carrier.GetRpc().GetMethod() != activityService+"StartActivityExecution" {
		fail("carrier is no StartActivityExecution")
	}
	if reset.GetRpc().GetMethod() != activityService+"ResetActivityExecution" || len(commands.performing[reset.GetId()]) != 1 {
		fail("reset must be one typed ResetActivityExecution that performs one class")
	}
	if held.GetPoll() == nil || settlement.GetPoll() == nil {
		fail("held and settlement must be bounded Describe polls")
	}
	for _, c := range refs {
		if durationMilliseconds(c.GetTimeout()) <= 0 && durationMilliseconds(a.r.GetBehavior().GetInstructionDefaults().GetTimeout()) <= 0 {
			fail("command %s has no positive bound", c.GetId())
		}
		if c.GetRegardless() {
			fail("command %s must require successful predecessors", c.GetId())
		}
	}
}

// resetOrder checks that the controller awaits the publication once, after the carrier, and reads
// held, resets and reads terminal after it, in that order.
func (a *realizing) resetOrder(e *umpirespb.ActivityResetSettlement, commands resetCommands, refs []*umpirespb.Command, fail func(string, ...any)) bool {
	carrier, held, reset, settlement := refs[0], refs[1], refs[2], refs[3]
	var publication *umpirespb.Command
	for _, c := range commands.controllers {
		if c.GetAwaitActivityPublication() == e.GetPending() {
			if publication != nil {
				fail("duplicate publication await")
			}
			publication = c
		}
	}
	if publication == nil {
		fail("absent pending publication await")
		return false
	}
	c := commands.controller
	if !controllerAfter(c, publication.GetId(), carrier.GetId()) || !controllerAfter(c, held.GetId(), publication.GetId()) || !controllerAfter(c, reset.GetId(), held.GetId()) || !controllerAfter(c, settlement.GetId(), reset.GetId()) {
		fail("publication, held, reset and settlement order is incomplete")
	}
	return true
}

// resetDisposition checks that the held attempt ends in exactly one SDK_PENDING disposition its
// selected timer selects, on the timer basis rather than an external answer, and that the reset
// performs no timer class.
func (a *realizing) resetDisposition(e *umpirespb.ActivityResetSettlement, commands resetCommands, activity *umpirespb.Script, reset *umpirespb.Command, fail func(string, ...any)) {
	pending := 0
	timer := a.d.ClassKey(e.GetTimer())
	for _, item := range activity.GetItems() {
		withheld := item.GetCommand().GetAttemptWithheld()
		if withheld.GetMode() == umpirespb.WITHHOLDING_MODE_SDK_PENDING && len(item.GetWhen()) == 1 && a.d.ClassKey(item.GetWhen()[0]) == timer && withheld.GetExternalSettlement() == "" {
			pending++
		}
	}
	if pending != 1 {
		fail("requires exactly one SDK_PENDING disposition of activity %s selected by the timer %s", e.GetActivity(), timer)
	}
	if slices.Contains(commands.performing[reset.GetId()], timer) {
		fail("the timer that applies the reset is no class the reset performs")
	}
}

// resetCleanup checks the cleanup: a bounded, always-run terminate of the activity.
func (a *realizing) resetCleanup(e *umpirespb.ActivityResetSettlement, fail func(string, ...any)) {
	cleanup := e.GetCleanup()
	if cleanup == nil || cleanup.GetId() == "" || !cleanup.GetRegardless() || cleanup.GetRpc().GetMethod() != activityService+"TerminateActivityExecution" || (durationMilliseconds(cleanup.GetTimeout()) <= 0 && durationMilliseconds(a.r.GetBehavior().GetInstructionDefaults().GetTimeout()) <= 0) {
		fail("cleanup requires a bounded always-run TerminateActivityExecution command")
	} else {
		guarded := proto.CloneOf(cleanup)
		guarded.Regardless = false
		a.rpc(commandOf{guarded, cleanup.GetId(), e.GetPosition(), true, a.r.GetCleanup()}, guarded.GetRpc())
	}
}

// resetIdentity checks that the reset, cleanup and both reads name the carrier's namespace and
// activity and the run it learns, and no workflow.
func (a *realizing) resetIdentity(e *umpirespb.ActivityResetSettlement, refs []*umpirespb.Command, fail func(string, ...any)) {
	carrier, held, reset, settlement := refs[0], refs[1], refs[2], refs[3]
	cleanup := e.GetCleanup()
	identity := assignmentOf(carrier.GetRpc().GetAssign(), "namespace")
	operation := assignmentOf(carrier.GetRpc().GetAssign(), "activity_id")
	run := learnedRun(carrier)
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
}

// resetReads checks the held read's running started state and the terminal read's exact terminal
// status, cleared run state and close time.
func (a *realizing) resetReads(refs []*umpirespb.Command, fail func(string, ...any)) {
	held, settlement := refs[1], refs[3]
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

// learnedRun is the learned value a carrier binds its response's run_id into, or "".
func learnedRun(carrier *umpirespb.Command) string {
	var run string
	for _, read := range carrier.GetRpc().GetReads() {
		if read.GetPath() != "run_id" || read.GetCardinality() != umpirespb.ResponseRead_CARDINALITY_ONE {
			continue
		}
		for _, target := range read.GetTargets() {
			if target.GetBind() != "" {
				run = target.GetBind()
			}
		}
	}
	return run
}
