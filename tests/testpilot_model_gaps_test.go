//go:build test_dep && integration

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/chasm/lib/activity"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/testing/testcontext"
	"go.temporal.io/server/common/testing/testlogger"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	model "umpire.local/model/host"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	temporalmodel "umpire.local/temporal"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

type modelGapPrivateT struct{ testlogger.CleanupCapableT }

func (t modelGapPrivateT) Log(...any)          { t.CleanupCapableT.Log("server log [redacted]") }
func (t modelGapPrivateT) Logf(string, ...any) { t.Log() }
func (t modelGapPrivateT) Errorf(string, ...any) {
	t.CleanupCapableT.Errorf("server log failure [redacted]")
}
func (t modelGapPrivateT) Fatal(...any)          { t.CleanupCapableT.Fatal("server log failure [redacted]") }
func (t modelGapPrivateT) Fatalf(string, ...any) { t.Fatal() }

type modelGapLogSink struct{ t testlogger.TestingT }

func (logger modelGapLogSink) emit(level string) {
	logger.t.Logf("gap server log level=%s [redacted]", level)
}
func (logger modelGapLogSink) Debug(string, ...tag.Tag)  { logger.emit("Debug") }
func (logger modelGapLogSink) Info(string, ...tag.Tag)   { logger.emit("Info") }
func (logger modelGapLogSink) Warn(string, ...tag.Tag)   { logger.emit("Warn") }
func (logger modelGapLogSink) Error(string, ...tag.Tag)  { logger.emit("Error") }
func (logger modelGapLogSink) DPanic(string, ...tag.Tag) { logger.emit("DPanic") }
func (logger modelGapLogSink) Fatal(string, ...tag.Tag)  { logger.t.Fatal("server fatal [redacted]") }
func (logger modelGapLogSink) Panic(string, ...tag.Tag)  { panic("server panic [redacted]") }

type modelGapLogger struct{ log.Logger }

func newModelGapLogger(t testlogger.CleanupCapableT) log.Logger {
	private := modelGapPrivateT{t}
	logger := testlogger.NewTestLogger(private, testlogger.FailOnExpectedErrorOnly, testlogger.WrapLogger(modelGapLogSink{t}))
	logger.Expect(testlogger.Error, ".*", tag.FailedAssertion)
	t.Cleanup(func() {
		if logger.Failure() != nil {
			private.Errorf("server logger failed during execution or cleanup")
		}
	})
	return modelGapLogger{logger}
}

type modelGapRecordedT struct {
	*testing.T
	output   strings.Builder
	failures int
	cleanups []func()
}

func (t *modelGapRecordedT) Log(values ...any)                   { _, _ = fmt.Fprintln(&t.output, values...) }
func (t *modelGapRecordedT) Logf(format string, values ...any)   { t.Log(fmt.Sprintf(format, values...)) }
func (t *modelGapRecordedT) Fail()                               { t.failures++ }
func (t *modelGapRecordedT) Failed() bool                        { return t.failures != 0 }
func (t *modelGapRecordedT) FailNow()                            { t.Fail() }
func (t *modelGapRecordedT) Errorf(format string, values ...any) { t.Logf(format, values...); t.Fail() }
func (t *modelGapRecordedT) Fatal(values ...any)                 { t.Log(values...); t.FailNow() }
func (t *modelGapRecordedT) Fatalf(format string, values ...any) {
	t.Fatal(fmt.Sprintf(format, values...))
}
func (t *modelGapRecordedT) Cleanup(cleanup func()) { t.cleanups = append(t.cleanups, cleanup) }

