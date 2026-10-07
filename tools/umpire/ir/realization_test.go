package ir

// What a reader rejects of a realization before any lowering (model/SEMANTICS.md, Admission):
// each case is one mutation of the Nexus caller realization the lifter emitted, reported at the
// position the lifter recorded for the declaration concerned.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

const admRealizationAt = "model/temporal/features/nexus/workflow/Realization.scala:"

// admKitAt is where the shared Temporal kit (model/temporal/realize) writes the declarations of the
// realizations it builds: their roles and correlation, the evidence of the Run's own record, and the
// reads they poll.
const admKitAt = "model/temporal/realize/Kit.scala:"

// requireLocated requires an error located in the realization's own file or in the kit.
func requireLocated(t *testing.T, err error, realization string) {
	t.Helper()
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), realization) || strings.Contains(err.Error(), admKitAt),
		"%q is located in neither %s nor %s", err, realization, admKitAt)
}

// admScript is a script of the Nexus caller realization, by id.
func admScript(t *testing.T, r *umpirespb.Realization, id string) *umpirespb.Script {
	t.Helper()
	for _, s := range r.GetScripts() {
		if s.GetId() == id {
			return s
		}
	}
	require.FailNow(t, "no script "+id)
	return nil
}

// admCommand is a command of a script, by id, wherever an item or a performance holds it.
func admCommand(t *testing.T, s *umpirespb.Script, id string) *umpirespb.Command {
	t.Helper()
	for _, item := range s.GetItems() {
		if item.GetCommand().GetId() == id {
			return item.GetCommand()
		}
		for _, p := range item.GetPerforms() {
			if p.GetCommand().GetId() == id {
				return p.GetCommand()
			}
		}
	}
	require.FailNow(t, "no command "+id)
	return nil
}

// admClass is a class the Nexus caller's machine binds: the asynchronous reply the handler's script
// performs first.
func admClass(t *testing.T, r *umpirespb.Realization) *umpirespb.ActionClass {
	t.Helper()
	return proto.CloneOf(admScript(t, r, "handler").GetItems()[0].GetPerforms()[0].GetStep())
}

// A fact that steps of several classes record, or one class records twice on a path, may have a kind
// of evidence for each: kinds that name the steps they confirm may record one fact, beside at most one
// kind that names none.
func TestKindsThatNameTheirStepsMayRecordOneFact(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	r := m.GetRealizations()[0]
	// The fact is one no exhaustive kind records: an exhaustive kind is the one kind of its fact.
	unclosed(r)
	r.Evidence[2].Records, r.Evidence[3].Records = r.Evidence[1].GetRecords(), r.Evidence[1].GetRecords()
	r.Evidence[2].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 1}}
	r.Evidence[3].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 2}}
	require.NoError(t, Validate(m))
}

func TestTheLiftedRealizationIsAdmitted(t *testing.T) {
	m := load(t)
	require.Len(t, m.GetRealizations(), 1)
	require.Equal(t, "nexusSystem", m.GetRealizations()[0].GetMachine())
}

// unclosed takes every exhaustive declaration and closing read out of a realization, so that a case
// states the whole of the one it is about.
func unclosed(r *umpirespb.Realization) {
	for _, e := range r.GetEvidence() {
		e.Exhaustive = false
	}
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			if item.GetCommand() != nil {
				item.GetCommand().Closes = nil
			}
		}
	}
}

func admWritten(kind any) *umpirespb.Operand {
	value := &umpirespb.ProtoValue{}
	switch k := kind.(type) {
	case *umpirespb.ProtoValue_Text:
		value.Kind = k
	case *umpirespb.ProtoValue_Number:
		value.Kind = k
	default:
	}
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: value}}
}

// admPayload reads a path of the value a poll is looking at, or of a Run Event's payload.
func admPayload(path string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: path,
		Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}}}}
}

func admGreater(left, right *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Greater{Greater: &umpirespb.Greater{Left: left, Right: right}}}
}

func admNot(of *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Not{Not: &umpirespb.Not{Of: of}}}
}

// runEvent makes the started kind of the Nexus caller realization the Run's own record of the start
// call's completion, keyed by the run, and then changes that source.
func runEvent(r *umpirespb.Realization, change func(*umpirespb.RunEventSource)) {
	unclosed(r)
	source := &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED, Script: "controller", Command: "start-workflow",
		Key: &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &umpirespb.Empty{}}}}
	change(source)
	r.Evidence[1].Operation = ""
	r.Evidence[1].From = &umpirespb.Evidence_RunEvent{RunEvent: source}
}

// A Run Event source that names its command, keys by the run or a path of the payload and guards on
// the payload alone is admitted. What a worker reports of an activation is admitted as the record of
// an attempt alone (TestTheAttemptARunEventRecordsIsOfAnActivitysScript).
func TestARunEventSourceIsAdmitted(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	projected := &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}
	path := func(p string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Of: projected, Path: p}}}
	}
	present := func(o *umpirespb.Operand) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: o}}}
	}
	runEvent(m.GetRealizations()[0], func(e *umpirespb.RunEventSource) {
		e.Key = path("activity_attempt.activity_run_id")
		e.Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{Operands: []*umpirespb.Operand{
			present(path("activity_attempt")),
			admGreater(path("activity_attempt.sdk_attempt"), admWritten(&umpirespb.ProtoValue_Number{Number: 0})),
			admNot(&umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: path("activity_attempt.delivery_id"),
				Right: admWritten(&umpirespb.ProtoValue_Text{Text: ""})}}})}}}}
	})
	require.NoError(t, Validate(m))
}

