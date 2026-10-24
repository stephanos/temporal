//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/canary/testharness"
	"umpire.local/model/runner"
	"umpire.local/temporal"
)

// canaryPlantedAPIKey is set in every harness process's environment, which the harness transport
// never uses, so the scans can prove no credential reaches what the process writes.
const canaryPlantedAPIKey = "canary-planted-api-key-3e8b51"

// canaryCleanupTimeout bounds removing a harness's namespace resources.
const canaryCleanupTimeout = 5 * time.Second

// canaryHarness is one test cluster namespace the canary's harness build runs against as a
// separate process: its coordinates, its test policy and the harness binary.
type canaryHarness struct {
	env         *testcore.TestEnv
	binary      string
	coordinates authority.Coordinates
	policy      *policy.Policy
	policyPath  string
	jobs        int
}

// canaryJob is one workflow dispatch: its Actions run, its output directory and its recovery
// record, as the protected workflow gives each job its own.
type canaryJob struct {
	runID    string
	output   string
	recovery string
}

func newCanaryHarness(t *testing.T, name string, edit func(*policy.Policy)) *canaryHarness {
	t.Helper()
	env := activityEnvironment(t)
	coordinates := authority.Coordinates{
		GRPC: env.FrontendGRPCAddress(), Namespace: "umpire-" + name, TaskQueue: "umpire-" + name + "-queue",
	}
	release, err := provision.Create(t.Context(), provision.Clients{Workflow: env.FrontendClient(), Operator: env.OperatorClient()},
		provision.Resources{Namespace: coordinates.Namespace, RetainNamespace: true})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), canaryCleanupTimeout)
		defer cancel()
		require.NoError(t, release(ctx))
	})

	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.Coordinates = coordinates.Digests()
	canary.AuthorityClass = policy.AuthorityHarness
	if edit != nil {
		edit(canary)
	}
	encoded, err := json.MarshalIndent(canary, "", "  ")
	require.NoError(t, err)
	policyPath := filepath.Join(t.TempDir(), "policy.json")
	require.NoError(t, os.WriteFile(policyPath, append(encoded, '\n'), 0o600))

	binary := filepath.Join(t.TempDir(), "umpire-canary")
	build := exec.Command("go", "build", "-tags", "canary_harness", "-o", binary, "go.temporal.io/server/tools/canary/cmd/umpire-canary")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build the harness: %s", output)
	return &canaryHarness{env: env, binary: binary, coordinates: coordinates, policy: canary, policyPath: policyPath}
}

func (h *canaryHarness) job(t *testing.T) canaryJob {
	t.Helper()
	h.jobs++
	output := filepath.Join(t.TempDir(), "canary-output")
	require.NoError(t, os.Mkdir(output, 0o755))
	return canaryJob{runID: strconv.Itoa(1000 + h.jobs), output: output, recovery: filepath.Join(t.TempDir(), "recovery.json")}
}

// environment is everything the process sees: the coordinates, the job's workflow context, the
// test policy and any hook, and no credential.
func (h *canaryHarness) environment(job canaryJob, extra map[string]string) []string {
	values := map[string]string{
		"PATH": os.Getenv("PATH"), "HOME": os.Getenv("HOME"),
		authority.VariableGRPC: h.coordinates.GRPC, authority.VariableNamespace: h.coordinates.Namespace,
		authority.VariableTaskQueue:   h.coordinates.TaskQueue,
		preflight.VariableEventName:   "workflow_dispatch",
		preflight.VariableRepository:  h.policy.Repository,
		preflight.VariableRef:         h.policy.TrustedRef,
		preflight.VariableWorkflowRef: h.policy.Repository + "/" + h.policy.WorkflowPath + "@" + h.policy.TrustedRef,
		preflight.VariableRunID:       job.runID,
		preflight.VariableRunAttempt:  "1",
		testharness.VariablePolicy:    h.policyPath,
		authority.VariableAPIKey:      canaryPlantedAPIKey,
	}
	for key, value := range extra {
		values[key] = value
	}
	var environment []string
	for key, value := range values {
		environment = append(environment, key+"="+value)
	}
	return environment
}

// invoke runs one mode to its end, scans everything it wrote for the raw coordinates, and returns
// its exit code and stdout.
func (h *canaryHarness) invoke(t *testing.T, job canaryJob, mode string, extra map[string]string) (int, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, h.binary, mode, "--output", job.output, "--recovery", job.recovery)
	command.Env = h.environment(job, extra)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	require.NotNil(t, command.ProcessState, "umpire-canary %s did not start: %v", mode, err)
	t.Logf("umpire-canary %s: exit %d\n%s%s", mode, command.ProcessState.ExitCode(), stdout.String(), stderr.String())
	host, _, _ := strings.Cut(h.coordinates.GRPC, ":")
	for _, raw := range []string{h.coordinates.GRPC, host, h.coordinates.Namespace, h.coordinates.TaskQueue, canaryPlantedAPIKey} {
		require.NotContains(t, stdout.String()+stderr.String(), raw, "a coordinate or credential reached the process's output")
	}
	// A published Run keeps its World's namespace and task queue, but never the gRPC target or a
	// credential.
	require.NoError(t, filepath.WalkDir(job.output, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		written, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, raw := range []string{h.coordinates.GRPC, host, canaryPlantedAPIKey} {
			require.NotContains(t, string(written), raw, "the gRPC target or a credential reached %s", path)
		}
		return nil
	}))
	return command.ProcessState.ExitCode(), stdout.Bytes()
}

