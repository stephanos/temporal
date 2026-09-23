package controller

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
)

// reconcileFixture is one job's recovery record and the fake server its lease lives on.
type reconcileFixture struct {
	*invokeFixture
	progress bytes.Buffer
	now      time.Time
}

func newReconcileFixture(t *testing.T) *reconcileFixture {
	t.Helper()
	return &reconcileFixture{invokeFixture: newInvokeFixture(t), now: time.Unix(1000, 0).Add(time.Hour)}
}

// record writes the job's recovery record: the lease it took or found and its iterations.
func (f *reconcileFixture) record(t *testing.T, lease *recovery.Lease, iterations ...recovery.Iteration) {
	t.Helper()
	store, err := recovery.Create(f.recovery, "1234567-1")
	require.NoError(t, err)
	require.NoError(t, store.Update(func(record *recovery.Record) {
		record.Phase = recovery.PhaseRunning
		record.Lease = lease
		record.Iterations = iterations
	}))
}

// took starts a lease run as the job's own and fences the given Runs on it, each left open or
// closed.
func (f *reconcileFixture) took(t *testing.T, open map[string]bool, runs ...string) Fence {
	t.Helper()
	fence, err := takeLease(t.Context(), target(f.server), f.policy.Lease, time.Hour, "request")
	require.NoError(t, err)
	for _, id := range runs {
		require.NoError(t, signalRunOpened(t.Context(), target(f.server), fence, id))
		f.server.open(id, time.Unix(3000, 0))
		if !open[id] {
			f.server.finish(id)
		}
	}
	f.server.calls = nil
	return fence
}

func (f *reconcileFixture) reconcile(t *testing.T, seams Seams) (Report, int) {
	t.Helper()
	return Reconcile(t.Context(), Reconciliation{
		Seams: seams, Recovery: f.recovery, Progress: &f.progress, Now: f.now,
		Lookup: func(key string) (string, bool) { value, ok := f.env[key]; return value, ok },
		service: func(*authority.Authority, string, io.Writer) (*lazyService, error) {
			return &lazyService{direct: f.server}, nil
		},
		wait: noWait,
	})
}

func (f *reconcileFixture) leaseState(t *testing.T) Observed {
	t.Helper()
	observed, err := leaseState(t.Context(), target(f.server), f.policy.Lease.WorkflowID)
	require.NoError(t, err)
	return observed
}

// No record, or a record with no lease, is nothing to reconcile, exit 0, and needs no credential.
func TestReconcileWithNothingToReconcile(t *testing.T) {
	noAuthority := func(f *reconcileFixture) Seams {
		seams := f.seams()
		seams.Authority = func(authority.Lookup) (*authority.Authority, error) { return nil, authority.ErrNoCredential }
		return seams
	}
	f := newReconcileFixture(t)
	report, code := f.reconcile(t, noAuthority(f))
	require.Equal(t, ExitAccepted, code)
	require.Equal(t, StatusNothingToReconcile, report.Status)
	require.Contains(t, report.Detail, "no recovery record")

	f = newReconcileFixture(t)
	f.record(t, nil)
	report, code = f.reconcile(t, noAuthority(f))
	require.Equal(t, ExitAccepted, code)
	require.Equal(t, StatusNothingToReconcile, report.Status)
	require.Contains(t, report.Detail, "may be held")
	require.Empty(t, f.server.calls)
}

// The job's own lease is reconciled at once: exactly its fenced workflows are verified or
// terminated, the lease run is terminated as reconciled, and the iterations its record shows
// unpublished are reported lost; no Verdict or receipt appears anywhere.
func TestReconcileClosesTheJobsOwnLease(t *testing.T) {
	f := newReconcileFixture(t)
	fence := f.took(t, map[string]bool{runID(2): true}, runID(1), runID(2))
	f.server.open("customer-workflow", time.Unix(1, 0))
	f.record(t, &recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldTook},
		recovery.Iteration{RunID: runID(1), Published: true}, recovery.Iteration{RunID: runID(2)})

	report, code := f.reconcile(t, f.seams())
	require.Equal(t, ExitAccepted, code, "%+v", report)
	require.Equal(t, StatusReconciled, report.Status)
	require.Equal(t, []string{runID(1), runID(2)}, report.Fenced)
	require.Equal(t, []string{runID(1), runID(2)}, report.Closed)
	require.Empty(t, report.Unverified)
	require.Equal(t, []string{runID(2)}, report.Lost, "the unpublished iteration is lost, with no receipt")
	require.Empty(t, report.PublicationUnknown)
	require.Contains(t, f.server.calls, "terminate "+runID(2))
	require.NotContains(t, f.server.calls, "terminate "+runID(1), "a closed fenced workflow is only verified")
	require.NotContains(t, f.server.calls, "terminate customer-workflow")
	require.Equal(t, LeaseReleased, f.leaseState(t).State)
	entries, err := os.ReadDir(f.output)
	require.NoError(t, err)
	require.Empty(t, entries, "reconcile writes no receipt or provenance")
}

