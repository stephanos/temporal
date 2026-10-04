package model

// What a reader rejects of the hints a realization declares of how the APIs it calls behave, and of
// its server steps (.plans/API_BEHAVIOR_HINTS.md): each case is one mutation of a valid behavior given
// the Nexus caller realization, reported at the position of the declaration concerned.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// admBehaviorAt is where the shared Temporal kit would write a realization's hints.
const admBehaviorAt = "model/temporal/realize/Behavior.scala"

func behaviorAt(line int32) *umpirespb.Position {
	return &umpirespb.Position{File: admBehaviorAt, Line: line}
}

func admTimeout(name string) *umpirespb.ActionClass {
	return &umpirespb.ActionClass{Action: "temporal.nexuscaller.Model$package$." + name}
}

// admBehave gives the Nexus caller realization the hints its Cases would need: a workflow task's
// effect is visible to the history at once and a handler's reply to a describe eventually; each cause
// it waits for is bounded; and its three timeouts are server timers at the deadline its requests set.
func admBehave(r *umpirespb.Realization) {
	const workflowService = "/temporal.api.workflowservice.v1.WorkflowService/"
	r.Behavior = &umpirespb.ApiBehavior{
		Visibility: []*umpirespb.Visibility{
			{Id: "visibility.workflowTask.getWorkflowExecutionHistory", Position: behaviorAt(10),
				Write: &umpirespb.Visibility_Cause{Cause: umpirespb.CAUSE_KIND_WORKFLOW_TASK}, Read: workflowService + "GetWorkflowExecutionHistory"},
			{Id: "visibility.handlerReply.describeWorkflowExecution", Position: behaviorAt(20),
				Write: &umpirespb.Visibility_Cause{Cause: umpirespb.CAUSE_KIND_HANDLER_REPLY}, Read: workflowService + "DescribeWorkflowExecution",
				EventuallyWithin: &umpirespb.WaitBound{Position: behaviorAt(21), IntervalMs: 250, AtMostMs: 2000}},
			{Id: "visibility.startWorkflowExecution.getWorkflowExecutionHistory", Position: behaviorAt(25),
				Write: &umpirespb.Visibility_Method{Method: workflowService + "StartWorkflowExecution"}, Read: workflowService + "GetWorkflowExecutionHistory"},
		},
		Causes: []*umpirespb.CauseBound{
			{Id: "cause.workflowTask", Position: behaviorAt(30), Kind: umpirespb.CAUSE_KIND_WORKFLOW_TASK,
				Bound: &umpirespb.WaitBound{Position: behaviorAt(31), IntervalMs: 250, AtMostMs: 5000}},
			{Id: "cause.handlerReply", Position: behaviorAt(40), Kind: umpirespb.CAUSE_KIND_HANDLER_REPLY,
				Bound: &umpirespb.WaitBound{Position: behaviorAt(41), IntervalMs: 250, AtMostMs: 5000}},
			{Id: "cause.timer", Position: behaviorAt(50), Kind: umpirespb.CAUSE_KIND_TIMER,
				Bound: &umpirespb.WaitBound{Position: behaviorAt(51), IntervalMs: 250, AtMostMs: 3000}},
		},
	}
	r.ServerSteps = []*umpirespb.ServerStep{
		{Position: behaviorAt(60), Step: admTimeout("scheduleToClose"), Kind: umpirespb.CAUSE_KIND_TIMER, DeadlineMs: 2000},
		{Position: behaviorAt(61), Step: admTimeout("scheduleToStart"), Kind: umpirespb.CAUSE_KIND_TIMER, DeadlineMs: 2000},
		{Position: behaviorAt(62), Step: admTimeout("startToClose"), Kind: umpirespb.CAUSE_KIND_TIMER, DeadlineMs: 2000},
	}
}

func TestARealizationsBehaviorIsAdmitted(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	admBehave(m.GetRealizations()[0])
	require.NoError(t, Validate(m))
}

