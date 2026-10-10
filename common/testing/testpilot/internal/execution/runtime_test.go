package execution

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// runtimeDriver stays local because execution opens a *PreparedProgram, not the facade's Program.
type runtimeDriver struct {
	identity contract.DriverIdentity
	session  *runtimeSession
}

func (d *runtimeDriver) Identity(context.Context) (contract.DriverIdentity, error) {
	return d.identity, nil
}
func (d *runtimeDriver) Validate(context.Context, *PreparedProgram) error { return nil }
func (d *runtimeDriver) Open(context.Context, string, *PreparedProgram) (contract.Session, error) {
	return d.session, nil
}

// runtimeSession stays local: it serves each instruction its own effect and completes a quarantine
// only when that effect completes, which a scripted Session would rebuild per test.
type runtimeSession struct {
	contract.Session
	mu            sync.Mutex
	effects       map[string]*testsupport.Effect
	invokeErr     map[string]error
	invocations   []string
	quarantined   []contract.EffectHandle
	diagnostics   []*testpilotspb.RunDiagnostic
	quarantine    int
	quarantineErr error
	closed        int
	closeErr      error
	closeTimeout  bool
}

func (s *runtimeSession) InvokeRPC(_ context.Context, c contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message) (contract.EffectHandle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invocations = append(s.invocations, c.InstructionID)
	return s.effects[c.InstructionID], s.invokeErr[c.InstructionID]
}
func (s *runtimeSession) Quarantine(_ context.Context, handle contract.EffectHandle) error {
	s.mu.Lock()
	if s.quarantineErr != nil {
		defer s.mu.Unlock()
		return s.quarantineErr
	}
	s.quarantined = append(s.quarantined, handle)
	s.quarantine++
	s.mu.Unlock()
	effect := handle.(*testsupport.Effect)
	go func() {
		<-effect.Done()
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

func newRuntimeEffect(result contract.EffectResult, complete bool) *testsupport.Effect {
	effect := &testsupport.Effect{Result: result, CancelCompletes: true}
	if complete {
		effect.Complete()
	}
	return effect
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
	policy.Limits.MaxDuration = duration.FromMilliseconds(1000)
	policy.Limits.CleanupDuration = duration.FromMilliseconds(1000)
	late := rpcNode("late")
	late.After = runsAfter("controller")
	quarantine := rpcNode("quarantine")
	quarantine.After = runsAfter("controller")
	after := rpcNode("after")
	after.After = runsAfter("controller", "call")
	after.Guard = alwaysRuns()
	c.Program.Entrypoints[0].Instructions = append(c.Program.Entrypoints[0].Instructions, late, quarantine, after)
	cleanupNode := rpcNode("cleanup")
	cleanupNode.Limits.Timeout = duration.FromMilliseconds(10)
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{cleanupNode}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	complete := newRuntimeEffect(effectResponse(prepared, "complete"), true)
	lateEffect := newRuntimeEffect(effectResponse(prepared, "late"), false)
	quarantined := newRuntimeEffect(effectResponse(prepared, "quarantined"), false)
	quarantined.CancelCompletes = false
	quarantined.DrainErr = context.DeadlineExceeded
	cleanup := newRuntimeEffect(effectResponse(prepared, "cleanup"), true)
	session := &runtimeSession{effects: map[string]*testsupport.Effect{
		"call": complete, "late": lateEffect, "quarantine": quarantined, "cleanup": cleanup,
	}}
	driver := &runtimeDriver{identity: contract.DriverIdentity{Profile: policy.Identity, Catalog: policy.CatalogIdentity}, session: session}

	run, verdict, err := Run(t.Context(), prepared, driver, &runtimeMonitor{stopSource: "scheduler.g0.n0.a1.completed"}, "run", c.CaseId)

	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition())
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, verdict.GetStatus())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.NotContains(t, session.invocations, "after")
	require.Contains(t, session.invocations, "cleanup")
	require.Contains(t, eventSources(run), "scheduler.g0.n1.a1.completed")
	require.Positive(t, lateEffect.Cancels())
	require.Positive(t, quarantined.Cancels())
	require.Contains(t, session.quarantined, quarantined)
	require.Equal(t, 1, session.closed)
	serialized, err := proto.Marshal(run)
	require.NoError(t, err)
	serializedVerdict, err := proto.Marshal(verdict)
	require.NoError(t, err)
	quarantined.Complete()
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
	policy.Limits.MaxDuration = duration.FromMilliseconds(10)
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
	require.Equal(t, testpilotspb.RUN_DISPOSITION_INCOMPLETE, result.run.GetDisposition())
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
	policy.Limits.MaxDuration = duration.FromMilliseconds(10)
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: &runtimeSession{}}, &canceledMonitor{}, "run", c.CaseId)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_INCOMPLETE, run.GetDisposition())
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
		wantDisposition  testpilotspb.RunDisposition
		wantCleanup      testpilotspb.CleanupStatus
		wantVerdict      testpilotspb.VerdictStatus
	}{
		{name: "complete", wantDisposition: testpilotspb.RUN_DISPOSITION_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
		{name: "early liveness closure", closeKind: testpilotspb.VERDICT_STATUS_INCONCLUSIVE, wantDisposition: testpilotspb.RUN_DISPOSITION_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED, wantVerdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{name: "execution failure", ordinaryErr: errors.New("effect failed"), wantDisposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE, wantCleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED, wantVerdict: testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{name: "close error preserves success", hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_DISPOSITION_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
		{name: "close timeout preserves success", hostCloseTimeout: true, wantDisposition: testpilotspb.RUN_DISPOSITION_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
		{name: "close error preserves violation", stop: true, hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_VIOLATED},
		{name: "close timeout preserves violation", stop: true, hostCloseTimeout: true, wantDisposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_VIOLATED},
		{name: "violation dominates cleanup and close", stop: true, cleanupErr: errors.New("cleanup failed"), hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_VIOLATED},
		{name: "cleanup and close do not replace success", cleanupErr: errors.New("cleanup failed"), hostCloseErr: errors.New("close failed"), wantDisposition: testpilotspb.RUN_DISPOSITION_COMPLETED, wantCleanup: testpilotspb.CLEANUP_STATUS_FAILED, wantVerdict: testpilotspb.VERDICT_STATUS_SATISFIED},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			policy.Limits.CleanupDuration = duration.FromMilliseconds(100)
			c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{rpcNode("cleanup")}
			c.Program.Cleanup.Instructions[0].Limits.Timeout = duration.FromMilliseconds(100)
			prepared, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			ordinary := newRuntimeEffect(effectResponse(prepared, "ordinary"), true)
			ordinary.WaitErr = test.ordinaryErr
			cleanup := newRuntimeEffect(effectResponse(prepared, "cleanup"), true)
			cleanup.WaitErr = test.cleanupErr
			session := &runtimeSession{effects: map[string]*testsupport.Effect{"call": ordinary, "cleanup": cleanup}, closeErr: test.hostCloseErr, closeTimeout: test.hostCloseTimeout}
			monitor := &runtimeMonitor{closeKind: test.closeKind}
			if test.stop {
				monitor.stopSource = "scheduler.g0.n0.a1.completed"
			}
			run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
			require.NoError(t, err)
			require.Equal(t, test.wantDisposition, run.GetDisposition())
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
	policy.Limits.CleanupDuration = duration.FromMilliseconds(10)
	cleanupNode := rpcNode("cleanup")
	cleanupNode.Limits.Timeout = duration.FromMilliseconds(10)
	c.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{cleanupNode}
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	session := &runtimeSession{effects: map[string]*testsupport.Effect{
		"call":    newRuntimeEffect(effectResponse(prepared, "ordinary"), true),
		"cleanup": newRuntimeEffect(effectResponse(prepared, "cleanup"), false),
	}}

	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, &runtimeMonitor{}, "run", c.CaseId)

	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, testpilotspb.CLEANUP_STATUS_FAILED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
	require.Contains(t, diagnosticCodes(run), "cleanup_failed")
}

func TestRunBoundsHostContextViolationAndQuarantineCapacityFailure(t *testing.T) {
	c, catalog, policy := fixture(t)
	policy.Limits.CleanupDuration = duration.FromMilliseconds(10)
	prepared, err := Prepare(c, catalog, policy)
	require.NoError(t, err)
	effect := newRuntimeEffect(effectResponse(prepared, "complete"), true)
	effect.OnCancel = func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}
	effect.DrainErr = context.DeadlineExceeded
	session := &runtimeSession{
		effects:       map[string]*testsupport.Effect{"call": effect},
		quarantineErr: errors.New("quarantine capacity exhausted"),
	}
	monitor := &runtimeMonitor{stopSource: "scheduler.g0.n0.a1.completed"}
	run, verdict, err := Run(t.Context(), prepared, &runtimeDriver{session: session}, monitor, "run", c.CaseId)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition())
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
