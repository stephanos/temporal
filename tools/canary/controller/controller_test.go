package controller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/model/umpire/runner"
)

var invokeCoordinates = authority.Coordinates{
	GRPC: "canary-frontend.example.internal:7233", Namespace: testNamespace, TaskQueue: "canary-queue-2b9",
}

// invokeFixture is one invocation against the fake server: the policy configured for the test
// coordinates and pinning the test guest, a plaintext transport, the dispatch's environment, and
// scripted Runs.
type invokeFixture struct {
	server   *fakeServer
	policy   *policy.Policy
	env      map[string]string
	output   string
	recovery string
	progress bytes.Buffer
	// statuses are the scripted Runs' verdicts in order; past them a Run is explained.
	statuses []string
	// between runs after each Run.
	between func(index int)
	// release is each iteration's Driver release; nil releases cleanly.
	release func(context.Context) error
	// ctx is the invocation's context; nil is the test's.
	ctx context.Context
	// started is when the invocation started; zero is now.
	started time.Time
	// runErr fails each scripted Run after its activity starts.
	runErr error
	// start replaces the World each scripted Run starts its activity from; nil starts the World.
	start func(world *anypb.Any) *anypb.Any
	runs  int
	ids   []string
	// publish replaces runner.Publish; nil keeps it.
	publish func(context.Context, string, *runner.Run) (runner.Publication, error)
}

func newInvokeFixture(t *testing.T) *invokeFixture {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.Coordinates = invokeCoordinates.Digests()
	hash := sha256.Sum256(testGuest)
	canary.ModelHash = hex.EncodeToString(hash[:])
	root := t.TempDir()
	output := filepath.Join(root, "output")
	require.NoError(t, os.Mkdir(output, 0o755))
	return &invokeFixture{
		server: newFakeServer(), policy: canary, output: output, recovery: filepath.Join(root, "recovery.json"),
		env: map[string]string{
			preflight.VariableEventName: "workflow_dispatch", preflight.VariableRepository: canary.Repository,
			preflight.VariableRef:         canary.TrustedRef,
			preflight.VariableWorkflowRef: canary.Repository + "/" + canary.WorkflowPath + "@" + canary.TrustedRef,
			preflight.VariableRunID:       "1234567", preflight.VariableRunAttempt: "1",
		},
	}
}

func (f *invokeFixture) seams() Seams {
	return Seams{
		Policy: func() (*policy.Policy, error) { return f.policy, nil },
		Guest:  testGuest,
		Authority: func(authority.Lookup) (*authority.Authority, error) {
			return &authority.Authority{
				Coordinates: invokeCoordinates,
				Transport:   authority.Transport{Target: invokeCoordinates.GRPC, Credentials: insecure.NewCredentials()},
				Redactor:    authority.NewRedactor(invokeCoordinates.GRPC, invokeCoordinates.TaskQueue),
			}, nil
		},
	}
}

func (f *invokeFixture) invoke(t *testing.T, seams Seams) (Summary, int) {
	t.Helper()
	ctx := f.ctx
	if ctx == nil {
		ctx = t.Context()
	}
	return Invoke(ctx, Invocation{
		Seams: seams, Output: f.output, Recovery: f.recovery, Progress: &f.progress, Started: f.startedAt(),
		Lookup:  func(key string) (string, bool) { value, ok := f.env[key]; return value, ok },
		publish: f.publish,
		service: func(*authority.Authority, string, io.Writer) (*lazyService, error) {
			return &lazyService{direct: f.server}, nil
		},
		prepare: func(run *invocation) {
			run.Wait = noWait
			run.wait = noWait
			run.openDriver = func() (runner.NewDriver, func(context.Context) error, error) {
				release := f.release
				if release == nil {
					release = func(context.Context) error { return nil }
				}
				return func(context.Context) (runner.Driver, error) { return &stubDriver{}, nil }, release, nil
			}
			run.runModel = func(ctx context.Context, world *anypb.Any, newDriver runner.NewDriver) (*runner.Run, error) {
				f.runs++
				started := world
				if f.start != nil {
					started = f.start(world)
				}
				id, err := startScripted(ctx, started, newDriver)
				if err != nil {
					return nil, err
				}
				f.ids = append(f.ids, id)
				f.server.open(id, time.Unix(3000, 0))
				f.server.finish(id)
				if f.between != nil {
					f.between(f.runs)
				}
				if f.runErr != nil {
					return nil, f.runErr
				}
				status := StatusExplained
				if f.runs <= len(f.statuses) {
					status = f.statuses[f.runs-1]
				}
				return scriptedRun(world, status), nil
			}
		},
	})
}

