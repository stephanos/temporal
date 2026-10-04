package delivery

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
)

// The expected-handle map built from the admitted plan is what validateHandles checks each handle
// against: a missing, duplicated, crossed, or unexpected handle rejects.
func TestCreateBundleUsesExactIdentity(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func([]testpilot.ReservationHandle) []testpilot.ReservationHandle
		err    error
	}{
		"partial": {func(handles []testpilot.ReservationHandle) []testpilot.ReservationHandle {
			return handles[:1]
		}, ErrInvalid},
		"duplicate": {func(handles []testpilot.ReservationHandle) []testpilot.ReservationHandle {
			return []testpilot.ReservationHandle{handles[0], handles[0]}
		}, ErrRouteConflict},
		"crossed origin": {func(handles []testpilot.ReservationHandle) []testpilot.ReservationHandle {
			handles[0].(*testsupport.Reservation).ID.Origin.RunID = "other"
			return handles
		}, ErrRouteConflict},
		"unexpected ordinal": {func(handles []testpilot.ReservationHandle) []testpilot.ReservationHandle {
			handles[0].(*testsupport.Reservation).ID.Ordinal = 1
			return handles
		}, ErrRouteConflict},
		"unexpected entrypoint": {func(handles []testpilot.ReservationHandle) []testpilot.ReservationHandle {
			handles[0].(*testsupport.Reservation).ID.EntrypointID = "other"
			return handles
		}, ErrRouteConflict},
		"malformed id": {func(handles []testpilot.ReservationHandle) []testpilot.ReservationHandle {
			handles[0].(*testsupport.Reservation).ID.ID = ""
			return handles
		}, ErrRouteConflict},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "existing-run", "existing-session")
			ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
			require.NoError(t, err)
			origin := f.origin
			origin.RunID = "run"
			workflow := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "workflow"})
			handler := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: "handler"})
			raw := tc.mutate([]testpilot.ReservationHandle{workflow, handler})
			handles := make([]testpilot.ReservationHandle, 0, len(raw))
			for _, handle := range raw {
				retained, retainErr := ledger.RetainReservation(context.Background(), handle)
				if retainErr != nil {
					handles = append(handles, handle)
					continue
				}
				handles = append(handles, retained)
			}
			_, err = ledger.CreateBundle(context.Background(), origin, f.plan, f.binding, handles)
			require.ErrorIs(t, err, tc.err)
		})
	}
}

