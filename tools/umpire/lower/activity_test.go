package lower

// An activity's script, the evidence a realization declares beyond a kind and a source, and what the
// standalone activity Model's own realization declares. The fixtures are in
// model/lifter/testdata/lifts/Realizations.scala; the activity realization is
// model/temporal/standaloneactivity/Realization.scala, and its Cases are in
// activity_cases_test.go.

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	failurepb "go.temporal.io/api/failure/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const activityRealizationAt = "model/temporal/standaloneactivity/Realization.scala"

var errandIdentity = cp.IdentityFor("temporal.case", "fixture", "errand")

func realizationNamed(t *testing.T, m *umpirespb.Model, name string) *umpirespb.Realization {
	t.Helper()
	for _, r := range m.GetRealizations() {
		if r.GetName() == name {
			return r
		}
	}
	require.FailNow(t, "no realization "+name)
	return nil
}

func scriptNamed(t *testing.T, r *umpirespb.Realization, id string) *umpirespb.Script {
	t.Helper()
	for _, s := range r.GetScripts() {
		if s.GetId() == id {
			return s
		}
	}
	require.FailNow(t, "no script "+id)
	return nil
}

func role(t *testing.T, c *testpilotspb.Case, kind testpilotspb.RoleKind) *testpilotspb.Role {
	t.Helper()
	for _, r := range c.GetProgram().GetRoles() {
		if r.GetKind() == kind {
			return r
		}
	}
	require.FailNow(t, "no role of kind "+kind.String())
	return nil
}

// The errand's path is a start, a delivery, a failed attempt, a second delivery and a completed
// attempt (Realizations.scala, retriedOnce). Its Case is read off the fixture: the controller
// starts the activity and polls the two listings; the activity entrypoint holds the path's two answers
// in order, the failing one as an attempt failure and never as a result; a delivery is the activation
// and has no instruction. The start names the namespace and the task queue by the bindings of the roles
// the activity runs under, and the run's id and the Case's own activity type. Ordinary preparation
// admits the Case under the Profile derived from it.
func TestAnActivityScriptLowersToItsAttemptsInOrder(t *testing.T) {
	p, err := NewProducer(liftedRealizations(t))
	require.NoError(t, err)
	l, err := p.Lower("errand.retry", errandIdentity)
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing)
	require.Empty(t, l.Unsupported)
	c := l.Case

	require.Equal(t, map[string][]string{"controller": {"start-activity", "await-listed", "await-closed"},
		"errand": {"fail-attempt", "complete-attempt"}}, instructionIDs(c))

	const activityType = "umpire-fixture-errand-errand"
	worker, queue := role(t, c, testpilotspb.ROLE_KIND_WORKER), role(t, c, testpilotspb.ROLE_KIND_TASK_QUEUE)
	protorequire.ProtoEqual(t, &testpilotspb.ActivityActivation{ActivityType: activityType, WorkerRoleId: worker.GetRoleId(),
		TaskQueueRoleId: queue.GetRoleId()}, c.GetProgram().GetEntrypoints()[1].GetActivity())

	protorequire.ProtoEqual(t, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{
		ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{Failure: &failurepb.Failure{Message: "not yet",
			FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "NotYet"}}}}}},
		instruction(t, c, "errand", "fail-attempt").GetInstruction())
	protorequire.ProtoEqual(t, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{
		Finish: &testpilotspb.Finish{Result: cp.Literal(cp.Text("done"))}}}, instruction(t, c, "errand", "complete-attempt").GetInstruction())

	started := instruction(t, c, "controller", "start-activity").GetInstruction().GetInvokeRpc()
	require.Equal(t, "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution", started.GetMethod())
	protorequire.ProtoSliceEqual(t, []*testpilotspb.RequestAssignment{
		cp.Assign("namespace", cp.Environment(worker.GetNamespaceBindingId())),
		cp.Assign("activity_id", cp.Run()),
		cp.Assign("activity_type.name", cp.Literal(cp.Text(activityType))),
		cp.Assign("task_queue.name", cp.Environment(queue.GetResourceBindingId())),
		cp.Assign("request_id", cp.Run()),
		cp.Assign("start_to_close_timeout.seconds", cp.Literal(cp.SignedInteger(300))),
	}, started.GetRequestAssignments())
	require.NotEmpty(t, worker.GetNamespaceBindingId())
	require.NotEmpty(t, queue.GetResourceBindingId())

	protorequire.ProtoEqual(t, cp.Equal(cp.Path(cp.ProjectedValue(), "status"), cp.Literal(cp.Enum("ACTIVITY_EXECUTION_STATUS_COMPLETED"))),
		instruction(t, c, "controller", "await-closed").GetInstruction().GetReadEvidence().GetUntil())
	// A conjunction, an order and a negation lower to the expressions of those names.
	protorequire.ProtoEqual(t, &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{
		cp.Present(cp.Path(cp.ProjectedValue(), "schedule_time")),
		{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_GREATER_THAN,
			Left: cp.Path(cp.ProjectedValue(), "state_transition_count"), Right: cp.Literal(cp.SignedInteger(0))}}},
		{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{
			Operand: cp.Equal(cp.Path(cp.ProjectedValue(), "activity_id"), cp.Literal(cp.Text("")))}}},
	}}}}, instruction(t, c, "controller", "await-listed").GetInstruction().GetReadEvidence().GetUntil())

	// The deliveries and the failed attempt record nothing a caller reads: each is a Known Gap.
	var silent []string
	for _, gap := range c.GetProvenance().GetKnownGaps() {
		silent = append(silent, strings.TrimPrefix(gap.GetSubject(), "fixture.realizations.errand.action.errand."))
	}
	require.Equal(t, []string{"deliver", "answer-failed"}, silent)

	encoded, err := protojson.Marshal(c)
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "errand-profile", Namespace: "namespace",
		TaskQueue: "task-queue"})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	require.True(t, proto.Equal(source, prepared.Snapshot()), "preparation carries the Case unchanged")

	// Identical inputs give identical bytes: the fixture read again, lowered by another producer.
	again, err := NewProducer(liftedRealizations(t))
	require.NoError(t, err)
	second, err := again.Lower("errand.retry", errandIdentity)
	require.NoError(t, err)
	first, err := proto.MarshalOptions{Deterministic: true}.Marshal(c)
	require.NoError(t, err)
	repeated, err := proto.MarshalOptions{Deterministic: true}.Marshal(second.Case)
	require.NoError(t, err)
	require.Equal(t, first, repeated)
}

