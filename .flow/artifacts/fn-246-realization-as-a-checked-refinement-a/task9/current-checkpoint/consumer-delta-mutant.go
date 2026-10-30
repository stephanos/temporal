//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/chasm/lib/activity"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/common/testing/testpilot/temporal/act"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	"umpire.local/model/runner"
	"umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// umpireSlice runs the umpire command as a separate process against one cluster.
type umpireSlice struct {
	t *testing.T
	// root owns the stores, so a later subtest can replay an earlier subtest's Runs.
	root    *testing.T
	binary  string
	guest   string
	address string
}

// umpireInvocation is one finished command: its exit code, output and the Runs it published.
type umpireInvocation struct {
	code           int
	stdout, stderr string
	runs           map[string]*runner.Run
}

func (s *umpireSlice) world(namespace, queue, input string, attempts int32) string {
	t := s.t
	t.Helper()
	policy := &commonpb.RetryPolicy{MaximumAttempts: attempts}
	if attempts > 1 {
		policy.InitialInterval, policy.MaximumInterval = durationpb.New(time.Second), durationpb.New(time.Second)
	}
	start, err := anypb.New(&workflowservice.StartActivityExecutionRequest{
		Namespace: namespace, ActivityId: "umpire-slice-" + uuid.NewString(),
		ActivityType: &commonpb.ActivityType{Name: "ModelSlice"}, TaskQueue: &taskqueuepb.TaskQueue{Name: queue},
		StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: policy,
		IdReusePolicy: enumspb.ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		Input: &commonpb.Payloads{Payloads: []*commonpb.Payload{{
			Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte(input),
		}}},
	})
	require.NoError(t, err)
	data, err := proto.Marshal(start)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "world.binpb")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

// run invokes the command in an empty working directory with only the given Driver
// environment, and requires that the published Runs are the only thing it wrote.
func (s *umpireSlice) run(driverEnv map[string]string, args ...string) umpireInvocation {
	return s.runObserved(driverEnv, nil, args...)
}

func (s *umpireSlice) runObserved(driverEnv map[string]string, observe func(<-chan struct{}) error, args ...string) umpireInvocation {
	t := s.t
	t.Helper()
	workdir, store := t.TempDir(), s.root.TempDir()
	args = append(append([]string(nil), args...), "--model", s.guest)
	if args[0] != "replay" {
		args = append(args, "--store", store)
	}
	command := exec.Command(s.binary, args...)
	command.Dir = workdir
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "UMPIRE_TEMPORAL_") {
			command.Env = append(command.Env, variable)
		}
	}
	for name, value := range driverEnv {
		command.Env = append(command.Env, name+"="+value)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	require.NoError(t, command.Start())
	finished := make(chan struct{})
	var observed chan error
	if observe != nil {
		observed = make(chan error, 1)
		go func() { observed <- observe(finished) }()
	}
	err := command.Wait()
	close(finished)
	if observed != nil {
		require.NoError(t, <-observed, "namespace must be served while the CLI is running")
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		require.NoError(t, err)
	}
	result := umpireInvocation{code: command.ProcessState.ExitCode(), stdout: stdout.String(), stderr: stderr.String(), runs: map[string]*runner.Run{}}
	left, err := os.ReadDir(workdir)
	require.NoError(t, err)
	require.Empty(t, left, "the command writes nothing but Runs")
	require.NoError(t, filepath.WalkDir(store, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		require.True(t, entry.Type().IsRegular(), path)
		require.Equal(t, ".binpb", filepath.Ext(path), "the store holds only Runs: %s", path)
		key, err := filepath.Rel(store, path)
		require.NoError(t, err)
		require.Contains(t, result.stdout, "published "+key)
		recorded, err := runner.Load(path)
		require.NoError(t, err)
		result.runs[path] = recorded
		return nil
	}))
	t.Logf("umpire %s: exit %d\n%s%s", args[0], result.code, result.stdout, result.stderr)
	return result
}

