package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/sdk/temporal"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const pinnedActivityHeader = "temporal-testpilot-reserved-activity-v1"

func textResult(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}
}

func requireOutcome(t *testing.T, want *testpilotspb.InstructionOutcome, got testpilot.EffectResult) {
	t.Helper()
	require.True(t, proto.Equal(want, got.Outcome), got.Outcome)
	require.Nil(t, got.Response)
}

// requireRefusal checks the error the SDK is answered with when the Driver refuses or fails an
// attempt: an application failure of the Driver's own type that the server does not retry.
func requireRefusal(t *testing.T, err error) {
	t.Helper()
	var refusal *temporal.ApplicationError
	require.ErrorAs(t, err, &refusal)
	require.True(t, refusal.NonRetryable())
	require.Equal(t, "umpire_worker", refusal.Type())
}

// The lowered activity script runs under the activation its reservation names: the attempt answers
// with the script's result, and the reservation settles succeeded, recording as typed facts the
// operation, the SDK attempt and the delivery it ran for and that the worker answered a completion.
func TestActivityActivationRunsItsScriptUnderItsReservation(t *testing.T) {
	prepared := preparedActivityFixture(t, standaloneActivity)
	host, definition := runtimeTestDriver(t, prepared)
	serverClosure(host)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)

	// The route is compared as the bytes it is, not as equivalent JSON.
	route := []byte(`{"version":1,"kind":"activity","session_id":"session-run","run_id":"run","origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller-1","InstructionID":"start-activity","Attempt":1},"reservation":{"Origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller-1","InstructionID":"start-activity","Attempt":1},"EntrypointID":"activity","Ordinal":0,"ID":"reservation-1"},"workflow_ordinal":0,"activity":{"namespace":"namespace","activity_id":"activity-id","activity_type":"activity-type","task_queue":"task-queue"}}`)
	require.Len(t, request.GetHeader().GetFields(), 1)
	require.Equal(t, string(route), string(request.GetHeader().GetFields()[pinnedActivityHeader].GetData()))

	var activated testpilot.Coordinate
	result, err := activityWorker(host, definition).activateActivity(t.Context(), activityDelivery(request, "activity-run"), func(ctx context.Context) (any, error) {
		routed, ok := ctx.Value(activityRouteKey{}).(routedActivity)
		require.True(t, ok)
		activated = routed.activation.Coordinate()
		return host.dynamicActivity(ctx, nil)
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(textResult("done"), result.(*testpilotspb.Value)), result)
	require.Equal(t, testpilot.Coordinate{RunID: "run", EntrypointID: "activity", ActivationID: "reservation-1", Attempt: 1}, activated)

	settled, err := settledActivity(t, session)
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, "delivery-1", completed), settled)
	// No later attempt is declared, so nothing waits on the server and the worker asks it nothing.
	require.Empty(t, session.activityWatches)
	require.NoError(t, session.Close(t.Context()))
	require.Nil(t, host.sessions["run"])
}

// Completing and failing an attempt are two instructions. A Finish completes with its result even
// when that result is a Temporal failure message: the worker answers a completion carrying it.
func TestActivityFinishCompletesEvenWithAFailureMessageAsItsResult(t *testing.T) {
	carried, err := anypb.New(&failurepb.Failure{Message: "a value, not an outcome", FailureInfo: &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: &failurepb.TimeoutFailureInfo{}}})
	require.NoError(t, err)
	value := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: carried}}
	prepared := preparedActivityFixture(t, standaloneActivity, func(program *testpilotspb.Program) {
		program.Entrypoints[1].Instructions[0].GetInstruction().GetFinish().Result = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: value}}
	})
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)

	result, err := activityWorker(host, definition).activateActivity(t.Context(), activityDelivery(request, "activity-run"), runScript(host))
	require.NoError(t, err)
	require.True(t, proto.Equal(value, result.(*testpilotspb.Value)), result)
	settled, err := settledActivity(t, session)
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, "delivery-1", completed), settled)
}

