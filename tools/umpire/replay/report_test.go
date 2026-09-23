package replay

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/campaign"
	"google.golang.org/protobuf/proto"
)

// execution is one replay over the corpus's violated subject: its Case and recorded Run, the fake
// bridge admitting it by its identity, and the scripted binder its reruns bind through. Whether
// each environment function was called is recorded.
type execution struct {
	request     Request
	environment Environment
	binder      *scriptedBinder
	fake        *fakeReplayBridge
	started     bool
	opened      bool
}

func newExecution(t *testing.T, script []attemptScript, sweep ...fakeEdit) *execution {
	t.Helper()
	prepare := preparer(t)
	caseBytes := loadCorpusCase(t, "violated")
	_, _, recorded := recordedRunOf(t, prepare, profileName, caseBytes)
	subject, err := Admit(t.Context(), caseBytes, recorded, prepare)
	require.NoError(t, err)
	bridge, fake := newFakeReplayBridge(t, json.RawMessage(subject.Canonical), subject.Case.GetCaseId(), sweep...)
	fake.identity = subject.Identity
	e := &execution{
		request: Request{Case: caseBytes, Run: recorded, Set: "set", Named: Named{Query: "q"}},
		binder:  &scriptedBinder{prepare: prepare, script: script},
		fake:    fake,
	}
	clock := time.Unix(0, 0)
	e.environment = Environment{
		Prepare: prepare,
		StartBridge: func(context.Context) (*Bridge, error) {
			e.started = true
			return bridge, nil
		},
		OpenBinder: func(context.Context) (campaign.Binder, func(context.Context) error, error) {
			e.opened = true
			return e.binder, func(context.Context) error { return nil }, nil
		},
		PromotionRoot: filepath.Join(t.TempDir(), "proposals"),
		Limits:        DefaultLimits,
		Now:           func() time.Time { return clock },
	}
	return e
}

func (e *execution) run(t *testing.T) Report {
	t.Helper()
	return Execute(t.Context(), e.request, e.environment)
}

// The report's field set is pinned: each answer apart, the key beside the identity, and no field
// about history replay.
func TestReportFieldSetIsPinned(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	rendered, err := e.run(t).Render()
	require.NoError(t, err)
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rendered, &document))
	require.Equal(t, []string{"admission", "cleanup", "identity", "key", "limits", "proposal", "reduction", "reproduction", "semanticReplay"}, sortedKeys(document))
	require.Equal(t, byte('\n'), rendered[len(rendered)-1])
}

// A reproduced subject whose reduction completes exits 0 with its proposal written under the
// promotion root; the key and the Case identity are reported apart.
func TestExecuteReproducesReducesAndWritesTheProposal(t *testing.T) {
	e := newExecution(t, nil, edit(1, "b"), fakeEdit{Edit: edit(0, "a").Edit, inapplicable: true})
	report := e.run(t)
	require.Empty(t, report.Failure)
	require.Equal(t, StatusAdmitted, report.Admission.Status)
	require.Equal(t, StatusReproduced, report.SemanticReplay.Status)
	require.NotEmpty(t, report.Key)
	require.Len(t, report.Identity, 64)
	require.NotContains(t, report.Key, report.Identity)
	require.Equal(t, ClassReproduced, report.Reproduction.Class)
	require.Len(t, report.Reproduction.Reruns, Attempts)
	require.Equal(t, "minimized", report.Reduction.Status)
	require.Equal(t, ProposalWritten, report.Proposal.Status)
	require.FileExists(t, report.Proposal.Written)
	require.Equal(t, StatusReleased, report.Cleanup.Status)
	require.Equal(t, ExitReproduced, report.ExitCode())
}

