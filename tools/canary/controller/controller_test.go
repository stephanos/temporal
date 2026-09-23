package controller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/recordedrun"
	"google.golang.org/grpc/credentials/insecure"
)

var invokeCoordinates = authority.Coordinates{
	GRPC: "canary-frontend.example.internal:7233", Namespace: testNamespace, TaskQueue: "canary-queue-2b9",
	HandlerQueue: "canary-handler-queue-5e3", NexusEndpoint: "canary-endpoint-8d4",
}

// recordedRun is the test cluster's recorded canary Run, under another Run ID.
func recordedRun(t *testing.T, runID string) *testpilotspb.Run {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "assessment", "testdata", "nexusCallerCanary-syncCompletion-run.json"))
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(encoded)
	require.NoError(t, err)
	decoded.Run.RunId = runID
	return decoded.Run
}

// invokeFixture is one invocation against the fake server: the policy configured for the test
// coordinates, a plaintext transport, the dispatch's environment, and iterations scripted from the
// recorded Run.
type invokeFixture struct {
	server   *fakeServer
	policy   *policy.Policy
	env      map[string]string
	output   string
	recovery string
	progress bytes.Buffer
	// run edits each scripted Run before it returns; nil returns it as recorded.
	run func(index int, run *testpilotspb.Run) *testpilotspb.Run
	// decide replaces fn-26's decision; nil keeps the real one.
	decide Decide
	// between runs after each Run.
	between func(index int)
	runs    int
}

func newInvokeFixture(t *testing.T) *invokeFixture {
	t.Helper()
	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.Coordinates = invokeCoordinates.Digests()
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
		Policy: func() (*policy.Policy, *evaluation.Profile, error) {
			profile, err := assessment.LoadProfile(f.policy.EvaluationProfile)
			return f.policy, profile, err
		},
		Authority: func(authority.Lookup) (*authority.Authority, error) {
			return &authority.Authority{
				Coordinates: invokeCoordinates,
				Transport:   authority.Transport{Target: invokeCoordinates.GRPC, Credentials: insecure.NewCredentials()},
				Redactor: authority.NewRedactor(invokeCoordinates.GRPC, invokeCoordinates.TaskQueue,
					invokeCoordinates.HandlerQueue, invokeCoordinates.NexusEndpoint),
			}, nil
		},
	}
}

func (f *invokeFixture) invoke(t *testing.T, seams Seams) (Summary, int) {
	t.Helper()
	return Invoke(t.Context(), Invocation{
		Seams: seams, Output: f.output, Recovery: f.recovery, Progress: &f.progress, Started: time.Now(),
		Lookup: func(key string) (string, bool) { value, ok := f.env[key]; return value, ok },
		service: func(*authority.Authority, string, io.Writer) (*lazyService, error) {
			return &lazyService{direct: f.server}, nil
		},
		prepare: func(run *invocation) {
			run.Wait = noWait
			run.wait = noWait
			run.openDriver = func() (testpilot.Driver, func(context.Context) error, error) {
				return &stubDriver{}, func(context.Context) error { return nil }, nil
			}
			run.runCase = func(ctx context.Context, driver testpilot.Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
				f.runs++
				id := runID(f.runs)
				if _, err := driver.Open(ctx, id, testpilot.PreparedProgram{}); err != nil {
					return nil, nil, err
				}
				f.server.open(id, time.Unix(3000, 0))
				f.server.finish(id)
				recorded := recordedRun(t, id)
				if f.run != nil {
					recorded = f.run(f.runs, recorded)
				}
				if f.between != nil {
					f.between(f.runs)
				}
				return recorded, recorded.GetVerdict(), nil
			}
			if f.decide != nil {
				run.Decide = f.decide
			}
		},
	})
}

func (f *invokeFixture) published(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(f.output)
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}

// decidedAs is fn-26's decision on the recorded Run with its subject edited first, as a rejected or
// incomplete iteration's receipt would be rendered.
func decidedAs(t *testing.T, canary *policy.Policy, edit func(*evaluation.Subject)) Decide {
	t.Helper()
	profile, err := assessment.LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	encoded, err := os.ReadFile(filepath.Join("..", "assessment", "testdata", "nexusCallerCanary-syncCompletion-run.json"))
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(encoded)
	require.NoError(t, err)
	return func(run *testpilotspb.Run, _ *testpilotspb.Verdict) Outcome {
		subject, err := assessment.Admit(canary, decoded.Driver, run)
		require.NoError(t, err)
		edit(subject)
		decision := evaluation.Assess(subject, *profile)
		receipt, err := evaluation.Render(subject, *profile, decision)
		require.NoError(t, err)
		return Outcome{Status: decision.Outcome, Receipt: receipt}
	}
}

