package delivery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/tools/umpire"
)

func TestCreateBundleUsesExactIdentityAndRetainsRejectedHandles(t *testing.T) {
	for name, mutate := range map[string]func(*fixture, []umpire.ReservationHandle) []umpire.ReservationHandle{
		"partial": func(_ *fixture, handles []umpire.ReservationHandle) []umpire.ReservationHandle { return handles[:1] },
		"duplicate": func(_ *fixture, handles []umpire.ReservationHandle) []umpire.ReservationHandle {
			return []umpire.ReservationHandle{handles[0], handles[0]}
		},
		"crossed origin": func(f *fixture, handles []umpire.ReservationHandle) []umpire.ReservationHandle {
			handles[0].(*fakeReservation).identity.Origin.RunID = "other"
			return handles
		},
		"crossed ordinal": func(f *fixture, handles []umpire.ReservationHandle) []umpire.ReservationHandle {
			handles[0].(*fakeReservation).identity.Ordinal = 1
			return handles
		},
		"malformed id": func(f *fixture, handles []umpire.ReservationHandle) []umpire.ReservationHandle {
			handles[0].(*fakeReservation).identity.ID = ""
			return handles
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "existing-run", "existing-session")
			ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 8, MaxHeaderBytes: 4096, MaxHandles: 8, MaxDiagnostics: 8}})
			require.NoError(t, err)
			origin := f.origin
			origin.RunID = "run"
			workflow := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "workflow"})
			handler := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: "handler"})
			raw := mutate(f, []umpire.ReservationHandle{workflow, handler})
			handles := make([]umpire.ReservationHandle, 0, len(raw))
			for _, handle := range raw {
				retained, retainErr := ledger.RetainReservation(context.Background(), handle)
				if retainErr != nil {
					handles = append(handles, handle)
					continue
				}
				handles = append(handles, retained)
			}
			bundle, err := ledger.CreateBundle(context.Background(), origin, f.plan, f.binding, handles)
			require.Error(t, err)
			require.Len(t, bundle.Handles(), lenNonNil(handles))
		})
	}

	f := newFixture(t, "run", "session")
	duplicatePlan := f.plan
	duplicatePlan.Routes = append(duplicatePlan.Routes, duplicatePlan.Routes[0])
	bundle, err := f.ledger.CreateBundle(context.Background(), umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "second", Attempt: 1}, duplicatePlan, f.binding, nil)
	require.Error(t, err)
	require.Empty(t, bundle.Handles())
}

func TestIdenticalConcurrentRunsRouteByIdentityUnderReorderedDelivery(t *testing.T) {
	first := newFixture(t, "run-one", "session-one")
	second := newFixture(t, "run-two", "session-two")
	firstHeader, secondHeader := workflowHeader(t, first), workflowHeader(t, second)

	secondWorkflow, err := second.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: secondHeader, Namespace: second.binding.Namespace, WorkflowID: second.binding.WorkflowID, WorkflowType: second.binding.WorkflowType, TaskQueue: second.binding.TaskQueue, TemporalRunID: "temporal-two"})
	require.NoError(t, err)
	firstWorkflow, err := first.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: firstHeader, Namespace: first.binding.Namespace, WorkflowID: first.binding.WorkflowID, WorkflowType: first.binding.WorkflowType, TaskQueue: first.binding.TaskQueue, TemporalRunID: "temporal-one"})
	require.NoError(t, err)

	firstNexus, err := first.ledger.PrepareNexus(context.Background(), firstWorkflow, "start-nexus", nil, nil)
	require.NoError(t, err)
	secondNexus, err := second.ledger.PrepareNexus(context.Background(), secondWorkflow, "start-nexus", nil, nil)
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
	require.Equal(t, int64(1), f.workflow.consumeCount.Load())

	delivery.TemporalRunID = "crossed"
	_, err = f.ledger.AdmitWorkflow(context.Background(), delivery)
	require.ErrorIs(t, err, ErrRouteConflict)
	require.Equal(t, int64(1), f.workflow.consumeCount.Load())

	nexusDispatch, err := f.ledger.PrepareNexus(context.Background(), first, "start-nexus", nil, nil)
	require.NoError(t, err)
	handler, err := f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: nexusDispatch.Header(), RequestID: "request"})
	require.NoError(t, err)
	handlerReplay, err := f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: nexusDispatch.Header(), RequestID: "request"})
	require.NoError(t, err)
	require.False(t, handler.Replay())
	require.True(t, handlerReplay.Replay())
	require.Equal(t, int64(1), f.handler.consumeCount.Load())
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
		require.Zero(t, f.workflow.consumeCount.Load())
	})

	t.Run("during admission", func(t *testing.T) {
		f := newFixture(t, "run", "session")
		header := workflowHeader(t, f)
		entered := make(chan struct{})
		f.workflow.consume = func(ctx context.Context) (umpire.Coordinate, error) {
			close(entered)
			<-ctx.Done()
			return umpire.Coordinate{}, ctx.Err()
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
		require.Equal(t, int64(1), f.workflow.cancelCount.Load())
	})
}

