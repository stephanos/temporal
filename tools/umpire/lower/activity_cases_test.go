package lower

// The Cases of the standalone activity Model. Every expectation is read off
// model/temporal/standaloneactivity: Claims.scala's Scenarios say which classes a path
// takes, Model.scala's protocol machine what each step records, and Realization.scala which command
// performs a class, which command a Case carries for one, and which kind of evidence confirms a step.

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
)

const (
	activityEvidence = "temporal.activity.standalone.evidence."
	activityActions  = "temporal.activity.standalone.action.activityProtocol."
	describeActivity = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
)

// activityCase is what one Query of the activity Model lowers to: the instructions of its controller
// and of its activity, in order; the classes of the steps one piece of each kind of evidence confirms,
// in path order; and the classes that record nothing evidence names, which a Case lists as Known Gaps.
type activityCase struct {
	controller, activity []string
	confirmed            map[string][]string
	silent               []string
}

const (
	plainStart   = "start-unset-unset-unset"
	startedOnce  = "attemptStart"
	answerDone   = "attemptResult-completed"
	answerFailed = "attemptResult-failed-true"
)

// The first three steps of most paths: the start call's answer says the activity is scheduled, and the
// first attempt the worker is delivered that it is started.
func begun(rest map[string][]string) map[string][]string {
	out := map[string][]string{"statusScheduled": {plainStart}, "statusStarted": {startedOnce}}
	maps.Copy(out, rest)
	return out
}

var activityCases = map[string]activityCase{
	// completed: start, attemptStart, attemptResult(completed).
	"completion": {[]string{"start-activity", "await-completed"}, []string{"complete-attempt"},
		begun(map[string][]string{"statusCompleted": {answerDone}}), nil},
	// nonRetryable: start, attemptStart, attemptResult(failed(false)).
	"nonRetryableFailure": {[]string{"start-activity", "await-failed"}, []string{"fail-activity"},
		begun(map[string][]string{"statusFailed": {"attemptResult-failed-false"}}), nil},
	// retriedThenCompleted: start, attemptStart, attemptResult(failed(true)), backoff, attemptStart,
	// attemptResult(completed). The second attempt's delivery is one piece of evidence: it confirms the
	// failure the server retried, the backoff, which records nothing, and the second attempt start.
	"retry": {[]string{"start-activity", "await-completed"}, []string{"fail-attempt", "complete-attempt"},
		begun(map[string][]string{"attemptCount": {answerFailed, "backoff", startedOnce}, "statusCompleted": {answerDone}}), []string{"backoff"}},
	// terminatedWhileScheduled: start, workerStop, control(terminate). The stop records nothing.
	"terminate": {[]string{"stop-worker", "start-activity", "terminate-activity", "await-terminated"}, []string{},
		map[string][]string{"statusScheduled": {plainStart}, "statusTerminated": {"workerStop", "control-terminate"}}, []string{"workerStop"}},
	// pausedThenCompleted: start, control(pause), control(unpause), attemptStart, attemptResult(completed).
	// The release schedules the activity again, which its own answer confirms. The path pauses an
	// activity no worker has taken, so the Case keeps its worker from polling until the release.
	"pauseResume": {[]string{"stop-worker-until-released", "start-activity", "pause-activity", "await-paused", "unpause-activity", "resume-worker",
		"await-completed"}, []string{"complete-attempt"},
		begun(map[string][]string{"statusPaused": {"control-pause"}, "statusScheduledAgain": {"control-unpause"}, "statusCompleted": {answerDone}}), nil},
	// scheduleToStartExpires: start(unset, expires, unset), workerStop, scheduleToStart.
	"scheduleToStartTimeout": {[]string{"stop-worker", "start-activity", "await-timed-out"}, []string{},
		map[string][]string{"statusScheduled": {"start-unset-expires-unset"}, "statusTimedOut": {"workerStop", "scheduleToStart"}}, []string{"workerStop"}},
}

