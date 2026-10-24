package controller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	"umpire.local/model/runner"
)

// testGuest stands for the Model guest; the scripted Runs never execute it.
var testGuest = []byte("guest")

// scriptedRun is a Run of the test guest from world with the verdict status names.
func scriptedRun(world *anypb.Any, status string) *runner.Run {
	hash := sha256.Sum256(testGuest)
	run := &runner.Run{ModelHash: hex.EncodeToString(hash[:]), ProtocolVersion: runner.ProtocolVersion, World: world, Seed: 1}
	switch status {
	case StatusUnexplained:
		run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Unexplained{Unexplained: &runv1.Unexplained{Event: 3}}}
	case StatusRefused:
		run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Refused{Refused: &runv1.Refused{Event: 3, Reason: "driver fault"}}}
	case StatusExplained:
		run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_QUIESCENT}}}
	default:
		// A Run with no verdict, which the controller counts as failed.
	}
	return run
}

// startScripted is what a scripted Run does first: start the World's activity through the fenced
// Driver, as the Temporal Driver would, and return its ID.
func startScripted(ctx context.Context, world *anypb.Any, newDriver runner.NewDriver) (string, error) {
	driver, err := newDriver(ctx)
	if err != nil {
		return "", err
	}
	if _, err := driver.Start(ctx, 1, world); err != nil {
		return "", err
	}
	start := new(workflowservice.StartActivityExecutionRequest)
	if err := world.UnmarshalTo(start); err != nil {
		return "", err
	}
	return start.GetActivityId(), nil
}

// script is one scripted invocation: each Run starts its activity through its fenced Driver on the
// fake server and leaves it open or closes it, and its verdict follows the statuses.
type script struct {
	server   *fakeServer
	statuses []string
	// leaveOpen leaves each Run's activity open, as a Run cut off by its deadline would.
	leaveOpen bool
	// runErr fails the Run after its activity starts.
	runErr error
	// between runs after each Run, before the next.
	between  func(index int)
	runs     int
	released int
	ids      []string
}

func (s *script) invocation(t *testing.T, canary *policy.Policy, started time.Time, progress *bytes.Buffer) (*invocation, *recovery.Store) {
	t.Helper()
	store, err := recovery.Create(filepath.Join(t.TempDir(), "recovery.json"), "1234-1")
	require.NoError(t, err)
	run := newInvocation(Config{
		Policy: canary, Scope: &preflight.Scope{InvocationID: "1234-1", Guest: testGuest},
		Namespace: testNamespace, TaskQueue: "queue", Transport: authority.Transport{Target: "localhost:7233", Credentials: insecure.NewCredentials()},
		Redactor: authority.NewRedactor(testNamespace), Service: s.server, Identity: "umpire-canary",
		Recovery: store, Progress: progress, Started: started, Wait: noWait,
	})
	run.openDriver = func() (runner.NewDriver, func(context.Context) error, error) {
		return func(context.Context) (runner.Driver, error) { return &stubDriver{}, nil },
			func(context.Context) error { s.released++; return nil }, nil
	}
	run.runModel = s.run
	return run, store
}

func (s *script) run(ctx context.Context, world *anypb.Any, newDriver runner.NewDriver) (*runner.Run, error) {
	s.runs++
	id, err := startScripted(ctx, world, newDriver)
	if err != nil {
		return nil, err
	}
	s.ids = append(s.ids, id)
	s.server.open(id, time.Unix(3000, 0))
	if !s.leaveOpen {
		s.server.finish(id)
	}
	if s.between != nil {
		s.between(s.runs)
	}
	if s.runErr != nil {
		return nil, s.runErr
	}
	status := StatusExplained
	if s.runs <= len(s.statuses) {
		status = s.statuses[s.runs-1]
	}
	return scriptedRun(world, status), nil
}

func activityIDs(iterations []Iteration) []string {
	var ids []string
	for _, iteration := range iterations {
		ids = append(ids, iteration.ActivityID)
	}
	return ids
}

