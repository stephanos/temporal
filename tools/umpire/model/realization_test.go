package model

// What a reader rejects of a realization before any lowering (model/scalav2/SEMANTICS.md, Admission):
// each case is one mutation of the Nexus caller realization the lifter emitted, reported at the
// position the lifter recorded for the declaration concerned.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/proto"
)

const admRealizationAt = "model/temporal/nexuscaller/Realization.scala:"

// admScript is a script of the Nexus caller realization, by id.
func admScript(t *testing.T, r *modelirspb.Realization, id string) *modelirspb.Script {
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
func admCommand(t *testing.T, s *modelirspb.Script, id string) *modelirspb.Command {
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
func admClass(t *testing.T, r *modelirspb.Realization) *modelirspb.ActionClass {
	t.Helper()
	return proto.CloneOf(admScript(t, r, "handler").GetItems()[0].GetPerforms()[0].GetStep())
}

// A fact that steps of several classes record, or one class records twice on a path, may have a kind
// of evidence for each: kinds that name the steps they confirm may record one fact, beside at most one
// kind that names none.
func TestKindsThatNameTheirStepsMayRecordOneFact(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	r := m.GetRealizations()[0]
	// The fact is one no exhaustive kind records: an exhaustive kind is the one kind of its fact.
	unclosed(r)
	r.Evidence[2].Records, r.Evidence[3].Records = r.Evidence[1].GetRecords(), r.Evidence[1].GetRecords()
	r.Evidence[2].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 1}}
	r.Evidence[3].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 2}}
	require.NoError(t, Validate(m))
}

func TestTheLiftedRealizationIsAdmitted(t *testing.T) {
	m := load(t)
	require.Len(t, m.GetRealizations(), 1)
	require.Equal(t, "nexusProtocol", m.GetRealizations()[0].GetMachine())
}