func TestTestpilotModelGapLoggerPreservesFailuresWithoutPrivateOutput(t *testing.T) {
	recorded := &modelGapRecordedT{T: t}
	logger := newModelGapLogger(recorded)
	secret := "private-token-payload-error-stack"
	logger.Info(secret, tag.NewStringTag("private-tag", secret))
	logger.Error(secret, tag.FailedAssertion, tag.NewStringTag("private-tag", secret))
	require.True(t, recorded.Failed(), "failed-assertion Error must still fail its test")
	beforeCleanup := recorded.failures
	for i := len(recorded.cleanups) - 1; i >= 0; i-- {
		recorded.cleanups[i]()
	}
	require.Greater(t, recorded.failures, beforeCleanup, "sticky logger failure must remain visible during cleanup")
	require.NotContains(t, recorded.output.String(), secret)
	require.Contains(t, recorded.output.String(), "level=Error [redacted]")
	require.Contains(t, recorded.output.String(), "server log failure [redacted]")
	for _, expectedCode := range []int32{int32(codes.InvalidArgument), 0} {
		observation, err := model.ObserveProbe(&annotationsv1.ProbeCase{
			Id: "loose-rejection", Code: expectedCode,
		}, status.Error(codes.InvalidArgument, secret))
		require.NoError(t, err)
		require.Equal(t, int32(codes.InvalidArgument), observation.Code)
		require.True(t, observation.MessageMatches)
		encoded, err := proto.Marshal(observation)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), secret, "loose observations must remain value-free")
	}
}

func isolatedProbeRequest(caseID, namespace string, baseline bool) *temporalv1.ProbeRequest {
	return &temporalv1.ProbeRequest{
		Operation: temporalv1.ProbeRequest_PREPARE, CaseId: caseID, Baseline: baseline,
		Namespace: namespace, ActivityId: uuid.NewString(), TaskQueue: "model-gap-" + uuid.NewString(),
	}
}

func TestTestpilotModelGapResourceIsolation(t *testing.T) {
	baseline := isolatedProbeRequest("case", "namespace", true)
	mutation := isolatedProbeRequest("case", "namespace", false)
	require.NotEqual(t, baseline.ActivityId, mutation.ActivityId)
	require.NotEqual(t, baseline.TaskQueue, mutation.TaskQueue)
	require.Empty(t, baseline.RunId)
	require.Empty(t, mutation.RunId)
	require.Empty(t, baseline.TaskToken)
	require.Empty(t, mutation.TaskToken)
}

func modelGapTestContext(t *testing.T) context.Context {
	t.Helper()
	started := time.Now()
	ctx := testcontext.For(t, testcontext.WithTimeout(20*time.Minute))
	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	expected := started.Add(20 * time.Minute)
	if goDeadline, bounded := t.Deadline(); bounded && goDeadline.Before(expected) {
		expected = goDeadline
	}
	require.WithinDuration(t, expected, deadline, time.Second)
	extended := testcontext.EnsureRemaining(ctx, t, 30*time.Minute)
	extendedDeadline, ok := extended.Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, extendedDeadline, "aggregate ceiling must not extend")
	return ctx
}

func TestTestpilotModelGapContextBudget(t *testing.T) {
	modelGapTestContext(t)
}

func newModelGapConnection(t *testing.T) (string, *grpc.ClientConn) {
	t.Helper()
	env := newTestpilotTestEnvironment(t,
		testcore.WithLogger(newModelGapLogger(t)),
		testcore.WithDynamicConfig(dynamicconfig.EnableChasm, true),
		testcore.WithDynamicConfig(activity.Enabled, true),
		testcore.WithDynamicConfig(activity.EnableStandaloneActivityOperatorCommands, true))
	connection, err := grpc.NewClient(env.FrontendGRPCAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	return env.Namespace().String(), connection
}

func modelGapStartRequest(namespace string) *workflowservice.StartActivityExecutionRequest {
	return &workflowservice.StartActivityExecutionRequest{
		Namespace: namespace, ActivityId: uuid.NewString(), RequestId: uuid.NewString(),
		ActivityType:           &commonpb.ActivityType{Name: "ModelSlice"},
		TaskQueue:              &taskqueuepb.TaskQueue{Name: "model-gap-" + uuid.NewString()},
		ScheduleToCloseTimeout: durationpb.New(time.Minute), StartToCloseTimeout: durationpb.New(20 * time.Second),
		RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 2, InitialInterval: durationpb.New(time.Second),
			MaximumInterval: durationpb.New(10 * time.Second), BackoffCoefficient: 2},
	}
}