// A rejected subject exits 3 before anything is opened, naming the reason; a recorded Run the
// offline replay does not reproduce names the semantic replay too.
func TestExecuteRejectsBeforeOpeningAnything(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	e.request.Case = append([]byte("  "), e.request.Case...)
	report := e.run(t)
	require.Equal(t, StatusRejected, report.Admission.Status)
	require.Equal(t, ReasonNoncanonical, report.Admission.Reason)
	require.Equal(t, StatusNotRun, report.SemanticReplay.Status)
	require.False(t, e.started || e.opened, "nothing is opened for a rejected subject")
	require.Equal(t, StatusNothing, report.Cleanup.Status)
	require.Equal(t, ExitToolingFailure, report.ExitCode())

	e = newExecution(t, nil, edit(0, "a"))
	driver, run, err := DecodeRecordedRun(e.request.Run)
	require.NoError(t, err)
	disagreeing := proto.CloneOf(run)
	disagreeing.Verdict.Rules[0].TerminalStateId = "elsewhere"
	e.request.Run, err = EncodeRecordedRun(driver, disagreeing)
	require.NoError(t, err)
	report = e.run(t)
	require.Equal(t, ReasonReplay, report.Admission.Reason)
	require.Equal(t, StatusRejected, report.SemanticReplay.Status)
	require.NotEmpty(t, report.SemanticReplay.Detail)
	require.False(t, e.started || e.opened)
	require.Equal(t, ExitToolingFailure, report.ExitCode())
}

// A subject the set does not produce is crossed at the bridge: the deployment is never opened.
func TestExecuteCrossedNeverOpensTheDeployment(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	e.fake.crossed = true
	report := e.run(t)
	require.Equal(t, ReasonCrossed, report.Admission.Reason)
	require.True(t, e.started)
	require.False(t, e.opened)
	require.Equal(t, StatusReleased, report.Cleanup.Status)
	require.Equal(t, ExitToolingFailure, report.ExitCode())
}

// The subject's reruns decide the exit code when they do not reproduce: 1 for not reproduced, 2
// for indeterminate, and the reduction is reported not attempted.
func TestExecuteExitsByTheSubjectsClass(t *testing.T) {
	for name, probe := range map[string]struct {
		script []attemptScript
		exit   int
	}{
		"not reproduced": {[]attemptScript{runSatisfied, nil}, ExitNotReproduced},
		"indeterminate":  {[]attemptScript{runIncomplete, nil}, ExitIndeterminate},
	} {
		t.Run(name, func(t *testing.T) {
			report := newExecution(t, probe.script, edit(0, "a")).run(t)
			require.False(t, report.Reduction.Attempted)
			require.Equal(t, ReductionNotAttempted, report.Reduction.Status)
			require.Equal(t, ProposalNone, report.Proposal.Status)
			require.Equal(t, probe.exit, report.ExitCode())
		})
	}
}

// An incomplete reduction exits 2 and proposes nothing; a proposal that cannot be written exits 3
// with the rest of the report standing; a deployment that cannot be opened exits 3.
func TestExecuteExitsOnIncompleteReductionsAndProposalFailures(t *testing.T) {
	report := newExecution(t, []attemptScript{nil, nil, runIncomplete, nil, runIncomplete}, edit(0, "a")).run(t)
	require.Equal(t, "incomplete", report.Reduction.Status)
	require.Equal(t, ProposalNone, report.Proposal.Status)
	require.Equal(t, ExitIndeterminate, report.ExitCode())

	e := newExecution(t, nil, edit(0, "a"))
	require.NoError(t, os.MkdirAll(e.environment.PromotionRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(e.environment.PromotionRoot, "set-candidate-0.lean"), []byte("reviewed"), 0o644))
	report = e.run(t)
	require.Equal(t, "minimized", report.Reduction.Status)
	require.Equal(t, ProposalWriteFailed, report.Proposal.Status)
	require.Contains(t, report.Proposal.Error, "file exists")
	require.Equal(t, ExitToolingFailure, report.ExitCode())

	e = newExecution(t, nil, edit(0, "a"))
	e.environment.OpenBinder = func(context.Context) (campaign.Binder, func(context.Context) error, error) {
		return nil, nil, errors.New("connection refused")
	}
	report = e.run(t)
	require.Contains(t, report.Failure, "connection refused")
	require.Nil(t, report.Reproduction)
	require.Equal(t, ExitToolingFailure, report.ExitCode())
}

// A stop during the subject's reruns is a stop, not a tooling failure: the subject is
// indeterminate, the reduction is not attempted, and the command exits 2.
func TestExecuteStoppedDuringTheSubjectsReruns(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	ctx, cancel := context.WithCancel(t.Context())
	e.binder.onRun = cancel
	report := Execute(ctx, e.request, e.environment)
	require.Empty(t, report.Failure)
	require.Equal(t, ClassIndeterminate, report.Reproduction.Class)
	require.True(t, report.Reduction.Stopped)
	require.False(t, report.Reduction.Attempted)
	require.Equal(t, ExitIndeterminate, report.ExitCode())
	require.Equal(t, "finish", e.fake.frames[len(e.fake.frames)-1].Frame, "the bridge is still told")
}

