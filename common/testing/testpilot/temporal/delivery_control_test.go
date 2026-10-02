//go:build test_dep

package temporal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/historyservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/namespace"
	serviceerrors "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/testing/testhooks"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/temporal/control"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// heldProgram is the worker outage Case with its stop and resume turned into a hold and a release
// of the same queue, as preparation hands it to a Driver.
func heldProgram(t *testing.T, mutate func(*testpilotspb.Case)) testpilot.PreparedProgram {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "tests", "testcore", "testpilot", "testdata", "workerOutageTests-survived-case.json"))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	for _, n := range source.GetProgram().GetEntrypoints()[0].GetInstructions() {
		switch fault := n.GetInstruction().GetInjectFault(); fault.GetKind() {
		case testpilotspb.FAULT_KIND_WORKER_STOP:
			fault.Kind = testpilotspb.FAULT_KIND_DELIVERY_HOLD
		case testpilotspb.FAULT_KIND_WORKER_RESUME:
			fault.Kind = testpilotspb.FAULT_KIND_DELIVERY_RELEASE
		default:
		}
	}
	if mutate != nil {
		mutate(source)
	}
	catalog, err := NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := DeriveProfile(source, catalog, Environment{Identity: "held", Namespace: "namespace", TaskQueue: "queue", DeliveryControl: true})
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	return facadetest.Capture(t, prepared)
}

// An environment without a delivery control, such as the canary's, refuses a Program that holds a
// delivery, and names the instruction; a release with no hold before it is refused everywhere.
func TestDeliveryControlIsRefusedWhereTheEnvironmentHasNone(t *testing.T) {
	program := heldProgram(t, nil)
	_, err := planDeliveries(program, false)
	require.ErrorIs(t, err, ErrNoDeliveryControl)
	require.ErrorContains(t, err, "controller/stop-worker requests DeliveryHold")
	plan, err := planDeliveries(program, true)
	require.NoError(t, err)
	require.Equal(t, map[string]deliveryQueue{"temporal.task-queue": {namespace: "namespace", queue: "queue"}}, plan)

	unheld := heldProgram(t, func(c *testpilotspb.Case) {
		c.GetProgram().GetEntrypoints()[0].GetInstructions()[0].GetInstruction().GetInjectFault().Kind = testpilotspb.FAULT_KIND_WORKER_STOP
	})
	_, err = planDeliveries(unheld, true)
	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorContains(t, err, "releases a delivery no earlier instruction holds")
}

// The start of the held queue's activity arms the hold before it is sent; the hold succeeds once
// the server holds the dispatch, and the release carries what admission decided for it.
func TestDeliverySessionHoldsTheStartedActivityAndReportsItsAdmission(t *testing.T) {
	hooks := testhooks.NewTestHooks()
	deliveries, err := control.NewDeliveries(func(h testhooks.Hook) func() { return h.Apply(hooks, namespace.ID("namespace-id")) })
	require.NoError(t, err)
	defer deliveries.Close()
	dispatch, ok := testhooks.Get(hooks, testhooks.ActivityDispatch, namespace.ID("namespace-id"))
	require.True(t, ok)
	respond, ok := testhooks.Get(hooks, testhooks.GRPCResponseFaultGeneratorByNamespaceID, namespace.ID("namespace-id"))
	require.True(t, ok)

	key := chasm.ExecutionKey{NamespaceID: "namespace-id", BusinessID: "activity", RunID: "run"}
	component := chasm.NewComponentRefByArchetypeID(key, 1)
	ref, err := component.Serialize(nil)
	require.NoError(t, err)
	var polls []*workflowservice.PollActivityTaskQueueRequest
	poll := func(ctx context.Context, request *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
		polls = append(polls, request)
		respond(ctx, "", &historyservice.RecordActivityTaskStartedRequest{ComponentRef: ref, Stamp: 3}, nil, serviceerrors.NewObsoleteMatchingTask("stamp mismatch"))
		<-ctx.Done()
		return nil, ctx.Err()
	}
	plan, err := planDeliveries(heldProgram(t, nil), true)
	require.NoError(t, err)
	session := newDeliverySession(deliveries, poll, plan)
	defer session.close()

	hold, err := session.inject(t.Context(), "temporal.task-queue", testpilotspb.FAULT_KIND_DELIVERY_HOLD)
	require.NoError(t, err)
	result, err := hold.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, "delivery_not_realized", result.Outcome.GetProtocolCode(), "a hold of a queue whose activity the Run never started realizes nothing")

	start := &workflowservice.StartActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity",
		ActivityType: &commonpb.ActivityType{Name: "type"}, TaskQueue: &taskqueuepb.TaskQueue{Name: "queue"}}
	method, err := protoreflectMethod(startActivityMethod)
	require.NoError(t, err)
	require.NoError(t, session.arm(method, start))
	require.ErrorIs(t, session.arm(method, start), ErrInvalid)
	dispatched := make(chan error, 1)
	go func() { dispatched <- dispatch(t.Context(), testhooks.ActivityDelivery{Execution: key, Stamp: 3}) }()
	hold, err = session.inject(t.Context(), "temporal.task-queue", testpilotspb.FAULT_KIND_DELIVERY_HOLD)
	require.NoError(t, err)
	result, err = hold.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.GetStatus())

	release, err := session.inject(t.Context(), "temporal.task-queue", testpilotspb.FAULT_KIND_DELIVERY_RELEASE)
	require.NoError(t, err)
	result, err = release.Wait(t.Context())
	require.NoError(t, err)
	require.NoError(t, <-dispatched)
	require.Equal(t, &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, DeliveryAdmission: &testpilotspb.DeliveryAdmission{
		ActivityId: "activity", ActivityRunId: "run", DeliveryId: "3", Decision: testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED,
	}}, result.Outcome)
	require.Len(t, polls, 1)
	require.Equal(t, "namespace", polls[0].GetNamespace())
	require.Equal(t, "queue", polls[0].GetTaskQueue().GetName())
}

func protoreflectMethod(name string) (protoreflect.MethodDescriptor, error) {
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	return descriptor.(protoreflect.MethodDescriptor), nil
}
