//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
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
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/canary/testharness"
	"google.golang.org/protobuf/types/known/durationpb"
)

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
	env := newTestpilotTestEnvironment(t)
	coordinates := authority.Coordinates{
		GRPC: env.FrontendGRPCAddress(), Namespace: "umpire-" + name, TaskQueue: "umpire-" + name + "-queue",
		HandlerQueue: "umpire-" + name + "-handler-queue", NexusEndpoint: "umpire-" + name + "-endpoint",
	}
	release, err := provision.Create(env.Context(), provision.Clients{Workflow: env.FrontendClient(), Operator: env.OperatorClient()}, provision.Resources{
		Namespace: coordinates.Namespace, TaskQueue: coordinates.TaskQueue, NexusEndpoint: coordinates.NexusEndpoint,
		NexusTaskQueue: coordinates.HandlerQueue, RetainNamespace: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testpilotCleanupTimeout)
		defer cancel()
		require.NoError(t, release(ctx))
	})

	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.Coordinates = coordinates.Digests()
	canary.EvaluationProfile = testharness.ProfileName
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
	build.Env = os.Environ()
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
		authority.VariableTaskQueue: h.coordinates.TaskQueue, authority.VariableHandlerQueue: h.coordinates.HandlerQueue,
		authority.VariableEndpoint:    h.coordinates.NexusEndpoint,
		preflight.VariableEventName:   "workflow_dispatch",
		preflight.VariableRepository:  h.policy.Repository,
		preflight.VariableRef:         h.policy.TrustedRef,
		preflight.VariableWorkflowRef: h.policy.Repository + "/" + h.policy.WorkflowPath + "@" + h.policy.TrustedRef,
		preflight.VariableRunID:       job.runID,
		preflight.VariableRunAttempt:  "1",
		testharness.VariablePolicy:    h.policyPath,
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

// command is one mode of one job as its own process.
func (h *canaryHarness) command(ctx context.Context, job canaryJob, mode string, extra map[string]string) (*exec.Cmd, *bytes.Buffer, *bytes.Buffer) {
	command := exec.CommandContext(ctx, h.binary, mode, "--output", job.output, "--recovery", job.recovery)
	command.Env = h.environment(job, extra)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	return command, &stdout, &stderr
}

// invoke runs one mode to its end, scans everything it wrote for the raw coordinates, and returns
// its exit code and stdout.
func (h *canaryHarness) invoke(t *testing.T, job canaryJob, mode string, extra map[string]string) (int, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command, stdout, stderr := h.command(ctx, job, mode, extra)
	err := command.Run()
	require.NotNil(t, command.ProcessState, "umpire-canary %s did not start: %v", mode, err)
	h.requireNoCoordinate(t, stdout.String()+stderr.String())
	return command.ProcessState.ExitCode(), stdout.Bytes()
}

func (h *canaryHarness) requireNoCoordinate(t *testing.T, written string) {
	t.Helper()
	host, _, _ := strings.Cut(h.coordinates.GRPC, ":")
	for _, raw := range []string{h.coordinates.GRPC, host, h.coordinates.Namespace, h.coordinates.TaskQueue, h.coordinates.HandlerQueue, h.coordinates.NexusEndpoint} {
		require.NotContains(t, written, raw, "a planted coordinate reached the process's output")
	}
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
	described, err := h.env.FrontendClient().DescribeWorkflowExecution(h.env.Context(), &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: h.coordinates.Namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: h.policy.Lease.WorkflowID},
	})
	require.NoError(t, err)
	info := described.GetWorkflowExecutionInfo()
	if info.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return "", info.GetStatus()
	}
	closing, err := h.env.FrontendClient().GetWorkflowExecutionHistory(h.env.Context(), &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: h.coordinates.Namespace, Execution: info.GetExecution(), HistoryEventFilterType: enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
	})
	require.NoError(t, err)
	events := closing.GetHistory().GetEvents()
	return events[len(events)-1].GetWorkflowExecutionTerminatedEventAttributes().GetReason(), info.GetStatus()
}