// Every iteration accepted: preflight, the lease, two Runs admitted, assessed and rendered in
// memory, cleanup releasing the lease, then each receipt and its provenance published and recorded;
// the exit is 0.
func TestInvokePublishesEveryAcceptedIteration(t *testing.T) {
	f := newInvokeFixture(t)
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitAccepted, code, "%+v\n%s", summary, f.progress.String())
	require.Equal(t, StatusAccepted, summary.Status)
	require.Equal(t, "1234567-1", summary.Invocation)
	require.Len(t, summary.Iterations, 2)
	require.Equal(t, &SummaryCleanup{Outcome: assessment.CleanupReleased, Fenced: []string{runID(1), runID(2)}, Unverified: []string{}}, summary.Cleanup)

	var want []string
	for index, iteration := range summary.Iterations {
		require.Equal(t, runID(index+1), iteration.RunID)
		require.Equal(t, StatusAccepted, iteration.Status)
		want = append(want, iteration.Receipt+".json", iteration.Provenance+".provenance.json")

		encoded, err := os.ReadFile(filepath.Join(f.output, iteration.Provenance+".provenance.json"))
		require.NoError(t, err)
		provenance, err := assessment.DecodeProvenance(encoded)
		require.NoError(t, err)
		require.Equal(t, iteration.Receipt, provenance.Receipt)
		require.Equal(t, index+1, provenance.Invocation.Iteration)
		require.Equal(t, f.policy.Coordinates, provenance.Coordinates)
		require.Equal(t, assessment.CleanupReleased, provenance.Cleanup.Invocation)
		require.Equal(t, []string{runID(1), runID(2)}, provenance.Fenced)
	}
	slices.Sort(want)
	require.Equal(t, want, f.published(t))

	record, err := recovery.Read(f.recovery)
	require.NoError(t, err)
	require.Equal(t, recovery.PhaseFinished, record.Phase)
	require.Equal(t, []recovery.Iteration{{RunID: runID(1), Published: true}, {RunID: runID(2), Published: true}}, record.Iterations)
	for _, raw := range []string{invokeCoordinates.GRPC, invokeCoordinates.TaskQueue, invokeCoordinates.HandlerQueue, invokeCoordinates.NexusEndpoint} {
		require.NotContains(t, f.progress.String(), raw)
	}
}

// A rejected or incomplete iteration ends the invocation with exit 1, and its receipt and
// provenance are still published.
func TestInvokeExitsOneForARejectedOrIncompleteIteration(t *testing.T) {
	for status, edit := range map[string]func(*evaluation.Subject){
		evaluation.DecisionRejected:   func(s *evaluation.Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED },
		evaluation.DecisionIncomplete: func(s *evaluation.Subject) { s.Cleanup = testpilotspb.CLEANUP_STATUS_FAILED },
	} {
		t.Run(status, func(t *testing.T) {
			f := newInvokeFixture(t)
			f.decide = decidedAs(t, f.policy, edit)
			summary, code := f.invoke(t, f.seams())
			require.Equal(t, ExitDecided, code, "%+v", summary)
			require.Equal(t, status, summary.Status)
			require.Len(t, summary.Iterations, 1, "no Run follows one not accepted")
			require.Len(t, f.published(t), 2)
		})
	}
}

// A Run fn-26 does not admit is unconstructible: no receipt, the invocation ends, exit 3.
func TestInvokeExitsThreeForAnUnconstructibleIteration(t *testing.T) {
	f := newInvokeFixture(t)
	f.run = func(_ int, run *testpilotspb.Run) *testpilotspb.Run {
		run.CaseId = "temporal.case.nexusCallerTests.syncCompletion"
		return run
	}
	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitFailed, code)
	require.Equal(t, StatusUnconstructible, summary.Status)
	require.Contains(t, summary.Detail, "crossed")
	require.Equal(t, []SummaryIteration{{RunID: runID(1), Status: StatusUnconstructible}}, summary.Iterations)
	require.Empty(t, f.published(t), "an unconstructible iteration publishes nothing")
	require.Equal(t, assessment.CleanupReleased, summary.Cleanup.Outcome)
}