func TestARealizationIsAdmittedBeforeItIsLowered(t *testing.T) {
	rpc := func(id string, after ...string) *umpirespb.Item {
		c := &umpirespb.Command{Id: id, Instruction: &umpirespb.Command_Rpc{Rpc: &umpirespb.Rpc{
			Role: "temporal.workflow-service", Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeNamespace"}}}
		if after != nil {
			c.After = &umpirespb.After{Commands: after}
		}
		return &umpirespb.Item{Command: c}
	}
	cases := []struct {
		name   string
		mutate func(t *testing.T, m *umpirespb.Model, r *umpirespb.Realization)
		want   string
	}{
		// A control that holds what a step dispatches names a class its machine binds, and a
		// task-queue role.
		{"a control that holds the dispatch of no class", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Controls = append(r.Controls, &umpirespb.Control{Id: "held", Role: "temporal.task-queue",
				Kind: &umpirespb.Control_HoldDispatched{HoldDispatched: &umpirespb.HoldDispatched{}}})
		}, "realization asyncNexus: control held holds what a step of no class dispatches"},
		{"a control that holds the dispatch of a class the machine does not bind", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Controls = append(r.Controls, &umpirespb.Control{Id: "held", Role: "temporal.task-queue",
				Kind: &umpirespb.Control_HoldDispatched{HoldDispatched: &umpirespb.HoldDispatched{Step: &umpirespb.ActionClass{Action: "temporal.shared.worker.worker.resume"}}}})
		}, "realization asyncNexus: control held: nexusSystem binds no action temporal.shared.worker.worker.resume"},
		{"a control that holds the deliveries of no task queue", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			step := admScript(t, r, "handler").GetItems()[0].GetPerforms()[0].GetStep()
			r.Controls = append(r.Controls, &umpirespb.Control{Id: "held", Role: "temporal.workflow-service",
				Kind: &umpirespb.Control_HoldDispatched{HoldDispatched: &umpirespb.HoldDispatched{Step: step}}})
		}, "realization asyncNexus: control held: role temporal.workflow-service is"},
		{"a control that holds a dispatch on no role", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			step := admScript(t, r, "handler").GetItems()[0].GetPerforms()[0].GetStep()
			r.Controls = append(r.Controls, &umpirespb.Control{Id: "held",
				Kind: &umpirespb.Control_HoldDispatched{HoldDispatched: &umpirespb.HoldDispatched{Step: step}}})
		}, "realization asyncNexus: control held holds deliveries and names no task-queue role"},
		// Absent and empty ids.
		{"a realization with no name", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Name = "" },
			"a realization has no name"},
		{"a realization of no machine", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Machine = "nope" },
			"realization asyncNexus: no machine nope"},
		{"a role with no id", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Roles[1].Id = "" },
			"realization asyncNexus: a role has no id"},
		{"a learned value with no id", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Learned[0].Id = "" },
			"realization asyncNexus: a learned value has no id"},
		{"a command with no id", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").Id = ""
		}, "realization asyncNexus: a command of script controller has no id"},
		{"evidence that names no recorded kind", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Evidence[1].Records = "" },
			"evidence temporal.features.nexus.workflow.evidence.started names no recorded kind"},
		{"evidence with no operation key", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Evidence[1].Operation = "" },
			"evidence temporal.features.nexus.workflow.evidence.started names no field that keys its operation"},
		{"evidence read from nowhere", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Evidence[1].From = nil },
			"evidence temporal.features.nexus.workflow.evidence.started is recorded nowhere"},
		{"evidence of no commitment", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Commitment = umpirespb.Evidence_COMMITMENT_UNSPECIFIED
		}, "evidence temporal.features.nexus.workflow.evidence.started is of no known commitment"},
		{"a role of no kind", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Roles[0].Kind = 0 },
			"role temporal.workflow-service is of no known kind"},
		{"a command with no instruction", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").Instruction = nil
		}, "command start-workflow of script controller names no instruction"},
		{"a script with no activation", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admScript(t, r, "workflow").Activation = nil
		}, "script workflow names no activation"},
		{"no correlation", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Correlation = nil },
			"realization asyncNexus declares no correlation"},

		// Duplicate declarations and bindings.
		{"two realizations of one name", func(_ *testing.T, m *umpirespb.Model, r *umpirespb.Realization) {
			m.Realizations = append(m.Realizations, proto.Clone(r).(*umpirespb.Realization))
		}, "two realizations named asyncNexus"},
		{"two roles of one id", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) { r.Roles[1].Id = r.Roles[0].GetId() },
			"two roles with id temporal.workflow-service of realization asyncNexus"},
		{"two commands of one id in a script", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").Id = "history"
		}, "two commands with id history of script controller of realization asyncNexus"},
		{"two kinds of evidence for one recorded kind", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[2].Records = r.Evidence[1].GetRecords()
		}, "two kinds of evidence recording nexusOperationStarted of realization asyncNexus"},
		{"evidence that confirms a step counted from below one", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Step: admClass(t, r)}}
		}, "evidence temporal.features.nexus.workflow.evidence.started confirms step 0 of class reply-async; the steps of a class on a path are counted from one"},
		{"evidence that confirms one step twice", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 2}, {Step: admClass(t, r), Occurrence: 2}}
		}, "evidence temporal.features.nexus.workflow.evidence.started confirms step 2 of class reply-async twice"},
		{"two kinds of evidence that confirm one step", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 1}}
			r.Evidence[2].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "evidence temporal.features.nexus.workflow.evidence.started and temporal.features.nexus.workflow.evidence.completed both confirm step 1 of class reply-async; one kind confirms a step"},
		{"evidence that confirms a step of no class", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Occurrence: 1}}
		}, "evidence temporal.features.nexus.workflow.evidence.started confirms a step of no class"},
		{"evidence that confirms a class written with too few inputs", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Step: &umpirespb.ActionClass{Action: admClass(t, r).GetAction()}, Occurrence: 1}}
		}, "realization asyncNexus: evidence temporal.features.nexus.workflow.evidence.started: temporal.features.nexus.handler.reply takes 1 inputs, not 0"},
		{"two kinds of evidence for one recorded kind, one of which names its steps", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[2].Records, r.Evidence[3].Records = r.Evidence[1].GetRecords(), r.Evidence[1].GetRecords()
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "two kinds of evidence recording nexusOperationStarted of realization asyncNexus"},
		{"an exhaustive kind that names the steps it confirms", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[1].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "evidence temporal.features.nexus.workflow.evidence.started is exhaustive and names the steps it confirms; an exhaustive kind reports every step that records its fact"},
		{"a fact an exhaustive kind records, recorded by a second kind that names its steps", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Records = r.Evidence[1].GetRecords()
			r.Evidence[6].Confirms = []*umpirespb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts records nexusOperationStarted, every occurrence of which the exhaustive temporal.features.nexus.workflow.evidence.started reports"},
		{"a learned value bound twice", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "handler"), "respond-sync").GetNexusReply().Binds = "completion-authority"
		}, "learned value completion-authority is bound by respond-async of script handler and by respond-sync of script handler; a learned value is bound once"},
		{"a class performed twice", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			performs := admScript(t, r, "handler").GetItems()[0].GetPerforms()
			performs[1].Step = performs[0].GetStep()
		}, "class reply-async is performed by respond-async of script handler and by respond-sync of script handler; a class is performed once"},

		// References that name nothing, or the wrong kind of thing.
		{"a call on an undeclared role", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").GetRpc().Role = "nobody"
		}, "command start-workflow of script controller: no role nobody"},
		{"a call on a worker", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").GetRpc().Role = "temporal.worker"
		}, "command start-workflow of script controller: role temporal.worker is a worker, not an endpoint"},
		{"a worker script whose task queue is an endpoint", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admScript(t, r, "workflow").GetWorkflow().TaskQueue = "temporal.nexus-endpoint"
		}, "script workflow: role temporal.nexus-endpoint is an endpoint, not a task queue"},
		{"a wait for an undeclared learned value", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			c := admCommand(t, admScript(t, r, "controller"), "await-completion-authority")
			c.Instruction = &umpirespb.Command_AwaitLearned{AwaitLearned: "nothing"}
		}, "command await-completion-authority of script controller: no learned value nothing"},
		{"a learned value read and never bound", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			for _, p := range admScript(t, r, "handler").GetItems()[0].GetPerforms() {
				p.GetCommand().GetNexusReply().Binds = ""
			}
		}, "learned value completion-authority is read and no command binds it"},
		{"a text read as a handle", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Learned[0].Kind = umpirespb.Learned_KIND_TEXT
		}, "learned value completion-authority is a text, not a handle"},
		{"a handle read as a value", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			rpc := admCommand(t, admScript(t, r, "controller"), "await-close").GetRpc()
			rpc.Assign[1].Value = &umpirespb.Operand{Kind: &umpirespb.Operand_LearnedValue{LearnedValue: "completion-authority"}}
		}, "command await-close of script controller: learned value completion-authority is a handle, not a text"},
		{"a poll of evidence that is no read", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-scheduled").GetPoll().Evidence = "temporal.features.nexus.workflow.evidence.started"
		}, "command await-scheduled of script controller: evidence temporal.features.nexus.workflow.evidence.started is a history event, which a poll does not read"},
		{"a read into an undeclared observation", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			targets := admCommand(t, admScript(t, r, "controller"), "history").GetRpc().GetReads()[0].GetTargets()
			targets[0].Target = &umpirespb.Target_Observe{Observe: "unseen"}
		}, "command history of script controller: no observation unseen"},
		{"a wait for a command of another script", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			c := admCommand(t, admScript(t, r, "workflow"), "await-nexus-operation")
			c.Instruction = &umpirespb.Command_AwaitCommand{AwaitCommand: "start-workflow"}
		}, "command await-nexus-operation of script workflow: no command start-workflow of script workflow"},
		{"a class the machine does not bind", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admScript(t, r, "controller").GetItems()[0].GetPerforms()[0].Step = &umpirespb.ActionClass{Action: "temporal.nexuscaller.Model$package$.nope"}
		}, "no action temporal.nexuscaller.Model$package$.nope"},
		{"a release of an undeclared control", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").Instruction = &umpirespb.Command_Release{Release: "gate"}
		}, "command await-close of script controller: no control gate"},
		{"a control that holds an undeclared channel", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Controls = append(r.Controls, &umpirespb.Control{Id: "gate", Kind: &umpirespb.Control_HoldDelivery{HoldDelivery: "no.channel"}})
		}, "control gate: no channel no.channel"},

		{"a learned value read by a command that runs regardless", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Learned = append(r.Learned, &umpirespb.Learned{Id: "run", Kind: umpirespb.Learned_KIND_TEXT})
			s := admScript(t, r, "controller")
			start := admCommand(t, s, "start-workflow").GetRpc()
			start.Reads = append(start.Reads, &umpirespb.ResponseRead{Path: "run_id", Cardinality: umpirespb.ResponseRead_CARDINALITY_ONE,
				Targets: []*umpirespb.Target{{Target: &umpirespb.Target_Bind{Bind: "run"}}}})
			closing := admCommand(t, s, "await-close")
			closing.Regardless = true
			closing.GetRpc().Assign = append(closing.GetRpc().Assign, &umpirespb.Assignment{Target: "execution.run_id",
				Value: &umpirespb.Operand{Kind: &umpirespb.Operand_LearnedValue{LearnedValue: "run"}}})
		}, "command await-close of script controller runs whatever became of the commands before it, and reads learned value run, which is read only once it is bound"},

		// Crossed correlations.
		{"runs and operations keyed by one field", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Correlation.Operation = r.GetCorrelation().GetRun()
		}, "the correlation keys its runs and its operations by one field, temporal.features.nexus.workflow.scope.run"},
		{"evidence lifted into an observation the correlation does not read", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			targets := admCommand(t, admScript(t, r, "controller"), "history").GetRpc().GetReads()[0].GetTargets()
			targets[1].Target = &umpirespb.Target_Lift{Lift: "history-event"}
		}, "command history of script controller lifts evidence into history-event, and the correlation reads correlated-evidence"},
		{"a value observed into the correlation's observation", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			targets := admCommand(t, admScript(t, r, "controller"), "history").GetRpc().GetReads()[0].GetTargets()
			targets[0].Target = &umpirespb.Target_Observe{Observe: "correlated-evidence"}
		}, "command history of script controller observes a value into correlated-evidence, which holds the correlation's evidence"},
		{"a correlation over an undeclared observation", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Correlation.Observation = "elsewhere"
		}, "the correlation: no observation elsewhere"},

		// Dependency cycles.
		{"a command that runs after itself", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").After = &umpirespb.After{Commands: []string{"await-close"}}
		}, "script controller: command await-close runs after itself"},
		{"two commands that each run after the other", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			s := admScript(t, r, "controller")
			s.Items = append(s.Items, rpc("left", "right"), rpc("right", "left"))
		}, "script controller: command left runs after itself, through right"},
		{"a command that runs after one of no script", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").After = &umpirespb.After{Commands: []string{"finish-workflow"}}
		}, "command await-close of script controller: no command finish-workflow of script controller"},

		// Shapes the IR does not have.
		{"an item that is nothing", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			s := admScript(t, r, "controller")
			s.Items = append(s.Items, &umpirespb.Item{})
		}, "script controller has an item that is neither a command nor the place steps are performed"},
		{"an item that is both", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			item := admScript(t, r, "controller").GetItems()[0]
			item.Command = rpc("extra").GetCommand()
		}, "script controller has an item that is both a command and the place steps are performed"},
		{"an operand of no kind", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").GetRpc().GetAssign()[0].Value = &umpirespb.Operand{}
		}, "command await-close of script controller: an operand of no known kind"},
		{"a message with no name", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "workflow"), "start-nexus-operation").GetWorkflowCommand().GetCommand().Message = ""
		}, "command start-nexus-operation of script workflow: a message with no name"},
		{"a message that sets a field twice", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			c := admCommand(t, admScript(t, r, "workflow"), "start-nexus-operation").GetWorkflowCommand().GetCommand()
			c.Fields = append(c.Fields, c.GetFields()[0])
		}, "command start-nexus-operation of script workflow: temporal.api.command.v1.Command sets command_type twice"},
		{"a deadline below zero", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").TimeoutMs = -1
		}, "command await-close of script controller has a deadline of -1 milliseconds"},
		// Evidence fields and the identities they name.
		{"an evidence field with no id", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Fields = []*umpirespb.EvidenceField{{Path: "attempt"}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts has a field with no id"},
		{"an evidence field declared twice", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Fields = []*umpirespb.EvidenceField{{Id: "attempt", Path: "attempt"}, {Id: "attempt", Path: "attempt"}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts declares field attempt twice"},
		{"an evidence field read from nowhere", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Fields = []*umpirespb.EvidenceField{{Id: "attempt"}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts: field attempt names no path"},
		{"an evidence field of no known role", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Fields = []*umpirespb.EvidenceField{{Id: "attempt", Path: "attempt", Role: 9}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts: field attempt names an identity of no known role"},
		{"a role on a field the kind does not retain", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Fields = []*umpirespb.EvidenceField{{Id: "attempt", Path: "attempt", Role: umpirespb.EvidenceField_ROLE_ATTEMPT, Redacted: true}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts: field attempt names the attempt and is redacted; an identity is read from a field the evidence retains"},
		{"two fields of one role", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Fields = []*umpirespb.EvidenceField{{Id: "attempt", Path: "attempt", Role: umpirespb.EvidenceField_ROLE_ATTEMPT},
				{Id: "again", Path: "attempt", Role: umpirespb.EvidenceField_ROLE_ATTEMPT}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts: fields attempt and again both name the attempt; one field of a kind names an identity"},
		{"evidence read from one message at no path", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].From = &umpirespb.Evidence_Single{Single: &umpirespb.ReadSource{Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"}}
		}, "evidence temporal.features.nexus.workflow.evidence.pendingAttempts reads no method or no path"},

		// Exhaustive kinds and their closing reads.
		{"an exhaustive kind no command closes", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			unclosed(r)
			r.Evidence[1].Exhaustive = true
		}, "evidence temporal.features.nexus.workflow.evidence.started is exhaustive and no command closes it"},
		{"a closing read of an undeclared kind", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"nothing"}
		}, "command history of script controller closes evidence nothing, which is not declared"},
		{"a closing read of a kind that is not exhaustive", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			unclosed(r)
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"temporal.features.nexus.workflow.evidence.started"}
		}, "command history of script controller closes evidence temporal.features.nexus.workflow.evidence.started, which is not exhaustive"},
		{"a kind closed by a command that does not read it", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			unclosed(r)
			r.Evidence[1].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "await-close").Closes = []string{"temporal.features.nexus.workflow.evidence.started"}
		}, "command await-close of script controller closes evidence temporal.features.nexus.workflow.evidence.started and does not read it: a history kind is closed by the read that lifts it, the Run's own record of a command by that command, and any other by a poll of it"},
		{"a polled kind closed by a history read", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			unclosed(r)
			r.Evidence[6].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"temporal.features.nexus.workflow.evidence.pendingAttempts"}
		}, "command history of script controller closes evidence temporal.features.nexus.workflow.evidence.pendingAttempts and does not read it"},
		{"a kind closed twice", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			unclosed(r)
			r.Evidence[1].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"temporal.features.nexus.workflow.evidence.started", "temporal.features.nexus.workflow.evidence.started"}
		}, "evidence temporal.features.nexus.workflow.evidence.started is closed by history of script controller and by history of script controller; an exhaustive kind has one closing read"},
		{"a closing read only some Cases carry", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			unclosed(r)
			r.Evidence[6].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").Closes = []string{"temporal.features.nexus.workflow.evidence.pendingAttempts"}
		}, "command pending-attempts of script controller closes evidence temporal.features.nexus.workflow.evidence.pendingAttempts and is not a command every Case carries"},

		// An activity's attempts.
		{"an attempt failure outside an activity", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "handler"), "respond-sync").Instruction = &umpirespb.Command_AttemptFailure{AttemptFailure: &umpirespb.AttemptFailure{
				Failure: &umpirespb.Proto{Message: "temporal.api.failure.v1.Failure"}}}
		}, "command respond-sync of script handler fails an attempt, and script handler is no activity's"},
		{"an attempt failure with no failure", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			s := admScript(t, r, "handler")
			s.Activation = &umpirespb.Script_Activity{Activity: &umpirespb.ActivityActivation{ActivityType: &umpirespb.Name{Prefix: "activity"},
				Worker: "temporal.worker", TaskQueue: "temporal.handler-task-queue"}}
			admCommand(t, s, "respond-sync").Instruction = &umpirespb.Command_AttemptFailure{AttemptFailure: &umpirespb.AttemptFailure{}}
		}, "command respond-sync of script handler: a message with no name"},
		{"a delivery that a command also performs", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			s := admScript(t, r, "handler")
			s.Activation = &umpirespb.Script_Activity{Activity: &umpirespb.ActivityActivation{ActivityType: &umpirespb.Name{Prefix: "activity"},
				Worker: "temporal.worker", TaskQueue: "temporal.handler-task-queue", Starts: []*umpirespb.ActionClass{s.GetItems()[0].GetPerforms()[0].GetStep()}}}
		}, "class reply-async is performed by the activation of script handler and by respond-async of script handler; a class is performed once"},
		{"a delivery of a class the machine does not bind", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			s := admScript(t, r, "handler")
			s.Activation = &umpirespb.Script_Activity{Activity: &umpirespb.ActivityActivation{ActivityType: &umpirespb.Name{Prefix: "activity"},
				Worker: "temporal.worker", TaskQueue: "temporal.handler-task-queue",
				Starts: []*umpirespb.ActionClass{{Action: "temporal.nexuscaller.Model$package$.nope"}}}}
		}, "no action temporal.nexuscaller.Model$package$.nope"},

		{"a conjunction of nothing", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{}}}
		}, "command pending-attempts of script controller: a conjunction of no operand"},
		{"a conjunction over an operand of no kind", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &umpirespb.Operand{Kind: &umpirespb.Operand_All{
				All: &umpirespb.All{Operands: []*umpirespb.Operand{{}}}}}
		}, "command pending-attempts of script controller: an operand of no known kind"},
		// What an operand computes is of the type its place takes.
		{"an order of a text", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admGreater(admPayload("attempt"), admWritten(&umpirespb.ProtoValue_Text{Text: "one"}))
		}, "command pending-attempts of script controller: orders a text, and only numbers are ordered"},
		{"an order of a condition", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admGreater(admNot(admPayload("attempt")), admWritten(&umpirespb.ProtoValue_Number{Number: 0}))
		}, "command pending-attempts of script controller: orders a condition, and only numbers are ordered"},
		{"a negation of a number", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admNot(admWritten(&umpirespb.ProtoValue_Number{Number: 1}))
		}, "command pending-attempts of script controller: negates a number, and only a condition is negated"},
		{"a negation of nothing", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admNot(nil)
		}, "command pending-attempts of script controller: an operand of no known kind"},
		{"a conjunction over a text", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &umpirespb.Operand{Kind: &umpirespb.Operand_All{
				All: &umpirespb.All{Operands: []*umpirespb.Operand{admWritten(&umpirespb.ProtoValue_Text{Text: "yes"})}}}}
		}, "command pending-attempts of script controller: joins a text, and only conditions are joined"},
		{"a comparison of a text with a number", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{
				Left: admWritten(&umpirespb.ProtoValue_Text{Text: "1"}), Right: admWritten(&umpirespb.ProtoValue_Number{Number: 1})}}}
		}, "command pending-attempts of script controller: compares a text with a number"},
		{"a comparison of a value that is no flag, number, text or enum value", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{
				Left:  admPayload("attempt"),
				Right: &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Utf8{Utf8: "one"}}}}}}}
		}, "command pending-attempts of script controller: compares a value that is no text, flag, integer or enum value"},
		{"a poll until a number", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admWritten(&umpirespb.ProtoValue_Number{Number: 1})
		}, "command pending-attempts of script controller polls until a number, and a poll's condition is a condition"},
		{"a Run Event whose guard is a number", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) { e.Guard = admWritten(&umpirespb.ProtoValue_Number{Number: 1}) })
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard is a number, and a guard is a condition"},
		{"a Run Event whose guard orders a text", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Guard = admGreater(admPayload("activity_attempt.sdk_attempt"), admWritten(&umpirespb.ProtoValue_Text{Text: "0"}))
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard orders a text, and only numbers are ordered"},
		{"a Run Event whose guard orders a text under a negation", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Guard = admNot(admGreater(admWritten(&umpirespb.ProtoValue_Text{Text: "0"}), admPayload("activity_attempt.sdk_attempt")))
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard orders a text, and only numbers are ordered"},
		{"a Run Event whose guard compares two names it writes out", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				name := &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: "A"}}}}
				e.Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: name, Right: name}}}
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard compares two enum values it writes out"},
		{"a Run Event whose guard writes out a name a Case binds", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				named := &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Named{Named: &umpirespb.Name{Prefix: "run-", Fixture: true}}}}}
				e.Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: admPayload("detail"), Right: named}}}
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard writes out a name a Case binds; a Run Event's guard reads the event's payload alone"},
		{"a Run Event whose guard reads a path it writes no field of", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: admPayload("activity_attempt[*]")}}}
			})
		}, `evidence temporal.features.nexus.workflow.evidence.started: its guard reads "activity_attempt[*]" of the path activity_attempt[*], and a guard reads a field or oneof<member>`},
		{"a Run Event whose guard negates the payload", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Guard = admNot(&umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}})
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard negates a message, and only a condition is negated"},

		{"a canceled answer outside an activity", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			admCommand(t, admScript(t, r, "handler"), "respond-sync").Instruction = &umpirespb.Command_AttemptCanceled{AttemptCanceled: &umpirespb.Empty{}}
		}, "command respond-sync of script handler cancels an attempt, and script handler is no activity's"},

		// Evidence the Run itself records.
		{"a Run Event of no known kind", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) { e.Kind = 0 })
		}, "evidence temporal.features.nexus.workflow.evidence.started is a Run Event of no known kind"},
		{"a Run Event of an undeclared script", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) { e.Script = "nobody" })
		}, "evidence temporal.features.nexus.workflow.evidence.started: no script nobody"},
		{"a Run Event of a command its script does not have", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) { e.Command = "finish-workflow" })
		}, "evidence temporal.features.nexus.workflow.evidence.started: no command finish-workflow of script controller"},
		{"a Run Event with no key", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) { e.Key = nil })
		}, "evidence temporal.features.nexus.workflow.evidence.started: a Run Event's key is the run's id or a path of its payload"},
		{"a Run Event keyed by an environment binding", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Key = &umpirespb.Operand{Kind: &umpirespb.Operand_Environment{Environment: "temporal.worker.namespace"}}
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: a Run Event's key is the run's id or a path of its payload"},
		{"a Run Event keyed by a path of something other than its payload", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Key = &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "activity_attempt.activity_run_id",
					Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Environment{Environment: "temporal.worker.namespace"}}}}}
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: a Run Event's key is the run's id or a path of its payload"},
		{"a Run Event whose guard reads the run", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) {
				e.Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{
					Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &umpirespb.Empty{}}}}}}
			})
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard reads the run's id; a Run Event's guard reads the event's payload alone"},
		{"a Run Event whose guard is of no kind", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(e *umpirespb.RunEventSource) { e.Guard = &umpirespb.Operand{} })
		}, "evidence temporal.features.nexus.workflow.evidence.started: its guard has an operand of no known kind"},
		{"a Run Event with an operation path of its own", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			runEvent(r, func(*umpirespb.RunEventSource) {})
			r.Evidence[1].Operation = "detail"
		}, "evidence temporal.features.nexus.workflow.evidence.started names a field that keys its operation, and a Run Event's key is its source's"},
		{"a poll of a Run Event", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.Evidence[6].Operation = ""
			r.Evidence[6].From = &umpirespb.Evidence_RunEvent{RunEvent: &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED,
				Script: "controller", Command: "start-workflow", Key: &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &umpirespb.Empty{}}}}}
		}, "command pending-attempts of script controller: evidence temporal.features.nexus.workflow.evidence.pendingAttempts is a Run Event, which a poll does not read"},

		// A required setting names its key and its value, and no key twice, whatever its case.
		{"a required setting with no value", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.RequiredSettings = []*umpirespb.RequiredSetting{{Key: "nexusoperation.enableStandalone"}}
		}, "realization asyncNexus: a required setting names no key or no value"},
		{"a setting required twice", func(_ *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			r.RequiredSettings = []*umpirespb.RequiredSetting{{Key: "nexusoperation.enableStandalone", Value: "true"}, {Key: "nexusoperation.enablestandalone", Value: "true"}}
		}, "realization asyncNexus: it requires setting nexusoperation.enablestandalone twice"},

		{"a result that is a message", func(t *testing.T, _ *umpirespb.Model, r *umpirespb.Realization) {
			c := admCommand(t, admScript(t, r, "workflow"), "finish-workflow")
			c.GetFinish().Result = &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{
				Kind: &umpirespb.ProtoValue_Message{Message: &umpirespb.Proto{Message: "temporal.api.common.v1.Payload"}}}}}
		}, "command finish-workflow of script workflow: a literal operand is a text, a flag, a number, an enum value or a name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := proto.Clone(load(t)).(*umpirespb.Model)
			c.mutate(t, m, m.GetRealizations()[0])
			err := Validate(m)
			require.ErrorContains(t, err, c.want)
			requireLocated(t, err, admRealizationAt)
		})
	}
}