// A declared attempt may fail. The first attempt answers with the failure its instruction carries,
// and its activation still succeeds, because it did what the Program said; the recorded fact says
// which failure Temporal was told. After a retryable failure the server's second attempt runs the
// second instruction under the second reservation. A non-retryable failure is offered the same way;
// what becomes of the attempts declared after it is the server's to say.
func TestActivityAttemptFailsAsItsInstructionSays(t *testing.T) {
	t.Run("retryably, and its retry completes", func(t *testing.T) {
		prepared := preparedActivityFixture(t, retriedActivity)
		host, definition := runtimeTestDriver(t, prepared)
		session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
		worker := activityWorker(host, definition)
		// An open reservation of another start, or of another entrypoint under this start, is not
		// an attempt of this activity, so the retry does not wait on it.
		anotherStart := activityOrigin("run")
		anotherStart.InstructionID = "another-start"
		for _, identity := range []testpilot.ReservationIdentity{
			{Origin: anotherStart, EntrypointID: "activity", ID: "another-start"},
			{Origin: activityOrigin("run"), EntrypointID: "another-entrypoint", ID: "another-entrypoint"},
		} {
			session.reservations[identity.ID] = newReservation(identity)
		}
		var activations []testpilot.Coordinate
		run := func(ctx context.Context) (any, error) {
			activations = append(activations, ctx.Value(activityRouteKey{}).(routedActivity).activation.Coordinate())
			return host.dynamicActivity(ctx, nil)
		}

		result, err := worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 1, "delivery-1"), run)
		require.Nil(t, result)
		var declared *temporal.ApplicationError
		require.ErrorAs(t, err, &declared)
		require.Equal(t, "transient", declared.Type())
		require.Equal(t, "not yet", declared.Message())
		require.False(t, declared.NonRetryable())
		first, err := settledAttempt(t, session, "reservation-1")
		require.NoError(t, err)
		requireOutcome(t, answered("activity-run", 1, "delivery-1", failedRetryable), first)
		require.False(t, session.reservations["reservation-2"].completed)

		bounded, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		result, err = worker.activateActivity(bounded, activityAttempt(request, "activity-run", 2, "delivery-2"), run)
		require.NoError(t, err)
		require.True(t, proto.Equal(textResult("done"), result.(*testpilotspb.Value)), result)
		second, err := settledAttempt(t, session, "reservation-2")
		require.NoError(t, err)
		requireOutcome(t, answered("activity-run", 2, "delivery-2", completed), second)
		// Each attempt is its own activation; the coordinate's attempt stays the carrying
		// instruction's, and the SDK attempt is the recorded fact above.
		require.Equal(t, []testpilot.Coordinate{
			{RunID: "run", EntrypointID: "activity", ActivationID: "reservation-1", Attempt: 1},
			{RunID: "run", EntrypointID: "activity", ActivationID: "reservation-2", Attempt: 1},
		}, activations)
	})

	t.Run("non-retryably", func(t *testing.T) {
		prepared := preparedActivityFixture(t, endedByFailureActivity)
		host, definition := runtimeTestDriver(t, prepared)
		_, asked := serverClosure(host)
		session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
		worker := activityWorker(host, definition)

		_, err := worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 1, "delivery-1"), runScript(host))
		var declared *temporal.ApplicationError
		require.ErrorAs(t, err, &declared)
		require.Equal(t, "refusal", declared.Type())
		require.True(t, declared.NonRetryable())
		first, err := settledAttempt(t, session, "reservation-1")
		require.NoError(t, err)
		requireOutcome(t, answered("activity-run", 1, "delivery-1", failedNonRetryable), first)
		// The worker only offered that failure. Until the server says the activity closed, the
		// attempt the script declares next keeps its reservation, and the worker asks the server.
		require.False(t, session.reservations["reservation-2"].settled())
		await.RequireTrue(t, func() bool { return asked.Load() == 1 }, 5*time.Second, 5*time.Millisecond)
	})
}

// An attempt runs the instruction of its own number however the attempts are scheduled. When the
// second attempt reaches the worker first, it waits for the first attempt to settle before it
// interprets anything, so it cannot take the first attempt's instruction, and each attempt is
// recorded under its own number.
func TestActivityAttemptsThatArriveInvertedRunTheirOwnInstructions(t *testing.T) {
	prepared := preparedActivityFixture(t, retriedActivity)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
	worker := activityWorker(host, definition)

	type answer struct {
		result any
		err    error
	}
	admittedSecond := make(chan struct{})
	second := make(chan answer, 1)
	go func() {
		result, err := worker.activateActivity(context.Background(), activityAttempt(request, "activity-run", 2, "delivery-2"), func(ctx context.Context) (any, error) {
			close(admittedSecond)
			return host.dynamicActivity(ctx, nil)
		})
		second <- answer{result, err}
	}()
	select {
	case <-admittedSecond:
	case got := <-second:
		require.FailNow(t, "the second attempt ended before it was admitted", got.err)
	}
	require.True(t, session.reservations["reservation-2"].consumed)
	require.False(t, session.reservations["reservation-2"].completed)

	_, err := worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 1, "delivery-1"), runScript(host))
	var declared *temporal.ApplicationError
	require.ErrorAs(t, err, &declared)
	require.Equal(t, "transient", declared.Type())

	got := <-second
	require.NoError(t, got.err)
	require.True(t, proto.Equal(textResult("done"), got.result.(*testpilotspb.Value)), got.result)
	first, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, "delivery-1", failedRetryable), first)
	later, err := settledAttempt(t, session, "reservation-2")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 2, "delivery-2", completed), later)
}

