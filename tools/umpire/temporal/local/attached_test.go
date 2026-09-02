package local

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestAttachedFactoryRejectsNilOrIncompleteAuthority(t *testing.T) {
	clientOne := &recordingAttachedClient{}
	var typedNil *recordingAttachedAuthority
	tests := []struct {
		name      string
		authority AttachedAuthority
	}{
		{name: "nil"},
		{name: "typed nil", authority: typedNil},
		{name: "nil client", authority: &recordingAttachedAuthority{
			namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
		}},
		{name: "empty namespace", authority: &recordingAttachedAuthority{
			client: clientOne, endpoint: "127.0.0.1:7233",
		}},
		{name: "empty endpoint", authority: &recordingAttachedAuthority{
			client: clientOne, namespace: "attached-namespace",
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory, err := NewAttachedFactory(test.authority)
			require.Error(t, err)
			require.Nil(t, factory)
		})
	}
}

func TestAttachedFactoryRejectsAuthorityDriftBeforeResourceAcquisition(t *testing.T) {
	clientOne := &recordingAttachedClient{}
	clientTwo := &recordingAttachedClient{}
	tests := []struct {
		name   string
		mutate func(*recordingAttachedAuthority)
	}{
		{name: "client", mutate: func(authority *recordingAttachedAuthority) {
			authority.client = clientTwo
		}},
		{name: "namespace", mutate: func(authority *recordingAttachedAuthority) {
			authority.namespace = "drifted-namespace"
		}},
		{name: "endpoint", mutate: func(authority *recordingAttachedAuthority) {
			authority.endpoint = "127.0.0.1:8233"
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authority := &recordingAttachedAuthority{
				client: clientOne, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
			}
			workers := &recordingAttachedWorkerFactory{}
			factory, err := newAttachedFactory(authority, workers.newWorker)
			require.NoError(t, err)
			test.mutate(authority)
			request := testRequest(t, "umpire.attached.drift."+test.name)
			command, ok := request.Command(umpireruntime.CommandPrepare)
			require.True(t, ok)

			environment, receipt := factory.Prepare(context.Background(), request, command)
			require.Nil(t, environment)
			require.Equal(t, umpireruntime.ReceiptFailed, receipt.Status())
			require.Empty(t, receipt.AcquiredResources())
			require.Equal(t, 0, workers.calls)
			require.NotContains(t, receiptText(receipt), "drifted")
		})
	}
}

func TestAttachedFactoryOwnsOnlyFreshRunWorkers(t *testing.T) {
	borrowedClient := &recordingAttachedClient{}
	authority := &recordingAttachedAuthority{
		client: borrowedClient, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}
	workers := &recordingAttachedWorkerFactory{}
	factory, err := newAttachedFactory(authority, workers.newWorker)
	require.NoError(t, err)

	identities := make([]Identities, 0, 2)
	for _, suffix := range []string{"one", "two"} {
		request := testRequest(t, "umpire.attached.reuse."+suffix)
		prepare, ok := request.Command(umpireruntime.CommandPrepare)
		require.True(t, ok)
		runtimeEnvironment, preparation := factory.Prepare(context.Background(), request, prepare)
		require.Equal(t, umpireruntime.ReceiptAccepted, preparation.Status())
		require.Equal(t, []umpireruntime.ResourceKind{
			umpireruntime.ResourceEnvironment,
		}, resourceKinds(preparation.AcquiredResources()))
		environment, ok := AsEnvironment(runtimeEnvironment)
		require.True(t, ok)
		identities = append(identities, environment.Identities())

		workerReceipt := environment.StartWorker(context.Background(), prepare, noopRegistration{})
		require.Equal(t, umpireruntime.ReceiptAccepted, workerReceipt.Status())
		require.Equal(t, []umpireruntime.ResourceKind{
			umpireruntime.ResourceWorker,
		}, resourceKinds(workerReceipt.AcquiredResources()))

		cleanup, ok := request.Command(umpireruntime.CommandCleanup)
		require.True(t, ok)
		first := environment.Cleanup(context.Background(), cleanup)
		second := environment.Cleanup(context.Background(), cleanup)
		require.Equal(t, umpireruntime.ReceiptAccepted, first.Status())
		require.Equal(t, []umpireruntime.ResourceKind{
			umpireruntime.ResourceEnvironment,
			umpireruntime.ResourceWorker,
		}, resourceKinds(first.ReleasedResources()))
		require.Equal(t, "0", receiptField(first, umpireruntime.EvidenceFieldOpenHandleCount))
		require.Equal(t, umpireruntime.ReceiptAccepted, second.Status())
		require.Empty(t, second.ReleasedResources())
	}

	require.Len(t, workers.workers, 2)
	require.NotEqual(t, workers.workers[0].taskQueue, workers.workers[1].taskQueue)
	for _, owned := range workers.workers {
		require.Equal(t, 1, owned.startCalls)
		require.Equal(t, 1, owned.stopCalls)
		require.Equal(t, 1, owned.closed)
		require.Same(t, borrowedClient, owned.client)
	}
	require.Equal(t, 0, borrowedClient.closeCalls)
	require.Equal(t, identities[0].Namespace, identities[1].Namespace)
	require.Equal(t, identities[0].Endpoint, identities[1].Endpoint)
	require.NotEqual(t, identities[0].TaskQueue, identities[1].TaskQueue)
	for _, identity := range []string{
		identities[0].Namespace, identities[0].Endpoint, identities[0].TaskQueue,
		identities[1].Namespace, identities[1].Endpoint, identities[1].TaskQueue,
	} {
		require.True(t, strings.HasPrefix(identity, "sha256:"))
	}
}