func (f *invokeFixture) startedAt() time.Time {
	if f.started.IsZero() {
		return time.Now()
	}
	return f.started
}

// published is every file under the output directory, relative to it.
func (f *invokeFixture) published(t *testing.T) []string {
	t.Helper()
	var names []string
	require.NoError(t, filepath.WalkDir(f.output, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(f.output, path)
		names = append(names, relative)
		return err
	}))
	slices.Sort(names)
	return names
}

// Every Run explained: preflight, the lease, two Runs recorded in memory, cleanup releasing the
// lease, then each Run published with runner.Publish and recorded; the exit is 0.
func TestInvokePublishesEveryExplainedRun(t *testing.T) {
	f := newInvokeFixture(t)
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitAccepted, code, "%+v\n%s", summary, f.progress.String())
	require.Equal(t, StatusExplained, summary.Status)
	require.Equal(t, "1234567-1", summary.Invocation)
	require.Len(t, summary.Iterations, 2)
	require.Equal(t, &SummaryCleanup{Outcome: CleanupReleased, Fenced: f.ids, Unverified: []string{}}, summary.Cleanup)

	var keys []string
	for index, iteration := range summary.Iterations {
		require.Equal(t, f.ids[index], iteration.ActivityID)
		require.Equal(t, StatusExplained, iteration.Status)
		keys = append(keys, iteration.Run)
		published, err := runner.Load(filepath.Join(f.output, iteration.Run))
		require.NoError(t, err)
		require.Equal(t, f.policy.ModelHash, published.GetModelHash())
		require.NotNil(t, published.GetVerdict().GetExplained())
	}
	slices.Sort(keys)
	require.Equal(t, keys, f.published(t), "the Runs are the only artifact")

	record, err := recovery.Read(f.recovery)
	require.NoError(t, err)
	require.Equal(t, recovery.PhaseFinished, record.Phase)
	require.Equal(t, []recovery.Iteration{{ActivityID: f.ids[0], Published: true}, {ActivityID: f.ids[1], Published: true}}, record.Iterations)
	for _, raw := range []string{invokeCoordinates.GRPC, invokeCoordinates.TaskQueue} {
		require.NotContains(t, f.progress.String(), raw)
	}
}

// An unexplained or refused Run ends the invocation with exit 1, and that Run is still published.
func TestInvokeExitsOneForAnUnexplainedOrRefusedRun(t *testing.T) {
	for _, status := range []string{StatusUnexplained, StatusRefused} {
		t.Run(status, func(t *testing.T) {
			f := newInvokeFixture(t)
			f.statuses = []string{status}
			summary, code := f.invoke(t, f.seams())
			require.Equal(t, ExitDecided, code, "%+v", summary)
			require.Equal(t, status, summary.Status)
			require.Len(t, summary.Iterations, 1, "no Run follows one not explained")
			require.Len(t, f.published(t), 1)
			require.Equal(t, []string{summary.Iterations[0].Run}, f.published(t))
		})
	}
}

// A Run that errors is a failed iteration: nothing is published, the invocation ends, exit 3.
func TestInvokeExitsThreeForAFailedIteration(t *testing.T) {
	f := newInvokeFixture(t)
	f.runErr = errors.New("the Run could not complete")
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitFailed, code)
	require.Equal(t, StatusFailed, summary.Status)
	require.Contains(t, summary.Detail, "could not complete")
	require.Equal(t, []SummaryIteration{{ActivityID: f.ids[0], Status: StatusFailed}}, summary.Iterations)
	require.Empty(t, f.published(t), "a failed iteration publishes nothing")
	require.Equal(t, CleanupReleased, summary.Cleanup.Outcome)
}