// A lease its job found open is left alone while it is younger than an invocation can be, and
// reconciled once it is older; its fenced Runs' publication is the earlier invocation's.
func TestReconcileGuardsAFoundLeaseByItsAge(t *testing.T) {
	f := newReconcileFixture(t)
	fence := f.took(t, map[string]bool{runID(1): true}, runID(1))
	f.record(t, &recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldFound})
	f.now = time.Unix(1000, 0).Add(f.policy.Limits.Invocation())

	report, code := f.reconcile(t, f.seams())
	require.Equal(t, ExitUncertain, code)
	require.Equal(t, StatusLeaseInUse, report.Status)
	require.Empty(t, f.server.calls[1:], "only the lease run was read")
	require.Equal(t, LeaseOpen, f.leaseState(t).State)

	f.now = time.Unix(1000, 0).Add(f.policy.Limits.Invocation() + f.policy.Limits.CleanupReserve() + time.Second)
	report, code = f.reconcile(t, f.seams())
	require.Equal(t, ExitAccepted, code, "%+v", report)
	require.Equal(t, StatusReconciled, report.Status)
	require.Equal(t, []string{runID(1)}, report.PublicationUnknown)
	require.Empty(t, report.Lost)
	require.Equal(t, LeaseReleased, f.leaseState(t).State)
}

// A found lease that timed out or was terminated by hand is closed by a fresh lease run
// terminated as reconciled, so the next dispatch reads the scope clean; a later lease run
// someone else holds is never touched.
func TestReconcileRecordsAClosedLeaseReconciled(t *testing.T) {
	for name, close := range map[string]func(f *reconcileFixture, fence Fence){
		"timed out": func(f *reconcileFixture, _ Fence) { f.server.timeOut(f.policy.Lease.WorkflowID) },
		"terminated by hand": func(f *reconcileFixture, fence Fence) {
			require.NoError(t, terminate(t.Context(), target(f.server), &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID}, "operator"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newReconcileFixture(t)
			fence := f.took(t, nil, runID(1))
			close(f, fence)
			f.record(t, &recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldFound})
			require.Equal(t, LeaseClosedOtherwise, f.leaseState(t).State)
			report, code := f.reconcile(t, f.seams())
			require.Equal(t, ExitAccepted, code, "%+v", report)
			require.Equal(t, StatusReconciled, report.Status)
			observed := f.leaseState(t)
			require.Equal(t, LeaseReleased, observed.State)
			require.NotEqual(t, fence.RunID, observed.RunID, "a fresh lease run records the scope reconciled")
		})
	}

	t.Run("a later lease run is another invocation's", func(t *testing.T) {
		f := newReconcileFixture(t)
		fence := f.took(t, nil, runID(1))
		f.server.timeOut(f.policy.Lease.WorkflowID)
		later := f.server.open(f.policy.Lease.WorkflowID, time.Unix(5000, 0))
		f.record(t, &recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldFound})
		report, code := f.reconcile(t, f.seams())
		require.Equal(t, ExitAccepted, code, "%+v", report)
		observed := f.leaseState(t)
		require.Equal(t, later, observed.RunID)
		require.Equal(t, LeaseOpen, observed.State, "the later run is left alone")
	})

	t.Run("a lease the server does not find", func(t *testing.T) {
		f := newReconcileFixture(t)
		f.record(t, &recovery.Lease{WorkflowID: f.policy.Lease.WorkflowID, RunID: "00000000-0000-4000-9000-000000000099", Held: recovery.HeldTook})
		report, code := f.reconcile(t, f.seams())
		require.Equal(t, ExitAccepted, code)
		require.Equal(t, StatusReconciled, report.Status)
		require.Contains(t, report.Detail, "clean")
	})
}

