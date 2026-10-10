package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/await"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
)

// The states of one declared attempt's reservation, as the worker README names them.
const (
	reserved                  = "reserved"
	admitted                  = "admitted"
	offeredCompleted          = "offered-completed"
	offeredFailedRetryable    = "offered-failed-retryable"
	offeredFailedNonRetryable = "offered-failed-non-retryable"
	offeredCanceled           = "offered-canceled"
	refusedAttempt            = "refused"
	releasedNotNeeded         = "released-not-needed"
	neverSeen                 = "never-seen"
)

func finishingAttempt(id, result string) *testpilotspb.InstructionNode {
	node := activityEntrypoint(facadetest.Text(result)).Instructions[0]
	node.InstructionId = id
	return node
}

// declaring makes the runtime fixture a standalone activity whose script is the attempts.
func declaring(attempts ...*testpilotspb.InstructionNode) func(*testpilotspb.Program) {
	return func(program *testpilotspb.Program) {
		standaloneActivity(program)
		program.Entrypoints[1].Instructions = attempts
	}
}

// lifecycle drives the three declared attempts of one started activity through a real Session and
// its ledger, the way the SDK's deliveries, the server and the Run do.
type lifecycle struct {
	t       *testing.T
	host    *Driver
	session *Session
	worker  *sdkWorkerInterceptor
	request *workflowservice.StartActivityExecutionRequest
	closed  chan string
	asked   *atomic.Int32

	mu      sync.Mutex
	runs    map[int32]int
	gates   map[int32]chan struct{}
	pending map[int32]chan error
	// requests are, per attempt under way, how the server tells its delivery that the activity's
	// cancellation is requested, which it does in answer to a heartbeat: heartbeats counts the ones
	// the worker sent, and asking holds the count at which each such delivery arrived.
	requests   map[int32]context.CancelCauseFunc
	asking     map[int32]int32
	heartbeats *atomic.Int32
}

// deliver hands the worker one delivery of an attempt and returns the refusal it was answered with,
// if any: a failure its instruction declares is the attempt's answer, not a refusal.
func (l *lifecycle) deliver(ctx context.Context, activityRunID string, attempt int32, deliveryID string) error {
	_, err := l.worker.activateActivity(ctx, activityAttempt(l.request, activityRunID, attempt, deliveryID), func(ctx context.Context) (any, error) {
		l.mu.Lock()
		l.runs[attempt]++
		gate := l.gates[attempt]
		l.mu.Unlock()
		if gate != nil {
			<-gate
		}
		return l.host.dynamicActivity(ctx, nil)
	})
	var answer *temporal.ApplicationError
	if errors.As(err, &answer) && answer.Type() != activationErrorType {
		return nil
	}
	// A cancellation its instruction declares is the attempt's answer too.
	var canceled *temporal.CanceledError
	if errors.As(err, &canceled) {
		return nil
	}
	return err
}

func (l *lifecycle) state(reservationID string) string {
	raw := l.session.reservations[reservationID]
	raw.mu.Lock()
	defer raw.mu.Unlock()
	if !raw.completed {
		if raw.consumed {
			return admitted
		}
		return reserved
	}
	attempt := raw.result.Outcome.GetActivityAttempt()
	if attempt == nil {
		return neverSeen
	}
	return map[testpilotspb.ActivityAttemptResponse]string{
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED:            offeredCompleted,
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE:     offeredFailedRetryable,
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE: offeredFailedNonRetryable,
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED:             offeredCanceled,
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED:                      refusedAttempt,
		testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED:                   releasedNotNeeded,
	}[attempt.GetResponse()]
}

type transition func(*lifecycle) error

// delivered is one delivery of an attempt, from arrival to its answer.
func delivered(attempt int32, deliveryID string) transition {
	return func(l *lifecycle) error {
		bounded, cancel := context.WithTimeout(l.t.Context(), 5*time.Second)
		defer cancel()
		return l.deliver(bounded, "activity-run", attempt, deliveryID)
	}
}

// deliveredForAnotherRun is the first attempt of another run of the same activity.
func deliveredForAnotherRun(l *lifecycle) error {
	return l.deliver(l.t.Context(), "another-run", 1, "delivery-1")
}