// MaxRoutes bounds a bundle at runtime whatever the admitted plan holds: its reservation and route
// lists, and the handles its reservation counts expect. Each case supplies exactly the handles its
// plan expects, so only the route limit rejects.
func TestCreateBundleRejectsPlanBeyondMaxRoutes(t *testing.T) {
	workflow := func(ordinal int64) reservationKey { return reservationKey{entrypoint: "workflow", ordinal: ordinal} }
	for name, tc := range map[string]struct {
		mutate  func(*testpilot.ReservationCarrierPlan)
		handles []reservationKey
	}{
		"reservations": {func(plan *testpilot.ReservationCarrierPlan) {
			plan.Reservations[1].Count = 0
			plan.Routes = nil
		}, []reservationKey{workflow(0)}},
		"routes": {func(plan *testpilot.ReservationCarrierPlan) {
			plan.Reservations = plan.Reservations[:1]
			plan.Routes = append(plan.Routes, plan.Routes[0])
		}, []reservationKey{workflow(0)}},
		"expected handles": {func(plan *testpilot.ReservationCarrierPlan) {
			plan.Reservations = []testpilot.ReservationTopology{{EntrypointID: "workflow", Kind: testpilot.WorkflowEntrypoint, Count: 2}}
			plan.Routes = nil
		}, []reservationKey{workflow(0), workflow(1)}},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "run", "session")
			ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 1, MaxHeaderBytes: 4096, MaxHandles: 2, MaxDiagnostics: 1}})
			require.NoError(t, err)
			plan := clonePlan(f.plan)
			tc.mutate(&plan)
			handles := make([]testpilot.ReservationHandle, 0, len(tc.handles))
			for _, key := range tc.handles {
				raw := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: f.origin, EntrypointID: key.entrypoint, Ordinal: key.ordinal, ID: fmt.Sprintf("%s-%d", key.entrypoint, key.ordinal)})
				retained, err := ledger.RetainReservation(context.Background(), raw)
				require.NoError(t, err)
				handles = append(handles, retained)
			}
			_, err = ledger.CreateBundle(context.Background(), f.origin, plan, f.binding, handles)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestIdenticalConcurrentRunsRouteByIdentityUnderReorderedDelivery(t *testing.T) {
	first := newFixture(t, "run-one", "session-one")
	second := newFixture(t, "run-two", "session-two")
	firstHeader, secondHeader := workflowHeader(t, first), workflowHeader(t, second)

	secondWorkflow, err := second.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: secondHeader, Namespace: second.binding.Namespace, WorkflowID: second.binding.WorkflowID, WorkflowType: second.binding.WorkflowType, TaskQueue: second.binding.TaskQueue, TemporalRunID: "temporal-two"})
	require.NoError(t, err)
	firstWorkflow, err := first.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: firstHeader, Namespace: first.binding.Namespace, WorkflowID: first.binding.WorkflowID, WorkflowType: first.binding.WorkflowType, TaskQueue: first.binding.TaskQueue, TemporalRunID: "temporal-one"})
	require.NoError(t, err)

	firstNexus, err := first.ledger.PrepareNexus(context.Background(), firstWorkflow, "start-nexus")
	require.NoError(t, err)
	secondNexus, err := second.ledger.PrepareNexus(context.Background(), secondWorkflow, "start-nexus")
	require.NoError(t, err)
	firstHandler, err := first.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: firstNexus.Header(), RequestID: "request-one"})
	require.NoError(t, err)
	secondHandler, err := second.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: secondNexus.Header(), RequestID: "request-two"})
	require.NoError(t, err)
	require.Equal(t, "run-one", firstHandler.Coordinate().RunID)
	require.Equal(t, "run-two", secondHandler.Coordinate().RunID)

	_, err = second.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: firstHeader, Namespace: first.binding.Namespace, WorkflowID: first.binding.WorkflowID, WorkflowType: first.binding.WorkflowType, TaskQueue: first.binding.TaskQueue, TemporalRunID: "temporal-one"})
	require.ErrorIs(t, err, ErrRouteCrossed)
	_, err = second.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: firstNexus.Header(), RequestID: "request-one"})
	require.ErrorIs(t, err, ErrRouteCrossed)
}

func TestMatchingReplayReusesAdmissionAndConflictsReject(t *testing.T) {
	f := newFixture(t, "run", "session")
	delivery := WorkflowDelivery{Header: workflowHeader(t, f), Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"}
	first, err := f.ledger.AdmitWorkflow(context.Background(), delivery)
	require.NoError(t, err)
	replay, err := f.ledger.AdmitWorkflow(context.Background(), delivery)
	require.NoError(t, err)
	require.False(t, first.Replay())
	require.True(t, replay.Replay())
	require.Equal(t, first.Coordinate(), replay.Coordinate())
	require.Equal(t, int64(1), f.workflow.Consumes())

	delivery.TemporalRunID = "crossed"
	_, err = f.ledger.AdmitWorkflow(context.Background(), delivery)
	require.ErrorIs(t, err, ErrRouteConflict)
	require.Equal(t, int64(1), f.workflow.Consumes())

	nexusDispatch, err := f.ledger.PrepareNexus(context.Background(), first, "start-nexus")
	require.NoError(t, err)
	handler, err := f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: nexusDispatch.Header(), RequestID: "request"})
	require.NoError(t, err)
	handlerReplay, err := f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: nexusDispatch.Header(), RequestID: "request"})
	require.NoError(t, err)
	require.False(t, handler.Replay())
	require.True(t, handlerReplay.Replay())
	require.Equal(t, int64(1), f.handler.Consumes())
	_, err = f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: nexusDispatch.Header(), RequestID: "crossed"})
	require.ErrorIs(t, err, ErrRouteConflict)
}

