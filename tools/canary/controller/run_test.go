package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// script is one scripted invocation: each Run opens its fenced Driver, starts its workflow on the
// fake server and leaves it open or closes it, then decide answers from statuses.
type script struct {
	server   *fakeServer
	statuses []string
	// leaveOpen leaves each Run's workflow open, as a Run cut off by its deadline would.
	leaveOpen bool
	// runErr fails the Run after its Driver opens.
	runErr error
	// between runs after each Run, before the next.
	between  func(index int)
	runs     int
	released int
	decided  int
}

func (s *script) invocation(t *testing.T, canary *policy.Policy, started time.Time, progress *bytes.Buffer) (*invocation, *recovery.Store) {
	t.Helper()
	store, err := recovery.Create(filepath.Join(t.TempDir(), "recovery.json"), "1234-1")
	require.NoError(t, err)
	bound, err := casebinding.Bind(canary, testpilotdriver.Environment{
		Namespace: testNamespace, TaskQueue: "queue", HandlerTaskQueue: "handler-queue", NexusEndpoint: "endpoint",
	})
	require.NoError(t, err)
	run := newInvocation(Config{
		Policy: canary, Scope: &preflight.Scope{InvocationID: "1234-1", Prepared: bound.Prepared, Profile: bound.Profile},
		Namespace: testNamespace, Transport: authority.Transport{Target: "localhost:7233", Credentials: insecure.NewCredentials()},
		Redactor: authority.NewRedactor(testNamespace), Service: s.server, Identity: "umpire-canary",
		Dial:   func(client.Options) (client.Client, error) { return nil, errors.New("no dial in a unit test") },
		Decide: s.decide, Recovery: store, Progress: progress, Started: started, Wait: noWait,
	})
	run.openDriver = func() (testpilot.Driver, func(context.Context) error, error) {
		return &stubDriver{}, func(context.Context) error { s.released++; return nil }, nil
	}
	run.runCase = s.run
	return run, store
}

func (s *script) run(ctx context.Context, driver testpilot.Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	s.runs++
	runID := fmt.Sprintf("testpilot.run.%d", s.runs)
	if _, err := driver.Open(ctx, runID, testpilot.PreparedProgram{}); err != nil {
		return nil, nil, err
	}
	s.server.open(runID, time.Unix(3000, 0))
	if !s.leaveOpen {
		s.server.finish(runID)
	}
	if s.between != nil {
		s.between(s.runs)
	}
	if s.runErr != nil {
		return nil, nil, s.runErr
	}
	return &testpilotspb.Run{RunId: runID}, &testpilotspb.Verdict{}, nil
}

func (s *script) decide(*testpilotspb.Run, *testpilotspb.Verdict) Outcome {
	s.decided++
	status := StatusAccepted
	if s.decided <= len(s.statuses) {
		status = s.statuses[s.decided-1]
	}
	return Outcome{Status: status, Receipt: []byte("receipt-" + status)}
}

func runIDs(iterations []Iteration) []string {
	var ids []string
	for _, iteration := range iterations {
		ids = append(ids, iteration.RunID)
	}
	return ids
}

// Two accepted Runs, serially, each fenced on one lease; cleanup verifies both closed and releases
// the lease, and the record says so.
func TestRunMakesThePolicysIterationsAndReleasesTheLease(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer()}
	var progress bytes.Buffer
	run, store := s.invocation(t, canary, time.Now(), &progress)
	result, err := run.run(t.Context())
	require.NoError(t, err)

	require.False(t, result.Unreconciled)
	require.Equal(t, recovery.HeldTook, result.Lease.Held)
	require.Len(t, result.Iterations, canary.Limits.Iterations)
	require.Equal(t, []string{"testpilot.run.1", "testpilot.run.2"}, runIDs(result.Iterations))
	for _, iteration := range result.Iterations {
		require.Equal(t, StatusAccepted, iteration.Outcome.Status)
		require.Equal(t, []byte("receipt-accepted"), iteration.Outcome.Receipt)
	}
	require.Empty(t, result.Stopped)
	require.Equal(t, 2, s.released, "each iteration releases its own Driver")

	require.Equal(t, []string{"testpilot.run.1", "testpilot.run.2"}, result.Cleanup.Fenced)
	require.Equal(t, result.Cleanup.Fenced, result.Cleanup.Closed)
	require.Empty(t, result.Cleanup.Unverified)
	require.True(t, result.Cleanup.Released)
	require.NoError(t, result.Cleanup.Err)
	observed, err := leaseState(t.Context(), target(s.server), canary.Lease.WorkflowID)
	require.NoError(t, err)
	require.Equal(t, Observed{State: LeaseReleased, RunID: result.Lease.RunID, Started: time.Unix(1000, 0).UTC()}, observed)

	record := store.Snapshot()
	require.Equal(t, recovery.PhaseReleased, record.Phase)
	require.Equal(t, result.Lease, record.Lease)
	require.Equal(t, []recovery.Iteration{{RunID: "testpilot.run.1"}, {RunID: "testpilot.run.2"}}, record.Iterations)
	require.Contains(t, progress.String(), "cleanup released the lease")
}