// A candidate rerun that cannot release is a tooling failure, exit 3, as the subject's would be.
func TestExecuteCandidateRerunFailureExitsThree(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	opened := e.environment.OpenBinder
	e.environment.OpenBinder = func(ctx context.Context) (campaign.Binder, func(context.Context) error, error) {
		binder, release, err := opened(ctx)
		e.binder.onRun = func() {
			if e.binder.binds > Attempts {
				e.binder.release = errors.New("namespace still held")
			}
		}
		return binder, release, err
	}
	report := e.run(t)
	require.Contains(t, report.Reduction.Failure, "namespace still held")
	require.Equal(t, ExitToolingFailure, report.ExitCode())
}

// A Query the bridge cannot recover is unrecovered, not crossed.
func TestExecuteNamesAnUnrecoveredQuery(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	e.fake.rejectAdmit = "set set has no Query q"
	report := e.run(t)
	require.Equal(t, StatusRejected, report.Admission.Status)
	require.Equal(t, ReasonUnrecovered, report.Admission.Reason)
	require.False(t, e.opened)
	require.Equal(t, ExitToolingFailure, report.ExitCode())
}

// A stop before the reruns -- while the deployment opens -- is a stop, exit 2, and the bridge is
// told; a stop that falls between the subject's two Runs keeps the Run that closed.
func TestExecuteStopsBeforeAndDuringTheReruns(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	ctx, cancel := context.WithCancel(t.Context())
	opened := e.environment.OpenBinder
	e.environment.OpenBinder = func(ctx context.Context) (campaign.Binder, func(context.Context) error, error) {
		cancel()
		_, _, _ = opened(ctx)
		return nil, nil, ctx.Err()
	}
	report := Execute(ctx, e.request, e.environment)
	require.Empty(t, report.Failure)
	require.True(t, report.Reduction.Stopped)
	require.Equal(t, ExitIndeterminate, report.ExitCode())
	require.Equal(t, "finish", e.fake.frames[len(e.fake.frames)-1].Frame)

	e = newExecution(t, nil, edit(0, "a"))
	ctx, cancel = context.WithCancel(t.Context())
	e.binder.onRun = cancel
	report = Execute(ctx, e.request, e.environment)
	require.Len(t, report.Reproduction.Reruns, 1, "the Run that closed before the stop is reported")
	require.Equal(t, 1, report.Reduction.Runs)
	require.Equal(t, ExitIndeterminate, report.ExitCode())
}

// A proposal that did not compile exits 3 with the rest of the report standing.
func TestExecuteProposalNotCompiledExitsThree(t *testing.T) {
	e := newExecution(t, nil, edit(0, "a"))
	e.fake.proposalError = "nonFoundResult: no trace"
	report := e.run(t)
	require.Equal(t, "minimized", report.Reduction.Status)
	require.Equal(t, ProposalNotCompiled, report.Proposal.Status)
	require.Equal(t, "nonFoundResult: no trace", report.Proposal.Error)
	require.Equal(t, ExitToolingFailure, report.ExitCode())
}

// A report over its cap turns an exit 0 into 2 and leaves every other exit as it was.
func TestExitCodeWithinTheReportCap(t *testing.T) {
	report := newExecution(t, nil, edit(0, "a")).run(t)
	require.Equal(t, ExitReproduced, report.ExitCode())
	rendered, err := report.Render()
	require.NoError(t, err)
	report.Limits.ReportBytes = int64(len(rendered))
	require.False(t, report.OverCap(len(rendered)))
	require.Equal(t, ExitReproduced, report.ExitCodeWithin(len(rendered)))
	report.Limits.ReportBytes = int64(len(rendered) - 1)
	require.True(t, report.OverCap(len(rendered)))
	require.Equal(t, ExitIndeterminate, report.ExitCodeWithin(len(rendered)))
	rejected := Report{Admission: AdmissionReport{Status: StatusRejected}, Limits: LimitsReport{ReportBytes: 1}}
	require.Equal(t, ExitToolingFailure, rejected.ExitCodeWithin(10))
}