func (s *umpireSlice) driverEnv(namespace, queue string) map[string]string {
	return map[string]string{act.EnvAddress: s.address, act.EnvNamespace: namespace, act.EnvTaskQueue: queue}
}

func explainedSteps(run *runner.Run) []string {
	var steps []string
	for _, event := range run.GetEvents() {
		explanation := event.GetJudgment().GetResponse().GetExplained()
		for i, step := range explanation.GetSteps() {
			steps = append(steps, step+" "+explanation.GetOutcomes()[i])
		}
	}
	return steps
}

func TestUmpireWholePrefixJudgmentPath(t *testing.T) {
	judgment := func(steps ...string) *runv1.Event {
		outcomes := make([]string, len(steps))
		for i := range outcomes {
			outcomes[i] = "Accepted"
		}
		return &runv1.Event{Kind: &runv1.Event_Judgment{Judgment: &runv1.Judgment{
			Response: &modelv1.JudgeResponse{Judgment: &modelv1.JudgeResponse_Explained{
				Explained: &modelv1.Explanation{Steps: steps, Outcomes: outcomes},
			}},
		}}}
	}
	raw := &runner.Run{Events: []*runv1.Event{
		judgment("Poll", "Fail"),
		judgment("Poll", "Fail", "Retry", "Complete"),
	}}
	sdk := proto.CloneOf(raw)
	sdk.Events = append([]*runv1.Event{judgment("Poll", "Fail")}, sdk.Events...)
	before := proto.CloneOf(sdk)
	require.Equal(t, []string{"Poll Accepted", "Fail Accepted", "Retry Accepted", "Complete Accepted"}, explainedSteps(raw))
	require.Equal(t, explainedSteps(raw), explainedSteps(sdk), "an extra unchanged observation repeats a complete witness")
	require.True(t, proto.Equal(before, sdk), "presentation preserves prior reports")
	sdk.Events[0] = judgment("Poll")
	require.Equal(t, explainedSteps(raw), explainedSteps(sdk), "an earlier partial witness does not change the final witness")

	different := &runner.Run{Events: []*runv1.Event{judgment("Poll", "Complete")}}
	require.NotEqual(t, explainedSteps(raw), explainedSteps(different), "different final witnesses remain different")
	repeated := &runner.Run{Events: []*runv1.Event{judgment("Client::Cancel", "Client::Cancel")}}
	repeated.Events[0].GetJudgment().GetResponse().GetExplained().Outcomes = []string{"Rejected(NotFound)", "Rejected(NotFound)"}
	require.Equal(t, []string{"Client::Cancel Rejected(NotFound)", "Client::Cancel Rejected(NotFound)"}, explainedSteps(repeated), "actual repeated edges remain in the witness")
	require.Empty(t, explainedSteps(&runner.Run{Events: []*runv1.Event{judgment()}}))
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return lines[len(lines)-1]
}

func requireAllExplained(t *testing.T, invocation umpireInvocation) {
	t.Helper()
	for _, run := range invocation.runs {
		if run.GetVerdict().GetUnexplained() != nil || run.GetVerdict().GetRefused() != nil {
			data, err := protojson.Marshal(run)
			require.NoError(t, err)
			t.Logf("deviation run=%s", data)
			for _, event := range run.Events {
				if event.GetProposal() != nil || event.GetSequence()+3 >= run.GetVerdict().GetUnexplained().GetEvent() {
					t.Logf("deviation event=%v", event)
				}
			}
		}
	}
	require.Equal(t, runner.ExitExplained, invocation.code, "stdout: %s\nstderr: %s", invocation.stdout, invocation.stderr)
	for path, run := range invocation.runs {
		require.NotNil(t, run.GetVerdict().GetExplained(), "%s: %v", path, run.GetVerdict())
	}
}

const unheldCancelScopeMessage = "unheld cancellation of an open activity is outside the atomic cancellation Model"