// Two explained Runs, serially, each on a fresh activity fenced on one lease; cleanup verifies both
// closed and releases the lease, and the record says so.
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
	require.Equal(t, s.ids, activityIDs(result.Iterations))
	require.NotEqual(t, s.ids[0], s.ids[1], "each iteration has its own activity")
	for _, iteration := range result.Iterations {
		require.True(t, IsActivityID(iteration.ActivityID))
		require.Equal(t, StatusExplained, iteration.Outcome.Status)
		require.NotNil(t, iteration.Outcome.Run)
	}
	require.Empty(t, result.Stopped)
	require.Equal(t, 2, s.released, "each iteration releases its own Driver")

	require.Equal(t, s.ids, result.Cleanup.Fenced)
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
	require.Equal(t, []recovery.Iteration{{ActivityID: s.ids[0]}, {ActivityID: s.ids[1]}}, record.Iterations)
	require.Contains(t, progress.String(), "cleanup released the lease")
}

// The loop never makes more Runs than the policy allows.
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

// An unexplained, refused or verdictless Run ends the invocation: no further Run is made against
// production after it, and cleanup still runs.
func TestRunStopsAfterTheFirstRunNotExplained(t *testing.T) {
	for _, status := range []string{StatusUnexplained, StatusRefused, "none"} {
		t.Run(status, func(t *testing.T) {
			s := &script{server: newFakeServer(), statuses: []string{status}}
			run, store := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.Len(t, result.Iterations, 1)
			require.Equal(t, 1, s.runs)
			want := status
			if status == "none" {
				want = StatusFailed
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
		require.Equal(t, StatusFailed, result.Iterations[0].Outcome.Status)
		require.Equal(t, s.ids[0], result.Iterations[0].ActivityID)
		require.Nil(t, result.Iterations[0].Outcome.Run, "an errored Run records nothing")
		require.True(t, result.Cleanup.Released)
	})
}

// No Run starts when what is left of the invocation limit is less than one Run's bound.
func TestRunStartsNoRunPastTheInvocationLimit(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer()}
	run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	// One second short of an iteration's whole bound, which is longer than the Run's own budget.
	run.Started = time.Now().Add(-canary.Limits.Invocation() + run.iterationBound() - time.Second)
	require.Greater(t, run.iterationBound(), runTimeout)
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
	r.mu.Lock()
	first := request.GetExecution().GetWorkflowId() == r.leaseID && len(r.runs[r.leaseID]) == 0
	r.mu.Unlock()
	if first {
		r.open(r.leaseID, time.Unix(1, 0))
	}
	return response, err
}

// Cleanup terminates exactly the fenced activities still open, and nothing else on the target.
func TestCleanupClosesOnlyTheFencedActivities(t *testing.T) {
	canary := testPolicy(t)
	s := &script{server: newFakeServer(), leaveOpen: true}
	s.server.open("customer-activity", time.Unix(1, 0))
	run, _ := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Equal(t, s.ids, result.Cleanup.Closed)
	require.True(t, result.Cleanup.Released)
	require.Contains(t, s.server.calls, "terminate "+s.ids[0])
	require.NotContains(t, s.server.calls, "terminate customer-activity")
	closed, err := activityClosed(t.Context(), target(s.server), "customer-activity", time.Second, noWait)
	require.NoError(t, err)
	require.False(t, closed, "an activity the fence does not name is never touched")
}

// A fenced activity that is not verified closed leaves the lease held and the scope uncertain,
// apart from the iterations' outcomes.
func TestCleanupLeavesTheLeaseHeldWhenAnActivityIsNotVerifiedClosed(t *testing.T) {
	canary := testPolicy(t)
	for name, stick := range map[string]func(*fakeServer, string){
		"an activity that stays open": func(server *fakeServer, id string) { server.terminal[id] = true },
		"a termination that fails": func(server *fakeServer, id string) {
			server.fail["terminate "+id] = serviceerror.NewUnavailable("down")
		},
		"an unreadable fence": func(server *fakeServer, _ string) {
			server.fail["history "+canary.Lease.WorkflowID] = serviceerror.NewUnavailable("down")
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := &script{server: newFakeServer(), leaveOpen: true}
			s.between = func(index int) {
				if index == 2 {
					stick(s.server, s.ids[1])
				}
			}
			run, store := s.invocation(t, canary, time.Now(), &bytes.Buffer{})
			result, err := run.run(t.Context())
			require.NoError(t, err)
			require.Len(t, result.Iterations, 2)
			for _, iteration := range result.Iterations {
				require.Equal(t, StatusExplained, iteration.Outcome.Status, "cleanup never changes an iteration's outcome")
			}
			require.False(t, result.Cleanup.Released)
			require.Error(t, result.Cleanup.Err)
			require.Contains(t, result.Cleanup.Unverified, s.ids[1])
			require.Equal(t, recovery.PhaseUncertain, store.Snapshot().Phase)
			delete(s.server.fail, "history "+canary.Lease.WorkflowID)
			observed, err := leaseState(t.Context(), target(s.server), canary.Lease.WorkflowID)
			require.NoError(t, err)
			require.Equal(t, LeaseOpen, observed.State, "the lease stays held")
		})
	}
}