// unclosed takes every exhaustive declaration and closing read out of a realization, so that a case
// states the whole of the one it is about.
func unclosed(r *modelirspb.Realization) {
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

func admWritten(kind any) *modelirspb.Operand {
	value := &modelirspb.ProtoValue{}
	switch k := kind.(type) {
	case *modelirspb.ProtoValue_Text:
		value.Kind = k
	case *modelirspb.ProtoValue_Number:
		value.Kind = k
	default:
	}
	return &modelirspb.Operand{Kind: &modelirspb.Operand_Literal{Literal: value}}
}

// admPayload reads a path of the value a poll is looking at, or of a Run Event's payload.
func admPayload(path string) *modelirspb.Operand {
	return &modelirspb.Operand{Kind: &modelirspb.Operand_Path{Path: &modelirspb.PathOf{Path: path,
		Of: &modelirspb.Operand{Kind: &modelirspb.Operand_Projected{Projected: &modelirspb.Empty{}}}}}}
}

func admGreater(left, right *modelirspb.Operand) *modelirspb.Operand {
	return &modelirspb.Operand{Kind: &modelirspb.Operand_Greater{Greater: &modelirspb.Greater{Left: left, Right: right}}}
}

func admNot(of *modelirspb.Operand) *modelirspb.Operand {
	return &modelirspb.Operand{Kind: &modelirspb.Operand_Not{Not: &modelirspb.Not{Of: of}}}
}

// runEvent makes the started kind of the Nexus caller realization the Run's own record of the start
// call's completion, keyed by the run, and then changes that source.
func runEvent(r *modelirspb.Realization, change func(*modelirspb.RunEventSource)) {
	unclosed(r)
	source := &modelirspb.RunEventSource{Kind: modelirspb.RunEventSource_KIND_INSTRUCTION_COMPLETED, Script: "controller", Command: "start-workflow",
		Key: &modelirspb.Operand{Kind: &modelirspb.Operand_Run{Run: &modelirspb.Empty{}}}}
	change(source)
	r.Evidence[1].Operation = ""
	r.Evidence[1].From = &modelirspb.Evidence_RunEvent{RunEvent: source}
}

// A Run Event source that names its command, keys by the run or a path of the payload and guards on
// the payload alone is admitted. What a worker reports of an activation is admitted as the record of
// an attempt alone (TestTheAttemptARunEventRecordsIsOfAnActivitysScript).
func TestARunEventSourceIsAdmitted(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	projected := &modelirspb.Operand{Kind: &modelirspb.Operand_Projected{Projected: &modelirspb.Empty{}}}
	path := func(p string) *modelirspb.Operand {
		return &modelirspb.Operand{Kind: &modelirspb.Operand_Path{Path: &modelirspb.PathOf{Of: projected, Path: p}}}
	}
	present := func(o *modelirspb.Operand) *modelirspb.Operand {
		return &modelirspb.Operand{Kind: &modelirspb.Operand_Present{Present: &modelirspb.Present{Of: o}}}
	}
	runEvent(m.GetRealizations()[0], func(e *modelirspb.RunEventSource) {
		e.Key = path("activity_attempt.activity_run_id")
		e.Guard = &modelirspb.Operand{Kind: &modelirspb.Operand_All{All: &modelirspb.All{Operands: []*modelirspb.Operand{
			present(path("activity_attempt")),
			admGreater(path("activity_attempt.sdk_attempt"), admWritten(&modelirspb.ProtoValue_Number{Number: 0})),
			admNot(&modelirspb.Operand{Kind: &modelirspb.Operand_Equal{Equal: &modelirspb.Equal{Left: path("activity_attempt.delivery_id"),
				Right: admWritten(&modelirspb.ProtoValue_Text{Text: ""})}}})}}}}
	})
	require.NoError(t, Validate(m))
}

func TestARealizationIsAdmittedBeforeItIsLowered(t *testing.T) {
	rpc := func(id string, after ...string) *modelirspb.Item {
		c := &modelirspb.Command{Id: id, Instruction: &modelirspb.Command_Rpc{Rpc: &modelirspb.Rpc{
			Role: "temporal.workflow-service", Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeNamespace"}}}
		if after != nil {
			c.After = &modelirspb.After{Commands: after}
		}
		return &modelirspb.Item{Command: c}
	}
	cases := []struct {
		name   string
		mutate func(t *testing.T, m *modelirspb.Model, r *modelirspb.Realization)
		want   string
	}{
		// A control that holds what a step dispatches names a class its machine binds, and a
		// task-queue role.
		{"a control that holds the dispatch of no class", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Controls = append(r.Controls, &modelirspb.Control{Id: "held", Role: "temporal.task-queue",
				Kind: &modelirspb.Control_HoldDispatched{HoldDispatched: &modelirspb.HoldDispatched{}}})
		}, "realization asyncNexus: control held holds what a step of no class dispatches"},
		{"a control that holds the dispatch of a class the machine does not bind", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Controls = append(r.Controls, &modelirspb.Control{Id: "held", Role: "temporal.task-queue",
				Kind: &modelirspb.Control_HoldDispatched{HoldDispatched: &modelirspb.HoldDispatched{Step: &modelirspb.ActionClass{Action: "temporal.worker.Worker$package$.workerResume"}}}})
		}, "realization asyncNexus: control held: nexusProtocol binds no action temporal.worker.Worker$package$.workerResume"},
		{"a control that holds the deliveries of no task queue", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			step := admScript(t, r, "handler").GetItems()[0].GetPerforms()[0].GetStep()
			r.Controls = append(r.Controls, &modelirspb.Control{Id: "held", Role: "temporal.workflow-service",
				Kind: &modelirspb.Control_HoldDispatched{HoldDispatched: &modelirspb.HoldDispatched{Step: step}}})
		}, "realization asyncNexus: control held: role temporal.workflow-service is"},
		{"a control that holds a dispatch on no role", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			step := admScript(t, r, "handler").GetItems()[0].GetPerforms()[0].GetStep()
			r.Controls = append(r.Controls, &modelirspb.Control{Id: "held",
				Kind: &modelirspb.Control_HoldDispatched{HoldDispatched: &modelirspb.HoldDispatched{Step: step}}})
		}, "realization asyncNexus: control held holds deliveries and names no task-queue role"},
		// Absent and empty ids.
		{"a realization with no name", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Name = "" },
			"a realization has no name"},
		{"a realization of no machine", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Machine = "nope" },
			"realization asyncNexus: no machine nope"},
		{"a role with no id", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Roles[1].Id = "" },
			"realization asyncNexus: a role has no id"},
		{"a learned value with no id", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Learned[0].Id = "" },
			"realization asyncNexus: a learned value has no id"},
		{"a command with no id", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").Id = ""
		}, "realization asyncNexus: a command of script controller has no id"},
		{"evidence that names no recorded kind", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Evidence[1].Records = "" },
			"evidence temporal.nexus.caller.evidence.started names no recorded kind"},
		{"evidence with no operation key", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Evidence[1].Operation = "" },
			"evidence temporal.nexus.caller.evidence.started names no field that keys its operation"},
		{"evidence read from nowhere", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Evidence[1].From = nil },
			"evidence temporal.nexus.caller.evidence.started is recorded nowhere"},
		{"evidence of no commitment", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Commitment = modelirspb.Evidence_COMMITMENT_UNSPECIFIED
		}, "evidence temporal.nexus.caller.evidence.started is of no known commitment"},
		{"a role of no kind", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Roles[0].Kind = 0 },
			"role temporal.workflow-service is of no known kind"},
		{"a command with no instruction", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").Instruction = nil
		}, "command start-workflow of script controller names no instruction"},
		{"a script with no activation", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admScript(t, r, "workflow").Activation = nil
		}, "script workflow names no activation"},
		{"no correlation", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Correlation = nil },
			"realization asyncNexus declares no correlation"},

		// Duplicate declarations and bindings.
		{"two realizations of one name", func(_ *testing.T, m *modelirspb.Model, r *modelirspb.Realization) {
			m.Realizations = append(m.Realizations, proto.Clone(r).(*modelirspb.Realization))
		}, "two realizations named asyncNexus"},
		{"two roles of one id", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) { r.Roles[1].Id = r.Roles[0].GetId() },
			"two roles with id temporal.workflow-service of realization asyncNexus"},
		{"two commands of one id in a script", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").Id = "history"
		}, "two commands with id history of script controller of realization asyncNexus"},
		{"two kinds of evidence for one recorded kind", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[2].Records = r.Evidence[1].GetRecords()
		}, "two kinds of evidence recording nexusOperationStarted of realization asyncNexus"},
		{"evidence that confirms a step counted from below one", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Step: admClass(t, r)}}
		}, "evidence temporal.nexus.caller.evidence.started confirms step 0 of class handlerReply-async; the steps of a class on a path are counted from one"},
		{"evidence that confirms one step twice", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 2}, {Step: admClass(t, r), Occurrence: 2}}
		}, "evidence temporal.nexus.caller.evidence.started confirms step 2 of class handlerReply-async twice"},
		{"two kinds of evidence that confirm one step", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 1}}
			r.Evidence[2].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "evidence temporal.nexus.caller.evidence.started and temporal.nexus.caller.evidence.completed both confirm step 1 of class handlerReply-async; one kind confirms a step"},
		{"evidence that confirms a step of no class", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Occurrence: 1}}
		}, "evidence temporal.nexus.caller.evidence.started confirms a step of no class"},
		{"evidence that confirms a class written with too few inputs", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Step: &modelirspb.ActionClass{Action: admClass(t, r).GetAction()}, Occurrence: 1}}
		}, "realization asyncNexus: evidence temporal.nexus.caller.evidence.started: temporal.nexuscaller.Model$package$.handlerReply takes 1 inputs, not 0"},
		{"two kinds of evidence for one recorded kind, one of which names its steps", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[2].Records, r.Evidence[3].Records = r.Evidence[1].GetRecords(), r.Evidence[1].GetRecords()
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "two kinds of evidence recording nexusOperationStarted of realization asyncNexus"},
		{"an exhaustive kind that names the steps it confirms", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[1].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "evidence temporal.nexus.caller.evidence.started is exhaustive and names the steps it confirms; an exhaustive kind reports every step that records its fact"},
		{"a fact an exhaustive kind records, recorded by a second kind that names its steps", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Records = r.Evidence[1].GetRecords()
			r.Evidence[6].Confirms = []*modelirspb.Taking{{Step: admClass(t, r), Occurrence: 1}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts records nexusOperationStarted, every occurrence of which the exhaustive temporal.nexus.caller.evidence.started reports"},
		{"a learned value bound twice", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "handler"), "respond-sync").GetNexusReply().Binds = "completion-authority"
		}, "learned value completion-authority is bound by respond-async of script handler and by respond-sync of script handler; a learned value is bound once"},
		{"a class performed twice", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			performs := admScript(t, r, "handler").GetItems()[0].GetPerforms()
			performs[1].Step = performs[0].GetStep()
		}, "class handlerReply-async is performed by respond-async of script handler and by respond-sync of script handler; a class is performed once"},

		// References that name nothing, or the wrong kind of thing.
		{"a call on an undeclared role", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").GetRpc().Role = "nobody"
		}, "command start-workflow of script controller: no role nobody"},
		{"a call on a worker", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "start-workflow").GetRpc().Role = "temporal.worker"
		}, "command start-workflow of script controller: role temporal.worker is a worker, not an endpoint"},
		{"a worker script whose task queue is an endpoint", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admScript(t, r, "workflow").GetWorkflow().TaskQueue = "temporal.nexus-endpoint"
		}, "script workflow: role temporal.nexus-endpoint is an endpoint, not a task queue"},
		{"a wait for an undeclared learned value", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			c := admCommand(t, admScript(t, r, "controller"), "await-completion-authority")
			c.Instruction = &modelirspb.Command_AwaitLearned{AwaitLearned: "nothing"}
		}, "command await-completion-authority of script controller: no learned value nothing"},
		{"a learned value read and never bound", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			for _, p := range admScript(t, r, "handler").GetItems()[0].GetPerforms() {
				p.GetCommand().GetNexusReply().Binds = ""
			}
		}, "learned value completion-authority is read and no command binds it"},
		{"a text read as a handle", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Learned[0].Kind = modelirspb.Learned_KIND_TEXT
		}, "learned value completion-authority is a text, not a handle"},
		{"a handle read as a value", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			rpc := admCommand(t, admScript(t, r, "controller"), "await-close").GetRpc()
			rpc.Assign[1].Value = &modelirspb.Operand{Kind: &modelirspb.Operand_LearnedValue{LearnedValue: "completion-authority"}}
		}, "command await-close of script controller: learned value completion-authority is a handle, not a text"},
		{"a poll of evidence that is no read", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-scheduled").GetPoll().Evidence = "temporal.nexus.caller.evidence.started"
		}, "command await-scheduled of script controller: evidence temporal.nexus.caller.evidence.started is a history event, which a poll does not read"},
		{"a read into an undeclared observation", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			targets := admCommand(t, admScript(t, r, "controller"), "history").GetRpc().GetReads()[0].GetTargets()
			targets[0].Target = &modelirspb.Target_Observe{Observe: "unseen"}
		}, "command history of script controller: no observation unseen"},
		{"a wait for a command of another script", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			c := admCommand(t, admScript(t, r, "workflow"), "await-nexus-operation")
			c.Instruction = &modelirspb.Command_AwaitCommand{AwaitCommand: "start-workflow"}
		}, "command await-nexus-operation of script workflow: no command start-workflow of script workflow"},
		{"a class the machine does not bind", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admScript(t, r, "controller").GetItems()[0].GetPerforms()[0].Step = &modelirspb.ActionClass{Action: "temporal.nexuscaller.Model$package$.nope"}
		}, "no action temporal.nexuscaller.Model$package$.nope"},
		{"a release of an undeclared control", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").Instruction = &modelirspb.Command_Release{Release: "gate"}
		}, "command await-close of script controller: no control gate"},
		{"a control that holds an undeclared channel", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Controls = append(r.Controls, &modelirspb.Control{Id: "gate", Kind: &modelirspb.Control_HoldDelivery{HoldDelivery: "no.channel"}})
		}, "control gate: no channel no.channel"},

		{"a learned value read by a command that runs regardless", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Learned = append(r.Learned, &modelirspb.Learned{Id: "run", Kind: modelirspb.Learned_KIND_TEXT})
			s := admScript(t, r, "controller")
			start := admCommand(t, s, "start-workflow").GetRpc()
			start.Reads = append(start.Reads, &modelirspb.ResponseRead{Path: "run_id", Cardinality: modelirspb.ResponseRead_CARDINALITY_ONE,
				Targets: []*modelirspb.Target{{Target: &modelirspb.Target_Bind{Bind: "run"}}}})
			closing := admCommand(t, s, "await-close")
			closing.Regardless = true
			closing.GetRpc().Assign = append(closing.GetRpc().Assign, &modelirspb.Assignment{Target: "execution.run_id",
				Value: &modelirspb.Operand{Kind: &modelirspb.Operand_LearnedValue{LearnedValue: "run"}}})
		}, "command await-close of script controller runs whatever became of the commands before it, and reads learned value run, which is read only once it is bound"},

		// Crossed correlations.
		{"runs and operations keyed by one field", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Correlation.Operation = r.GetCorrelation().GetRun()
		}, "the correlation keys its runs and its operations by one field, temporal.nexus.caller.scope.run"},
		{"evidence lifted into an observation the correlation does not read", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			targets := admCommand(t, admScript(t, r, "controller"), "history").GetRpc().GetReads()[0].GetTargets()
			targets[1].Target = &modelirspb.Target_Lift{Lift: "history-event"}
		}, "command history of script controller lifts evidence into history-event, and the correlation reads correlated-evidence"},
		{"a value observed into the correlation's observation", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			targets := admCommand(t, admScript(t, r, "controller"), "history").GetRpc().GetReads()[0].GetTargets()
			targets[0].Target = &modelirspb.Target_Observe{Observe: "correlated-evidence"}
		}, "command history of script controller observes a value into correlated-evidence, which holds the correlation's evidence"},
		{"a correlation over an undeclared observation", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Correlation.Observation = "elsewhere"
		}, "the correlation: no observation elsewhere"},

		// Dependency cycles.
		{"a command that runs after itself", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").After = &modelirspb.After{Commands: []string{"await-close"}}
		}, "script controller: command await-close runs after itself"},
		{"two commands that each run after the other", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			s := admScript(t, r, "controller")
			s.Items = append(s.Items, rpc("left", "right"), rpc("right", "left"))
		}, "script controller: command left runs after itself, through right"},
		{"a command that runs after one of no script", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").After = &modelirspb.After{Commands: []string{"finish-workflow"}}
		}, "command await-close of script controller: no command finish-workflow of script controller"},

		// Shapes the IR does not have.
		{"an item that is nothing", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			s := admScript(t, r, "controller")
			s.Items = append(s.Items, &modelirspb.Item{})
		}, "script controller has an item that is neither a command nor the place steps are performed"},
		{"an item that is both", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			item := admScript(t, r, "controller").GetItems()[0]
			item.Command = rpc("extra").GetCommand()
		}, "script controller has an item that is both a command and the place steps are performed"},
		{"an operand of no kind", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").GetRpc().GetAssign()[0].Value = &modelirspb.Operand{}
		}, "command await-close of script controller: an operand of no known kind"},
		{"a message with no name", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "workflow"), "start-nexus-operation").GetWorkflowCommand().GetCommand().Message = ""
		}, "command start-nexus-operation of script workflow: a message with no name"},
		{"a message that sets a field twice", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			c := admCommand(t, admScript(t, r, "workflow"), "start-nexus-operation").GetWorkflowCommand().GetCommand()
			c.Fields = append(c.Fields, c.GetFields()[0])
		}, "command start-nexus-operation of script workflow: temporal.api.command.v1.Command sets command_type twice"},
		{"a deadline below zero", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "await-close").TimeoutMs = -1
		}, "command await-close of script controller has a deadline of -1 milliseconds"},
		// Evidence fields and the identities they name.
		{"an evidence field with no id", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Fields = []*modelirspb.EvidenceField{{Path: "attempt"}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts has a field with no id"},
		{"an evidence field declared twice", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Fields = []*modelirspb.EvidenceField{{Id: "attempt", Path: "attempt"}, {Id: "attempt", Path: "attempt"}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts declares field attempt twice"},
		{"an evidence field read from nowhere", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Fields = []*modelirspb.EvidenceField{{Id: "attempt"}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts: field attempt names no path"},
		{"an evidence field of no known role", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Fields = []*modelirspb.EvidenceField{{Id: "attempt", Path: "attempt", Role: 9}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts: field attempt names an identity of no known role"},
		{"a role on a field the kind does not retain", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Fields = []*modelirspb.EvidenceField{{Id: "attempt", Path: "attempt", Role: modelirspb.EvidenceField_ROLE_ATTEMPT, Redacted: true}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts: field attempt names the attempt and is redacted; an identity is read from a field the evidence retains"},
		{"two fields of one role", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Fields = []*modelirspb.EvidenceField{{Id: "attempt", Path: "attempt", Role: modelirspb.EvidenceField_ROLE_ATTEMPT},
				{Id: "again", Path: "attempt", Role: modelirspb.EvidenceField_ROLE_ATTEMPT}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts: fields attempt and again both name the attempt; one field of a kind names an identity"},
		{"evidence read from one message at no path", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].From = &modelirspb.Evidence_Single{Single: &modelirspb.ReadSource{Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"}}
		}, "evidence temporal.nexus.caller.evidence.pendingAttempts reads no method or no path"},

		// Exhaustive kinds and their closing reads.
		{"an exhaustive kind no command closes", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			unclosed(r)
			r.Evidence[1].Exhaustive = true
		}, "evidence temporal.nexus.caller.evidence.started is exhaustive and no command closes it"},
		{"a closing read of an undeclared kind", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"nothing"}
		}, "command history of script controller closes evidence nothing, which is not declared"},
		{"a closing read of a kind that is not exhaustive", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			unclosed(r)
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"temporal.nexus.caller.evidence.started"}
		}, "command history of script controller closes evidence temporal.nexus.caller.evidence.started, which is not exhaustive"},
		{"a kind closed by a command that does not read it", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			unclosed(r)
			r.Evidence[1].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "await-close").Closes = []string{"temporal.nexus.caller.evidence.started"}
		}, "command await-close of script controller closes evidence temporal.nexus.caller.evidence.started and does not read it: a history kind is closed by the read that lifts it, the Run's own record of a command by that command, and any other by a poll of it"},
		{"a polled kind closed by a history read", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			unclosed(r)
			r.Evidence[6].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"temporal.nexus.caller.evidence.pendingAttempts"}
		}, "command history of script controller closes evidence temporal.nexus.caller.evidence.pendingAttempts and does not read it"},
		{"a kind closed twice", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			unclosed(r)
			r.Evidence[1].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "history").Closes = []string{"temporal.nexus.caller.evidence.started", "temporal.nexus.caller.evidence.started"}
		}, "evidence temporal.nexus.caller.evidence.started is closed by history of script controller and by history of script controller; an exhaustive kind has one closing read"},
		{"a closing read only some Cases carry", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			unclosed(r)
			r.Evidence[6].Exhaustive = true
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").Closes = []string{"temporal.nexus.caller.evidence.pendingAttempts"}
		}, "command pending-attempts of script controller closes evidence temporal.nexus.caller.evidence.pendingAttempts and is not a command every Case carries"},

		// An activity's attempts.
		{"an attempt failure outside an activity", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "handler"), "respond-sync").Instruction = &modelirspb.Command_AttemptFailure{AttemptFailure: &modelirspb.AttemptFailure{
				Failure: &modelirspb.Proto{Message: "temporal.api.failure.v1.Failure"}}}
		}, "command respond-sync of script handler fails an attempt, and script handler is no activity's"},
		{"an attempt failure with no failure", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			s := admScript(t, r, "handler")
			s.Activation = &modelirspb.Script_Activity{Activity: &modelirspb.ActivityActivation{ActivityType: &modelirspb.Name{Prefix: "activity"},
				Worker: "temporal.worker", TaskQueue: "temporal.handler-task-queue"}}
			admCommand(t, s, "respond-sync").Instruction = &modelirspb.Command_AttemptFailure{AttemptFailure: &modelirspb.AttemptFailure{}}
		}, "command respond-sync of script handler: a message with no name"},
		{"a delivery that a command also performs", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			s := admScript(t, r, "handler")
			s.Activation = &modelirspb.Script_Activity{Activity: &modelirspb.ActivityActivation{ActivityType: &modelirspb.Name{Prefix: "activity"},
				Worker: "temporal.worker", TaskQueue: "temporal.handler-task-queue", Starts: []*modelirspb.ActionClass{s.GetItems()[0].GetPerforms()[0].GetStep()}}}
		}, "class handlerReply-async is performed by the activation of script handler and by respond-async of script handler; a class is performed once"},
		{"a delivery of a class the machine does not bind", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			s := admScript(t, r, "handler")
			s.Activation = &modelirspb.Script_Activity{Activity: &modelirspb.ActivityActivation{ActivityType: &modelirspb.Name{Prefix: "activity"},
				Worker: "temporal.worker", TaskQueue: "temporal.handler-task-queue",
				Starts: []*modelirspb.ActionClass{{Action: "temporal.nexuscaller.Model$package$.nope"}}}}
		}, "no action temporal.nexuscaller.Model$package$.nope"},

		{"a conjunction of nothing", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &modelirspb.Operand{Kind: &modelirspb.Operand_All{All: &modelirspb.All{}}}
		}, "command pending-attempts of script controller: a conjunction of no operand"},
		{"a conjunction over an operand of no kind", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &modelirspb.Operand{Kind: &modelirspb.Operand_All{
				All: &modelirspb.All{Operands: []*modelirspb.Operand{{}}}}}
		}, "command pending-attempts of script controller: an operand of no known kind"},
		// What an operand computes is of the type its place takes.
		{"an order of a text", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admGreater(admPayload("attempt"), admWritten(&modelirspb.ProtoValue_Text{Text: "one"}))
		}, "command pending-attempts of script controller: orders a text, and only numbers are ordered"},
		{"an order of a condition", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admGreater(admNot(admPayload("attempt")), admWritten(&modelirspb.ProtoValue_Number{Number: 0}))
		}, "command pending-attempts of script controller: orders a condition, and only numbers are ordered"},
		{"a negation of a number", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admNot(admWritten(&modelirspb.ProtoValue_Number{Number: 1}))
		}, "command pending-attempts of script controller: negates a number, and only a condition is negated"},
		{"a negation of nothing", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admNot(nil)
		}, "command pending-attempts of script controller: an operand of no known kind"},
		{"a conjunction over a text", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &modelirspb.Operand{Kind: &modelirspb.Operand_All{
				All: &modelirspb.All{Operands: []*modelirspb.Operand{admWritten(&modelirspb.ProtoValue_Text{Text: "yes"})}}}}
		}, "command pending-attempts of script controller: joins a text, and only conditions are joined"},
		{"a comparison of a text with a number", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &modelirspb.Operand{Kind: &modelirspb.Operand_Equal{Equal: &modelirspb.Equal{
				Left: admWritten(&modelirspb.ProtoValue_Text{Text: "1"}), Right: admWritten(&modelirspb.ProtoValue_Number{Number: 1})}}}
		}, "command pending-attempts of script controller: compares a text with a number"},
		{"a comparison of a value that is no flag, number, text or enum value", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = &modelirspb.Operand{Kind: &modelirspb.Operand_Equal{Equal: &modelirspb.Equal{
				Left:  admPayload("attempt"),
				Right: &modelirspb.Operand{Kind: &modelirspb.Operand_Literal{Literal: &modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_Utf8{Utf8: "one"}}}}}}}
		}, "command pending-attempts of script controller: compares a value that is no text, flag, integer or enum value"},
		{"a poll until a number", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "controller"), "pending-attempts").GetPoll().Until = admWritten(&modelirspb.ProtoValue_Number{Number: 1})
		}, "command pending-attempts of script controller polls until a number, and a poll's condition is a condition"},
		{"a Run Event whose guard is a number", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) { e.Guard = admWritten(&modelirspb.ProtoValue_Number{Number: 1}) })
		}, "evidence temporal.nexus.caller.evidence.started: its guard is a number, and a guard is a condition"},
		{"a Run Event whose guard orders a text", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Guard = admGreater(admPayload("activity_attempt.sdk_attempt"), admWritten(&modelirspb.ProtoValue_Text{Text: "0"}))
			})
		}, "evidence temporal.nexus.caller.evidence.started: its guard orders a text, and only numbers are ordered"},
		{"a Run Event whose guard orders a text under a negation", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Guard = admNot(admGreater(admWritten(&modelirspb.ProtoValue_Text{Text: "0"}), admPayload("activity_attempt.sdk_attempt")))
			})
		}, "evidence temporal.nexus.caller.evidence.started: its guard orders a text, and only numbers are ordered"},
		{"a Run Event whose guard compares two names it writes out", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				name := &modelirspb.Operand{Kind: &modelirspb.Operand_Literal{Literal: &modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_EnumName{EnumName: "A"}}}}
				e.Guard = &modelirspb.Operand{Kind: &modelirspb.Operand_Equal{Equal: &modelirspb.Equal{Left: name, Right: name}}}
			})
		}, "evidence temporal.nexus.caller.evidence.started: its guard compares two enum values it writes out"},
		{"a Run Event whose guard writes out a name a Case binds", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				named := &modelirspb.Operand{Kind: &modelirspb.Operand_Literal{Literal: &modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_Named{Named: &modelirspb.Name{Prefix: "run-", Fixture: true}}}}}
				e.Guard = &modelirspb.Operand{Kind: &modelirspb.Operand_Equal{Equal: &modelirspb.Equal{Left: admPayload("detail"), Right: named}}}
			})
		}, "evidence temporal.nexus.caller.evidence.started: its guard writes out a name a Case binds; a Run Event's guard reads the event's payload alone"},
		{"a Run Event whose guard reads a path it writes no field of", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Guard = &modelirspb.Operand{Kind: &modelirspb.Operand_Present{Present: &modelirspb.Present{Of: admPayload("activity_attempt[*]")}}}
			})
		}, `evidence temporal.nexus.caller.evidence.started: its guard reads "activity_attempt[*]" of the path activity_attempt[*], and a guard reads a field or oneof<member>`},
		{"a Run Event whose guard negates the payload", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Guard = admNot(&modelirspb.Operand{Kind: &modelirspb.Operand_Projected{Projected: &modelirspb.Empty{}}})
			})
		}, "evidence temporal.nexus.caller.evidence.started: its guard negates a message, and only a condition is negated"},

		{"a canceled answer outside an activity", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			admCommand(t, admScript(t, r, "handler"), "respond-sync").Instruction = &modelirspb.Command_AttemptCanceled{AttemptCanceled: &modelirspb.Empty{}}
		}, "command respond-sync of script handler cancels an attempt, and script handler is no activity's"},

		// Evidence the Run itself records.
		{"a Run Event of no known kind", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) { e.Kind = 0 })
		}, "evidence temporal.nexus.caller.evidence.started is a Run Event of no known kind"},
		{"a Run Event of an undeclared script", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) { e.Script = "nobody" })
		}, "evidence temporal.nexus.caller.evidence.started: no script nobody"},
		{"a Run Event of a command its script does not have", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) { e.Command = "finish-workflow" })
		}, "evidence temporal.nexus.caller.evidence.started: no command finish-workflow of script controller"},
		{"a Run Event with no key", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) { e.Key = nil })
		}, "evidence temporal.nexus.caller.evidence.started: a Run Event's key is the run's id or a path of its payload"},
		{"a Run Event keyed by an environment binding", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Key = &modelirspb.Operand{Kind: &modelirspb.Operand_Environment{Environment: "temporal.worker.namespace"}}
			})
		}, "evidence temporal.nexus.caller.evidence.started: a Run Event's key is the run's id or a path of its payload"},
		{"a Run Event keyed by a path of something other than its payload", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Key = &modelirspb.Operand{Kind: &modelirspb.Operand_Path{Path: &modelirspb.PathOf{Path: "activity_attempt.activity_run_id",
					Of: &modelirspb.Operand{Kind: &modelirspb.Operand_Environment{Environment: "temporal.worker.namespace"}}}}}
			})
		}, "evidence temporal.nexus.caller.evidence.started: a Run Event's key is the run's id or a path of its payload"},
		{"a Run Event whose guard reads the run", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) {
				e.Guard = &modelirspb.Operand{Kind: &modelirspb.Operand_Present{Present: &modelirspb.Present{
					Of: &modelirspb.Operand{Kind: &modelirspb.Operand_Run{Run: &modelirspb.Empty{}}}}}}
			})
		}, "evidence temporal.nexus.caller.evidence.started: its guard reads the run's id; a Run Event's guard reads the event's payload alone"},
		{"a Run Event whose guard is of no kind", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(e *modelirspb.RunEventSource) { e.Guard = &modelirspb.Operand{} })
		}, "evidence temporal.nexus.caller.evidence.started: its guard has an operand of no known kind"},
		{"a Run Event with an operation path of its own", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			runEvent(r, func(*modelirspb.RunEventSource) {})
			r.Evidence[1].Operation = "detail"
		}, "evidence temporal.nexus.caller.evidence.started names a field that keys its operation, and a Run Event's key is its source's"},
		{"a poll of a Run Event", func(_ *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			r.Evidence[6].Operation = ""
			r.Evidence[6].From = &modelirspb.Evidence_RunEvent{RunEvent: &modelirspb.RunEventSource{Kind: modelirspb.RunEventSource_KIND_INSTRUCTION_COMPLETED,
				Script: "controller", Command: "start-workflow", Key: &modelirspb.Operand{Kind: &modelirspb.Operand_Run{Run: &modelirspb.Empty{}}}}}
		}, "command pending-attempts of script controller: evidence temporal.nexus.caller.evidence.pendingAttempts is a Run Event, which a poll does not read"},

		{"a result that is a message", func(t *testing.T, _ *modelirspb.Model, r *modelirspb.Realization) {
			c := admCommand(t, admScript(t, r, "workflow"), "finish-workflow")
			c.GetFinish().Result = &modelirspb.Operand{Kind: &modelirspb.Operand_Literal{Literal: &modelirspb.ProtoValue{
				Kind: &modelirspb.ProtoValue_Message{Message: &modelirspb.Proto{Message: "temporal.api.common.v1.Payload"}}}}}
		}, "command finish-workflow of script workflow: a literal operand is a text, a flag, a number, an enum value or a name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := proto.Clone(load(t)).(*modelirspb.Model)
			c.mutate(t, m, m.GetRealizations()[0])
			err := Validate(m)
			require.ErrorContains(t, err, c.want)
			require.ErrorContains(t, err, admRealizationAt, "the error is located")
		})
	}
}

