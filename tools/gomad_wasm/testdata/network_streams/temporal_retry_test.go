package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/common/testing/testcontext"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var networkTemporalStarted = time.Now()

type networkRPCAttempt struct {
	StartedNanos uint64 `json:"started_nanos"`
	EndedNanos   uint64 `json:"ended_nanos"`
	Code         string `json:"code"`
}

type networkSDKConnection struct {
	OpenedNanos uint64 `json:"opened_nanos"`
	LocalPort   int    `json:"local_port"`
	RemotePort  int    `json:"remote_port"`
}

type networkTemporalObservation struct {
	WorkflowID  string                 `json:"workflow_id"`
	RunID       string                 `json:"run_id"`
	Result      string                 `json:"result"`
	History     []byte                 `json:"history"`
	EventIDs    []int64                `json:"event_ids"`
	Types       []string               `json:"types"`
	Attempts    []networkRPCAttempt    `json:"attempts"`
	Connections []networkSDKConnection `json:"connections"`
}

func NetworkRecoveryWorkflow(ctx workflow.Context) (string, error) {
	if err := workflow.Sleep(ctx, 90*time.Second); err != nil {
		return "", err
	}
	return "durable timer survived rpc-reset", nil
}

func TestTemporalWorkflowRecoversNamedNetworkReset(t *testing.T) {
	testcontext.For(t, testcontext.WithTimeout(time.Hour))
	env := testcore.NewEnv(t, testcore.WithInMemorySQLitePersistence(), testcore.WithDisableTestloggerFailure())
	env.SdkWorker().RegisterWorkflow(NetworkRecoveryWorkflow)
	var mu sync.Mutex
	var attempts []networkRPCAttempt
	var connections []networkSDKConnection
	elapsed := func() uint64 { return uint64(time.Since(networkTemporalStarted)) }
	interceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
		start := elapsed()
		err := invoke(ctx, method, req, reply, cc, options...)
		if strings.HasSuffix(method, "/GetWorkflowExecutionHistory") {
			mu.Lock()
			attempts = append(attempts, networkRPCAttempt{StartedNanos: start, EndedNanos: elapsed(), Code: status.Code(err).String()})
			mu.Unlock()
		}
		return err
	}
	dial := func(ctx context.Context, address string) (net.Conn, error) {
		connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		local, localOK := connection.LocalAddr().(*net.TCPAddr)
		remote, remoteOK := connection.RemoteAddr().(*net.TCPAddr)
		if !localOK || !remoteOK {
			return nil, errors.Join(fmt.Errorf("SDK connection is not TCP: %T/%T", connection.LocalAddr(), connection.RemoteAddr()), connection.Close())
		}
		mu.Lock()
		connections = append(connections, networkSDKConnection{OpenedNanos: elapsed(), LocalPort: local.Port, RemotePort: remote.Port})
		mu.Unlock()
		return connection, nil
	}
	// The inner interceptor sees individual attempts beneath the SDK retry layer.
	sdk, err := client.Dial(client.Options{HostPort: env.FrontendGRPCAddress(), Namespace: env.Namespace().String(),
		ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(interceptor), grpc.WithContextDialer(dial)}}})
	if err != nil {
		t.Fatal(err)
	}
	defer sdk.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	run, err := sdk.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "network-reset-recovery", TaskQueue: env.WorkerTaskQueue()}, NetworkRecoveryWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	var result string
	if err := run.Get(ctx, &result); err != nil {
		t.Fatal(err)
	}
	if result != "durable timer survived rpc-reset" {
		t.Fatalf("workflow result changed: %q", result)
	}
	response, err := env.FrontendClient().GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: env.Namespace().String(), Execution: &commonpb.WorkflowExecution{WorkflowId: run.GetID(), RunId: run.GetRunID()}, MaximumPageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.NextPageToken) != 0 || response.History == nil {
		t.Fatal("history is missing or exceeds the fixture bound")
	}
	history, err := proto.MarshalOptions{Deterministic: true}.Marshal(response.History)
	if err != nil {
		t.Fatal(err)
	}
	observation := networkTemporalObservation{WorkflowID: run.GetID(), RunID: run.GetRunID(), Result: result, History: history}
	counts := map[enumspb.EventType]int{}
	for index, event := range response.History.Events {
		if event.EventId != int64(index+1) {
			t.Fatalf("duplicate or lost history event at %d: %d", index, event.EventId)
		}
		counts[event.EventType]++
		observation.EventIDs = append(observation.EventIDs, event.EventId)
		observation.Types = append(observation.Types, event.EventType.String())
		if strings.HasPrefix(event.EventType.String(), "ActivityTask") {
			t.Fatalf("network recovery fixture unexpectedly used an activity: %s", event.EventType)
		}
	}
	for _, eventType := range []enumspb.EventType{enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED, enumspb.EVENT_TYPE_TIMER_STARTED, enumspb.EVENT_TYPE_TIMER_FIRED, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED} {
		if counts[eventType] != 1 {
			t.Fatalf("lost or duplicate workflow transition %s: count=%d", eventType, counts[eventType])
		}
	}
	mu.Lock()
	observation.Attempts = append([]networkRPCAttempt(nil), attempts...)
	observation.Connections = append([]networkSDKConnection(nil), connections...)
	mu.Unlock()
	failedAttempt := -1
	for index, attempt := range observation.Attempts {
		if attempt.Code == codes.Unavailable.String() {
			failedAttempt = index
		}
		if failedAttempt >= 0 && index > failedAttempt && attempt.Code == codes.OK.String() {
			failedAttempt = -2
			break
		}
	}
	if failedAttempt != -2 {
		t.Fatalf("no concrete unavailable GetWorkflowExecutionHistory attempt followed by SDK recovery: %+v", observation.Attempts)
	}
	if len(observation.Connections) < 2 {
		t.Fatalf("SDK did not reconnect after reset: %+v", observation.Connections)
	}
	data, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("GOMAD_NETWORK_TEMPORAL_OBSERVATION %s\n", data)
}
