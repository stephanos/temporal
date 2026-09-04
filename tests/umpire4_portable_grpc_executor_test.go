//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/executor"
	"go.temporal.io/server/tools/umpire/executorgrpc"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestUmpirePortableGRPCExecutor(t *testing.T) {
	env, attachedFactory := newUmpireTestEnvironment(t)
	noToolchainPath := t.TempDir()
	t.Setenv("PATH", noToolchainPath)

	normal := loadPortableGRPCPlan(t, "normal")
	duplicate := loadPortableGRPCPlan(t, "duplicate-delivery")
	external := externalPortableGRPCPlan(t, normal)
	factory := &recordingEnvironmentFactory{delegate: attachedFactory}
	adapter := &portableGRPCTestAdapter{factory: factory}
	resident := &portableGRPCCompletionExecutor{
		delegate: executor.NewPortable(adapter, trustedPortableGRPCPlans(normal, duplicate)),
	}
	client := startPortableGRPCExecutor(t, resident)

	entered, release := adapter.arm(portableGRPCBlockPreparation)
	type callResult struct {
		result *umpirespb.ExecutionResult
		err    error
	}
	firstResult := make(chan callResult, 1)
	go func() {
		result, err := client.Execute(env.Context(), external)
		firstResult <- callResult{result: result, err: err}
	}()
	var earlyFirst *callResult
	await.RequireTrue(t, func() bool {
		select {
		case <-entered:
			return true
		case result := <-firstResult:
			earlyFirst = &result
			return true
		default:
			return false
		}
	}, 10*time.Second, 10*time.Millisecond)
	if earlyFirst != nil {
		require.NoError(t, earlyFirst.err, "deep executor: %v", resident.lastError())
		require.FailNow(t, "first execution completed before entering the controlled preparation phase")
	}

	overlapErrors := make(chan error, 9)
	var overlap sync.WaitGroup
	for range 9 {
		overlap.Add(1)
		go func() {
			defer overlap.Done()
			_, err := client.Execute(env.Context(), external)
			overlapErrors <- err
		}()
	}
	overlap.Wait()
	close(overlapErrors)
	for err := range overlapErrors {
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	}
	require.Equal(t, 1, adapter.requestCount())
	close(release)
	externalCall := <-firstResult
	require.NoError(t, externalCall.err)
	requirePortableGRPCResult(t, externalCall.result, umpirespb.CLAIM_SCOPE_PLAN_LOCAL, umpirespb.EXECUTION_DECISION_PASS)
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())

	modelResult, err := client.Execute(env.Context(), normal)
	require.NoError(t, err)
	requirePortableGRPCResult(t, modelResult, umpirespb.CLAIM_SCOPE_MODEL_BOUND, umpirespb.EXECUTION_DECISION_PASS)
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())

	negativeResult, err := client.Execute(env.Context(), duplicate)
	require.NoError(t, err)
	requirePortableGRPCResult(t, negativeResult, umpirespb.CLAIM_SCOPE_MODEL_BOUND, umpirespb.EXECUTION_DECISION_FAIL)
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())
	require.NotEqual(t, externalCall.result.GetRunIdentity(), modelResult.GetRunIdentity())
	require.NotEqual(t, modelResult.GetRunIdentity(), negativeResult.GetRunIdentity())
	require.Len(t, factory.identities, 3)
	requireFreshCorrelations(t, adapter.requestAt(0), adapter.requestAt(1))
	requireFreshCorrelations(t, adapter.requestAt(1), adapter.requestAt(2))
	require.NotEqual(t, factory.identities[0].TaskQueue, factory.identities[1].TaskQueue)
	require.NotEqual(t, factory.identities[1].TaskQueue, factory.identities[2].TaskQueue)

	requestsBeforeRejection := adapter.requestCount()
	malformed := proto.CloneOf(normal)
	malformed.PlanId = ""
	_, err = client.Execute(env.Context(), malformed)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	unknown := proto.CloneOf(normal)
	unknown.ProtoReflect().SetUnknown([]byte{0xf8, 0x7f, 0x01})
	_, err = client.Execute(env.Context(), unknown)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	crossedEvidence := proto.CloneOf(normal)
	crossedEvidence.Verification.Observation.Profile.Definition = proto.CloneOf(
		duplicate.GetVerification().GetObservation().GetProfile().GetDefinition(),
	)
	_, err = client.Execute(env.Context(), crossedEvidence)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	forged := proto.CloneOf(normal)
	forged.GetModelCompiled().CompilerContract.BehaviorFingerprint =
		"sha256:0000000000000000000000000000000000000000000000000000000000000000"
	forged, err = testplan.Seal(forged)
	require.NoError(t, err)
	_, err = client.Execute(env.Context(), forged)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))

	beyondLimit := proto.CloneOf(normal)
	beyondLimit.Limits.Execution.MaxActions = testplan.MaximumActionCount + 1
	_, err = client.Execute(env.Context(), beyondLimit)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Equal(t, requestsBeforeRejection, adapter.requestCount())

	adapter.arm(portableGRPCFailObservation)
	closureFailure, err := client.Execute(env.Context(), normal)
	require.NoError(t, err)
	require.Equal(t, umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE, closureFailure.GetCleanupStatus())
	require.Equal(t, umpirespb.EXECUTION_DECISION_INCONCLUSIVE, closureFailure.GetDecision())
	require.Empty(t, closureFailure.GetProperties())
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())

	for _, cancellation := range []struct {
		name string
		call func(context.Context, context.CancelFunc)
		code codes.Code
	}{
		{
			name: "cancellation",
			call: func(_ context.Context, cancel context.CancelFunc) { cancel() },
			code: codes.Canceled,
		},
		{
			name: "deadline",
			call: func(ctx context.Context, _ context.CancelFunc) { <-ctx.Done() },
			code: codes.DeadlineExceeded,
		},
	} {
		t.Run(cancellation.name, func(t *testing.T) {
			requestsBeforeCancellation := adapter.requestCount()
			entered, _ := adapter.arm(portableGRPCBlockObservation)
			completed := resident.armCompletion()
			ctx, cancel := context.WithTimeout(env.Context(), 250*time.Millisecond)
			defer cancel()
			callDone := make(chan error, 1)
			go func() {
				_, err := client.Execute(ctx, normal)
				callDone <- err
			}()
			requirePortableGRPCSignal(t, entered)
			cancellation.call(ctx, cancel)
			require.Equal(t, cancellation.code, status.Code(<-callDone))
			requirePortableGRPCSignal(t, completed)
			require.Equal(t, requestsBeforeCancellation+1, adapter.requestCount())
			requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())

			fresh, err := client.Execute(env.Context(), external)
			require.NoError(t, err)
			require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, fresh.GetDecision())
			require.Equal(t, requestsBeforeCancellation+2, adapter.requestCount())
			requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())
		})
	}

	adapter.arm(portableGRPCFailCleanup)
	poisonedResult, err := client.Execute(env.Context(), normal)
	require.NoError(t, err)
	require.Equal(t, umpirespb.EXECUTION_CLEANUP_STATUS_FAILED, poisonedResult.GetCleanupStatus())
	require.Equal(t, umpirespb.EXECUTION_DECISION_INCONCLUSIVE, poisonedResult.GetDecision())
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())
	requestsBeforePoisonedReuse := adapter.requestCount()
	_, err = client.Execute(env.Context(), normal)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Equal(t, requestsBeforePoisonedReuse, adapter.requestCount())

	for _, tool := range []string{"go", "lake", "lean", "make", "mise", "sh"} {
		_, err = exec.LookPath(tool)
		require.Error(t, err, "the tagged runtime must not have a toolchain executable available")
	}
	require.Equal(t, noToolchainPath, os.Getenv("PATH"))
}

