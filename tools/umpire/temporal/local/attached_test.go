package local

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/testing/await"
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

func TestDistinctAttachedFactoriesPrepareConcurrentlyWithSeparateBindings(t *testing.T) {
	type prepared struct {
		environment Environment
		receipt     umpireruntime.Receipt
	}
	requests := []umpireruntime.CheckedRunRequest{
		testRequest(t, "umpire.attached.concurrent.one"),
		testRequest(t, "umpire.attached.concurrent.two"),
	}
	clients := []*recordingAttachedClient{{}, {}}
	namespaces := []string{"attached-namespace-one", "attached-namespace-two"}
	endpoints := []string{"127.0.0.1:7233", "127.0.0.1:7234"}
	factories := make([]umpireruntime.EnvironmentFactory, len(requests))
	for index := range factories {
		factory, err := newAttachedFactory(&recordingAttachedAuthority{
			client: clients[index], namespace: namespaces[index], endpoint: endpoints[index],
		}, (&recordingAttachedWorkerFactory{}).newWorker)
		require.NoError(t, err)
		factories[index] = factory
	}
	commands := make([]umpireruntime.Command, len(requests))
	for index, request := range requests {
		var ok bool
		commands[index], ok = request.Command(umpireruntime.CommandPrepare)
		require.True(t, ok)
	}

	results := make([]prepared, len(requests))
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			runtimeEnvironment, receipt := factories[index].Prepare(
				context.Background(), requests[index], commands[index],
			)
			environment, _ := AsEnvironment(runtimeEnvironment)
			results[index] = prepared{environment: environment, receipt: receipt}
		}()
	}
	close(start)
	group.Wait()

	for index, result := range results {
		require.NotNil(t, result.environment)
		require.Equal(t, umpireruntime.ReceiptAccepted, result.receipt.Status())
		require.Equal(t, []umpireruntime.ResourceKind{
			umpireruntime.ResourceEnvironment,
		}, resourceKinds(result.receipt.AcquiredResources()))
		require.Same(t, clients[index], result.environment.Client())
		cleanup, ok := requests[index].Command(umpireruntime.CommandCleanup)
		require.True(t, ok)
		closed := result.environment.Cleanup(context.Background(), cleanup)
		require.Equal(t, umpireruntime.ReceiptAccepted, closed.Status())
		require.Equal(t, "0", receiptField(closed, umpireruntime.EvidenceFieldOpenHandleCount))
		require.Equal(t, 0, clients[index].closeCalls)
	}
	first := results[0].environment.Identities()
	second := results[1].environment.Identities()
	require.NotEqual(t, first.Namespace, second.Namespace)
	require.NotEqual(t, first.Endpoint, second.Endpoint)
	require.NotEqual(t, first.TaskQueue, second.TaskQueue)
}

func TestAttachedFactoryRejectsConcurrentPreparationUntilCleanup(t *testing.T) {
	type prepared struct {
		environment umpireruntime.Environment
		receipt     umpireruntime.Receipt
	}
	borrowedClient := &recordingAttachedClient{}
	workers := &recordingAttachedWorkerFactory{}
	factory, err := newAttachedFactory(&recordingAttachedAuthority{
		client: borrowedClient, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}, workers.newWorker)
	require.NoError(t, err)
	requests := []umpireruntime.CheckedRunRequest{
		testRequest(t, "umpire.attached.same-factory.one"),
		testRequest(t, "umpire.attached.same-factory.two"),
	}
	commands := make([]umpireruntime.Command, len(requests))
	for index, request := range requests {
		var ok bool
		commands[index], ok = request.Command(umpireruntime.CommandPrepare)
		require.True(t, ok)
	}

	results := make([]prepared, len(requests))
	start := make(chan struct{})
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			results[index].environment, results[index].receipt = factory.Prepare(
				context.Background(), requests[index], commands[index],
			)
		}()
	}
	close(start)
	group.Wait()

	accepted := -1
	for index, result := range results {
		switch result.receipt.Status() {
		case umpireruntime.ReceiptAccepted:
			require.Equal(t, -1, accepted)
			accepted = index
			require.NotNil(t, result.environment)
		case umpireruntime.ReceiptFailed:
			require.Nil(t, result.environment)
			require.Empty(t, result.receipt.AcquiredResources())
		default:
			require.FailNow(t, "unexpected preparation status", result.receipt.Status())
		}
	}
	require.NotEqual(t, -1, accepted)

	environment, ok := AsEnvironment(results[accepted].environment)
	require.True(t, ok)
	cleanup, ok := requests[accepted].Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		environment.Cleanup(context.Background(), cleanup).Status())

	nextRequest := testRequest(t, "umpire.attached.same-factory.after-cleanup")
	nextPrepare, ok := nextRequest.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	nextRuntimeEnvironment, nextReceipt := factory.Prepare(
		context.Background(), nextRequest, nextPrepare,
	)
	require.Equal(t, umpireruntime.ReceiptAccepted, nextReceipt.Status())
	nextEnvironment, ok := AsEnvironment(nextRuntimeEnvironment)
	require.True(t, ok)
	nextCleanup, ok := nextRequest.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		nextEnvironment.Cleanup(context.Background(), nextCleanup).Status())
	require.Equal(t, 0, workers.calls)
	require.Equal(t, 0, borrowedClient.closeCalls)
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