// An earlier problem of a realization hides no later one: admission reports every problem.
func TestEveryProblemOfARealizationIsReported(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	r := m.GetRealizations()[0]
	r.Roles[0].Kind = 0
	r.Correlation.Observation = "elsewhere"
	admCommand(t, admScript(t, r, "controller"), "await-close").After = &modelirspb.After{Commands: []string{"await-close"}}
	err := Validate(m)
	require.ErrorContains(t, err, "role temporal.workflow-service is of no known kind")
	require.ErrorContains(t, err, "the correlation: no observation elsewhere")
	require.ErrorContains(t, err, "command await-close runs after itself")
}

// A realization no one can name, or of a machine the Model does not declare, is still read whole: its
// roles, its correlation, its bindings and its cycles are reported beside what is wrong with its
// envelope. Only what needs the machine, the classes its items name, is left unchecked.
func TestARealizationWithABrokenEnvelopeIsStillReadWhole(t *testing.T) {
	broken := func(t *testing.T, r *modelirspb.Realization) {
		r.Roles[0].Kind = 0
		r.Correlation.Observation = "elsewhere"
		admCommand(t, admScript(t, r, "controller"), "await-close").After = &modelirspb.After{Commands: []string{"await-close"}}
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
		envelope func(r *modelirspb.Realization)
		want     string
	}{
		{"no name", func(r *modelirspb.Realization) { r.Name = "" }, "a realization has no name"},
		{"no machine", func(r *modelirspb.Realization) { r.Machine = "nope" }, "realization asyncNexus: no machine nope"},
		{"no id", func(r *modelirspb.Realization) { r.Id = "" }, "realization asyncNexus has no id"},
		{"neither a name nor an id", func(r *modelirspb.Realization) { r.Name, r.Id = "", "" }, "a realization has no id"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := proto.Clone(load(t)).(*modelirspb.Model)
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

// A realization is named by its id as well as its name: both are required, and neither is shared.
func TestARealizationIsNamedByItsIdAndItsName(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	again := proto.Clone(m.GetRealizations()[0]).(*modelirspb.Realization)
	again.Name = "another"
	m.Realizations = append(m.Realizations, again)
	err := Validate(m)
	require.ErrorContains(t, err, "two realizations with id temporal.nexuscaller.NexusRealization$.asyncNexus")
	require.ErrorContains(t, err, admRealizationAt)

	m = proto.Clone(load(t)).(*modelirspb.Model)
	m.GetRealizations()[0].Id = ""
	err = Validate(m)
	require.ErrorContains(t, err, "realization asyncNexus has no id")
	require.ErrorContains(t, err, admRealizationAt)

	// A machine the Model does not declare leaves the classes unread, and reports no class for it.
	m = proto.Clone(load(t)).(*modelirspb.Model)
	m.GetRealizations()[0].Machine = "nope"
	err = Validate(m)
	require.ErrorContains(t, err, "realization asyncNexus: no machine nope")
	require.NotContains(t, err.Error(), "binds no action")
	require.Len(t, problems(err), 1)
}

// The table a producer of Cases reads is the check table: the machine's rows with its hole rows as
// unknown pairs, never as disabled ones, and the same identity. It carries the state fields and the
// Abstraction Claims beside them, which enter no fingerprint.
func TestTheRealizersTableIsTheCheckTableWithFieldsAndClaims(t *testing.T) {
	m := lifted(t, "declarations")
	realizer, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	checking, realizing := bind(m, DefaultScope), realizer.b
	holed := 0
	for _, decl := range m.GetMachines() {
		mm := checking.machines[decl.GetName()]
		checked, err := checking.claimed(mm)
		require.NoError(t, err)
		realized, err := realizing.claimed(realizing.machines[decl.GetName()])
		require.NoError(t, err)
		var unknown, holes []string
		for _, u := range realized.Unknown {
			unknown = append(unknown, u.Row)
		}
		for _, h := range mm.Holes {
			holes = append(holes, h.Row)
			holed++
		}
		require.Equal(t, holes, unknown, decl.GetName())
		require.Equal(t, checked.Rows, realized.Rows)
		require.Equal(t, checked.TargetFingerprint(), realized.TargetFingerprint())
		require.Empty(t, checked.FieldValues(realized.States[0]), "a check reads no state fields")
	}
	require.Positive(t, holed, "the fixture declares hole rows")

	realizer, err = NewRealizer(load(t), DefaultScope)
	require.NoError(t, err)
	nexus := realizer.b
	table, err := nexus.claimed(nexus.machines["nexusProtocol"])
	require.NoError(t, err)
	const field = "temporal.nexus.caller.state-field.nexusProtocol."
	require.Equal(t, []Atom{{ID: field + "phase", Value: "succeeded"}, {ID: field + "attempts", Value: "1"},
		{ID: field + "scheduleToClose", Value: "unset"}, {ID: field + "scheduleToStart", Value: "unset"},
		{ID: field + "startToClose", Value: "unset"}, {ID: field + "nexusProduct", Value: "succeeded"}},
		table.FieldValues("succeeded-1-unset-unset-unset"))
	require.Contains(t, table.Claims(), Claim{Member: "temporal.nexus.caller.action.nexusProtocol.handlerReply-handlerError-false",
		Action: "temporal.nexus.caller.action.handlerReply", Field: "reply", ClassName: "handlerError (retryable := false)", Example: "BadRequest"})
}

// A Realizer binds only a Model admission lets through, and gives only the Queries that Model
// declares, by the key Check gives each one's receipt: a Query it binds always has a receipt.
func TestARealizerGivesOnlyTheQueriesOfAnAdmittedModel(t *testing.T) {
	m := load(t)
	unadmitted := proto.Clone(m).(*modelirspb.Model)
	unadmitted.GetRealizations()[0].Name = ""
	realizer, err := NewRealizer(unadmitted, DefaultScope)
	require.Nil(t, realizer)
	require.ErrorContains(t, err, "a realization has no name")

	realizer, err = NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	answered := 0
	for _, receipt := range Check(m, DefaultScope).Receipts {
		if receipt.Subject != QuerySubject {
			continue
		}
		answered++
		declared, err := realizer.Declared(receipt.Key)
		require.NoError(t, err)
		require.Equal(t, receipt.Key.Name, declared.Query.GetName())
		require.Equal(t, receipt.Position, where(declared.Query.GetPosition()))
		bound, err := realizer.Find(receipt.Key)
		require.NoError(t, err)
		require.Equal(t, receipt.Key.Name, bound.Name)
		require.Equal(t, receipt.Limits, bound.Limits)
	}
	require.Equal(t, len(m.GetQueries()), answered)

	// A declared Query copied under another name is no Query of the Model: no key names it.
	key := ClaimKey{Family: "temporal.nexus.caller", Owner: "nexusProtocol", Name: "syncCompletion"}
	_, err = realizer.Find(key)
	require.NoError(t, err)
	for _, c := range []struct {
		name string
		key  ClaimKey
		want string
		at   string
	}{
		{"another name", ClaimKey{Family: key.Family, Owner: key.Owner, Name: "syncCompletionAgain"}, "no Query syncCompletionAgain", m.GetSource()},
		{"another machine", ClaimKey{Family: key.Family, Owner: "nexusProduct", Name: key.Name},
			"query syncCompletion runs on nexusProtocol of temporal.nexus.caller, not on nexusProduct of temporal.nexus.caller",
			"model/temporal/nexuscaller/Claims.scala:"},
		{"another family", ClaimKey{Family: "temporal.worker", Owner: key.Owner, Name: key.Name},
			"query syncCompletion runs on nexusProtocol of temporal.nexus.caller, not on nexusProtocol of temporal.worker",
			"model/temporal/nexuscaller/Claims.scala:"},
		{"no name", ClaimKey{Family: key.Family, Owner: key.Owner}, "no Query ", m.GetSource()},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, find := range []func() error{
				func() error { _, err := realizer.Find(c.key); return err },
				func() error { _, err := realizer.Declared(c.key); return err },
			} {
				err := find()
				require.ErrorContains(t, err, c.want)
				var located *Error
				require.ErrorAs(t, err, &located)
				require.Contains(t, located.Position, c.at)
			}
		})
	}
}

// A reader that types paths may find several values at one, where a path fans out over a repeated
// field. Several values are no operand of a comparison or an order, and have no fields.
func TestSeveralValuesAreNoOperand(t *testing.T) {
	several := func(Typed, string) (Typed, error) { return Typed{Shape: SeveralShape}, nil }
	path := admPayload("items[*].name")
	for name, test := range map[string]struct {
		operand *modelirspb.Operand
		says    string
	}{
		"compared": {&modelirspb.Operand{Kind: &modelirspb.Operand_Equal{Equal: &modelirspb.Equal{Left: admWritten(&modelirspb.ProtoValue_Text{Text: "a"}), Right: path}}}, "compares several values"},
		"ordered":  {admGreater(path, admWritten(&modelirspb.ProtoValue_Number{Number: 0})), "orders several values, and only numbers are ordered"},
		"read":     {&modelirspb.Operand{Kind: &modelirspb.Operand_Path{Path: &modelirspb.PathOf{Path: "first", Of: path}}}, "reads first of several values, which is no message"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := TypeOf(test.operand, nil, several)
			require.Equal(t, &Mistype{Says: test.says}, err)
		})
	}
}