// Preflight refuses by name before any record, lease or read of the target.
func TestInvokeRefusesBeforeAnything(t *testing.T) {
	for name, test := range map[string]struct {
		edit   func(*invokeFixture, *Seams)
		status string
	}{
		"another ref": {func(f *invokeFixture, _ *Seams) { f.env[preflight.VariableRef] = "refs/heads/feature" }, preflight.StatusWorkflowContext},
		"an unconfigured policy": {func(f *invokeFixture, _ *Seams) {
			f.policy.Coordinates.TaskQueue = policy.Unconfigured
		}, preflight.StatusPolicyUnconfigured},
		"another target": {func(f *invokeFixture, _ *Seams) { f.policy.Coordinates.GRPC = policy.Digest("elsewhere:7233") }, preflight.StatusCoordinateMismatch},
		"another guest":  {func(_ *invokeFixture, s *Seams) { s.Guest = []byte("another guest") }, preflight.StatusModelMismatch},
		"no credential": {func(_ *invokeFixture, s *Seams) {
			s.Authority = func(authority.Lookup) (*authority.Authority, error) { return nil, authority.ErrNoCredential }
		}, StatusAuthorityUnavailable},
		"no policy": {func(_ *invokeFixture, s *Seams) {
			s.Policy = func() (*policy.Policy, error) { return nil, errors.New("unreadable") }
		}, StatusPolicyUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			f := newInvokeFixture(t)
			seams := f.seams()
			test.edit(f, &seams)
			summary, code := f.invoke(t, seams)
			require.Equal(t, ExitFailed, code)
			require.Equal(t, test.status, summary.Status)
			require.Empty(t, summary.Iterations)
			require.Empty(t, f.server.calls, "nothing reached the target")
			_, err := os.Stat(f.recovery)
			require.ErrorIs(t, err, os.ErrNotExist, "no recovery record: reconcile finds nothing to reconcile")
			require.Empty(t, f.published(t))
		})
	}
}

// An unreconciled lease is exit 2 before any Run; an uncertain cleanup is exit 2 with every
// recorded Run still published.
func TestInvokeExitsTwoWhenTheScopeIsNotClean(t *testing.T) {
	t.Run("an unreconciled lease", func(t *testing.T) {
		f := newInvokeFixture(t)
		f.server.open(f.policy.Lease.WorkflowID, time.Unix(1, 0))
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitUncertain, code)
		require.Equal(t, StatusLeaseUnreconciled, summary.Status)
		require.Zero(t, f.runs)
		require.Empty(t, f.published(t))
		record, err := recovery.Read(f.recovery)
		require.NoError(t, err)
		require.Equal(t, recovery.HeldFound, record.Lease.Held)
	})

	t.Run("an uncertain cleanup outranks an unexplained Run", func(t *testing.T) {
		f := newInvokeFixture(t)
		f.statuses = []string{StatusUnexplained}
		f.between = func(int) { f.server.fail["history "+f.policy.Lease.WorkflowID] = serviceerror.NewUnavailable("down") }
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitUncertain, code, "%+v", summary)
		require.Equal(t, StatusCleanupUncertain, summary.Status)
		require.Equal(t, CleanupUncertain, summary.Cleanup.Outcome)
		require.Equal(t, f.ids, summary.Cleanup.Fenced, "an unreadable fence falls back to the iterations' own activities")
		require.Len(t, f.published(t), 1, "publication follows the cleanup attempt whatever its outcome")
	})
}

// A key that already holds other bytes is a publication conflict, exit 3, and is never overwritten.
func TestInvokeExitsThreeOnAPublicationConflict(t *testing.T) {
	f := newInvokeFixture(t)
	var planted string
	f.publish = func(ctx context.Context, root string, run *runner.Run) (runner.Publication, error) {
		if planted == "" {
			_, digest, err := runner.Canonical(run)
			require.NoError(t, err)
			planted = filepath.Join(root, run.GetModelHash(), "1", digest+runner.RunExtension)
			require.NoError(t, os.MkdirAll(filepath.Dir(planted), 0o755))
			require.NoError(t, os.WriteFile(planted, []byte("other\n"), 0o644))
		}
		return runner.Publish(ctx, root, run)
	}
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitFailed, code)
	require.Equal(t, StatusPublicationConflict, summary.Status)
	kept, err := os.ReadFile(planted)
	require.NoError(t, err)
	require.Equal(t, "other\n", string(kept))
	require.Len(t, f.published(t), 1, "nothing else is published when a key conflicts")
	record, err := recovery.Read(f.recovery)
	require.NoError(t, err)
	require.Equal(t, recovery.PhasePublishing, record.Phase)
}

// A Driver that does not release after an explained Run stops the invocation as a tooling
// failure, exit 3, and the Run is still published.
func TestInvokeExitsThreeWhenADriverDoesNotRelease(t *testing.T) {
	f := newInvokeFixture(t)
	f.release = func(context.Context) error { return errors.New("the worker did not stop") }
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitFailed, code)
	require.Equal(t, StatusToolingFailure, summary.Status)
	require.Contains(t, summary.Detail, "did not release")
	require.Len(t, summary.Iterations, 1)
	require.Equal(t, StatusExplained, summary.Iterations[0].Status)
	require.Len(t, f.published(t), 1, "the recorded Run is published")
}

