package conformance

// The lowered Cases of the standalone activity Model, run live through Testpilot's own executor and
// recorder against a Driver that plays the activity, and the Runs they record replayed. No server is
// involved: what is exercised is the Case, as lowered, in the runtime it is lowered for. The evidence
// of each Run is what the Case's own declarations lift, from the Run's record of the controller's
// calls and of the attempts the worker was delivered, and from the reads of the activity's status.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// activityDriver plays a lowered activity Case against no server, through the public facade alone. It
// answers every call of the controller, reads each status the script gives a poll, and settles each
// attempt of the activity as the Case's own activity entrypoint answers it: delivered under the number
// the server would count, in one activity run.
//
// A Run records an attempt once it is answered, and the status an activity ends in is read after its
// last answer. The Driver keeps that order: it settles the attempts once the controller reaches the
// read named final, and answers that read once the Run has recorded every attempt.
type activityDriver struct {
	identity testpilot.DriverIdentity
	// statuses is the status each poll of the controller reads, by its instruction.
	statuses map[string]enumspb.ActivityExecutionStatus
	final    string
	// recorded is closed once the Run has recorded every attempt.
	recorded <-chan struct{}
}

func (d *activityDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (d *activityDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *activityDriver) Open(_ context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	return &activitySession{driver: d, runID: runID, program: program.Snapshot(), reached: make(chan struct{})}, nil
}

type activitySession struct {
	driver  *activityDriver
	runID   string
	program *testpilotspb.Program
	// reached is closed when the controller reaches the final read.
	reached  chan struct{}
	once     sync.Once
	reserved atomic.Int64
}

func (s *activitySession) Reserve(_ context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	var answers []*testpilotspb.InstructionNode
	for _, entrypoint := range s.program.GetEntrypoints() {
		if entrypoint.GetEntrypointId() == request.EntrypointID && entrypoint.GetActivity() != nil {
			answers = entrypoint.GetInstructions()
		}
	}
	if int64(len(answers)) != request.Count {
		return nil, fmt.Errorf("%d attempts of %s are reserved, and its script answers %d", request.Count, request.EntrypointID, len(answers))
	}
	var out []testpilot.ReservationHandle
	for ordinal, answer := range answers {
		response := testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
		if failure := answer.GetInstruction().GetActivityAttemptFailure(); failure != nil {
			response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE
			if failure.GetFailure().GetApplicationFailureInfo().GetNonRetryable() {
				response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE
			}
		}
		out = append(out, attempt{released: s.reached,
			identity: testpilot.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: int64(ordinal),
				ID: "attempt-" + strconv.FormatInt(s.reserved.Add(1), 10)},
			outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED,
				ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: int32(ordinal) + 1,
					DeliveryId: "delivery-" + strconv.Itoa(ordinal+1), Response: response}}})
	}
	return out, nil
}

func succeeded(response proto.Message) effect {
	return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
}

func (s *activitySession) InvokeRPC(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	return succeeded(dynamicpb.NewMessage(method.Output())), nil
}

func (s *activitySession) PollRPC(ctx context.Context, at testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message, _ time.Duration,
	accepts testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	status, scripted := s.driver.statuses[at.InstructionID]
	if !scripted || method.Name() != "DescribeActivityExecution" {
		return nil, errUnscripted
	}
	response := &workflowservice.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{ActivityId: s.runID, Status: status}}
	if accepted, err := accepts(ctx, response); err != nil || !accepted {
		return nil, fmt.Errorf("the status played never ends the poll %s: %w", at.InstructionID, err)
	}
	if at.InstructionID != s.driver.final {
		return succeeded(response), nil
	}
	s.once.Do(func() { close(s.reached) })
	return awaited{effect: succeeded(response), until: s.driver.recorded}, nil
}
func (*activitySession) InvokeHandle(context.Context, testpilot.Coordinate, testpilot.OpaqueHandle, proto.Message) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (*activitySession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return succeeded(nil), nil
}
func (*activitySession) Bridge(context.Context) (testpilot.HandleBridge, error)   { return nil, nil }
func (*activitySession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*activitySession) Close(context.Context) error                              { return nil }
func (*activitySession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

// awaited is an effect that completes once something else has happened.
type awaited struct {
	effect
	until <-chan struct{}
}

func (a awaited) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	select {
	case <-a.until:
		return a.effect.Wait(ctx)
	case <-ctx.Done():
		return testpilot.EffectResult{}, ctx.Err()
	}
}