// The loop never makes more Runs than the policy allows, whatever decide says.
func TestRunStopsAtThePolicysIterations(t *testing.T) {
	canary := testPolicy(t)
	canary.Limits.Iterations = 3
	s := &script{server: newFakeServer()}
	run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Iterations, 3)
	require.Equal(t, 3, s.runs)
}

// A rejected, incomplete or unconstructible iteration ends the invocation: no further Run is made
// against production after it, and cleanup still runs.
func TestRunStopsAfterTheFirstIterationNotAccepted(t *testing.T) {
	for _, status := range []string{StatusRejected, StatusIncomplete, StatusUnconstructible, "bogus"} {
		t.Run(status, func(t *testing.T) {
			s := &script{server: newFakeServer(), statuses: []string{status}}
			run, store := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.Len(t, result.Iterations, 1)
			require.Equal(t, 1, s.runs)
			want := status
			if status == "bogus" {
				want = StatusUnconstructible
			}
			require.Equal(t, want, result.Iterations[0].Outcome.Status)
			require.Contains(t, result.Stopped, "iteration 1 was "+want)
			require.True(t, result.Cleanup.Released)
			require.Equal(t, recovery.PhaseReleased, store.Snapshot().Phase)
		})
	}

	t.Run("a Run that errors", func(t *testing.T) {
		s := &script{server: newFakeServer(), runErr: errors.New("the Run could not complete")}
		run, _ := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
		result, err := run.run(t.Context())
		require.NoError(t, err)
		require.Len(t, result.Iterations, 1)
		require.Equal(t, StatusUnconstructible, result.Iterations[0].Outcome.Status)
		require.Equal(t, "testpilot.run.1", result.Iterations[0].RunID)
		require.Zero(t, s.decided, "an errored Run is never decided")
		require.True(t, result.Cleanup.Released)
	})
}

// No Run starts when what is left of the invocation limit is less than one Run's bound.
func TestRunStartsNoRunPastTheInvocationLimit(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer()}
	run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	// One second short of an iteration's whole bound, which is longer than the Run's own limits.
	run.Started = time.Now().Add(-canary.Limits.Invocation() + run.iterationBound() - time.Second)
	require.Greater(t, run.iterationBound(), 2*time.Minute)
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Empty(t, result.Iterations)
	require.Zero(t, s.runs)
	require.Contains(t, result.Stopped, "invocation limit")
	require.True(t, result.Cleanup.Released, "a held lease is released on every exit")
}

// An open lease, one that timed out or one closed any way but a canary termination is refused
// before anything starts, and recorded as found; a lease the canary released is taken anew.
func TestRunRefusesAnUnreconciledLease(t *testing.T) {
	canary := testPolicy(t)
	leaseID := canary.Lease.WorkflowID
	for name, prepare := range map[string]func(*fakeServer) string{
		"an open lease": func(server *fakeServer) string { return server.open(leaseID, time.Unix(1, 0)) },
		"a timed-out lease": func(server *fakeServer) string {
			runID := server.open(leaseID, time.Unix(1, 0))
			server.timeOut(leaseID)
			return runID
		},
		"a lease terminated by hand": func(server *fakeServer) string {
			runID := server.open(leaseID, time.Unix(1, 0))
			require.NoError(t, terminate(t.Context(), target(server), &commonpb.WorkflowExecution{WorkflowId: leaseID, RunId: runID}, "operator"))
			return runID
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := &script{server: newFakeServer()}
			runID := prepare(s.server)
			s.server.calls = nil
			run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.True(t, result.Unreconciled)
			require.Equal(t, &recovery.Lease{WorkflowID: leaseID, RunID: runID, Held: recovery.HeldFound}, result.Lease)
			require.Empty(t, result.Iterations)
			require.Nil(t, result.Cleanup, "a found lease is never cleaned up by run")
			require.Zero(t, s.runs)
			require.NotContains(t, s.server.calls, "start "+leaseID)
			require.NotContains(t, s.server.calls, "terminate "+leaseID)
			record := store.Snapshot()
			require.Equal(t, recovery.PhaseRefused, record.Phase)
			require.Equal(t, result.Lease, record.Lease)
		})
	}

	t.Run("a released lease", func(t *testing.T) {
		s := &script{server: newFakeServer()}
		first, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		result, err := first.run(t.Context())
		require.NoError(t, err)
		require.True(t, result.Cleanup.Released)
		second, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		again, err := second.run(t.Context())
		require.NoError(t, err)
		require.False(t, again.Unreconciled)
		require.NotEqual(t, result.Lease.RunID, again.Lease.RunID, "each invocation holds its own fence")
	})

	t.Run("a lease taken between the read and the start", func(t *testing.T) {
		s := &script{server: newFakeServer()}
		run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		run.target.Service = raced{fakeServer: s.server, leaseID: leaseID}
		result, err := run.run(t.Context())
		require.NoError(t, err)
		require.True(t, result.Unreconciled)
		require.Equal(t, recovery.HeldFound, result.Lease.Held)
		require.NotEmpty(t, result.Lease.RunID)
		require.Zero(t, s.runs)
	})

	t.Run("an unreadable lease", func(t *testing.T) {
		s := &script{server: newFakeServer()}
		s.server.fail["describe "+leaseID] = serviceerror.NewUnavailable("down")
		run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
		_, err := run.run(t.Context())
		require.Error(t, err, "a tooling failure before any lease")
		require.NotContains(t, s.server.calls, "start "+leaseID)
		require.Nil(t, store.Snapshot().Lease)
	})
}

