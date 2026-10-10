package conformance

// The lowered Case of the held race, run through Testpilot's own executor and recorder against a
// Driver that plays the server, and the Runs it records replayed. No server is involved: what is
// exercised is what the Case and the assessment make of each thing a server and a Driver can do at the
// release, the one instruction that delivers the stale message and records what admission committed.
// The expectations are read off system/Record.scala (heldDispatch, staleDeliveryRejected), Realization.scala
// (heldDelivery) and specimens/activity.md (A1, A1', A10).

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/common/testing/testpilot/duration"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	raceFamily   = "temporal.features.activity.standalone.system"
	raceMachine  = "heldDispatch"
	raceQuery    = "heldDispatch.staleDelivery"
	raceProperty = "staleDeliveryRejected"
)

func raceModel(t testing.TB) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone-race.json"))
	require.NoError(t, err)
	return m
}

// loweredRace is the held race lowered to its Case and prepared under a Profile whose environment
// supplies a delivery control, with the Query's assessment bound to it.
func loweredRace(t testing.TB) *bound {
	t.Helper()
	m := raceModel(t)
	producer, err := lower.NewProducer(m)
	require.NoError(t, err)
	lowering, err := producer.Lower(raceQuery, lower.IdentityFor("temporal.case", "standaloneActivityRace", raceQuery))
	require.NoError(t, err)
	require.Equal(t, lower.Lowered, lowering.Standing, "%v", lowering.Unsupported)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(lowering.Case, catalog, temporal.Environment{Identity: "race-profile", Namespace: "namespace",
		TaskQueue: "task-queue", DeliveryControl: true})
	require.NoError(t, err)
	plain, err := testpilot.Prepare(lowering.Case, profile)
	require.NoError(t, err)
	factory, err := Prepare(m, check.ClaimKey{Family: raceFamily, Owner: raceMachine, Name: raceQuery}, lowering.Case, generous)
	require.NoError(t, err)
	assessed, err := plain.WithAssessment(factory)
	require.NoError(t, err)
	return &bound{source: lowering.Case, plain: plain, assessed: assessed, factory: factory}
}

// raceDriver plays the held race against no server: every call of the controller is answered, the
// status its poll reads is paused, the hold is realized, and the release ends as the row says.
type raceDriver struct {
	identity testpilot.DriverIdentity
	hold     func() effect
	release  func(runID string) effect
}

func (d *raceDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (d *raceDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *raceDriver) Open(_ context.Context, runID string, _ testpilot.PreparedProgram) (testpilot.Session, error) {
	return &raceSession{driver: d, runID: runID}, nil
}

type raceSession struct {
	driver *raceDriver
	runID  string
}

func (*raceSession) Reserve(context.Context, testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	return nil, errUnscripted
}
func (*raceSession) InvokeRPC(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	return succeeded(dynamicpb.NewMessage(method.Output())), nil
}
func (s *raceSession) PollRPC(ctx context.Context, at testpilot.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message, _ time.Duration,
	accepts testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	response := &workflowservice.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{ActivityId: s.runID,
		Status: enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED}}
	if accepted, err := accepts(ctx, response); err != nil || !accepted {
		return nil, errUnscripted
	}
	return succeeded(response), nil
}
func (*raceSession) InvokeHandle(context.Context, testpilot.Coordinate, testpilot.OpaqueHandle, proto.Message) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (s *raceSession) InjectFault(_ context.Context, _ testpilot.Coordinate, _ string, kind testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if kind == testpilotspb.FAULT_KIND_DELIVERY_HOLD {
		return s.driver.hold(), nil
	}
	return s.driver.release(s.runID), nil
}
func (*raceSession) Bridge(context.Context) (testpilot.HandleBridge, error)   { return nil, nil }
func (*raceSession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*raceSession) Close(context.Context) error                              { return nil }
func (*raceSession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

func outcome(status testpilotspb.InstructionOutcomeStatus, code string, admission *testpilotspb.DeliveryAdmission) effect {
	return effect{Outcome: &testpilotspb.InstructionOutcome{Status: status, ProtocolCode: code, DeliveryAdmission: admission}}
}

func decided(activity string, decision testpilotspb.DeliveryAdmissionDecision, attempt int32) effect {
	return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, "", &testpilotspb.DeliveryAdmission{ActivityId: activity,
		ActivityRunId: "activity-run", DeliveryId: "1", Decision: decision, Attempt: attempt})
}

func raceClaim(t *testing.T, assessment *testpilot.Assessment, id string) testpilot.PropertyAssessment {
	t.Helper()
	for _, property := range assessment.Properties {
		if property.ID == id {
			return property
		}
	}
	require.FailNow(t, "the assessment reads no claim "+id)
	return testpilot.PropertyAssessment{}
}