// attempt is one attempt of an activity the worker was delivered and answered, settled once released.
type attempt struct {
	identity testpilot.ReservationIdentity
	outcome  *testpilotspb.InstructionOutcome
	released <-chan struct{}
}

func (a attempt) Identity() testpilot.ReservationIdentity { return a.identity }
func (a attempt) Consume(context.Context) (testpilot.Coordinate, error) {
	return a.identity.Origin, nil
}
func (a attempt) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	select {
	case <-a.released:
		return testpilot.EffectResult{Outcome: proto.CloneOf(a.outcome)}, nil
	case <-ctx.Done():
		return testpilot.EffectResult{}, ctx.Err()
	}
}
func (attempt) Cancel(context.Context) error { return nil }
func (attempt) Drain(context.Context) error  { return nil }

// watching is an assessment factory that says when a Run has recorded as many attempts as its Case
// answers. It assesses nothing itself: every event goes to the factory it watches for.
type watching struct {
	testpilot.AssessmentFactory
	attempts int
	recorded chan struct{}
}

func (w *watching) New(ctx context.Context) (testpilot.Assessor, error) {
	inner, err := w.AssessmentFactory.New(ctx)
	if err != nil {
		return nil, err
	}
	if w.attempts == 0 {
		close(w.recorded)
	}
	return &watcher{inner: inner, of: w}, nil
}

type watcher struct {
	testpilot.SingleUse
	inner testpilot.Assessor
	of    *watching
	seen  int
}

func (w *watcher) Observe(ctx context.Context, event *testpilotspb.RunEvent) (testpilot.Established, error) {
	established, err := w.inner.Observe(ctx, event)
	if event.GetOutcome().GetActivityAttempt() != nil {
		if w.seen++; w.seen == w.of.attempts {
			close(w.of.recorded)
		}
	}
	return established, err
}

func (w *watcher) Close(ctx context.Context, closure testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
	return w.inner.Close(ctx, closure)
}

// playedKinds is, for each Run Event of a Run that carries evidence, its sequence and the kind of
// evidence it carries, by the last part of the kind's id: read off the Run, and not off an assessment.
func playedKinds(t testing.TB, source *testpilotspb.Case, run *testpilotspb.Run, prefix string) ([]int64, []string) {
	t.Helper()
	defined := map[string]string{}
	for _, name := range source.GetProvenance().GetLocalNames() {
		defined[name.GetLocalName()] = name.GetDefinitionId()
	}
	var sequences []int64
	var kinds []string
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			if observation.GetObservationId() != evidenceObservation {
				continue
			}
			evidence := &testpilotspb.CorrelatedEvidence{}
			require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(evidence))
			id, renamed := defined[evidence.GetKind()]
			if !renamed {
				id = evidence.GetKind()
			}
			sequences, kinds = append(sequences, event.GetSequence()), append(kinds, strings.TrimPrefix(id, prefix))
		}
	}
	return sequences, kinds
}