func (h *canaryHarness) run(t *testing.T, job canaryJob, extra map[string]string) (int, controller.Summary) {
	t.Helper()
	code, stdout := h.invoke(t, job, "run", extra)
	var summary controller.Summary
	if code != testharness.CrashExit {
		require.NoError(t, json.Unmarshal(stdout, &summary), "%s", stdout)
	}
	return code, summary
}

func (h *canaryHarness) reconcile(t *testing.T, job canaryJob) (int, controller.Report) {
	t.Helper()
	code, stdout := h.invoke(t, job, "reconcile", nil)
	var report controller.Report
	require.NoError(t, json.Unmarshal(stdout, &report), "%s", stdout)
	return code, report
}

// leaseClose is the lease's latest run's close reason, or "" while it is open.
func (h *canaryHarness) leaseClose(t *testing.T) (string, enumspb.WorkflowExecutionStatus) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	described, err := h.env.FrontendClient().DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: h.coordinates.Namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: h.policy.Lease.WorkflowID},
	})
	require.NoError(t, err)
	info := described.GetWorkflowExecutionInfo()
	if info.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return "", info.GetStatus()
	}
	closing, err := h.env.FrontendClient().GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: h.coordinates.Namespace, Execution: info.GetExecution(), HistoryEventFilterType: enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
	})
	require.NoError(t, err)
	events := closing.GetHistory().GetEvents()
	require.NotEmpty(t, events)
	return events[len(events)-1].GetWorkflowExecutionTerminatedEventAttributes().GetReason(), info.GetStatus()
}

// activityStatus is a fenced activity's status as the server reports it; one fenced but never
// started is unspecified.
func (h *canaryHarness) activityStatus(t *testing.T, id string) enumspb.ActivityExecutionStatus {
	t.Helper()
	described, err := h.env.FrontendClient().DescribeActivityExecution(t.Context(), &workflowservice.DescribeActivityExecutionRequest{
		Namespace: h.coordinates.Namespace, ActivityId: id,
	})
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return enumspb.ACTIVITY_EXECUTION_STATUS_UNSPECIFIED
	}
	require.NoError(t, err)
	return described.GetInfo().GetStatus()
}

// publishedRuns are the Runs under a job's output directory, by key.
func publishedRuns(t *testing.T, job canaryJob) map[string]*runner.Run {
	t.Helper()
	runs := map[string]*runner.Run{}
	require.NoError(t, filepath.WalkDir(job.output, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		key, err := filepath.Rel(job.output, path)
		require.NoError(t, err)
		run, err := runner.Load(path)
		require.NoError(t, err, "the output holds only Runs: %s", key)
		runs[key] = run
		return nil
	}))
	return runs
}

// The canary end to end against the test cluster, as the protected workflow runs it: the harness
// binary as a separate process with no credential. Preflight passes, the lease is taken, two
// serial Runs of the committed activity Model are explained, each on a fresh fenced activity, and
// are published with runner.Publish after cleanup, which releases the lease; each published Run
// replays offline; the job's reconcile then finds nothing lost.
func TestTestpilotCanaryHarnessEndToEnd(t *testing.T) {
	h := newCanaryHarness(t, "canary-e2e", nil)
	job := h.job(t)
	code, summary := h.run(t, job, nil)
	require.Equal(t, controller.ExitAccepted, code, "%+v", summary)
	require.Equal(t, controller.StatusExplained, summary.Status)
	require.Len(t, summary.Iterations, 2)
	require.NotEqual(t, summary.Iterations[0].ActivityID, summary.Iterations[1].ActivityID, "each Run has its own fenced activity")
	require.Equal(t, controller.CleanupReleased, summary.Cleanup.Outcome)
	require.Equal(t, []string{summary.Iterations[0].ActivityID, summary.Iterations[1].ActivityID}, summary.Cleanup.Fenced)

	runs := publishedRuns(t, job)
	require.Len(t, runs, 2, "the Runs are the only artifact")
	guest := temporal.Guest()
	for _, iteration := range summary.Iterations {
		require.Equal(t, controller.StatusExplained, iteration.Status)
		run, ok := runs[iteration.Run]
		require.True(t, ok, "the summary names the published Run's key")
		require.Equal(t, h.policy.ModelHash, run.GetModelHash())
		require.NotNil(t, run.GetVerdict().GetExplained())
		require.Equal(t, "sdk", run.GetWorker().GetKind())
		require.Equal(t, h.coordinates.Namespace, run.GetWorker().GetNamespace())
		require.Equal(t, h.coordinates.TaskQueue, run.GetWorker().GetTaskQueue())
		replayed, err := runner.New(guest, nil).Run(t.Context(), runner.ModeReplay, runner.Options{Recorded: run})
		require.NoError(t, err)
		require.NotNil(t, replayed.GetVerdict().GetExplained(), "the published Run replays offline")
		require.NotEqual(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, h.activityStatus(t, iteration.ActivityID), "the fenced activity is closed")
	}
	reason, _ := h.leaseClose(t)
	require.Equal(t, controller.ReasonReleased, reason)

	code, report := h.reconcile(t, job)
	require.Equal(t, controller.ExitAccepted, code, "%+v", report)
	require.Equal(t, controller.StatusReconciled, report.Status)
	require.Empty(t, report.Lost)
	require.Empty(t, report.Terminated)
}