// An earlier problem of a realization hides no later one: admission reports every problem.
func TestEveryProblemOfARealizationIsReported(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	r := m.GetRealizations()[0]
	r.Roles[0].Kind = 0
	r.Correlation.Observation = "elsewhere"
	admCommand(t, admScript(t, r, "controller"), "await-close").After = &umpirespb.After{Commands: []string{"await-close"}}
	err := Validate(m)
	require.ErrorContains(t, err, "role temporal.workflow-service is of no known kind")
	require.ErrorContains(t, err, "the correlation: no observation elsewhere")
	require.ErrorContains(t, err, "command await-close runs after itself")
}

// A realization no one can name, or of a machine the Model does not declare, is still read whole: its
// roles, its correlation, its bindings and its cycles are reported beside what is wrong with its
// envelope. Only what needs the machine, the classes its items name, is left unchecked.
func TestARealizationWithABrokenEnvelopeIsStillReadWhole(t *testing.T) {
	broken := func(t *testing.T, r *umpirespb.Realization) {
		r.Roles[0].Kind = 0
		r.Correlation.Observation = "elsewhere"
		admCommand(t, admScript(t, r, "controller"), "await-close").After = &umpirespb.After{Commands: []string{"await-close"}}
		admCommand(t, admScript(t, r, "handler"), "respond-sync").GetNexusReply().Binds = "completion-authority"
		admScript(t, r, "workflow").Activation = nil
	}
	inside := []string{
		"role temporal.workflow-service is of no known kind",
		"the correlation: no observation elsewhere",
		"script controller: command await-close runs after itself",
		"learned value completion-authority is bound by respond-async of script handler and by respond-sync of script handler",
		"script workflow names no activation",
	}
	for _, c := range []struct {
		name     string
		envelope func(r *umpirespb.Realization)
		want     string
	}{
		{"no name", func(r *umpirespb.Realization) { r.Name = "" }, "a realization has no name"},
		{"no machine", func(r *umpirespb.Realization) { r.Machine = "nope" }, "realization asyncNexus: no machine nope"},
		{"no id", func(r *umpirespb.Realization) { r.Id = "" }, "realization asyncNexus has no id"},
		{"neither a name nor an id", func(r *umpirespb.Realization) { r.Name, r.Id = "", "" }, "a realization has no id"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := proto.Clone(load(t)).(*umpirespb.Model)
			r := m.GetRealizations()[0]
			c.envelope(r)
			broken(t, r)
			err := Validate(m)
			require.ErrorContains(t, err, c.want)
			for _, want := range inside {
				require.ErrorContains(t, err, want)
			}
		})
	}
}

