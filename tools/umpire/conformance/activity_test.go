package conformance

// The lowered Cases of the standalone activity Model, replayed on constructed Runs that a live Run
// would not record: evidence out of the path's order, and evidence on a Run Event its source does not
// take. The Run's own record is evidence here, so the Run Event that carries a piece of evidence is
// itself read. The expectations are read off model/temporal/features/activity/standalone
// (system/System.scala, Realization.scala). The Runs the Cases record live are in played_test.go.

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/proto"
)

const (
	activityFamily   = "temporal.features.activity.standalone.system"
	activityMachine  = "activitySystem"
	activityEvidence = "temporal.features.activity.standalone.system.evidence."
	activityRunID    = testpilot.RunIDPrefix + "00000000-0000-4000-8000-000000000001"
)

func activityModel(t testing.TB) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone.json"))
	require.NoError(t, err)
	return m
}

// realizationNamed is the realization a Model declares under a name: a Model declares several.
func realizationNamed(t testing.TB, m *umpirespb.Model, name string) *umpirespb.Realization {
	t.Helper()
	for _, r := range m.GetRealizations() {
		if r.GetName() == name {
			return r
		}
	}
	require.FailNow(t, "no realization "+name)
	return nil
}

// loweredActivity is one Query of the activity Model lowered to its Case, prepared as a black-box
// consumer prepares it, with the Query's assessment bound to it.
func loweredActivity(t testing.TB, m *umpirespb.Model, query string) *bound {
	t.Helper()
	producer, err := lower.NewProducer(m)
	require.NoError(t, err)
	lowering, err := producer.Lower(query, lower.IdentityFor("temporal.case", "standaloneActivityTests", query))
	require.NoError(t, err)
	require.Equal(t, lower.Lowered, lowering.Standing, "%v", lowering.Unsupported)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(lowering.Case, catalog, temporal.Environment{Identity: query + "-profile", Namespace: "namespace",
		TaskQueue: "task-queue", DeliveryControl: true})
	require.NoError(t, err)
	plain, err := testpilot.Prepare(lowering.Case, profile)
	require.NoError(t, err)
	factory, err := Prepare(m, check.ClaimKey{Family: activityFamily, Owner: activityMachine, Name: query}, lowering.Case, generous)
	require.NoError(t, err)
	assessed, err := plain.WithAssessment(factory)
	require.NoError(t, err)
	return &bound{source: lowering.Case, plain: plain, assessed: assessed, factory: factory}
}

// recorded is one Run Event of a constructed activity Run, with the kind of evidence it carries, by
// the last part of the kind's id, or none.
type recorded struct {
	event *testpilotspb.RunEvent
	kind  string
}