// Evidence that is the Run's record of an attempt says which: the attempt, counted from one, of the
// activity one script runs. A Run records an attempt once it is answered, so what the declaration
// names decides when the evidence reaches a Run, and a declaration that names no attempt of an
// activity is refused where it is written.
func TestTheAttemptARunEventRecordsIsOfAnActivitysScript(t *testing.T) {
	const started = "temporal.activity.standalone.evidence.statusStarted"
	source := func(t *testing.T, m *modelirspb.Model) *modelirspb.RunEventSource {
		for _, e := range m.GetRealizations()[0].GetEvidence() {
			if e.GetId() == started {
				e.GetRunEvent().Attempt = &modelirspb.AttemptOf{Script: "activity", Number: 1}
				return e.GetRunEvent()
			}
		}
		require.FailNow(t, "no evidence "+started)
		return nil
	}
	for name, test := range map[string]struct {
		change func(m *modelirspb.Model, source *modelirspb.RunEventSource)
		want   string
	}{
		"the first attempt of the activity's script": {func(*modelirspb.Model, *modelirspb.RunEventSource) {}, ""},
		"an attempt of no script": {func(_ *modelirspb.Model, s *modelirspb.RunEventSource) { s.GetAttempt().Script = "" },
			"evidence " + started + " is the record of an attempt of no script"},
		"an attempt of a script the realization does not declare": {func(_ *modelirspb.Model, s *modelirspb.RunEventSource) { s.GetAttempt().Script = "nobody" },
			"evidence " + started + " is the record of an attempt of script nobody, which the realization does not declare"},
		"an attempt of the controller's script": {func(_ *modelirspb.Model, s *modelirspb.RunEventSource) { s.GetAttempt().Script = "controller" },
			"evidence " + started + " is the record of an attempt of script controller, which no activity activates"},
		"an attempt counted from below one": {func(_ *modelirspb.Model, s *modelirspb.RunEventSource) { s.GetAttempt().Number = 0 },
			"evidence " + started + " is the record of attempt 0 of script activity; the attempts of an activity are counted from one"},
		"an attempt of a script that starts with no delivery": {func(m *modelirspb.Model, _ *modelirspb.RunEventSource) {
			for _, s := range m.GetRealizations()[0].GetScripts() {
				if s.GetActivity() != nil {
					s.GetActivity().Starts = nil
				}
			}
		}, "evidence " + started + " is the record of an attempt of script activity, which starts with no delivery: no step of a path is an attempt of it"},
		"an attempt recorded as the completion of a call": {func(_ *modelirspb.Model, s *modelirspb.RunEventSource) {
			s.Kind = modelirspb.RunEventSource_KIND_INSTRUCTION_COMPLETED
		}, "evidence " + started + " is the record of an attempt and of no diagnostic: a Run records an attempt as what a worker reports of an activation"},
		// What a worker reports of an activation reaches a Run once the activation is answered, whatever
		// the realization says of it: with no attempt named, when that is would be anyone's guess.
		"what a worker reports of an activation, declared the record of no attempt": {func(_ *modelirspb.Model, s *modelirspb.RunEventSource) { s.Attempt = nil },
			"evidence " + started + " is what a worker reports of an activation and is declared the record of no attempt: " +
				"a Run records it once the attempt is answered, and the realization says which attempt that is"},
	} {
		t.Run(name, func(t *testing.T) {
			m, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity.json"))
			require.NoError(t, err)
			test.change(m, source(t, m))
			err = Validate(m)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
			require.ErrorContains(t, err, "model/temporal/standaloneactivity/Realization.scala:", "the error is located")
		})
	}
}