// A lease closed under a running invocation is a stale fence: the next activity's fence fails, so
// its Run never starts, and cleanup cannot release a lease it no longer holds.
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
	require.Equal(t, StatusExplained, result.Iterations[0].Outcome.Status)
	require.Equal(t, StatusFailed, result.Iterations[1].Outcome.Status)
	require.Empty(t, result.Iterations[1].ActivityID, "the second activity was never fenced")
	require.Equal(t, 1, s.runs, "the second Run never started")
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
	stuckDriver := func() (runner.NewDriver, func(context.Context) error, error) {
		return func(context.Context) (runner.Driver, error) { return &stubDriver{}, nil }, func(context.Context) error { return stuck }, nil
	}
	s := &script{server: newFakeServer()}
	run, _ := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.openDriver = stuckDriver
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Iterations, 1)
	require.Equal(t, StatusExplained, result.Iterations[0].Outcome.Status, "the Run's own outcome stands")
	require.Contains(t, result.Stopped, "did not release")
	require.True(t, result.Cleanup.Released)

	s = &script{server: newFakeServer(), runErr: errors.New("the Run failed")}
	run, _ = s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.openDriver = stuckDriver
	result, err = run.run(t.Context())
	require.NoError(t, err)
	require.Equal(t, StatusFailed, result.Iterations[0].Outcome.Status)
	require.ErrorIs(t, result.Iterations[0].Outcome.Err, stuck)
}

// A panic in the Run is a failed iteration, and cleanup still runs.
func TestAPanicIsAFailedIterationAndCleanupStillRuns(t *testing.T) {
	s := &script{server: newFakeServer()}
	run, store := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	run.runModel = func(context.Context, *anypb.Any, runner.NewDriver) (*runner.Run, error) { panic("run broke") }
	result, err := run.run(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Iterations, 1)
	require.Equal(t, StatusFailed, result.Iterations[0].Outcome.Status)
	require.ErrorContains(t, result.Iterations[0].Outcome.Err, "run broke")
	require.True(t, result.Cleanup.Released)
	require.Equal(t, recovery.PhaseReleased, store.Snapshot().Phase)
}

// The canary's World is one fresh activity on the canary queue whose worker completes it.
func TestTheWorldIsOneCompletingActivity(t *testing.T) {
	s := &script{server: newFakeServer()}
	run, _ := s.invocation(t, testPolicy(t), time.Now(), &bytes.Buffer{})
	world, err := run.world(activityID(1))
	require.NoError(t, err)
	start := new(workflowservice.StartActivityExecutionRequest)
	require.NoError(t, world.UnmarshalTo(start))
	require.Equal(t, testNamespace, start.GetNamespace())
	require.Equal(t, activityID(1), start.GetActivityId())
	require.Equal(t, "queue", start.GetTaskQueue().GetName())
	require.Equal(t, int32(1), start.GetRetryPolicy().GetMaximumAttempts())
	require.Equal(t, "Complete", string(start.GetInput().GetPayloads()[0].GetData()))
}

func TestRunRequiresEveryInput(t *testing.T) {
	_, err := Run(t.Context(), Config{})
	require.Error(t, err)
}