func published(t *testing.T, job canaryJob) []string {
	t.Helper()
	entries, err := os.ReadDir(job.output)
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// The canary end to end against the test cluster, as the protected workflow runs it: the harness
// binary as a separate process with no credential. Preflight passes, the lease is taken, the Case
// is prepared once, two serial Runs close satisfied with fresh fenced identities, each is admitted,
// assessed accepted under canary-harness and published with its provenance after cleanup, which
// releases the lease; the job's reconcile then finds nothing lost.
func TestTestpilotCanaryHarnessEndToEnd(t *testing.T) {
	h := newCanaryHarness(t, "canary-e2e", nil)
	job := h.job(t)
	code, summary := h.run(t, job, nil)
	require.Equal(t, controller.ExitAccepted, code, "%+v", summary)
	require.Equal(t, controller.StatusAccepted, summary.Status)
	require.Len(t, summary.Iterations, 2)
	require.NotEqual(t, summary.Iterations[0].RunID, summary.Iterations[1].RunID, "each Run has its own fenced identity")
	require.Equal(t, assessment.CleanupReleased, summary.Cleanup.Outcome)
	require.Equal(t, []string{summary.Iterations[0].RunID, summary.Iterations[1].RunID}, summary.Cleanup.Fenced)
	require.Len(t, published(t, job), 4)
	for index, iteration := range summary.Iterations {
		require.Equal(t, controller.StatusAccepted, iteration.Status)
		encoded, err := os.ReadFile(filepath.Join(job.output, iteration.Provenance+".provenance.json"))
		require.NoError(t, err)
		provenance, err := assessment.DecodeProvenance(encoded)
		require.NoError(t, err)
		require.Equal(t, policy.AuthorityHarness, provenance.AuthorityClass, "a harness receipt is never a production one")
		require.Equal(t, index+1, provenance.Invocation.Iteration)
		require.Equal(t, iteration.Receipt, provenance.Receipt)
		receipt, err := os.ReadFile(filepath.Join(job.output, iteration.Receipt+".json"))
		require.NoError(t, err)
		require.Contains(t, string(receipt), `"name":"canary-harness"`)
		require.Contains(t, string(receipt), `"trust":"test-cluster-harness"`)
	}
	reason, _ := h.leaseClose(t)
	require.Equal(t, controller.ReasonReleased, reason)

	code, report := h.reconcile(t, job)
	require.Equal(t, controller.ExitAccepted, code, "%+v", report)
	require.Equal(t, controller.StatusReconciled, report.Status)
	require.Empty(t, report.Lost)
	require.Empty(t, report.Terminated)
}

// A process lost before its first Run and one lost during a Run hold the lease: the next dispatch
// refuses on it, the job's own reconcile closes exactly what it fenced and reports the lost
// iteration with no receipt, provenance or Verdict, and a dispatch after it proceeds.
func TestTestpilotCanaryHarnessRecoversALostProcess(t *testing.T) {
	for _, phase := range []string{controller.PhaseLeased, controller.PhaseRunOpened} {
		t.Run(phase, func(t *testing.T) {
			h := newCanaryHarness(t, "canary-lost-"+strings.ReplaceAll(phase, "-", ""), nil)
			lost := h.job(t)
			code, _ := h.run(t, lost, map[string]string{testharness.VariableCrash: phase})
			require.Equal(t, testharness.CrashExit, code)
			_, status := h.leaseClose(t)
			require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, status, "the lost process's lease stays held")

			refused := h.job(t)
			code, summary := h.run(t, refused, nil)
			require.Equal(t, controller.ExitUncertain, code, "%+v", summary)
			require.Equal(t, controller.StatusLeaseUnreconciled, summary.Status)
			require.Empty(t, published(t, refused))

			code, report := h.reconcile(t, lost)
			require.Equal(t, controller.ExitAccepted, code, "%+v", report)
			require.Equal(t, controller.StatusReconciled, report.Status)
			record, err := recovery.Read(lost.recovery)
			require.NoError(t, err)
			if phase == controller.PhaseRunOpened {
				require.Len(t, record.Iterations, 1)
				require.Equal(t, []string{record.Iterations[0].RunID}, report.Fenced)
				require.Equal(t, []string{record.Iterations[0].RunID}, report.Lost, "the lost iteration is named, with no receipt")
			} else {
				require.Empty(t, report.Fenced)
				require.Empty(t, report.Lost)
			}
			require.Empty(t, published(t, lost), "no receipt, provenance or Verdict is fabricated")
			reason, _ := h.leaseClose(t)
			require.Equal(t, controller.ReasonReconciled, reason)

			after := h.job(t)
			code, summary = h.run(t, after, nil)
			require.Equal(t, controller.ExitAccepted, code, "%+v", summary)
			require.Len(t, summary.Iterations, 2)
		})
	}
}