// The inventory of the errand's Case has an entry for each delivery class the script starts with, and
// every other declaration, as for any realization.
func TestTheInventoryNamesTheDeliveriesOfAnActivityScript(t *testing.T) {
	m := liftedRealizations(t)
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("errand.retry", errandIdentity)
	require.NoError(t, err)
	var got [][2]string
	as := map[string][]string{}
	for _, e := range l.Inventory {
		got = append(got, [2]string{e.Kind, e.ID})
		as[e.Kind+" "+e.ID] = e.As
	}
	require.ElementsMatch(t, declared(t, m, realizationNamed(t, m, "errandRealization")), got)
	require.Equal(t, []string{"program.entrypoints[errand]"}, as["activation errand [deliver]"])
}

// An activity's script holds the answers to its attempts and nothing else, and an attempt fails with
// an application failure or one that names no kind: what Testpilot would refuse is refused here, where
// it was written. A delivery the script does not start with is a step nothing performs.
func TestAnActivityScriptAnswersItsAttempts(t *testing.T) {
	answer := func(t *testing.T, m *umpirespb.Model, id string) *umpirespb.Command {
		for _, p := range scriptNamed(t, realizationNamed(t, m, "errandRealization"), "errand").GetItems()[0].GetPerforms() {
			if p.GetCommand().GetId() == id {
				return p.GetCommand()
			}
		}
		require.FailNow(t, "no command "+id)
		return nil
	}
	for _, c := range []struct {
		name   string
		mutate func(t *testing.T, m *umpirespb.Model)
		want   string
	}{
		{"a command that is no answer", func(t *testing.T, m *umpirespb.Model) {
			answer(t, m, "complete-attempt").Instruction = &umpirespb.Command_Fault{Fault: &umpirespb.Fault{Role: "temporal.task-queue",
				Kind: umpirespb.Fault_KIND_WORKER_STOP}}
		}, "command complete-attempt of activity script errand is no answer to an attempt: an attempt ends with a result or a failure"},
		{"a failure that is no failure", func(t *testing.T, m *umpirespb.Model) {
			answer(t, m, "fail-attempt").GetAttemptFailure().GetFailure().Message = "temporal.api.common.v1.Payload"
		}, "temporal.api.common.v1.Payload is written where a temporal.api.failure.v1.Failure belongs"},
		{"a failure of another kind than the application's", func(t *testing.T, m *umpirespb.Model) {
			answer(t, m, "fail-attempt").GetAttemptFailure().GetFailure().GetFields()[1] = &umpirespb.ProtoField{Name: "timeout_failure_info",
				Value: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Message{Message: &umpirespb.Proto{Message: "temporal.api.failure.v1.TimeoutFailureInfo"}}}}
		}, "command fail-attempt fails its attempt with a timeout_failure_info; an attempt fails with an application failure or one that names no kind"},
		{"a delivery the script does not start with", func(t *testing.T, m *umpirespb.Model) {
			scriptNamed(t, realizationNamed(t, m, "errandRealization"), "errand").GetActivity().Starts = nil
		}, "scenario retriedOnce takes deliver, a step of runner, and no script of realization errandRealization performs it"},
		{"a poll whose condition reads the run", func(t *testing.T, m *umpirespb.Model) {
			poll := scriptNamed(t, realizationNamed(t, m, "errandRealization"), "controller").GetItems()[2].GetCommand().GetPoll()
			poll.GetUntil().GetEqual().Right = &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &umpirespb.Empty{}}}
		}, "command await-closed polls until a condition that reads the run's id; a poll's condition reads only the value the poll is looking at"},
		{"a poll whose condition orders an enum value", func(t *testing.T, m *umpirespb.Model) {
			poll := scriptNamed(t, realizationNamed(t, m, "errandRealization"), "controller").GetItems()[2].GetCommand().GetPoll()
			poll.Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Greater{Greater: &umpirespb.Greater{Left: poll.GetUntil().GetEqual().GetLeft(),
				Right: &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Number{Number: 0}}}}}}}
		}, "command await-closed polls until a condition that orders an enum value, and only numbers are ordered"},
		{"a poll whose condition reads a path of a path, the outer misspelled", func(t *testing.T, m *umpirespb.Model) {
			poll := scriptNamed(t, realizationNamed(t, m, "errandRealization"), "controller").GetItems()[2].GetCommand().GetPoll()
			element := &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}
			poll.Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Path{
				Path: &umpirespb.PathOf{Path: "secnods", Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "schedule_time", Of: element}}}}}}}}}
		}, "google.protobuf.Timestamp has no field secnods"},
		{"a poll whose condition reads a path of a text", func(t *testing.T, m *umpirespb.Model) {
			poll := scriptNamed(t, realizationNamed(t, m, "errandRealization"), "controller").GetItems()[2].GetCommand().GetPoll()
			element := &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}
			poll.Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Path{
				Path: &umpirespb.PathOf{Path: "length", Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "activity_id", Of: element}}}}}}}}}
		}, "command await-closed polls until a condition that reads length of a text, which is no message"},
		{"a poll until a text of the element", func(t *testing.T, m *umpirespb.Model) {
			poll := scriptNamed(t, realizationNamed(t, m, "errandRealization"), "controller").GetItems()[2].GetCommand().GetPoll()
			poll.Until = &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "activity_id",
				Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}}}}
		}, "command await-closed polls until a text, and a poll's condition is a condition"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := liftedRealizations(t)
			c.mutate(t, m)
			p, err := NewProducer(m)
			require.NoError(t, err)
			l, err := p.Lower("errand.retry", errandIdentity)
			require.Nil(t, l)
			require.ErrorContains(t, err, c.want)
			var located *umpiremodel.Error
			require.ErrorAs(t, err, &located)
			require.Contains(t, located.Position, liftsDir)
		})
	}
}

