package temporalstorage

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	namespacepb "go.temporal.io/api/namespace/v1"
	replicationpb "go.temporal.io/api/replication/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/adminservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/rpc"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/service/history/tasks"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReplicatedNamespaceReadiness(t *testing.T) {
	t.Parallel()
	ready := &workflowservice.DescribeNamespaceResponse{
		NamespaceInfo:     &namespacepb.NamespaceInfo{Id: "registered-namespace-id", State: enumspb.NAMESPACE_STATE_REGISTERED},
		IsGlobalNamespace: true,
		ReplicationConfig: &replicationpb.NamespaceReplicationConfig{
			ActiveClusterName: "active",
			Clusters:          []*replicationpb.ClusterReplicationConfig{{ClusterName: "active"}, {ClusterName: "standby"}},
		},
	}
	for _, test := range []struct {
		name   string
		mutate func(*workflowservice.DescribeNamespaceResponse)
		want   bool
	}{
		{name: "registered-global-namespace", want: true},
		{name: "replica-order-independent", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters[0], s.ReplicationConfig.Clusters[1] = s.ReplicationConfig.Clusters[1], s.ReplicationConfig.Clusters[0]
		}, want: true},
		{name: "missing-namespace-info", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.NamespaceInfo = nil }},
		{name: "missing-namespace-id", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.NamespaceInfo.Id = "" }},
		{name: "different-namespace-id", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.NamespaceInfo.Id = "another-namespace-id" }},
		{name: "unspecified-state", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.NamespaceInfo.State = enumspb.NAMESPACE_STATE_UNSPECIFIED
		}},
		{name: "deprecated-state", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.NamespaceInfo.State = enumspb.NAMESPACE_STATE_DEPRECATED
		}},
		{name: "deleted-state", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.NamespaceInfo.State = enumspb.NAMESPACE_STATE_DELETED
		}},
		{name: "local-namespace", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.IsGlobalNamespace = false }},
		{name: "missing-replication-config", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.ReplicationConfig = nil }},
		{name: "missing-active-cluster", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.ReplicationConfig.ActiveClusterName = "" }},
		{name: "wrong-active-cluster", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.ReplicationConfig.ActiveClusterName = "standby" }},
		{name: "missing-replica-membership", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.ReplicationConfig.Clusters = nil }},
		{name: "nil-active-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.ReplicationConfig.Clusters[0] = nil }},
		{name: "nil-standby-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) { s.ReplicationConfig.Clusters[1] = nil }},
		{name: "missing-active-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters = s.ReplicationConfig.Clusters[1:]
		}},
		{name: "missing-standby-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters = s.ReplicationConfig.Clusters[:1]
		}},
		{name: "wrong-active-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters[0].ClusterName = "other"
		}},
		{name: "wrong-standby-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters[1].ClusterName = "other"
		}},
		{name: "duplicate-active-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters[1].ClusterName = "active"
		}},
		{name: "extra-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters = append(s.ReplicationConfig.Clusters, &replicationpb.ClusterReplicationConfig{ClusterName: "other"})
		}},
		{name: "extra-nil-replica", mutate: func(s *workflowservice.DescribeNamespaceResponse) {
			s.ReplicationConfig.Clusters = append(s.ReplicationConfig.Clusters, nil)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := proto.Clone(ready).(*workflowservice.DescribeNamespaceResponse)
			if test.mutate != nil {
				test.mutate(response)
			}
			require.Equal(t, test.want, replicatedNamespaceReady(response, "registered-namespace-id", "active", "standby"))
		})
	}
	for _, test := range []struct {
		name                         string
		response                     *workflowservice.DescribeNamespaceResponse
		namespaceID, active, standby string
	}{
		{name: "nil-response", namespaceID: "registered-namespace-id", active: "active", standby: "standby"},
		{name: "empty-response", response: &workflowservice.DescribeNamespaceResponse{}, namespaceID: "registered-namespace-id", active: "active", standby: "standby"},
		{name: "missing-expected-id", response: ready, active: "active", standby: "standby"},
		{name: "missing-expected-active", response: ready, namespaceID: "registered-namespace-id", standby: "standby"},
		{name: "missing-expected-standby", response: ready, namespaceID: "registered-namespace-id", active: "active"},
		{name: "identical-expected-replicas", response: ready, namespaceID: "registered-namespace-id", active: "active", standby: "active"},
		{name: "all-zero-values"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.False(t, replicatedNamespaceReady(test.response, test.namespaceID, test.active, test.standby))
		})
	}
}