// raced starts another holder's lease run just before the canary's own start reaches the server.
type raced struct {
	*fakeServer
	leaseID string
}

func (r raced) DescribeWorkflowExecution(ctx context.Context, request *workflowservice.DescribeWorkflowExecutionRequest, options ...grpc.CallOption) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	response, err := r.fakeServer.DescribeWorkflowExecution(ctx, request, options...)
	if request.GetExecution().GetWorkflowId() == r.leaseID && len(r.runs[r.leaseID]) == 0 {
		r.open(r.leaseID, time.Unix(1, 0))
	}
	return response, err
}

// Cleanup terminates exactly the fenced workflows still open, and nothing else on the target.
func TestCleanupClosesOnlyTheFencedWorkflows(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer(), leaveOpen: true}
	s.server.open("customer-workflow", time.Unix(1, 0))
	run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"testpilot.run.1", "testpilot.run.2"}, result.Cleanup.Closed)
	require.True(t, result.Cleanup.Released)
	require.Contains(t, s.server.calls, "terminate testpilot.run.1")
	require.NotContains(t, s.server.calls, "terminate customer-workflow")
	closed, err := workflowClosed(t.Context(), target(s.server), "customer-workflow", time.Second, noWait)
	require.NoError(t, err)
	require.False(t, closed, "a workflow the fence does not name is never touched")
}

// A fenced workflow that is not verified closed leaves the lease held and the scope uncertain,
// apart from the iterations' outcomes.
func TestCleanupLeavesTheLeaseHeldWhenAWorkflowIsNotVerifiedClosed(t *testing.T) {
	canary := testPolicy(t)
	for name, stick := range map[string]func(*fakeServer){
		"a workflow that stays open": func(server *fakeServer) { server.terminal["testpilot.run.2"] = true },
		"a termination that fails": func(server *fakeServer) {
			server.fail["terminate testpilot.run.2"] = serviceerror.NewUnavailable("down")
		},
		"an unreadable fence": func(server *fakeServer) {
			server.fail["history "+canary.Lease.WorkflowID] = serviceerror.NewUnavailable("down")
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := &script{server: newFakeServer(), leaveOpen: true}
			s.between = func(index int) {
				if index == 2 {
					stick(s.server)
				}
			}
			run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.Len(t, result.Iterations, 2)
			for _, iteration := range result.Iterations {
				require.Equal(t, StatusAccepted, iteration.Outcome.Status, "cleanup never changes an iteration's outcome")
			}
			require.False(t, result.Cleanup.Released)
			require.Error(t, result.Cleanup.Err)
			require.Contains(t, result.Cleanup.Unverified, "testpilot.run.2")
			require.Equal(t, recovery.PhaseUncertain, store.Snapshot().Phase)
			observed, err := leaseState(t.Context(), Target{Service: s.server, Namespace: testNamespace}, canary.Lease.WorkflowID)
			if err == nil {
				require.Equal(t, LeaseOpen, observed.State, "the lease stays held")
			}
		})
	}
}

