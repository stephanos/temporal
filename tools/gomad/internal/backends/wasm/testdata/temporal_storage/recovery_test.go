package temporalstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	replicationpb "go.temporal.io/api/replication/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/cluster"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/membership/static"
	"go.temporal.io/server/common/persistence"
	persistencetests "go.temporal.io/server/common/persistence/persistence-tests"
	"go.temporal.io/server/common/primitives"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type observation struct {
	ClusterID string   `json:"cluster_id"`
	History   []byte   `json:"history"`
	EventIDs  []int64  `json:"event_ids"`
	Types     []string `json:"types"`
}

func persistentCluster(t *testing.T, initialize bool) *testcore.TestCluster {
	t.Helper()
	database := os.Getenv("GOMAD_TEMPORAL_DATABASE")
	if database == "" {
		t.Fatal("GOMAD_TEMPORAL_DATABASE is required")
	}
	name := os.Getenv("GOMAD_TEMPORAL_CLUSTER")
	if name == "" {
		name = "recovery"
	}
	version := int64(1)
	if name == "standby" {
		version = 2
	}
	hosts := map[primitives.ServiceName]static.Hosts{}
	for i, service := range []primitives.ServiceName{primitives.FrontendService, primitives.HistoryService, primitives.MatchingService, primitives.WorkerService} {
		hosts[service] = static.Hosts{All: []string{fmt.Sprintf("127.0.0.1:%d", 7233+i)}}
	}
	cfg := &testcore.TestClusterConfig{
		DisablePProf: true,
		ClusterMetadata: cluster.Config{EnableGlobalNamespace: name != "recovery", CurrentClusterName: name, MasterClusterName: name, FailoverVersionIncrement: 10,
			ClusterInformation: map[string]cluster.ClusterInformation{name: {Enabled: true, InitialFailoverVersion: version}}},
		HistoryConfig:             testcore.HistoryConfig{NumHistoryShards: 1},
		EnableHistoryTaskRecorder: name != "recovery",
		Persistence: persistencetests.TestBaseOptions{StoreType: "sql", SQLDBPluginName: "sqlite", DBName: database,
			SchemaDir: os.Getenv("GOMAD_TEMPORAL_SCHEMA"), ConnectAttributes: map[string]string{"journal_mode": "wal", "synchronous": "full", "busy_timeout": "30000"}},
		DynamicConfigOverrides: map[dynamicconfig.Key]any{dynamicconfig.NamespaceCacheRefreshInterval.Key(): 100 * time.Millisecond},
	}
	c, err := testcore.NewPersistentTestCluster(t, cfg, log.NewNoopLogger(), initialize, hosts, "127.0.0.1:7243")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func observe(t *testing.T, c *testcore.TestCluster) observation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	metadata, err := c.TestBase().ClusterMetadataManager.GetClusterMetadata(ctx, &persistence.GetClusterMetadataRequest{ClusterName: c.ClusterName()})
	if err != nil {
		t.Fatal(err)
	}
	history, err := c.FrontendClient().GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace: "recovery", Execution: &commonpb.WorkflowExecution{WorkflowId: "committed"}, MaximumPageSize: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(history.NextPageToken) != 0 {
		t.Fatal("history exceeds fixture bound")
	}
	if len(history.History.Events) == 0 {
		t.Fatal("committed history is empty")
	}
	completed := history.History.Events[len(history.History.Events)-1].GetWorkflowExecutionCompletedEventAttributes()
	if completed == nil || len(completed.GetResult().GetPayloads()) != 1 || !bytes.Equal(completed.Result.Payloads[0].Data, []byte("durable-result")) {
		t.Fatal("committed workflow result is missing or changed")
	}
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(history.History)
	if err != nil {
		t.Fatal(err)
	}
	out := observation{ClusterID: metadata.ClusterId, History: data}
	for _, event := range history.History.Events {
		out.EventIDs = append(out.EventIDs, event.EventId)
		out.Types = append(out.Types, event.EventType.String())
	}
	return out
}

func emit(t *testing.T, prefix string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s%s\n", prefix, data)
}

func TestCommittedWorkflowCrashRecovery(t *testing.T) {
	initialize := os.Getenv("GOMAD_TEMPORAL_PHASE") == "commit"
	c := persistentCluster(t, initialize)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if initialize {
		registerRecoveryNamespace(t, ctx, c, false)
		commitWorkflow(t, ctx, c)
	}
	value := observe(t, c)
	if len(value.Types) != 5 || value.Types[len(value.Types)-1] != enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED.String() {
		t.Fatalf("committed history is incomplete: %v", value.Types)
	}
	emit(t, "GOMAD_TEMPORAL_OBSERVATION ", value)
	if initialize {
		// The host discards this incarnation at the acknowledged commit marker.
		select {}
	}
	if err := c.TearDownCluster(); err != nil {
		t.Fatal(err)
	}
}

func registerRecoveryNamespace(t *testing.T, ctx context.Context, c *testcore.TestCluster, global bool) {
	t.Helper()
	registration := &workflowservice.RegisterNamespaceRequest{Namespace: "recovery", WorkflowExecutionRetentionPeriod: durationpb.New(24 * time.Hour), IsGlobalNamespace: global}
	if global {
		registration.ActiveClusterName = "active"
		registration.Clusters = []*replicationpb.ClusterReplicationConfig{{ClusterName: "active"}, {ClusterName: "standby"}}
	}
	registrationContext, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		_, err := c.FrontendClient().RegisterNamespace(registrationContext, registration)
		if err == nil {
			break
		}
		var invalid *serviceerror.InvalidArgument
		// AddOrUpdateRemoteCluster commits metadata before the ordinary minute
		// cache refresh. Retry only this observed readiness error, not arbitrary
		// invalid requests or other registration failures.
		if !global || !errors.As(err, &invalid) || invalid.Message != "Invalid cluster name: standby" {
			t.Fatal(err)
		}
		select {
		case <-registrationContext.Done():
			t.Fatalf("remote cluster metadata readiness: %v: %v", registrationContext.Err(), err)
		case <-time.After(time.Second):
		}
	}
}

func commitWorkflow(t *testing.T, ctx context.Context, c *testcore.TestCluster) {
	t.Helper()
	_, err := c.FrontendClient().StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace: "recovery", WorkflowId: "committed", WorkflowType: &commonpb.WorkflowType{Name: "Recovery"},
		TaskQueue: &taskqueuepb.TaskQueue{Name: "recovery"}, RequestId: "start-committed", WorkflowTaskTimeout: durationpb.New(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := c.FrontendClient().PollWorkflowTaskQueue(ctx, &workflowservice.PollWorkflowTaskQueueRequest{Namespace: "recovery", TaskQueue: &taskqueuepb.TaskQueue{Name: "recovery"}, Identity: "recovery-worker"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.FrontendClient().RespondWorkflowTaskCompleted(ctx, &workflowservice.RespondWorkflowTaskCompletedRequest{
		Namespace: "recovery", TaskToken: task.TaskToken, Identity: "recovery-worker",
		Commands: []*commandpb.Command{{CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_CompleteWorkflowExecutionCommandAttributes{CompleteWorkflowExecutionCommandAttributes: &commandpb.CompleteWorkflowExecutionCommandAttributes{
				Result: &commonpb.Payloads{Payloads: []*commonpb.Payload{{Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("durable-result")}}},
			}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
}