func checkModelGapNamespaceAdmission(t *testing.T, namespace string, service workflowservice.WorkflowServiceClient,
	start func(*testing.T, *workflowservice.StartActivityExecutionRequest) (*workflowservice.StartActivityExecutionResponse, error),
) {
	t.Helper()
	for _, method := range []string{"StartActivityExecution", "DescribeActivityExecution", "PollActivityExecution", "PollActivityTaskQueue", "RequestCancelActivityExecution"} {
		for _, supplied := range []bool{false, true} {
			t.Run(fmt.Sprintf("namespace_%s_%t", method, supplied), func(t *testing.T) {
				requestNamespace, expected := "", codes.InvalidArgument
				if supplied {
					requestNamespace, expected = namespace, codes.OK
				}
				seed := modelGapStartRequest(namespace)
				invoke := func() error {
					if method == "StartActivityExecution" {
						seed.Namespace = requestNamespace
						_, err := start(t, seed)
						return err
					}
					started, err := start(t, seed)
					require.Equal(t, codes.OK, status.Code(err))
					ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
					defer cancel()
					if method == "PollActivityExecution" {
						task, pollErr := service.PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{Namespace: namespace, TaskQueue: seed.TaskQueue})
						require.Equal(t, codes.OK, status.Code(pollErr))
						require.NotEmpty(t, task.GetTaskToken())
						_, completeErr := service.RespondActivityTaskCompleted(ctx, &workflowservice.RespondActivityTaskCompletedRequest{TaskToken: task.TaskToken})
						require.Equal(t, codes.OK, status.Code(completeErr))
					}
					switch method {
					case "DescribeActivityExecution":
						_, err = service.DescribeActivityExecution(ctx, &workflowservice.DescribeActivityExecutionRequest{Namespace: requestNamespace, ActivityId: seed.ActivityId, RunId: started.RunId})
					case "PollActivityExecution":
						_, err = service.PollActivityExecution(ctx, &workflowservice.PollActivityExecutionRequest{Namespace: requestNamespace, ActivityId: seed.ActivityId, RunId: started.RunId})
					case "PollActivityTaskQueue":
						_, err = service.PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{Namespace: requestNamespace, TaskQueue: seed.TaskQueue})
					case "RequestCancelActivityExecution":
						_, err = service.RequestCancelActivityExecution(ctx, &workflowservice.RequestCancelActivityExecutionRequest{Namespace: requestNamespace, ActivityId: seed.ActivityId, RunId: started.RunId})
					default:
						t.Fatalf("unknown static admission control %s", method)
					}
					return err
				}
				rpcErr := invoke()
				require.Equal(t, expected, status.Code(rpcErr))
				if !supplied {
					matchesReason := strings.Contains(status.Convert(rpcErr).Message(), "Namespace is empty")
					require.True(t, matchesReason, "namespace explanation must match without exposing the error")
				}
			})
		}
	}
}