func replicatedNamespaceReady(response *workflowservice.DescribeNamespaceResponse, namespaceID, active, standby string) bool {
	if namespaceID == "" || active == "" || standby == "" || active == standby ||
		response.GetNamespaceInfo().GetId() != namespaceID ||
		response.GetNamespaceInfo().GetState() != enumspb.NAMESPACE_STATE_REGISTERED || !response.GetIsGlobalNamespace() {
		return false
	}
	config := response.GetReplicationConfig()
	if config.GetActiveClusterName() != active || len(config.GetClusters()) != 2 {
		return false
	}
	first, second := config.Clusters[0].GetClusterName(), config.Clusters[1].GetClusterName()
	return first == active && second == standby || first == standby && second == active
}

func waitNamespaceReplicated(t *testing.T, ctx context.Context, active, standby workflowservice.WorkflowServiceClient) {
	t.Helper()
	readinessContext, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	registered, err := active.DescribeNamespace(readinessContext, &workflowservice.DescribeNamespaceRequest{Namespace: "recovery"})
	require.NoError(t, err)
	namespaceID := registered.GetNamespaceInfo().GetId()
	require.NotEmpty(t, namespaceID)
	await.Require(readinessContext, t, func(t *await.T) {
		response, err := standby.DescribeNamespace(t.Context(), &workflowservice.DescribeNamespaceRequest{Namespace: "recovery"})
		require.NoError(t, err)
		require.True(t, replicatedNamespaceReady(response, namespaceID, "active", "standby"), "standby namespace is not ready: %v", response)
	}, 90*time.Second, time.Second)
}

func TestReplicationReadinessRequiresRemoteAcknowledgment(t *testing.T) {
	for _, route := range []struct {
		frontend, history string
		valid             bool
	}{
		{"10.0.0.2:7233", "10.0.0.2:7234", true},
		{"10.0.0.2:7233", "10.0.0.1:7234", false},
		{"10.0.0.2:7233", "127.0.0.1:7234", false},
		{"10.0.0.2:7233", "10.0.0.2:7233", false},
		{"10.0.0.2:7234", "10.0.0.2:7234", false},
		{"10.0.0.2:7233", "peer:7234", false},
		{"10.0.0.2:7233", "0.0.0.0:7234", false},
		{"", "10.0.0.2:7234", false},
	} {
		t.Run("route-"+route.history, func(t *testing.T) {
			require.Equal(t, route.valid, validatePeerHistoryRoute(route.frontend, route.history) == nil)
		})
	}
	start := time.Unix(1700000000, 0)
	now := start.Add(time.Second)
	ready := &historyservice.GetReplicationStatusResponse{Shards: []*historyservice.ShardReplicationStatus{{
		ShardId: 1, MaxReplicationTaskId: 10, ShardLocalTime: timestamppb.New(now),
		RemoteClusters: map[string]*historyservice.ShardReplicationStatusPerCluster{"standby": {AckedTaskId: 10, AckedTaskVisibilityTime: timestamppb.New(now)}},
	}}}
	require.True(t, replicationConnected(ready, "standby", start, now))
	require.False(t, replicationConnected(nil, "standby", start, now))
	require.False(t, replicationConnected(&historyservice.GetReplicationStatusResponse{}, "standby", start, now))
	for _, test := range []struct {
		name   string
		mutate func(*historyservice.GetReplicationStatusResponse)
	}{
		{"nil-shard", func(s *historyservice.GetReplicationStatusResponse) { s.Shards[0] = nil }},
		{"extra-shard", func(s *historyservice.GetReplicationStatusResponse) { s.Shards = append(s.Shards, s.Shards[0]) }},
		{"unacquired-shard", func(s *historyservice.GetReplicationStatusResponse) { s.Shards[0].MaxReplicationTaskId = 0 }},
		{"missing-remote", func(s *historyservice.GetReplicationStatusResponse) { s.Shards[0].RemoteClusters = nil }},
		{"nil-remote", func(s *historyservice.GetReplicationStatusResponse) { s.Shards[0].RemoteClusters["standby"] = nil }},
		{"behind-ack", func(s *historyservice.GetReplicationStatusResponse) {
			s.Shards[0].RemoteClusters["standby"].AckedTaskId = 9
		}},
		{"missing-local-time", func(s *historyservice.GetReplicationStatusResponse) { s.Shards[0].ShardLocalTime = nil }},
		{"stale-local-time", func(s *historyservice.GetReplicationStatusResponse) {
			s.Shards[0].ShardLocalTime = timestamppb.New(start.Add(-time.Second))
		}},
		{"missing-ack-time", func(s *historyservice.GetReplicationStatusResponse) {
			s.Shards[0].RemoteClusters["standby"].AckedTaskVisibilityTime = nil
		}},
		{"future-ack-time", func(s *historyservice.GetReplicationStatusResponse) {
			s.Shards[0].RemoteClusters["standby"].AckedTaskVisibilityTime = timestamppb.New(now.Add(time.Second))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := proto.Clone(ready).(*historyservice.GetReplicationStatusResponse)
			test.mutate(value)
			require.False(t, replicationConnected(value, "standby", start, now))
		})
	}
}