// activityLimits is, for each Query that lowers to no Case, the one thing that keeps it from one, and
// the text of the line of Realization.scala it is named at.
//
//   - startToCloseTimeout (start, attemptStart, startToClose) starts an attempt and lets it run out
//     its deadline, so the attempt is given no answer, and an activity entrypoint's instructions are
//     answers: nothing waits.
//   - cancel and cancelRequest (start, attemptStart, control(requestCancel), attemptResult(canceled))
//     request the cancellation while the attempt is held. A Run records an attempt once it is answered,
//     so the record that confirms the attempt start reaches the Run after the cancel request's answer,
//     which confirms the step after it: the Run would carry the path's evidence out of the path's order.
var activityLimits = map[string]struct {
	gap     Unsupported
	written string
}{
	"startToCloseTimeout": {Unsupported{Construct: "attempt that gives no answer", ID: "activity", Owner: "none: a recorded limit of the prototype"},
		"activityScript = Script("},
	"cancel": {Unsupported{Construct: "attempt record that follows later evidence", ID: activityEvidence + "statusStarted",
		Owner: "none: a recorded limit of the prototype"}, "Evidence.runEvent("},
	"cancelRequest": {Unsupported{Construct: "attempt record that follows later evidence", ID: activityEvidence + "statusStarted",
		Owner: "none: a recorded limit of the prototype"}, "Evidence.runEvent("},
}

func activityIdentity(query string) cp.Identity {
	return cp.IdentityFor("temporal.case", "standaloneActivityTests", query)
}

// definitions maps each of a Case's own names to the Definition ID it stands for.
func definitions(c *testpilotspb.Case) map[string]string {
	out := map[string]string{}
	for _, n := range c.GetProvenance().GetLocalNames() {
		out[n.GetLocalName()] = n.GetDefinitionId()
	}
	return out
}

func defined(names map[string]string, local string) string {
	if id, ok := names[local]; ok {
		return id
	}
	return local
}