func TestCancellationBeforeAndDuringAdmissionIsAtomic(t *testing.T) {
	t.Run("before admission", func(t *testing.T) {
		f := newFixture(t, "run", "session")
		header := workflowHeader(t, f)
		release, err := f.ledger.Stop(context.Background())
		require.NoError(t, err)
		require.Equal(t, 2, release.Unused())
		_, err = f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: header, Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
		require.ErrorIs(t, err, ErrRouteStale)
		require.Zero(t, f.workflow.Consumes())
	})

	t.Run("during admission", func(t *testing.T) {
		f := newFixture(t, "run", "session")
		header := workflowHeader(t, f)
		entered := make(chan struct{})
		f.workflow.OnConsume = func(ctx context.Context) (testpilot.Coordinate, error) {
			close(entered)
			<-ctx.Done()
			return testpilot.Coordinate{}, ctx.Err()
		}
		ctx, cancel := context.WithCancel(context.Background())
		admitted := make(chan error, 1)
		go func() {
			_, err := f.ledger.AdmitWorkflow(ctx, WorkflowDelivery{Header: header, Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
			admitted <- err
		}()
		<-entered
		lockCtx, lockCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer lockCancel()
		_, err := f.ledger.Stop(lockCtx)
		requireContextError(t, err)
		cancel()
		requireContextError(t, <-admitted)
		_, err = f.ledger.Stop(context.Background())
		require.NoError(t, err)
		require.Equal(t, int64(1), f.workflow.Cancels())
	})
}

func TestTriggerFailuresRetireRoutesAndCancelAdmittedWork(t *testing.T) {
	for _, disposition := range []TriggerStatus{TriggerRejected, TriggerCanceled, TriggerNonSuccess, TriggerUncertain} {
		t.Run(disposition.String(), func(t *testing.T) {
			f := newFixture(t, "run", "session")
			header := workflowHeader(t, f)
			admitWorkflow(t, f, "temporal-run")
			release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, disposition)
			require.NoError(t, err)
			require.Equal(t, 1, release.Unused())
			require.Equal(t, int64(1), f.workflow.Cancels())
			require.Equal(t, int64(1), f.handler.Cancels())
			_, err = f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: header, Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
			require.ErrorIs(t, err, ErrRouteStale)
		})
	}

	f := newFixture(t, "run-success", "session-success")
	workflow := admitWorkflow(t, f, "temporal-run")
	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}))
	release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
	require.NoError(t, err)
	require.Zero(t, release.Unused())
	release, err = f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
	require.NoError(t, err)
	require.Zero(t, release.Unused())
	require.Zero(t, f.workflow.Cancels())
	require.Zero(t, f.handler.Cancels())
	_, err = f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus")
	require.NoError(t, err)
}

func TestParentTerminalReleasesUnusedOnce(t *testing.T) {
	f := newFixture(t, "run", "session")
	workflow := admitWorkflow(t, f, "temporal-run")
	f.handler.OnCancel = func(context.Context) error {
		if f.handler.Cancels() == 1 {
			return errors.New("temporary cancellation failure")
		}
		return nil
	}
	first, err := f.ledger.ParentTerminal(context.Background(), workflow)
	require.ErrorIs(t, err, ErrLifecycle)
	second, err := f.ledger.ParentTerminal(context.Background(), workflow)
	require.NoError(t, err)
	require.Equal(t, 1, first.Unused())
	require.Zero(t, second.Unused())
	require.Zero(t, f.workflow.Cancels())
	require.Equal(t, int64(2), f.handler.Cancels())

	dispatch, err := f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus")
	require.ErrorIs(t, err, ErrRouteStale)
	require.Empty(t, dispatch.Header())
}

func TestParentTerminalDoesNotCancelAdmittedHandler(t *testing.T) {
	f := newFixture(t, "run", "session")
	workflow := admitWorkflow(t, f, "temporal-run")
	dispatch, err := f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus")
	require.NoError(t, err)
	_, err = f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: dispatch.Header(), RequestID: "request"})
	require.NoError(t, err)

	first, err := f.ledger.ParentTerminal(context.Background(), workflow)
	require.NoError(t, err)
	second, err := f.ledger.ParentTerminal(context.Background(), workflow)
	require.NoError(t, err)
	require.Zero(t, first.Unused())
	require.Zero(t, second.Unused())
	require.Zero(t, f.workflow.Cancels())
	require.Zero(t, f.handler.Cancels())
	replay, err := f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: dispatch.Header(), RequestID: "request"})
	require.NoError(t, err)
	require.True(t, replay.Replay())
	_, err = f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: dispatch.Header(), RequestID: "crossed"})
	require.ErrorIs(t, err, ErrRouteConflict)
}

