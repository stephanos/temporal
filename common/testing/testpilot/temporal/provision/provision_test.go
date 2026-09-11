package provision

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

// fakeWorkflow answers only the two calls provisioning makes. Any other call panics on the nil
// embedded interface, which is what says the package reached past its declared surface.
type fakeWorkflow struct {
	workflowservice.WorkflowServiceClient
	registered         []*workflowservice.RegisterNamespaceRequest
	registerErr        error
	describes          int
	describeUntilReady int
	describeErr        error
}

func (f *fakeWorkflow) RegisterNamespace(_ context.Context, request *workflowservice.RegisterNamespaceRequest, _ ...grpc.CallOption) (*workflowservice.RegisterNamespaceResponse, error) {
	f.registered = append(f.registered, request)
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	return &workflowservice.RegisterNamespaceResponse{}, nil
}

func (f *fakeWorkflow) DescribeNamespace(_ context.Context, _ *workflowservice.DescribeNamespaceRequest, _ ...grpc.CallOption) (*workflowservice.DescribeNamespaceResponse, error) {
	f.describes++
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	if f.describes <= f.describeUntilReady {
		return nil, serviceerror.NewNamespaceNotFound("not yet")
	}
	return &workflowservice.DescribeNamespaceResponse{}, nil
}

type fakeOperator struct {
	operatorservice.OperatorServiceClient
	createdEndpoints   []string
	createErr          error
	deletedEndpoints   []string
	deleteEndpointErr  error
	deletedNamespaces  []string
	deleteNamespaceErr error
}

func (f *fakeOperator) CreateNexusEndpoint(_ context.Context, request *operatorservice.CreateNexusEndpointRequest, _ ...grpc.CallOption) (*operatorservice.CreateNexusEndpointResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.createdEndpoints = append(f.createdEndpoints, request.GetSpec().GetName())
	return &operatorservice.CreateNexusEndpointResponse{
		Endpoint: &nexuspb.Endpoint{Id: "endpoint-id", Version: 7, Spec: request.GetSpec()},
	}, nil
}

func (f *fakeOperator) DeleteNexusEndpoint(_ context.Context, request *operatorservice.DeleteNexusEndpointRequest, _ ...grpc.CallOption) (*operatorservice.DeleteNexusEndpointResponse, error) {
	if f.deleteEndpointErr != nil {
		return nil, f.deleteEndpointErr
	}
	f.deletedEndpoints = append(f.deletedEndpoints, request.GetId())
	return &operatorservice.DeleteNexusEndpointResponse{}, nil
}

func (f *fakeOperator) DeleteNamespace(_ context.Context, request *operatorservice.DeleteNamespaceRequest, _ ...grpc.CallOption) (*operatorservice.DeleteNamespaceResponse, error) {
	if f.deleteNamespaceErr != nil {
		return nil, f.deleteNamespaceErr
	}
	f.deletedNamespaces = append(f.deletedNamespaces, request.GetNamespace())
	return &operatorservice.DeleteNamespaceResponse{}, nil
}

func fastResources(resources Resources) Resources {
	resources.ReadyTimeout = 2 * time.Second
	resources.ReadyInterval = time.Millisecond
	return resources
}

func TestCreateRegistersWaitsForTheCacheAndCreatesTheEndpoint(t *testing.T) {
	workflow := &fakeWorkflow{describeUntilReady: 3}
	operator := &fakeOperator{}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: operator},
		fastResources(Resources{
			Namespace: "umpire-ns", TaskQueue: "umpire-queue", NexusEndpoint: "umpire-endpoint",
		}))

	require.NoError(t, err)
	require.Len(t, workflow.registered, 1)
	require.Equal(t, "umpire-ns", workflow.registered[0].GetNamespace())
	require.Equal(t, DefaultRetention, workflow.registered[0].GetWorkflowExecutionRetentionPeriod().AsDuration())
	require.Equal(t, 4, workflow.describes, "the namespace is polled until the cache serves it")
	require.Equal(t, []string{"umpire-endpoint"}, operator.createdEndpoints)

	require.NoError(t, cleanup(context.Background()))
	require.Equal(t, []string{"endpoint-id"}, operator.deletedEndpoints)
	require.Equal(t, []string{"umpire-ns"}, operator.deletedNamespaces)
}

// A caller whose cluster is discarded wholesale keeps the namespace, and says so; every other
// caller deletes it.
func TestCreateRetainsTheNamespaceWhenAsked(t *testing.T) {
	workflow := &fakeWorkflow{}
	operator := &fakeOperator{}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: operator},
		fastResources(Resources{
			Namespace: "umpire-ns", TaskQueue: "umpire-queue", NexusEndpoint: "umpire-endpoint",
			RetainNamespace: true,
		}))

	require.NoError(t, err)
	require.NoError(t, cleanup(context.Background()))
	require.Equal(t, []string{"endpoint-id"}, operator.deletedEndpoints)
	require.Empty(t, operator.deletedNamespaces)
}