func TestAttachedWorkerStartCancellationReturnsBeforeBlockedStartAndCleansEventually(t *testing.T) {
	startEntered := make(chan struct{})
	startRelease := make(chan struct{})
	workers := &recordingAttachedWorkerFactory{
		startEntered: startEntered,
		startBlock:   startRelease,
	}
	factory, err := newAttachedFactory(&recordingAttachedAuthority{
		client: &recordingAttachedClient{}, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}, workers.newWorker)
	require.NoError(t, err)
	request := testRequest(t, "umpire.attached.start-canceled")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	runtimeEnvironment, preparation := factory.Prepare(context.Background(), request, prepare)
	require.Equal(t, umpireruntime.ReceiptAccepted, preparation.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)

	startContext, cancelStart := context.WithCancel(context.Background())
	startReceipts := make(chan umpireruntime.Receipt, 1)
	go func() {
		startReceipts <- environment.StartWorker(startContext, prepare, noopRegistration{})
	}()
	<-startEntered
	cancelStart()

	var startReceipt umpireruntime.Receipt
	select {
	case startReceipt = <-startReceipts:
	case <-time.After(time.Second):
		close(startRelease)
		<-startReceipts
		require.FailNow(t, "worker start did not honor cancellation while blocked")
	}
	require.Equal(t, umpireruntime.ReceiptCanceled, startReceipt.Status())
	require.Equal(t, []umpireruntime.ResourceKind{
		umpireruntime.ResourceWorker,
	}, resourceKinds(startReceipt.AcquiredResources()))

	cleanup, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	cleanupReceipts := make(chan umpireruntime.Receipt, 1)
	go func() {
		cleanupReceipts <- environment.Cleanup(context.Background(), cleanup)
	}()
	close(startRelease)
	require.Equal(t, umpireruntime.ReceiptAccepted, (<-cleanupReceipts).Status())
	require.Equal(t, 1, workers.workers[0].stopCount())
	require.Equal(t, 1, workers.workers[0].closedCount())
}

func TestAttachedWorkerFailuresRetainOnlyAcquiredResources(t *testing.T) {
	tests := []struct {
		name         string
		workers      *recordingAttachedWorkerFactory
		wantAcquired []umpireruntime.ResourceKind
		wantReleased []umpireruntime.ResourceKind
	}{
		{
			name:         "worker factory failure",
			workers:      &recordingAttachedWorkerFactory{newWorkerErr: errors.New("private worker factory failure")},
			wantAcquired: []umpireruntime.ResourceKind{},
			wantReleased: []umpireruntime.ResourceKind{umpireruntime.ResourceEnvironment},
		},
		{
			name:         "nil worker",
			workers:      &recordingAttachedWorkerFactory{returnNil: true},
			wantAcquired: []umpireruntime.ResourceKind{},
			wantReleased: []umpireruntime.ResourceKind{umpireruntime.ResourceEnvironment},
		},
		{
			name:         "worker start failure",
			workers:      &recordingAttachedWorkerFactory{startErr: errors.New("private worker start failure")},
			wantAcquired: []umpireruntime.ResourceKind{umpireruntime.ResourceWorker},
			wantReleased: []umpireruntime.ResourceKind{
				umpireruntime.ResourceEnvironment,
				umpireruntime.ResourceWorker,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			borrowedClient := &recordingAttachedClient{}
			factory, err := newAttachedFactory(&recordingAttachedAuthority{
				client: borrowedClient, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
			}, test.workers.newWorker)
			require.NoError(t, err)
			request := testRequest(t, "umpire.attached.worker-failure."+strings.ReplaceAll(test.name, " ", "-"))
			prepare, ok := request.Command(umpireruntime.CommandPrepare)
			require.True(t, ok)
			runtimeEnvironment, preparation := factory.Prepare(context.Background(), request, prepare)
			require.Equal(t, umpireruntime.ReceiptAccepted, preparation.Status())
			environment, ok := AsEnvironment(runtimeEnvironment)
			require.True(t, ok)

			started := environment.StartWorker(context.Background(), prepare, noopRegistration{})
			require.Equal(t, umpireruntime.ReceiptFailed, started.Status())
			require.Equal(t, test.wantAcquired, resourceKinds(started.AcquiredResources()))
			require.NotContains(t, receiptText(started), "private")

			cleanup, ok := request.Command(umpireruntime.CommandCleanup)
			require.True(t, ok)
			closed := environment.Cleanup(context.Background(), cleanup)
			require.Equal(t, umpireruntime.ReceiptAccepted, closed.Status())
			require.Equal(t, test.wantReleased, resourceKinds(closed.ReleasedResources()))
			require.Equal(t, "0", receiptField(closed, umpireruntime.EvidenceFieldOpenHandleCount))
			require.Equal(t, 0, borrowedClient.closeCalls)
		})
	}
}

