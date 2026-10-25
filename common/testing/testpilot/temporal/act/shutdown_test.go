package act

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

func TestCloseCancelsWorkerBeforePollHold(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	server := newFakeServer(t, func(string) ([]byte, error) { t.Error("held poll reached server"); return nil, nil })
	entered := make(chan struct{})
	parentAtReturn := make(chan error, 1)
	opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
		close(entered)
		_, err := conn.WorkflowService().PollActivityTaskQueue(ctx, &workflowservice.PollActivityTaskQueueRequest{Namespace: "namespace"})
		parentErr := ctx.Err()
		parentAtReturn <- parentErr
		if parentErr != nil {
			return nil
		}
		return err
	}, func(context.Context, WorkerRPC) error { return nil })
	driver := testDriver(t, server.address, opts)
	workerReceive(ctx, t, entered)
	originalCancel := driver.polls.cancel
	paused := make(chan error, 1)
	release := make(chan struct{})
	driver.polls.cancel = func() {
		originalCancel()
		select {
		case parentErr := <-parentAtReturn:
			paused <- parentErr
		case <-ctx.Done():
			paused <- ctx.Err()
		}
		select {
		case <-release:
		case <-ctx.Done():
		}
	}
	closed := make(chan error, 1)
	go func() { closed <- driver.Close() }()
	parentErr := workerReceive(ctx, t, paused)
	close(release)
	closeErr := workerReceive(ctx, t, closed)
	require.ErrorIs(t, parentErr, context.Canceled, "poll must observe worker shutdown while Close is paused")
	require.NoError(t, closeErr, "normal raw-shaped canceled poll must not become DriverFault")
	require.Empty(t, driver.workerCapture.calls, "the poll hold canceled before actual invocation")
}

func TestCloseRetainsLateFollowValidation(t *testing.T) {
	validation := errors.New("actual terminal acknowledgment validation failed")
	for _, test := range []struct {
		name     string
		returned error
		fault    bool
	}{
		{name: "canceled", returned: context.Canceled},
		{name: "wrapped-deadline", returned: fmt.Errorf("unfinished wait: %w", context.DeadlineExceeded)},
		{name: "cancellation-only-join", returned: errors.Join(context.Canceled, context.DeadlineExceeded)},
		{name: "validation", returned: validation, fault: true},
		{name: "mixed-join", returned: errors.Join(context.Canceled, validation), fault: true},
		{name: "wrapped-mixed-join", returned: fmt.Errorf("terminal shutdown: %w", errors.Join(context.DeadlineExceeded, validation)), fault: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			guard, cancelGuard := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancelGuard()
			response := &workflowservice.RespondActivityTaskFailedResponse{}
			if test.fault {
				response.Failures = []*failurepb.Failure{{Message: "actual-invalid-ack"}}
			}
			server := newFakeServer(t, func(string) ([]byte, error) { return marshal(t, response), nil })
			acknowledged := make(chan struct{})
			chosen := &workflowservice.RespondActivityTaskFailedRequest{Namespace: "namespace", TaskToken: []byte("late-token"), Failure: &failurepb.Failure{Message: "chosen-failure"}}
			opts := workerTestOptions(func(ctx context.Context, conn client.Client, _ []byte) error {
				actual, err := conn.WorkflowService().RespondActivityTaskFailed(ctx, chosen)
				if err != nil {
					return err
				}
				if !proto.Equal(response, actual) {
					return errors.New("fixture acknowledgment differs")
				}
				close(acknowledged)
				<-ctx.Done()
				return test.returned
			}, func(context.Context, WorkerRPC) error { return nil })
			driver := testDriver(t, server.address, opts)
			workerReceive(guard, t, acknowledged)
			require.NoError(t, driver.workerContext.Err(), "Follow must defer its result until Close cancels Run")
			err := driver.Close()
			if !test.fault {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, validation)
			var carrier interface{ Evidence() *anypb.Any }
			require.ErrorAs(t, err, &carrier)
			snapshot := new(temporalv1.WorkerTransportFault)
			require.NoError(t, carrier.Evidence().UnmarshalTo(snapshot))
			require.Len(t, snapshot.GetCalls(), 2)
			completed := snapshot.GetCalls()[1]
			require.Equal(t, uint32(WorkerRPCCompleted), completed.GetStage())
			require.True(t, completed.GetSent())
			require.True(t, completed.GetAnswered())
			require.Nil(t, completed.GetCallError())
			request := new(workflowservice.RespondActivityTaskFailedRequest)
			require.NoError(t, completed.GetRequest().UnmarshalTo(request))
			require.True(t, proto.Equal(chosen, request))
			actual := new(workflowservice.RespondActivityTaskFailedResponse)
			require.NoError(t, completed.GetResponse().UnmarshalTo(actual))
			require.True(t, proto.Equal(response, actual), "late fault retains already captured invalid acknowledgment bytes")
		})
	}
}