// The tally declares a field its evidence carries without its value (Realizations.scala,
// `identity`). A lift reads a value for every field its evidence declares, so no Case carries a field
// without one: the field is named where it was written, as a limit no task owns, and no Case is built
// around it. What the tally declares beside it, evidence read from one message, the fields it keeps
// and the Run's own record, a Case carries, and none of them is named.
func TestARedactedFieldIsNamedAsALimit(t *testing.T) {
	p, err := NewProducer(liftedRealizations(t))
	require.NoError(t, err)
	l, err := p.Lower("tally.opens", cp.IdentityFor("temporal.case", "fixture", "tally"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, l.Standing)
	require.Nil(t, l.Case)
	require.Empty(t, l.Inventory)
	require.Empty(t, l.OffPath)

	require.Len(t, l.Unsupported, 1)
	gap := l.Unsupported[0]
	require.NotEmpty(t, gap.Why)
	require.Contains(t, gap.Position, realizationsAt)
	require.Contains(t, lineOf(t, gap.Position), `EvidenceField.typed(`)
	gap.Why, gap.Position = "", ""
	require.Equal(t, Unsupported{Construct: "redacted evidence field", ID: "fixture.realizations.tally.evidence.opened/identity",
		Owner: "none: a recorded limit of the prototype"}, gap)
}

// With the redacted field retained and a cleanup named, the tally lowers: its evidence is read from the one message a
// response holds, by a poll that closes the kind, and keeps the three fields. Ordinary preparation
// admits the Case. The Run's record of the push records a fact the door never records, so that kind is
// off the path.
func TestEvidenceReadFromOneMessageWithItsFieldsLowers(t *testing.T) {
	m := liftedRealizations(t)
	tally := realizationNamed(t, m, "tallyRealization")
	for _, f := range tally.GetEvidence()[0].GetFields() {
		f.Redacted = false
	}
	// The fixture names no cleanup, which every Program has.
	tally.Cleanup = "cleanup"
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("tally.opens", cp.IdentityFor("temporal.case", "fixture", "tally"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
	c := l.Case

	require.Len(t, c.GetProgram().GetEvidence(), 1)
	declared := c.GetProgram().GetEvidence()[0]
	protorequire.ProtoEqual(t, &testpilotspb.ReadSource{Method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution",
		Path: "info", Single: true}, declared.GetRead())
	protorequire.ProtoSliceEqual(t, []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "run", Path: "run_id"}, {FieldId: "attempt", Path: "attempt"},
		{FieldId: "identity", Path: "last_worker_identity"}}, declared.GetFields())
	require.Equal(t, declared.GetEvidenceId(), instruction(t, c, "controller", "await-opened").GetInstruction().GetReadEvidence().GetEvidenceId())
	rules := c.GetContract().GetCorrelated().GetProjectionRules()
	require.Len(t, rules, 1)
	retained := func(id string, kind testpilotspb.ScalarKind) *testpilotspb.CorrelatedFieldPolicy {
		return &testpilotspb.CorrelatedFieldPolicy{FieldId: id, Type: &testpilotspb.ScalarType{Kind: kind}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}
	}
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedFieldPolicy{retained("run", testpilotspb.SCALAR_KIND_TEXT),
		retained("attempt", testpilotspb.SCALAR_KIND_UINT64), retained("identity", testpilotspb.SCALAR_KIND_TEXT)}, rules[0].GetFields())

	dispositions, as := map[string]Disposition{}, map[string][]string{}
	for _, e := range l.Inventory {
		dispositions[e.Kind+" "+e.ID], as[e.Kind+" "+e.ID] = e.Disposition, e.As
	}
	require.Equal(t, OffPath, dispositions["evidence fixture.realizations.tally.evidence.pushed"])
	require.Equal(t, []string{"program.evidence[" + declared.GetEvidenceId() + "]", "contract.correlated.sources",
		"program.entrypoints[controller].instructions[await-opened]"}, as["evidence fixture.realizations.tally.evidence.opened"])

	encoded, err := protojson.Marshal(c)
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: "tally-profile", Namespace: "namespace", TaskQueue: "task-queue"})
	require.NoError(t, err)
	_, err = testpilot.Prepare(source, profile)
	require.NoError(t, err)
}

