package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"umpire.local/temporal/reference/worker"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// The shutdown/queue controls choose their typed terminal independently of the
// native converter, then exercise the same production Session binding and ticket.
func terminalTicketSession(t *testing.T, bridge *worker.Bridge, task *workflowservice.PollActivityTaskQueueResponse, terminal *anypb.Any) *worker.ActivitySession {
	t.Helper()
	identity := worker.IdentityFromPoll(heartbeatNamespace, "production-queue", task)
	key, err := anypb.New(&temporalv1.ActivityTaskKey{Scenario: identity.Scenario, Attempt: identity.Attempt, ActivityType: identity.ActivityType})
	require.NoError(t, err)
	program, err := anypb.New(&temporalv1.WorkerProgram{Namespace: heartbeatNamespace, TaskQueue: "production-queue", Rules: []*temporalv1.Rule{{Key: key, Respond: []*anypb.Any{proto.CloneOf(terminal)}}}})
	require.NoError(t, err)
	encoded, err := proto.Marshal(program)
	require.NoError(t, err)
	table, err := worker.Decode(encoded)
	require.NoError(t, err)
	session, err := table.NewSession(bridge, identity, "", nil)
	require.NoError(t, err)
	return session
}

func TestProductionSDKSessionTerminalTicketPrecedesHandlerReturn(t *testing.T) {
	probe := newSDKActivityProbe()
	fixture := startSDKContinuation(t, productionProgram(t, nil, "completed"), nil, nil, sdkContinuationOptions{hold: "terminal", interceptor: probe})
	heartbeatReceive(fixture.ctx, t, probe.handlerReturned)
	started := heartbeatReceive(fixture.ctx, t, fixture.capture.blocked)
	require.Equal(t, workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName, started.Method)
	require.NoError(t, fixture.bridge.Err(), "actual terminal invocation requires the already registered ticket")
	require.Empty(t, fixture.service.terminal, "SDK return alone cannot produce a captured acknowledgment")
	select {
	case <-fixture.done:
		t.Fatal("follower finished before actual terminal transport")
	default:
	}
	close(fixture.capture.release)
	checkHeartbeatTerminal(t, heartbeatReceive(fixture.ctx, t, fixture.service.terminal), "completed")
	for {
		actual := heartbeatReceive(fixture.ctx, t, fixture.capture.completed)
		if actual.Method == workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName {
			require.Nil(t, actual.CallError)
			require.True(t, actual.Sent)
			require.True(t, actual.Answered)
			break
		}
	}
	fixture.stop()
	require.NoError(t, fixture.err)
	require.NoError(t, fixture.closeErr)
}

func TestProductionSDKSessionRejectsActualUndeclaredBit(t *testing.T) {
	fixture := startSDKContinuation(t, productionProgram(t, []bool{false}, "completed"), nil, nil)
	call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
	call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: true}}
	heartbeatReceive(fixture.ctx, t, fixture.done)
	require.ErrorContains(t, fixture.err, "outside admitted domain")
	fixture.stop()
	requireNoCompletedTerminal(t, fixture)
}

func TestProductionSDKSessionContinuesFalseAfterTrueNotification(t *testing.T) {
	probe := newSDKActivityProbe()
	bits := []bool{true, false, true, false}
	fixture := startSDKContinuation(t, productionProgram(t, bits, "completed"), nil, nil, sdkContinuationOptions{interceptor: probe})
	for index, bit := range bits {
		call := heartbeatReceive(fixture.ctx, t, fixture.service.heartbeats)
		var detail int
		require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(call.request.GetDetails(), &detail))
		require.Equal(t, index, detail, "one actual capture selects one exact history row")
		require.Equal(t, heartbeatNamespace, call.request.GetNamespace())
		require.Equal(t, heartbeatToken, call.request.GetTaskToken())
		require.Empty(t, fixture.service.terminal)
		call.reply <- heartbeatReply{response: &workflowservice.RecordActivityTaskHeartbeatResponse{CancelRequested: bit}}
		if index == 0 {
			require.True(t, temporal.IsCanceledError(heartbeatReceive(fixture.ctx, t, probe.canceled)))
		}
	}
	checkHeartbeatTerminal(t, heartbeatReceive(fixture.ctx, t, fixture.service.terminal), "completed")
	heartbeatReceive(fixture.ctx, t, probe.handlerReturned)
	fixture.stop()
	require.NoError(t, fixture.err)
	require.NoError(t, fixture.closeErr)
}