// answered is the completion of one of the controller's calls, as the Run records it.
func answered(command string) *testpilotspb.RunEvent {
	return reported(testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, command,
		&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"})
}

// attemptRecord is the Run's record of one attempt the worker was delivered: an activation the start
// call carries.
func attemptRecord(number int32, response testpilotspb.ActivityAttemptResponse) *testpilotspb.RunEvent {
	return reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", attemptOf(number, "token-"+strconv.Itoa(int(number)), response))
}

// statusRead is the event a poll of the activity's description emits for the status it read.
func statusRead(command string) *testpilotspb.RunEvent {
	return &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
		Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", ActivationId: "controller", InstructionId: command, Attempt: 1}}
}

// activityRun writes out a closed, complete Run of a lowered activity Case that recorded these events
// in order. Each piece of evidence is what the Case's own declaration of its kind lifts from the event
// that carries it: under the Case's name, source and scope for the kind, at the next ordinal of its
// source, keyed by the Run, after the piece before it where that came from another source, as
// Testpilot's executor chains what one operation's instructions lift, and with the fields the
// declaration keeps, read from the attempt the event records.
func activityRun(t testing.TB, source *testpilotspb.Case, events ...recorded) *testpilotspb.Run {
	t.Helper()
	local := map[string]string{}
	for _, name := range source.GetProvenance().GetLocalNames() {
		local[name.GetDefinitionId()] = name.GetLocalName()
	}
	run := &testpilotspb.Run{RunId: activityRunID, CaseId: source.GetCaseId(), ProgramId: source.GetProgram().GetProgramId(),
		Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}}
	record := func(event *testpilotspb.RunEvent, id string) {
		event.Sequence = int64(len(run.GetEvents())) + 1
		event.ElapsedMilliseconds = event.GetSequence() - 1
		event.SourceId = id
		run.Events = append(run.Events, event)
	}
	record(&testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED}, "opened")
	ordinals := map[string]int64{}
	var last *testpilotspb.CorrelatedIdentity
	for i, item := range events {
		event := proto.CloneOf(item.event)
		if item.kind != "" {
			name, renamed := local[activityEvidence+item.kind]
			if !renamed {
				name = activityEvidence + item.kind
			}
			var declared *testpilotspb.EvidenceDeclaration
			for _, e := range source.GetProgram().GetEvidence() {
				if e.GetEvidenceId() == name {
					declared = e
				}
			}
			require.NotNil(t, declared, "the Case carries %s", item.kind)
			evidence := &testpilotspb.CorrelatedEvidence{Kind: name, Operation: activityRunID, Identity: &testpilotspb.CorrelatedIdentity{
				EvidenceSource: declared.GetEvidenceSource(), Ordinal: ordinals[declared.GetEvidenceSource()], Scope: declared.GetScope()}}
			ordinals[declared.GetEvidenceSource()]++
			if last != nil && last.GetEvidenceSource() != declared.GetEvidenceSource() {
				evidence.Parents = append(evidence.Parents, proto.CloneOf(last))
			}
			last = evidence.GetIdentity()
			attempt := event.GetOutcome().GetActivityAttempt()
			for _, field := range declared.GetFields() {
				switch field.GetPath() {
				case "activity_attempt.sdk_attempt":
					evidence.Fields = append(evidence.Fields, numberField(field.GetFieldId(), uint64(attempt.GetSdkAttempt())))
				case "activity_attempt.delivery_id":
					evidence.Fields = append(evidence.Fields, textField(field.GetFieldId(), attempt.GetDeliveryId()))
				case "activity_attempt.activity_run_id":
					evidence.Fields = append(evidence.Fields, textField(field.GetFieldId(), attempt.GetActivityRunId()))
				default:
					require.FailNow(t, "a field this builder does not read: "+field.GetPath())
				}
			}
			event.Observations = []*testpilotspb.ObservationResult{{ObservationId: evidenceObservation, Value: evidenceValue(t, evidence)}}
		}
		record(event, "event."+strconv.Itoa(i))
	}
	record(&testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_RUN_CLOSED}, "closed")
	return run
}

// The Contract of the retry's Case authorizes its evidence in the witness's order alone: the second
// attempt's record before the first's is no step the activity can take from where it is, and the
// Contract's own evaluation fails at that event.
func TestTheRetryContractRefusesItsEvidenceOutOfOrder(t *testing.T) {
	b := loweredActivity(t, activityModel(t), "retry")
	run := activityRun(t, b.source, recorded{answered("start-activity"), "statusScheduled"},
		recorded{attemptRecord(2, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), "attemptCount"},
		recorded{attemptRecord(1, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE), "statusStarted"})
	_, _, err := b.plain.Evaluate(t.Context(), run)
	require.ErrorContains(t, err, "event 3:", "the Contract's evaluation fails on the second attempt's record")
}

// The lowering takes two orders from the runtime that no declaration fixes: the completion of the call
// that carries an attempt is recorded before the attempt's record, and an attempt's record before the
// status read after its answer. A Run that records either pair the other way round holds evidence the
// Contract does not authorize where it stands, so its evaluation fails at that event and gives no
// satisfied Verdict: the first attempt's record is no step of an activity that has not started, and a
// completed status is no step of an activity no attempt of which has started.
func TestARunThatReversesAnOrderTheLoweringTakesFromTheRuntimeIsNeverSatisfied(t *testing.T) {
	b := loweredActivity(t, activityModel(t), "completion")
	start := recorded{answered("start-activity"), "statusScheduled"}
	record := recorded{attemptRecord(1, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), "statusStarted"}
	read := recorded{statusRead("await-completed"), "statusCompleted"}

	verdict, _, err := b.plain.Evaluate(t.Context(), activityRun(t, b.source, start, record, read))
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus(), "the Run in the order the lowering takes is satisfied")

	for name, test := range map[string]struct {
		events []recorded
		at     string
	}{
		"the attempt's record before its carrier's completion": {[]recorded{record, start, read}, "event 2:"},
		"the last status read before the attempt's record":     {[]recorded{start, read, record}, "event 3:"},
	} {
		t.Run(name, func(t *testing.T) {
			verdict, _, err := b.plain.Evaluate(t.Context(), activityRun(t, b.source, test.events...))
			require.ErrorContains(t, err, test.at)
			require.NotEqual(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
		})
	}
}