func TestTriggerFailuresRetireRoutesAndCancelAdmittedWork(t *testing.T) {
	for _, disposition := range []TriggerDisposition{TriggerRejected, TriggerCanceled, TriggerNonSuccess, TriggerUncertain} {
		t.Run(disposition.String(), func(t *testing.T) {
			f := newFixture(t, "run", "session")
			header := workflowHeader(t, f)
			admitWorkflow(t, f, "temporal-run")
			release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, disposition)
			require.NoError(t, err)
			require.Equal(t, 1, release.Unused())
			require.Equal(t, int64(1), f.workflow.cancelCount.Load())
			require.Equal(t, int64(1), f.handler.cancelCount.Load())
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
	require.Zero(t, f.workflow.cancelCount.Load())
	require.Zero(t, f.handler.cancelCount.Load())
	_, err = f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus", nil, nil)
	require.NoError(t, err)
}

func TestParentTerminalReleasesUnusedOnce(t *testing.T) {
	f := newFixture(t, "run", "session")
	workflow := admitWorkflow(t, f, "temporal-run")
	f.handler.cancel = func(context.Context) error {
		if f.handler.cancelCount.Load() == 1 {
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
	require.Zero(t, f.workflow.cancelCount.Load())
	require.Equal(t, int64(2), f.handler.cancelCount.Load())

	dispatch, err := f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus", nil, nil)
	require.ErrorIs(t, err, ErrRouteStale)
	require.Empty(t, dispatch.Header())
}

func TestParentTerminalDoesNotCancelAdmittedHandler(t *testing.T) {
	f := newFixture(t, "run", "session")
	workflow := admitWorkflow(t, f, "temporal-run")
	dispatch, err := f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus", nil, nil)
	require.NoError(t, err)
	_, err = f.ledger.AdmitNexus(context.Background(), NexusDelivery{Header: dispatch.Header(), RequestID: "request"})
	require.NoError(t, err)

	first, err := f.ledger.ParentTerminal(context.Background(), workflow)
	require.NoError(t, err)
	second, err := f.ledger.ParentTerminal(context.Background(), workflow)
	require.NoError(t, err)
	require.Zero(t, first.Unused())
	require.Zero(t, second.Unused())
	require.Zero(t, f.workflow.cancelCount.Load())
	require.Zero(t, f.handler.cancelCount.Load())
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
				for _, raw := range []*fakeReservation{f.workflow, f.handler} {
					raw.finish()
				}
				for _, handle := range f.bundle.Handles() {
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
	for _, raw := range []*fakeReservation{f.workflow, f.handler} {
		raw.finish()
	}
	for _, handle := range f.bundle.Handles() {
		require.NoError(t, handle.Drain(context.Background()))
	}

	newHandles := func(instruction string) ([]umpire.ReservationHandle, []*fakeReservation) {
		origin := f.origin
		origin.InstructionID = instruction
		raw := []*fakeReservation{
			newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: instruction + "-workflow"}),
			newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: instruction + "-handler"}),
		}
		handles := make([]umpire.ReservationHandle, 0, len(raw))
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
	second, err := f.ledger.CreateBundle(context.Background(), secondOrigin, f.plan, f.binding, secondHandles)
	require.NoError(t, err)
	for _, raw := range secondRaw {
		raw.finish()
	}
	for _, handle := range second.Handles() {
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
	makeBundle := func(instruction string) (Bundle, *fakeReservation, *fakeReservation, error) {
		origin := base.origin
		origin.RunID = "run"
		origin.InstructionID = instruction
		workflow := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: instruction + "-workflow"})
		handler := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: instruction + "-handler"})
		retainedWorkflow, err := ledger.RetainReservation(context.Background(), workflow)
		if err != nil {
			return Bundle{}, workflow, handler, err
		}
		retainedHandler, err := ledger.RetainReservation(context.Background(), handler)
		if err != nil {
			return Bundle{handles: []umpire.EffectHandle{retainedWorkflow}}, workflow, handler, err
		}
		bundle, err := ledger.CreateBundle(context.Background(), origin, base.plan, base.binding, []umpire.ReservationHandle{retainedWorkflow, retainedHandler})
		return bundle, workflow, handler, err
	}
	first, workflow, handler, err := makeBundle("first")
	require.NoError(t, err)
	_, err = ledger.TriggerTerminal(context.Background(), first, TriggerRejected)
	require.NoError(t, err)
	_, _, _, err = makeBundle("second")
	require.ErrorIs(t, err, ErrCapacity)
	require.Equal(t, int64(1), workflow.cancelCount.Load())
	require.Equal(t, int64(1), handler.cancelCount.Load())

	workflow.finish()
	handler.finish()
	for _, handle := range first.Handles() {
		require.NoError(t, handle.Drain(context.Background()))
	}
	_, _, _, err = makeBundle("second")
	require.NoError(t, err)
}