func checkModelGapScopeBoundaries(t *testing.T, namespace string, connection *grpc.ClientConn) {
	t.Helper()
	service := workflowservice.NewWorkflowServiceClient(connection)
	start := func(t *testing.T, request *workflowservice.StartActivityExecutionRequest) (*workflowservice.StartActivityExecutionResponse, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cleanupCancel()
			_, cleanupErr := service.TerminateActivityExecution(cleanupCtx, &workflowservice.TerminateActivityExecutionRequest{
				Namespace: namespace, ActivityId: request.ActivityId,
			})
			require.True(t, cleanupErr == nil || status.Code(cleanupErr) == codes.NotFound, "cleanup rpc_code=%s", status.Code(cleanupErr))
		})
		return service.StartActivityExecution(ctx, request)
	}
	checkModelGapNamespaceAdmission(t, namespace, service, start)
	for _, testCase := range []struct {
		name     string
		maximum  time.Duration
		attempts int32
		code     codes.Code
	}{
		{"retry_inverted", 10 * time.Second, 2, codes.InvalidArgument},
		{"retry_ordered", 2002 * time.Second, 2, codes.OK},
		{"retry_disabled", 10 * time.Second, 1, codes.OK},
		{"retry_default_maximum", 0, 2, codes.OK},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := modelGapStartRequest(namespace)
			request.RetryPolicy.InitialInterval = durationpb.New(1001 * time.Second)
			request.RetryPolicy.MaximumInterval = durationpb.New(testCase.maximum)
			request.RetryPolicy.MaximumAttempts = testCase.attempts
			_, rpcErr := start(t, request)
			require.Equal(t, testCase.code, status.Code(rpcErr), "retry rpc_code=%s", status.Code(rpcErr))
			if rpcErr != nil {
				matchesReason := strings.Contains(status.Convert(rpcErr).Message(), "MaximumInterval cannot be less")
				require.True(t, matchesReason, "retry explanation must match without exposing the error")
			}
		})
	}
	for _, normalName := range []string{"", "x"} {
		t.Run(fmt.Sprintf("normal_queue_name_%d", len(normalName)), func(t *testing.T) {
			request := modelGapStartRequest(namespace)
			_, startErr := start(t, request)
			require.Equal(t, codes.OK, status.Code(startErr))
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			_, rpcErr := service.PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{
				Namespace: namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: request.TaskQueue.Name, Kind: enumspb.TASK_QUEUE_KIND_NORMAL, NormalName: normalName},
			})
			expected := codes.OK
			if normalName != "" {
				expected = codes.InvalidArgument
			}
			require.Equal(t, expected, status.Code(rpcErr), "queue relation rpc_code=%s", status.Code(rpcErr))
		})
	}
	for _, tokenKind := range []string{"empty", "malformed", "issued"} {
		t.Run("worker_token_"+tokenKind, func(t *testing.T) {
			request := modelGapStartRequest(namespace)
			_, startErr := start(t, request)
			require.Equal(t, codes.OK, status.Code(startErr))
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			task, pollErr := service.PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{Namespace: namespace, TaskQueue: request.TaskQueue})
			require.Equal(t, codes.OK, status.Code(pollErr))
			require.NotEmpty(t, task.GetTaskToken())
			token, expected, pattern := task.TaskToken, codes.OK, ""
			switch tokenKind {
			case "empty":
				token, expected, pattern = nil, codes.InvalidArgument, "Task token not set"
			case "malformed":
				token, expected, pattern = []byte{1}, codes.InvalidArgument, "deserializing task token"
			default:
				require.Equal(t, "issued", tokenKind)
			}
			_, rpcErr := service.RespondActivityTaskCompleted(ctx, &workflowservice.RespondActivityTaskCompletedRequest{TaskToken: token})
			require.Equal(t, expected, status.Code(rpcErr), "opaque token rpc_code=%s", status.Code(rpcErr))
			matchesReason := strings.Contains(status.Convert(rpcErr).Message(), pattern)
			require.True(t, matchesReason, "token explanation must match without exposing the error")
		})
	}
}

func TestTestpilotModelGapScopeBoundaries(t *testing.T) {
	namespace, connection := newModelGapConnection(t)
	checkModelGapScopeBoundaries(t, namespace, connection)
}

