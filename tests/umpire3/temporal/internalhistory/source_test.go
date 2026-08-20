package internalhistory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	umpire3temporal "go.temporal.io/server/tests/umpire3/temporal"
	"google.golang.org/grpc"
)

type fakeHistoryClient struct {
	requests  []*historyservice.GetWorkflowExecutionHistoryRequest
	responses []*historyservice.GetWorkflowExecutionHistoryResponse
}

func (c *fakeHistoryClient) GetWorkflowExecutionHistory(
	_ context.Context,
	request *historyservice.GetWorkflowExecutionHistoryRequest,
	_ ...grpc.CallOption,
) (*historyservice.GetWorkflowExecutionHistoryResponse, error) {
	c.requests = append(c.requests, request)
	response := c.responses[0]
	c.responses = c.responses[1:]
	return response, nil
}

func TestSourceReadsAllInternalHistoryPages(t *testing.T) {
	t.Parallel()

	client := &fakeHistoryClient{responses: []*historyservice.GetWorkflowExecutionHistoryResponse{
		{Response: &workflowservice.GetWorkflowExecutionHistoryResponse{
			History: &historypb.History{Events: []*historypb.HistoryEvent{{
				EventId: 1, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			}}},
			NextPageToken: []byte("next"),
		}},
		{Response: &workflowservice.GetWorkflowExecutionHistoryResponse{
			History: &historypb.History{Events: []*historypb.HistoryEvent{{
				EventId: 2, EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
			}}},
		}},
	}}
	source, err := New(client, "namespace-id", "cluster-a")
	require.NoError(t, err)

	history, err := source.ReadHistory(context.Background(), umpire3temporal.HistoryRequest{
		Namespace: "namespace", WorkflowID: "workflow", RunID: "run",
	})
	require.NoError(t, err)
	require.Equal(t, "temporal-history-service", history.Source)
	require.Equal(t, "cluster-a/history-service", history.SourceIdentity)
	require.Len(t, history.Events, 2)
	require.Equal(t, int64(2), history.Events[1].ID)
	require.Equal(t, []byte("next"), client.requests[1].GetRequest().GetNextPageToken())
	require.Equal(t, &commonpb.WorkflowExecution{WorkflowId: "workflow", RunId: "run"}, client.requests[0].GetRequest().GetExecution())
}