func TestStartResponsePinsAcrossWorkflowTerminalOrdering(t *testing.T) {
	for _, completeHandles := range []bool{false, true} {
		name := "terminal-before-response"
		if completeHandles {
			name = "all-handles-complete-before-response"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "run", "session")
			workflow := admitWorkflow(t, f, "temporal-run")
			_, err := f.ledger.ParentTerminal(context.Background(), workflow)
			require.NoError(t, err)
			if completeHandles {
				for _, raw := range []*testsupport.Reservation{f.workflow, f.handler} {
					raw.Complete()
				}
				for _, handle := range f.handles {
					require.NoError(t, handle.Drain(context.Background()))
				}
			}
			require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}))
			release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
			require.NoError(t, err)
			require.Zero(t, release.Unused())
			if completeHandles {
				require.ErrorIs(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}), ErrRouteStale)
			}
		})
	}
}

func TestCompletedBeforeResponseBundlesRemainBoundedUntilFinalization(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.ledger.config.Limits.MaxRoutes = 2
	admitWorkflow(t, f, "temporal-run")
	for _, raw := range []*testsupport.Reservation{f.workflow, f.handler} {
		raw.Complete()
	}
	for _, handle := range f.handles {
		require.NoError(t, handle.Drain(context.Background()))
	}

	newHandles := func(instruction string) ([]testpilot.ReservationHandle, []*testsupport.Reservation) {
		origin := f.origin
		origin.InstructionID = instruction
		raw := []*testsupport.Reservation{
			testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: instruction + "-workflow"}),
			testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: instruction + "-handler"}),
		}
		handles := make([]testpilot.ReservationHandle, 0, len(raw))
		for _, reservation := range raw {
			retained, err := f.ledger.RetainReservation(context.Background(), reservation)
			require.NoError(t, err)
			handles = append(handles, retained)
		}
		return handles, raw
	}
	secondHandles, secondRaw := newHandles("second")
	secondOrigin := f.origin
	secondOrigin.InstructionID = "second"
	_, err := f.ledger.CreateBundle(context.Background(), secondOrigin, f.plan, f.binding, secondHandles)
	require.NoError(t, err)
	for _, raw := range secondRaw {
		raw.Complete()
	}
	for _, handle := range secondHandles {
		require.NoError(t, handle.Drain(context.Background()))
	}

	thirdHandles, _ := newHandles("third")
	thirdOrigin := f.origin
	thirdOrigin.InstructionID = "third"
	_, err = f.ledger.CreateBundle(context.Background(), thirdOrigin, f.plan, f.binding, thirdHandles)
	require.ErrorIs(t, err, ErrCapacity)

	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}))
	_, err = f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
	require.NoError(t, err)
	_, err = f.ledger.CreateBundle(context.Background(), thirdOrigin, f.plan, f.binding, thirdHandles)
	require.NoError(t, err)
}

func TestCapacityReleasesOnlyAfterActualHandleCompletion(t *testing.T) {
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 2, MaxHeaderBytes: 4096, MaxHandles: 2, MaxDiagnostics: 2}})
	require.NoError(t, err)
	base := newFixture(t, "base", "base-session")
	makeBundle := func(instruction string) (Bundle, []testpilot.ReservationHandle, *testsupport.Reservation, *testsupport.Reservation, error) {
		origin := base.origin
		origin.RunID = "run"
		origin.InstructionID = instruction
		workflow := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: instruction + "-workflow"})
		handler := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: instruction + "-handler"})
		retainedWorkflow, err := ledger.RetainReservation(context.Background(), workflow)
		if err != nil {
			return Bundle{}, nil, workflow, handler, err
		}
		retainedHandler, err := ledger.RetainReservation(context.Background(), handler)
		if err != nil {
			return Bundle{}, nil, workflow, handler, err
		}
		handles := []testpilot.ReservationHandle{retainedWorkflow, retainedHandler}
		bundle, err := ledger.CreateBundle(context.Background(), origin, base.plan, base.binding, handles)
		return bundle, handles, workflow, handler, err
	}
	first, firstHandles, workflow, handler, err := makeBundle("first")
	require.NoError(t, err)
	_, err = ledger.TriggerTerminal(context.Background(), first, TriggerRejected)
	require.NoError(t, err)
	_, _, _, _, err = makeBundle("second")
	require.ErrorIs(t, err, ErrCapacity)
	require.Equal(t, int64(1), workflow.Cancels())
	require.Equal(t, int64(1), handler.Cancels())

	workflow.Complete()
	handler.Complete()
	for _, handle := range firstHandles {
		require.NoError(t, handle.Drain(context.Background()))
	}
	_, _, _, _, err = makeBundle("second")
	require.NoError(t, err)
}