func isUnheldCancelScopeRefusal(run *runner.Run) bool {
	refused := run.GetVerdict().GetRefused()
	events := run.GetEvents()
	if refused == nil || len(events) < 4 || refused.GetEvent() != uint64(len(events)-1) {
		return false
	}
	event := events[refused.GetEvent()]
	refusal := event.GetRefusal()
	if event.GetSequence() != refused.GetEvent() || event.GetDriverEvent() != 0 ||
		refusal.GetMethod() != modelv1.Method_METHOD_PROPOSE ||
		refusal.GetRefusal().GetCode() != modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED ||
		refusal.GetRefusal().GetMessage() != unheldCancelScopeMessage ||
		refused.GetReason() != "model refused METHOD_PROPOSE: REFUSAL_CODE_UNSUPPORTED: "+unheldCancelScopeMessage {
		return false
	}
	request := new(modelv1.ProposeRequest)
	if proto.Unmarshal(refusal.GetRequest(), request) != nil || request.GetSeed()%2 != 0 ||
		request.GetSeed() != runner.ProposalSeed(run.GetSeed(), event.GetRequest()) {
		return false
	}
	var proposals uint64
	for _, previous := range events[:len(events)-1] {
		if previous.GetRefusal() != nil || previous.GetDriverFault() != nil || previous.GetAnswer() != nil {
			return false
		}
		if proposal := previous.GetProposal(); proposal != nil {
			proposals++
			if proposal.GetResponse().GetAction() == "Client::Cancel" {
				return false
			}
		}
		if judgment := previous.GetJudgment(); judgment != nil {
			if judgment.GetResponse().GetExplained() == nil {
				return false
			}
			observation := new(realizationv1.Observation)
			if judgment.GetRequest().GetObservation().UnmarshalTo(observation) != nil ||
				observation.GetMethod() == workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName ||
				observation.GetRequest().MessageIs(new(workflowservice.RequestCancelActivityExecutionRequest)) {
				return false
			}
		}
	}
	return event.GetRequest() == proposals+1 && event.GetRequest() >= 2
}

func requireSupportedActivityRuns(t *testing.T, invocation umpireInvocation) {
	t.Helper()
	explained := umpireInvocation{code: runner.ExitExplained, runs: map[string]*runner.Run{}}
	expectedCode := runner.ExitExplained
	for path, run := range invocation.runs {
		if run.GetVerdict().GetRefused() == nil {
			explained.runs[path] = run
		} else {
			require.True(t, isUnheldCancelScopeRefusal(run), "%s: unexpected refusal: %v", path, run.GetVerdict())
			expectedCode = runner.ExitRefused
		}
	}
	require.Equal(t, expectedCode, invocation.code, "stdout: %s\nstderr: %s", invocation.stdout, invocation.stderr)
	require.NotEmpty(t, explained.runs, "the mixed exploration must exercise supported Runs")
	requireAllExplained(t, explained)
}

func (s *umpireSlice) program(world string) string {
	t := s.t
	t.Helper()
	output := filepath.Join(t.TempDir(), "program.binpb")
	command := exec.Command(s.binary, "program", "--model", s.guest, "--world", world, "--out", output)
	data, err := command.CombinedOutput()
	require.NoError(t, err, "%s", data)
	return output
}

func (s *umpireSlice) mutateProgram(path, mutation string) string {
	t := s.t
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	envelope := new(anypb.Any)
	require.NoError(t, proto.Unmarshal(data, envelope))
	program := new(temporalv1.WorkerProgram)
	require.NoError(t, envelope.UnmarshalTo(program))
	if mutation == "empty" {
		program.Rules = nil
	} else {
		var completion *anypb.Any
		for _, rule := range program.GetRules() {
			key := new(temporalv1.ActivityTaskKey)
			require.NoError(t, rule.GetKey().UnmarshalTo(key))
			if key.GetScenario() == "Complete" {
				completion = proto.CloneOf(rule.GetRespond()[0])
				break
			}
		}
		require.NotNil(t, completion)
		changed := false
		for _, rule := range program.GetRules() {
			key := new(temporalv1.ActivityTaskKey)
			require.NoError(t, rule.GetKey().UnmarshalTo(key))
			switch {
			case mutation == "complete-instead-of-fail" && key.GetScenario() == "Fail":
				rule.Respond = []*anypb.Any{proto.CloneOf(completion)}
				changed = true
			case mutation == "wrong-result" && key.GetScenario() == "Complete":
				response := new(workflowservice.RespondActivityTaskCompletedRequest)
				require.NoError(t, completion.UnmarshalTo(response))
				response.Result.Payloads[0].Data = []byte("Different")
				rule.Respond = []*anypb.Any{modelSliceAny(t, response)}
				changed = true
			}
		}
		require.True(t, changed, "the control must change a response row")
	}
	data, err = proto.Marshal(modelSliceAny(t, program))
	require.NoError(t, err)
	output := filepath.Join(t.TempDir(), mutation+".binpb")
	require.NoError(t, os.WriteFile(output, data, 0o600))
	return output
}

