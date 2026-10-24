package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// The adversarial authority and containment matrix, in process. Each mutation fails at the
// boundary that owns it and touches nothing unrelated. The cases this package already proves at
// their own boundary are not repeated here: target drift (TestInvokeRefusesBeforeAnything), a
// lease collision (TestTakeLeaseFailsOnAnyOpenRun, and the race in TestRunRefusesAnUnreconciledLease),
// a stale fence (TestAStaleFenceStopsTheInvocation), a start outside the fence
// (TestTheFenceAdmitsOnlyItsOwnActivity), another Model (TestInvokeRefusesBeforeAnything),
// a timed-out or hand-terminated lease (TestRunRefusesAnUnreconciledLease,
// TestReconcileRecordsAClosedLeaseReconciled), a tampered record (TestReconcileRefusesAnotherScope),
// cleanup uncertainty (TestCleanupLeavesTheLeaseHeldWhenAnActivityIsNotVerifiedClosed), an
// iteration past the limit (TestRunStopsAtThePolicysIterations), and a publication conflict
// (TestInvokeExitsThreeOnAPublicationConflict). The live cases -- a crash at each phase -- are the
// harness's, in tests/testpilot_canary_test.go.

// A credential and every raw coordinate planted in a failure's text reach no output: not the
// summary, the progress, nor the recovery record. A published Run carries its World, whose
// namespace and task queue the Model needs to replay it; it never carries the credential or the
// gRPC target.
func TestAPlantedCredentialReachesNoOutput(t *testing.T) {
	const apiKey = "canary-api-key-9b2e7d41c0"
	credentialed := map[string]string{
		authority.VariableGRPC: invokeCoordinates.GRPC, authority.VariableNamespace: invokeCoordinates.Namespace,
		authority.VariableTaskQueue: invokeCoordinates.TaskQueue, authority.VariableAPIKey: apiKey,
	}
	host, _, _ := strings.Cut(invokeCoordinates.GRPC, ":")
	secrets := []string{apiKey, invokeCoordinates.GRPC, host, invokeCoordinates.Namespace, invokeCoordinates.TaskQueue}
	planted := "Bearer " + apiKey + " to " + invokeCoordinates.GRPC + " for " + invokeCoordinates.Namespace + " via " +
		invokeCoordinates.TaskQueue

	for name, test := range map[string]struct {
		plant     func(*invokeFixture)
		published int
	}{
		"in a Run's error": {plant: func(f *invokeFixture) { f.runErr = errors.New("the Run failed: " + planted) }},
		"in a Driver's release error": {plant: func(f *invokeFixture) {
			f.release = func(context.Context) error { return errors.New("the worker did not stop: " + planted) }
		}, published: 1},
	} {
		t.Run(name, func(t *testing.T) {
			f := newInvokeFixture(t)
			test.plant(f)
			seams := f.seams()
			seams.Authority = func(authority.Lookup) (*authority.Authority, error) {
				return authority.Load(func(key string) (string, bool) { value, ok := credentialed[key]; return value, ok })
			}
			summary, code := f.invoke(t, seams)
			require.Equal(t, ExitFailed, code, "%+v", summary)
			require.Contains(t, summary.Detail, authority.Redacted, "the failure is reported, redacted")
			require.Len(t, f.published(t), test.published)

			encoded, err := json.Marshal(summary)
			require.NoError(t, err)
			written := map[string]string{"the summary": string(encoded), "the progress": f.progress.String()}
			record, err := os.ReadFile(f.recovery)
			require.NoError(t, err)
			written["the recovery record"] = string(record)
			for where, text := range written {
				for _, secret := range secrets {
					require.NotContains(t, text, secret, "a planted value reached %s", where)
				}
			}
			for _, name := range f.published(t) {
				document, err := os.ReadFile(filepath.Join(f.output, name))
				require.NoError(t, err)
				for _, secret := range []string{apiKey, invokeCoordinates.GRPC, host} {
					require.NotContains(t, string(document), secret, "a planted value reached %s", name)
				}
			}
		})
	}
}

