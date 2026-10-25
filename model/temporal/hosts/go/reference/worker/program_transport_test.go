package worker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	deploymentpb "go.temporal.io/api/deployment/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
)

func completedRPC(t *testing.T, request, response proto.Message) RPC {
	t.Helper()
	name := string(request.ProtoReflect().Descriptor().Name())
	return RPC{Invocation: 1, Sequence: 2, Stage: RPCCompleted, Method: "/temporal.api.workflowservice.v1.WorkflowService/" + strings.TrimSuffix(name, "Request"), Request: historyAny(t, request), Response: historyAny(t, response), Sent: true, Answered: true}
}
func TestHeartbeatResultRequiresSuccessfulActualResponse(t *testing.T) {
	request := &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: "ns", TaskToken: []byte("token")}
	for _, bit := range []bool{false, true} {
		got, err := HeartbeatResult(historyAny(t, request), completedRPC(t, request, &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: bit}))
		require.NoError(t, err)
		require.Equal(t, bit, got)
	}
	for _, test := range []struct {
		name string
		edit func(*RPC)
	}{
		{name: "missing-completion", edit: func(rpc *RPC) { rpc.Stage = RPCStarted }},
		{name: "missing-response", edit: func(rpc *RPC) { rpc.Response = nil }},
		{name: "transport-error-zero-reply", edit: func(rpc *RPC) { rpc.TransportError = "local deadline" }},
		{name: "server-error-zero-reply", edit: func(rpc *RPC) { rpc.ServerError = &realizationv1.Error{Code: 7} }},
		{name: "ambiguous-call-error-zero-reply", edit: func(rpc *RPC) { rpc.CallError = historyAny(t, &statuspb.Status{Code: 13}) }},
		{name: "unanswered", edit: func(rpc *RPC) { rpc.Answered = false }},
		{name: "wrong-response-type", edit: func(rpc *RPC) { rpc.Response = historyAny(t, &workflowservice.RespondActivityTaskCompletedResponse{}) }},
		{name: "paused", edit: func(rpc *RPC) {
			rpc.Response = historyAny(t, &workflowservice.RecordActivityTaskHeartbeatResponse{ActivityPaused: true})
		}},
		{name: "reset", edit: func(rpc *RPC) {
			rpc.Response = historyAny(t, &workflowservice.RecordActivityTaskHeartbeatResponse{ActivityReset: true})
		}},
		{name: "unknown-response-field", edit: func(rpc *RPC) {
			response := &workflowservice.RecordActivityTaskHeartbeatResponse{}
			response.ProtoReflect().SetUnknown([]byte{0x20, 1})
			rpc.Response = historyAny(t, response)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			rpc := completedRPC(t, request, &workflowservice.RecordActivityTaskHeartbeatResponse{})
			test.edit(&rpc)
			_, err := HeartbeatResult(historyAny(t, request), rpc)
			require.Error(t, err)
		})
	}
}
func TestTerminalResultChecksChosenPayloadAndAcknowledgment(t *testing.T) {
	request := &workflowservice.RespondActivityTaskFailedRequest{Namespace: "ns", TaskToken: []byte("token"), Failure: &failurepb.Failure{Message: "chosen", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "proof", NonRetryable: true}}}}
	expected := historyAny(t, request)
	actual := proto.CloneOf(request)
	actual.Identity = "SDK-worker"
	actual.DeploymentOptions = &deploymentpb.WorkerDeploymentOptions{DeploymentName: "SDK-deployment"}
	require.NoError(t, TerminalResult(expected, completedRPC(t, actual, &workflowservice.RespondActivityTaskFailedResponse{})))
	for _, test := range []struct {
		name string
		edit func(*RPC)
		want string
	}{
		{name: "wrong-terminal-method", edit: func(rpc *RPC) {
			rpc.Method = workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
		}, want: "terminal kind"},
		{name: "crossed-token", edit: func(rpc *RPC) {
			changed := proto.CloneOf(request)
			changed.TaskToken = []byte("crossed")
			rpc.Request = historyAny(t, changed)
		}, want: "payload or assignment"},
		{name: "wrong-payload", edit: func(rpc *RPC) {
			changed := proto.CloneOf(request)
			changed.Failure.Message = "wrong"
			rpc.Request = historyAny(t, changed)
		}, want: "payload or assignment"},
		{name: "wrong-response-type", edit: func(rpc *RPC) { rpc.Response = historyAny(t, &workflowservice.RespondActivityTaskCanceledResponse{}) }, want: "mismatched message type"},
		{name: "validation-failure", edit: func(rpc *RPC) {
			rpc.Response = historyAny(t, &workflowservice.RespondActivityTaskFailedResponse{Failures: []*failurepb.Failure{{Message: "validation rejected"}}})
		}, want: "validation failures"},
		{name: "missing-acknowledgment", edit: func(rpc *RPC) { rpc.Response = nil }, want: "lacks request or response"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rpc := completedRPC(t, request, &workflowservice.RespondActivityTaskFailedResponse{})
			test.edit(&rpc)
			require.ErrorContains(t, TerminalResult(expected, rpc), test.want)
		})
	}
}