// An interrupt ends the iterations, not the publication: every Run already recorded is published
// after the cleanup attempt, and the invocation exits 3 as interrupted.
func TestAnInterruptStillPublishesWhatWasRecorded(t *testing.T) {
	f := newInvokeFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f.ctx = ctx
	f.between = func(index int) {
		if index == 1 {
			cancel()
		}
	}
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitFailed, code, "%+v", summary)
	require.Equal(t, StatusInterrupted, summary.Status)
	require.Len(t, summary.Iterations, 1, "no Run starts after the interrupt")
	require.Equal(t, CleanupReleased, summary.Cleanup.Outcome, "cleanup runs under its own context")
	require.Len(t, f.published(t), 1)
	record, err := recovery.Read(f.recovery)
	require.NoError(t, err)
	require.Equal(t, []recovery.Iteration{{ActivityID: f.ids[0], Published: true}}, record.Iterations)
}

// The other named failures, each exit 3: no iteration at all within the limit, a publication that
// fails, a publication it cannot record, and a Driver that does not release after a Run that was
// not explained.
func TestInvokeNamesEveryOtherFailure(t *testing.T) {
	t.Run("no iteration within the limit", func(t *testing.T) {
		f := newInvokeFixture(t)
		f.started = time.Now().Add(-f.policy.Limits.Invocation() + time.Minute)
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitFailed, code)
		require.Equal(t, StatusNoIteration, summary.Status)
		require.Zero(t, f.runs)
		require.Equal(t, CleanupReleased, summary.Cleanup.Outcome, "the lease taken is released")
	})
	t.Run("a publication that fails", func(t *testing.T) {
		f := newInvokeFixture(t)
		seams := f.seams()
		seams.Hook = func(phase string) {
			if phase == PhaseCleaned {
				require.NoError(t, os.RemoveAll(f.output))
			}
		}
		summary, code := f.invoke(t, seams)
		require.Equal(t, ExitFailed, code)
		require.Equal(t, StatusPublicationFailed, summary.Status)
	})
	t.Run("a publication it cannot record", func(t *testing.T) {
		f := newInvokeFixture(t)
		var before []byte
		f.publish = func(ctx context.Context, root string, run *runner.Run) (runner.Publication, error) {
			publication, err := runner.Publish(ctx, root, run)
			// The record's path becomes a directory that cannot be replaced.
			before, _ = os.ReadFile(f.recovery)
			require.NoError(t, os.Remove(f.recovery))
			require.NoError(t, os.MkdirAll(filepath.Join(f.recovery, "occupied"), 0o700))
			return publication, err
		}
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitFailed, code)
		require.Equal(t, StatusPublicationUnreported, summary.Status)
		require.Contains(t, summary.Detail, "was published")
		require.Len(t, f.published(t), 1, "the first Run is published, and nothing after it")
		record, err := recovery.Decode(before)
		require.NoError(t, err)
		require.Equal(t, recovery.PhasePublishing, record.Phase, "the last record written never reads as finished")
		require.False(t, record.Iterations[0].Published)
	})
	t.Run("a Driver that does not release after an unexplained Run", func(t *testing.T) {
		f := newInvokeFixture(t)
		f.statuses = []string{StatusUnexplained}
		f.release = func(context.Context) error { return errors.New("the worker did not stop") }
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitFailed, code, "a tooling failure outranks the unexplained Run")
		require.Equal(t, StatusToolingFailure, summary.Status)
		require.Contains(t, f.progress.String(), "did not release")
		require.Len(t, f.published(t), 1, "the unexplained Run is still published")
	})
}

func TestOutcomeKeepsTheHighestExit(t *testing.T) {
	var result outcome
	result.raise(StatusUnexplained, ExitDecided, "first")
	result.raise(StatusCleanupUncertain, ExitUncertain, "second")
	result.raise(StatusRefused, ExitDecided, "third")
	require.Equal(t, StatusCleanupUncertain, result.summary.Status)
	require.Equal(t, ExitUncertain, result.code)
	result.raise(StatusPublicationConflict, ExitFailed, "fourth")
	require.Equal(t, StatusPublicationConflict, result.summary.Status)
	require.Equal(t, "fourth", result.summary.Detail)
}
