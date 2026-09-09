package execution

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/await"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type runtimeDriver struct {
	identity DriverIdentity
	session  *runtimeSession
}

func (d *runtimeDriver) Identity(context.Context) (DriverIdentity, error) { return d.identity, nil }
func (d *runtimeDriver) Validate(context.Context, *PreparedProgram) error { return nil }
func (d *runtimeDriver) Open(context.Context, string, *PreparedProgram) (Session, error) {
	return d.session, nil
}

type runtimeSession struct {
	Session
	mu            sync.Mutex
	effects       map[string]*runtimeEffect
	invokeErr     map[string]error
	invocations   []string
	quarantined   []EffectHandle
	diagnostics   []*testpilotspb.RunDiagnostic
	quarantine    int
	quarantineErr error
	closed        int
	closeErr      error
	closeTimeout  bool
}

func (s *runtimeSession) InvokeRPC(_ context.Context, c Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message) (EffectHandle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invocations = append(s.invocations, c.InstructionID)
	return s.effects[c.InstructionID], s.invokeErr[c.InstructionID]
}
func (s *runtimeSession) Quarantine(_ context.Context, handle EffectHandle) error {
	s.mu.Lock()
	if s.quarantineErr != nil {
		defer s.mu.Unlock()
		return s.quarantineErr
	}
	s.quarantined = append(s.quarantined, handle)
	s.quarantine++
	s.mu.Unlock()
	effect := handle.(*runtimeEffect)
	go func() {
		<-effect.done
		s.mu.Lock()
		defer s.mu.Unlock()
		s.quarantine--
		s.diagnostics = append(s.diagnostics, &testpilotspb.RunDiagnostic{Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: "quarantine_completed"})
	}()
	return nil
}
func (s *runtimeSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed++
	if s.closeTimeout {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.closeErr
}
func (s *runtimeSession) Diagnose(_ context.Context, _ string, diagnostic *testpilotspb.RunDiagnostic) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diagnostics = append(s.diagnostics, proto.CloneOf(diagnostic))
	return nil
}

type runtimeEffect struct {
	done            chan struct{}
	result          EffectResult
	completeOnce    sync.Once
	cancelCompletes bool
	canceled        atomic.Bool
	waitErr         error
	drainErr        error
	cancelFn        func(context.Context) error
}

func newRuntimeEffect(result EffectResult, complete bool) *runtimeEffect {
	effect := &runtimeEffect{done: make(chan struct{}), result: result, cancelCompletes: true}
	if complete {
		effect.complete()
	}
	return effect
}
func (e *runtimeEffect) Wait(ctx context.Context) (EffectResult, error) {
	select {
	case <-e.done:
		return e.result, e.waitErr
	case <-ctx.Done():
		select {
		case <-e.done:
			return e.result, e.waitErr
		default:
			return EffectResult{}, ctx.Err()
		}
	}
}
func (e *runtimeEffect) Cancel(ctx context.Context) error {
	e.canceled.Store(true)
	if e.cancelFn != nil {
		return e.cancelFn(ctx)
	}
	if e.cancelCompletes {
		e.complete()
	}
	return nil
}
func (e *runtimeEffect) Drain(ctx context.Context) error {
	if e.drainErr != nil {
		return e.drainErr
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return nil
	}
}

func (e *runtimeEffect) complete() {
	e.completeOnce.Do(func() { close(e.done) })
}

type runtimeMonitor struct {
	Monitor
	stopSource string
	violated   bool
	closeKind  testpilotspb.VerdictStatus
	closeErr   error
}

func (m *runtimeMonitor) Observe(_ context.Context, event *testpilotspb.RunEvent) (Decision, error) {
	if event.GetSourceId() == m.stopSource {
		m.violated = true
		return Stop, nil
	}
	return Continue, nil
}
func (m *runtimeMonitor) Close(context.Context, *testpilotspb.Run) (*testpilotspb.Verdict, error) {
	if m.violated {
		return &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_VIOLATED}, m.closeErr
	}
	kind := m.closeKind
	if kind == testpilotspb.VERDICT_STATUS_UNSPECIFIED {
		kind = testpilotspb.VERDICT_STATUS_SATISFIED
	}
	return &testpilotspb.Verdict{Status: kind}, m.closeErr
}