// Evidence that is the Run's record of an attempt says which: the attempt, counted from one, of the
// activity one script runs. A Run records an attempt once it is answered, so what the declaration
// names decides when the evidence reaches a Run, and a declaration that names no attempt of an
// activity is refused where it is written.
func TestTheAttemptARunEventRecordsIsOfAnActivitysScript(t *testing.T) {
	const started = "temporal.features.activity.standalone.system.evidence.statusStarted"
	source := func(t *testing.T, m *umpirespb.Model) *umpirespb.RunEventSource {
		for _, e := range m.GetRealizations()[0].GetEvidence() {
			if e.GetId() == started {
				e.GetRunEvent().Attempt = &umpirespb.AttemptOf{Script: "attempts", Number: 1}
				return e.GetRunEvent()
			}
		}
		require.FailNow(t, "no evidence "+started)
		return nil
	}
	for name, test := range map[string]struct {
		change func(m *umpirespb.Model, source *umpirespb.RunEventSource)
		want   string
	}{
		"the first attempt of the activity's script": {func(*umpirespb.Model, *umpirespb.RunEventSource) {}, ""},
		"an attempt of no script": {func(_ *umpirespb.Model, s *umpirespb.RunEventSource) { s.GetAttempt().Script = "" },
			"evidence " + started + " is the record of an attempt of no script"},
		"an attempt of a script the realization does not declare": {func(_ *umpirespb.Model, s *umpirespb.RunEventSource) { s.GetAttempt().Script = "nobody" },
			"evidence " + started + " is the record of an attempt of script nobody, which the realization does not declare"},
		"an attempt of the controller's script": {func(_ *umpirespb.Model, s *umpirespb.RunEventSource) { s.GetAttempt().Script = "controller" },
			"evidence " + started + " is the record of an attempt of script controller, which no activity activates"},
		"an attempt counted from below one": {func(_ *umpirespb.Model, s *umpirespb.RunEventSource) { s.GetAttempt().Number = 0 },
			"evidence " + started + " is the record of attempt 0 of script attempts; the attempts of an activity are counted from one"},
		"an attempt of a script that starts with no delivery": {func(m *umpirespb.Model, _ *umpirespb.RunEventSource) {
			for _, s := range m.GetRealizations()[0].GetScripts() {
				if s.GetActivity() != nil {
					s.GetActivity().Starts = nil
				}
			}
		}, "evidence " + started + " is the record of an attempt of script attempts, which starts with no delivery: no step of a path is an attempt of it"},
		"an attempt recorded as the completion of a call": {func(_ *umpirespb.Model, s *umpirespb.RunEventSource) {
			s.Kind = umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED
		}, "evidence " + started + " is the record of an attempt and of no diagnostic: a Run records an attempt as what a worker reports of an activation"},
		// What a worker reports of an activation reaches a Run once the activation is answered, whatever
		// the realization says of it: with no attempt named, when that is would be anyone's guess.
		"what a worker reports of an activation, declared the record of no attempt": {func(_ *umpirespb.Model, s *umpirespb.RunEventSource) { s.Attempt = nil },
			"evidence " + started + " is what a worker reports of an activation and is declared the record of no attempt: " +
				"a Run records it once the attempt is answered, and the realization says which attempt that is"},
	} {
		t.Run(name, func(t *testing.T) {
			m, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone.json"))
			require.NoError(t, err)
			test.change(m, source(t, m))
			err = Validate(m)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
			requireLocated(t, err, "model/temporal/features/activity/standalone/system/Realization.scala:")
		})
	}
}