func startPortableGRPCExecutor(
	t *testing.T,
	resident executorgrpc.Executor,
) umpirespb.UmpireExecutorClient {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := executorgrpc.New(resident)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		serveErr := <-serveDone
		require.True(t, serveErr == nil || errors.Is(serveErr, grpc.ErrServerStopped))
	})
	connection, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(int(testplan.MaximumPlanBytes)),
			grpc.MaxCallRecvMsgSize(int(testplan.MaximumResultBytes)),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	return umpirespb.NewUmpireExecutorClient(connection)
}

func loadPortableGRPCPlan(t *testing.T, name string) *umpirespb.PortableTestPlan {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "tools", "umpire", "portableevaluation", "testdata", "portable-test-plan-v1", name, "plan.pb",
	))
	require.NoError(t, err)
	plan := new(umpirespb.PortableTestPlan)
	require.NoError(t, (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, plan))
	return plan
}

func externalPortableGRPCPlan(
	t *testing.T,
	model *umpirespb.PortableTestPlan,
) *umpirespb.PortableTestPlan {
	t.Helper()
	external := proto.CloneOf(model)
	sources := make([]*umpirespb.SourceLocation, len(model.GetModelCompiled().GetSources()))
	for index, source := range model.GetModelCompiled().GetSources() {
		sources[index] = proto.CloneOf(source)
	}
	external.Provenance = &umpirespb.PortableTestPlan_External{External: &umpirespb.ExternalPlanProvenance{
		Sources: sources,
	}}
	sealed, err := testplan.Seal(external)
	require.NoError(t, err)
	return sealed
}