// Every find Query of the activity Model lowers to a Case, or names the one thing that keeps it from
// one. Six lower: the completion, the retry and the pause and resume among them. Each Case carries the
// commands its path performs and the reads its path's classes call for, confirms each step by its own
// evidence, prepares unchanged under the Profile derived from it, as any black-box consumer prepares a
// Case, and is the same bytes when the Model is read and lowered again.
//
// Three do not lower, each for one limit no task owns, named where the realization declares what
// meets it (activityLimits).
func TestEveryQueryOfTheActivityModelLowersOrNamesItsLimit(t *testing.T) {
	m := loaded(t, "activity")
	p, err := NewProducer(m)
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	require.ElementsMatch(t, finds(m), slices.Concat(slices.Collect(maps.Keys(activityCases)), slices.Collect(maps.Keys(activityLimits))),
		"every find Query of the Model is accounted for")

	for query, want := range activityCases {
		t.Run(query, func(t *testing.T) {
			l, err := p.Lower(query, activityIdentity(query))
			require.NoError(t, err)
			require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
			require.Empty(t, l.Unsupported)
			require.Empty(t, l.OffPath)
			c := l.Case
			require.Equal(t, map[string][]string{"controller": want.controller, "activity": want.activity}, instructionIDs(c))

			// The inventory accounts for everything the realization declares, as its message tree lists it.
			var inventory [][2]string
			for _, e := range l.Inventory {
				inventory = append(inventory, [2]string{e.Kind, e.ID})
				require.Contains(t, e.Position, activityRealizationAt)
			}
			require.ElementsMatch(t, declared(t, m, m.GetRealizations()[0]), inventory)

			names := definitions(c)
			confirmed := map[string][]string{}
			for _, rule := range c.GetContract().GetCorrelated().GetProjectionRules() {
				require.Equal(t, testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED, rule.GetMeaning())
				steps := []string{}
				for _, output := range rule.GetOutputs() {
					steps = append(steps, output.GetAction().GetValue())
				}
				confirmed[strings.TrimPrefix(defined(names, rule.GetKind()), activityEvidence)] = steps
			}
			require.Equal(t, want.confirmed, confirmed)

			var silent []string
			for _, gap := range c.GetProvenance().GetKnownGaps() {
				silent = append(silent, strings.TrimPrefix(gap.GetSubject(), activityActions))
			}
			require.Equal(t, want.silent, silent)

			encoded, err := protojson.Marshal(c)
			require.NoError(t, err)
			source, err := testpilot.DecodeCaseProtoJSON(encoded)
			require.NoError(t, err)
			profile, err := temporal.DeriveProfile(source, catalog, temporal.Environment{Identity: query + "-profile", Namespace: "namespace",
				TaskQueue: "task-queue"})
			require.NoError(t, err)
			prepared, err := testpilot.Prepare(source, profile)
			require.NoError(t, err)
			require.True(t, proto.Equal(source, prepared.Snapshot()), "preparation carries the Case unchanged")

			again, err := NewProducer(loaded(t, "activity"))
			require.NoError(t, err)
			second, err := again.Lower(query, activityIdentity(query))
			require.NoError(t, err)
			first, err := proto.MarshalOptions{Deterministic: true}.Marshal(c)
			require.NoError(t, err)
			repeated, err := proto.MarshalOptions{Deterministic: true}.Marshal(second.Case)
			require.NoError(t, err)
			require.Equal(t, first, repeated)
		})
	}

	for query, want := range activityLimits {
		t.Run(query, func(t *testing.T) {
			l, err := p.Lower(query, activityIdentity(query))
			require.NoError(t, err)
			require.Equal(t, NotSupported, l.Standing)
			require.Nil(t, l.Case)
			require.Empty(t, l.OffPath)
			require.Len(t, l.Unsupported, 1)
			gap := l.Unsupported[0]
			require.NotEmpty(t, gap.Why)
			require.Contains(t, gap.Position, activityRealizationAt)
			require.Contains(t, lineOf(t, gap.Position), want.written)
			gap.Why, gap.Position = "", ""
			require.Equal(t, want.gap, gap)
		})
	}
}

