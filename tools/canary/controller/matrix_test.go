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
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/umpire/evaluation"
	"google.golang.org/protobuf/proto"
)

// The adversarial authority and containment matrix, in process. Each mutation fails at the
// boundary that owns it and touches nothing unrelated. The cases this package already proves at
// their own boundary are not repeated here: target drift (TestInvokeRefusesBeforeAnything), a
// lease collision (TestTakeLeaseFailsOnAnyOpenRun, and the race in TestRunRefusesAnUnreconciledLease),
// a stale fence (TestAStaleFenceStopsTheInvocation), a second open of one Driver
// (TestFencedDriverFencesBeforeItOpens), another Case (TestInvokeExitsThreeForAnUnconstructibleIteration),
// a timed-out or hand-terminated lease (TestRunRefusesAnUnreconciledLease,
// TestReconcileRecordsAClosedLeaseReconciled), a tampered record (TestReconcileRefusesAnotherScope),
// cleanup uncertainty (TestCleanupLeavesTheLeaseHeldWhenAWorkflowIsNotVerifiedClosed), an
// iteration past the limit (TestRunStopsAtThePolicysIterations), and a publication conflict
// (TestInvokeExitsThreeOnAPublicationConflict). The live cases -- routing drift, a crash at each
// phase -- are the harness's, in tests/testpilot_canary_test.go.

// A credential and every raw coordinate planted in a failure's text reach no output: not the
// summary, the progress, a published receipt or provenance, nor the recovery record.
func TestAPlantedCredentialReachesNoOutput(t *testing.T) {
	const apiKey = "canary-api-key-9b2e7d41c0"
	credentialed := map[string]string{
		authority.VariableGRPC: invokeCoordinates.GRPC, authority.VariableNamespace: invokeCoordinates.Namespace,
		authority.VariableTaskQueue: invokeCoordinates.TaskQueue, authority.VariableHandlerQueue: invokeCoordinates.HandlerQueue,
		authority.VariableEndpoint: invokeCoordinates.NexusEndpoint, authority.VariableAPIKey: apiKey,
	}
	host, _, _ := strings.Cut(invokeCoordinates.GRPC, ":")
	secrets := []string{apiKey, invokeCoordinates.GRPC, host, invokeCoordinates.Namespace, invokeCoordinates.TaskQueue,
		invokeCoordinates.HandlerQueue, invokeCoordinates.NexusEndpoint}
	planted := "Bearer " + apiKey + " to " + invokeCoordinates.GRPC + " for " + invokeCoordinates.Namespace + " via " +
		invokeCoordinates.TaskQueue + ", " + invokeCoordinates.HandlerQueue + " and " + invokeCoordinates.NexusEndpoint

	for name, test := range map[string]struct {
		plant     func(*invokeFixture)
		published int
	}{
		"in a Run's error": {plant: func(f *invokeFixture) { f.runErr = errors.New("the Run failed: " + planted) }},
		"in a Driver's release error": {plant: func(f *invokeFixture) {
			f.release = func(context.Context) error { return errors.New("the worker did not stop: " + planted) }
		}, published: 2},
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
			for _, name := range f.published(t) {
				document, err := os.ReadFile(filepath.Join(f.output, name))
				require.NoError(t, err)
				written[name] = string(document)
			}
			for where, text := range written {
				for _, secret := range secrets {
					require.NotContains(t, text, secret, "a planted value reached %s", where)
				}
			}
		})
	}
}

