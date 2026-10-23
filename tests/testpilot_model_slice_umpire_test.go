//go:build test_dep && integration

package tests

import (
	"bytes"
	"errors"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	modelv1 "umpire.local/model/umpire/model/v1"
	runv1 "umpire.local/model/umpire/run/v1"
	"umpire.local/model/umpire/runner"
	"umpire.local/temporal"
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
	err := command.Run()
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

// firstSeed is the smallest Run seed whose proposals at the given request numbers choose the
// action other than Cancel: with two enabled actions the Model takes the one at seed % 2, and
// Cancel is always first.
func firstSeed(requests ...uint64) uint64 {
	for seed := uint64(1); ; seed++ {
		if !slices.ContainsFunc(requests, func(request uint64) bool { return runner.ProposalSeed(seed, request)%2 == 0 }) {
			return seed
		}
	}
}

// firstCancelSeed is the smallest Run seed whose first action proposal is Cancel.
func firstCancelSeed() uint64 {
	for seed := uint64(1); ; seed++ {
		if runner.ProposalSeed(seed, 2)%2 == 0 {
			return seed
		}
	}
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

// proposedAction is the action of a Run's proposal for one request.
func proposedAction(run *runner.Run, request uint64) string {
	for _, event := range run.GetEvents() {
		if event.GetRequest() == request && event.GetProposal() != nil {
			return event.GetProposal().GetResponse().GetAction()
		}
	}
	return ""
}

// requestSteps are the steps and outcomes explained while one request was pending.
func requestSteps(run *runner.Run, request uint64) []string {
	var steps []string
	for _, event := range run.GetEvents() {
		explanation := event.GetJudgment().GetResponse().GetExplained()
		if event.GetRequest() != request {
			continue
		}
		for i, step := range explanation.GetSteps() {
			steps = append(steps, step+" "+explanation.GetOutcomes()[i])
		}
	}
	return steps
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return lines[len(lines)-1]
}

func answers(run *runner.Run) []*runv1.Event {
	var found []*runv1.Event
	for _, event := range run.GetEvents() {
		if event.GetAnswer() != nil {
			found = append(found, event)
		}
	}
	return found
}

// readsAroundACallback reports whether one request has a continued read, then the Answer of a
// callback, then a final Explained read.
func readsAroundACallback(run *runner.Run) bool {
	events := run.GetEvents()
	for i, event := range events {
		if event.GetAnswer() == nil || i == 0 {
			continue
		}
		before, request := events[i-1].GetJudgment().GetResponse(), event.GetRequest()
		if before.GetExplained() == nil || !before.GetContinuePolling() || events[i-1].GetRequest() != request {
			continue
		}
		for _, after := range events[i+1:] {
			judgment := after.GetJudgment().GetResponse()
			if after.GetRequest() == request && judgment.GetExplained() != nil && !judgment.GetContinuePolling() {
				return true
			}
		}
	}
	return false
}

func requireAllExplained(t *testing.T, invocation umpireInvocation) {
	t.Helper()
	require.Equal(t, runner.ExitExplained, invocation.code, "stdout: %s\nstderr: %s", invocation.stdout, invocation.stderr)
	for path, run := range invocation.runs {
		require.NotNil(t, run.GetVerdict().GetExplained(), "%s: %v", path, run.GetVerdict())
	}
}

func TestTestpilotModelSliceThroughUmpire(t *testing.T) {
	// Build before the cluster starts: the environment bounds the test's own deadline.
	binary := buildUmpireCommand(t, "umpire")
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
	require.Equal(t, temporal.Guest(), committed, "the command runs the embedded guest's committed bytes")
	s := &umpireSlice{t: t, root: t, binary: binary, guest: guest, address: env.FrontendGRPCAddress()}
	service := env.SdkClient().WorkflowService()
	namespace := env.Namespace().String()
	const budget = "6"
	explore := func(driverEnv map[string]string, world string, seed uint64) umpireInvocation {
		return s.run(driverEnv, "explore", "--world", world, "--seed", strconv.FormatUint(seed, 10), "--budget", budget,
			"--event-timeout", "60s", "--run-timeout", "2m")
	}

	var complete umpireInvocation
	t.Run("Complete", func(t *testing.T) {
		s.t = t
		provisioned, queue := "umpire-slice-"+uuid.NewString(), "umpire-slice-"+uuid.NewString()
		driverEnv := s.driverEnv(provisioned, queue)
		driverEnv[act.EnvCreate] = "true"
		seeds := max(firstSeed(2, 3), firstCancelSeed())
		complete = explore(driverEnv, s.world(provisioned, queue, "Complete", 1), seeds)
		requireAllExplained(t, complete)
		require.Len(t, complete.runs, int(seeds))
		require.Contains(t, complete.stdout, "explained: proposal_budget")
		var callback, completed bool
		for _, run := range complete.runs {
			require.Equal(t, modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP, run.GetEvents()[1].GetProposal().GetResponse().GetKind())
			callback = callback || readsAroundACallback(run)
			completed = completed || slices.Contains(explainedSteps(run), "Worker::RespondCompleted Accepted")
		}
		require.True(t, callback, "a read continued, a callback was answered, and a final read was explained")
		require.True(t, completed, "the completion was explained")
		require.NotContains(t, complete.stderr, "cleanup")
		// The delete returns once the namespace is renamed to `<name>-deleted-<id>`.
		await.Requiref(t.Context(), t, func(t *await.T) {
			_, err := service.DescribeNamespace(t.Context(), &workflowservice.DescribeNamespaceRequest{Namespace: provisioned})
			var notFound *serviceerror.NamespaceNotFound
			require.ErrorAs(t, err, &notFound)
		}, 30*time.Second, 100*time.Millisecond, "the namespace %s the Driver provisioned was not deleted", provisioned)
	})

	t.Run("Cancellation", func(t *testing.T) {
		var first, afterCompletion bool
		for _, run := range complete.runs {
			if proposedAction(run, 2) == "Client::Cancel" {
				first = first || slices.Contains(requestSteps(run, 2), "Client::Cancel Accepted")
			}
			steps := explainedSteps(run)
			if completedAt := slices.Index(steps, "Worker::RespondCompleted Accepted"); completedAt >= 0 {
				afterCompletion = afterCompletion || slices.Contains(steps[completedAt:], "Client::Cancel Rejected(NotFound)")
			}
		}
		require.True(t, first, "a Run's first action was Cancel, accepted and explained")
		require.True(t, afterCompletion, "a Cancel after completion was rejected NotFound and explained")
	})

	t.Run("RetryThenComplete", func(t *testing.T) {
		s.t = t
		queue := "umpire-slice-" + uuid.NewString()
		seeds := firstSeed(2, 3, 4)
		invocation := explore(s.driverEnv(namespace, queue), s.world(namespace, queue, "RetryThenComplete", 2), seeds)
		requireAllExplained(t, invocation)
		var retried bool
		for _, run := range invocation.runs {
			steps := explainedSteps(run)
			retried = retried || len(answers(run)) == 2 && slices.Contains(steps, "Worker::RespondFailed { retryable: true } Accepted") &&
				slices.Contains(steps, "Worker::RespondCompleted Accepted")
		}
		require.True(t, retried, "a Run answered both attempts and explained the retry and the completion")
	})

	t.Run("Fail", func(t *testing.T) {
		s.t = t
		queue := "umpire-slice-" + uuid.NewString()
		invocation := explore(s.driverEnv(namespace, queue), s.world(namespace, queue, "Fail", 1), firstSeed(2))
		requireAllExplained(t, invocation)
		var failed bool
		for _, run := range invocation.runs {
			failed = failed || slices.Contains(explainedSteps(run), "Worker::RespondFailed { retryable: false } Accepted")
		}
		require.True(t, failed, "the failure was explained")
	})

	var refusedRun, refusedLine string
	t.Run("HostRefusal", func(t *testing.T) {
		s.t = t
		queue := "umpire-slice-" + uuid.NewString()
		world := s.world(namespace, queue, "HostRefusal", 1)
		invocation := explore(s.driverEnv(namespace, queue), world, firstSeed(2))
		require.Equal(t, runner.ExitRefused, invocation.code, "stdout: %s\nstderr: %s", invocation.stdout, invocation.stderr)
		require.Contains(t, invocation.stdout, "model refused METHOD_ANSWER: REFUSAL_CODE_UNSUPPORTED")
		for path, run := range invocation.runs {
			if run.GetVerdict().GetRefused() == nil {
				continue
			}
			refusal := run.GetEvents()[run.GetVerdict().GetRefused().GetEvent()].GetRefusal()
			require.Equal(t, modelv1.Method_METHOD_ANSWER, refusal.GetMethod())
			require.Equal(t, modelv1.RefusalCode_REFUSAL_CODE_UNSUPPORTED, refusal.GetRefusal().GetCode())
			refusedRun = path
		}
		require.NotEmpty(t, refusedRun)
		refusedLine = lastLine(invocation.stdout)
		start := new(workflowservice.StartActivityExecutionRequest)
		data, err := os.ReadFile(world)
		require.NoError(t, err)
		packed := new(anypb.Any)
		require.NoError(t, proto.Unmarshal(data, packed))
		require.NoError(t, packed.UnmarshalTo(start))
		described, err := service.DescribeActivityExecution(t.Context(), &workflowservice.DescribeActivityExecutionRequest{
			Namespace: namespace, ActivityId: start.GetActivityId(),
		})
		require.NoError(t, err)
		require.Equal(t, enumspb.ACTIVITY_EXECUTION_STATUS_TERMINATED, described.GetInfo().GetStatus(),
			"Close terminated the activity; the Driver never answered the refused task itself")
	})

	t.Run("UnreachableEndpoint", func(t *testing.T) {
		s.t = t
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		closed := listener.Addr().String()
		require.NoError(t, listener.Close())
		queue := "umpire-slice-" + uuid.NewString()
		driverEnv := s.driverEnv(namespace, queue)
		driverEnv[act.EnvAddress] = closed
		invocation := explore(driverEnv, s.world(namespace, queue, "Complete", 1), 1)
		require.Equal(t, runner.ExitRefused, invocation.code)
		require.Contains(t, invocation.stdout, "refused at event 2: driver fault: the server did not answer")
		require.Len(t, invocation.runs, 1)
	})

	t.Run("ReplayWithoutServer", func(t *testing.T) {
		s.t = t
		var replayed bool
		for path, run := range complete.runs {
			if len(answers(run)) == 0 {
				continue
			}
			invocation := s.run(nil, "replay", "--run", path)
			require.Equal(t, runner.ExitExplained, invocation.code, "stdout: %s\nstderr: %s", invocation.stdout, invocation.stderr)
			require.Contains(t, invocation.stdout, "explained: proposal_budget")
			replayed = true
			break
		}
		require.True(t, replayed, "a callback-bearing Run was replayed")
		require.NotEmpty(t, refusedRun)
		invocation := s.run(nil, "replay", "--run", refusedRun)
		require.Equal(t, runner.ExitRefused, invocation.code, "stdout: %s\nstderr: %s", invocation.stdout, invocation.stderr)
		require.Contains(t, refusedLine, "model refused METHOD_ANSWER: REFUSAL_CODE_UNSUPPORTED")
		require.Equal(t, refusedLine, lastLine(invocation.stdout), "replay reproduces the refusal at the same event")
	})
}