// The retry's Case, part by part. Its two attempts are the activity entrypoint's two instructions in
// path order, the failing one an attempt failure the server retries and never a result. The second
// attempt's evidence is the Run's own record of the attempt the worker was delivered under the number
// 2, at the start call's instruction and keyed by the Run, with the attempt, the delivery and the
// activity run kept as fields; the status the activity ends in is the one message Describe returns.
func TestTheRetryCaseFailsItsFirstAttemptAndReadsItsSecondFromTheRunsRecord(t *testing.T) {
	p, err := NewProducer(loaded(t, "activity"))
	require.NoError(t, err)
	l, err := p.Lower("retry", activityIdentity("retry"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
	c := l.Case

	protorequire.ProtoEqual(t, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{
		ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{Failure: &failurepb.Failure{Message: "attempt failed",
			FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "AttemptFailed"}}}}}},
		instruction(t, c, "activity", "fail-attempt").GetInstruction())
	protorequire.ProtoEqual(t, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{
		Finish: &testpilotspb.Finish{Result: cp.Literal(cp.Text("done"))}}}, instruction(t, c, "activity", "complete-attempt").GetInstruction())

	names := definitions(c)
	declared := map[string]*testpilotspb.EvidenceDeclaration{}
	for _, d := range c.GetProgram().GetEvidence() {
		bare := proto.CloneOf(d)
		bare.EvidenceId, bare.EvidenceSource, bare.Scope = "", "", nil
		declared[strings.TrimPrefix(defined(names, d.GetEvidenceId()), activityEvidence)] = bare
	}
	require.ElementsMatch(t, []string{"statusScheduled", "statusStarted", "attemptCount", "statusCompleted"}, slices.Collect(maps.Keys(declared)))

	all := func(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: operands}}}
	}
	attempt := func(number int64) *testpilotspb.EvidenceDeclaration {
		projected := func(path string) *testpilotspb.Expression { return cp.Path(cp.ProjectedValue(), path) }
		return &testpilotspb.EvidenceDeclaration{
			Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{
				Kind:        testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
				Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "start-activity"},
				RunKeyed:    true,
				// The guard the realization writes, a delivered attempt, and then the attempt the record is
				// declared of, which the lowering states so that the runtime selects the record by it.
				Guard: all(
					all(cp.Present(projected("activity_attempt")), &testpilotspb.Expression{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{
						Operand: cp.Equal(projected("activity_attempt.delivery_id"), cp.Literal(cp.Text("")))}}}),
					cp.Equal(projected("activity_attempt.sdk_attempt"), cp.Literal(cp.SignedInteger(number))))}},
			Fields: []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "attempt", Path: "activity_attempt.sdk_attempt"},
				{FieldId: "delivery", Path: "activity_attempt.delivery_id"}, {FieldId: "activityRun", Path: "activity_attempt.activity_run_id"}},
		}
	}
	protorequire.ProtoEqual(t, attempt(1), declared["statusStarted"])
	protorequire.ProtoEqual(t, attempt(2), declared["attemptCount"])
	protorequire.ProtoEqual(t, &testpilotspb.EvidenceDeclaration{Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{
		Kind:        testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
		Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "start-activity"},
		RunKeyed:    true,
		Guard:       cp.Equal(cp.Path(cp.ProjectedValue(), "status"), cp.Literal(cp.Enum("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED")))}}},
		declared["statusScheduled"])
	protorequire.ProtoEqual(t, &testpilotspb.EvidenceDeclaration{Operation: "activity_id",
		Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: describeActivity, Path: "info", Single: true}}},
		declared["statusCompleted"])

	retained := func(id string, kind testpilotspb.ScalarKind) *testpilotspb.CorrelatedFieldPolicy {
		return &testpilotspb.CorrelatedFieldPolicy{FieldId: id, Type: &testpilotspb.ScalarType{Kind: kind}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}
	}
	identity := []*testpilotspb.CorrelatedFieldPolicy{retained("attempt", testpilotspb.SCALAR_KIND_UINT64), retained("delivery", testpilotspb.SCALAR_KIND_TEXT),
		retained("activityRun", testpilotspb.SCALAR_KIND_TEXT)}
	for _, rule := range c.GetContract().GetCorrelated().GetProjectionRules() {
		switch strings.TrimPrefix(defined(names, rule.GetKind()), activityEvidence) {
		case "statusStarted", "attemptCount":
			protorequire.ProtoSliceEqual(t, identity, rule.GetFields())
		default:
			require.Empty(t, rule.GetFields(), rule.GetKind())
		}
	}
}

