//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/executor"
	"go.temporal.io/server/tools/umpire/executorhttp"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
	"google.golang.org/protobuf/proto"
)

func TestUmpirePortableCanaryExecutor(t *testing.T) {
	env, attachedFactory := newUmpireTestEnvironment(t)
	noToolchainPath := t.TempDir()
	t.Setenv("PATH", noToolchainPath)

	factory := &recordingEnvironmentFactory{delegate: attachedFactory}
	adapter := &recordingNexusAdapter{factory: factory}
	server := httptest.NewServer(executorhttp.New(executor.New(adapter)))
	t.Cleanup(server.Close)

	normal := executePortableFixture(t, env.Context(), server.URL, "normal")
	requirePortableResult(t, normal, portableResultSnapshot{
		tooling:            umpirespb.TOOLING_STATUS_SUCCEEDED,
		operational:        umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		observation:        umpirespb.OBSERVATION_STATUS_ACCEPTED,
		implementationLink: umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED,
		semantic:           umpirespb.EVALUATION_STATUS_SATISFIED,
		cleanup:            umpirespb.CLEANUP_STATUS_COMPLETE,
		decision:           umpirespb.CANARY_DECISION_PASS,
		property:           umpirespb.SEMANTIC_STATUS_SATISFIED,
		clauses: []clauseSnapshot{
			{"workflow-nexus.property.clause.delivery", umpirespb.SEMANTIC_STATUS_SATISFIED},
			{"workflow-nexus.property.clause.ownership", umpirespb.SEMANTIC_STATUS_SATISFIED},
			{"workflow-nexus.property.clause.uniqueness", umpirespb.SEMANTIC_STATUS_SATISFIED},
		},
	})
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())
	require.Len(t, factory.identities, 1)
	require.Len(t, adapter.requests, 1)

	crossed := executePortableFixturePair(t, env.Context(), server.URL, "normal", "duplicate-delivery")
	require.Equal(t, portableResultSnapshot{
		tooling:            umpirespb.TOOLING_STATUS_INVALID_INPUT,
		operational:        umpirespb.OPERATIONAL_STATUS_INCOMPLETE,
		observation:        umpirespb.OBSERVATION_STATUS_UNKNOWN,
		implementationLink: umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
		semantic:           umpirespb.EVALUATION_STATUS_INCOMPLETE,
		cleanup:            umpirespb.CLEANUP_STATUS_INCOMPLETE,
		decision:           umpirespb.CANARY_DECISION_INCONCLUSIVE,
	}, portableResultSnapshot{
		tooling:            crossed.GetToolingStatus(),
		operational:        crossed.GetOperationalStatus(),
		observation:        crossed.GetObservation().GetStatus(),
		implementationLink: crossed.GetImplementationLink().GetStatus(),
		semantic:           crossed.GetSemanticStatus(),
		cleanup:            crossed.GetCleanupStatus(),
		decision:           crossed.GetDecision(),
	})
	require.Len(t, factory.identities, 1)
	require.Len(t, adapter.requests, 1)

	duplicateResult := executePortableFixture(t, env.Context(), server.URL, "duplicate-delivery")
	requirePortableResult(t, duplicateResult, portableResultSnapshot{
		tooling:            umpirespb.TOOLING_STATUS_SUCCEEDED,
		operational:        umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		observation:        umpirespb.OBSERVATION_STATUS_ACCEPTED,
		implementationLink: umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED,
		semantic:           umpirespb.EVALUATION_STATUS_VIOLATED,
		cleanup:            umpirespb.CLEANUP_STATUS_COMPLETE,
		decision:           umpirespb.CANARY_DECISION_FAIL,
		property:           umpirespb.SEMANTIC_STATUS_VIOLATED,
		clauses: []clauseSnapshot{
			{"workflow-nexus.property.clause.delivery", umpirespb.SEMANTIC_STATUS_SATISFIED},
			{"workflow-nexus.property.clause.ownership", umpirespb.SEMANTIC_STATUS_SATISFIED},
			{"workflow-nexus.property.clause.uniqueness", umpirespb.SEMANTIC_STATUS_VIOLATED},
		},
	})
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())
	require.Len(t, factory.identities, 2)
	require.Len(t, adapter.requests, 2)
	require.NotEqual(t, normal.GetRunIdentity(), duplicateResult.GetRunIdentity())
	require.NotEqual(t, factory.identities[0].TaskQueue, factory.identities[1].TaskQueue)
	require.Equal(t, factory.identities[0].Namespace, factory.identities[1].Namespace)
	require.Equal(t, factory.identities[0].Endpoint, factory.identities[1].Endpoint)
	requireFreshCorrelations(t, adapter.requests[0], adapter.requests[1])

	workerEndpoints := make([]string, 0, len(adapter.requests))
	for _, request := range adapter.requests {
		_, err := env.SdkClient().DescribeWorkflowExecution(
			env.Context(), workflowCorrelation(t, request), "",
		)
		require.NoError(t, err, "the resident executor must use the disposable NewEnv cluster")
		workerEndpoints = append(workerEndpoints, workflowNexusEndpoint(
			t, env.Context(), env.SdkClient(), workflowCorrelation(t, request),
		))
	}
	require.NotEqual(t, workerEndpoints[0], workerEndpoints[1])
	for _, tool := range []string{"go", "lake", "lean", "make", "mise", "sh"} {
		_, err := exec.LookPath(tool)
		require.Error(t, err, "the tagged runtime must not have a toolchain executable available")
	}
	require.Equal(t, noToolchainPath, os.Getenv("PATH"))
}