// A process lost at each phase is recovered by its own job's reconcile, which closes exactly what
// it fenced and reports the lost iterations, publishing nothing. Lost with the lease held -- before
// its first Run, during a Run, after a Run closed -- the next dispatch refuses until that
// reconcile; lost after cleanup released the lease and before publication, the next dispatch
// proceeds and the lost job's reconcile still names both unpublished iterations lost, leaving the
// later invocation's lease run alone. A dispatch after it proceeds.
func TestTestpilotCanaryHarnessRecoversALostProcess(t *testing.T) {
	for _, test := range []struct {
		phase string
		// held is whether the lost process leaves the lease held.
		held bool
		// fenced is how many activities the lost process fenced, each lost.
		fenced int
	}{
		{phase: controller.PhaseLeased, held: true},
		{phase: controller.PhaseRunOpened, held: true, fenced: 1},
		{phase: controller.PhaseIterationClosed, held: true, fenced: 1},
		{phase: controller.PhaseCleaned, fenced: 2},
	} {
		t.Run(test.phase, func(t *testing.T) {
			h := newCanaryHarness(t, "canary-lost-"+strings.ReplaceAll(test.phase, "-", ""), nil)
			lost := h.job(t)
			code, _ := h.run(t, lost, map[string]string{testharness.VariableCrash: test.phase})
			require.Equal(t, testharness.CrashExit, code)
			record, err := recovery.Read(lost.recovery)
			require.NoError(t, err)
			require.Len(t, record.Iterations, test.fenced)
			fenced := []string{}
			for _, iteration := range record.Iterations {
				require.False(t, iteration.Published)
				fenced = append(fenced, iteration.ActivityID)
			}

			next := h.job(t)
			code, summary := h.run(t, next, nil)
			if test.held {
				_, status := h.leaseClose(t)
				require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, status, "the lost process's lease stays held")
				require.Equal(t, controller.ExitUncertain, code, "%+v", summary)
				require.Equal(t, controller.StatusLeaseUnreconciled, summary.Status)
				require.Empty(t, publishedRuns(t, next))
			} else {
				require.Equal(t, controller.ExitAccepted, code, "a released lease is a clean scope: %+v", summary)
				require.Len(t, publishedRuns(t, next), 2)
			}

			code, report := h.reconcile(t, lost)
			require.Equal(t, controller.ExitAccepted, code, "%+v", report)
			require.Equal(t, controller.StatusReconciled, report.Status)
			require.Equal(t, fenced, report.Fenced)
			require.Equal(t, fenced, report.Closed)
			require.Equal(t, fenced, report.Lost, "each lost iteration is named")
			require.Empty(t, publishedRuns(t, lost), "no Run is fabricated")
			for _, id := range fenced {
				require.NotEqual(t, enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, h.activityStatus(t, id))
			}
			reason, _ := h.leaseClose(t)
			if test.held {
				require.Equal(t, controller.ReasonReconciled, reason)
			} else {
				require.Equal(t, controller.ReasonReleased, reason, "the later invocation released its own lease run")
			}

			after := h.job(t)
			code, summary = h.run(t, after, nil)
			require.Equal(t, controller.ExitAccepted, code, "%+v", summary)
			require.Len(t, summary.Iterations, 2)
		})
	}
}

// The harness refuses a policy that names production's authority class, before anything else, and
// names what it refused.
func TestTestpilotCanaryHarnessRefusesAProductionPolicy(t *testing.T) {
	h := newCanaryHarness(t, "canary-production", func(p *policy.Policy) { p.AuthorityClass = policy.AuthorityProtectedWorkflow })
	job := h.job(t)
	code, summary := h.run(t, job, nil)
	require.Equal(t, controller.ExitFailed, code)
	require.Equal(t, controller.StatusPolicyUnavailable, summary.Status)
	require.Contains(t, summary.Detail, strconv.Quote(policy.AuthorityProtectedWorkflow))
	require.Empty(t, summary.Iterations)
	_, err := os.Stat(job.recovery)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Empty(t, publishedRuns(t, job))
}