func trustedPortableGRPCPlans(
	plans ...*umpirespb.PortableTestPlan,
) testplan.ModelProvenanceVerifier {
	trusted := make([]testplan.ModelProvenanceBinding, len(plans))
	for index, plan := range plans {
		trusted[index] = testplan.ModelProvenanceBinding{
			PlanChecksum:  bytes.Clone(plan.GetPlanChecksum()),
			ModelCompiled: proto.CloneOf(plan.GetModelCompiled()),
		}
	}
	return func(
		_ context.Context,
		requested testplan.ModelProvenanceBinding,
	) (testplan.ModelProvenanceBinding, error) {
		for _, binding := range trusted {
			if bytes.Equal(requested.PlanChecksum, binding.PlanChecksum) &&
				proto.Equal(requested.ModelCompiled, binding.ModelCompiled) {
				return testplan.ModelProvenanceBinding{
					PlanChecksum:  bytes.Clone(binding.PlanChecksum),
					ModelCompiled: proto.CloneOf(binding.ModelCompiled),
				}, nil
			}
		}
		return testplan.ModelProvenanceBinding{}, errors.New("portable model provenance is not trusted")
	}
}

func requirePortableGRPCResult(
	t *testing.T,
	result *umpirespb.ExecutionResult,
	scope umpirespb.ClaimScope,
	decision umpirespb.ExecutionDecision,
) {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetRunIdentity())
	require.Equal(t, scope, result.GetClaimScope())
	require.Equal(t, umpirespb.EXECUTION_TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.EXECUTION_OPERATIONAL_STATUS_SUCCEEDED, result.GetOperationalStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.TRACE_PROJECTION_STATUS_APPLIED, result.GetTraceProjection().GetStatus())
	require.Equal(t, umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE, result.GetCleanupStatus())
	require.Equal(t, decision, result.GetDecision())
	require.NotEmpty(t, result.GetEvidenceLinks())
	require.NotNil(t, result.GetWork())
	require.LessOrEqual(t, result.GetWork().GetTotal(), result.GetWork().GetLimit())
	if decision == umpirespb.EXECUTION_DECISION_PASS {
		require.Equal(t, umpirespb.EXECUTION_EVALUATION_STATUS_SATISFIED, result.GetSemanticStatus())
	} else {
		require.Equal(t, umpirespb.EXECUTION_EVALUATION_STATUS_VIOLATED, result.GetSemanticStatus())
	}
}

type portableGRPCControlMode int

const (
	portableGRPCControlNone portableGRPCControlMode = iota
	portableGRPCBlockPreparation
	portableGRPCBlockObservation
	portableGRPCFailObservation
	portableGRPCFailCleanup
)

type portableGRPCControl struct {
	mode    portableGRPCControlMode
	entered chan struct{}
	release chan struct{}
}

type portableGRPCTestAdapter struct {
	nexus.Binding
	factory  *recordingEnvironmentFactory
	mu       sync.Mutex
	control  *portableGRPCControl
	requests []umpireruntime.CheckedRunRequest
}