// A Run whose Driver sends a request outside its fence -- another activity, a workflow start, a
// read instruction wrapping one, a signal-with-start -- fails at the fence, before it reaches the
// Driver: the iteration fails, nothing is published, and cleanup acts only on the
// fenced activities.
func TestARunCrossingItsFenceFailsAtItsBoundary(t *testing.T) {
	for name, start := range map[string]func(*testing.T) *anypb.Any{
		"another activity": func(t *testing.T) *anypb.Any {
			return packed(t, &workflowservice.StartActivityExecutionRequest{ActivityId: "customer-activity"})
		},
		"a workflow": func(t *testing.T) *anypb.Any {
			return packed(t, &workflowservice.StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"})
		},
		"a read instruction that starts a workflow": func(t *testing.T) *anypb.Any {
			return packed(t, &temporalv1.ObserveActivity{
				Request:        packed(t, &workflowservice.StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"}),
				PollIntervalMs: 1, BackoffMs: 1, ReadTimeoutMs: 1,
			})
		},
		"a signal-with-start": func(t *testing.T) *anypb.Any {
			return packed(t, &workflowservice.SignalWithStartWorkflowExecutionRequest{WorkflowId: "customer-workflow"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newInvokeFixture(t)
			f.start = func(*anypb.Any) *anypb.Any { return start(t) }
			summary, code := f.invoke(t, f.seams())
			require.Equal(t, ExitFailed, code, "%+v", summary)
			require.Equal(t, StatusFailed, summary.Status)
			require.Contains(t, summary.Detail, ErrOutsideFence.Error())
			require.Empty(t, f.ids, "the start never reached the Driver")
			require.Empty(t, f.published(t))
			require.Len(t, summary.Cleanup.Fenced, 1)
			require.True(t, IsActivityID(summary.Cleanup.Fenced[0]))
			require.Equal(t, CleanupReleased, summary.Cleanup.Outcome)
			require.NotContains(t, f.server.calls, "terminate customer-activity")
		})
	}
}

// A second dispatch, and its job's reconcile, during a live Run of the first touch nothing of it:
// the dispatch refuses on the held lease with no Run and nothing published, the reconcile leaves a
// lease younger than an invocation alone, neither starts, signals or terminates anything, and the
// first invocation completes with only its own Runs fenced.
func TestASecondDispatchDuringALiveRunTouchesNothingOfIt(t *testing.T) {
	f := newInvokeFixture(t)
	second := newInvokeFixture(t)
	second.server = f.server
	second.env[preflight.VariableRunID] = "7654321"
	var (
		dispatched     Summary
		dispatchedCode int
		reconciled     Report
		reconciledCode int
		calls          []string
	)
	seams := f.seams()
	seams.Hook = func(phase string) {
		if phase != PhaseRunOpened || f.runs != 1 {
			return
		}
		f.server.mu.Lock()
		before := len(f.server.calls)
		f.server.mu.Unlock()
		dispatched, dispatchedCode = second.invoke(t, second.seams())
		job := &reconcileFixture{invokeFixture: second, now: time.Unix(1000, 0).Add(time.Minute)}
		reconciled, reconciledCode = job.reconcile(t, second.seams())
		f.server.mu.Lock()
		calls = append(calls, f.server.calls[before:]...)
		f.server.mu.Unlock()
	}
	summary, code := f.invoke(t, seams)

	require.Equal(t, ExitUncertain, dispatchedCode, "%+v", dispatched)
	require.Equal(t, StatusLeaseUnreconciled, dispatched.Status)
	require.Zero(t, second.runs)
	require.Empty(t, second.published(t))
	require.Equal(t, ExitUncertain, reconciledCode, "%+v", reconciled)
	require.Equal(t, StatusLeaseInUse, reconciled.Status)
	require.NotEmpty(t, calls)
	for _, call := range calls {
		require.False(t, mutates(call), "the second job only reads: %s", call)
	}

	require.Equal(t, ExitAccepted, code, "%+v", summary)
	require.Len(t, summary.Iterations, 2)
	require.Equal(t, f.ids, summary.Cleanup.Fenced)
	require.Len(t, f.published(t), 2)
}

// A tenfold request is capped by two sets of limits, each pinned here so raising one is a reviewed
// change: the policy's (iterations, time, the cleanup reserve, the lease's run timeout and
// progress) and the Runner's per-Run budgets the canary runs under (proposals, events, the event
// and run timeouts, and so the longest an iteration can take).
func TestATenfoldRequestIsCappedByEveryLimit(t *testing.T) {
	t.Run("the policy", func(t *testing.T) {
		canary := testPolicy(t)
		require.Equal(t, policy.Limits{
			Iterations: 2, InvocationSeconds: 600, CleanupReserveSeconds: 120, LeaseRunTimeoutSeconds: 86400, ProgressBytes: 65536,
		}, canary.Limits)
		tenfold := canary.Limits
		tenfold.Iterations *= 10
		require.ErrorContains(t, tenfold.Validate(), "iterations", "no policy asks for ten times the iterations")

		s := &script{server: newFakeServer()}
		run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		result, err := run.run(t.Context())
		require.NoError(t, err)
		require.Len(t, result.Iterations, canary.Limits.Iterations, "every Run explained, and still no more than the policy's")
		require.Equal(t, canary.Limits.Iterations, s.runs)
	})

	t.Run("the Runner's budgets", func(t *testing.T) {
		require.Equal(t, [4]int64{32, 1024, 30000, 300000}, [4]int64{
			runner.DefaultBudget, runner.DefaultEventBudget, runner.DefaultEventTimeout.Milliseconds(), runner.DefaultRunTimeout.Milliseconds(),
		}, "proposals, events, event and run timeout")
		require.Equal(t, 2*time.Minute, runTimeout, "the canary's own run budget")
		canary := testPolicy(t)
		s := &script{server: newFakeServer()}
		run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		require.Equal(t, 160*time.Second, run.iterationBound())
		require.GreaterOrEqual(t, canary.Limits.Invocation(), time.Duration(canary.Limits.Iterations)*run.iterationBound(),
			"the policy's iterations fit the invocation limit")
	})
}

// An unexplained Run stays unexplained whatever else the invocation meets: the published Run's
// verdict is the Model's alone, and the invocation's own failures change only the exit.
func TestAnUnexplainedRunStaysUnexplained(t *testing.T) {
	for name, test := range map[string]struct {
		also   func(*invokeFixture)
		status string
		code   int
	}{
		"unexplained": {status: StatusUnexplained, code: ExitDecided},
		"unexplained, and the invocation's cleanup uncertain": {also: func(f *invokeFixture) {
			f.between = func(int) { f.server.fail["history "+f.policy.Lease.WorkflowID] = serviceerror.NewUnavailable("down") }
		}, status: StatusCleanupUncertain, code: ExitUncertain},
		"unexplained, and the Driver did not release": {also: func(f *invokeFixture) {
			f.release = func(context.Context) error { return errors.New("the worker did not stop") }
		}, status: StatusToolingFailure, code: ExitFailed},
	} {
		t.Run(name, func(t *testing.T) {
			f := newInvokeFixture(t)
			f.statuses = []string{StatusUnexplained}
			if test.also != nil {
				test.also(f)
			}
			summary, code := f.invoke(t, f.seams())
			require.Equal(t, test.code, code, "%+v", summary)
			require.Equal(t, test.status, summary.Status)
			require.Len(t, summary.Iterations, 1)
			require.Equal(t, StatusUnexplained, summary.Iterations[0].Status)
			published, err := runner.Load(filepath.Join(f.output, summary.Iterations[0].Run))
			require.NoError(t, err)
			require.NotNil(t, published.GetVerdict().GetUnexplained())
		})
	}
}

// No state crosses Runs or leases. Invocations running at once on one server, each on its own
// lease, fence and close only their own Runs; invocations one after another on one lease each open fresh Drivers,
// each opened once, and each lease run's fence names only its own invocation's Runs. The package's
// tests run under the race detector, which this exercises across the concurrent invocations.
func TestNoStateCrossesRunsOrLeases(t *testing.T) {
	t.Run("concurrent invocations", func(t *testing.T) {
		const invocations = 4
		server := newFakeServer()
		runs := make([]*invocation, invocations)
		scripts := make([]*script, invocations)
		stores := make([]*recovery.Store, invocations)
		for index := range invocations {
			canary := testPolicy(t)
			canary.Lease.WorkflowID += "-" + strconv.Itoa(index)
			scripts[index] = &script{server: server}
			runs[index], stores[index] = scripts[index].invocation(t, canary, time.Now(), &bytes.Buffer{})
		}
		results := make([]*Result, invocations)
		errs := make([]error, invocations)
		var wg sync.WaitGroup
		for index := range invocations {
			wg.Go(func() { results[index], errs[index] = runs[index].run(t.Context()) })
		}
		wg.Wait()
		for index := range invocations {
			require.NoError(t, errs[index])
			own := scripts[index].ids
			require.Len(t, own, 2)
			require.Equal(t, own, activityIDs(results[index].Iterations))
			require.Equal(t, own, results[index].Cleanup.Fenced)
			require.True(t, results[index].Cleanup.Released)
			require.Equal(t, []recovery.Iteration{{ActivityID: own[0]}, {ActivityID: own[1]}}, stores[index].Snapshot().Iterations)
		}
	})

	t.Run("serial invocations on one lease", func(t *testing.T) {
		canary := testPolicy(t)
		server := newFakeServer()
		var drivers []*stubDriver
		var results []*Result
		var owns [][]string
		for range 2 {
			s := &script{server: server}
			run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
			run.openDriver = func() (runner.NewDriver, func(context.Context) error, error) {
				return func(context.Context) (runner.Driver, error) {
					driver := &stubDriver{}
					drivers = append(drivers, driver)
					return driver, nil
				}, func(context.Context) error { return nil }, nil
			}
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.True(t, result.Cleanup.Released)
			own := s.ids
			require.Equal(t, own, result.Cleanup.Fenced)
			require.Equal(t, []recovery.Iteration{{ActivityID: own[0]}, {ActivityID: own[1]}}, store.Snapshot().Iterations)
			results = append(results, result)
			owns = append(owns, own)
		}
		require.NotEqual(t, results[0].Lease.RunID, results[1].Lease.RunID)
		for index, result := range results {
			fenced, err := fencedIDs(t.Context(), target(server), Fence{WorkflowID: result.Lease.WorkflowID, RunID: result.Lease.RunID})
			require.NoError(t, err)
			require.Equal(t, owns[index], fenced, "each lease run fences its own activities only")
		}
		require.Len(t, drivers, 4, "one fresh Driver per iteration")
		for _, driver := range drivers {
			require.Len(t, driver.started, 1)
		}
	})
}

// A stale recovery record -- a job that crashed after cleanup released its lease, reconciled only
// after a later invocation took the lease -- acts on its own lease run alone: it names its
// unpublished iterations lost and never touches the later lease run or the activity it fences.
func TestAStaleRecoveryRecordNeverReachesALaterLease(t *testing.T) {
	f := newReconcileFixture(t)
	stale := f.took(t, nil, activityID(1))
	require.NoError(t, terminate(t.Context(), target(f.server), &commonpb.WorkflowExecution{WorkflowId: stale.WorkflowID, RunId: stale.RunID}, ReasonReleased))
	later, err := takeLease(t.Context(), target(f.server), f.policy.Lease, time.Hour, "later")
	require.NoError(t, err)
	require.NoError(t, signalRunOpened(t.Context(), target(f.server), later, activityID(2)))
	f.server.open(activityID(2), time.Unix(3000, 0))
	f.record(t, &recovery.Lease{WorkflowID: stale.WorkflowID, RunID: stale.RunID, Held: recovery.HeldTook}, recovery.Iteration{ActivityID: activityID(1)})
	f.server.calls = nil

	report, code := f.reconcile(t, f.seams())
	require.Equal(t, ExitAccepted, code, "%+v", report)
	require.Equal(t, StatusReconciled, report.Status)
	require.Equal(t, []string{activityID(1)}, report.Fenced)
	require.Equal(t, []string{activityID(1)}, report.Lost)
	require.Empty(t, report.Terminated)
	for _, call := range f.server.calls {
		require.False(t, mutates(call), "reconcile only reads here: %s", call)
	}
	observed := f.leaseState(t)
	require.Equal(t, Observed{State: LeaseOpen, RunID: later.RunID, Started: time.Unix(1000, 0).UTC()}, observed, "the later lease run is left alone")
	closed, err := activityClosed(t.Context(), target(f.server), activityID(2), time.Second, noWait)
	require.NoError(t, err)
	require.False(t, closed, "the later invocation's activity is left alone")
}

// mutates says whether a fake server call changes the target: a start, a signal or a termination.
func mutates(call string) bool {
	return strings.HasPrefix(call, "start ") || strings.HasPrefix(call, "signal ") || strings.HasPrefix(call, "terminate ")
}