func TestSchedulerVisibleReservationProxyOwnsLifecycle(t *testing.T) {
	f := newFixture(t, "run", "session")
	handles := f.handles
	require.Len(t, handles, 2)
	_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.workflow.Cancels())
	require.Equal(t, int64(1), f.handler.Cancels())

	f.workflow.Complete()
	f.handler.Complete()
	_, err = handles[0].Wait(context.Background())
	require.NoError(t, err)
	require.NoError(t, handles[1].Drain(context.Background()))

	origin := f.origin
	origin.InstructionID = "next"
	next := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "next-workflow"})
	_, err = f.ledger.RetainReservation(context.Background(), next)
	require.NoError(t, err)
}

func TestStopCancelsAndReleasesUnboundReservation(t *testing.T) {
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 1, MaxHeaderBytes: 4096, MaxHandles: 1, MaxDiagnostics: 1}})
	require.NoError(t, err)
	raw := testsupport.NewReservation(testpilot.ReservationIdentity{
		Origin:       testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1},
		EntrypointID: "workflow",
		ID:           "reservation",
	})
	retained, err := ledger.RetainReservation(context.Background(), raw)
	require.NoError(t, err)

	release, err := ledger.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, release.Unused())
	require.Equal(t, int64(1), raw.Cancels())
	release, err = ledger.Stop(context.Background())
	require.NoError(t, err)
	require.Zero(t, release.Unused())
	require.Equal(t, int64(1), raw.Cancels())

	raw.Complete()
	require.NoError(t, retained.Drain(context.Background()))
}

func TestStopAfterAdmissionCancelsOnlyOwnedRoutes(t *testing.T) {
	f := newFixture(t, "run", "session")
	header := workflowHeader(t, f)
	entered := make(chan struct{})
	finishConsume := make(chan struct{})
	f.workflow.OnConsume = func(context.Context) (testpilot.Coordinate, error) {
		close(entered)
		<-finishConsume
		return f.workflow.Activation, nil
	}
	admission := make(chan error, 1)
	go func() {
		_, err := f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: header, Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
		admission <- err
	}()
	<-entered
	stopStarted := make(chan struct{})
	stopRelease := make(chan Release, 1)
	stopErr := make(chan error, 1)
	go func() {
		close(stopStarted)
		release, err := f.ledger.Stop(context.Background())
		stopRelease <- release
		stopErr <- err
	}()
	<-stopStarted
	close(finishConsume)
	require.NoError(t, <-admission)
	require.NoError(t, <-stopErr)
	require.Equal(t, 1, (<-stopRelease).Unused())
	require.Equal(t, int64(1), f.workflow.Cancels())
	require.Equal(t, int64(1), f.handler.Cancels())
}

func TestCrossSessionLifecycleCannotCancelForeignHandles(t *testing.T) {
	first := newFixture(t, "run-one", "session-one")
	second := newFixture(t, "run-two", "session-two")
	firstWorkflow := admitWorkflow(t, first, "temporal-one")

	_, err := second.ledger.TriggerTerminal(context.Background(), first.bundle, TriggerRejected)
	require.ErrorIs(t, err, ErrRouteCrossed)
	_, err = second.ledger.ParentTerminal(context.Background(), firstWorkflow)
	require.ErrorIs(t, err, ErrRouteCrossed)
	require.Zero(t, first.workflow.Cancels())
	require.Zero(t, first.handler.Cancels())
	require.Zero(t, second.workflow.Cancels())
	require.Zero(t, second.handler.Cancels())
}