func TestRunStopDrainsQuarantinesAndCannotSuppressFreshCleanup(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxTotalDurationMilliseconds = 1000
	c.Program.Limits.MaxCleanupDurationMilliseconds = 1000
	late := rpcNode("late")
	quarantine := rpcNode("quarantine")
	after := rpcNode("after")
	after.Dependencies = []*testpilotspb.InstructionRef{{EntrypointId: "controller", InstructionId: "call"}}
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, late, quarantine, after)
	cleanupNode := rpcNode("cleanup")
	cleanupNode.Limits.TimeoutMilliseconds = 10
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionDefinition{cleanupNode}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	complete := newRuntimeEffect(effectResponse(prepared, "complete"), true)
	lateEffect := newRuntimeEffect(effectResponse(prepared, "late"), false)
	quarantined := newRuntimeEffect(effectResponse(prepared, "quarantined"), false)
	quarantined.cancelCompletes = false
	quarantined.drainErr = context.DeadlineExceeded
	cleanup := newRuntimeEffect(effectResponse(prepared, "cleanup"), true)
	session := &runtimeSession{effects: map[string]*runtimeEffect{
		"call": complete, "late": lateEffect, "quarantine": quarantined, "cleanup": cleanup,
	}}
	driver := &runtimeDriver{identity: DriverIdentity{Profile: policy.Identity, Catalog: policy.CatalogIdentity}, session: session}

	run, verdict, err := Run(t.Context(), prepared, driver, &runtimeMonitor{stopSource: "scheduler.g0.n0.a1.completed"}, "run", c.CaseId)

	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR, run.GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, verdict.GetStatus())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.NotContains(t, session.invocations, "after")
	require.Contains(t, session.invocations, "cleanup")
	require.Contains(t, eventSources(run), "scheduler.g0.n1.a1.completed")
	require.True(t, lateEffect.canceled.Load())
	require.True(t, quarantined.canceled.Load())
	require.Contains(t, session.quarantined, quarantined)
	require.Equal(t, 1, session.closed)
	serialized, err := proto.Marshal(run)
	require.NoError(t, err)
	serializedVerdict, err := proto.Marshal(verdict)
	require.NoError(t, err)
	quarantined.complete()
	await.RequireTrue(t, func() bool {
		session.mu.Lock()
		defer session.mu.Unlock()
		return len(session.diagnostics) > 0 && session.quarantine == 0
	}, time.Second, time.Millisecond)
	afterBytes, err := proto.Marshal(run)
	require.NoError(t, err)
	require.Equal(t, serialized, afterBytes)
	afterVerdictBytes, err := proto.Marshal(verdict)
	require.NoError(t, err)
	require.Equal(t, serializedVerdict, afterVerdictBytes)
}

func TestRunRejectsInvalidInputsBeforeOpening(t *testing.T) {
	_, _, err := Run(context.TODO(), nil, nil, nil, "", "")
	require.Error(t, err)
	require.NotErrorIs(t, err, context.Canceled)
}

type deadlineMonitor struct {
	Monitor
	expired chan struct{}
	release chan struct{}
}

func (m *deadlineMonitor) Observe(context.Context, *testpilotspb.RunEvent) (Decision, error) {
	return Continue, nil
}

func (m *deadlineMonitor) Close(ctx context.Context, _ *testpilotspb.Run) (*testpilotspb.Verdict, error) {
	<-ctx.Done()
	close(m.expired)
	<-m.release
	return &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_SATISFIED}, nil
}

func TestRunWaitsForLateMonitorAndThenReportsDeadlineViolation(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Entrypoints[0].Instructions = nil
	c.Program.Limits.MaxTotalDurationMilliseconds = 10
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	monitor := &deadlineMonitor{expired: make(chan struct{}), release: make(chan struct{})}
	session := &runtimeSession{}
	done := make(chan struct {
		run     *testpilotspb.Run
		verdict *testpilotspb.Verdict
		err     error
	}, 1)
	go func() {
		run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
		done <- struct {
			run     *testpilotspb.Run
			verdict *testpilotspb.Verdict
			err     error
		}{run: run, verdict: verdict, err: err}
	}()
	<-monitor.expired
	select {
	case <-done:
		t.Fatal("Run returned while the synchronous Monitor callback was still executing")
	default:
	}
	close(monitor.release)
	result := <-done
	// The recorder's close failed, so Run surfaces it beside the record it still produced.
	require.ErrorIs(t, result.err, context.DeadlineExceeded)
	require.Equal(t, testpilotspb.RUN_STATUS_INCOMPLETE, result.run.GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, result.verdict.GetStatus())
	require.Contains(t, diagnosticCodes(result.run), "close_failed")
}

type canceledMonitor struct{ Monitor }

func (*canceledMonitor) Observe(context.Context, *testpilotspb.RunEvent) (Decision, error) {
	return Continue, nil
}

func (*canceledMonitor) Close(ctx context.Context, _ *testpilotspb.Run) (*testpilotspb.Verdict, error) {
	<-ctx.Done()
	return &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE}, ctx.Err()
}

func TestRunConformingMonitorCancellationIsInconclusive(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Entrypoints[0].Instructions = nil
	c.Program.Limits.MaxTotalDurationMilliseconds = 10
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: &runtimeSession{}}, &canceledMonitor{}, "run", c.CaseId)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, testpilotspb.RUN_STATUS_INCOMPLETE, run.GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	require.Contains(t, diagnosticCodes(run), "close_failed")
}