type portableResultSnapshot struct {
	tooling            umpirespb.ToolingStatus
	operational        umpirespb.OperationalStatus
	observation        umpirespb.ObservationStatus
	implementationLink umpirespb.ImplementationLinkStatus
	semantic           umpirespb.EvaluationStatus
	cleanup            umpirespb.CleanupStatus
	decision           umpirespb.CanaryDecision
	property           umpirespb.SemanticStatus
	clauses            []clauseSnapshot
}

type clauseSnapshot struct {
	definitionID string
	status       umpirespb.SemanticStatus
}

func requirePortableResult(
	t *testing.T,
	result *umpirespb.EvaluationResult,
	want portableResultSnapshot,
) {
	t.Helper()
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetRunIdentity())
	require.NotNil(t, result.GetObservation())
	require.NotNil(t, result.GetImplementationLink())
	require.Len(t, result.GetProperties(), 1)
	require.Equal(t, want, portableResultSnapshot{
		tooling:            result.GetToolingStatus(),
		operational:        result.GetOperationalStatus(),
		observation:        result.GetObservation().GetStatus(),
		implementationLink: result.GetImplementationLink().GetStatus(),
		semantic:           result.GetSemanticStatus(),
		cleanup:            result.GetCleanupStatus(),
		decision:           result.GetDecision(),
		property:           result.GetProperties()[0].GetStatus(),
		clauses:            clauseSnapshots(result),
	})
	require.NotNil(t, result.GetObservation().GetTrace())
	require.NotEmpty(t, result.GetObservation().GetEvidenceLinks())
	require.NotNil(t, result.GetImplementationLink().GetTrace())
	require.NotNil(t, result.GetWork())
	require.Positive(t, result.GetWork().GetTotal())
	require.LessOrEqual(t, result.GetWork().GetTotal(), result.GetWork().GetLimit())
	require.NotEmpty(t, result.GetKnownGaps())
	require.Empty(t, result.GetDiagnostics())
	require.Empty(t, result.GetObservation().GetDiagnostics())
	require.Empty(t, result.GetImplementationLink().GetDiagnostics())
	require.Empty(t, result.GetProperties()[0].GetDiagnostics())
}

func clauseSnapshots(result *umpirespb.EvaluationResult) []clauseSnapshot {
	clauses := result.GetProperties()[0].GetClauses()
	snapshots := make([]clauseSnapshot, 0, len(clauses))
	for _, clause := range clauses {
		snapshots = append(snapshots, clauseSnapshot{
			definitionID: clause.GetClauseDefinitionId(),
			status:       clause.GetStatus(),
		})
	}
	return snapshots
}

type recordingEnvironmentFactory struct {
	delegate   umpireruntime.EnvironmentFactory
	identities []local.Identities
}

func (f *recordingEnvironmentFactory) Prepare(
	ctx context.Context,
	request umpireruntime.CheckedRunRequest,
	command umpireruntime.Command,
) (umpireruntime.Environment, umpireruntime.Receipt) {
	environment, receipt := f.delegate.Prepare(ctx, request, command)
	if attached, ok := local.AsEnvironment(environment); ok {
		f.identities = append(f.identities, attached.Identities())
	}
	return environment, receipt
}

func requireNoNexusEndpoints(
	t *testing.T,
	ctx context.Context,
	operatorClient operatorservice.OperatorServiceClient,
) {
	t.Helper()
	require.Eventually(t, func() bool {
		response, err := operatorClient.ListNexusEndpoints(
			ctx, &operatorservice.ListNexusEndpointsRequest{},
		)
		require.NoError(t, err)
		return len(response.GetEndpoints()) == 0
	}, time.Second, 10*time.Millisecond)
}