func TestTestpilotModelGaps(t *testing.T) {
	modelGapTestContext(t)
	host, err := model.New(t.Context(), temporalmodel.Guest())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, host.Close(context.Background())) })
	inventory, err := temporalmodel.Probe(t.Context(), host, &temporalv1.ProbeRequest{})
	require.NoError(t, err)
	require.LessOrEqual(t, proto.Size(inventory), 1<<20, "inventory must fit the unchanged probe frame budget")
	require.NotEmpty(t, inventory.GetCases(), "probe inventory must not silently execute zero cases")
	identities := make(map[string]bool)
	for _, testCase := range inventory.Cases {
		require.NotEmpty(t, testCase.Id)
		require.False(t, identities[testCase.Id], "duplicate probe identity")
		identities[testCase.Id] = true
	}
	namespace, connection := newModelGapConnection(t)
	checkModelGapScopeBoundaries(t, namespace, connection)
	service := workflowservice.NewWorkflowServiceClient(connection)
	var observations []*annotationsv1.ProbeObservation
	for _, testCase := range inventory.Cases {
		t.Run(testCase.Id, func(t *testing.T) {
			if testCase.Generic && testCase.Applicability != annotationsv1.ProbeApplicability_APPLICABLE {
				observation, observeErr := model.ObserveProbe(testCase, nil)
				require.NoError(t, observeErr)
				observations = append(observations, observation)
				return
			}
			for _, baseline := range []bool{true, false} {
				func() {
					ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
					defer cancel()
					request := isolatedProbeRequest(testCase.Id, namespace, baseline)
					prepare := func(selector string) *annotationsv1.ProbeResponse {
						preparedRequest := proto.Clone(request).(*temporalv1.ProbeRequest)
						preparedRequest.CaseId, preparedRequest.Baseline = selector, true
						prepared, err := temporalmodel.Probe(ctx, host, preparedRequest)
						require.NoError(t, err, "prerequisite generation")
						require.LessOrEqual(t, proto.Size(prepared), 1<<20)
						return prepared
					}
					defer func() {
						cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cleanupCancel()
						_, cleanupErr := service.TerminateActivityExecution(cleanupCtx, &workflowservice.TerminateActivityExecutionRequest{
							Namespace: request.Namespace, ActivityId: request.ActivityId, RunId: request.RunId,
						})
						require.True(t, cleanupErr == nil || status.Code(cleanupErr) == codes.NotFound, "cleanup rpc_code=%s", status.Code(cleanupErr))
					}()
					if testCase.Prerequisite != "start" {
						response, rpcErr := temporalmodel.DispatchProbe(ctx, connection, prepare(""))
						require.NotNil(t, response, "prerequisite must reach actual RPC dispatch")
						require.True(t, rpcErr == nil, "provision start rpc_code=%s", status.Code(rpcErr))
						request.RunId = response.(*workflowservice.StartActivityExecutionResponse).RunId
						switch testCase.Prerequisite {
						case "scheduled", "poll":
						case "completed", "token", "canceled_token":
							task, pollErr := service.PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{
								Namespace: request.Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: request.TaskQueue}, Identity: "model-gap",
							})
							require.True(t, pollErr == nil, "provision poll rpc_code=%s", status.Code(pollErr))
							require.NotEmpty(t, task.GetTaskToken(), "prerequisite requires an actual polled task")
							request.TaskToken = task.TaskToken
							if testCase.Prerequisite == "completed" || testCase.Prerequisite == "canceled_token" {
								selector := "RespondActivityTaskCompleted"
								if testCase.Prerequisite == "canceled_token" {
									selector = "RequestCancelActivityExecution"
								}
								prerequisiteResponse, prerequisiteErr := temporalmodel.DispatchProbe(ctx, connection, prepare(selector))
								require.NotNil(t, prerequisiteResponse, "prerequisite must reach actual RPC dispatch")
								require.True(t, prerequisiteErr == nil, "provision state rpc_code=%s", status.Code(prerequisiteErr))
							}
						default:
							t.Fatalf("unsupported prerequisite %q", testCase.Prerequisite)
						}
					}
					prepared, prepareErr := temporalmodel.Probe(ctx, host, request)
					require.NoError(t, prepareErr, "probe generation")
					require.LessOrEqual(t, proto.Size(prepared), 1<<20, "prepared witness must use a bounded transport frame")
					require.Equal(t, testCase.Method, prepared.Method)
					observationCase := testCase
					if testCase.Generic {
						require.Len(t, prepared.Cases, 1)
						observationCase = prepared.Cases[0]
						require.Equal(t, testCase.Id, observationCase.Id)
						require.Equal(t, testCase.Method, observationCase.Method)
						if observationCase.Applicability != annotationsv1.ProbeApplicability_APPLICABLE {
							require.False(t, baseline)
							observation, observeErr := model.ObserveProbe(observationCase, nil)
							require.NoError(t, observeErr)
							observations = append(observations, observation)
							return
						}
					}
					response, rpcErr := temporalmodel.DispatchProbe(ctx, connection, prepared)
					require.NotNil(t, response, "probe must reach actual RPC dispatch")
					if started, ok := response.(*workflowservice.StartActivityExecutionResponse); ok && rpcErr == nil {
						encoded := prepared.Message
						if prepared.Recipe != nil {
							encoded, err = model.ExpandProbeRecipe(prepared.Recipe)
							require.NoError(t, err)
						}
						actual := new(workflowservice.StartActivityExecutionRequest)
						require.NoError(t, proto.Unmarshal(encoded, actual))
						request.Namespace, request.ActivityId, request.RunId = actual.Namespace, actual.ActivityId, started.RunId
					}
					if baseline {
						require.True(t, rpcErr == nil, "valid baseline rpc_code=%s", status.Code(rpcErr))
						return
					}
					observation, observeErr := model.ObserveProbe(observationCase, rpcErr)
					require.NoError(t, observeErr)
					observations = append(observations, observation)
				}()
			}
		})
	}
	require.Len(t, observations, len(inventory.Cases), "every inventory identity requires an observation or explicit local refusal")
	classification := &temporalv1.ProbeRequest{
		Operation: temporalv1.ProbeRequest_CLASSIFY, Observations: observations,
	}
	require.LessOrEqual(t, proto.Size(classification), 1<<20, "classification must fit the unchanged probe frame budget")
	report, err := temporalmodel.Probe(t.Context(), host, classification)
	require.NoError(t, err)
	require.LessOrEqual(t, proto.Size(report), 1<<20, "report must fit the unchanged probe frame budget")
	t.Logf("Probe frame bytes: inventory=%d classify_request=%d classify_response=%d", proto.Size(inventory), proto.Size(classification), proto.Size(report))
	require.NotEmpty(t, report.Report)
	repository, err := filepath.EvalSymlinks("..")
	require.NoError(t, err)
	repository, err = filepath.Abs(repository)
	require.NoError(t, err)
	flowDirectory, err := filepath.EvalSymlinks(filepath.Join(repository, ".flow"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repository, ".flow"), flowDirectory, "report parent must not alias goldens")
	temporary := filepath.Join(flowDirectory, "tmp")
	require.NoError(t, os.MkdirAll(temporary, 0755))
	resolved, err := filepath.EvalSymlinks(temporary)
	require.NoError(t, err)
	require.Equal(t, temporary, resolved, "report directory must not alias goldens")
	root, err := os.OpenRoot(temporary)
	require.NoError(t, err)
	defer func() { require.NoError(t, root.Close()) }()
	file, err := root.OpenFile("fn2174-live-gap-report.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	require.NoError(t, err)
	_, writeErr := file.WriteString(report.Report)
	closeErr := file.Close()
	require.NoError(t, writeErr)
	require.NoError(t, closeErr)
	require.False(t, report.Fails, "catalog gap classification failed; value-free report:\n%s", report.Report)
	golden, err := os.ReadFile(filepath.Join("..", "model", "temporal", "feature", "activity", "expected", "gaps.txt"))
	require.NoError(t, err, "gaps golden must be explicitly reviewed and created")
	require.Equal(t, string(golden), report.Report, "stale gap golden")
}