// A Run that crosses its fence fails at the boundary that owns it. A Run naming another Run than
// the one its Driver fenced is refused by publication, which publishes nothing; a workflow start
// outside the fence is refused by the fenced Session before it reaches the Driver, so the Run is
// unconstructible. Cleanup acts only on the fenced Runs either way.
func TestARunCrossingItsFenceFailsAtItsBoundary(t *testing.T) {
	t.Run("another Run than the fenced one", func(t *testing.T) {
		f := newInvokeFixture(t)
		f.run = func(index int, run *testpilotspb.Run) *testpilotspb.Run {
			run.RunId = runID(100 + index)
			return run
		}
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitFailed, code, "%+v", summary)
		require.Equal(t, StatusPublicationFailed, summary.Status)
		require.Contains(t, summary.Detail, "different Runs")
		require.Empty(t, f.published(t), "a crossed receipt is never published")
		require.Equal(t, []string{runID(1), runID(2)}, summary.Cleanup.Fenced)
	})

	t.Run("a start outside the fence", func(t *testing.T) {
		f := newInvokeFixture(t)
		var delegated []string
		f.session = func(ctx context.Context, session testpilot.Session) error {
			_, err := session.InvokeRPC(ctx, testpilot.Coordinate{}, casebinding.EndpointRole, workflowServiceMethod("StartWorkflowExecution"),
				&workflowservice.StartWorkflowExecutionRequest{WorkflowId: "customer-workflow"})
			delegated = session.(*fencedSession).Session.(*recordingSession).invoked
			return err
		}
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitFailed, code, "%+v", summary)
		require.Equal(t, StatusUnconstructible, summary.Status)
		require.Contains(t, summary.Detail, ErrOutsideFence.Error())
		require.Empty(t, delegated, "the start never reached the Driver")
		require.Empty(t, f.published(t))
		require.Equal(t, []string{runID(1)}, summary.Cleanup.Fenced)
		require.Equal(t, assessment.CleanupReleased, summary.Cleanup.Outcome)
	})
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
	require.Equal(t, []string{runID(1), runID(2)}, summary.Cleanup.Fenced)
	require.Len(t, f.published(t), 4)
}

// A tenfold request is capped by three sets of limits, each pinned here so raising one is a
// reviewed change: the policy's (iterations, time, the cleanup reserve, the lease's run timeout and
// progress), the Temporal Profile's DefaultCeilings the canary's Driver Profile carries (a Run's
// RPCs, worker activations, events and duration, and so the longest an iteration can take), and
// fn-26's admission caps, which every receipt records and which refuse a Run past them.
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
		require.Len(t, result.Iterations, canary.Limits.Iterations, "every Run accepted, and still no more than the policy's")
		require.Equal(t, canary.Limits.Iterations, s.runs)
	})

	t.Run("the Temporal Profile", func(t *testing.T) {
		canary := testPolicy(t)
		bound, err := casebinding.Bind(canary, testpilotdriver.Environment{
			Namespace: testNamespace, TaskQueue: "queue", HandlerTaskQueue: "handler-queue", NexusEndpoint: "endpoint",
		})
		require.NoError(t, err)
		program, contract, correlated := testpilotdriver.DefaultCeilings()
		require.True(t, proto.Equal(program, bound.Profile.ProgramLimits))
		require.True(t, proto.Equal(contract, bound.Profile.ContractLimits))
		require.True(t, proto.Equal(correlated, bound.Profile.CorrelatedLimits))
		limits := bound.Profile.ProgramLimits
		require.Equal(t, [6]int64{16, 8, 512, 32768, 30000, 20000}, [6]int64{
			int64(limits.GetMaxAttempts()), int64(limits.GetMaxActivations()), int64(limits.GetMaxRunEvents()),
			int64(limits.GetMaxRequestBytes()), limits.GetMaxTotalDurationMilliseconds(), limits.GetMaxCleanupDurationMilliseconds(),
		}, "attempts, activations, events, request bytes, total and cleanup duration")

		s := &script{server: newFakeServer()}
		run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		require.Equal(t, 150*time.Second, run.iterationBound())
		require.GreaterOrEqual(t, canary.Limits.Invocation(), time.Duration(canary.Limits.Iterations)*run.iterationBound(),
			"the policy's iterations fit the invocation limit")
	})

	t.Run("fn-26's caps", func(t *testing.T) {
		require.Equal(t, evaluation.Caps{CaseBytes: 4 << 20, RunBytes: 16 << 20, RunEvents: 65536, ReceiptBytes: 1 << 20}, evaluation.AdmissionCaps())

		f := newInvokeFixture(t)
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitAccepted, code, "%+v", summary)
		for _, iteration := range summary.Iterations {
			encoded, err := os.ReadFile(filepath.Join(f.output, iteration.Receipt+".json"))
			require.NoError(t, err)
			receipt, err := evaluation.DecodeReceipt(encoded)
			require.NoError(t, err)
			require.Equal(t, evaluation.AdmissionCaps(), receipt.Caps, "the receipt records the caps it was admitted under")
		}

		f = newInvokeFixture(t)
		f.run = func(_ int, run *testpilotspb.Run) *testpilotspb.Run {
			for len(run.Events) <= evaluation.MaxRunEvents {
				run.Events = append(run.Events, run.GetEvents()[0])
			}
			return run
		}
		summary, code = f.invoke(t, f.seams())
		require.Equal(t, ExitFailed, code)
		require.Equal(t, StatusUnconstructible, summary.Status)
		require.Contains(t, summary.Detail, evaluation.ReasonOversized)
		require.Empty(t, f.published(t), "a Run past the caps has no receipt")
	})
}

