// Package programfixture supplies issued programs to worker transport tests.
package programfixture

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	modelv1 "github.com/temporalio/umpire/hosts/go/gen/umpire/model/v1"
	"github.com/temporalio/umpire/hosts/go/host"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"umpire.local/temporal"
	"umpire.local/temporal/reference/worker"
)

// Issue obtains the entire production manifest without constructing response rows.
func Issue(t *testing.T, namespace, queue, activityType string) []byte {
	t.Helper()
	h, err := host.New(t.Context(), temporal.Guest())
	require.NoError(t, err)
	defer func() { require.NoError(t, h.Close(t.Context())) }()
	start, err := anypb.New(&workflowservice.StartActivityExecutionRequest{
		Namespace: namespace, ActivityId: "fixture-activity", RequestId: "11111111-1111-4111-8111-111111111111",
		ActivityType: &commonpb.ActivityType{Name: activityType}, TaskQueue: &taskqueuepb.TaskQueue{Name: queue},
		StartToCloseTimeout: durationpb.New(20 * time.Second), RetryPolicy: &commonpb.RetryPolicy{MaximumAttempts: 1},
	})
	require.NoError(t, err)
	reset, err := h.Reset(t.Context(), &modelv1.ResetRequest{Configuration: start})
	require.NoError(t, err)
	data, err := proto.Marshal(reset.GetProgram())
	require.NoError(t, err)
	table, err := worker.Decode(data)
	require.NoError(t, err)
	require.Len(t, table.Rows(), 39)
	return data
}

// Select follows an exact task-local path in the issued table, including its heartbeat bit.
func Select(t *testing.T, data []byte, bit bool, terminal string) string {
	t.Helper()
	table, err := worker.Decode(data)
	require.NoError(t, err)
	for _, row := range table.Rows() {
		key := row.Key
		if key.GetAttempt() != 1 || len(key.GetHeartbeatHistory()) != 0 || !row.Response.MessageIs(new(workflowservice.RecordActivityTaskHeartbeatRequest)) {
			continue
		}
		next := table.Match(key.GetScenario(), 1, key.GetActivityType(), bit)
		if next == nil {
			continue
		}
		switch terminal {
		case "completed":
			if !next.MessageIs(new(workflowservice.RespondActivityTaskCompletedRequest)) {
				continue
			}
		case "failed":
			response := new(workflowservice.RespondActivityTaskFailedRequest)
			if next.UnmarshalTo(response) != nil || !response.GetFailure().GetApplicationFailureInfo().GetNonRetryable() {
				continue
			}
		case "canceled":
			if !next.MessageIs(new(workflowservice.RespondActivityTaskCanceledRequest)) {
				continue
			}
		default:
			t.Fatalf("unknown terminal %s", terminal)
		}
		return key.GetScenario()
	}
	t.Fatalf("no issued exact heartbeat path bit=%v terminal=%s", bit, terminal)
	return ""
}
