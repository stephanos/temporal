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
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/adminservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/rpc"
	"go.temporal.io/server/service/history/tasks"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
			commitWorkflow(t, ctx, c, true)
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