// A Run records an attempt once it is answered, so the record of an attempt is evidence in the path's
// order only where no step between the last step it confirms and that attempt's answer has evidence of
// its own: evidence of such a step reaches the Run first (published_test.go has the rule). The paths
// here are the activity Model's own.
//
// The cancel paths request the cancellation while the first attempt is held, and so does a completion
// path with a pause between the attempt start and its answer (the protocol machine takes a pause of a
// held attempt as a pause request, which the caller reads as paused). Where the answer follows the
// attempt start, or only a step that records nothing lies between, the record is in order: the retry's
// second attempt confirms the failure, the backoff and the second attempt start, and its answer is the
// next step.
func TestAnAttemptRecordThatFollowsLaterEvidenceIsNamed(t *testing.T) {
	const started = activityEvidence + "statusStarted"
	// between puts steps of other Scenarios between the last attempt start and the answer of a Query's
	// Scenario, each named by the Scenario it is taken from and its position there.
	type taken struct {
		scenario string
		at       int
	}
	between := func(query, scenario string, steps ...taken) func(*umpirespb.Model) {
		return func(m *umpirespb.Model) {
			classes := map[string][]*umpirespb.ActionClass{}
			for _, s := range m.GetScenarios() {
				classes[s.GetName()] = s.GetActions()
			}
			for _, s := range m.GetScenarios() {
				if s.GetName() != scenario {
					continue
				}
				last := len(s.GetActions()) - 1
				actions := slices.Clone(s.GetActions()[:last])
				for _, step := range steps {
					actions = append(actions, classes[step.scenario][step.at])
				}
				s.Actions = append(actions, s.GetActions()[last])
			}
			for _, q := range m.GetQueries() {
				if q.GetName() == query {
					q.GetLimits().Steps, q.GetLimits().Actions = 8, 8
				}
			}
		}
	}
	pause, stop := taken{"pausedThenCompleted", 1}, taken{"terminatedWhileScheduled", 1}
	for name, test := range map[string]struct {
		query  string
		change func(*umpirespb.Model)
		want   []string
	}{
		"completion":             {"completion", nil, nil},
		"nonRetryableFailure":    {"nonRetryableFailure", nil, nil},
		"retry":                  {"retry", nil, nil},
		"pauseResume":            {"pauseResume", nil, nil},
		"terminate":              {"terminate", nil, nil},
		"scheduleToStartTimeout": {"scheduleToStartTimeout", nil, nil},
		"startToCloseTimeout":    {"startToCloseTimeout", nil, nil},
		"cancel":                 {"cancel", nil, []string{started}},
		"cancelRequest":          {"cancelRequest", nil, []string{started}},
		// A pause of a held attempt is a pause request, which the caller reads as paused.
		"a pause between an attempt and its answer": {"completion", between("completion", "completed", pause), []string{started}},
		// A worker stop records nothing, so nothing is recorded before the attempt's record.
		"a step that records nothing between an attempt and its answer": {"completion", between("completion", "completed", stop), nil},
		// The record of the second attempt is late where evidence lies before the second answer.
		"a pause between the second attempt and its answer": {"retry", between("retry", "retriedThenCompleted", pause),
			[]string{activityEvidence + "attemptCount"}},
	} {
		t.Run(name, func(t *testing.T) {
			m := loaded(t, "activity")
			if test.change != nil {
				test.change(m)
			}
			m, err := umpiremodel.WithTotals(m)
			require.NoError(t, err)
			p, err := NewProducer(m)
			require.NoError(t, err)
			a, _, err := p.ask(test.query)
			require.NoError(t, err)
			// The rule reads the path and the declarations, whatever else the producer makes of the path.
			l, _ := p.check(a, activityIdentity(test.query))
			require.NotNil(t, l)
			require.NotEmpty(t, l.confirmations, "the producer says which kind confirms each step")
			var late []string
			for _, gap := range l.late() {
				late = append(late, gap.ID)
			}
			require.Equal(t, test.want, late)
		})
	}
}

// An attempt with no answer is told from the path and the scripts alone: a path that takes the class
// an activity's script starts with more often than it takes the classes the script's commands perform
// leaves an attempt unanswered. The count is per script, and a path whose every attempt is answered,
// or that starts none, has no such gap.
func TestAnAttemptIsUnansweredWhereThePathStartsMoreThanItAnswers(t *testing.T) {
	m := loaded(t, "activity")
	p, err := NewProducer(m)
	require.NoError(t, err)
	for query, want := range map[string]int{"completion": 0, "retry": 0, "terminate": 0, "startToCloseTimeout": 1} {
		t.Run(query, func(t *testing.T) {
			a, _, err := p.ask(query)
			require.NoError(t, err)
			l, problems := p.check(a, activityIdentity(query))
			require.Empty(t, problems)
			require.Len(t, l.unanswered(), want)
		})
	}
}

