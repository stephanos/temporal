package local

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
			{kind: ownedEnvironment},
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
	require.Equal(t, []string{"worker", "environment"}, backend.releaseOrder)
	require.Len(t, first.ReleasedResources(), 2)
	require.Empty(t, second.ReleasedResources())
}

func TestCleanupFailureRetainsOwnershipAndReturnsOnlyClosedCode(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup-failure")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{{kind: ownedEnvironment}},
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
	backend := &recordingAuthority{resources: []ownedResource{{kind: ownedEnvironment}}}
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
	require.Equal(t, "2", receiptField(canceledReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, backend.releaseOrder)

	retriedReceipt := environment.Cleanup(context.Background(), cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, retriedReceipt.Status())
	require.Equal(t, []string{"worker", "environment"}, backend.releaseOrder)
	require.Equal(t, "0", receiptField(retriedReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
}

func TestLifecycleFactsHaveDistinctOperationIdentities(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.fact-identities")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{resources: []ownedResource{{kind: ownedEnvironment}}}
	runtimeEnvironment, preparationReceipt := newFactory(
		&recordingStarter{authority: backend},
	).Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, preparationReceipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	workerReceipt := environment.StartWorker(
		context.Background(), prepareCommand, noopRegistration{},
	)
	require.Equal(t, umpireruntime.ReceiptAccepted, workerReceipt.Status())
	realizeCommand, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	observeCommand, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	operationIdentity := operationCorrelation(request)
	require.NoError(t, environment.RecordOperationCount(prepareCommand, operationIdentity, 1))
	require.NoError(t, environment.RecordControlCount(realizeCommand, operationIdentity, 1))
	require.NoError(t, environment.CloseIsolationInputs(observeCommand))
	isolationReceipt := environment.Isolate(context.Background(), request.IsolationCommand())
	require.Equal(t, umpireruntime.ReceiptAccepted, isolationReceipt.Status())
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	cleanupReceipt := environment.Cleanup(context.Background(), cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, cleanupReceipt.Status())

	identities := map[string]struct{}{}
	for _, receipt := range []umpireruntime.Receipt{
		preparationReceipt, workerReceipt, isolationReceipt, cleanupReceipt,
	} {
		facts := receipt.Facts()
		require.Len(t, facts, 1)
		identities[facts[0].DefinitionID()] = struct{}{}
	}
	require.Len(t, identities, 4)
}

func TestIsolationRequiresOneClosedOperationAndControlCollection(t *testing.T) {
	tests := []struct {
		name           string
		operationCount uint64
		controlCount   uint64
		closeInputs    bool
		executionCount int
		wantStatus     umpireruntime.ReceiptStatus
	}{
		{name: "exact closure", operationCount: 1, controlCount: 1, closeInputs: true, executionCount: 1, wantStatus: umpireruntime.ReceiptAccepted},
		{name: "second operation", operationCount: 2, controlCount: 1, closeInputs: true, executionCount: 1, wantStatus: umpireruntime.ReceiptFailed},
		{name: "second control", operationCount: 1, controlCount: 2, closeInputs: true, executionCount: 1, wantStatus: umpireruntime.ReceiptFailed},
		{name: "second namespace execution", operationCount: 1, controlCount: 1, closeInputs: true, executionCount: 2, wantStatus: umpireruntime.ReceiptFailed},
		{name: "open collection inputs", operationCount: 1, controlCount: 1, executionCount: 1, wantStatus: umpireruntime.ReceiptCanceled},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest(t, "umpire.local.environment.isolation."+strings.ReplaceAll(test.name, " ", "-"))
			prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
			require.True(t, ok)
			backend := &recordingAuthority{
				resources:               []ownedResource{{kind: ownedEnvironment}},
				isolationExecutionCount: test.executionCount,
			}
			runtimeEnvironment, receipt := newFactory(
				&recordingStarter{authority: backend},
			).Prepare(context.Background(), request, prepareCommand)
			require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
			environment, ok := AsEnvironment(runtimeEnvironment)
			require.True(t, ok)
			realizeCommand, ok := request.Command(umpireruntime.CommandRealize)
			require.True(t, ok)
			observeCommand, ok := request.Command(umpireruntime.CommandObserve)
			require.True(t, ok)
			operationIdentity := operationCorrelation(request)

			require.NoError(t, environment.RecordOperationCount(
				prepareCommand, operationIdentity, test.operationCount,
			))
			require.NoError(t, environment.RecordControlCount(
				realizeCommand, operationIdentity, test.controlCount,
			))
			if test.closeInputs {
				require.NoError(t, environment.CloseIsolationInputs(observeCommand))
			}

			isolationReceipt := environment.Isolate(
				context.Background(), request.IsolationCommand(),
			)
			require.Equal(t, test.wantStatus, isolationReceipt.Status())

			cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
			require.True(t, ok)
			cleanupReceipt := environment.Cleanup(context.Background(), cleanupCommand)
			require.Equal(t, umpireruntime.ReceiptAccepted, cleanupReceipt.Status())
		})
	}
}

func TestCleanupDeadlineReturnsTimeoutCompatibleReceipt(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup-deadline")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{resources: []ownedResource{{kind: ownedEnvironment}}}
	runtimeEnvironment, receipt := newFactory(
		&recordingStarter{authority: backend},
	).Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	deadline, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	cleanupReceipt := environment.Cleanup(deadline, cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptCanceled, cleanupReceipt.Status())
	require.Equal(t, "umpire.runtime.code.timed-out", errorCode(cleanupReceipt))
	require.Equal(t, umpireruntime.EvidenceSourceCleanup, receiptSource(cleanupReceipt))
	require.Equal(t, "1", receiptField(cleanupReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, backend.releaseOrder)
}

func TestCleanupDeadlineReachedDuringStopReturnsTimeoutCompatibleReceipt(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup-stop-deadline")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{{kind: ownedEnvironment}},
		stopFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	runtimeEnvironment, receipt := newFactory(
		&recordingStarter{authority: backend},
	).Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	cleanupReceipt := environment.Cleanup(deadline, cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptCanceled, cleanupReceipt.Status())
	require.Equal(t, "umpire.runtime.code.timed-out", errorCode(cleanupReceipt))
	require.Equal(t, "1", receiptField(cleanupReceipt, umpireruntime.EvidenceFieldOpenHandleCount))
	require.Empty(t, cleanupReceipt.ReleasedResources())
}

func TestConcreteCleanupFailureDominatesExpiredDeadline(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.cleanup-deadline-failure")
	prepareCommand, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{{kind: ownedEnvironment}},
		stopFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return errors.Join(ctx.Err(), errors.New("private concrete cleanup failure"))
		},
	}
	runtimeEnvironment, receipt := newFactory(
		&recordingStarter{authority: backend},
	).Prepare(context.Background(), request, prepareCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)
	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	deadline, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	cleanupReceipt := environment.Cleanup(deadline, cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptFailed, cleanupReceipt.Status())
	require.Equal(t, "umpire.runtime.code.cleanup-failed", errorCode(cleanupReceipt))
	require.NotContains(t, receiptText(cleanupReceipt), "private")
	require.Empty(t, cleanupReceipt.ReleasedResources())
}