func TestAttachedIsolationScopesEachReusedNamespaceRun(t *testing.T) {
	requests := []umpireruntime.CheckedRunRequest{
		testRequest(t, "umpire.attached.isolation.one"),
		testRequest(t, "umpire.attached.isolation.two"),
	}
	workflowIdentities := []string{
		workflowCorrelation(requests[0]),
		workflowCorrelation(requests[1]),
	}
	borrowedClient := &isolationAttachedClient{workflowIdentities: workflowIdentities}
	workers := &recordingAttachedWorkerFactory{}
	factory, err := newAttachedFactory(&recordingAttachedAuthority{
		client: borrowedClient, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}, workers.newWorker)
	require.NoError(t, err)

	for index, request := range requests {
		prepare, ok := request.Command(umpireruntime.CommandPrepare)
		require.True(t, ok)
		runtimeEnvironment, preparation := factory.Prepare(context.Background(), request, prepare)
		require.Equal(t, umpireruntime.ReceiptAccepted, preparation.Status())
		environment, ok := AsEnvironment(runtimeEnvironment)
		require.True(t, ok)
		require.Equal(t, umpireruntime.ReceiptAccepted,
			environment.StartWorker(context.Background(), prepare, noopRegistration{}).Status())
		realize, ok := request.Command(umpireruntime.CommandRealize)
		require.True(t, ok)
		observe, ok := request.Command(umpireruntime.CommandObserve)
		require.True(t, ok)
		operation := operationCorrelation(request)
		require.NoError(t, environment.RecordOperationCount(prepare, operation, 1))
		require.NoError(t, environment.RecordControlCount(realize, operation, 1))
		require.NoError(t, environment.CloseIsolationInputs(observe))

		isolation := environment.Isolate(context.Background(), request.IsolationCommand())
		require.Equal(t, umpireruntime.ReceiptAccepted, isolation.Status())
		require.Equal(t, "WorkflowId = \""+workflowIdentities[index]+"\"",
			borrowedClient.queries[index])

		cleanup, ok := request.Command(umpireruntime.CommandCleanup)
		require.True(t, ok)
		require.Equal(t, umpireruntime.ReceiptAccepted,
			environment.Cleanup(context.Background(), cleanup).Status())
	}
}