func TestAttachedWorkerStopCancellationReturnsBeforeBlockedStopAndClosesOnce(t *testing.T) {
	stopEntered := make(chan struct{})
	stopRelease := make(chan struct{})
	workers := &recordingAttachedWorkerFactory{
		stopEntered: stopEntered,
		stopBlock:   stopRelease,
	}
	factory, err := newAttachedFactory(&recordingAttachedAuthority{
		client: &recordingAttachedClient{}, namespace: "attached-namespace", endpoint: "127.0.0.1:7233",
	}, workers.newWorker)
	require.NoError(t, err)
	request := testRequest(t, "umpire.attached.stop-canceled")
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

	cleanupContext, cancelCleanup := context.WithCancel(context.Background())
	cleanupReceipts := make(chan umpireruntime.Receipt, 1)
	go func() {
		cleanupReceipts <- environment.Cleanup(cleanupContext, cleanup)
	}()
	<-stopEntered
	cancelCleanup()

	var canceled umpireruntime.Receipt
	select {
	case canceled = <-cleanupReceipts:
	case <-time.After(time.Second):
		close(stopRelease)
		<-cleanupReceipts
		require.FailNow(t, "worker stop did not honor cancellation while blocked")
	}
	require.Equal(t, umpireruntime.ReceiptCanceled, canceled.Status())
	require.Equal(t, "2", receiptField(canceled, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, canceled.ReleasedResources())

	close(stopRelease)
	await.RequireTrue(t, func() bool {
		return workers.workers[0].closedCount() == 1
	}, time.Second, time.Millisecond)
	closed := environment.Cleanup(context.Background(), cleanup)
	require.Equal(t, umpireruntime.ReceiptAccepted, closed.Status())
	require.Equal(t, "0", receiptField(closed, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Equal(t, 1, workers.workers[0].stopCount())
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
	calls        int
	newWorkerErr error
	returnNil    bool
	startErr     error
	stopErrors   []error
	workers      []*recordingAttachedWorker
	startBlock   <-chan struct{}
	startEntered chan<- struct{}
	stopBlock    <-chan struct{}
	stopEntered  chan<- struct{}
}

func (f *recordingAttachedWorkerFactory) newWorker(
	sdkClient client.Client,
	taskQueue string,
	options worker.Options,
) (attachedWorker, error) {
	f.calls++
	if f.newWorkerErr != nil {
		return nil, f.newWorkerErr
	}
	if f.returnNil {
		return nil, nil
	}
	owned := &recordingAttachedWorker{
		client: sdkClient, taskQueue: taskQueue, identity: options.Identity,
		startErr: f.startErr, stopErrors: append([]error{}, f.stopErrors...), startBlock: f.startBlock,
		startEntered: f.startEntered, stopBlock: f.stopBlock, stopEntered: f.stopEntered,
	}
	f.workers = append(f.workers, owned)
	return owned, nil
}

type recordingAttachedWorker struct {
	worker.Registry
	mu           sync.Mutex
	client       client.Client
	taskQueue    string
	identity     string
	startCalls   int
	startErr     error
	stopCalls    int
	closed       int
	stopErrors   []error
	startBlock   <-chan struct{}
	startEntered chan<- struct{}
	stopBlock    <-chan struct{}
	stopEntered  chan<- struct{}
}

func (w *recordingAttachedWorker) Start() error {
	w.mu.Lock()
	w.startCalls++
	startErr := w.startErr
	startEntered := w.startEntered
	startBlock := w.startBlock
	w.mu.Unlock()
	if startEntered != nil {
		close(startEntered)
	}
	if startBlock != nil {
		<-startBlock
	}
	return startErr
}

func (w *recordingAttachedWorker) Stop() error {
	w.mu.Lock()
	w.stopCalls++
	stopEntered := w.stopEntered
	stopBlock := w.stopBlock
	var err error
	if len(w.stopErrors) > 0 {
		err = w.stopErrors[0]
		w.stopErrors = w.stopErrors[1:]
	}
	w.mu.Unlock()
	if stopEntered != nil {
		close(stopEntered)
	}
	if stopBlock != nil {
		<-stopBlock
	}
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.closed++
	w.mu.Unlock()
	return nil
}

func (w *recordingAttachedWorker) stopCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopCalls
}

func (w *recordingAttachedWorker) closedCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}