// Preflight refuses by name before any record, lease or read of the target.
func TestInvokeRefusesBeforeAnything(t *testing.T) {
	for name, test := range map[string]struct {
		edit   func(*invokeFixture, *Seams)
		status string
	}{
		"another ref": {func(f *invokeFixture, _ *Seams) { f.env[preflight.VariableRef] = "refs/heads/feature" }, preflight.StatusWorkflowContext},
		"an unconfigured policy": {func(f *invokeFixture, _ *Seams) {
			f.policy.Coordinates.NexusEndpoint = policy.Unconfigured
		}, preflight.StatusPolicyUnconfigured},
		"another target": {func(f *invokeFixture, _ *Seams) { f.policy.Coordinates.GRPC = policy.Digest("elsewhere:7233") }, preflight.StatusCoordinateMismatch},
		"no credential": {func(_ *invokeFixture, s *Seams) {
			s.Authority = func(authority.Lookup) (*authority.Authority, error) { return nil, authority.ErrNoCredential }
		}, StatusAuthorityUnavailable},
		"no policy": {func(_ *invokeFixture, s *Seams) {
			s.Policy = func() (*policy.Policy, *evaluation.Profile, error) { return nil, nil, errors.New("unreadable") }
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

// An unreconciled lease is exit 2 before any Run; an uncertain cleanup is exit 2 with every decided
// iteration still published, its provenance saying so.
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

	t.Run("an uncertain cleanup outranks a rejection", func(t *testing.T) {
		f := newInvokeFixture(t)
		f.decide = decidedAs(t, f.policy, func(s *evaluation.Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED })
		f.between = func(int) { f.server.fail["history "+f.policy.Lease.WorkflowID] = serviceerror.NewUnavailable("down") }
		summary, code := f.invoke(t, f.seams())
		require.Equal(t, ExitUncertain, code, "%+v", summary)
		require.Equal(t, StatusCleanupUncertain, summary.Status)
		require.Equal(t, assessment.CleanupUncertain, summary.Cleanup.Outcome)
		require.Equal(t, []string{runID(1)}, summary.Cleanup.Fenced, "an unreadable fence falls back to the iterations' own Runs")
		names := f.published(t)
		require.Len(t, names, 2, "publication follows the cleanup attempt whatever its outcome")
		for _, name := range names {
			if strings.HasSuffix(name, ".provenance.json") {
				encoded, err := os.ReadFile(filepath.Join(f.output, name))
				require.NoError(t, err)
				provenance, err := assessment.DecodeProvenance(encoded)
				require.NoError(t, err)
				require.Equal(t, assessment.CleanupUncertain, provenance.Cleanup.Invocation)
			}
		}
	})
}

// A name that already holds other bytes is a publication conflict, exit 3, and is never overwritten.
func TestInvokeExitsThreeOnAPublicationConflict(t *testing.T) {
	f := newInvokeFixture(t)
	profile, err := assessment.LoadProfile(f.policy.EvaluationProfile)
	require.NoError(t, err)
	// The first iteration's receipt, as Invoke will render it under the Case prepared for these
	// coordinates.
	bound, err := casebinding.Bind(f.policy, invokeCoordinates.Driver())
	require.NoError(t, err)
	first := decide(f.policy, profile, &preflight.Scope{Prepared: bound.Prepared}, recordedRun(t, runID(1)))
	require.Equal(t, StatusAccepted, first.Status, "%v", first.Err)
	receipt := first.Receipt
	planted := filepath.Join(f.output, evaluation.ReceiptIdentity(receipt)+".json")
	require.NoError(t, os.WriteFile(planted, []byte("other\n"), 0o644))

	summary, code := f.invoke(t, f.seams())
	require.Equal(t, ExitFailed, code)
	require.Equal(t, StatusPublicationConflict, summary.Status)
	kept, err := os.ReadFile(planted)
	require.NoError(t, err)
	require.Equal(t, "other\n", string(kept))
	require.Equal(t, []string{filepath.Base(planted)}, f.published(t), "nothing else is published when a name conflicts")
	record, err := recovery.Read(f.recovery)
	require.NoError(t, err)
	require.Equal(t, recovery.PhasePublishing, record.Phase)
}

func TestOutcomeKeepsTheHighestExit(t *testing.T) {
	var result outcome
	result.raise(StatusRejected, ExitDecided, "first")
	result.raise(StatusCleanupUncertain, ExitUncertain, "second")
	result.raise(StatusIncomplete, ExitDecided, "third")
	require.Equal(t, StatusCleanupUncertain, result.summary.Status)
	require.Equal(t, ExitUncertain, result.code)
	result.raise(StatusPublicationConflict, ExitFailed, "fourth")
	require.Equal(t, StatusPublicationConflict, result.summary.Status)
	require.Equal(t, "fourth", result.summary.Detail)
}