// One delivery of an attempt runs its instruction; every other delivery of it, the same task again
// or the attempt under another delivery identity, before or after the first one answered, is given
// the same answer. Nothing runs twice and nothing settles twice. An attempt the script does not
// declare is never answered from another attempt's record: it is refused, and the Driver says so.
func TestDuplicateDeliveryOfAnAttemptIsAnsweredOnceRun(t *testing.T) {
	prepared := preparedActivityFixture(t, retriedActivity)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
	worker := activityWorker(host, definition)
	runs := 0
	counted := func(ctx context.Context) (any, error) {
		runs++
		return host.dynamicActivity(ctx, nil)
	}
	var declared *temporal.ApplicationError

	for _, deliveryID := range []string{"delivery-1", "delivery-1", "redelivery-1"} {
		_, err := worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 1, deliveryID), counted)
		require.ErrorAs(t, err, &declared)
		require.Equal(t, "transient", declared.Type())
	}
	require.Equal(t, 1, runs)
	first, err := settledAttempt(t, session, "reservation-1")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, "delivery-1", failedRetryable), first)
	require.False(t, session.reservations["reservation-2"].consumed)

	for range 2 {
		result, err := worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 2, "delivery-2"), counted)
		require.NoError(t, err)
		require.True(t, proto.Equal(textResult("done"), result.(*testpilotspb.Value)), result)
	}
	require.Equal(t, 2, runs)
	second, err := settledAttempt(t, session, "reservation-2")
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 2, "delivery-2", completed), second)
	// The first attempt, delivered late, is still answered as it was.
	_, err = worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 1, "delivery-1"), counted)
	require.ErrorAs(t, err, &declared)
	require.Equal(t, "transient", declared.Type())

	diagnostics := captureDiagnostics(t, session)
	_, err = worker.activateActivity(t.Context(), activityAttempt(request, "activity-run", 3, "delivery-3"), counted)
	require.ErrorIs(t, err, delivery.ErrAttemptUndeclared)
	requireRefusal(t, err)
	require.Equal(t, 2, runs)
	require.Equal(t, []diagnosed{{
		Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT, Code: "activity_attempt_undeclared",
		Detail: `activity run "activity-run" delivered attempt 3, which its script does not declare`,
	}}, diagnostics.all())
}

// A redelivery is answered only from the answer its attempt recorded. Where the Session holds none,
// the redelivery is refused rather than run again.
func TestRedeliveryWithoutARecordedAnswerIsRefused(t *testing.T) {
	prepared := preparedActivityFixture(t, standaloneActivity)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
	worker := activityWorker(host, definition)
	_, err := worker.activateActivity(t.Context(), activityDelivery(request, "activity-run"), runScript(host))
	require.NoError(t, err)

	delete(session.activityAnswers, "reservation-1")
	ran := false
	_, err = worker.activateActivity(t.Context(), activityDelivery(request, "activity-run"), func(context.Context) (any, error) {
		ran = true
		return nil, nil
	})
	require.ErrorIs(t, err, ErrClosed)
	requireRefusal(t, err)
	require.False(t, ran)
}