// Evidence that is the Run's own record is read only from a Run Event its source takes: the event of
// the source's kind, at the source's command, that the source's guard holds of
// (TestOnlyADeliveredAttemptIsEvidenceOfAnAttemptStart). Evidence of such a kind on any other event is
// refused at that event, and is never read as the fact it claims; a guard that cannot be evaluated on
// the event is an error and no refusal.
func TestEvidenceOfTheRunsRecordIsReadOnlyFromAnEventItsSourceTakes(t *testing.T) {
	const completed = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
	started := activityEvidence + "statusStarted"
	notTaken := func(kind string) string {
		return "evidence of kind \"" + kind + "\" on a Run Event its source does not take: the Run's record of controller/start-activity under its guard"
	}
	for name, test := range map[string]struct {
		model   func(*umpirespb.Model)
		carrier *testpilotspb.RunEvent
		want    error
	}{
		"the first attempt's record": {carrier: attemptRecord(1, completed)},
		"the second attempt's record, with the first's evidence": {carrier: attemptRecord(2, completed),
			want: &EvidenceError{Event: 3, Message: notTaken("evidence.statusStarted")}},
		// An outcome that records no attempt is the record of none: the source is declared the record of
		// the first attempt, and that is read before its guard.
		"a record of no attempt": {carrier: reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity",
			&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}),
			want: &EvidenceError{Event: 3, Message: notTaken("evidence.statusStarted")}},
		"an attempt another command carries": {carrier: reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "await-completed",
			attemptOf(1, "token-1", completed)), want: &EvidenceError{Event: 3, Message: notTaken("evidence.statusStarted")}},
		"the start call's own completion": {carrier: reported(testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, "start-activity",
			attemptOf(1, "token-1", completed)), want: &EvidenceError{Event: 3, Message: notTaken("evidence.statusStarted")}},
		// Declared the start call's own completion, which is the record of no attempt, and with the
		// guard's first operand gone, the source compares the delivery of an outcome that holds no
		// attempt, which is an error at that event, and not a guard that does not hold.
		"a guard that cannot be evaluated on the event": {
			model: func(m *umpirespb.Model) {
				for _, e := range realizationNamed(t, m, "standalone").GetEvidence() {
					if e.GetId() == started {
						e.GetRunEvent().Kind, e.GetRunEvent().Attempt = umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED, nil
						all := e.GetRunEvent().GetGuard().GetAll()
						all.Operands = all.GetOperands()[1:]
					}
				}
			},
			carrier: reported(testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, "start-activity",
				&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}),
			want: &GuardError{Event: 3, Message: "compares an absent value"}},
	} {
		t.Run(name, func(t *testing.T) {
			m := activityModel(t)
			if test.model != nil {
				test.model(m)
			}
			b := loweredActivity(t, m, "completion")
			run := activityRun(t, b.source, recorded{answered("start-activity"), "statusScheduled"}, recorded{test.carrier, "statusStarted"},
				recorded{statusRead("await-completed"), "statusCompleted"})
			assessor, err := b.factory.New(t.Context())
			require.NoError(t, err)
			var failed error
			for _, event := range run.GetEvents() {
				if _, failed = assessor.Observe(t.Context(), event); failed != nil {
					break
				}
			}
			if test.want == nil {
				require.NoError(t, failed)
				return
			}
			require.Equal(t, test.want, failed)
		})
	}
}