func TestReservationProxyLifecycleHonorsLockContextAndCancelState(t *testing.T) {
	f := newFixture(t, "run", "session")
	handle := f.handles[0]
	require.NoError(t, handle.Cancel(context.Background()))
	require.NoError(t, handle.Cancel(context.Background()))
	require.Equal(t, int64(1), f.handler.Cancels()+f.workflow.Cancels())

	for _, raw := range []*testsupport.Reservation{f.workflow, f.handler} {
		raw.Complete()
	}
	f.ledger.mu.Lock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := handle.Drain(ctx)
	cancel()
	f.ledger.mu.Unlock()
	requireContextError(t, err)
	require.NoError(t, handle.Drain(context.Background()))
}

func TestQuarantineKeepsReservationOwnershipAndUnwrapsExactHandle(t *testing.T) {
	first := newFixture(t, "run-one", "session-one")
	second := newFixture(t, "run-two", "session-two")
	handle := first.handles[0]
	called := false
	var finished CompletionFunc
	err := first.ledger.Quarantine(context.Background(), handle, func(_ context.Context, raw testpilot.EffectHandle, notify CompletionFunc) error {
		called = true
		require.Same(t, first.handler, raw)
		finished = notify
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, finished)
	require.NoError(t, first.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error {
		t.Fatal("duplicate quarantine callback")
		return nil
	}))

	called = false
	err = second.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, ErrRouteCrossed)
	require.False(t, called)
	_, err = first.ledger.Stop(context.Background())
	require.NoError(t, err)
	require.NoError(t, first.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error { return nil }))

	finished()
	finished()
	next := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: first.handler.ID.Origin, EntrypointID: "handler", ID: "next"})
	retained, err := first.ledger.RetainReservation(context.Background(), next)
	require.ErrorIs(t, err, ErrRouteStale)
	require.NotNil(t, retained)
}

func TestQuarantineRegistrationRetryAndActualFinishReleaseCapacity(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.ledger.config.Limits.MaxHandles = 2
	handle := f.handles[0]
	registerCount := 0
	registration := func(_ context.Context, raw testpilot.EffectHandle, finished CompletionFunc) error {
		registerCount++
		if registerCount == 1 {
			return errors.New("temporary registration failure")
		}
		require.Same(t, f.handler, raw)
		f.handler.Complete()
		finished()
		return nil
	}
	require.ErrorIs(t, f.ledger.Quarantine(context.Background(), handle, registration), ErrLifecycle)
	blocked := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: f.handler.ID.Origin, EntrypointID: "handler", ID: "blocked"})
	cleanup, err := f.ledger.RetainReservation(context.Background(), blocked)
	require.ErrorIs(t, err, ErrCapacity)
	require.NoError(t, cleanup.Cancel(context.Background()))
	require.NoError(t, f.ledger.Quarantine(context.Background(), handle, registration))
	require.Equal(t, 2, registerCount)

	next := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: f.handler.ID.Origin, EntrypointID: "handler", ID: "next"})
	retained, err := f.ledger.RetainReservation(context.Background(), next)
	require.NoError(t, err)
	require.NotNil(t, retained)
}

func TestQuarantineConcurrentRegistrationIsRetryableAndThenIdempotent(t *testing.T) {
	f := newFixture(t, "run", "session")
	handle := f.handles[0]
	entered := make(chan struct{})
	returnRegistration := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- f.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error {
			close(entered)
			<-returnRegistration
			return nil
		})
	}()
	<-entered
	err := f.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error {
		t.Fatal("concurrent quarantine callback")
		return nil
	})
	require.ErrorIs(t, err, ErrLifecycle)
	close(returnRegistration)
	require.NoError(t, <-first)
	require.NoError(t, f.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error {
		t.Fatal("duplicate quarantine callback")
		return nil
	}))
}

func TestQuarantineCompletionRemainsAuthoritativeAfterRegistrationError(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.ledger.config.Limits.MaxHandles = 2
	handle := f.handles[0]
	err := f.ledger.Quarantine(context.Background(), handle, func(_ context.Context, _ testpilot.EffectHandle, finished CompletionFunc) error {
		finished()
		return errors.New("registration returned after completion")
	})
	require.ErrorIs(t, err, ErrLifecycle)
	require.ErrorIs(t, f.ledger.Quarantine(context.Background(), handle, func(context.Context, testpilot.EffectHandle, CompletionFunc) error { return nil }), ErrRouteStale)

	next := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: f.handler.ID.Origin, EntrypointID: "handler", ID: "next-after-completion"})
	retained, err := f.ledger.RetainReservation(context.Background(), next)
	require.NoError(t, err)
	require.NotNil(t, retained)
}