// What a kind of evidence reads is checked against the descriptors it names before any gap is said: a
// single read that reaches no single message, and a field that is no text, flag or integer of the
// recorded message, are errors at the declaration.
func TestTheFieldsAndTheSingleReadOfEvidenceAreCheckedAgainstTheirDescriptors(t *testing.T) {
	const opened = "evidence fixture.realizations.tally.evidence.opened"
	payload := func(path string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: path,
			Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}}}}
	}
	// enumName is an enum value written out by its name.
	type enumName string
	written := func(kind any) *umpirespb.Operand {
		value := &umpirespb.ProtoValue{}
		switch k := kind.(type) {
		case string:
			value.Kind = &umpirespb.ProtoValue_Text{Text: k}
		case int:
			value.Kind = &umpirespb.ProtoValue_Number{Number: int64(k)}
		case enumName:
			value.Kind = &umpirespb.ProtoValue_EnumName{EnumName: string(k)}
		default:
		}
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: value}}
	}
	const pushed, outcome = "evidence fixture.realizations.tally.evidence.pushed", "temporal.server.api.testpilot.v1.InstructionOutcome"
	for _, c := range []struct {
		name   string
		mutate func(e *umpirespb.Evidence)
		want   string
	}{
		{"a Run Event's guard over a field its payload does not have", func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetEqual().Left = payload("state")
		}, "temporal.server.api.testpilot.v1.InstructionOutcome has no field state"},
		{"a Run Event's key at a path its payload does not have", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Key = payload("activity_attempt.run_id")
		}, "temporal.server.api.testpilot.v1.ActivityAttempt has no field run_id"},
		{"a Run Event's guard over a path of a path, the inner misspelled", func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetEqual().Left = &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "sdk_attempt", Of: payload("activity_attemtp")}}}
		}, pushed + ": its guard reads activity_attemtp, and temporal.server.api.testpilot.v1.InstructionOutcome has no field activity_attemtp"},
		{"a Run Event's guard over a path of a path, the outer misspelled", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{
				Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "sdk_attemtp", Of: payload("activity_attempt")}}}}}}
		}, pushed + ": its guard reads sdk_attemtp, and temporal.server.api.testpilot.v1.ActivityAttempt has no field sdk_attemtp"},
		{"a Run Event's guard that is a number", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Guard = payload("activity_attempt.sdk_attempt")
		}, pushed + ": its guard is a number, and a guard is a condition"},
		{"a Run Event's guard that is a message", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Guard = payload("activity_attempt")
		}, pushed + ": its guard is a message, and a guard is a condition"},
		{"a Run Event's guard that orders a text", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Greater{Greater: &umpirespb.Greater{
				Left: payload("activity_attempt.delivery_id"), Right: written(0)}}}
		}, pushed + ": its guard orders a text, and only numbers are ordered"},
		{"a Run Event's guard that compares a number with a text", func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetEqual().Left, e.GetRunEvent().GetGuard().GetEqual().Right = payload("activity_attempt.sdk_attempt"), written("1")
		}, pushed + ": its guard compares a number with a text"},
		{"a Run Event's guard that compares an enum with a name it does not have", func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetEqual().GetRight().GetLiteral().Kind = &umpirespb.ProtoValue_EnumName{EnumName: "INSTRUCTION_OUTCOME_STATUS_NOPE"}
		}, pushed + ": its guard compares a value of temporal.server.api.testpilot.v1.InstructionOutcomeStatus with INSTRUCTION_OUTCOME_STATUS_NOPE, which it does not have"},
		{"a Run Event's guard that negates a number", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_Not{Not: &umpirespb.Not{Of: payload("activity_attempt.sdk_attempt")}}}
		}, pushed + ": its guard negates a number, and only a condition is negated"},
		{"a Run Event's guard that compares values of two enums", func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetEqual().Right = payload("activity_attempt.response")
		}, pushed + ": its guard compares a value of temporal.server.api.testpilot.v1.InstructionOutcomeStatus with one of temporal.server.api.testpilot.v1.ActivityAttemptResponse"},
		{"a Run Event's guard that compares a name an enum does not have with it", func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetEqual().Left, e.GetRunEvent().GetGuard().GetEqual().Right = written(enumName("INSTRUCTION_OUTCOME_STATUS_NOPE")), payload("status")
		}, pushed + ": its guard compares a value of temporal.server.api.testpilot.v1.InstructionOutcomeStatus with INSTRUCTION_OUTCOME_STATUS_NOPE, which it does not have"},
		{"a Run Event's guard that joins a text", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Guard = &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{
				Operands: []*umpirespb.Operand{e.GetRunEvent().GetGuard(), payload("activity_attempt.delivery_id")}}}}
		}, pushed + ": its guard joins a text, and only conditions are joined"},
		{"a Run Event's key that is each of several texts", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Key = payload("value.list_value.values[*].text_value")
		}, pushed + ": its key reads value.list_value.values[*].text_value, which is no single text or integer of " + outcome},
		{"a Run Event's key that is a message", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Key = payload("activity_attempt")
		}, pushed + ": its key reads activity_attempt, which is no single text or integer of " + outcome},
		{"a Run Event's key that is an enum", func(e *umpirespb.Evidence) {
			e.GetRunEvent().Key = payload("status")
		}, pushed + ": its key reads status, which is no single text or integer of " + outcome},
		{"a Run Event's field that is an enum", func(e *umpirespb.Evidence) {
			e.Fields = []*umpirespb.EvidenceField{{Id: "offered", Path: "activity_attempt.response"}}
		}, "evidence fixture.realizations.tally.evidence.pushed: field offered reads activity_attempt.response, which is no single text, flag or integer of temporal.server.api.testpilot.v1.InstructionOutcome"},
		{"a single read of a repeated field", func(e *umpirespb.Evidence) { e.GetSingle().Path = "callbacks" },
			opened + " is read from callbacks, which is no single message of temporal.api.workflowservice.v1.DescribeActivityExecutionResponse"},
		{"a single read of a scalar", func(e *umpirespb.Evidence) { e.GetSingle().Path = "run_id" },
			opened + " is read from run_id, which is no single message of temporal.api.workflowservice.v1.DescribeActivityExecutionResponse"},
		{"a field the recorded message does not have", func(e *umpirespb.Evidence) { e.GetFields()[1].Path = "attempts" },
			"temporal.api.activity.v1.ActivityExecutionInfo has no field attempts"},
		{"a field that is a message", func(e *umpirespb.Evidence) { e.GetFields()[1].Path = "schedule_time" },
			opened + ": field attempt reads schedule_time, which is no single text, flag or integer of temporal.api.activity.v1.ActivityExecutionInfo"},
		{"a field that is an enum", func(e *umpirespb.Evidence) { e.GetFields()[1].Path = "status" },
			opened + ": field attempt reads status, which is no single text, flag or integer of temporal.api.activity.v1.ActivityExecutionInfo"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := liftedRealizations(t)
			// A case about the Run's own record changes the tally's second kind, and any other its first.
			changed := realizationNamed(t, m, "tallyRealization").GetEvidence()[0]
			if strings.Contains(c.name, "Run Event") {
				changed = realizationNamed(t, m, "tallyRealization").GetEvidence()[1]
			}
			c.mutate(changed)
			p, err := NewProducer(m)
			require.NoError(t, err)
			l, err := p.Lower("tally.opens", cp.IdentityFor("temporal.case", "fixture", "tally"))
			require.Nil(t, l)
			require.ErrorContains(t, err, c.want)
			var located *umpiremodel.Error
			require.ErrorAs(t, err, &located)
			require.Contains(t, located.Position, realizationsAt)
		})
	}
}