func (a *portableGRPCTestAdapter) arm(mode portableGRPCControlMode) (<-chan struct{}, chan struct{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	control := &portableGRPCControl{
		mode: mode, entered: make(chan struct{}), release: make(chan struct{}),
	}
	a.control = control
	return control.entered, control.release
}

func (a *portableGRPCTestAdapter) CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	request, err := a.Binding.CheckRequest(admitted, runIdentity)
	if err == nil {
		a.mu.Lock()
		a.requests = append(a.requests, request)
		a.mu.Unlock()
	}
	return request, err
}

func (a *portableGRPCTestAdapter) requestCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.requests)
}

func (a *portableGRPCTestAdapter) requestAt(index int) umpireruntime.CheckedRunRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.requests[index]
}

func (a *portableGRPCTestAdapter) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	return a.factory
}

func (a *portableGRPCTestAdapter) NewParticipant(
	request umpireruntime.CheckedRunRequest,
) (umpireruntime.Participant, error) {
	participant, err := a.Binding.NewParticipant(request)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	control := a.control
	a.control = nil
	a.mu.Unlock()
	return &portableGRPCTestParticipant{delegate: participant, control: control}, nil
}

type portableGRPCTestParticipant struct {
	delegate umpireruntime.Participant
	control  *portableGRPCControl
}

func (p *portableGRPCTestParticipant) Prepare(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	if p.control != nil && p.control.mode == portableGRPCBlockPreparation {
		close(p.control.entered)
		select {
		case <-ctx.Done():
		case <-p.control.release:
		}
	}
	return p.delegate.Prepare(ctx, environment, command)
}

func (p *portableGRPCTestParticipant) Realize(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	return p.delegate.Realize(ctx, environment, command)
}

func (p *portableGRPCTestParticipant) Observe(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	if p.control != nil && p.control.mode == portableGRPCBlockObservation {
		close(p.control.entered)
		<-ctx.Done()
	}
	receipt := p.delegate.Observe(ctx, environment, command)
	if p.control != nil && p.control.mode == portableGRPCFailObservation {
		return portableGRPCReceiptWithStatus(receipt, umpireruntime.ReceiptFailed)
	}
	return receipt
}

func (p *portableGRPCTestParticipant) Cleanup(
	ctx context.Context,
	environment umpireruntime.Environment,
	command umpireruntime.Command,
) umpireruntime.Receipt {
	receipt := p.delegate.Cleanup(ctx, environment, command)
	if p.control != nil && p.control.mode == portableGRPCFailCleanup {
		return portableGRPCReceiptWithStatus(receipt, umpireruntime.ReceiptFailed)
	}
	return receipt
}

func portableGRPCReceiptWithStatus(
	receipt umpireruntime.Receipt,
	receiptStatus umpireruntime.ReceiptStatus,
) umpireruntime.Receipt {
	var (
		controlled umpireruntime.Receipt
		err        error
	)
	if receipt.ControlAttempted() {
		controlled, err = umpireruntime.NewControlReceipt(
			receipt.Command(), receiptStatus, receipt.Facts(), receipt.AcquiredResources(), receipt.ReleasedResources(),
		)
	} else {
		controlled, err = umpireruntime.NewReceipt(
			receipt.Command(), receiptStatus, receipt.Facts(), receipt.AcquiredResources(), receipt.ReleasedResources(),
		)
	}
	if err != nil {
		return umpireruntime.Receipt{}
	}
	return controlled
}

type portableGRPCCompletionExecutor struct {
	delegate executorgrpc.Executor
	mu       sync.Mutex
	complete chan struct{}
	lastErr  error
}

func (e *portableGRPCCompletionExecutor) armCompletion() <-chan struct{} {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.complete = make(chan struct{})
	return e.complete
}

func (e *portableGRPCCompletionExecutor) Execute(
	ctx context.Context,
	plan *umpirespb.PortableTestPlan,
) (*umpirespb.ExecutionResult, error) {
	result, err := e.delegate.Execute(ctx, plan)
	e.mu.Lock()
	e.lastErr = err
	complete := e.complete
	e.complete = nil
	e.mu.Unlock()
	if complete != nil {
		close(complete)
	}
	return result, err
}

func (e *portableGRPCCompletionExecutor) lastError() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastErr
}

func requirePortableGRPCSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	await.RequireTrue(t, func() bool {
		select {
		case <-signal:
			return true
		default:
			return false
		}
	}, 10*time.Second, 10*time.Millisecond)
}