// What a realization declares of evidence that the Case does not carry as declared is an error of the
// inventory, for each thing a kind can declare: where it is recorded, the fields it keeps, and the
// steps it confirms.
func TestTheInventoryOfAnActivityCaseDoesNotCloseOverAChangedDeclaration(t *testing.T) {
	const started = activityEvidence + "statusStarted"
	kind := func(r *umpirespb.Realization, id string) *umpirespb.Evidence {
		for _, e := range r.GetEvidence() {
			if e.GetId() == id {
				return e
			}
		}
		return nil
	}
	for name, test := range map[string]struct {
		change func(e *umpirespb.Evidence)
		want   string
	}{
		"another command's record": {func(e *umpirespb.Evidence) { e.GetRunEvent().Command = "pause-activity" },
			`is "run event KIND_DIAGNOSTIC of controller/pause-activity keyed by the run"`},
		"another guard": {func(e *umpirespb.Evidence) {
			e.GetRunEvent().GetGuard().GetAll().Operands = e.GetRunEvent().GetGuard().GetAll().GetOperands()[:1]
		}, "evidence " + started + " is the Run's record under a guard, and the Case's program.evidence[evidence.statusStarted] declares it under another"},
		"another attempt": {func(e *umpirespb.Evidence) { e.GetRunEvent().GetAttempt().Number = 2 },
			"evidence " + started + " is the Run's record under a guard, and the Case's program.evidence[evidence.statusStarted] declares it under another"},
		"no guard": {func(e *umpirespb.Evidence) { e.GetRunEvent().Guard = nil },
			"evidence " + started + " is the Run's record under a guard, and the Case's program.evidence[evidence.statusStarted] declares it under another"},
		"a field fewer than the Case keeps": {func(e *umpirespb.Evidence) { e.Fields = e.GetFields()[:2] },
			"evidence " + started + " keeps 2 fields, and the Case's program.evidence[evidence.statusStarted] keeps 3"},
		"a field carried without its value": {func(e *umpirespb.Evidence) { e.GetFields()[2].Redacted = true },
			"field activityRun of evidence " + started + " of realization standalone is carried without its value, which is in no part of the Case"},
		"a field the Case does not keep": {func(e *umpirespb.Evidence) {
			e.Fields = append(e.Fields, &umpirespb.EvidenceField{Id: "offered", Path: "activity_attempt.activity_run_id"})
		}, "evidence " + started + " keeps field offered at activity_attempt.activity_run_id, and the Case's program.evidence[evidence.statusStarted] does not"},
		"a field at another path": {func(e *umpirespb.Evidence) { e.GetFields()[0].Path = "activity_attempt.delivery_id" },
			"evidence " + started + " keeps field attempt at activity_attempt.delivery_id, and the Case's program.evidence[evidence.statusStarted] does not"},
		"a step it does not confirm": {func(e *umpirespb.Evidence) {
			e.Confirms = append(e.Confirms, proto.CloneOf(e.GetConfirms()[0]))
			e.GetConfirms()[1].Occurrence = 2
		}, "evidence " + started + " confirms 2 steps, and the Case's rule for it confirms 1"},
	} {
		t.Run(name, func(t *testing.T) {
			m := loaded(t, "activity")
			p, err := NewProducer(m)
			require.NoError(t, err)
			a, _, err := p.ask("completion")
			require.NoError(t, err)
			l, problems := p.check(a, activityIdentity("completion"))
			require.Empty(t, problems)
			produced, err := cp.Produce(l.query, activityIdentity("completion"), l.realization, p.source(a.q))
			require.NoError(t, err)
			_, err = l.inventory(produced)
			require.NoError(t, err)
			test.change(kind(l.a.r, started))
			_, err = l.inventory(produced)
			require.ErrorContains(t, err, test.want)
			require.ErrorContains(t, err, activityRealizationAt)
		})
	}
}