// A lease closed under a running invocation is a stale fence: the next Run's fence fails, so it
// never opens, and cleanup cannot release a lease it no longer holds.
func TestAStaleFenceStopsTheInvocation(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer()}
	s.between = func(index int) {
		if index == 1 {
			lease := s.server.latest(canary.Lease.WorkflowID)
			require.NoError(t, terminate(t.Context(), target(s.server),
				&commonpb.WorkflowExecution{WorkflowId: canary.Lease.WorkflowID, RunId: lease.runID}, "operator"))
		}
	}
	run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Iterations, 2)
	require.Equal(t, StatusAccepted, result.Iterations[0].Outcome.Status)
	require.Equal(t, StatusUnconstructible, result.Iterations[1].Outcome.Status)
	require.Empty(t, result.Iterations[1].RunID, "the second Run never opened")
	require.Len(t, s.server.runs["testpilot.run.1"], 1)
	require.NotContains(t, s.server.runs, "testpilot.run.2")
	require.False(t, result.Cleanup.Released)
	require.Equal(t, recovery.PhaseUncertain, store.Snapshot().Phase)
}

// Progress is redacted and never grows past the policy's limit.
func TestProgressIsRedactedAndBounded(t *testing.T) {
	canary := testPolicy(t)
	canary.Limits.ProgressBytes = 40
	s := &script{server: newFakeServer()}
	var progress bytes.Buffer
	run, _ := s.invocation(t, canary, time.Now(), &progress)
	run.logf("namespace %s", testNamespace)
	require.NoError(t, run.progress.Close())
	require.NotContains(t, progress.String(), testNamespace)
	require.Contains(t, progress.String(), authority.Redacted)

	progress.Reset()
	run, _ = s.invocation(t, canary, time.Now(), &progress)
	_, err := run.run(t.Context())
	require.NoError(t, err)
	require.NoError(t, run.progress.Close())
	require.LessOrEqual(t, progress.Len(), 40)
	require.True(t, strings.HasPrefix(progress.String(), "lease taken"))
}

// Cleanup's deadline is absolute: an invocation whose iterations ran past its limit and reserve
// cleans up nothing more and leaves the lease held, rather than outliving what reconcile assumes.
func TestCleanupEndsAtTheInvocationLimitPlusTheReserve(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer()}
	run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	s.between = func(index int) {
		if index == 1 {
			run.Started = time.Now().Add(-canary.Limits.Invocation() - canary.Limits.CleanupReserve())
		}
	}
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.False(t, result.Cleanup.Released)
	require.ErrorIs(t, result.Cleanup.Err, context.DeadlineExceeded)
	require.Equal(t, recovery.PhaseUncertain, store.Snapshot().Phase)
	observed, err := leaseState(t.Context(), target(s.server), canary.Lease.WorkflowID)
	require.NoError(t, err)
	require.Equal(t, LeaseOpen, observed.State, "the lease stays held for reconcile")
}

// A Driver that does not release stops the loop, so no Driver starts beside it, and a release that
// fails after an errored Run is reported with it.
func TestADriverThatDoesNotReleaseStopsTheLoop(t *testing.T) {
	stuck := errors.New("the worker did not stop")
	s := &script{server: newFakeServer()}
	run, _ := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.openDriver = func() (testpilot.Driver, func(context.Context) error, error) {
		return &stubDriver{}, func(context.Context) error { return stuck }, nil
	}
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Iterations, 1)
	require.Equal(t, StatusAccepted, result.Iterations[0].Outcome.Status, "the Run's own outcome stands")
	require.Contains(t, result.Stopped, "did not release")
	require.True(t, result.Cleanup.Released)

	s = &script{server: newFakeServer(), runErr: errors.New("the Run failed")}
	run, _ = s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.openDriver = func() (testpilot.Driver, func(context.Context) error, error) {
		return &stubDriver{}, func(context.Context) error { return stuck }, nil
	}
	result, err = run.run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StatusUnconstructible, result.Iterations[0].Outcome.Status)
	require.ErrorIs(t, result.Iterations[0].Outcome.Err, stuck)
}

// A panic in the Run or in decide is an unconstructible iteration, and cleanup still runs.
func TestAPanicIsUnconstructibleAndCleanupStillRuns(t *testing.T) {
	s := &script{server: newFakeServer()}
	run, store := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.Decide = func(*testpilotspb.Run, *testpilotspb.Verdict) Outcome { panic("decide broke") }
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Iterations, 1)
	require.Equal(t, StatusUnconstructible, result.Iterations[0].Outcome.Status)
	require.ErrorContains(t, result.Iterations[0].Outcome.Err, "decide broke")
	require.True(t, result.Cleanup.Released)
	require.Equal(t, recovery.PhaseReleased, store.Snapshot().Phase)

	s = &script{server: newFakeServer()}
	run, _ = s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.runCase = func(context.Context, testpilot.Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
		panic("run broke")
	}
	result, err = run.run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StatusUnconstructible, result.Iterations[0].Outcome.Status)
	require.True(t, result.Cleanup.Released)
}

func TestRunRequiresEveryInput(t *testing.T) {
	_, err := Run(t.Context(), Config{})
	require.Error(t, err)
}