// Deliveries of one attempt that race run its instruction once: the others wait for the one that
// was admitted and return its answer.
func TestConcurrentDeliveriesOfOneAttemptRunItsScriptOnce(t *testing.T) {
	prepared := preparedActivityFixture(t, standaloneActivity)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
	worker := activityWorker(host, definition)

	const racers = 6
	entered, proceed := make(chan struct{}, racers), make(chan struct{})
	type answer struct {
		result any
		err    error
	}
	answers := make(chan answer, racers)
	for range racers {
		go func() {
			result, err := worker.activateActivity(context.Background(), activityAttempt(request, "activity-run", 1, "delivery-1"), func(ctx context.Context) (any, error) {
				entered <- struct{}{}
				<-proceed
				return host.dynamicActivity(ctx, nil)
			})
			answers <- answer{result, err}
		}()
	}
	<-entered
	// A delivery that gives up while the admitted one is still under way is refused: there is no
	// answer yet, and it is never handed an unfinished one.
	impatient, giveUp := context.WithTimeout(t.Context(), 20*time.Millisecond)
	_, err := worker.activateActivity(impatient, activityAttempt(request, "activity-run", 1, "delivery-1"), func(context.Context) (any, error) {
		return nil, errors.New("a second delivery ran")
	})
	giveUp()
	requireRefusal(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// Every other delivery is now either waiting on the admitted one or yet to arrive; none may run.
	close(proceed)
	for range racers {
		got := <-answers
		require.NoError(t, got.err)
		require.True(t, proto.Equal(textResult("done"), got.result.(*testpilotspb.Value)), got.result)
	}
	require.Empty(t, entered)
	settled, err := settledActivity(t, session)
	require.NoError(t, err)
	requireOutcome(t, answered("activity-run", 1, "delivery-1", completed), settled)
}

// What the Driver itself does to an attempt is recorded as what the worker answered Temporal with.
// The worker answers every attempt it fails, a canceled one included, with a non-retryable
// application failure of its own type, so the recorded fact is a refusal with the cause as detail:
// the Run never says canceled or timed out where Temporal was told failed. The reservation settles
// with that outcome and no error of its own, so the outcome is what reaches the Run.
func TestActivityAttemptTheDriverFailsIsRecordedAsRefused(t *testing.T) {
	for name, test := range map[string]struct {
		shape func(*testpilotspb.Program)
		run   func(*Driver, *Session) func(context.Context) (any, error)
		cause string
	}{
		"an SDK failure": {
			run: func(*Driver, *Session) func(context.Context) (any, error) {
				return func(context.Context) (any, error) { return nil, errors.New("converter failed") }
			},
			cause: "converter failed",
		},
		"a timeout": {
			run: func(*Driver, *Session) func(context.Context) (any, error) {
				return func(context.Context) (any, error) { return nil, context.DeadlineExceeded }
			},
			cause: "context deadline exceeded",
		},
		"a panic": {
			run: func(*Driver, *Session) func(context.Context) (any, error) {
				return func(context.Context) (any, error) { panic("fault") }
			},
			cause: "activity activation panicked",
		},
		// The Run cancels the reservation while the attempt is under way: the script sees its own
		// activation canceled and performs nothing.
		"a cancellation by the Run": {
			run: func(host *Driver, session *Session) func(context.Context) (any, error) {
				return func(ctx context.Context) (any, error) {
					if err := session.reservations["reservation-1"].Cancel(context.Background()); err != nil {
						return nil, err
					}
					return host.dynamicActivity(ctx, nil)
				}
			},
			cause: "context canceled",
		},
		// The attempt's instruction is disabled, so the script says nothing of this attempt.
		"an attempt whose instruction is disabled": {
			shape: func(program *testpilotspb.Program) {
				standaloneActivity(program)
				program.Entrypoints[1].Instructions[0].Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: false}}}}
			},
			run:   func(host *Driver, _ *Session) func(context.Context) (any, error) { return runScript(host) },
			cause: "the activity attempt's instruction is disabled",
		},
	} {
		t.Run(name, func(t *testing.T) {
			shape := test.shape
			if shape == nil {
				shape = standaloneActivity
			}
			prepared := preparedActivityFixture(t, shape)
			host, definition := runtimeTestDriver(t, prepared)
			session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)
			worker := activityWorker(host, definition)

			result, err := worker.activateActivity(t.Context(), activityDelivery(request, "activity-run"), test.run(host, session))
			require.Nil(t, result)
			requireRefusal(t, err)
			require.ErrorContains(t, err, test.cause)
			settled, waitErr := settledActivity(t, session)
			require.NoError(t, waitErr)
			requireOutcome(t, refused("activity-run", 1, "delivery-1", test.cause), settled)

			// The same delivery again is answered with the failure already reported.
			ran := false
			_, again := worker.activateActivity(t.Context(), activityDelivery(request, "activity-run"), func(context.Context) (any, error) {
				ran = true
				return nil, nil
			})
			requireRefusal(t, again)
			require.False(t, ran)
		})
	}
}

