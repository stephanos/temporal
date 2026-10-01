package goir

// What a reader rejects of a realization before any lowering (model/scalav2/SEMANTICS.md, Admission):
// each case is one mutation of the Nexus caller realization the lifter emitted, reported at the
// position the lifter recorded for the declaration concerned.

import (
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/umpire"
	"google.golang.org/protobuf/proto"
)

const admRealizationAt = "model/scalav2/scala/temporal/nexuscaller/Realization.scala:"

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

func TestTheLiftedRealizationIsAdmitted(t *testing.T) {
	m := load(t)
	require.Len(t, m.GetRealizations(), 1)
	require.Equal(t, "nexusProtocol", m.GetRealizations()[0].GetMachine())
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
	require.Equal(t, []umpire.Atom{{ID: field + "phase", Value: "succeeded"}, {ID: field + "attempts", Value: "1"},
		{ID: field + "scheduleToClose", Value: "unset"}, {ID: field + "scheduleToStart", Value: "unset"},
		{ID: field + "startToClose", Value: "unset"}, {ID: field + "nexusProduct", Value: "succeeded"}},
		table.FieldValues("succeeded-1-unset-unset-unset"))
	require.Contains(t, table.Claims(), umpire.Claim{Member: "temporal.nexus.caller.action.nexusProtocol.handlerReply-handlerError-false",
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
			"model/scalav2/scala/temporal/nexuscaller/Claims.scala:"},
		{"another family", ClaimKey{Family: "temporal.worker", Owner: key.Owner, Name: key.Name},
			"query syncCompletion runs on nexusProtocol of temporal.nexus.caller, not on nexusProtocol of temporal.worker",
			"model/scalav2/scala/temporal/nexuscaller/Claims.scala:"},
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