// abandoned is a delivery whose own context ends while it is still under way.
func abandoned(attempt int32, deliveryID string) transition {
	return func(l *lifecycle) error {
		bounded, cancel := context.WithTimeout(l.t.Context(), 50*time.Millisecond)
		defer cancel()
		return l.deliver(bounded, "activity-run", attempt, deliveryID)
	}
}

// arrived starts a delivery and leaves it under way. A held one stops before its instruction until
// it is answered; any other goes as far as the worker lets it.
func arrived(attempt int32, deliveryID string, held bool) transition {
	return func(l *lifecycle) error {
		l.mu.Lock()
		if held {
			l.gates[attempt] = make(chan struct{})
		}
		answer := make(chan error, 1)
		l.pending[attempt] = answer
		l.mu.Unlock()
		go func() { answer <- l.deliver(context.Background(), "activity-run", attempt, deliveryID) }()
		return nil
	}
}

// arrivedCancelable starts a delivery whose context the server can cancel, as the SDK's does when a
// heartbeat is answered with a requested cancellation, and leaves it under way.
func arrivedCancelable(attempt int32, deliveryID string) transition {
	return func(l *lifecycle) error {
		ctx, request := context.WithCancelCause(context.Background())
		answer := make(chan error, 1)
		l.mu.Lock()
		l.pending[attempt], l.requests[attempt], l.asking[attempt] = answer, request, l.heartbeats.Load()
		l.mu.Unlock()
		go func() { answer <- l.deliver(ctx, "activity-run", attempt, deliveryID) }()
		return nil
	}
}

// cancellationRequested is the server answering a heartbeat of the delivery under way with the
// activity's requested cancellation, and the delivery's answer to that.
func cancellationRequested(attempt int32) transition {
	return func(l *lifecycle) error {
		l.mu.Lock()
		request, arrivedAt := l.requests[attempt], l.asking[attempt]
		l.mu.Unlock()
		await.RequireTrue(l.t, func() bool { return l.heartbeats.Load() > arrivedAt }, 5*time.Second, 5*time.Millisecond)
		request(temporal.NewCanceledError())
		return answeredDelivery(attempt)(l)
	}
}

// answeredDelivery lets a delivery under way finish and returns its refusal, if any.
func answeredDelivery(attempt int32) transition {
	return func(l *lifecycle) error {
		l.mu.Lock()
		gate, answer := l.gates[attempt], l.pending[attempt]
		delete(l.gates, attempt)
		l.mu.Unlock()
		if gate != nil {
			close(gate)
		}
		select {
		case err := <-answer:
			return err
		case <-l.t.Context().Done():
			return l.t.Context().Err()
		}
	}
}

// serverClosed is the server reporting the activity closed.
func serverClosed(l *lifecycle) error {
	l.closed <- ""
	return nil
}

// anotherRunClosed is the server answering the worker's poll with the outcome of another run.
func anotherRunClosed(l *lifecycle) error {
	l.closed <- "another-run"
	return nil
}

// deliveredFor is a delivery of an attempt that names another activity run.
func deliveredFor(activityRunID string, attempt int32, deliveryID string) transition {
	return func(l *lifecycle) error { return l.deliver(l.t.Context(), activityRunID, attempt, deliveryID) }
}

// deliveredAs is a delivery that carries this activity's route under another activity's ID.
func deliveredAs(activityID string, attempt int32) transition {
	return func(l *lifecycle) error {
		crossed := activityAttempt(l.request, "activity-run", attempt, "delivery-crossed")
		crossed.ActivityID = activityID
		_, err := l.worker.activateActivity(l.t.Context(), crossed, func(context.Context) (any, error) {
			return nil, errors.New("a crossed delivery ran")
		})
		return err
	}
}

// runReleased is the Run canceling one reservation.
func runReleased(reservationID string) transition {
	return func(l *lifecycle) error { return l.session.reservations[reservationID].Cancel(l.t.Context()) }
}