// A task the Driver cannot tie to a live reservation is refused before any script runs and before
// any reservation is consumed.
func TestActivityDeliveryIsRefusedBeforeItsScriptRuns(t *testing.T) {
	lateDelivery := diagnosed{Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: "activity_delivery_late", Detail: "reserved worker delivery rejected after Run closure"}
	for name, test := range map[string]struct {
		disposition delivery.TriggerStatus
		deliver     func(*testing.T, *Session, delivery.ActivityDelivery) delivery.ActivityDelivery
		err         error
		diagnostics []diagnosed
	}{
		"an activity type the worker does not register": {
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				input.ActivityType = "foreign"
				return input
			},
			err: ErrRegistrationConflict,
		},
		"another queue's task": {
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				input.TaskQueue = "foreign"
				return input
			},
			err: ErrRegistrationConflict,
		},
		"a task that carries no route": {
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				input.Header = nil
				return input
			},
			err: delivery.ErrRouteMissing,
		},
		"another activity of the same type": {
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				input.ActivityID = "other"
				return input
			},
			err: delivery.ErrRouteCrossed,
		},
		"a run the start did not answer with": {
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				input.ActivityRunID = "other-run"
				return input
			},
			err: delivery.ErrRouteConflict,
			// The Session says the attempt named a run its start did not reserve.
			diagnostics: []diagnosed{{
				Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT, Code: "activity_run_crossed",
				Detail: `attempt 1 names activity run "other-run", which is not the run its start reserved`,
			}},
		},
		"a task with no attempt number": {
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				input.Attempt = 0
				return input
			},
			err: delivery.ErrInvalid,
		},
		"a start that failed": {
			disposition: delivery.TriggerNonSuccess,
			deliver: func(_ *testing.T, _ *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				return input
			},
			err:         delivery.ErrRouteStale,
			diagnostics: []diagnosed{lateDelivery},
		},
		"a Run that closed": {
			deliver: func(t *testing.T, session *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				require.NoError(t, session.Close(t.Context()))
				return input
			},
			err:         delivery.ErrRouteStale,
			diagnostics: []diagnosed{lateDelivery},
		},
		"a Run whose worker failed": {
			deliver: func(_ *testing.T, session *Session, input delivery.ActivityDelivery) delivery.ActivityDelivery {
				session.workerFailed("task-queue", errors.New("worker failed"))
				return input
			},
			err:         delivery.ErrRouteStale,
			diagnostics: []diagnosed{lateDelivery},
		},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, standaloneActivity)
			host, definition := runtimeTestDriver(t, prepared)
			disposition := test.disposition
			if disposition == 0 {
				disposition = delivery.TriggerSucceeded
			}
			session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", disposition)
			raw := reservationForEntrypoint(t, session, "activity")
			diagnostics := captureDiagnostics(t, session)
			input := test.deliver(t, session, activityDelivery(request, "activity-run"))

			ran := false
			_, err := activityWorker(host, definition).activateActivity(t.Context(), input, func(context.Context) (any, error) {
				ran = true
				return nil, nil
			})
			require.ErrorIs(t, err, test.err)
			requireRefusal(t, err)
			require.False(t, ran)
			require.False(t, raw.consumed)
			require.Equal(t, test.diagnostics, diagnostics.all())
		})
	}
}

// Runs that overlap on one worker keep their own activations, whatever order their tasks arrive in
// and even when two Runs start the same physical activity binding.
func TestConcurrentRunsAdmitTheirOwnActivityDeliveries(t *testing.T) {
	for name, bindings := range map[string][2]delivery.ActivityBinding{
		"distinct activities": {activityBinding("activity-a"), activityBinding("activity-b")},
		"the same binding":    {activityBinding("activity-id"), activityBinding("activity-id")},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, standaloneActivity)
			host, definition := runtimeTestDriver(t, prepared)
			sessionA, _, requestA := activityTestSession(t, host, definition, prepared, "run-a", bindings[0], "activity-run-a", delivery.TriggerSucceeded)
			sessionB, _, requestB := activityTestSession(t, host, definition, prepared, "run-b", bindings[1], "activity-run-b", delivery.TriggerSucceeded)
			worker := activityWorker(host, definition)

			enteredB, proceedB := make(chan struct{}), make(chan struct{})
			activations := make(chan testpilot.Coordinate, 2)
			record := func(ctx context.Context) (any, error) {
				activations <- ctx.Value(activityRouteKey{}).(routedActivity).activation.Coordinate()
				return host.dynamicActivity(ctx, nil)
			}
			doneB := make(chan error, 1)
			go func() {
				_, err := worker.activateActivity(context.Background(), activityDelivery(requestB, "activity-run-b"), func(ctx context.Context) (any, error) {
					close(enteredB)
					<-proceedB
					return record(ctx)
				})
				doneB <- err
			}()
			select {
			case <-enteredB:
			case err := <-doneB:
				require.FailNow(t, "run B's attempt ended before its script ran", err)
			}
			_, err := worker.activateActivity(t.Context(), activityDelivery(requestA, "activity-run-a"), record)
			require.NoError(t, err)
			close(proceedB)
			require.NoError(t, <-doneB)

			require.Equal(t, testpilot.Coordinate{RunID: "run-a", EntrypointID: "activity", ActivationID: "reservation-1", Attempt: 1}, <-activations)
			require.Equal(t, testpilot.Coordinate{RunID: "run-b", EntrypointID: "activity", ActivationID: "reservation-1", Attempt: 1}, <-activations)
			for session, activityRunID := range map[*Session]string{sessionA: "activity-run-a", sessionB: "activity-run-b"} {
				settled, err := settledActivity(t, session)
				require.NoError(t, err)
				requireOutcome(t, answered(activityRunID, 1, "delivery-1", completed), settled)
			}
		})
	}
}