// An exhaustive kind of evidence is carried by every Case, with its closing read: the inventory names
// the instruction, which is the command the realization says closes the kind. The Nexus caller's
// history kinds are closed by the history read (Realization.scala); the scheduled event, read by a
// poll that stops at the first one, is not exhaustive and names none.
func TestAnExhaustiveKindIsCarriedWithItsClosingRead(t *testing.T) {
	p, err := NewProducer(loaded(t, "nexus-caller"))
	require.NoError(t, err)
	l, err := p.Lower("syncCompletion", nexusIdentity("syncCompletion"))
	require.NoError(t, err)
	as := map[string][]string{}
	for _, e := range l.Inventory {
		as[e.Kind+" "+e.ID] = e.As
	}
	local := map[string]string{}
	for _, n := range l.Case.GetProvenance().GetLocalNames() {
		local[n.GetDefinitionId()] = n.GetLocalName()
	}
	const completed, scheduled = "temporal.nexus.caller.evidence.completed", "temporal.nexus.caller.evidence.scheduled"
	require.Equal(t, []string{"program.evidence[" + local[completed] + "]", "contract.correlated.sources",
		"program.entrypoints[controller].instructions[history]"}, as["evidence "+completed])
	require.Equal(t, []string{"program.evidence[" + local[scheduled] + "]", "contract.correlated.sources"}, as["evidence "+scheduled])
	// The started event is exhaustive too, so the Case carries it though no step of the path records it.
	const started = "temporal.nexus.caller.evidence.started"
	require.Equal(t, []string{"program.evidence[" + local[started] + "]", "contract.correlated.sources",
		"program.entrypoints[controller].instructions[history]"}, as["evidence "+started])
}