// A proved violation stays rejected whatever else the invocation meets, and a Run that proves
// nothing is incomplete, never accepted: the published receipt's decision is fn-26's alone, and
// the invocation's own failures change only the exit and the provenance.
func TestAProvedViolationStaysViolated(t *testing.T) {
	violated := func(s *evaluation.Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED }
	for name, test := range map[string]struct {
		edit     func(*evaluation.Subject)
		also     func(*invokeFixture)
		decision string
		status   string
		code     int
	}{
		"violated": {edit: violated, decision: evaluation.DecisionRejected, status: StatusRejected, code: ExitDecided},
		"violated, and the Run's cleanup failed": {edit: func(s *evaluation.Subject) {
			violated(s)
			s.Cleanup = testpilotspb.CLEANUP_STATUS_FAILED
		}, decision: evaluation.DecisionRejected, status: StatusRejected, code: ExitDecided},
		"violated, and the invocation's cleanup uncertain": {edit: violated, also: func(f *invokeFixture) {
			f.between = func(int) { f.server.fail["history "+f.policy.Lease.WorkflowID] = serviceerror.NewUnavailable("down") }
		}, decision: evaluation.DecisionRejected, status: StatusCleanupUncertain, code: ExitUncertain},
		"violated, and the Driver did not release": {edit: violated, also: func(f *invokeFixture) {
			f.release = func(context.Context) error { return errors.New("the worker did not stop") }
		}, decision: evaluation.DecisionRejected, status: StatusToolingFailure, code: ExitFailed},
		"inconclusive": {edit: func(s *evaluation.Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_INCONCLUSIVE },
			decision: evaluation.DecisionIncomplete, status: StatusIncomplete, code: ExitDecided},
	} {
		t.Run(name, func(t *testing.T) {
			f := newInvokeFixture(t)
			f.decide = decidedAs(t, f.policy, test.edit)
			if test.also != nil {
				test.also(f)
			}
			summary, code := f.invoke(t, f.seams())
			require.Equal(t, test.code, code, "%+v", summary)
			require.Equal(t, test.status, summary.Status)
			require.Len(t, summary.Iterations, 1)
			require.Equal(t, test.decision, summary.Iterations[0].Status)
			encoded, err := os.ReadFile(filepath.Join(f.output, summary.Iterations[0].Receipt+".json"))
			require.NoError(t, err)
			receipt, err := evaluation.DecodeReceipt(encoded)
			require.NoError(t, err)
			require.Equal(t, test.decision, receipt.Decision)
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
		stores := make([]*recovery.Store, invocations)
		for index := range invocations {
			canary := testPolicy(t)
			canary.Lease.WorkflowID += "-" + strconv.Itoa(index)
			s := &script{server: server, first: 10 * index}
			runs[index], stores[index] = s.invocation(t, canary, time.Now(), &bytes.Buffer{})
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
			own := []string{runID(10*index + 1), runID(10*index + 2)}
			require.Equal(t, own, runIDs(results[index].Iterations))
			require.Equal(t, own, results[index].Cleanup.Fenced)
			require.True(t, results[index].Cleanup.Released)
			require.Equal(t, []recovery.Iteration{{RunID: own[0]}, {RunID: own[1]}}, stores[index].Snapshot().Iterations)
		}
	})

	t.Run("serial invocations on one lease", func(t *testing.T) {
		canary := testPolicy(t)
		server := newFakeServer()
		var drivers []*stubDriver
		var results []*Result
		for index := range 2 {
			s := &script{server: server, first: 10 * index}
			run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
			run.openDriver = func() (testpilot.Driver, func(context.Context) error, error) {
				driver := &stubDriver{}
				drivers = append(drivers, driver)
				return driver, func(context.Context) error { return nil }, nil
			}
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.True(t, result.Cleanup.Released)
			own := []string{runID(10*index + 1), runID(10*index + 2)}
			require.Equal(t, own, result.Cleanup.Fenced)
			require.Equal(t, []recovery.Iteration{{RunID: own[0]}, {RunID: own[1]}}, store.Snapshot().Iterations)
			results = append(results, result)
		}
		require.NotEqual(t, results[0].Lease.RunID, results[1].Lease.RunID)
		for index, result := range results {
			fenced, err := fencedIDs(t.Context(), target(server), Fence{WorkflowID: result.Lease.WorkflowID, RunID: result.Lease.RunID})
			require.NoError(t, err)
			require.Equal(t, []string{runID(10*index + 1), runID(10*index + 2)}, fenced, "each lease run fences its own Runs only")
		}
		require.Len(t, drivers, 4, "one fresh Driver per iteration")
		for _, driver := range drivers {
			require.Equal(t, 1, driver.opens)
		}
	})
}

// A stale recovery record -- a job that crashed after cleanup released its lease, reconciled only
// after a later invocation took the lease -- acts on its own lease run alone: it names its
// unpublished iterations lost and never touches the later lease run or the Run it fences.
func TestAStaleRecoveryRecordNeverReachesALaterLease(t *testing.T) {
	f := newReconcileFixture(t)
	stale := f.took(t, nil, runID(1))
	require.NoError(t, terminate(t.Context(), target(f.server), &commonpb.WorkflowExecution{WorkflowId: stale.WorkflowID, RunId: stale.RunID}, ReasonReleased))
	later, err := takeLease(t.Context(), target(f.server), f.policy.Lease, time.Hour, "later")
	require.NoError(t, err)
	require.NoError(t, signalRunOpened(t.Context(), target(f.server), later, runID(2)))
	f.server.open(runID(2), time.Unix(3000, 0))
	f.record(t, &recovery.Lease{WorkflowID: stale.WorkflowID, RunID: stale.RunID, Held: recovery.HeldTook}, recovery.Iteration{RunID: runID(1)})
	f.server.calls = nil

	report, code := f.reconcile(t, f.seams())
	require.Equal(t, ExitAccepted, code, "%+v", report)
	require.Equal(t, StatusReconciled, report.Status)
	require.Equal(t, []string{runID(1)}, report.Fenced)
	require.Equal(t, []string{runID(1)}, report.Lost)
	require.Empty(t, report.Terminated)
	for _, call := range f.server.calls {
		require.False(t, mutates(call), "reconcile only reads here: %s", call)
	}
	observed := f.leaseState(t)
	require.Equal(t, Observed{State: LeaseOpen, RunID: later.RunID, Started: time.Unix(1000, 0).UTC()}, observed, "the later lease run is left alone")
	closed, err := workflowClosed(t.Context(), target(f.server), runID(2), time.Second, noWait)
	require.NoError(t, err)
	require.False(t, closed, "the later invocation's Run is left alone")
}

// mutates says whether a fake server call changes the target: a start, a signal or a termination.
func mutates(call string) bool {
	return strings.HasPrefix(call, "start ") || strings.HasPrefix(call, "signal ") || strings.HasPrefix(call, "terminate ")
}