func TestSchedulerVisibleReservationProxyOwnsLifecycle(t *testing.T) {
	f := newFixture(t, "run", "session")
	handles := f.bundle.Handles()
	require.Len(t, handles, 2)
	_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.NoError(t, err)
	require.Equal(t, int64(1), f.workflow.cancelCount.Load())
	require.Equal(t, int64(1), f.handler.cancelCount.Load())

	f.workflow.finish()
	f.handler.finish()
	_, err = handles[0].Wait(context.Background())
	require.NoError(t, err)
	require.NoError(t, handles[1].Drain(context.Background()))

	origin := f.origin
	origin.InstructionID = "next"
	next := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "next-workflow"})
	_, err = f.ledger.RetainReservation(context.Background(), next)
	require.NoError(t, err)
}

func TestStopCancelsAndReleasesUnboundReservation(t *testing.T) {
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 1, MaxHeaderBytes: 4096, MaxHandles: 1, MaxDiagnostics: 1}})
	require.NoError(t, err)
	raw := newFakeReservation(umpire.ReservationIdentity{
		Origin:       umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1},
		EntrypointID: "workflow",
		ID:           "reservation",
	})
	retained, err := ledger.RetainReservation(context.Background(), raw)
	require.NoError(t, err)

	release, err := ledger.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, release.Unused())
	require.Equal(t, int64(1), raw.cancelCount.Load())
	release, err = ledger.Stop(context.Background())
	require.NoError(t, err)
	require.Zero(t, release.Unused())
	require.Equal(t, int64(1), raw.cancelCount.Load())

	raw.finish()
	require.NoError(t, retained.Drain(context.Background()))
}

func TestStopAfterAdmissionCancelsOnlyOwnedRoutes(t *testing.T) {
	f := newFixture(t, "run", "session")
	header := workflowHeader(t, f)
	entered := make(chan struct{})
	finishConsume := make(chan struct{})
	f.workflow.consume = func(context.Context) (umpire.Coordinate, error) {
		close(entered)
		<-finishConsume
		return f.workflow.activation, nil
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
	require.Equal(t, int64(1), f.workflow.cancelCount.Load())
	require.Equal(t, int64(1), f.handler.cancelCount.Load())
}

func TestCrossSessionLifecycleCannotCancelForeignHandles(t *testing.T) {
	first := newFixture(t, "run-one", "session-one")
	second := newFixture(t, "run-two", "session-two")
	firstWorkflow := admitWorkflow(t, first, "temporal-one")

	_, err := second.ledger.TriggerTerminal(context.Background(), first.bundle, TriggerRejected)
	require.ErrorIs(t, err, ErrRouteCrossed)
	_, err = second.ledger.ParentTerminal(context.Background(), firstWorkflow)
	require.ErrorIs(t, err, ErrRouteCrossed)
	require.Zero(t, first.workflow.cancelCount.Load())
	require.Zero(t, first.handler.cancelCount.Load())
	require.Zero(t, second.workflow.cancelCount.Load())
	require.Zero(t, second.handler.cancelCount.Load())
}

func TestReservationProxyLifecycleHonorsLockContextAndCancelState(t *testing.T) {
	f := newFixture(t, "run", "session")
	handle := f.bundle.Handles()[0]
	require.NoError(t, handle.Cancel(context.Background()))
	require.NoError(t, handle.Cancel(context.Background()))
	require.Equal(t, int64(1), f.handler.cancelCount.Load()+f.workflow.cancelCount.Load())

	for _, raw := range []*fakeReservation{f.workflow, f.handler} {
		raw.finish()
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
	handle := first.bundle.Handles()[0]
	called := false
	var finished CompletionFunc
	err := first.ledger.Quarantine(context.Background(), handle, func(_ context.Context, raw umpire.EffectHandle, notify CompletionFunc) error {
		called = true
		require.Same(t, first.handler, raw)
		finished = notify
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, finished)
	require.NoError(t, first.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error {
		t.Fatal("duplicate quarantine callback")
		return nil
	}))

	called = false
	err = second.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, ErrRouteCrossed)
	require.False(t, called)
	_, err = first.ledger.Stop(context.Background())
	require.NoError(t, err)
	require.NoError(t, first.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error { return nil }))

	finished()
	finished()
	next := newFakeReservation(umpire.ReservationIdentity{Origin: first.handler.identity.Origin, EntrypointID: "handler", ID: "next"})
	retained, err := first.ledger.RetainReservation(context.Background(), next)
	require.ErrorIs(t, err, ErrRouteStale)
	require.NotNil(t, retained)
}