// A field evidence keeps is the scalar its descriptor makes it, which is what the Contract's field
// policy states: a text, a flag, or an unsigned integer for every integer kind. Anything else, and
// several values, is refused where the field is written.
func TestAFieldOfEvidenceIsCarriedAsTheScalarItsDescriptorMakesIt(t *testing.T) {
	request, err := messageNamed(nil, "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryRequest")
	require.NoError(t, err)
	e := &umpirespb.Evidence{Id: "evidence", Position: &umpirespb.Position{File: "Realization.scala", Line: 1}}
	for path, want := range map[string]testpilotspb.ScalarKind{
		"namespace":                 testpilotspb.SCALAR_KIND_TEXT,
		"execution.workflow_id":     testpilotspb.SCALAR_KIND_TEXT,
		"wait_new_event":            testpilotspb.SCALAR_KIND_BOOLEAN,
		"maximum_page_size":         testpilotspb.SCALAR_KIND_UINT64,
		"next_page_token":           testpilotspb.SCALAR_KIND_UNSPECIFIED,
		"history_event_filter_type": testpilotspb.SCALAR_KIND_UNSPECIFIED,
		"execution":                 testpilotspb.SCALAR_KIND_UNSPECIFIED,
	} {
		t.Run(path, func(t *testing.T) {
			kind, err := carriedField(e, &umpirespb.EvidenceField{Id: "field", Path: path}, request)
			if want == testpilotspb.SCALAR_KIND_UNSPECIFIED {
				require.ErrorContains(t, err, "evidence evidence: field field reads "+path+", which is no single text, flag or integer of "+string(request.FullName()))
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, kind)
		})
	}
}

// The identity a field names reaches a Case in the fingerprint of its projection alone: the same Case
// lowered from a realization whose field names no attempt differs in that fingerprint and in nothing
// else of its Program and its Contract.
func TestTheIdentityAFieldNamesIsInTheProjectionFingerprint(t *testing.T) {
	lower := func(change func(*umpirespb.Realization)) *testpilotspb.Case {
		m := loaded(t, "activity")
		change(m.GetRealizations()[0])
		p, err := NewProducer(m)
		require.NoError(t, err)
		l, err := p.Lower("completion", activityIdentity("completion"))
		require.NoError(t, err)
		require.Equal(t, Lowered, l.Standing)
		return l.Case
	}
	declared := lower(func(*umpirespb.Realization) {})
	unnamed := lower(func(r *umpirespb.Realization) {
		for _, e := range r.GetEvidence() {
			if e.GetId() == activityEvidence+"statusStarted" {
				require.Equal(t, umpirespb.EvidenceField_ROLE_ATTEMPT, e.GetFields()[0].GetRole())
				e.GetFields()[0].Role = umpirespb.EvidenceField_ROLE_UNSPECIFIED
			}
		}
	})
	require.NotEqual(t, declared.GetContract().GetCorrelated().GetProjectionFingerprint(), unnamed.GetContract().GetCorrelated().GetProjectionFingerprint())
	protorequire.ProtoEqual(t, declared.GetProgram(), unnamed.GetProgram())
	unnamed.GetContract().GetCorrelated().ProjectionFingerprint = declared.GetContract().GetCorrelated().GetProjectionFingerprint()
	protorequire.ProtoEqual(t, declared.GetContract(), unnamed.GetContract())
}