func replicationConnected(status *historyservice.GetReplicationStatusResponse, remote string, start, now time.Time) bool {
	if status == nil || len(status.Shards) != 1 || status.Shards[0] == nil {
		return false
	}
	shard := status.Shards[0]
	ack := shard.RemoteClusters[remote]
	if shard.MaxReplicationTaskId <= 0 || shard.ShardLocalTime == nil || ack == nil || ack.AckedTaskId < shard.MaxReplicationTaskId || ack.AckedTaskVisibilityTime == nil {
		return false
	}
	localTime, ackTime := shard.ShardLocalTime.AsTime(), ack.AckedTaskVisibilityTime.AsTime()
	return !localTime.Before(start) && !localTime.After(now) && !ackTime.Before(start) && !ackTime.After(now)
}

func validatePeerHistoryRoute(frontend, history string) error {
	frontendHost, frontendPort, frontendError := net.SplitHostPort(frontend)
	historyHost, historyPort, historyError := net.SplitHostPort(history)
	frontendIP, historyIP := net.ParseIP(frontendHost), net.ParseIP(historyHost)
	if frontendError != nil || historyError != nil || frontendPort != "7233" || historyPort != "7234" ||
		frontendIP == nil || historyIP == nil || !frontendIP.Equal(historyIP) || historyIP.IsLoopback() || historyIP.IsUnspecified() {
		return errors.New("remote history must use the registered peer IP and existing static history port 7234")
	}
	return nil
}

func waitReplicationConnected(t *testing.T, ctx context.Context, client historyservice.HistoryServiceClient, source, target string, started time.Time) {
	t.Helper()
	require.Eventually(t, func() bool {
		status, err := client.GetReplicationStatus(ctx, &historyservice.GetReplicationStatusRequest{})
		if err != nil {
			t.Logf("replication readiness %s -> %s: %v", source, target, err)
			return false
		}
		now := time.Now()
		connected := replicationConnected(status, target, started, now)
		emit(t, "GOMAD_TEMPORAL_REPLICATION_STATUS ", struct {
			Cluster, Remote string
			At              time.Time
			Connected       bool
			Status          *historyservice.GetReplicationStatusResponse
		}{source, target, now, connected, status})
		return connected
	}, 90*time.Second, time.Second)
}