func invocationRun(t *testing.T, invocation umpireInvocation, seed uint64) (string, *runner.Run) {
	t.Helper()
	for path, run := range invocation.runs {
		if run.GetSeed() == seed {
			return path, run
		}
	}
	t.Fatalf("seed %d was not recorded", seed)
	return "", nil
}

func requireWorkerRun(t *testing.T, run *runner.Run, kind, namespace, queue string) {
	t.Helper()
	require.Equal(t, kind, run.GetWorker().GetKind())
	require.NotEmpty(t, run.GetWorker().GetVersion())
	require.Equal(t, namespace, run.GetWorker().GetNamespace())
	require.Equal(t, queue, run.GetWorker().GetTaskQueue())
	require.NotNil(t, run.GetEvents()[0].GetReset_().GetResponse().GetProgram())
	start := new(workflowservice.StartActivityExecutionRequest)
	require.NoError(t, run.GetEvents()[1].GetProposal().GetResponse().GetRequest().UnmarshalTo(start))
	require.Equal(t, "application-input", string(start.GetInput().GetPayloads()[0].GetData()))
	require.Contains(t, []string{"Complete", "RetryThenComplete", "Fail"}, string(start.GetHeader().GetFields()["umpire-scenario"].GetData()))
	for _, event := range run.GetEvents() {
		require.Nil(t, event.GetAnswer(), "programmed workers never call Answer")
		if proposal := event.GetProposal().GetResponse(); proposal != nil {
			require.NotContains(t, proposal.GetAction(), "Worker::")
			if proposal.GetRequest().MessageIs(new(temporalv1.ObserveActivity)) {
				require.Equal(t, modelv1.ProposalKind_PROPOSAL_KIND_READ, proposal.GetKind())
				require.Empty(t, proposal.GetAction())
			}
		}
	}
}