func TestRunTerminalPrecedence(t *testing.T) {
	for _, test := range []struct {
		name             string
		ordinaryErr      error
		stop             bool
		closeKind        testpilotspb.VerdictStatus
		cleanupErr       error
		hostCloseErr     error
		hostCloseTimeout bool
		wantDisposition  testpilotspb.RunStatus
		wantCleanup      testpilotspb.CleanupStatus
		wantVerdict      testpilotspb.VerdictStatus
	}{
		{name: "complete", wantDisposition: testpilotspb.RUN_STATUS_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
		{name: "early liveness closure", closeKind: testpilotspb.VERDICT_STATUS_INCONCLUSIVE, wantDisposition: testpilotspb.RUN_STATUS_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED, wantVerdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{name: "execution failure", ordinaryErr: errors.New("effect failed"), wantDisposition: testpilotspb.RUN_STATUS_INCOMPLETE, wantCleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED, wantVerdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{name: "close error preserves success", hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_STATUS_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
		{name: "close timeout preserves success", hostCloseTimeout: true, wantDisposition: testpilotspb.RUN_STATUS_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
		{name: "close error preserves violation", stop: true, hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_VIOLATED},
		{name: "close timeout preserves violation", stop: true, hostCloseTimeout: true, wantDisposition: testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_VIOLATED},
		{name: "violation dominates cleanup and close", stop: true, cleanupErr: errors.New("cleanup failed"), hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_VIOLATED},
		{name: "cleanup and close do not replace success", cleanupErr: errors.New("cleanup failed"), hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_STATUS_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			c.Program.Limits.MaxCleanupDurationMilliseconds = 100
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionDefinition{rpcNode("cleanup")}
			c.Program.Cleanup.Instructions[0].Limits.TimeoutMilliseconds = 100
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			ordinary := newRuntimeEffect(effectResponse(prepared, "ordinary"), true)
			ordinary.waitErr = test.ordinaryErr
			cleanup := newRuntimeEffect(effectResponse(prepared, "cleanup"), true)
			cleanup.waitErr = test.cleanupErr
			session := &runtimeSession{effects: map[string]*runtimeEffect{"call": ordinary, "cleanup": cleanup}, closeErr: test.hostCloseErr, closeTimeout: test.hostCloseTimeout}
			monitor := &runtimeMonitor{closeKind: test.closeKind}
			if test.stop {
				monitor.stopSource = "scheduler.g0.n0.a1.completed"
			}
			run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
			require.NoError(t, err)
			require.Equal(t, test.wantDisposition, run.GetStatus())
			require.Equal(t, test.wantCleanup, run.GetCleanup().GetStatus())
			require.Equal(t, test.wantVerdict, verdict.GetStatus())
			require.True(t, proto.Equal(verdict, run.GetVerdict()))
			var cleanupDiagnosticIDs []string
			var cleanupDiagnosticCodes []string
			for _, diagnostic := range run.GetDiagnostics() {
				if diagnostic.GetCode() == "cleanup_failed" || diagnostic.GetCode() == "driver_close_failed" {
					cleanupDiagnosticIDs = append(cleanupDiagnosticIDs, diagnostic.GetDiagnosticId())
					cleanupDiagnosticCodes = append(cleanupDiagnosticCodes, diagnostic.GetCode())
				}
			}
			require.Equal(t, cleanupDiagnosticIDs, run.GetCleanup().GetDiagnosticIds())
			if test.hostCloseErr != nil || test.hostCloseTimeout {
				require.Contains(t, cleanupDiagnosticCodes, "driver_close_failed")
			}
		})
	}
}

func TestRunCleanupDeadlineDoesNotReplaceOrdinarySuccess(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxCleanupDurationMilliseconds = 10
	cleanupNode := rpcNode("cleanup")
	cleanupNode.Limits.TimeoutMilliseconds = 10
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionDefinition{cleanupNode}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	session := &runtimeSession{effects: map[string]*runtimeEffect{
		"call":    newRuntimeEffect(effectResponse(prepared, "ordinary"), true),
		"cleanup": newRuntimeEffect(effectResponse(prepared, "cleanup"), false),
	}}

	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, &runtimeMonitor{}, "run", c.CaseId)

	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, run.GetStatus())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Contains(t, diagnosticCodes(run), "cleanup_failed")
}

func TestRunBoundsHostContextViolationAndQuarantineCapacityFailure(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxCleanupDurationMilliseconds = 10
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	effect := newRuntimeEffect(effectResponse(prepared, "complete"), true)
	effect.cancelFn = func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}
	effect.drainErr = context.DeadlineExceeded
	session := &runtimeSession{
		effects:       map[string]*runtimeEffect{"call": effect},
		quarantineErr: errors.New("quarantine capacity exhausted"),
	}
	monitor := &runtimeMonitor{stopSource: "scheduler.g0.n0.a1.completed"}
	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR, run.GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, verdict.GetStatus())
	require.Subset(t, diagnosticCodes(run), []string{"effect_cancel_context_violated", "quarantine_failed"})
}

func diagnosticCodes(run *testpilotspb.Run) []string {
	codes := make([]string, 0, len(run.GetDiagnostics()))
	for _, diagnostic := range run.GetDiagnostics() {
		codes = append(codes, diagnostic.GetCode())
	}
	return codes
}

func eventSources(run *testpilotspb.Run) []string {
	sources := make([]string, 0, len(run.GetEvents()))
	for _, event := range run.GetEvents() {
		sources = append(sources, event.GetSourceId())
	}
	return sources
}
