package tests

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/nexus/nexustest"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tests/umpire3/execution/participant"
)

func TestUmpire3ParticipantProcessCrashAndRestartResumesRealSDKProgram(t *testing.T) {
	nexusEnv := newNexusTestEnv(t, true)
	env := nexusEnv.TestEnv
	nexusEndpoint := nexusEnv.createRandomExternalNexusServer(context.Background(), t, nexustest.Handler{
		OnStartOperation: func(
			context.Context,
			string,
			string,
			*nexus.LazyValue,
			nexus.StartOperationOptions,
		) (nexus.HandlerStartOperationResult[any], error) {
			return &nexus.HandlerStartOperationResultSync[any]{Value: "umpire3-process-ok"}, nil
		},
	})
	program := participant.Program{
		FormatVersion: participant.FormatVersion,
		Identifier:    "process-restart",
		Commands: []participant.Command{
			{
				Identifier: "blocking", Kind: participant.CommandTimer,
				Response: participant.ResponseBlocking, MaxBlockNanos: int64(3 * time.Second),
			},
			{Identifier: "workflow", Kind: participant.CommandWorkflow, Response: participant.ResponseSynchronous},
			{Identifier: "activity", Kind: participant.CommandActivity, Response: participant.ResponseAsynchronous},
			{Identifier: "nexus", Kind: participant.CommandNexus, Response: participant.ResponseDeferred},
			{Identifier: "update", Kind: participant.CommandUpdate, Response: participant.ResponseSynchronous},
			{Identifier: "signal", Kind: participant.CommandSignal, Response: participant.ResponseSynchronous},
			{Identifier: "query", Kind: participant.CommandQuery, Response: participant.ResponseSynchronous},
			{Identifier: "timer", Kind: participant.CommandTimer, Response: participant.ResponseAsynchronous},
			{Identifier: "child", Kind: participant.CommandChild, Response: participant.ResponseDeferred},
			{Identifier: "cancellation", Kind: participant.CommandCancellation, Response: participant.ResponseSynchronous},
			{Identifier: "retry", Kind: participant.CommandRetry, Response: participant.ResponseSynchronous},
			{Identifier: "failure", Kind: participant.CommandFailure, Response: participant.ResponseFailure},
		},
	}
	encoded, err := json.Marshal(program)
	require.NoError(t, err)
	directory := t.TempDir()
	programPath := filepath.Join(directory, "program.json")
	reportPath := filepath.Join(directory, "report.json")
	binaryPath := filepath.Join(directory, "umpire3-participant")
	require.NoError(t, os.WriteFile(programPath, encoded, 0o600))
	build := exec.CommandContext(context.Background(), "go", "build", "-tags", "test_dep", "-o", binaryPath,
		"./umpire3/cmd/umpire3-participant")
	buildOutput, err := build.CombinedOutput()
	require.NoError(t, err, string(buildOutput))

	workflowID := "umpire3-process-restart"
	taskQueue := "umpire3-process-restart-" + env.NamespaceID().String()
	command := []string{
		binaryPath,
		"-program", programPath,
		"-output", reportPath,
		"-address", env.FrontendGRPCAddress(),
		"-namespace", env.Namespace().String(),
		"-task-queue", taskQueue,
		"-workflow-id", workflowID,
		"-nexus-endpoint", nexusEndpoint,
		"-nexus-service", "service",
		"-nexus-operation", "operation",
		"-timeout", "2m",
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	first := exec.CommandContext(ctx, command[0], command[1:]...)
	require.NoError(t, first.Start())
	t.Cleanup(func() {
		if first.Process != nil {
			_ = first.Process.Kill()
		}
	})
	await.RequireTrue(t, func() bool {
		return umpire3HistoryContains(t, env, workflowID,
			enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED)
	}, 30*time.Second, 100*time.Millisecond)
	description, err := env.FrontendClient().DescribeWorkflowExecution(context.Background(),
		&workflowservice.DescribeWorkflowExecutionRequest{
			Namespace: env.Namespace().String(), Execution: &commonpb.WorkflowExecution{WorkflowId: workflowID},
		})
	require.NoError(t, err)
	initialRunID := description.GetWorkflowExecutionInfo().GetExecution().GetRunId()
	require.NotEmpty(t, initialRunID)

	require.NoError(t, first.Process.Kill())
	require.Error(t, first.Wait())
	second := exec.CommandContext(ctx, command[0], command[1:]...)
	output, err := second.CombinedOutput()
	require.NoError(t, err, string(output))

	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	var report struct {
		Results         []participant.Result `json:"results"`
		CleanupComplete bool                 `json:"cleanupComplete"`
	}
	require.NoError(t, json.Unmarshal(reportBytes, &report))
	require.True(t, report.CleanupComplete)
	require.Len(t, report.Results, len(program.Commands))
	for index, command := range program.Commands {
		require.Equal(t, command.Identifier, report.Results[index].CommandID)
		require.NotEmpty(t, report.Results[index].SourceIdentity)
		require.NotEmpty(t, report.Results[index].WorkflowID)
		require.Equal(t, initialRunID, report.Results[index].RunID)
		require.NotEmpty(t, report.Results[index].Lineage)
		require.NotEmpty(t, report.Results[index].PayloadDigest)
	}
	require.Equal(t, "accepted", report.Results[2].Status)
	require.Equal(t, "deferred", report.Results[3].Status)
	require.Equal(t, "accepted", report.Results[7].Status)
	require.Equal(t, "deferred", report.Results[8].Status)
	require.Equal(t, "failed", report.Results[len(report.Results)-1].Status)
}

func umpire3HistoryContains(
	t *testing.T,
	env *testcore.TestEnv,
	workflowID string,
	eventType enumspb.EventType,
) bool {
	t.Helper()
	response, err := env.FrontendClient().GetWorkflowExecutionHistory(context.Background(),
		&workflowservice.GetWorkflowExecutionHistoryRequest{
			Namespace:              env.Namespace().String(),
			Execution:              &commonpb.WorkflowExecution{WorkflowId: workflowID},
			HistoryEventFilterType: enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
			SkipArchival:           true,
		})
	if err != nil {
		return false
	}
	for _, event := range response.GetHistory().GetEvents() {
		if event.GetEventType() == eventType {
			return true
		}
	}
	return false
}
