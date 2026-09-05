//go:build test_dep && integration

package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/caseartifact"
	umpiretemporal "go.temporal.io/server/tools/umpire/temporal"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func TestUmpireAsyncNexusCase(t *testing.T) {
	env := newUmpireTestEnvironment(t)
	caseSource := loadUmpireCase(t, "async-nexus")
	catalog, err := umpiretemporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := umpire.ProfileSpec{
		Identity: "temporal-async-nexus-case-profile",
		Catalog:  catalog,
		Roles: []umpire.RolePolicy{
			{
				ID: "temporal.workflow-service", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT,
				Methods: []string{
					"/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					"/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory",
				},
				ReservationCarriers: []umpire.ReservationCarrierPolicy{{
					Method: "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					Shapes: []umpire.ReservationCarrierShape{
						{Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, MaximumCount: 1},
						{Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, MaximumCount: 1},
					},
				}},
			},
			{ID: "temporal.worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT},
		},
		Capabilities: []umpire.Capability{
			umpire.InvokeRPC, umpire.AwaitSlot, umpire.CompleteNexusOperation,
			umpire.StartNexusOperation, umpire.Await, umpire.Finish, umpire.RespondNexus,
		},
		ProgramLimits:  proto.CloneOf(caseSource.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(caseSource.GetContract().GetLimits()),
	}
	prepared, err := umpire.PrepareCase(caseSource, profile)
	require.NoError(t, err)
	_, err = env.RegisterNamespace(namespace.Name("default"), 1, enumspb.ARCHIVAL_STATE_DISABLED, "", "")
	require.NoError(t, err)
	caseClient, err := client.Dial(client.Options{HostPort: env.FrontendGRPCAddress(), Namespace: "default"})
	require.NoError(t, err)
	t.Cleanup(caseClient.Close)

	const taskQueue = "umpire-async-nexus-workflow-queue"
	const endpointName = "umpire-async-nexus-endpoint"
	const cleanupTimeout = 5 * time.Second
	created, err := env.OperatorClient().CreateNexusEndpoint(env.Context(), &operatorservice.CreateNexusEndpointRequest{
		Spec: &nexus.EndpointSpec{
			Name: endpointName,
			Target: &nexus.EndpointTarget{Variant: &nexus.EndpointTarget_Worker_{
				Worker: &nexus.EndpointTarget_Worker{Namespace: "default", TaskQueue: taskQueue},
			}},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		_, err := env.OperatorClient().DeleteNexusEndpoint(ctx, &operatorservice.DeleteNexusEndpointRequest{
			Id: created.GetEndpoint().GetId(), Version: created.GetEndpoint().GetVersion(),
		})
		require.NoError(t, err)
	})

	host, err := umpiretemporal.New(umpiretemporal.Options{
		Profile: profile,
		ServerEndpoints: map[string]umpiretemporal.Endpoint{
			"temporal.workflow-service": {Target: env.FrontendGRPCAddress(), Credentials: insecure.NewCredentials()},
		},
		SystemCallbackBaseURL: "http://" + env.HttpAPIAddress(),
		SDKClient:             caseClient, Namespace: "default", WorkerRoleID: "temporal.worker",
		TaskQueues:        []umpiretemporal.RoleBinding{{RoleID: "temporal.task-queue", Value: taskQueue}},
		NexusEndpoints:    []umpiretemporal.RoleBinding{{RoleID: "temporal.nexus-endpoint", Value: endpointName}},
		WorkerStopTimeout: cleanupTimeout,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		require.NoError(t, host.Close(ctx))
	})

	run, verdict, err := prepared.Run(env.Context(), host)
	require.NoError(t, err)
	require.Equal(t, umpirespb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	require.Equal(t, umpirespb.VERDICT_KIND_SATISFIED, verdict.GetKind())
	require.True(t, proto.Equal(verdict, run.GetVerdict()))
	require.Len(t, verdict.GetRules(), 1)
	require.Equal(t, umpirespb.RULE_VERDICT_KIND_SATISFIED, verdict.GetRules()[0].GetKind())
	require.Len(t, verdict.GetRules()[0].GetSupportingEventSequences(), 3)
	requireHistoryOnlyEvidence(t, run, verdict.GetSupportingEventSequences())
}

func loadUmpireCase(t testing.TB, name string) *umpirespb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "tools", "umpire", "temporal", "testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := caseartifact.DecodeProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

func requireHistoryOnlyEvidence(t testing.TB, run *umpirespb.Run, sequences []int64) {
	t.Helper()
	for _, sequence := range sequences {
		require.Positive(t, sequence)
		require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
		event := run.GetEvents()[sequence-1]
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		require.NotEmpty(t, event.GetObservations())
	}
}