// Each lowered Case of the activity Model runs, live, and its Run is replayed to the same Verdict and
// the same assessment. The Driver plays the path of the Case's Query: the calls answered, the statuses
// its polls read, and the attempts its activity entrypoint answers. The Run then carries the evidence
// the Case's declarations lift from it, one piece per step the path confirms by evidence, in path
// order; the Contract reads it as the witness and is satisfied.
//
// The claims are read off the protocol machine's object, its properties and its step functions:
//
//   - completion, pauseResume (`completes`) and nonRetryableFailure (`nonRetryableFails`) are
//     satisfied: the status the Run ends on is recorded by the claim's class alone, every step of that
//     class the machine has satisfies the claim, and an activity that is over takes no answer.
//   - terminate (`terminated`) stays open: a control of an activity that is over is not found and
//     records nothing (Protocol.control), so a second terminate after the one the Run shows is a
//     step of the claim's class that no evidence reports and on which the claim fails. The executions
//     that explain the Run disagree.
//   - retry (`retryCompletes`) stays open: the claim fixes the whole state the activity ends in, its
//     three deadlines included, and the start's answer is one fact for all eight start classes, so the
//     executions that explain the Run disagree.
//   - scheduleToStartTimeout (`scheduleToStartFires`) stays open: each of the three deadlines records
//     the timed-out status under one evidence name, so on an execution where another deadline fired
//     the claim, which is about the schedule-to-start deadline, is never read.
func TestALoweredActivityCaseRunsLiveAndReplaysAlike(t *testing.T) {
	m := activityModel(t)
	const (
		paused     = enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED
		completed  = enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED
		failed     = enumspb.ACTIVITY_EXECUTION_STATUS_FAILED
		terminated = enumspb.ACTIVITY_EXECUTION_STATUS_TERMINATED
		expired    = enumspb.ACTIVITY_EXECUTION_STATUS_TIMED_OUT
	)
	// A claim's assessment, given the Run Events that carry the Run's evidence and the Run's id, which
	// is the operation its evidence names.
	type assessed func(support []int64, runID string) testpilot.PropertyAssessment
	satisfied := func(id string) assessed {
		return func(support []int64, _ string) testpilot.PropertyAssessment {
			return testpilot.PropertyAssessment{ID: id, Status: testpilot.PropertySatisfied, SupportingEventSequences: support}
		}
	}
	open := func(query, id string, why reason) assessed {
		return func(_ []int64, runID string) testpilot.PropertyAssessment {
			return testpilot.PropertyAssessment{ID: id, Status: testpilot.PropertyInconclusive, Reason: ir.ExpectationID(why),
				Detail: activityMachine + `, run="standaloneActivityTests-` + query + `";` + runID + ": " + wording[why]}
		}
	}
	for query, test := range map[string]struct {
		statuses map[string]enumspb.ActivityExecutionStatus
		final    string
		attempts int
		kinds    []string
		property assessed
	}{
		"completion": {map[string]enumspb.ActivityExecutionStatus{"await-completed": completed}, "await-completed", 1,
			[]string{"statusScheduled", "statusStarted", "statusCompleted"}, satisfied("completes")},
		"nonRetryableFailure": {map[string]enumspb.ActivityExecutionStatus{"await-failed": failed}, "await-failed", 1,
			[]string{"statusScheduled", "statusStarted", "statusFailed"}, satisfied("nonRetryableFails")},
		"retry": {map[string]enumspb.ActivityExecutionStatus{"await-completed": completed}, "await-completed", 2,
			[]string{"statusScheduled", "statusStarted", "attemptCount", "statusCompleted"}, open("retry", "retryCompletes", whyDisagreement)},
		"pauseResume": {map[string]enumspb.ActivityExecutionStatus{"await-paused": paused, "await-completed": completed}, "await-completed", 1,
			[]string{"statusScheduled", "statusPaused", "statusScheduledAgain", "statusStarted", "statusCompleted"}, satisfied("completes")},
		"terminate": {map[string]enumspb.ActivityExecutionStatus{"await-terminated": terminated}, "await-terminated", 0,
			[]string{"statusScheduled", "statusTerminated"}, open("terminate", "terminated", whyDisagreement)},
		"scheduleToStartTimeout": {map[string]enumspb.ActivityExecutionStatus{"await-timed-out": expired}, "await-timed-out", 0,
			[]string{"statusScheduled", "statusTimedOut"}, open("scheduleToStartTimeout", "scheduleToStartFires", whyNeverRead)},
	} {
		t.Run(query, func(t *testing.T) {
			b := loweredActivity(t, m, query)
			recorded := make(chan struct{})
			watched, err := b.plain.WithAssessment(&watching{AssessmentFactory: b.factory, attempts: test.attempts, recorded: recorded})
			require.NoError(t, err)
			run, verdict, live, err := watched.Run(t.Context(), &activityDriver{identity: b.plain.Identity(), statuses: test.statuses, final: test.final,
				recorded: recorded})
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "%v", run.GetDiagnostics())
			require.Empty(t, run.GetDiagnostics())
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())

			support, kinds := playedKinds(t, b.source, run, activityEvidence)
			require.Equal(t, test.kinds, kinds)
			binding := b.factory.Binding()
			require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
				Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, SupportingEventSequences: support},
				Properties:  []testpilot.PropertyAssessment{test.property(support, run.GetRunId())}}, live)

			plainVerdict, _, err := b.plain.Evaluate(t.Context(), run)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, plainVerdict, verdict)
			replayed, evaluation, err := b.assessed.Evaluate(t.Context(), run, live)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, live, evaluation.Assessment)
		})
	}
}