func TestCreateSkipsTheEndpointWhenTheCaseBindsNone(t *testing.T) {
	workflow := &fakeWorkflow{}
	operator := &fakeOperator{}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: operator},
		fastResources(Resources{Namespace: "umpire-ns", TaskQueue: "umpire-queue"}))

	require.NoError(t, err)
	require.Empty(t, operator.createdEndpoints)
	require.NoError(t, cleanup(context.Background()))
	require.Empty(t, operator.deletedEndpoints)
	require.Equal(t, []string{"umpire-ns"}, operator.deletedNamespaces)
}

// A caller that asked to own the resources is told they already exist rather than silently reusing
// someone else's namespace.
func TestCreateReportsAnExistingNamespace(t *testing.T) {
	workflow := &fakeWorkflow{registerErr: serviceerror.NewNamespaceAlreadyExists("already there")}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: &fakeOperator{}},
		fastResources(Resources{Namespace: "umpire-ns", TaskQueue: "umpire-queue"}))

	require.Nil(t, cleanup)
	require.ErrorContains(t, err, `register namespace "umpire-ns"`)
	var existing *serviceerror.NamespaceAlreadyExists
	require.ErrorAs(t, err, &existing)
}

// Create never leaves a caller holding a half-built environment: a failure after the namespace
// landed deletes it before returning.
func TestCreateRollsBackTheNamespaceWhenTheEndpointFails(t *testing.T) {
	workflow := &fakeWorkflow{}
	operator := &fakeOperator{createErr: errors.New("endpoint name is taken")}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: operator},
		fastResources(Resources{
			Namespace: "umpire-ns", TaskQueue: "umpire-queue", NexusEndpoint: "umpire-endpoint",
		}))

	require.Nil(t, cleanup)
	require.ErrorContains(t, err, "endpoint name is taken")
	require.Equal(t, []string{"umpire-ns"}, operator.deletedNamespaces)
}

func TestCreateReportsANamespaceTheCacheNeverServes(t *testing.T) {
	workflow := &fakeWorkflow{describeUntilReady: 1_000_000}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: &fakeOperator{}},
		Resources{
			Namespace: "umpire-ns", TaskQueue: "umpire-queue",
			ReadyTimeout: 20 * time.Millisecond, ReadyInterval: time.Millisecond,
		})

	require.Nil(t, cleanup)
	require.ErrorContains(t, err, `namespace "umpire-ns" was not served within`)
}

// A DescribeNamespace failure that is not "not found yet" is a real failure, not something to poll
// through until the deadline.
func TestCreateStopsPollingOnAnUnrelatedDescribeFailure(t *testing.T) {
	workflow := &fakeWorkflow{describeErr: errors.New("frontend is unavailable")}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: &fakeOperator{}},
		fastResources(Resources{Namespace: "umpire-ns", TaskQueue: "umpire-queue"}))

	require.Nil(t, cleanup)
	require.ErrorContains(t, err, "frontend is unavailable")
	require.Equal(t, 1, workflow.describes)
}

// One stuck resource never hides the others: cleanup keeps going and reports both.
func TestCleanupReportsEveryResourceItCouldNotRemove(t *testing.T) {
	workflow := &fakeWorkflow{}
	operator := &fakeOperator{
		deleteEndpointErr:  errors.New("endpoint is still referenced"),
		deleteNamespaceErr: errors.New("namespace deletion in progress"),
	}

	cleanup, err := Create(context.Background(), Clients{Workflow: workflow, Operator: operator},
		fastResources(Resources{
			Namespace: "umpire-ns", TaskQueue: "umpire-queue", NexusEndpoint: "umpire-endpoint",
		}))
	require.NoError(t, err)

	err = cleanup(context.Background())

	require.ErrorContains(t, err, "delete Nexus endpoint umpire-endpoint")
	require.ErrorContains(t, err, "delete namespace umpire-ns")
}

func TestCreateRejectsAnIncompleteRequestBeforeAnyServerCall(t *testing.T) {
	workflow := &fakeWorkflow{}
	for _, probe := range []struct {
		name      string
		clients   Clients
		resources Resources
	}{
		{"no clients", Clients{}, Resources{Namespace: "umpire-ns"}},
		{"no namespace", Clients{Workflow: workflow, Operator: &fakeOperator{}}, Resources{}},
		{"endpoint without queue", Clients{Workflow: workflow, Operator: &fakeOperator{}},
			Resources{Namespace: "umpire-ns", NexusEndpoint: "umpire-endpoint"}},
	} {
		t.Run(probe.name, func(t *testing.T) {
			cleanup, err := Create(context.Background(), probe.clients, probe.resources)

			require.Nil(t, cleanup)
			require.ErrorIs(t, err, ErrInvalid)
			require.Empty(t, workflow.registered)
		})
	}
}