// The activity realization depends on observing no state the activity passes through on its own. With
// a running worker a scheduled activity is started, and a started one answered, before a poll need see
// either, so a poll of DescribeActivityExecution reads only a status the activity stays in until the
// controller acts or for good: paused, and the five terminal statuses.
func TestTheActivityRealizationPollsNoTransientState(t *testing.T) {
	m := loaded(t, "activity")
	stable := map[string]string{
		"statusPaused":     "ACTIVITY_EXECUTION_STATUS_PAUSED",
		"statusCompleted":  "ACTIVITY_EXECUTION_STATUS_COMPLETED",
		"statusFailed":     "ACTIVITY_EXECUTION_STATUS_FAILED",
		"statusCanceled":   "ACTIVITY_EXECUTION_STATUS_CANCELED",
		"statusTerminated": "ACTIVITY_EXECUTION_STATUS_TERMINATED",
		"statusTimedOut":   "ACTIVITY_EXECUTION_STATUS_TIMED_OUT",
	}
	r := m.GetRealizations()[0]
	records := map[string]string{}
	for _, e := range r.GetEvidence() {
		records[e.GetId()] = e.GetRecords()
	}
	polled := map[string]string{}
	for _, s := range r.GetScripts() {
		for _, c := range commandsOf(s) {
			poll := c.GetPoll()
			if poll == nil {
				continue
			}
			until := poll.GetUntil().GetEqual()
			require.Equal(t, "status", until.GetLeft().GetPath().GetPath(), "%s reads the status alone", c.GetId())
			polled[records[poll.GetEvidence()]] = until.GetRight().GetLiteral().GetEnumName()
		}
	}
	require.Equal(t, stable, polled)

	// What the activity passes through is read from the Run's own record instead. A call's answer is
	// the completion of the command that made it, where it succeeded; an attempt's start is the worker's
	// report of an activation the start call carries, declared the record of one numbered attempt of the
	// activity's script, where a delivery was made: its delivery is not empty. Each is keyed by the run. No guard and no field
	// reads what the worker then offered. What the guard admits is tested where it is evaluated
	// (tools/umpire/conformance, TestOnlyADeliveredAttemptIsEvidenceOfAnAttemptStart).
	//
	// Each kind that could confirm more steps than one names the ones it does: the first attempt's
	// record the first attempt start; the second attempt's the failure the server retried and the second
	// attempt start; and the release's answer the release, which schedules the activity as the start
	// does.
	projected := &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}
	path := func(p string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Of: projected, Path: p}}}
	}
	present := func(p string) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: path(p)}}}
	}
	run := &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &umpirespb.Empty{}}}
	accepted := func(command string) *umpirespb.RunEventSource {
		return &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED, Script: "controller", Command: command, Key: run,
			Guard: &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: path("status"),
				Right: &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{
					Kind: &umpirespb.ProtoValue_EnumName{EnumName: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}}}}}}
	}
	delivered := func(attempt int64) *umpirespb.RunEventSource {
		return &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_DIAGNOSTIC, Script: "controller", Command: "start-activity", Key: run,
			Attempt: &umpirespb.AttemptOf{Script: "activity", Number: attempt},
			Guard: &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{Operands: []*umpirespb.Operand{
				present("activity_attempt"),
				{Kind: &umpirespb.Operand_Not{Not: &umpirespb.Not{Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{
					Left:  path("activity_attempt.delivery_id"),
					Right: &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Text{}}}}}}}}}},
			}}}}}
	}
	identity := []*umpirespb.EvidenceField{
		{Id: "attempt", Path: "activity_attempt.sdk_attempt", Role: umpirespb.EvidenceField_ROLE_ATTEMPT},
		{Id: "delivery", Path: "activity_attempt.delivery_id", Role: umpirespb.EvidenceField_ROLE_DELIVERY},
		{Id: "activityRun", Path: "activity_attempt.activity_run_id"},
	}
	type taking struct {
		class      string
		occurrence int64
	}
	type recordedKind struct {
		records string
		source  *umpirespb.RunEventSource
		fields  []*umpirespb.EvidenceField
		names   []taking
	}
	realizer, err := umpiremodel.NewRealizer(m, umpiremodel.DefaultScope)
	require.NoError(t, err)
	const evidence = "temporal.activity.standalone.evidence."
	// The Run numbers its own record, so the kinds read from it count in one source, in the order the
	// Run records them; each status is read by a poll of its own, and counts in a source of its own.
	const recordSource = "temporal.activity.standalone.source.record"
	polledSources := map[string]bool{}
	got := map[string]recordedKind{}
	for _, e := range r.GetEvidence() {
		if _, read := stable[e.GetRecords()]; read && e.GetSingle() != nil {
			require.Equal(t, "info", e.GetSingle().GetPath(), e.GetRecords())
			require.Empty(t, e.GetConfirms(), e.GetRecords())
			require.NotEqual(t, recordSource, e.GetSource(), e.GetRecords())
			require.False(t, polledSources[e.GetSource()], "%s is read by a poll of its own", e.GetRecords())
			polledSources[e.GetSource()] = true
			continue
		}
		require.Equal(t, recordSource, e.GetSource(), e.GetId())
		require.Empty(t, e.GetOperation(), e.GetId())
		bare := proto.CloneOf(e)
		unpositioned(bare.ProtoReflect())
		kind := recordedKind{records: e.GetRecords(), source: bare.GetRunEvent(), fields: bare.GetFields()}
		for _, named := range e.GetConfirms() {
			kind.names = append(kind.names, taking{realizer.ClassKey(named.GetStep()), named.GetOccurrence()})
		}
		got[strings.TrimPrefix(e.GetId(), evidence)] = kind
	}
	want := map[string]recordedKind{
		"statusScheduled":       {"statusScheduled", accepted("start-activity"), nil, nil},
		"statusScheduledAgain":  {"statusScheduled", accepted("unpause-activity"), nil, []taking{{"control-unpause", 1}}},
		"statusCancelRequested": {"statusCancelRequested", accepted("request-cancel-activity"), nil, nil},
		"statusStarted":         {"statusStarted", delivered(1), identity, []taking{{"attemptStart", 1}}},
		"attemptCount":          {"attemptCount", delivered(2), identity, []taking{{"attemptResult-failed-true", 1}, {"attemptStart", 2}}},
	}
	require.ElementsMatch(t, slices.Collect(maps.Keys(want)), slices.Collect(maps.Keys(got)))
	for id, kind := range want {
		require.Equal(t, kind.records, got[id].records, id)
		protorequire.ProtoEqual(t, kind.source, got[id].source)
		protorequire.ProtoSliceEqual(t, kind.fields, got[id].fields)
		require.Equal(t, kind.names, got[id].names, id)
	}
}