func TestQuarantineRegistrationRetryAndActualFinishReleaseCapacity(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.ledger.config.Limits.MaxHandles = 2
	handle := f.bundle.Handles()[0]
	registerCount := 0
	registration := func(_ context.Context, raw umpire.EffectHandle, finished CompletionFunc) error {
		registerCount++
		if registerCount == 1 {
			return errors.New("temporary registration failure")
		}
		require.Same(t, f.handler, raw)
		f.handler.finish()
		finished()
		return nil
	}
	require.ErrorIs(t, f.ledger.Quarantine(context.Background(), handle, registration), ErrLifecycle)
	blocked := newFakeReservation(umpire.ReservationIdentity{Origin: f.handler.identity.Origin, EntrypointID: "handler", ID: "blocked"})
	cleanup, err := f.ledger.RetainReservation(context.Background(), blocked)
	require.ErrorIs(t, err, ErrCapacity)
	require.NoError(t, cleanup.Cancel(context.Background()))
	require.NoError(t, f.ledger.Quarantine(context.Background(), handle, registration))
	require.Equal(t, 2, registerCount)

	next := newFakeReservation(umpire.ReservationIdentity{Origin: f.handler.identity.Origin, EntrypointID: "handler", ID: "next"})
	retained, err := f.ledger.RetainReservation(context.Background(), next)
	require.NoError(t, err)
	require.NotNil(t, retained)
}

func TestQuarantineConcurrentRegistrationIsRetryableAndThenIdempotent(t *testing.T) {
	f := newFixture(t, "run", "session")
	handle := f.bundle.Handles()[0]
	entered := make(chan struct{})
	returnRegistration := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- f.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error {
			close(entered)
			<-returnRegistration
			return nil
		})
	}()
	<-entered
	err := f.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error {
		t.Fatal("concurrent quarantine callback")
		return nil
	})
	require.ErrorIs(t, err, ErrLifecycle)
	close(returnRegistration)
	require.NoError(t, <-first)
	require.NoError(t, f.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error {
		t.Fatal("duplicate quarantine callback")
		return nil
	}))
}

func TestQuarantineCompletionRemainsAuthoritativeAfterRegistrationError(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.ledger.config.Limits.MaxHandles = 2
	handle := f.bundle.Handles()[0]
	err := f.ledger.Quarantine(context.Background(), handle, func(_ context.Context, _ umpire.EffectHandle, finished CompletionFunc) error {
		finished()
		return errors.New("registration returned after completion")
	})
	require.ErrorIs(t, err, ErrLifecycle)
	require.ErrorIs(t, f.ledger.Quarantine(context.Background(), handle, func(context.Context, umpire.EffectHandle, CompletionFunc) error { return nil }), ErrRouteStale)

	next := newFakeReservation(umpire.ReservationIdentity{Origin: f.handler.identity.Origin, EntrypointID: "handler", ID: "next-after-completion"})
	retained, err := f.ledger.RetainReservation(context.Background(), next)
	require.NoError(t, err)
	require.NotNil(t, retained)
}