// Closing a Run while its activity attempt is under way cancels that attempt through its
// reservation and still releases the Session. The worker answers Temporal with a failure, so the
// reservation records a refusal, with the cancellation as its cause.
func TestSessionCloseCancelsAnActivityAttemptUnderWay(t *testing.T) {
	prepared := preparedActivityFixture(t, standaloneActivity)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := activityTestSession(t, host, definition, prepared, "run", activityBinding("activity-id"), "activity-run", delivery.TriggerSucceeded)

	// The task's own context outlives the test's interest only briefly, so an attempt the close
	// fails to cancel ends as a timeout the assertions below reject.
	task, expire := context.WithTimeout(context.Background(), 5*time.Second)
	defer expire()
	entered := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := activityWorker(host, definition).activateActivity(task, activityDelivery(request, "activity-run"), func(ctx context.Context) (any, error) {
			close(entered)
			<-ctx.Done()
			return host.dynamicActivity(ctx, nil)
		})
		done <- err
	}()
	select {
	case <-entered:
	case err := <-done:
		require.FailNow(t, "the attempt ended before its script ran", err)
	}
	require.NoError(t, session.Close(t.Context()))
	err := <-done
	require.ErrorIs(t, err, context.Canceled)
	requireRefusal(t, err)
	settled, err := settledActivity(t, session)
	require.NoError(t, err)
	requireOutcome(t, refused("activity-run", 1, "delivery-1", "context canceled"), settled)
	require.Nil(t, host.sessions["run"])
}

func TestCreateActivityCarrierRejectsForeignPhysicalActivityBinding(t *testing.T) {
	prepared := preparedActivityFixture(t, standaloneActivity)
	for name, binding := range map[string]delivery.ActivityBinding{
		"namespace":      {Namespace: "foreign", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "task-queue"},
		"type":           {Namespace: "namespace", ActivityID: "activity-id", ActivityType: "foreign", TaskQueue: "task-queue"},
		"queue":          {Namespace: "namespace", ActivityID: "activity-id", ActivityType: "activity-type", TaskQueue: "foreign"},
		"no activity id": {Namespace: "namespace", ActivityType: "activity-type", TaskQueue: "task-queue"},
	} {
		t.Run(name, func(t *testing.T) {
			host, definition := runtimeTestDriver(t, prepared)
			session, err := newSession(host, "run", "session-run", definition, SessionOptions{Bridge: newTestBridge()})
			require.NoError(t, err)
			host.sessions["run"] = session
			handles, err := session.Reserve(t.Context(), testpilot.ReservationRequest{Origin: activityOrigin("run"), EntrypointID: "activity", Count: 1})
			require.NoError(t, err)
			plan, exists := prepared.ReservationCarrier("controller", "start-activity")
			require.True(t, exists)
			_, err = session.CreateActivityCarrier(t.Context(), activityOrigin("run"), plan, binding, handles)
			require.ErrorIs(t, err, ErrInvalid)
			require.Empty(t, session.carriers)
			require.Zero(t, host.routeAssociations)
		})
	}
}