func TestARealizationsBehaviorIsAdmittedBeforeItIsLowered(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, r *umpirespb.Realization)
		want   string
	}{
		// Every hint is named, by an id no other hint takes.
		{"a hint with no id", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Visibility[0].Id = "" },
			admBehaviorAt + ":10: realization asyncNexus: a hint has no id"},
		{"a hint written nowhere with no id", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Causes[0].Id, r.Behavior.Causes[0].Position = "", nil
		}, "{realization}: realization asyncNexus: a hint has no id"},
		{"two hints of one id", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Causes[0].Id = r.Behavior.Visibility[0].GetId()
		}, admBehaviorAt + ":30: two hints with id visibility.workflowTask.getWorkflowExecutionHistory of realization asyncNexus"},

		// A visibility names a write and a read, and no other declares the pair.
		{"a visibility of no write", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Visibility[0].Write = nil },
			admBehaviorAt + ":10: realization asyncNexus: visibility visibility.workflowTask.getWorkflowExecutionHistory names no write"},
		{"a visibility of no method", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Visibility[2].Write = &umpirespb.Visibility_Method{}
		}, admBehaviorAt + ":25: realization asyncNexus: visibility visibility.startWorkflowExecution.getWorkflowExecutionHistory names no write"},
		{"a visibility of an unspecified cause", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Visibility[0].Write = &umpirespb.Visibility_Cause{}
		}, admBehaviorAt + ":10: realization asyncNexus: visibility visibility.workflowTask.getWorkflowExecutionHistory names no write"},
		{"a visibility of an unknown cause", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Visibility[0].Write = &umpirespb.Visibility_Cause{Cause: 99}
		}, admBehaviorAt + ":10: realization asyncNexus: visibility visibility.workflowTask.getWorkflowExecutionHistory names no write"},
		{"a visibility of no read", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Visibility[1].Read = "" },
			admBehaviorAt + ":20: realization asyncNexus: visibility visibility.handlerReply.describeWorkflowExecution names no read"},
		{"a pair declared twice", func(_ *testing.T, r *umpirespb.Realization) {
			again := proto.CloneOf(r.Behavior.Visibility[0])
			again.Id, again.Position = "visibility.again", behaviorAt(12)
			r.Behavior.Visibility = append(r.Behavior.Visibility, again)
		}, admBehaviorAt + ":12: realization asyncNexus: visibility visibility.workflowTask.getWorkflowExecutionHistory and visibility.again both declare " +
			"when a workflow task is visible to /temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory; one hint declares a pair"},

		// A cause bound is of a known kind, which no other bounds, and declares its bound.
		{"a cause bound of no kind", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Causes[0].Kind = umpirespb.CAUSE_KIND_UNSPECIFIED
		}, admBehaviorAt + ":30: realization asyncNexus: cause bound cause.workflowTask is of no known kind"},
		{"a cause bound of an unknown kind", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Causes[0].Kind = 99 },
			admBehaviorAt + ":30: realization asyncNexus: cause bound cause.workflowTask is of no known kind"},
		{"two cause bounds of one kind", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Causes[1].Kind = umpirespb.CAUSE_KIND_WORKFLOW_TASK
		}, admBehaviorAt + ":40: realization asyncNexus: cause bounds cause.workflowTask and cause.handlerReply both bound a workflow task; one hint bounds a kind"},
		{"a cause bound with no bound", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Causes[2].Bound = nil },
			admBehaviorAt + ":50: realization asyncNexus: cause bound cause.timer declares no bound"},

		// A bound and its interval are positive, and the interval is no greater than the bound.
		{"an interval of zero", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Causes[0].Bound.IntervalMs = 0 },
			admBehaviorAt + ":31: realization asyncNexus: cause bound cause.workflowTask looks every 0 milliseconds; an interval is positive"},
		{"a negative bound", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior.Causes[0].Bound.AtMostMs = -1 },
			admBehaviorAt + ":31: realization asyncNexus: cause bound cause.workflowTask waits at most -1 milliseconds; a bound is positive"},
		{"an interval greater than its bound", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Visibility[1].EventuallyWithin.IntervalMs = 3000
		},
			admBehaviorAt + ":21: realization asyncNexus: visibility visibility.handlerReply.describeWorkflowExecution looks every 3000 milliseconds and waits at most 2000; " +
				"an interval is no greater than its bound"},
		{"a bound written nowhere of its own", func(_ *testing.T, r *umpirespb.Realization) {
			r.Behavior.Visibility[1].EventuallyWithin.Position, r.Behavior.Visibility[1].EventuallyWithin.AtMostMs = nil, 0
		}, admBehaviorAt + ":20: realization asyncNexus: visibility visibility.handlerReply.describeWorkflowExecution waits at most 0 milliseconds; a bound is positive"},

		// A server step is of a known kind the realization bounds, and only a timer's names a deadline.
		{"a server step of no kind", func(_ *testing.T, r *umpirespb.Realization) { r.ServerSteps[0].Kind = umpirespb.CAUSE_KIND_UNSPECIFIED },
			admBehaviorAt + ":60: realization asyncNexus: server step scheduleToClose is of no known kind"},
		{"a server step of a kind no hint bounds", func(_ *testing.T, r *umpirespb.Realization) {
			r.ServerSteps[0].Kind, r.ServerSteps[0].DeadlineMs = umpirespb.CAUSE_KIND_DELIVERY, 0
		}, admBehaviorAt + ":60: realization asyncNexus: server step scheduleToClose is a delivery, and the realization bounds no delivery"},
		{"server steps and no behavior", func(_ *testing.T, r *umpirespb.Realization) { r.Behavior = nil },
			admBehaviorAt + ":60: realization asyncNexus: server step scheduleToClose is a timer, and the realization bounds no timer"},
		{"a timer with no deadline", func(_ *testing.T, r *umpirespb.Realization) { r.ServerSteps[1].DeadlineMs = 0 },
			admBehaviorAt + ":61: realization asyncNexus: server step scheduleToStart is a timer and names no positive deadline"},
		{"a step that is no timer with a deadline", func(_ *testing.T, r *umpirespb.Realization) {
			r.ServerSteps[1].Kind = umpirespb.CAUSE_KIND_HANDLER_REPLY
		}, admBehaviorAt + ":61: realization asyncNexus: server step scheduleToStart names a deadline of 2000 milliseconds, and only a timer's step has one"},

		// A server step is a class of the machine, declared once, which no command performs.
		{"a server step of no class", func(_ *testing.T, r *umpirespb.Realization) { r.ServerSteps[0].Step = nil },
			admBehaviorAt + ":60: realization asyncNexus: a server step is of no class"},
		{"a server step of a class the machine does not bind", func(_ *testing.T, r *umpirespb.Realization) {
			r.ServerSteps[0].Step = &umpirespb.ActionClass{Action: "temporal.worker.Worker$package$.workerResume"}
		}, admBehaviorAt + ":60: realization asyncNexus: a server step: nexusProtocol binds no action temporal.worker.Worker$package$.workerResume"},
		{"a server step declared twice", func(_ *testing.T, r *umpirespb.Realization) { r.ServerSteps[2].Step = admTimeout("scheduleToClose") },
			admBehaviorAt + ":62: realization asyncNexus: server step scheduleToClose is declared twice"},
		{"a server step some command performs", func(t *testing.T, r *umpirespb.Realization) {
			r.ServerSteps[0].Step, r.ServerSteps[0].Kind, r.ServerSteps[0].DeadlineMs = admClass(t, r), umpirespb.CAUSE_KIND_HANDLER_REPLY, 0
		}, admBehaviorAt + ":60: realization asyncNexus: server step handlerReply-async is performed by respond-async of script handler; a server step is one no command performs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := proto.Clone(load(t)).(*umpirespb.Model)
			r := m.GetRealizations()[0]
			admBehave(r)
			// A hint written nowhere is reported where the realization is, which moves with its file.
			at := fmt.Sprintf("%s:%d", r.GetPosition().GetFile(), r.GetPosition().GetLine())
			c.mutate(t, r)
			require.ErrorContains(t, Validate(m), strings.ReplaceAll(c.want, "{realization}", at))
		})
	}
}