func TestTestpilotModelSliceThroughUmpire(t *testing.T) {
	binary := buildCommand(t, "tools/umpire/cmd/umpire")
	env := newTestpilotTestEnvironment(t,
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(activity.Enabled, true),
		testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true),
		testcore.WithWorkerService("UMPIRE_TEMPORAL_CREATE deletes its namespace through the system delete-namespace workflow"),
		testcore.WithDynamicConfig(dynamicconfig.TransferProcessorUpdateAckInterval, time.Second),
		testcore.WithDynamicConfig(dynamicconfig.VisibilityProcessorUpdateAckInterval, time.Second),
	)
	guest, err := filepath.Abs(filepath.Join("..", "model", "temporal", "hosts", "go", "umpire_temporal_guest.wasm"))
	require.NoError(t, err)
	committed, err := os.ReadFile(guest)
	require.NoError(t, err)
	require.Equal(t, temporal.Guest(), committed)
	s := &umpireSlice{t: t, root: t, binary: binary, guest: guest, address: env.FrontendGRPCAddress()}
	namespace := env.Namespace().String()
	var raw umpireInvocation
	var replayPath string
	var replayRun *runner.Run
	for _, kind := range []string{"raw", "sdk"} {
		t.Run(kind, func(t *testing.T) {
			s.t = t
			queue := "umpire-slice-" + uuid.NewString()
			world := s.world(namespace, queue, "application-input", 2)
			driverEnv := s.driverEnv(namespace, queue)
			driverEnv[act.EnvExecutorKind], driverEnv[act.EnvProgram] = kind, s.program(world)
			invocation := s.run(driverEnv, "explore", "--world", world, "--seed", "21", "--budget", "7", "--event-timeout", "60s", "--run-timeout", "2m")
			requireSupportedActivityRuns(t, invocation)
			require.Len(t, invocation.runs, 21)
			for _, run := range invocation.runs {
				requireWorkerRun(t, run, kind, namespace, queue)
			}
			t.Run("UnheldCancelScope", func(t *testing.T) {
				_, run := invocationRun(t, invocation, 4)
				require.True(t, isUnheldCancelScopeRefusal(run), "verdict: %v", run.GetVerdict())
				require.Equal(t, uint64(2), run.GetEvents()[run.GetVerdict().GetRefused().GetEvent()].GetRequest())
				require.Empty(t, explainedSteps(run), "the refusal precedes the first open-activity action")
				for _, forged := range []struct {
					name   string
					change func(*runner.Run)
				}{
					{"reason", func(run *runner.Run) { run.Verdict.GetRefused().Reason += " forged" }},
					{"verdict", func(run *runner.Run) {
						run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Unexplained{Unexplained: &runv1.Unexplained{Event: 3}}}
					}},
					{"cancel-rpc", func(run *runner.Run) {
						run.Events[2].GetJudgment().Request.Observation = modelSliceAny(t, &realizationv1.Observation{
							Method:  workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName,
							Request: modelSliceAny(t, new(workflowservice.RequestCancelActivityExecutionRequest)),
						})
					}},
				} {
					changed := proto.CloneOf(run)
					forged.change(changed)
					require.False(t, isUnheldCancelScopeRefusal(changed), "forged %s must not pass the scope control", forged.name)
				}
			})
			t.Run("Complete", func(t *testing.T) {
				path, run := invocationRun(t, invocation, 3)
				require.Contains(t, explainedSteps(run), "Worker::RespondCompleted Accepted")
				if kind == "raw" {
					replayPath, replayRun = path, run
				}
			})
			t.Run("Fail", func(t *testing.T) {
				_, run := invocationRun(t, invocation, 1)
				require.Contains(t, explainedSteps(run), "Worker::RespondFailed { retryable: false } Accepted")
			})
			t.Run("RetryThenComplete", func(t *testing.T) {
				_, run := invocationRun(t, invocation, 21)
				steps := explainedSteps(run)
				failed := slices.Index(steps, "Worker::RespondFailed { retryable: true } Accepted")
				require.GreaterOrEqual(t, failed, 0)
				require.Contains(t, steps[failed+1:], "Worker::Poll(2) Accepted")
				require.Contains(t, steps[failed+1:], "Worker::RespondCompleted Accepted")
			})
			t.Run("CancelBeforeWorkerStart", func(t *testing.T) {
				_, run := invocationRun(t, invocation, 19)
				instruction := new(temporalv1.TemporalInstruction)
				require.NoError(t, run.GetEvents()[3].GetProposal().GetResponse().GetRequest().UnmarshalTo(instruction))
				require.True(t, instruction.GetHoldWorkerPolls())
				require.Equal(t, "Client::Cancel Accepted", explainedSteps(run)[0])
				for _, step := range explainedSteps(run) {
					require.NotContains(t, step, "Worker::")
				}
			})
			t.Run("CancelAfterCompletion", func(t *testing.T) {
				_, run := invocationRun(t, invocation, 3)
				steps := explainedSteps(run)
				completed := slices.Index(steps, "Worker::RespondCompleted Accepted")
				require.GreaterOrEqual(t, completed, 0)
				require.Contains(t, steps[completed+1:], "Client::Cancel Rejected(NotFound)")
			})
			if kind == "raw" {
				raw = invocation
			} else {
				for _, seed := range []uint64{1, 3, 4, 19, 21} {
					_, rawRun := invocationRun(t, raw, seed)
					_, sdkRun := invocationRun(t, invocation, seed)
					require.Equal(t, explainedSteps(rawRun), explainedSteps(sdkRun), "seed %d has identical judgments", seed)
					require.True(t, proto.Equal(rawRun.GetVerdict(), sdkRun.GetVerdict()), "seed %d has identical verdicts", seed)
				}
			}
			t.Run("HostRefusal", func(t *testing.T) {
				s.t = t
				data, readErr := os.ReadFile(world)
				require.NoError(t, readErr)
				configuration := new(anypb.Any)
				require.NoError(t, proto.Unmarshal(data, configuration))
				start := new(workflowservice.StartActivityExecutionRequest)
				require.NoError(t, configuration.UnmarshalTo(start))
				start.ActivityId = ""
				data, marshalErr := proto.Marshal(modelSliceAny(t, start))
				require.NoError(t, marshalErr)
				invalid := filepath.Join(t.TempDir(), "invalid-world.binpb")
				require.NoError(t, os.WriteFile(invalid, data, 0o600))
				refused := s.run(driverEnv, "explore", "--world", invalid, "--seed", "1", "--budget", "2")
				require.Equal(t, runner.ExitRefused, refused.code, "%s%s", refused.stdout, refused.stderr)
				_, run := invocationRun(t, refused, 1)
				refusal := run.GetEvents()[run.GetVerdict().GetRefused().GetEvent()].GetRefusal()
				require.Equal(t, modelv1.Method_METHOD_PROPOSE, refusal.GetMethod())
				require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_MALFORMED_REQUEST, refusal.GetRefusal().GetCode())
			})
		})
	}
	t.Run("Controls", func(t *testing.T) {
		for _, control := range []struct {
			name string
			code int
		}{
			{"complete-instead-of-fail", runner.ExitUnexplained},
			{"wrong-result", runner.ExitUnexplained},
			{"empty", runner.ExitRefused},
		} {
			t.Run(control.name, func(t *testing.T) {
				s.t = t
				queue := "umpire-control-" + uuid.NewString()
				world := s.world(namespace, queue, "application-input", 2)
				driverEnv := s.driverEnv(namespace, queue)
				driverEnv[act.EnvExecutorKind], driverEnv[act.EnvProgram] = "raw", s.mutateProgram(s.program(world), control.name)
				invocation := s.run(driverEnv, "explore", "--world", world, "--seed", "3", "--budget", "7")
				require.Equal(t, control.code, invocation.code, "%s%s", invocation.stdout, invocation.stderr)
				detected := false
				for _, run := range invocation.runs {
					if control.code == runner.ExitUnexplained && run.GetVerdict().GetUnexplained() != nil {
						event := run.GetEvents()[run.GetVerdict().GetUnexplained().GetEvent()]
						observation := new(realizationv1.Observation)
						require.NoError(t, event.GetJudgment().GetRequest().GetObservation().UnmarshalTo(observation))
						require.Equal(t, workflowservice.WorkflowService_DescribeActivityExecution_FullMethodName, observation.GetMethod())
						require.NotNil(t, event.GetJudgment().GetResponse().GetUnexplained())
						response := new(workflowservice.DescribeActivityExecutionResponse)
						require.NoError(t, observation.GetResponse().UnmarshalTo(response))
						require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, response.GetInfo().GetStatus())
						require.Len(t, response.GetOutcome().GetResult().GetPayloads(), 1)
						expected := "Complete"
						if control.name == "wrong-result" {
							expected = "Different"
						}
						require.Equal(t, expected, string(response.GetOutcome().GetResult().GetPayloads()[0].GetData()))
						detected = true
					}
					if control.name == "empty" && run.GetVerdict().GetRefused() != nil {
						require.Contains(t, run.GetVerdict().GetRefused().GetReason(), "unprogrammed")
						require.NotNil(t, run.GetEvents()[run.GetVerdict().GetRefused().GetEvent()].GetRefusal())
						detected = true
					}
				}
				require.True(t, detected, "the control must be detected by the Model")
			})
		}
	})
	t.Run("ProvisionLifecycle", func(t *testing.T) {
		s.t = t
		provisioned, queue := "umpire-provision-"+uuid.NewString(), "umpire-provision-"+uuid.NewString()
		world := s.world(provisioned, queue, "application-input", 2)
		driverEnv := s.driverEnv(provisioned, queue)
		driverEnv[act.EnvCreate] = "true"
		driverEnv[act.EnvExecutorKind], driverEnv[act.EnvProgram] = "raw", s.program(world)
		service := env.SdkClient().WorkflowService()
		request := &workflowservice.DescribeNamespaceRequest{Namespace: provisioned}
		_, err := service.DescribeNamespace(t.Context(), request)
		var missing *serviceerror.NamespaceNotFound
		require.ErrorAs(t, err, &missing, "the CLI must create a fresh namespace")
		observer := func(finished <-chan struct{}) error {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-finished:
					return errors.New("CLI exited before namespace creation was observed")
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				response, err := service.DescribeNamespace(ctx, request)
				if err == nil {
					if response.GetNamespaceInfo().GetName() != provisioned {
						return fmt.Errorf("DescribeNamespace returned %q, expected %q", response.GetNamespaceInfo().GetName(), provisioned)
					}
					select {
					case <-finished:
						return errors.New("CLI exited before namespace creation was observed")
					default:
						return nil
					}
				}
				var missing *serviceerror.NamespaceNotFound
				if !errors.As(err, &missing) {
					return err
				}
				select {
				case <-finished:
					return errors.New("CLI exited before namespace creation was observed")
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
				}
			}
		}
		invocation := s.runObserved(driverEnv, observer, "explore", "--world", world, "--seed", "21", "--budget", "7", "--event-timeout", "60s", "--run-timeout", "2m")
		requireSupportedActivityRuns(t, invocation)
		require.Len(t, invocation.runs, 21)
		for _, run := range invocation.runs {
			requireWorkerRun(t, run, "raw", provisioned, queue)
		}
		await.Requiref(t.Context(), t, func(t *await.T) {
			_, err := service.DescribeNamespace(t.Context(), request)
			var missing *serviceerror.NamespaceNotFound
			require.ErrorAs(t, err, &missing)
		}, 30*time.Second, 100*time.Millisecond, "the CLI did not delete namespace %s", provisioned)
	})
	t.Run("UnreachableEndpoint", func(t *testing.T) {
		s.t = t
		listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, listenErr)
		closed := listener.Addr().String()
		require.NoError(t, listener.Close())
		queue := "umpire-unreachable-" + uuid.NewString()
		world := s.world(namespace, queue, "application-input", 2)
		driverEnv := s.driverEnv(namespace, queue)
		driverEnv[act.EnvAddress], driverEnv[act.EnvExecutorKind], driverEnv[act.EnvProgram] = closed, "raw", s.program(world)
		invocation := s.run(driverEnv, "explore", "--world", world, "--seed", "1")
		require.Equal(t, runner.ExitRefused, invocation.code)
		require.Contains(t, invocation.stdout, "driver fault: the server did not answer")
	})
	t.Run("ReplayWithoutServer", func(t *testing.T) {
		s.t = t
		require.NotEmpty(t, replayPath)
		invocation := s.run(nil, "replay", "--run", replayPath)
		require.Equal(t, runner.ExitExplained, invocation.code, "%s%s", invocation.stdout, invocation.stderr)
		require.Equal(t, "explained: proposal_budget", lastLine(invocation.stdout))
		replayed, replayErr := runner.New(temporal.Guest(), func(context.Context) (runner.Driver, error) {
			t.Fatal("replay tried to make a Driver")
			return nil, errors.New("no server")
		}).Run(t.Context(), runner.ModeReplay, runner.Options{Recorded: replayRun})
		require.NoError(t, replayErr)
		require.True(t, proto.Equal(replayRun.GetWorker(), replayed.GetWorker()))
		require.Equal(t, explainedSteps(replayRun), explainedSteps(replayed))
	})
}
