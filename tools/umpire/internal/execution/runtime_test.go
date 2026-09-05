package execution

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/await"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type runtimeDriver struct {
	identity DriverIdentity
	session  *runtimeSession
}

func (d *runtimeDriver) Identity(context.Context) (DriverIdentity, error) { return d.identity, nil }
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
	diagnostics   []*umpirespb.RunDiagnostic
	quarantine    int
	quarantineErr error
	closed        int
	closeErr      error
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
		s.diagnostics = append(s.diagnostics, &umpirespb.RunDiagnostic{Kind: umpirespb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: "quarantine_completed"})
	}()
	return nil
}
func (s *runtimeSession) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed++
	return s.closeErr
}
func (s *runtimeSession) Diagnose(_ context.Context, _ string, diagnostic *umpirespb.RunDiagnostic) error {
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
	closeKind  umpirespb.VerdictKind
	closeErr   error
}

func (m *runtimeMonitor) Observe(_ context.Context, event *umpirespb.RunEvent) (Decision, error) {
	if event.GetSourceId() == m.stopSource {
		m.violated = true
		return Stop, nil
	}
	return Continue, nil
}
func (m *runtimeMonitor) Close(context.Context, *umpirespb.Run) (*umpirespb.Verdict, error) {
	if m.violated {
		return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_VIOLATED}, m.closeErr
	}
	kind := m.closeKind
	if kind == umpirespb.VERDICT_KIND_UNSPECIFIED {
		kind = umpirespb.VERDICT_KIND_SATISFIED
	}
	return &umpirespb.Verdict{Kind: kind}, m.closeErr
}

func TestRunStopDrainsQuarantinesAndCannotSuppressFreshCleanup(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxTotalDurationMilliseconds = 1000
	c.Program.Limits.MaxCleanupDurationMilliseconds = 1000
	late := rpcNode("late")
	quarantine := rpcNode("quarantine")
	after := rpcNode("after")
	after.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
	c.Program.Entrypoints[0].Nodes = append(c.Program.Entrypoints[0].Nodes, late, quarantine, after)
	cleanupNode := rpcNode("cleanup")
	cleanupNode.Bounds.TimeoutMilliseconds = 10
	c.Program.Cleanup.Nodes = []*umpirespb.InstructionNode{cleanupNode}
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
	require.Equal(t, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition())
	require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, verdict.GetKind())
	require.Equal(t, umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
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

func (m *deadlineMonitor) Observe(context.Context, *umpirespb.RunEvent) (Decision, error) {
	return Continue, nil
}

func (m *deadlineMonitor) Close(ctx context.Context, _ *umpirespb.Run) (*umpirespb.Verdict, error) {
	<-ctx.Done()
	close(m.expired)
	<-m.release
	return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_SATISFIED}, nil
}

func TestRunWaitsForLateMonitorAndThenReportsDeadlineViolation(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Entrypoints[0].Nodes = nil
	c.Program.Limits.MaxTotalDurationMilliseconds = 10
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	monitor := &deadlineMonitor{expired: make(chan struct{}), release: make(chan struct{})}
	session := &runtimeSession{}
	done := make(chan struct {
		run     *umpirespb.Run
		verdict *umpirespb.Verdict
		err     error
	}, 1)
	go func() {
		run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
		done <- struct {
			run     *umpirespb.Run
			verdict *umpirespb.Verdict
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
	require.NoError(t, result.err)
	require.Equal(t, umpirespb.RUN_DISPOSITION_INCOMPLETE, result.run.GetDisposition())
	require.Equal(t, umpirespb.VERDICT_KIND_INCONCLUSIVE, result.verdict.GetKind())
	require.Contains(t, diagnosticCodes(result.run), "close_failed")
}

type canceledMonitor struct{ Monitor }

func (*canceledMonitor) Observe(context.Context, *umpirespb.RunEvent) (Decision, error) {
	return Continue, nil
}

func (*canceledMonitor) Close(ctx context.Context, _ *umpirespb.Run) (*umpirespb.Verdict, error) {
	<-ctx.Done()
	return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_INCONCLUSIVE}, ctx.Err()
}

func TestRunConformingMonitorCancellationIsInconclusive(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Entrypoints[0].Nodes = nil
	c.Program.Limits.MaxTotalDurationMilliseconds = 10
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: &runtimeSession{}}, &canceledMonitor{}, "run", c.CaseId)
	require.NoError(t, err)
	require.Equal(t, umpirespb.RUN_DISPOSITION_INCOMPLETE, run.GetDisposition())
	require.Equal(t, umpirespb.VERDICT_KIND_INCONCLUSIVE, verdict.GetKind())
	require.Contains(t, diagnosticCodes(run), "close_failed")
}