func TestAttachedCleanupCancellationRetainsOwnedWorker(t *testing.T) {
	borrowedClient := &recordingAttachedClient{}
	workers := &recordingAttachedWorkerFactory{}
	factory, err := newAttachedFactory(&recordingAttachedAuthority{
		client: borrowedClient, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}, workers.newWorker)
	require.NoError(t, err)
	request := testRequest(t, "umpire.attached.cleanup-canceled")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	runtimeEnvironment, preparation := factory.Prepare(context.Background(), request, prepare)
	require.Equal(t, umpireruntime.ReceiptAccepted, preparation.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		environment.StartWorker(context.Background(), prepare, noopRegistration{}).Status())
	cleanup, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	canceledReceipt := environment.Cleanup(canceled, cleanup)
	require.Equal(t, umpireruntime.ReceiptCanceled, canceledReceipt.Status())
	require.Equal(t, "2", receiptField(canceledReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, canceledReceipt.ReleasedResources())
	require.Equal(t, 0, workers.workers[0].stopCalls)

	retried := environment.Cleanup(context.Background(), cleanup)
	require.Equal(t, umpireruntime.ReceiptAccepted, retried.Status())
	require.Equal(t, "0", receiptField(retried, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Equal(t, 1, workers.workers[0].stopCalls)
	require.Equal(t, 0, borrowedClient.closeCalls)
}

func TestAttachedCleanupFailureRetainsOwnershipUntilClosed(t *testing.T) {
	borrowedClient := &recordingAttachedClient{}
	workers := &recordingAttachedWorkerFactory{stopErrors: []error{
		errors.New("private attached worker cleanup failure"), nil,
	}}
	factory, err := newAttachedFactory(&recordingAttachedAuthority{
		client: borrowedClient, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}, workers.newWorker)
	require.NoError(t, err)
	request := testRequest(t, "umpire.attached.cleanup-failure")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	runtimeEnvironment, preparation := factory.Prepare(context.Background(), request, prepare)
	require.Equal(t, umpireruntime.ReceiptAccepted, preparation.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		environment.StartWorker(context.Background(), prepare, noopRegistration{}).Status())
	cleanup, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)

	failed := environment.Cleanup(context.Background(), cleanup)
	require.Equal(t, umpireruntime.ReceiptFailed, failed.Status())
	require.Equal(t, "umpire.runtime.code.cleanup-failed", errorCode(failed))
	require.Equal(t, "2", receiptField(failed, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, failed.ReleasedResources())
	require.NotContains(t, receiptText(failed), "private")

	closed := environment.Cleanup(context.Background(), cleanup)
	require.Equal(t, umpireruntime.ReceiptAccepted, closed.Status())
	require.Equal(t, "0", receiptField(closed, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Equal(t, 2, workers.workers[0].stopCalls)
	require.Equal(t, 1, workers.workers[0].closed)
	require.Equal(t, 0, borrowedClient.closeCalls)
}

func resourceKinds(resources []umpireruntime.Resource) []umpireruntime.ResourceKind {
	kinds := make([]umpireruntime.ResourceKind, len(resources))
	for index, resource := range resources {
		kinds[index] = resource.Kind()
	}
	return kinds
}

func workflowCorrelation(request umpireruntime.CheckedRunRequest) string {
	for _, correlation := range request.Correlations() {
		if correlation.Kind() == umpireruntime.CorrelationWorkflow {
			return correlation.Identity()
		}
	}
	return ""
}

type recordingAttachedAuthority struct {
	client    client.Client
	namespace string
	endpoint  string
}

func (a *recordingAttachedAuthority) SDKClient() client.Client { return a.client }
func (a *recordingAttachedAuthority) Namespace() string        { return a.namespace }
func (a *recordingAttachedAuthority) Endpoint() string         { return a.endpoint }

type recordingAttachedClient struct {
	client.Client
	closeCalls int
}

func (c *recordingAttachedClient) Close() { c.closeCalls++ }

type isolationAttachedClient struct {
	client.Client
	workflowIdentities []string
	queries            []string
}

func (c *isolationAttachedClient) ListWorkflow(
	_ context.Context,
	request *workflowservice.ListWorkflowExecutionsRequest,
) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	c.queries = append(c.queries, request.GetQuery())
	for _, identity := range c.workflowIdentities {
		if request.GetQuery() == "WorkflowId = \""+identity+"\"" {
			return &workflowservice.ListWorkflowExecutionsResponse{
				Executions: []*workflowpb.WorkflowExecutionInfo{{
					Execution: &commonpb.WorkflowExecution{WorkflowId: identity},
				}},
			}, nil
		}
	}
	executions := make([]*workflowpb.WorkflowExecutionInfo, len(c.workflowIdentities))
	for index, identity := range c.workflowIdentities {
		executions[index] = &workflowpb.WorkflowExecutionInfo{
			Execution: &commonpb.WorkflowExecution{WorkflowId: identity},
		}
	}
	return &workflowservice.ListWorkflowExecutionsResponse{Executions: executions}, nil
}

type recordingAttachedWorkerFactory struct {
	calls      int
	stopErrors []error
	workers    []*recordingAttachedWorker
}

func (f *recordingAttachedWorkerFactory) newWorker(
	sdkClient client.Client,
	taskQueue string,
	options worker.Options,
) (attachedWorker, error) {
	f.calls++
	owned := &recordingAttachedWorker{
		client: sdkClient, taskQueue: taskQueue, identity: options.Identity,
		stopErrors: append([]error{}, f.stopErrors...),
	}
	f.workers = append(f.workers, owned)
	return owned, nil
}

type recordingAttachedWorker struct {
	worker.Registry
	client     client.Client
	taskQueue  string
	identity   string
	startCalls int
	stopCalls  int
	closed     int
	stopErrors []error
}

func (w *recordingAttachedWorker) Start() error {
	w.startCalls++
	return nil
}

func (w *recordingAttachedWorker) Stop(context.Context) error {
	w.stopCalls++
	if len(w.stopErrors) > 0 {
		err := w.stopErrors[0]
		w.stopErrors = w.stopErrors[1:]
		if err != nil {
			return err
		}
	}
	w.closed++
	return nil
}