func TestIndependentClusterReplicationRecovery(t *testing.T) {
	name := os.Getenv("GOMAD_TEMPORAL_CLUSTER")
	if name != "active" && name != "standby" {
		t.Fatal("independent cluster role must be active or standby")
	}
	initialize := os.Getenv("GOMAD_TEMPORAL_PHASE") == "commit"
	started := time.Now()
	c := persistentCluster(t, initialize)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if initialize {
		address := os.Getenv("GOMAD_TEMPORAL_REMOTE_FRONTEND")
		if address == "" {
			t.Fatal("remote guest frontend address is required")
		}
		historyAddress := os.Getenv("GOMAD_TEMPORAL_REMOTE_HISTORY")
		require.NoError(t, validatePeerHistoryRoute(address, historyAddress))
		peerConnection, err := rpc.Dial(historyAddress, nil, log.NewNoopLogger(), metrics.NoopMetricsHandler)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, peerConnection.Close()) })
		peerHistory := historyservice.NewHistoryServiceClient(peerConnection)
		require.Eventually(t, func() bool {
			_, err := c.AdminClient().AddOrUpdateRemoteCluster(ctx, &adminservice.AddOrUpdateRemoteClusterRequest{
				FrontendAddress: address, FrontendHttpAddress: os.Getenv("GOMAD_TEMPORAL_REMOTE_HTTP"),
				EnableRemoteClusterConnection: true, EnableReplication: true,
			})
			return err == nil
		}, 5*time.Minute, time.Second)
		remote := "active"
		if name == "active" {
			remote = "standby"
		}
		// Mirror both directions of tests/xdc/base.go's pre-workload barrier.
		// Metadata registration alone does not establish replication readers.
		waitReplicationConnected(t, ctx, c.HistoryClient(), name, remote, started)
		waitReplicationConnected(t, ctx, peerHistory, remote, name, started)
		if name == "active" {
			registerRecoveryNamespace(t, ctx, c, true)
			peerFrontendConnection, err := rpc.Dial(address, nil, log.NewNoopLogger(), metrics.NoopMetricsHandler)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, peerFrontendConnection.Close()) })
			waitNamespaceReplicated(t, ctx, c.FrontendClient(), workflowservice.NewWorkflowServiceClient(peerFrontendConnection))
			commitWorkflow(t, ctx, c)
			recorded := c.GetHistoryTaskRecorder()
			if recorded == nil {
				t.Fatal("replication task recorder is required")
			}
			type replicationTaskEvidence struct {
				ID, RangeID                int64
				ShardID                    int32
				Type, Namespace, Workflow  string
				RecordedAt, VisibilityTime time.Time
			}
			var evidence []replicationTaskEvidence
			for _, task := range recorded.GetAllRecordedTasks()[tasks.CategoryReplication] {
				if task.WorkflowID != "committed" {
					continue
				}
				evidence = append(evidence, replicationTaskEvidence{task.Task.GetTaskID(), task.RangeID, task.ShardID, task.TaskType, task.NamespaceID, task.WorkflowID, task.Timestamp, task.Task.GetVisibilityTime()})
			}
			emit(t, "GOMAD_TEMPORAL_REPLICATION_TASKS ", evidence)
		}
	}
	// The standby observation comes from its own independently persisted history,
	// populated by Temporal's ordinary remote replication stream.
	require.Eventually(t, func() bool {
		response, err := c.FrontendClient().GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
			Namespace: "recovery", Execution: &commonpb.WorkflowExecution{WorkflowId: "committed"}, MaximumPageSize: 100,
		})
		return err == nil && len(response.History.Events) == 5 && response.History.Events[4].EventType == enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED
	}, 10*time.Minute, time.Second)
	emit(t, "GOMAD_TEMPORAL_OBSERVATION ", observe(t, c))
	if initialize {
		select {}
	}
	if err := c.TearDownCluster(); err != nil {
		t.Fatal(err)
	}
}