// Activity routes count against the Driver's route ceiling as workflow routes do, and leave the
// index when their Session's tombstone is evicted.
func TestActivityRoutesAreBoundedAndReleased(t *testing.T) {
	prepared := preparedActivityFixture(t, standaloneActivity)
	host, definition := runtimeTestDriver(t, prepared)
	host.options.maximum, host.options.diagnostics = 2, 1
	first, _, _ := activityTestSession(t, host, definition, prepared, "run-a", activityBinding("activity-a"), "activity-run-a", delivery.TriggerSucceeded)
	second, _, _ := activityTestSession(t, host, definition, prepared, "run-b", activityBinding("activity-b"), "activity-run-b", delivery.TriggerSucceeded)
	require.Equal(t, 2, host.routeAssociations)

	third, err := newSession(host, "run-c", "session-run-c", definition, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	host.sessions["run-c"] = third
	handles, err := third.Reserve(t.Context(), testpilot.ReservationRequest{Origin: activityOrigin("run-c"), EntrypointID: "activity", Count: 1})
	require.NoError(t, err)
	plan, exists := prepared.ReservationCarrier("controller", "start-activity")
	require.True(t, exists)
	_, err = third.CreateActivityCarrier(t.Context(), activityOrigin("run-c"), plan, activityBinding("activity-c"), handles)
	require.ErrorIs(t, err, ErrCapacity)

	// A closed Session stays indexed as a tombstone, so its late deliveries are diagnosed, until a
	// newer tombstone or a newer route needs its place.
	require.NoError(t, first.Close(t.Context()))
	require.NoError(t, second.Close(t.Context()))
	require.Equal(t, map[delivery.ActivityBinding][]*Session{activityBinding("activity-b"): {second}}, host.activityRoutes)
	_, err = third.CreateActivityCarrier(t.Context(), activityOrigin("run-c"), plan, activityBinding("activity-c"), handles)
	require.NoError(t, err)
	require.Equal(t, 2, host.routeAssociations)
	fourth, _, _ := activityTestSession(t, host, definition, prepared, "run-d", activityBinding("activity-d"), "activity-run-d", delivery.TriggerSucceeded)
	require.Equal(t, 2, host.routeAssociations)
	require.Equal(t, map[delivery.ActivityBinding][]*Session{activityBinding("activity-c"): {third}, activityBinding("activity-d"): {fourth}}, host.activityRoutes)
}

func TestActivityProgramRegistersItsActivityTypes(t *testing.T) {
	for name, test := range map[string]struct {
		shape func(*testpilotspb.Program)
		want  queueRegistration
		calls []string
	}{
		"an activity alone": {
			shape: standaloneActivity,
			want:  queueRegistration{namespace: "namespace", queue: "task-queue", workflows: []string{}, activities: []string{"activity-type"}, nexus: []nexusRegistration{}},
			calls: []string{"dynamic workflow", "dynamic activity"},
		},
		"beside a workflow and its handler": {
			shape: besideWorkflow,
			want:  queueRegistration{namespace: "namespace", queue: "task-queue", workflows: []string{"workflow-type"}, activities: []string{"activity-type"}, nexus: []nexusRegistration{{service: "service", operation: "operation"}}},
			calls: []string{"dynamic workflow", "dynamic activity", "nexus service service operation"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, test.shape)
			host, definition := runtimeTestDriver(t, prepared)
			require.Equal(t, []queueRegistration{test.want}, definition.registrations)
			entry := definition.entries["activity"]
			entry.plan = testpilot.EntrypointPlan{}
			require.Equal(t, entryDefinition{namespace: "namespace", queue: "task-queue", activityType: "activity-type"}, entry)

			registrar := &recordingRegistrar{}
			require.NoError(t, host.register(registrar, "task-queue", definition.registrations[0]))
			require.Equal(t, test.calls, registrar.calls)
		})
	}
}

// One queue runs one script per activity type, as it runs one per workflow type.
func TestDefinitionRejectsTwoScriptsForOneActivityType(t *testing.T) {
	definition := programDefinitionFor(t, preparedActivityFixture(t, standaloneActivity))
	entry := definition.entries["activity"]

	delete(definition.entries, "activity")
	require.ErrorIs(t, definition.addEntry(entry, nil), ErrRegistrationConflict)

	delete(definition.entries, "activity")
	entry.activityType = "another-type"
	require.NoError(t, definition.addEntry(entry, nil))
	require.Equal(t, map[string]map[string]struct{}{"task-queue": {"activity-type": {}, "another-type": {}}}, definition.queueActivities)
}

// A queue's registration includes its activity types, so a Run that needs an activity never shares
// the worker of a Run whose registration has none.
func TestActivityRegistrationPoolsOnlyWithTheSameActivities(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	withActivity := pinnedRegistration()
	withActivity.activities = []string{"activity-type"}
	acquire := func(runID string, registration queueRegistration) error {
		lease, err := registry.acquire(t.Context(), runID, []queueRegistration{registration}, false, nil)
		if err == nil {
			t.Cleanup(func() { require.NoError(t, newOutage(lease, OutagePlan{}).Restore(context.Background())) })
		}
		return err
	}
	require.NoError(t, acquire("run-1", pinnedRegistration()))
	require.ErrorIs(t, acquire("run-2", withActivity), ErrRegistrationConflict)
	require.Equal(t, 1, starts)

	require.NoError(t, acquire("run-3", queueRegistration{queue: "activities", activities: []string{"b", "a"}}))
	require.NoError(t, acquire("run-4", queueRegistration{queue: "activities", activities: []string{"a", "b"}}))
	require.ErrorIs(t, acquire("run-5", queueRegistration{queue: "activities", activities: []string{"a"}}), ErrRegistrationConflict)
	require.Equal(t, 2, starts)

	for name, registration := range map[string]queueRegistration{
		"a duplicate activity type": {queue: "other", activities: []string{"a", "a"}},
		"an unnamed activity type":  {queue: "other", activities: []string{""}},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, acquire("run-invalid", registration), ErrRegistrationConflict)
		})
	}
}