func TestRunTerminalPrecedence(t *testing.T) {
	for _, test := range []struct {
		name            string
		ordinaryErr     error
		stop            bool
		closeKind       umpirespb.VerdictKind
		cleanupErr      error
		hostCloseErr    error
		wantDisposition umpirespb.RunDisposition
		wantCleanup     umpirespb.RunCleanupStatus
		wantVerdict     umpirespb.VerdictKind
	}{
		{name: "complete", wantDisposition: umpirespb.RUN_DISPOSITION_COMPLETED, wantCleanup: umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED, wantVerdict: umpirespb.VERDICT_KIND_SATISFIED},
		{name: "early liveness closure", closeKind: umpirespb.VERDICT_KIND_INCONCLUSIVE, wantDisposition: umpirespb.RUN_DISPOSITION_COMPLETED, wantCleanup: umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED, wantVerdict: umpirespb.VERDICT_KIND_INCONCLUSIVE},
		{name: "execution failure", ordinaryErr: errors.New("effect failed"), wantDisposition: umpirespb.RUN_DISPOSITION_INCOMPLETE, wantCleanup: umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED, wantVerdict: umpirespb.VERDICT_KIND_INCONCLUSIVE},
		{name: "violation dominates cleanup and close", stop: true, cleanupErr: errors.New("cleanup failed"), hostCloseErr: errors.New("close failed"), wantDisposition: umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, wantCleanup: umpirespb.RUN_CLEANUP_STATUS_FAILED, wantVerdict: umpirespb.VERDICT_KIND_VIOLATED},
		{name: "cleanup and close do not replace success", cleanupErr: errors.New("cleanup failed"), hostCloseErr: errors.New("close failed"), wantDisposition: umpirespb.RUN_DISPOSITION_COMPLETED, wantCleanup: umpirespb.RUN_CLEANUP_STATUS_FAILED, wantVerdict: umpirespb.VERDICT_KIND_SATISFIED},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			c.Program.Cleanup.Nodes = []*umpirespb.InstructionNode{rpcNode("cleanup")}
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			ordinary := newRuntimeEffect(effectResponse(prepared, "ordinary"), true)
			ordinary.waitErr = test.ordinaryErr
			cleanup := newRuntimeEffect(effectResponse(prepared, "cleanup"), true)
			cleanup.waitErr = test.cleanupErr
			session := &runtimeSession{effects: map[string]*runtimeEffect{"call": ordinary, "cleanup": cleanup}, closeErr: test.hostCloseErr}
			monitor := &runtimeMonitor{closeKind: test.closeKind}
			if test.stop {
				monitor.stopSource = "scheduler.g0.n0.a1.completed"
			}
			run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
			require.NoError(t, err)
			require.Equal(t, test.wantDisposition, run.GetDisposition())
			require.Equal(t, test.wantCleanup, run.GetCleanup().GetStatus())
			require.Equal(t, test.wantVerdict, verdict.GetKind())
		})
	}
}

func TestRunCleanupDeadlineDoesNotReplaceOrdinarySuccess(t *testing.T) {
	c, catalog, policy := fixture(t)
	c.Program.Limits.MaxCleanupDurationMilliseconds = 10
	cleanupNode := rpcNode("cleanup")
	cleanupNode.Bounds.TimeoutMilliseconds = 10
	c.Program.Cleanup.Nodes = []*umpirespb.InstructionNode{cleanupNode}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	session := &runtimeSession{effects: map[string]*runtimeEffect{
		"call":    newRuntimeEffect(effectResponse(prepared, "ordinary"), true),
		"cleanup": newRuntimeEffect(effectResponse(prepared, "cleanup"), false),
	}}

	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, &runtimeMonitor{}, "run", c.CaseId)

	require.NoError(t, err)
	require.Equal(t, umpirespb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, umpirespb.RUN_CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
	require.Equal(t, umpirespb.VERDICT_KIND_SATISFIED, verdict.GetKind())
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
	require.Equal(t, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition())
	require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, verdict.GetKind())
	require.Subset(t, diagnosticCodes(run), []string{"effect_cancel_context_violated", "quarantine_failed"})
}

func diagnosticCodes(run *umpirespb.Run) []string {
	codes := make([]string, 0, len(run.GetDiagnostics()))
	for _, diagnostic := range run.GetDiagnostics() {
		codes = append(codes, diagnostic.GetCode())
	}
	return codes
}

func eventSources(run *umpirespb.Run) []string {
	sources := make([]string, 0, len(run.GetEvents()))
	for _, event := range run.GetEvents() {
		sources = append(sources, event.GetSourceId())
	}
	return sources
}