func TestRetainReservationReturnsCleanupProxyOnRejection(t *testing.T) {
	ledger, err := New(Config{RunID: "run", SessionID: "session", Limits: Limits{MaxRoutes: 1, MaxHeaderBytes: 4096, MaxHandles: 1, MaxDiagnostics: 1}})
	require.NoError(t, err)
	origin := umpire.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1}
	first := newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "workflow", ID: "first"})
	_, err = ledger.RetainReservation(context.Background(), first)
	require.NoError(t, err)

	for name, handle := range map[string]*fakeReservation{
		"capacity": newFakeReservation(umpire.ReservationIdentity{Origin: origin, EntrypointID: "handler", ID: "second"}),
		"identity": newFakeReservation(umpire.ReservationIdentity{Origin: umpire.Coordinate{RunID: "other", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "start", Attempt: 1}, EntrypointID: "handler", ID: "crossed"}),
	} {
		t.Run(name, func(t *testing.T) {
			retained, err := ledger.RetainReservation(context.Background(), handle)
			require.Error(t, err)
			require.NotNil(t, retained)
			require.NotSame(t, handle, retained)
			require.NoError(t, retained.Cancel(context.Background()))
			require.Equal(t, int64(1), handle.cancelCount.Load())
		})
	}
}

func TestTriggerCancellationRetriesFailuresAndAttemptsEveryHandle(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.workflow.cancel = func(context.Context) error {
		if f.workflow.cancelCount.Load() == 1 {
			return errors.New("temporary cancellation failure")
		}
		return nil
	}
	_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.ErrorIs(t, err, ErrLifecycle)
	require.Equal(t, int64(1), f.workflow.cancelCount.Load())
	require.Equal(t, int64(1), f.handler.cancelCount.Load())

	_, err = f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.NoError(t, err)
	require.Equal(t, int64(2), f.workflow.cancelCount.Load())
	require.Equal(t, int64(1), f.handler.cancelCount.Load())
}

func TestTriggerTerminalRetriesFailedAdmissionCancellation(t *testing.T) {
	f := newFixture(t, "run", "session")
	f.workflow.consume = func(context.Context) (umpire.Coordinate, error) {
		return umpire.Coordinate{}, errors.New("activation admission failed")
	}
	f.workflow.cancel = func(context.Context) error {
		if f.workflow.cancelCount.Load() == 1 {
			return errors.New("temporary cancellation failure")
		}
		return nil
	}
	_, err := f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: workflowHeader(t, f), Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
	require.ErrorIs(t, err, ErrLifecycle)
	require.Equal(t, int64(1), f.workflow.cancelCount.Load())

	release, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerRejected)
	require.NoError(t, err)
	require.Equal(t, 1, release.Unused())
	require.Equal(t, int64(2), f.workflow.cancelCount.Load())
	require.Equal(t, int64(1), f.handler.cancelCount.Load())
}

func TestWaitReturnsIndependentFailureSnapshotAfterStop(t *testing.T) {
	f := newFixture(t, "run", "session")
	activation := admitWorkflow(t, f, "temporal-run")
	f.workflow.result.Outcome.ProtocolCode = "worker_failure"
	f.workflow.waitErr = errors.New("activation failed")
	f.workflow.finish()
	result, err := activation.Handle().Wait(context.Background())
	require.EqualError(t, err, "activation failed")
	require.Equal(t, "worker_failure", result.Outcome.ProtocolCode)

	_, err = f.ledger.Stop(context.Background())
	require.NoError(t, err)
	f.workflow.result.Outcome.ProtocolCode = "mutated"
	require.Equal(t, "worker_failure", result.Outcome.ProtocolCode)
	require.Equal(t, "run", activation.Coordinate().RunID)
}

func TestLateTerminalCannotMutateReleasedBundle(t *testing.T) {
	f := newFixture(t, "run", "session")
	activation := admitWorkflow(t, f, "temporal-run")
	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}))
	_, err := f.ledger.TriggerTerminal(context.Background(), f.bundle, TriggerSucceeded)
	require.NoError(t, err)
	for _, raw := range []*fakeReservation{f.workflow, f.handler} {
		raw.finish()
	}
	for _, handle := range f.bundle.Handles() {
		require.NoError(t, handle.Drain(context.Background()))
	}
	_, err = f.ledger.ParentTerminal(context.Background(), activation)
	require.ErrorIs(t, err, ErrRouteStale)
}

func lenNonNil(handles []umpire.ReservationHandle) int {
	count := 0
	for _, handle := range handles {
		if handle != nil {
			count++
		}
	}
	return count
}