// unpositioned takes every source position out of a message, so that a declaration is compared as
// what it declares.
func unpositioned(m protoreflect.Message) {
	m.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.Message() == nil || field.IsMap():
		case field.Message().FullName() == (&umpirespb.Position{}).ProtoReflect().Descriptor().FullName():
			m.Clear(field)
		case field.IsList():
			for i := range value.List().Len() {
				unpositioned(value.List().Get(i).Message())
			}
		default:
			unpositioned(value.Message())
		}
		return true
	})
}

// What Testpilot cannot run of a command stands in the way of the Queries whose path takes the command
// and of no other. The errand's controller is given a command that holds a delivery, carried for the
// canceled answer alone, with the control it holds: the control is the realization's and stands in the
// way of every Query, and the command stands in the way of the Query whose path cancels,
// `errand.withdrawn`, and is off the path of the retry, where it is listed and blocks nothing.
func TestAnUnsupportedCommandBlocksOnlyTheQueriesWhosePathTakesIt(t *testing.T) {
	m := liftedRealizations(t)
	errand := realizationNamed(t, m, "errandRealization")
	race := realizationNamed(t, m, "pauseRace")
	require.NotEmpty(t, race.GetControls(), "the held race declares a control")
	errand.Controls = append(errand.Controls, proto.CloneOf(race.GetControls()[0]))
	var canceled *umpirespb.ActionClass
	for _, performance := range scriptNamed(t, errand, "errand").GetItems()[0].GetPerforms() {
		if performance.GetCommand().GetId() == "cancel-attempt" {
			canceled = performance.GetStep()
		}
	}
	require.NotNil(t, canceled)
	controller := scriptNamed(t, errand, "controller")
	controller.Items = append(controller.Items, &umpirespb.Item{Position: controller.GetItems()[0].GetPosition(), When: []*umpirespb.ActionClass{canceled},
		Command: &umpirespb.Command{Id: "hold-it", Position: controller.GetItems()[0].GetPosition(),
			Instruction: &umpirespb.Command_Hold{Hold: race.GetControls()[0].GetId()}}})
	p, err := NewProducer(m)
	require.NoError(t, err)
	type gap struct{ construct, id, owner string }
	control := gap{"hold-delivery control", race.GetControls()[0].GetId(), ownerNone}
	command := gap{"hold-delivery command", "controller/hold-it", ownerNone}
	listed := func(entries []Unsupported) []gap {
		var out []gap
		for _, u := range entries {
			require.NotEmpty(t, u.Why)
			out = append(out, gap{u.Construct, u.ID, u.Owner})
		}
		return out
	}

	blocked, err := p.Lower("errand.withdrawn", cp.IdentityFor("temporal.case", "fixture", "withdrawn"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, blocked.Standing)
	require.Nil(t, blocked.Case)
	require.Equal(t, []gap{control, command}, listed(blocked.Unsupported))
	require.Empty(t, blocked.OffPath)

	free, err := p.Lower("errand.retry", errandIdentity)
	require.NoError(t, err)
	require.Equal(t, NotSupported, free.Standing)
	require.Equal(t, []gap{control}, listed(free.Unsupported))
	require.Equal(t, []gap{command}, listed(free.OffPath))
}

// A path that answers its attempt as canceled lowers, the answer as the instruction of that name
// (Realizations.scala, canceledOnce), and the inventory of a path that does not take it
// accounts for the command as one its path does not perform. The errand's machine has no step that
// requests a cancellation, so nothing in its Case asks the server to cancel: what a Run of it would do
// is the Scenario's to say, and no concern of how a command is lowered.
func TestACanceledAnswerLowersToItsInstruction(t *testing.T) {
	m := liftedRealizations(t)
	p, err := NewProducer(m)
	require.NoError(t, err)
	withdrawn, err := p.Lower("errand.withdrawn", cp.IdentityFor("temporal.case", "fixture", "withdrawn"))
	require.NoError(t, err)
	require.Equal(t, Lowered, withdrawn.Standing, "%v", withdrawn.Unsupported)
	require.Empty(t, withdrawn.OffPath)
	protorequire.ProtoEqual(t, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptCancellation{
		ActivityAttemptCancellation: &testpilotspb.ActivityAttemptCancellation{}}}, instruction(t, withdrawn.Case, "errand", "cancel-attempt").GetInstruction())

	free, err := p.Lower("errand.retry", errandIdentity)
	require.NoError(t, err)
	require.Equal(t, Lowered, free.Standing)
	require.Empty(t, free.Unsupported)
	require.Empty(t, free.OffPath)
	var got [][2]string
	dispositions := map[string]Disposition{}
	for _, e := range free.Inventory {
		got = append(got, [2]string{e.Kind, e.ID})
		dispositions[e.Kind+" "+e.ID] = e.Disposition
	}
	require.ElementsMatch(t, declared(t, m, realizationNamed(t, m, "errandRealization")), got, "the inventory accounts for every declaration")
	require.Equal(t, OffPath, dispositions["command errand/cancel-attempt [answer-canceled]"])
	require.Equal(t, OffPath, dispositions["command controller/await-withdrawn"])
	require.Equal(t, InCase, dispositions["command errand/complete-attempt [answer-completed]"])
}