// What a Run of the held race concludes follows from what its release recorded, by one precedence,
// and the recorded Run replays to the same Verdict and the same assessment whole:
//
//   - The server rejects the stale message (A1'): the commit record is the evidence of the rejection,
//     the Contract is satisfied, the corrected design explains the Run, and the claim holds on every
//     execution, since the release's record of admissions is exhaustive and holds none.
//   - The server admits it, as the stale design would (A1): the commit record is evidence of an
//     admission after the pause, which no execution of the corrected design takes. The Run does not
//     conform, and nothing reads as satisfied.
//   - The decision is not observed, as when admission committed and its answer timed out or was lost
//     (A7, A8): the release is no success and records no decision, so its record closes nothing.
//     Nothing is concluded: no admission is inferred, and none is excluded.
//   - The Driver could not realize the hold: nothing after it runs, the Run records no evidence, and
//     nothing is concluded, of conformance either.
//   - The record names another activity's decision: it is keyed to that activity, where the Contract
//     refuses it as a step no activity takes first, and is never evidence of this one's admission.
func TestTheHeldRaceConcludesWhatItsReleaseRecorded(t *testing.T) {
	const (
		succeededStatus = testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
		rejected        = testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED
		admitted        = testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED
	)
	held := func() effect { return outcome(succeededStatus, "", nil) }
	for _, test := range []struct {
		name        string
		hold        func() effect
		release     func(runID string) effect
		disposition testpilotspb.RunDisposition
		verdict     testpilotspb.VerdictStatus
		conformance testpilot.ConformanceStatus
		property    testpilot.PropertyStatus
		crossed     bool
	}{
		{"the server rejects the stale message", held, func(runID string) effect { return decided(runID, rejected, 0) },
			testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.VERDICT_STATUS_SATISFIED, testpilot.ConformanceConformant, testpilot.PropertySatisfied, false},
		{"the server admits the stale message", held, func(runID string) effect { return decided(runID, admitted, 1) },
			testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilot.ConformanceNonconformant, testpilot.PropertyInconclusive, false},
		{"the decision times out after the commit", held, func(string) effect {
			return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, "", nil)
		}, testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilot.ConformanceConformant, testpilot.PropertyInconclusive, false},
		{"the answer is lost", held, func(string) effect {
			return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, "delivery_not_realized", nil)
		}, testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilot.ConformanceConformant, testpilot.PropertyInconclusive, false},
		{"the hold is not realized", func() effect {
			return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, "delivery_not_realized", nil)
		}, func(runID string) effect { return decided(runID, rejected, 0) },
			testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, testpilot.ConformanceInconclusive, testpilot.PropertyInconclusive, false},
		{"the record names another activity", held, func(string) effect { return decided("another", rejected, 0) },
			0, 0, "", testpilot.PropertyInconclusive, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			b := loweredRace(t)
			run, verdict, live, err := b.assessed.Run(t.Context(), &raceDriver{identity: b.plain.Identity(), hold: test.hold, release: test.release})
			require.NoError(t, err)
			if test.crossed {
				// Evidence keyed to an activity the Run never started confirms nothing of this one, and
				// is no step the other can take: whatever is made of it, it is never satisfaction.
				require.NotEqual(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
			} else {
				require.Equal(t, test.disposition, run.GetDisposition(), "%v", run.GetDiagnostics())
				require.Equal(t, test.verdict, verdict.GetStatus())
				require.Nil(t, live.Failure)
				require.Equal(t, test.conformance, live.Conformance.Status, live.Conformance.Detail)
				require.Equal(t, test.property, raceClaim(t, live, raceProperty).Status)
			}
			for _, property := range live.Properties {
				require.NotEqual(t, testpilot.PropertyViolated, property.Status, property.ID)
				if test.crossed && property.ID == raceProperty {
					require.NotEqual(t, testpilot.PropertySatisfied, property.Status)
				}
			}
			replayed, evaluation, err := b.assessed.Evaluate(t.Context(), run, live)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, live, evaluation.Assessment)
		})
	}
}

// A Run of the held race is assessed by what it recorded and by nothing of when: every elapsed time
// rewritten, the Verdict and the assessment are the ones the Run had. And with the commit record
// taken out, the same Run no longer says the stale message reached admission: its Contract is not
// satisfied and the claim is inconclusive, never satisfied and never violated (A10).
func TestTheHeldRaceIsReadByItsEvidenceAndNotItsClock(t *testing.T) {
	b := loweredRace(t)
	run, verdict, live, err := b.assessed.Run(t.Context(), &raceDriver{identity: b.plain.Identity(),
		hold:    func() effect { return outcome(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, "", nil) },
		release: func(runID string) effect { return decided(runID, testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED, 0) }})
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Equal(t, testpilot.PropertySatisfied, raceClaim(t, live, raceProperty).Status)

	skewed := proto.CloneOf(run)
	for _, event := range skewed.GetEvents() {
		event.Elapsed = duration.FromMilliseconds(86400000 * (event.GetSequence() - 1))
	}
	replayed, evaluation, err := b.assessed.Evaluate(t.Context(), skewed, live)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, verdict, replayed)
	require.Equal(t, live, evaluation.Assessment)

	uncommitted := proto.CloneOf(run)
	removed := 0
	for _, event := range uncommitted.GetEvents() {
		if event.GetOutcome().GetDeliveryAdmission() != nil {
			event.GetOutcome().DeliveryAdmission, event.Observations = nil, nil
			removed++
		}
	}
	require.Equal(t, 1, removed)
	replayed, evaluation, err = b.assessed.Evaluate(t.Context(), uncommitted, nil)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, replayed.GetStatus())
	require.Nil(t, evaluation.Assessment.Failure)
	require.NotEqual(t, testpilot.ConformanceNonconformant, evaluation.Assessment.Conformance.Status)
	require.Equal(t, testpilot.PropertyInconclusive, raceClaim(t, evaluation.Assessment, raceProperty).Status)
}