// A fenced workflow that cannot be verified closed, or a fence that cannot be read, leaves the
// lease held and the scope uncertain, exit 2.
func TestReconcileLeavesAnUncertainScopeHeld(t *testing.T) {
	for name, stick := range map[string]func(*reconcileFixture){
		"a workflow that stays open": func(f *reconcileFixture) { f.server.terminal[runID(1)] = true },
		"a termination that fails": func(f *reconcileFixture) {
			f.server.fail["terminate "+runID(1)] = serviceerror.NewUnavailable("down")
		},
		"a foreign signal on the lease": func(f *reconcileFixture) {
			fence := Fence{WorkflowID: f.policy.Lease.WorkflowID, RunID: f.server.latest(f.policy.Lease.WorkflowID).runID}
			require.NoError(t, signalRunOpened(t.Context(), target(f.server), fence, "customer-workflow"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newReconcileFixture(t)
			fence := f.took(t, map[string]bool{runID(1): true}, runID(1))
			stick(f)
			f.record(t, &recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldTook})
			report, code := f.reconcile(t, f.seams())
			require.Equal(t, ExitUncertain, code, "%+v", report)
			require.Equal(t, StatusReconcileUncertain, report.Status)
			require.Equal(t, LeaseOpen, f.leaseState(t).State, "the lease stays held")
		})
	}
}

// A tampered record, another scope or no credential is exit 3, and nothing is touched.
func TestReconcileRefusesAnotherScope(t *testing.T) {
	t.Run("a tampered record", func(t *testing.T) {
		f := newReconcileFixture(t)
		require.NoError(t, os.WriteFile(f.recovery, []byte("{}\n"), 0o600))
		report, code := f.reconcile(t, f.seams())
		require.Equal(t, ExitFailed, code)
		require.Equal(t, StatusRecoveryUnreadable, report.Status)
	})
	t.Run("a record readable by others", func(t *testing.T) {
		f := newReconcileFixture(t)
		f.record(t, &recovery.Lease{WorkflowID: f.policy.Lease.WorkflowID, RunID: "run", Held: recovery.HeldTook})
		require.NoError(t, os.Chmod(f.recovery, 0o644))
		report, code := f.reconcile(t, f.seams())
		require.Equal(t, ExitFailed, code)
		require.Equal(t, StatusRecoveryUnreadable, report.Status)
	})
	for name, test := range map[string]struct {
		edit   func(*reconcileFixture, *Seams)
		status string
	}{
		"an unconfigured policy": {func(f *reconcileFixture, _ *Seams) { f.policy.Coordinates.GRPC = policy.Unconfigured }, preflight.StatusPolicyUnconfigured},
		"another target":         {func(f *reconcileFixture, _ *Seams) { f.policy.Coordinates.Namespace = policy.Digest("customer") }, preflight.StatusCoordinateMismatch},
		"another lease":          {func(f *reconcileFixture, _ *Seams) { f.policy.Lease.WorkflowID = "another-lease" }, StatusRecoveryUnreadable},
		"no credential": {func(_ *reconcileFixture, s *Seams) {
			s.Authority = func(authority.Lookup) (*authority.Authority, error) { return nil, authority.ErrNoCredential }
		}, StatusAuthorityUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			f := newReconcileFixture(t)
			fence := f.took(t, map[string]bool{runID(1): true}, runID(1))
			f.record(t, &recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldTook})
			seams := f.seams()
			test.edit(f, &seams)
			report, code := f.reconcile(t, seams)
			require.Equal(t, ExitFailed, code)
			require.Equal(t, test.status, report.Status)
			require.Empty(t, f.server.calls)
			observed, err := leaseState(t.Context(), target(f.server), fence.WorkflowID)
			require.NoError(t, err)
			require.Equal(t, LeaseOpen, observed.State)
		})
	}
}