type noopRegistration struct{}

func (noopRegistration) Register(worker.Registry) {}

func operationCorrelation(request umpireruntime.CheckedRunRequest) string {
	for _, correlation := range request.Correlations() {
		if correlation.Kind() == umpireruntime.CorrelationOperation {
			return correlation.Identity()
		}
	}
	return ""
}

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
	resources               []ownedResource
	client                  client.Client
	clientErr               error
	connectCalls            int
	workerErr               error
	stopErr                 error
	stopFunc                func(context.Context) error
	releaseOrder            []string
	isolationExecutionCount int
}

type recordingIsolationProbe struct {
	executionCount int
}

func (p recordingIsolationProbe) Verify(context.Context, string) error {
	if p.executionCount == 1 {
		return nil
	}
	return errors.New("namespace execution count is not one")
}

func (a *recordingAuthority) isolationProbe(string) executionIsolationProbe {
	count := a.isolationExecutionCount
	if count == 0 {
		count = 1
	}
	return recordingIsolationProbe{executionCount: count}
}

func (a *recordingAuthority) Connect(context.Context) error {
	a.connectCalls++
	return a.clientErr
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

func (a *recordingAuthority) Stop(ctx context.Context) error {
	if a.stopFunc != nil {
		return a.stopFunc(ctx)
	}
	if a.stopErr != nil {
		return a.stopErr
	}
	for _, kind := range []ownedResourceKind{ownedWorker, ownedEnvironment} {
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