// One declared attempt's reservation moves through the states the worker README names, and through
// no other path. Each scenario drives a started activity with three declared attempts through the
// real Session and ledger, checks the state of all three reservations after every transition and
// every refused one, and then holds what each reservation settled with and how often each
// instruction ran to the whole expected value, so ordering, single settlement and the release rule
// are shown together.
func TestDeclaredActivityAttemptsFollowTheirLifecycle(t *testing.T) {
	failingThenEnding := declaring(failingAttempt("first", "transient", false), failingAttempt("second", "refusal", true), finishingAttempt("third", "done"))
	completing := declaring(finishingAttempt("first", "one"), finishingAttempt("second", "two"), finishingAttempt("third", "three"))
	disabled := finishingAttempt("second", "never")
	disabled.Guard = cel.Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: false}})
	refusingSecond := declaring(failingAttempt("first", "transient", false), disabled, finishingAttempt("third", "done"))
	canceling := declaring(cancelingAttempt("first"), finishingAttempt("second", "late"), finishingAttempt("third", "later"))

	type step struct {
		name string
		do   transition
		err  error
		want [3]string
	}
	for name, test := range map[string]struct {
		script   func(*testpilotspb.Program)
		steps    []step
		runs     map[int32]int
		outcomes [3]*testpilotspb.InstructionOutcome
		// heartbeats says whether any attempt of the scenario waits for the server to ask it to
		// cancel, which is the only reason one heartbeats.
		heartbeats bool
		// diagnostics names, in order, what the Session reported to the Run: each refusal for an
		// identity that names another activity run, an undeclared attempt, a released position or
		// a crossed answer of the server.
		diagnostics []string
	}{
		"the declared path, then the server closes the activity": {
			script: failingThenEnding,
			steps: []step{
				{"the first attempt is delivered and fails as declared", delivered(1, "delivery-1"), nil, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"the same delivery again settles nothing", delivered(1, "delivery-1"), nil, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"the attempt redelivered settles nothing", delivered(1, "redelivery-1"), nil, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"an attempt the script does not declare has no reservation", delivered(4, "delivery-4"), delivery.ErrAttemptUndeclared, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"the second attempt named under another activity run consumes nothing", deliveredFor("another-run", 2, "delivery-2"), delivery.ErrRouteConflict, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"a redelivery of the first attempt named under another activity run is not answered", deliveredFor("another-run", 1, "delivery-1"), delivery.ErrRouteConflict, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"this activity's route under another activity's ID reaches no Session", deliveredAs("another-activity", 2), delivery.ErrRouteCrossed, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"the second attempt offers a failure that is not retried, which releases nothing", delivered(2, "delivery-2"), nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, releasedNotNeeded}},
				{"a released position admits nothing", delivered(3, "delivery-3"), delivery.ErrRouteStale, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, releasedNotNeeded}},
				{"a settled attempt is still answered", delivered(2, "delivery-2"), nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1, 2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", failedRetryable), answered("activity-run", 2, "delivery-2", failedNonRetryable), notNeeded("activity-run"),
			},
			diagnostics: []string{"activity_attempt_undeclared", "activity_run_crossed", "activity_run_crossed", "activity_delivery_late"},
		},
		"the server answers with the outcome of another run": {
			script: completing,
			steps: []step{
				{"the first attempt offers its completion", delivered(1, "delivery-1"), nil, [3]string{offeredCompleted, reserved, reserved}},
				{"another run's outcome releases nothing", anotherRunClosed, nil, [3]string{offeredCompleted, reserved, reserved}},
				{"the second attempt still finds its reservation", delivered(2, "delivery-2"), nil, [3]string{offeredCompleted, offeredCompleted, reserved}},
			},
			runs: map[int32]int{1: 1, 2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", completed), answered("activity-run", 2, "delivery-2", completed), nil,
			},
			diagnostics: []string{"activity_closure_crossed"},
		},
		"a completion the server never accepts is followed by the next attempt": {
			script: completing,
			steps: []step{
				{"the first attempt is admitted", arrived(1, "delivery-1", true), nil, [3]string{admitted, reserved, reserved}},
				{"it offers its completion", answeredDelivery(1), nil, [3]string{offeredCompleted, reserved, reserved}},
				{"the server issues the second attempt, which finds its reservation", delivered(2, "delivery-2"), nil, [3]string{offeredCompleted, offeredCompleted, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{offeredCompleted, offeredCompleted, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1, 2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", completed), answered("activity-run", 2, "delivery-2", completed), notNeeded("activity-run"),
			},
		},
		"attempts that arrive inverted settle in order": {
			script: failingThenEnding,
			steps: []step{
				{"the second attempt arrives first and waits", arrived(2, "delivery-2", false), nil, [3]string{reserved, admitted, reserved}},
				{"the first attempt is delivered, and the second follows it", delivered(1, "delivery-1"), nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, reserved}},
				{"the second attempt was answered as declared", answeredDelivery(2), nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, reserved}},
				{"the Run releases the third, which the server said nothing about", runReleased("reservation-3"), nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, neverSeen}},
				{"the server closing the activity changes no settled reservation", serverClosed, nil, [3]string{offeredFailedRetryable, offeredFailedNonRetryable, neverSeen}},
			},
			runs: map[int32]int{1: 1, 2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", failedRetryable), answered("activity-run", 2, "delivery-2", failedNonRetryable), {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED},
			},
		},
		"a refused attempt": {
			script: refusingSecond,
			steps: []step{
				{"the first attempt fails as declared", delivered(1, "delivery-1"), nil, [3]string{offeredFailedRetryable, reserved, reserved}},
				{"the second attempt's instruction is disabled", delivered(2, "delivery-2"), errAttemptDisabled, [3]string{offeredFailedRetryable, refusedAttempt, reserved}},
				{"its redelivery is refused the same way and settles nothing", delivered(2, "redelivery-2"), errAttemptDisabled, [3]string{offeredFailedRetryable, refusedAttempt, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{offeredFailedRetryable, refusedAttempt, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1, 2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", failedRetryable), refused("activity-run", 2, "delivery-2", errAttemptDisabled.Error()), notNeeded("activity-run"),
			},
		},
		// The worker may answer canceled only what the server asked it to cancel, and learns of that
		// through the attempt's heartbeat. The canceled answer is an offer like any other.
		"a cancellation the server requests is answered, and the server then closes the activity": {
			script: canceling, heartbeats: true,
			steps: []step{
				{"the first attempt is admitted and waits for the server to ask", arrivedCancelable(1, "delivery-1"), nil, [3]string{admitted, reserved, reserved}},
				{"the server asks, and the attempt offers the cancellation, which releases nothing", cancellationRequested(1), nil, [3]string{offeredCanceled, reserved, reserved}},
				{"a redelivery waits for the server to ask it too", arrivedCancelable(1, "redelivery-1"), nil, [3]string{offeredCanceled, reserved, reserved}},
				{"the server asks, and the redelivery is answered canceled again, which settles nothing", cancellationRequested(1), nil, [3]string{offeredCanceled, reserved, reserved}},
				{"a redelivery the server never asks is not answered canceled", abandoned(1, "redelivery-2"), context.DeadlineExceeded, [3]string{offeredCanceled, reserved, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{offeredCanceled, releasedNotNeeded, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", canceledAnswer), notNeeded("activity-run"), notNeeded("activity-run"),
			},
		},
		"a canceled answer the server never accepts is followed by the next attempt": {
			script: canceling, heartbeats: true,
			steps: []step{
				{"the first attempt is admitted and waits for the server to ask", arrivedCancelable(1, "delivery-1"), nil, [3]string{admitted, reserved, reserved}},
				{"the server asks, and the attempt offers the cancellation", cancellationRequested(1), nil, [3]string{offeredCanceled, reserved, reserved}},
				{"the server issues the second attempt, which finds its reservation and the first's outcome", delivered(2, "delivery-2"), nil, [3]string{offeredCanceled, offeredCompleted, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{offeredCanceled, offeredCompleted, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1, 2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", canceledAnswer), answered("activity-run", 2, "delivery-2", completed), notNeeded("activity-run"),
			},
		},
		"a cancellation the server never requests is refused": {
			script: canceling, heartbeats: true,
			steps: []step{
				{"the first attempt's delivery ends while it waits for the server to ask", abandoned(1, "delivery-1"), context.DeadlineExceeded, [3]string{refusedAttempt, reserved, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{refusedAttempt, releasedNotNeeded, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				refused("activity-run", 1, "delivery-1", context.DeadlineExceeded.Error()), notNeeded("activity-run"), notNeeded("activity-run"),
			},
		},
		"the Run cancels an attempt that waits for the server to ask": {
			script: canceling, heartbeats: true,
			steps: []step{
				{"the first attempt is admitted and waits", arrivedCancelable(1, "delivery-1"), nil, [3]string{admitted, reserved, reserved}},
				{"the Run cancels its reservation, which is no request of the server", runReleased("reservation-1"), nil, [3]string{refusedAttempt, reserved, reserved}},
				{"the attempt was refused, not answered canceled", answeredDelivery(1), context.Canceled, [3]string{refusedAttempt, reserved, reserved}},
			},
			runs: map[int32]int{1: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				refused("activity-run", 1, "delivery-1", context.Canceled.Error()), nil, nil,
			},
		},
		"an earlier attempt the worker never sees": {
			script: failingThenEnding,
			steps: []step{
				{"the second attempt gives up waiting for the first", abandoned(2, "delivery-2"), context.DeadlineExceeded, [3]string{reserved, refusedAttempt, reserved}},
				{"the server closing the activity releases only what follows the last attempt delivered", serverClosed, nil, [3]string{reserved, refusedAttempt, releasedNotNeeded}},
				{"the Run releases the first", runReleased("reservation-1"), nil, [3]string{neverSeen, refusedAttempt, releasedNotNeeded}},
				{"a reservation the Run released admits nothing", delivered(1, "delivery-1"), delivery.ErrLifecycle, [3]string{neverSeen, refusedAttempt, releasedNotNeeded}},
			},
			runs: map[int32]int{2: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}, refused("activity-run", 2, "delivery-2", context.DeadlineExceeded.Error()), notNeeded("activity-run"),
			},
		},
		"the Run cancels an attempt under way": {
			script: completing,
			steps: []step{
				{"the first attempt is admitted", arrived(1, "delivery-1", true), nil, [3]string{admitted, reserved, reserved}},
				{"the Run cancels its reservation, which settles nothing by itself", runReleased("reservation-1"), nil, [3]string{admitted, reserved, reserved}},
				{"the attempt is refused", answeredDelivery(1), context.Canceled, [3]string{refusedAttempt, reserved, reserved}},
				{"the server closes the activity", serverClosed, nil, [3]string{refusedAttempt, releasedNotNeeded, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				refused("activity-run", 1, "delivery-1", context.Canceled.Error()), notNeeded("activity-run"), notNeeded("activity-run"),
			},
		},
		"an attempt of another run": {
			script: completing,
			steps: []step{
				{"it conflicts and consumes nothing", deliveredForAnotherRun, delivery.ErrRouteConflict, [3]string{reserved, reserved, reserved}},
				{"the server closing the activity releases nothing while no attempt was delivered", serverClosed, nil, [3]string{reserved, reserved, reserved}},
			},
			runs:        map[int32]int{},
			outcomes:    [3]*testpilotspb.InstructionOutcome{},
			diagnostics: []string{"activity_run_crossed"},
		},
		"a reservation the Run released before its attempt": {
			script: failingThenEnding,
			steps: []step{
				{"the Run releases the second reservation", runReleased("reservation-2"), nil, [3]string{reserved, neverSeen, reserved}},
				{"the first attempt fails as declared", delivered(1, "delivery-1"), nil, [3]string{offeredFailedRetryable, neverSeen, reserved}},
				{"the second attempt finds no reservation to consume", delivered(2, "delivery-2"), delivery.ErrLifecycle, [3]string{offeredFailedRetryable, neverSeen, reserved}},
				{"the server closing the activity does not settle it again", serverClosed, nil, [3]string{offeredFailedRetryable, neverSeen, releasedNotNeeded}},
			},
			runs: map[int32]int{1: 1},
			outcomes: [3]*testpilotspb.InstructionOutcome{
				answered("activity-run", 1, "delivery-1", failedRetryable), {Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}, notNeeded("activity-run"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, test.script)
			host, definition := runtimeTestDriver(t, prepared)
			closed, asked := serverClosure(host)
			session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
			diagnostics := captureDiagnostics(t, session)
			l := &lifecycle{
				t: t, host: host, session: session, worker: activityWorker(host, definition), request: request, closed: closed, asked: asked,
				runs: map[int32]int{}, gates: map[int32]chan struct{}{}, pending: map[int32]chan error{}, requests: map[int32]context.CancelCauseFunc{},
				asking: map[int32]int32{}, heartbeats: countHeartbeats(host),
			}
			states := func() [3]string {
				return [3]string{l.state("reservation-1"), l.state("reservation-2"), l.state("reservation-3")}
			}
			require.Equal(t, [3]string{reserved, reserved, reserved}, states())

			for _, step := range test.steps {
				err := step.do(l)
				if step.err == nil {
					require.NoError(t, err, step.name)
				} else {
					require.ErrorIs(t, err, step.err, step.name)
					requireRefusal(t, err)
				}
				await.Require(t.Context(), t, func(t *await.T) { require.Equal(t, step.want, states(), step.name) }, 5*time.Second, 5*time.Millisecond)
			}

			l.mu.Lock()
			require.Equal(t, test.runs, l.runs)
			l.mu.Unlock()
			// The server is asked once per activity, and only once an attempt settled.
			await.Require(t.Context(), t, func(t *await.T) { require.Equal(t, test.diagnostics, diagnostics.codes()) }, 5*time.Second, 5*time.Millisecond)
			require.Len(t, session.activityWatches, min(len(test.runs), 1))
			await.RequireTrue(t, func() bool { return int(asked.Load()) == min(len(test.runs), 1) }, 5*time.Second, 5*time.Millisecond)
			for index, want := range test.outcomes {
				raw := session.reservations[[3]string{"reservation-1", "reservation-2", "reservation-3"}[index]]
				if want == nil {
					require.False(t, raw.settled())
					continue
				}
				settled, err := settledAttempt(t, session, raw.identity.ID)
				require.NoError(t, err)
				requireOutcome(t, want, settled)
			}
			// An attempt heartbeats only to learn whether the server asks it to cancel.
			require.Equal(t, test.heartbeats, l.heartbeats.Load() > 0)
			require.NoError(t, session.Close(t.Context()))
		})
	}
}

// The worker's evidence that an activity closed is the server's answer alone. When the server
// cannot be asked, the later declared attempts stay reserved and the Session says why, and when
// the Session closes it stops asking and reports nothing.
func TestActivityClosureIsOnlyWhatTheServerReports(t *testing.T) {
	for name, test := range map[string]struct {
		closure     func(ctx context.Context, returned chan<- error) error
		close       bool
		returned    error
		diagnostics []diagnosed
	}{
		"the server cannot be asked": {
			closure:  func(context.Context, chan<- error) error { return errors.New("unavailable") },
			returned: nil,
			diagnostics: []diagnosed{{
				Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT, Code: "activity_closure_unobserved",
				Detail: `activity run "activity-run": unavailable`,
			}},
		},
		"the Session closes first": {
			closure: func(ctx context.Context, _ chan<- error) error {
				<-ctx.Done()
				return ctx.Err()
			},
			close:    true,
			returned: context.Canceled,
		},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, endedByFailureActivity)
			host, definition := runtimeTestDriver(t, prepared)
			returned := make(chan error, 1)
			host.options.activityClosed = func(ctx context.Context, _, _, activityRunID string) (string, error) {
				err := test.closure(ctx, returned)
				if ctx.Err() != nil {
					returned <- ctx.Err()
				} else {
					returned <- nil
				}
				return activityRunID, err
			}
			session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
			diagnostics := captureDiagnostics(t, session)
			_, err := activityWorker(host, definition).activateActivity(t.Context(), activityAttempt(request, "activity-run", 1, "delivery-1"), runScript(host))
			var declared *temporal.ApplicationError
			require.ErrorAs(t, err, &declared)
			if test.close {
				require.NoError(t, session.Close(t.Context()))
			}
			select {
			case <-t.Context().Done():
				require.FailNow(t, "the worker never stopped asking the server")
			case got := <-returned:
				require.Equal(t, test.returned, got)
			}
			await.Require(t.Context(), t, func(t *await.T) { require.Equal(t, test.diagnostics, diagnostics.all()) }, 5*time.Second, 5*time.Millisecond)
			if !test.close {
				require.False(t, session.reservations["reservation-2"].settled())
				return
			}
			// Closing releases the reservation as the Run does, never as an attempt not needed.
			settled, err := settledAttempt(t, session, "reservation-2")
			require.NoError(t, err)
			requireOutcome(t, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}, settled)
		})
	}
}
