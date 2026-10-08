package realization

import (
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

const activityService = "/temporal.api.workflowservice.v1.WorkflowService/"

func (a *realizing) externalSettlements() {
	controllers := map[string]*umpirespb.Command{}
	activities := map[string]*umpirespb.Script{}
	var controller *umpirespb.Script
	for _, s := range a.r.GetScripts() {
		if s.GetActivity() != nil {
			activities[s.GetId()] = s
		}
		if s.GetController() != nil {
			controller = s
			for _, item := range s.GetItems() {
				if c := item.GetCommand(); c != nil {
					controllers[c.GetId()] = c
				}
				for _, p := range item.GetPerforms() {
					controllers[p.GetCommand().GetId()] = p.GetCommand()
				}
			}
		}
	}
	seenAnswers, seenSlots := map[string]bool{}, map[string]bool{}
	for _, e := range a.r.GetExternalSettlements() {
		at := e.GetPosition()
		fail := func(format string, args ...any) {
			a.report(at, "external settlement "+e.GetAnswer()+": "+format, args...)
		}
		scheduled := e.GetActivity() == "" && e.GetPending() == "" && e.GetHeld() == "" && e.GetAttempt() == 0 && e.GetRequestCancel() == ""
		if e.GetAnswer() == "" || seenAnswers[e.GetAnswer()] || (!scheduled && (e.GetPending() == "" || seenSlots[e.GetPending()] || a.learned[e.GetPending()] != nil)) {
			fail("missing or duplicate answer/publication identity")
			continue
		}
		seenAnswers[e.GetAnswer()] = true
		if !scheduled {
			seenSlots[e.GetPending()] = true
		}
		activity := activities[e.GetActivity()]
		if !scheduled && (activity == nil || e.GetAttempt() < 1) {
			fail("no positive attempt of a declared activity script")
			continue
		}
		var refs []*umpirespb.Command
		ids := []string{e.GetCarrier(), e.GetAnswer(), e.GetSettlement()}
		if !scheduled {
			ids = append(ids, e.GetHeld())
		}
		for _, id := range ids {
			c := controllers[id]
			if id == "" || c == nil {
				fail("no controller command %s", id)
			}
			refs = append(refs, c)
		}
		if slices.Contains(refs, nil) {
			continue
		}
		carrier, answer, settlement := refs[0], refs[1], refs[2]
		held := controllers[e.GetHeld()]
		method := answer.GetRpc().GetMethod()
		if method != activityService+"RespondActivityTaskFailedById" && method != activityService+"RespondActivityTaskCanceledById" && method != activityService+"RespondActivityTaskCompletedById" {
			fail("answer is no typed ById unary method")
		}
		if scheduled != (method == activityService+"RespondActivityTaskCompletedById") {
			fail("completion requires the explicit controller-only basis; failure and cancellation require a held attempt")
		}
		if carrier.GetRpc().GetMethod() != activityService+"StartActivityExecution" {
			fail("carrier is no StartActivityExecution")
		}
		if (!scheduled && held.GetPoll() == nil) || settlement.GetPoll() == nil {
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
		if !scheduled {
			var publication *umpirespb.Command
			for _, c := range controllers {
				if c.GetAwaitActivityPublication() != "" && c.GetAwaitActivityPublication() == e.GetPending() {
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
			if !controllerAfter(controller, publication.GetId(), carrier.GetId()) || !controllerAfter(controller, held.GetId(), publication.GetId()) || !controllerAfter(controller, answer.GetId(), held.GetId()) || !controllerAfter(controller, settlement.GetId(), answer.GetId()) {
				fail("publication, held, answer and settlement order is incomplete")
			}
			if method == activityService+"RespondActivityTaskCanceledById" {
				cancel := controllers[e.GetRequestCancel()]
				if cancel == nil || cancel.GetRpc().GetMethod() != activityService+"RequestCancelActivityExecution" || !controllerAfter(controller, cancel.GetId(), publication.GetId()) || !controllerAfter(controller, held.GetId(), cancel.GetId()) {
					fail("cancellation requires request-cancel after publication and before held evidence")
				}
			} else if e.GetRequestCancel() != "" {
				fail("request-cancel belongs only to cancellation")
			}
		} else if !controllerAfter(controller, answer.GetId(), carrier.GetId()) || !controllerAfter(controller, settlement.GetId(), answer.GetId()) {
			fail("carrier, answer and settlement order is incomplete")
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
		for _, rpc := range []*umpirespb.Rpc{answer.GetRpc(), cleanup.GetRpc()} {
			workflow := assignmentOf(rpc.GetAssign(), "workflow_id")
			if identity == nil || operation == nil || run == "" || !sameExternalOperand(identity, assignmentOf(rpc.GetAssign(), "namespace")) || !sameExternalOperand(operation, assignmentOf(rpc.GetAssign(), "activity_id")) || assignmentOf(rpc.GetAssign(), "run_id").GetLearnedValue() != run || (workflow != nil && (workflow.GetLiteral() == nil || workflow.GetLiteral().GetText() != "")) {
				fail("crossed or incomplete namespace/activity/learned execution run identity")
			}
		}
		terminal := "ACTIVITY_EXECUTION_STATUS_FAILED"
		if scheduled {
			terminal = "ACTIVITY_EXECUTION_STATUS_COMPLETED"
		}
		if method == activityService+"RespondActivityTaskCanceledById" {
			terminal = "ACTIVITY_EXECUTION_STATUS_CANCELED"
		}
		reads := []*umpirespb.Command{settlement}
		if !scheduled {
			reads = append(reads, held)
		}
		for _, c := range reads {
			poll := c.GetPoll()
			from := a.evidence[poll.GetEvidence()].GetSingle()
			if from.GetMethod() != activityService+"DescribeActivityExecution" || from.GetPath() != "info" || !sameExternalOperand(identity, assignmentOf(poll.GetAssign(), "namespace")) || !sameExternalOperand(operation, assignmentOf(poll.GetAssign(), "activity_id")) || assignmentOf(poll.GetAssign(), "run_id").GetLearnedValue() != run {
				fail("command %s lacks Describe info of the learned execution", c.GetId())
			}
		}
		enum := func(name string) *umpirespb.Operand {
			return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: name}}}}
		}
		if !externalEqual(settlement.GetPoll().GetUntil(), "status", enum(terminal)) || !externalEqual(settlement.GetPoll().GetUntil(), "run_state", enum("PENDING_ACTIVITY_STATE_UNSPECIFIED")) || !externalPresent(settlement.GetPoll().GetUntil(), "close_time") {
			fail("settlement lacks exact terminal status, cleared run state and close time")
		}
		if !scheduled {
			state := "PENDING_ACTIVITY_STATE_STARTED"
			if method == activityService+"RespondActivityTaskCanceledById" {
				state = "PENDING_ACTIVITY_STATE_CANCEL_REQUESTED"
			}
			if !externalEqual(held.GetPoll().GetUntil(), "status", enum("ACTIVITY_EXECUTION_STATUS_RUNNING")) || !externalEqual(held.GetPoll().GetUntil(), "run_state", enum(state)) {
				fail("held evidence lacks exact running pending state")
			}
		}
		if method == activityService+"RespondActivityTaskCanceledById" {
			cancel := controllers[e.GetRequestCancel()].GetRpc()
			if !sameExternalOperand(identity, assignmentOf(cancel.GetAssign(), "namespace")) || !sameExternalOperand(operation, assignmentOf(cancel.GetAssign(), "activity_id")) || assignmentOf(cancel.GetAssign(), "run_id").GetLearnedValue() != run {
				fail("request-cancel crosses the learned execution")
			}
			if include := assignmentOf(settlement.GetPoll().GetAssign(), "include_outcome"); !include.GetLiteral().GetFlag() {
				fail("cancellation settlement must include outcome")
			}
		}
		if !scheduled {
			matches := 0
			var answerClasses []string
			for _, item := range controller.GetItems() {
				for _, p := range item.GetPerforms() {
					if p.GetCommand().GetId() == e.GetAnswer() {
						answerClasses = append(answerClasses, a.d.ClassKey(p.GetStep()))
					}
				}
			}
			for _, item := range activity.GetItems() {
				if item.GetCommand().GetAttemptWithheld().GetExternalSettlement() == e.GetAnswer() {
					matches++
					if item.GetCommand().GetAttemptWithheld().GetMode() != umpirespb.WITHHOLDING_MODE_SDK_PENDING {
						fail("external disposition must publish SDK_PENDING")
					}
					if len(item.GetWhen()) != 1 || len(answerClasses) != 1 || a.d.ClassKey(item.GetWhen()[0]) != answerClasses[0] {
						fail("pending disposition must select the controller answer's one class")
					}
				}
			}
			if matches != 1 {
				fail("requires exactly one matching pending disposition")
			}
		}
	}
	for _, c := range controllers {
		if id := c.GetAwaitActivityPublication(); id != "" && !seenSlots[id] {
			a.report(c.GetPosition(), "command %s awaits unknown activity publication %s", c.GetId(), id)
		}
	}
}

func externalEqual(condition *umpirespb.Operand, path string, value *umpirespb.Operand) bool {
	if eq := condition.GetEqual(); eq != nil {
		matches := func(p, v *umpirespb.Operand) bool {
			return p.GetPath().GetPath() == path && p.GetPath().GetOf().GetProjected() != nil && sameExternalOperand(v, value)
		}
		return matches(eq.GetLeft(), eq.GetRight()) || matches(eq.GetRight(), eq.GetLeft())
	}
	return slices.ContainsFunc(condition.GetAll().GetOperands(), func(o *umpirespb.Operand) bool { return externalEqual(o, path, value) })
}

func externalPresent(condition *umpirespb.Operand, path string) bool {
	if of := condition.GetPresent().GetOf(); of != nil {
		return of.GetPath().GetPath() == path && of.GetPath().GetOf().GetProjected() != nil
	}
	return slices.ContainsFunc(condition.GetAll().GetOperands(), func(o *umpirespb.Operand) bool { return externalPresent(o, path) })
}

func sameExternalOperand(left, right *umpirespb.Operand) bool {
	if left == nil || right == nil {
		return false
	}
	l, r := proto.CloneOf(left), proto.CloneOf(right)
	l.Position, r.Position = nil, nil
	return proto.Equal(l, r)
}

func assignmentOf(assign []*umpirespb.Assignment, path string) *umpirespb.Operand {
	for _, a := range assign {
		if a.GetTarget() == path {
			return a.GetValue()
		}
	}
	return nil
}

func controllerAfter(s *umpirespb.Script, later, earlier string) bool {
	before := map[string][]string{}
	previous := ""
	for _, item := range s.GetItems() {
		commands := []*umpirespb.Command{item.GetCommand()}
		for _, p := range item.GetPerforms() {
			commands = append(commands, p.GetCommand())
		}
		var ids []string
		for _, c := range commands {
			if c == nil {
				continue
			}
			ids = append(ids, c.GetId())
			if c.GetAfter() != nil {
				before[c.GetId()] = c.GetAfter().GetCommands()
			} else if previous != "" {
				before[c.GetId()] = []string{previous}
			}
		}
		if len(ids) > 0 {
			previous = ids[len(ids)-1]
		}
	}
	seen := map[string]bool{}
	var reaches func(string) bool
	reaches = func(id string) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		for _, parent := range before[id] {
			if parent == earlier || reaches(parent) {
				return true
			}
		}
		return false
	}
	return reaches(later)
}