// The Run's own record of a command is closed by that command: its record is the read. The held
// race declares the admission its release observes exhaustive, closed by the release, which is a
// performance, since a Case that does not release has no record whose silence could say anything.
// Any other command does not read the record, and closes nothing.
func TestTheRunsOwnRecordIsClosedByTheCommandItRecords(t *testing.T) {
	race := func(t *testing.T) (*modelirspb.Model, *modelirspb.Realization) {
		m, err := Load(filepath.Join("..", "..", "..", "model", "ir", "activity-race.json"))
		require.NoError(t, err)
		for _, r := range m.GetRealizations() {
			if r.GetName() == "heldDelivery" {
				return m, r
			}
		}
		t.Fatal("heldDelivery realization missing")
		return nil, nil
	}
	const admitted = "temporal.activity.standalone.evidence.attemptAdmitted"
	m, r := race(t)
	release := admCommand(t, admScript(t, r, "controller"), "release-dispatch")
	require.Equal(t, []string{admitted}, release.GetCloses())
	require.NoError(t, Validate(m))

	m, r = race(t)
	admCommand(t, admScript(t, r, "controller"), "release-dispatch").Closes = nil
	admCommand(t, admScript(t, r, "controller"), "hold-dispatch").Closes = []string{admitted}
	require.ErrorContains(t, Validate(m), "command hold-dispatch of script controller closes evidence "+admitted+" and does not read it")
}