// A path that ends in a step nothing confirms is an error, however many classes it takes again: the
// retry's path with a worker stop after its last step ends in a step that records nothing evidence
// names, which no later step confirms (the producer's evidence.action-unmapped). The producer reads
// such a path whole now, so the refusal is its own.
func TestAPathThatEndsInAStepNothingConfirmsIsAnError(t *testing.T) {
	m := loaded(t, "activity")
	var stop *umpirespb.ActionClass
	for _, s := range m.GetScenarios() {
		if s.GetName() == "terminatedWhileScheduled" {
			stop = s.GetActions()[1]
		}
	}
	require.NotNil(t, stop)
	for _, s := range m.GetScenarios() {
		if s.GetName() == "retriedThenCompleted" {
			s.Actions = append(s.Actions, stop)
		}
	}
	for _, q := range m.GetQueries() {
		if q.GetName() == "retry" {
			q.GetLimits().Steps, q.GetLimits().Actions = 7, 7
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", cp.IdentityFor("temporal.case", "standaloneActivityTests", "retry"))
	require.Nil(t, l)
	var refused *cp.Error
	require.ErrorAs(t, err, &refused)
	require.Equal(t, &cp.Error{Definition: "temporal.activity.standalone.action.activityProtocol.workerStop", Construct: "evidence.action-unmapped"}, refused)
	require.ErrorContains(t, err, "model/temporal/standaloneactivity/Claims.scala:")
}

// A path that takes a class again with no kind of evidence that names the second step is an error of
// the realization, and no gap: the retry with the second attempt's kind naming only the failure it
// follows leaves the second attempt start to the first attempt's kind, which names the first alone.
func TestAClassTakenAgainThatNoKindNamesIsAnError(t *testing.T) {
	m := loaded(t, "activity")
	for _, e := range m.GetRealizations()[0].GetEvidence() {
		if e.GetId() == "temporal.activity.standalone.evidence.attemptCount" {
			e.Confirms = e.GetConfirms()[:1]
		}
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", cp.IdentityFor("temporal.case", "standaloneActivityTests", "retry"))
	require.Nil(t, l)
	var refused *cp.Error
	require.ErrorAs(t, err, &refused)
	require.Equal(t, &cp.Error{Definition: "temporal.activity.standalone.action.activityProtocol.attemptStart", Construct: "evidence.taking-unrecorded"}, refused)
	// The Queries whose path takes the class once are not touched by it.
	completion, err := p.Lower("completion", cp.IdentityFor("temporal.case", "standaloneActivityTests", "completion"))
	require.NoError(t, err)
	require.Equal(t, Lowered, completion.Standing)
}