// The Driver validates, with no I/O, that it can realize every activity activation the Program
// reserves: the start names the bound namespace and queue by binding identity, and only
// StartActivityExecution carries an activity.
func TestDriverValidatesActivityCarriersBeforeOpen(t *testing.T) {
	valid := preparedActivityFixture(t, standaloneActivity)
	host := symbolicRuntimeDriver(t, valid.Limits())
	require.NoError(t, host.Validate(t.Context(), valid))
	beside := preparedActivityFixture(t, besideWorkflow)
	require.NoError(t, symbolicRuntimeDriver(t, beside.Limits()).Validate(t.Context(), beside))
	retried := preparedActivityFixture(t, retriedActivity)
	require.NoError(t, symbolicRuntimeDriver(t, retried.Limits()).Validate(t.Context(), retried))

	// A start that reserves no activation is an ordinary call: the Program runs no script for the
	// activity, so nothing ties the request to this worker's bindings.
	observed := preparedActivityFixture(t, standaloneActivity, func(program *testpilotspb.Program) {
		program.Entrypoints = program.Entrypoints[:1]
		for _, assignment := range program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc().RequestAssignments[:2] {
			assignment.Value = facadetest.Text("elsewhere")
		}
	})
	require.NoError(t, symbolicRuntimeDriver(t, observed.Limits()).Validate(t.Context(), observed))

	otherNamespace := func(profile *testpilot.ProfileSpec) {
		profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-namespace", Value: "namespace"}, testpilot.EnvironmentBinding{ID: "other-queue", Value: "task-queue"})
	}
	start := func(program *testpilotspb.Program) *testpilotspb.InvokeRpc {
		return program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc()
	}
	for name, modifiers := range map[string][]any{
		"a namespace reached through another binding": {otherNamespace, func(program *testpilotspb.Program) {
			start(program).RequestAssignments[0].Value = symbolicEnvironment("other-namespace")
		}},
		"a literal namespace": {func(program *testpilotspb.Program) {
			start(program).RequestAssignments[0].Value = facadetest.Text("namespace")
		}},
		"a task queue reached through another binding": {otherNamespace, func(program *testpilotspb.Program) {
			start(program).RequestAssignments[1].Value = symbolicEnvironment("other-queue")
		}},
		"no task queue": {func(program *testpilotspb.Program) {
			start(program).RequestAssignments = start(program).RequestAssignments[:1]
		}},
		"another worker role": {func(profile *testpilot.ProfileSpec) {
			profile.Roles = append(profile.Roles, testpilot.RolePolicy{ID: "other-worker", Kind: testpilotspb.ROLE_KIND_WORKER})
		}, func(program *testpilotspb.Program) {
			program.Roles = append(program.Roles, &testpilotspb.Role{RoleId: "other-worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"})
			program.Entrypoints[1].GetActivity().WorkerRoleId = "other-worker"
		}},
		// The Profile lets a workflow start carry the activity; this Driver delivers an activity
		// only through the request that started it.
		"an activity carried by a workflow start": {func(profile *testpilot.ProfileSpec) {
			carriers := profile.Roles[0].ReservationCarriers
			carriers[0].Shapes = []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 8}}
			profile.Roles[0].ReservationCarriers = carriers[:1]
		}, func(program *testpilotspb.Program) {
			call := start(program)
			call.Method = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
			call.RequestAssignments = call.RequestAssignments[:2]
		}},
		// The Profile lets the activity start carry a workflow as well; the request starts only
		// the activity, so nothing would ever deliver that workflow's reservation.
		"a workflow carried by the activity start": {func(profile *testpilot.ProfileSpec) {
			carriers := profile.Roles[0].ReservationCarriers
			carriers[1].Shapes = append(carriers[1].Shapes, testpilot.ReservationCarrierShape{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 8})
			profile.Roles[0].ReservationCarriers = carriers[1:]
		}, func(program *testpilotspb.Program) {
			program.Entrypoints = append(program.Entrypoints, &testpilotspb.Entrypoint{
				EntrypointId: "workflow",
				Activation:   &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "workflow-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
			})
		}},
		"two activities carried by one start": {func(program *testpilotspb.Program) {
			second := activityEntrypoint(facadetest.Text("done"))
			second.EntrypointId = "activity-second"
			second.GetActivity().ActivityType = "activity-type-second"
			program.Entrypoints = append(program.Entrypoints, second)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedActivityFixture(t, standaloneActivity, modifiers...)
			host := symbolicRuntimeDriver(t, prepared.Limits())
			acquisitions := 0
			host.registry = newWorkerRegistry(8, func(string, string, queueRegistration) (managedWorker, error) {
				acquisitions++
				return &fakeManagedWorker{}, nil
			})
			require.ErrorIs(t, host.Validate(t.Context(), prepared), ErrInvalid)
			require.Zero(t, acquisitions)
			require.Empty(t, host.registry.groups)
		})
	}
}