// A second dispatch and its reconcile during a live first invocation touch nothing of it: the run
// refuses on the held lease, the reconcile leaves a lease younger than an invocation alone, and the
// first invocation completes.
func TestTestpilotCanaryHarnessLeavesALiveInvocationAlone(t *testing.T) {
	h := newCanaryHarness(t, "canary-live", nil)
	first := h.job(t)
	resume := filepath.Join(t.TempDir(), "resume")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	command, stdout, stderr := h.command(ctx, first, "run", map[string]string{testharness.VariablePause: controller.PhaseLeased + ":" + resume})
	require.NoError(t, command.Start())
	require.Eventually(t, func() bool {
		record, err := recovery.Read(first.recovery)
		return err == nil && record.Lease != nil && record.Phase == recovery.PhaseRunning
	}, time.Minute, 50*time.Millisecond, "the first invocation holds the lease")

	second := h.job(t)
	code, summary := h.run(t, second, nil)
	require.Equal(t, controller.ExitUncertain, code)
	require.Equal(t, controller.StatusLeaseUnreconciled, summary.Status)
	code, report := h.reconcile(t, second)
	require.Equal(t, controller.ExitUncertain, code, "%+v", report)
	require.Equal(t, controller.StatusLeaseInUse, report.Status)
	_, status := h.leaseClose(t)
	require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, status)

	require.NoError(t, os.WriteFile(resume, nil, 0o600))
	require.NoError(t, command.Wait(), "%s", stderr.String())
	h.requireNoCoordinate(t, stdout.String()+stderr.String())
	var completed controller.Summary
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &completed))
	require.Equal(t, controller.StatusAccepted, completed.Status)
	require.Len(t, published(t, first), 4)
}

// A lease left to time out is refused by the next dispatch until that job's reconcile records the
// scope reconciled; the dispatch after it proceeds.
func TestTestpilotCanaryHarnessRecoversATimedOutLease(t *testing.T) {
	h := newCanaryHarness(t, "canary-timeout", nil)
	_, err := h.env.FrontendClient().StartWorkflowExecution(h.env.Context(), &workflowservice.StartWorkflowExecutionRequest{
		Namespace: h.coordinates.Namespace, WorkflowId: h.policy.Lease.WorkflowID,
		WorkflowType:       &commonpb.WorkflowType{Name: h.policy.Lease.WorkflowType},
		TaskQueue:          &taskqueuepb.TaskQueue{Name: h.policy.Lease.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		WorkflowRunTimeout: durationpb.New(time.Second), RequestId: "a lease left to time out",
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, status := h.leaseClose(t)
		return status == enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT
	}, time.Minute, 100*time.Millisecond)

	refused := h.job(t)
	code, summary := h.run(t, refused, nil)
	require.Equal(t, controller.ExitUncertain, code, "%+v", summary)
	require.Equal(t, controller.StatusLeaseUnreconciled, summary.Status)
	code, report := h.reconcile(t, refused)
	require.Equal(t, controller.ExitAccepted, code, "%+v", report)
	require.Equal(t, controller.StatusReconciled, report.Status)
	reason, _ := h.leaseClose(t)
	require.Equal(t, controller.ReasonReconciled, reason)

	after := h.job(t)
	code, summary = h.run(t, after, nil)
	require.Equal(t, controller.ExitAccepted, code, "%+v", summary)
}

// The harness refuses a policy that names production's Evaluation Profile, before anything else.
func TestTestpilotCanaryHarnessRefusesAProductionPolicy(t *testing.T) {
	h := newCanaryHarness(t, "canary-production", func(p *policy.Policy) { p.EvaluationProfile = "production-canary" })
	job := h.job(t)
	code, summary := h.run(t, job, nil)
	require.Equal(t, controller.ExitFailed, code)
	require.Equal(t, controller.StatusPolicyUnavailable, summary.Status)
	_, err := os.Stat(job.recovery)
	require.ErrorIs(t, err, os.ErrNotExist)
}
