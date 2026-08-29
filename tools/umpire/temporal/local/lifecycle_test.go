package local

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestCleanupIsBoundedOrderedAndIdempotent(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{
			{kind: ownedWorker},
			{kind: ownedClient},
			{kind: ownedServer},
		},
	}
	factory := newFactory(&recordingStarter{authority: backend})
	runtimeEnvironment, receipt := factory.Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)

	first := environment.Cleanup(context.Background(), cleanupCommand)
	second := environment.Cleanup(context.Background(), cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, first.Status())
	require.Equal(t, umpireruntime.ReceiptAccepted, second.Status())
	require.Equal(t, []string{"worker", "client", "server"}, backend.releaseOrder)
	require.Len(t, first.ReleasedResources(), 3)
	require.Empty(t, second.ReleasedResources())
}

func TestCleanupFailureRetainsOwnershipAndReturnsOnlyClosedCode(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup-failure")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{{kind: ownedServer}},
		stopErr:   errors.New("raw shutdown failure with /tmp/private-path"),
	}
	factory := newFactory(&recordingStarter{authority: backend})
	runtimeEnvironment, receipt := factory.Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)

	cleanupReceipt := environment.Cleanup(context.Background(), cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptFailed, cleanupReceipt.Status())
	require.Equal(t, "umpire.runtime.code.cleanup-failed", errorCode(cleanupReceipt))
	require.NotContains(t, receiptText(cleanupReceipt), "private-path")
	require.Empty(t, cleanupReceipt.ReleasedResources())
}

func TestCanceledCleanupRetainsOwnershipAndCanBeRetried(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup-canceled")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{resources: []ownedResource{{kind: ownedServer}}}
	factory := newFactory(&recordingStarter{authority: backend})
	runtimeEnvironment, receipt := factory.Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	workerReceipt := environment.StartWorker(
		context.Background(), prepareCommand, noopRegistration{},
	)
	require.Equal(t, umpireruntime.ReceiptAccepted, workerReceipt.Status())
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	canceledReceipt := environment.Cleanup(canceled, cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptCanceled, canceledReceipt.Status())
	require.Equal(t, "umpire.runtime.code.canceled", errorCode(canceledReceipt))
	require.Equal(t, umpireruntime.EvidenceSourceCleanup, receiptSource(canceledReceipt))
	require.Equal(t, "3", receiptField(canceledReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, backend.releaseOrder)

	retriedReceipt := environment.Cleanup(context.Background(), cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, retriedReceipt.Status())
	require.Equal(t, []string{"worker", "client", "server"}, backend.releaseOrder)
	require.Equal(t, "0", receiptField(retriedReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
}

type noopRegistration struct{}

func (noopRegistration) Register(worker.Registry) {}

func receiptSource(receipt umpireruntime.Receipt) string {
	facts := receipt.Facts()
	if len(facts) != 1 {
		return ""
	}
	return facts[0].SourceDefinitionID()
}

func receiptField(receipt umpireruntime.Receipt, definitionID string) string {
	for _, fact := range receipt.Facts() {
		for _, field := range fact.Fields() {
			if field.DefinitionID() == definitionID {
				return field.Value()
			}
		}
	}
	return ""
}

type recordingStarter struct {
	calls     int
	authority temporalAuthority
	err       error
}

func (s *recordingStarter) Start(context.Context) (temporalAuthority, error) {
	s.calls++
	return s.authority, s.err
}

type recordingAuthority struct {
	resources    []ownedResource
	client       client.Client
	clientErr    error
	workerErr    error
	stopErr      error
	releaseOrder []string
}

func (a *recordingAuthority) Connect(context.Context) error {
	if a.clientErr != nil {
		return a.clientErr
	}
	if !containsOwnedKind(a.resources, ownedClient) {
		a.resources = append(a.resources, ownedResource{kind: ownedClient})
	}
	return nil
}

func (a *recordingAuthority) SDKClient() client.Client { return a.client }

func (a *recordingAuthority) StartWorker(
	context.Context,
	string,
	string,
	WorkerRegistration,
) error {
	if !containsOwnedKind(a.resources, ownedWorker) {
		a.resources = append(a.resources, ownedResource{kind: ownedWorker})
	}
	return a.workerErr
}

func (a *recordingAuthority) Namespace() string { return "runtime.namespace.private" }
func (a *recordingAuthority) Endpoint() string  { return "127.0.0.1:12345" }

func (a *recordingAuthority) OwnedResources() []ownedResource {
	return append([]ownedResource{}, a.resources...)
}

func (a *recordingAuthority) Stop(context.Context) error {
	if a.stopErr != nil {
		return a.stopErr
	}
	for _, kind := range []ownedResourceKind{ownedWorker, ownedClient, ownedServer} {
		if containsOwnedKind(a.resources, kind) {
			a.releaseOrder = append(a.releaseOrder, string(kind))
		}
	}
	a.resources = []ownedResource{}
	return nil
}

func containsOwnedKind(resources []ownedResource, kind ownedResourceKind) bool {
	for _, resource := range resources {
		if resource.kind == kind {
			return true
		}
	}
	return false
}
