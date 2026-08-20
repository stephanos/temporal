package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

func TestTaskAckRunsThroughPublicTaskProtocolEvidence(t *testing.T) {
	t.Parallel()

	transport := &fakeWorkflowTaskTransport{backlogAbsent: true}
	result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment: taskAckExperiment(t),
		Environment: NewTaskAckFactory(func(context.Context) (ClusterInfo, error) {
			return ClusterInfo{BuildID: "build", ConfigurationID: "configuration", Namespace: "namespace"}, nil
		}, transport),
	})
	require.NoError(t, err)
	require.Equal(t, umpire3runtime.ClaimConforming, result.Claim.Kind)
	require.Equal(t, "public-grpc-history", result.Environment.EvidenceProfile)
	require.Equal(t, []string{"enqueue", "deliver", "acknowledge", "cleanup"}, transport.calls)
	require.Equal(t, "workflow", result.Observations[0].EntityIdentity)
	require.Equal(t, []string{"namespace", "workflow", "run"}, result.Observations[0].Lineage)
}

func TestTaskAckNegativeControlViolatesGeneratedMonitor(t *testing.T) {
	t.Parallel()

	result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment: taskAckExperiment(t),
		Environment: NewTaskAckFactory(func(context.Context) (ClusterInfo, error) {
			return ClusterInfo{BuildID: "build", Namespace: "namespace"}, nil
		}, &fakeWorkflowTaskTransport{}),
	})
	require.NoError(t, err)
	require.Equal(t, umpire3runtime.ClaimViolating, result.Claim.Kind)
	require.Equal(t, "observe-workflow-task-acknowledged", result.Claim.Checkpoint)
}

func TestTaskAckRejectsChangedDeliveryLineage(t *testing.T) {
	t.Parallel()

	transport := &fakeWorkflowTaskTransport{differentRun: true, backlogAbsent: true}
	result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment: taskAckExperiment(t),
		Environment: NewTaskAckFactory(func(context.Context) (ClusterInfo, error) {
			return ClusterInfo{BuildID: "build", Namespace: "namespace"}, nil
		}, transport),
	})
	require.NoError(t, err)
	require.Equal(t, umpire3runtime.ClaimInconclusive, result.Claim.Kind)
	require.Contains(t, result.Claim.Reason, "does not match enqueued lineage")
}

func taskAckExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	suite, err := compiler.Compile(context.Background(), regress.NewScenario(
		"workflow-task-ack",
		protocol.TargetIDFoundationBacklogAck,
		[]regress.Resource{regress.WorkflowTask("workflow-task")},
		regress.OnePath(
			regress.EnqueueWorkflowTask("enqueue"),
			regress.DeliverWorkflowTask("deliver"),
			regress.AcknowledgeWorkflowTask("acknowledge"),
			regress.RequireTaskDeliveryAcknowledgedRemovesBacklog(),
		),
	), compiler.Limits{
		MaxPaths: 4, MaxActions: 8, MaxStates: 64, MaxMemoryBytes: 1 << 20, MaxTime: time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	return suite.Experiments[0]
}

type fakeWorkflowTaskTransport struct {
	backlogAbsent bool
	differentRun  bool
	calls         []string
}

func (f *fakeWorkflowTaskTransport) Enqueue(context.Context) (WorkflowTaskIdentity, error) {
	f.calls = append(f.calls, "enqueue")
	return WorkflowTaskIdentity{
		WorkflowID: "workflow", RunID: "run", Source: "frontend-start", Reference: "workflow/run",
	}, nil
}

func (f *fakeWorkflowTaskTransport) Deliver(
	_ context.Context,
	identity WorkflowTaskIdentity,
) (WorkflowTaskDelivery, error) {
	f.calls = append(f.calls, "deliver")
	if f.differentRun {
		identity.RunID = "different-run"
	}
	identity.Source = "frontend-poll"
	identity.Reference += "/task"
	return WorkflowTaskDelivery{WorkflowTaskIdentity: identity, TaskToken: []byte("task-token")}, nil
}

func (f *fakeWorkflowTaskTransport) Acknowledge(
	context.Context,
	WorkflowTaskDelivery,
) (WorkflowTaskAcknowledgement, error) {
	f.calls = append(f.calls, "acknowledge")
	return WorkflowTaskAcknowledgement{
		BacklogAbsent: f.backlogAbsent, Source: "frontend-complete", Reference: "workflow/run/task/complete",
	}, nil
}

func (f *fakeWorkflowTaskTransport) Cleanup(context.Context, WorkflowTaskIdentity) error {
	f.calls = append(f.calls, "cleanup")
	return nil
}