// The Run's own record is the record of a controller's instruction: a Run records events under the
// coordinates of the controller's instructions and of no other script's. Evidence that is the record
// of a command of the activity's script is refused where it is written. A record keyed by a path of
// its payload, and not by the Run, lowers to a declaration that names that path as its operation key.
func TestTheRunsRecordIsOfAControllersInstructionKeyedByTheRunOrByItsPayload(t *testing.T) {
	const scheduled = activityEvidence + "statusScheduled"
	changed := func(change func(*umpirespb.RunEventSource)) *Producer {
		m := loaded(t, "activity")
		for _, e := range m.GetRealizations()[0].GetEvidence() {
			if e.GetId() == scheduled {
				change(e.GetRunEvent())
			}
		}
		p, err := NewProducer(m)
		require.NoError(t, err)
		return p
	}

	l, err := changed(func(source *umpirespb.RunEventSource) {
		source.Script, source.Command = "activity", "complete-attempt"
	}).
		Lower("completion", activityIdentity("completion"))
	require.Nil(t, l)
	require.ErrorContains(t, err, "evidence "+scheduled+" is the Run's record of a command of script activity, which no controller runs")
	require.ErrorContains(t, err, activityRealizationAt)

	l, err = changed(func(source *umpirespb.RunEventSource) {
		source.Key = &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "protocol_code",
			Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &umpirespb.Empty{}}}}}}
	}).Lower("completion", activityIdentity("completion"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
	names := definitions(l.Case)
	for _, d := range l.Case.GetProgram().GetEvidence() {
		if defined(names, d.GetEvidenceId()) == scheduled {
			require.Equal(t, "protocol_code", d.GetOperation())
			require.False(t, d.GetRunEvent().GetRunKeyed())
			return
		}
	}
	require.FailNow(t, "the Case declares "+scheduled)
}

// Evidence that is the Run's own record is the record of an instruction, which is part of what
// carries the kind: a Case that declares the kind and carries no such instruction does not close.
func TestARecordWhoseInstructionTheCaseDoesNotCarryDoesNotClose(t *testing.T) {
	p, err := NewProducer(loaded(t, "activity"))
	require.NoError(t, err)
	a, _, err := p.ask("completion")
	require.NoError(t, err)
	l, problems := p.check(a, activityIdentity("completion"))
	require.Empty(t, problems)
	produced, err := cp.Produce(l.query, activityIdentity("completion"), l.realization, p.source(a.q))
	require.NoError(t, err)
	inventory, err := l.inventory(produced)
	require.NoError(t, err)
	for _, e := range inventory {
		if e.Kind == "evidence" && e.ID == activityEvidence+"statusScheduled" {
			require.Contains(t, e.As, "program.entrypoints[controller].instructions[start-activity]")
		}
	}
	controller := produced.GetProgram().GetEntrypoints()[0]
	require.Equal(t, "start-activity", controller.GetInstructions()[0].GetInstructionId())
	controller.Instructions = controller.GetInstructions()[1:]
	_, err = l.inventory(produced)
	require.ErrorContains(t, err, "evidence "+activityEvidence+"statusScheduled is the Run's record of controller/start-activity, and the Case carries no such instruction")
	require.ErrorContains(t, err, activityRealizationAt)
}

// A pause is read back only of an activity no worker has taken. With a running worker the first
// attempt is delivered, and may be answered, before the pause lands, and the pause of a held attempt is
// a request whose release schedules nothing. So the Case of a path that pauses stops its worker's
// polling before the start and resumes it after the release, on the task queue its activity runs on;
// a path that does not pause does neither.
func TestAPathThatPausesKeepsItsWorkerFromPollingUntilTheRelease(t *testing.T) {
	p, err := NewProducer(loaded(t, "activity"))
	require.NoError(t, err)
	l, err := p.Lower("pauseResume", activityIdentity("pauseResume"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
	queue := role(t, l.Case, testpilotspb.ROLE_KIND_TASK_QUEUE).GetRoleId()
	require.Equal(t, queue, l.Case.GetProgram().GetEntrypoints()[1].GetActivity().GetTaskQueueRoleId())
	fault := func(id string) *testpilotspb.InjectFault {
		return instruction(t, l.Case, "controller", id).GetInstruction().GetInjectFault()
	}
	protorequire.ProtoEqual(t, &testpilotspb.InjectFault{RoleId: queue, Kind: testpilotspb.FAULT_KIND_WORKER_STOP}, fault("stop-worker-until-released"))
	protorequire.ProtoEqual(t, &testpilotspb.InjectFault{RoleId: queue, Kind: testpilotspb.FAULT_KIND_WORKER_RESUME}, fault("resume-worker"))

	for _, query := range []string{"completion", "retry", "terminate"} {
		other, err := p.Lower(query, activityIdentity(query))
		require.NoError(t, err)
		require.NotContains(t, instructionIDs(other.Case)["controller"], "stop-worker-until-released", query)
		require.NotContains(t, instructionIDs(other.Case)["controller"], "resume-worker", query)
	}
}