// The Run's own record of a command is closed by that command: its record is the read. The held
// race declares the admission its release observes exhaustive, closed by the release, which is a
// performance, since a Case that does not release has no record whose silence could say anything.
// Any other command does not read the record, and closes nothing.
func TestTheRunsOwnRecordIsClosedByTheCommandItRecords(t *testing.T) {
	race := func(t *testing.T) (*umpirespb.Model, *umpirespb.Realization) {
		m, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone-race.json"))
		require.NoError(t, err)
		for _, r := range m.GetRealizations() {
			if r.GetName() == "heldDelivery" {
				return m, r
			}
		}
		t.Fatal("heldDelivery realization missing")
		return nil, nil
	}
	const admitted = "temporal.features.activity.standalone.system.evidence.attemptAdmitted"
	m, r := race(t)
	release := admCommand(t, admScript(t, r, "controller"), "release-dispatch")
	require.Equal(t, []string{admitted}, release.GetCloses())
	require.NoError(t, Validate(m))

	m, r = race(t)
	admCommand(t, admScript(t, r, "controller"), "release-dispatch").Closes = nil
	admCommand(t, admScript(t, r, "controller"), "hold-dispatch").Closes = []string{admitted}
	require.ErrorContains(t, Validate(m), "command hold-dispatch of script controller closes evidence "+admitted+" and does not read it")
}