func TestRetainReservationReturnsCleanupProxyOnRejection(t *testing.T) {
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 1, MaxHeaderBytes: 4096, MaxHandles: 1, MaxDiagnostics: 1}})
	require.NoError(t, err)
	origin := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1}
	first := testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "first"})
	_, err = ledger.RetainReservation(context.Background(), first)
	require.NoError(t, err)

	for name, handle := range map[string]*testsupport.Reservation{
		"capacity": testsupport.NewReservation(testpilot.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: "second"}),
		"identity": testsupport.NewReservation(testpilot.ReservationIdentity{Origin: testpilot.Coordinate{RunID: "other", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1}, EntrypointID: "handler", ID: "crossed"}),
	} {
		t.Run(name, func(t *testing.T) {
			retained, err := ledger.RetainReservation(context.Background(), handle)
			require.Error(t, err)
			require.NotNil(t, retained)
			require.NotSame(t, handle, retained)
			require.NoError(t, retained.Cancel(context.Background()))
			require.Equal(t, int64(1), handle.Cancels())
		})
	}
}

func TestTriggerCancellationRetriesFailuresAndAttemptsEveryHandle(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.workflow.OnCancel = func(context.Context) error {
		if f.workflow.Cancels() == 1 {
			return errors.New("temporary cancellation failure")
		}
		return nil
	}
	_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.ErrorIs(t, err, ErrLifecycle)
	require.Equal(t, int64(1), f.workflow.Cancels())
	require.Equal(t, int64(1), f.handler.Cancels())

	_, err = f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.NoError(t, err)
	require.Equal(t, int64(2), f.workflow.Cancels())
	require.Equal(t, int64(1), f.handler.Cancels())
}

func TestTriggerTerminalRetriesFailedAdmissionCancellation(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.workflow.OnConsume = func(context.Context) (testpilot.Coordinate, error) {
		return testpilot.Coordinate{}, errors.New("activation admission failed")
	}
	f.workflow.OnCancel = func(context.Context) error {
		if f.workflow.Cancels() == 1 {
			return errors.New("temporary cancellation failure")
		}
		return nil
	}
	_, err := f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: workflowHeader(t, f), Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
	require.ErrorIs(t, err, ErrLifecycle)
	require.Equal(t, int64(1), f.workflow.Cancels())

	release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.NoError(t, err)
	require.Equal(t, 1, release.Unused())
	require.Equal(t, int64(2), f.workflow.Cancels())
	require.Equal(t, int64(1), f.handler.Cancels())
}

func TestWaitReturnsIndependentFailureSnapshotAfterStop(t *testing.T) {
	f := newFixture(t, "run", "session")
	activation := admitWorkflow(t, f, "temporal-run")
	f.workflow.Result.Outcome.ProtocolCode = "worker_failure"
	f.workflow.WaitErr = errors.New("activation failed")
	f.workflow.Complete()
	var handle testpilot.EffectHandle
	for _, retained := range f.handles {
		if retained.Identity().EntrypointID == "workflow" {
			handle = retained
		}
	}
	require.NotNil(t, handle)
	result, err := handle.Wait(context.Background())
	require.EqualError(t, err, "activation failed")
	require.Equal(t, "worker_failure", result.Outcome.ProtocolCode)

	_, err = f.ledger.Stop(context.Background())
	require.NoError(t, err)
	f.workflow.Result.Outcome.ProtocolCode = "mutated"
	require.Equal(t, "worker_failure", result.Outcome.ProtocolCode)
	require.Equal(t, "run", activation.Coordinate().RunID)
}

func TestLateTerminalCannotMutateReleasedBundle(t *testing.T) {
	f := newFixture(t, "run", "session")
	activation := admitWorkflow(t, f, "temporal-run")
	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}))
	_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
	require.NoError(t, err)
	for _, raw := range []*testsupport.Reservation{f.workflow, f.handler} {
		raw.Complete()
	}
	for _, handle := range f.handles {
		require.NoError(t, handle.Drain(context.Background()))
	}
	_, err = f.ledger.ParentTerminal(context.Background(), activation)
	require.ErrorIs(t, err, ErrRouteStale)
}