func requireFreshCorrelations(
	t *testing.T,
	first umpireruntime.CheckedRunRequest,
	second umpireruntime.CheckedRunRequest,
) {
	t.Helper()
	firstCorrelations := first.Correlations()
	secondCorrelations := second.Correlations()
	require.Len(t, firstCorrelations, 5)
	require.Len(t, secondCorrelations, 5)
	for index := range firstCorrelations {
		require.Equal(t, firstCorrelations[index].Kind(), secondCorrelations[index].Kind())
		require.NotEmpty(t, firstCorrelations[index].Identity())
		require.NotEmpty(t, secondCorrelations[index].Identity())
		require.NotEqual(t, firstCorrelations[index].Identity(), secondCorrelations[index].Identity())
	}
}

func workflowNexusEndpoint(
	t *testing.T,
	ctx context.Context,
	sdkClient client.Client,
	workflowID string,
) string {
	t.Helper()
	history := sdkClient.GetWorkflowHistory(
		ctx, workflowID, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	for history.HasNext() {
		event, err := history.Next()
		require.NoError(t, err)
		if event.GetEventType() == enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED {
			endpoint := event.GetNexusOperationScheduledEventAttributes().GetEndpoint()
			require.NotEmpty(t, endpoint)
			return endpoint
		}
	}
	require.FailNow(t, "workflow history has no Nexus operation endpoint")
	return ""
}

type recordingNexusAdapter struct {
	nexus.Binding
	factory  umpireruntime.EnvironmentFactory
	requests []umpireruntime.CheckedRunRequest
}

func (a *recordingNexusAdapter) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	return a.factory
}

func (a *recordingNexusAdapter) CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	request, err := a.Binding.CheckRequest(admitted, runIdentity)
	if err == nil {
		a.requests = append(a.requests, request)
	}
	return request, err
}

func workflowCorrelation(t *testing.T, request umpireruntime.CheckedRunRequest) string {
	t.Helper()
	for _, correlation := range request.Correlations() {
		if correlation.Kind() == umpireruntime.CorrelationWorkflow {
			return correlation.Identity()
		}
	}
	require.FailNow(t, "checked request has no workflow correlation")
	return ""
}

func executePortableFixture(
	t *testing.T,
	ctx context.Context,
	serverURL string,
	fixture string,
) *umpirespb.EvaluationResult {
	t.Helper()
	return executePortableFixturePair(t, ctx, serverURL, fixture, fixture)
}

func executePortableFixturePair(
	t *testing.T,
	ctx context.Context,
	serverURL string,
	contractFixture string,
	inputFixture string,
) *umpirespb.EvaluationResult {
	t.Helper()
	contract, err := os.ReadFile(filepath.Join(
		"..", "tools", "umpire", "portableevaluation", "testdata", contractFixture, "contract.pb",
	))
	require.NoError(t, err)
	inputRoot := filepath.Join(
		"..", "tools", "umpire", "temporal", "nexus", "testdata", "caller-closure-input-set",
	)
	if inputFixture == "duplicate-delivery" {
		inputRoot = filepath.Join(
			"..", "tools", "umpire", "temporal", "nexus", "testdata",
			"caller-closure-duplicate-delivery-input-set",
		)
	}
	experiment, err := os.ReadFile(filepath.Join(inputRoot, "artifacts", "experiment.json"))
	require.NoError(t, err)
	configuration, err := os.ReadFile(filepath.Join(inputRoot, "artifacts", "runtime-configuration.json"))
	require.NoError(t, err)
	requestBytes, err := (proto.MarshalOptions{Deterministic: true}).Marshal(&umpirespb.ExecuteRequest{
		EvaluationContract: contract,
		Input: &umpirespb.EvaluationInput{
			Experiment: experiment, RuntimeConfig: configuration,
		},
	})
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		serverURL+executorhttp.ExecutePath,
		bytes.NewReader(requestBytes),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", executorhttp.ProtobufContentType)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
	require.Equal(t, http.StatusOK, response.StatusCode)
	responseBytes, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	require.NoError(t, err)
	var executionResponse umpirespb.ExecuteResponse
	require.NoError(t, (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(
		responseBytes,
		&executionResponse,
	))
	require.NotNil(t, executionResponse.GetResult())
	return executionResponse.GetResult()
}
