package worker

import (
	"context"
	"encoding/base64"
	"slices"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/common/testing/testpilot"
)

// These pins hold what a Program of workflow and Nexus-handler entrypoints registers, pools and
// puts on the wire. They were written against the Driver before it registered any activity, and an
// activity registration must leave every one of them as it is.

const (
	pinnedWorkflowHeader = "temporal-testpilot-reserved-workflow-v1"
	pinnedNexusHeader    = "temporal-testpilot-reserved-nexus-v1"
)

func pinnedRegistration() queueRegistration {
	return queueRegistration{queue: "task-queue", workflows: []string{"workflow-type"}, nexus: []nexusRegistration{{service: "service", operation: "operation"}}}
}

// recordingRegistrar records what a registration writes to an SDK worker, in call order.
type recordingRegistrar struct {
	calls []string
}

func (r *recordingRegistrar) RegisterDynamicWorkflow(interface{}, workflow.DynamicRegisterOptions) {
	r.calls = append(r.calls, "dynamic workflow")
}

func (r *recordingRegistrar) RegisterDynamicActivity(interface{}, activity.DynamicRegisterOptions) {
	r.calls = append(r.calls, "dynamic activity")
}

func (r *recordingRegistrar) RegisterNexusService(service *nexus.Service) {
	call := "nexus service " + service.Name
	for _, operation := range slices.Sorted(func(yield func(string) bool) {
		for _, name := range []string{"operation", "other"} {
			if service.Operation(name) != nil && !yield(name) {
				return
			}
		}
	}) {
		call += " " + operation
	}
	r.calls = append(r.calls, call)
}

func TestWorkflowAndNexusProgramRegistersOnlyWorkflowsAndNexusServices(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	require.Equal(t, []queueRegistration{pinnedRegistration()}, definition.registrations)

	registrar := &recordingRegistrar{}
	require.NoError(t, host.register(registrar, "task-queue", definition.registrations[0]))
	require.Equal(t, []string{"dynamic workflow", "nexus service service operation"}, registrar.calls)
}

// Two Runs of the same workflow and Nexus registration share one worker, keyed by its queue.
func TestWorkflowAndNexusRegistrationPoolsByQueue(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	_, definition := runtimeTestDriver(t, prepared)
	type built struct {
		key, queue   string
		registration queueRegistration
	}
	var workers []built
	registry := newWorkerRegistry(2, func(key, queue string, registration queueRegistration) (managedWorker, error) {
		workers = append(workers, built{key: key, queue: queue, registration: registration})
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	for _, runID := range []string{"run-a", "run-b"} {
		lease, err := registry.acquire(t.Context(), runID, definition.registrations, false, nil)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, newOutage(lease, OutagePlan{}).Restore(context.Background())) })
	}
	require.Equal(t, []built{{key: "task-queue", queue: "task-queue", registration: pinnedRegistration()}}, workers)
}

// The route a workflow start carries and the route its Nexus dispatch carries, byte for byte.
func TestWorkflowAndNexusRouteHeadersAreByteIdentical(t *testing.T) {
	prepared := preparedRuntimeFixture(t, replySynchronous)
	host, definition := runtimeTestDriver(t, prepared)
	session, _, request := runtimeTestSession(t, host, definition, prepared, "run", "workflow")

	// The routes are compared as the bytes they are, not as equivalent JSON.
	routes := struct{ workflow, nexus []byte }{
		workflow: []byte(`{"version":1,"kind":"workflow","session_id":"session-run","run_id":"run","origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller-1","InstructionID":"call","Attempt":1},"reservation":{"Origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller-1","InstructionID":"call","Attempt":1},"EntrypointID":"workflow","Ordinal":0,"ID":"reservation-1"},"binding":{"namespace":"namespace","workflow_id":"workflow","workflow_type":"workflow-type","task_queue":"task-queue"},"workflow_ordinal":0}`),
		nexus:    []byte(`{"version":1,"kind":"nexus","session_id":"session-run","run_id":"run","origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller-1","InstructionID":"call","Attempt":1},"reservation":{"Origin":{"RunID":"run","EntrypointID":"controller","ActivationID":"controller-1","InstructionID":"call","Attempt":1},"EntrypointID":"handler","Ordinal":0,"ID":"reservation-2"},"binding":{"namespace":"namespace","workflow_id":"workflow","workflow_type":"workflow-type","task_queue":"task-queue"},"workflow_reservation":"reservation-1","workflow_entrypoint":"workflow","workflow_ordinal":0,"workflow_run_id":"temporal-run","source_instruction_id":"start"}`),
	}

	require.Len(t, request.GetHeader().GetFields(), 1)
	carried := request.GetHeader().GetFields()[pinnedWorkflowHeader]
	require.Equal(t, map[string][]byte{"encoding": []byte("binary/temporal-testpilot-reservation-route")}, carried.GetMetadata())
	require.Equal(t, string(routes.workflow), string(carried.GetData()))

	routed, err := host.admitWorkflow(workflowDelivery(request, "temporal-run"))
	require.NoError(t, err)
	require.Equal(t, testpilot.Coordinate{RunID: "run", EntrypointID: "workflow", ActivationID: "reservation-1", Attempt: 1}, routed.activation.Coordinate())
	header, err := session.preparedNexusHeader(routed.activation, "start")
	require.NoError(t, err)
	require.Len(t, header, 1)
	dispatched, err := base64.RawURLEncoding.DecodeString(header[pinnedNexusHeader])
	require.NoError(t, err)
	require.Equal(t, string(routes.nexus), string(dispatched))
}
