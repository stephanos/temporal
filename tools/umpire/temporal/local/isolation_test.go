package local

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestIsolationCollectionTransitions(t *testing.T) {
	request := testRequest(t, "umpire.local.isolation.transitions")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	realize, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	observe, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	operation := operationCorrelation(request)
	crossed := testRequest(t, "umpire.local.isolation.transitions.crossed")
	crossedPrepare, ok := crossed.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	crossedRealize, ok := crossed.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	crossedObserve, ok := crossed.Command(umpireruntime.CommandObserve)
	require.True(t, ok)

	tests := []struct {
		name    string
		mutate  func(*testing.T, *isolationCollection) error
		wantErr string
	}{
		{
			name: "wrong operation command",
			mutate: func(_ *testing.T, collection *isolationCollection) error {
				return collection.recordOperationCount(crossedPrepare, operation, 1)
			},
			wantErr: "unsupported isolation operation record",
		},
		{
			name: "wrong operation correlation",
			mutate: func(_ *testing.T, collection *isolationCollection) error {
				return collection.recordOperationCount(prepare, operation+"-crossed", 1)
			},
			wantErr: "unsupported isolation operation record",
		},
		{
			name: "duplicate operation",
			mutate: func(t *testing.T, collection *isolationCollection) error {
				require.NoError(t, collection.recordOperationCount(prepare, operation, 1))
				return collection.recordOperationCount(prepare, operation, 1)
			},
			wantErr: "unsupported isolation operation record",
		},
		{
			name: "wrong control command",
			mutate: func(_ *testing.T, collection *isolationCollection) error {
				return collection.recordControlCount(crossedRealize, operation, 1)
			},
			wantErr: "unsupported isolation control record",
		},
		{
			name: "wrong control correlation",
			mutate: func(_ *testing.T, collection *isolationCollection) error {
				return collection.recordControlCount(realize, operation+"-crossed", 1)
			},
			wantErr: "unsupported isolation control record",
		},
		{
			name: "duplicate control",
			mutate: func(t *testing.T, collection *isolationCollection) error {
				require.NoError(t, collection.recordControlCount(realize, operation, 1))
				return collection.recordControlCount(realize, operation, 1)
			},
			wantErr: "unsupported isolation control record",
		},
		{
			name: "wrong close command",
			mutate: func(_ *testing.T, collection *isolationCollection) error {
				return collection.closeInputs(crossedObserve)
			},
			wantErr: "unsupported isolation collection close",
		},
		{
			name: "duplicate close",
			mutate: func(t *testing.T, collection *isolationCollection) error {
				require.NoError(t, collection.closeInputs(observe))
				return collection.closeInputs(observe)
			},
			wantErr: "unsupported isolation collection close",
		},
		{
			name: "operation after close",
			mutate: func(t *testing.T, collection *isolationCollection) error {
				require.NoError(t, collection.closeInputs(observe))
				return collection.recordOperationCount(prepare, operation, 1)
			},
			wantErr: "unsupported isolation operation record",
		},
		{
			name: "control after close",
			mutate: func(t *testing.T, collection *isolationCollection) error {
				require.NoError(t, collection.closeInputs(observe))
				return collection.recordControlCount(realize, operation, 1)
			},
			wantErr: "unsupported isolation control record",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collection := newIsolationCollection(prepare, realize, observe, operation)
			require.EqualError(t, test.mutate(t, &collection), test.wantErr)
			require.Equal(t, isolationDecisionFailed, collection.decision())
		})
	}
}

func TestIsolationCollectionDecision(t *testing.T) {
	request := testRequest(t, "umpire.local.isolation.decision")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	realize, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	observe, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	operation := operationCorrelation(request)
	crossed := testRequest(t, "umpire.local.isolation.decision.crossed")
	crossedPrepare, ok := crossed.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)

	tests := []struct {
		name       string
		collection func(*testing.T) isolationCollection
		want       isolationDecision
		wantRepeat isolationDecision
	}{
		{
			name: "incomplete initialization",
			collection: func(_ *testing.T) isolationCollection {
				return newIsolationCollection(umpireruntime.Command{}, realize, observe, operation)
			},
			want: isolationDecisionFailed, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "zero counts",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 0))
				require.NoError(t, collection.recordControlCount(realize, operation, 0))
				require.NoError(t, collection.closeInputs(observe))
				return collection
			},
			want: isolationDecisionCanceled, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "one count each and closed",
			collection: func(t *testing.T) isolationCollection {
				return readyIsolationCollection(t, prepare, realize, observe, operation)
			},
			want: isolationDecisionReady, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "multiple operations",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 2))
				require.NoError(t, collection.recordControlCount(realize, operation, 1))
				require.NoError(t, collection.closeInputs(observe))
				return collection
			},
			want: isolationDecisionFailed, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "multiple controls",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 1))
				require.NoError(t, collection.recordControlCount(realize, operation, 2))
				require.NoError(t, collection.closeInputs(observe))
				return collection
			},
			want: isolationDecisionFailed, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "missing operation record",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordControlCount(realize, operation, 1))
				require.NoError(t, collection.closeInputs(observe))
				return collection
			},
			want: isolationDecisionCanceled, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "missing control record",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 1))
				require.NoError(t, collection.closeInputs(observe))
				return collection
			},
			want: isolationDecisionCanceled, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "open inputs",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 1))
				require.NoError(t, collection.recordControlCount(realize, operation, 1))
				return collection
			},
			want: isolationDecisionCanceled, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "invalid and incomplete",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.EqualError(t, collection.recordOperationCount(crossedPrepare, operation, 1), "unsupported isolation operation record")
				return collection
			},
			want: isolationDecisionFailed, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "multiple operations and open",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 2))
				require.NoError(t, collection.recordControlCount(realize, operation, 1))
				return collection
			},
			want: isolationDecisionFailed, wantRepeat: isolationDecisionFailed,
		},
		{
			name: "multiple operations and missing control",
			collection: func(t *testing.T) isolationCollection {
				collection := newIsolationCollection(prepare, realize, observe, operation)
				require.NoError(t, collection.recordOperationCount(prepare, operation, 2))
				require.NoError(t, collection.closeInputs(observe))
				return collection
			},
			want: isolationDecisionFailed, wantRepeat: isolationDecisionFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collection := test.collection(t)
			require.Equal(t, test.want, collection.decision())
			require.Equal(t, test.wantRepeat, collection.decision())
		})
	}
}

func TestIsolationCollectionInvalidationIsPermanent(t *testing.T) {
	request := testRequest(t, "umpire.local.isolation.permanent-invalidation")
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	realize, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	observe, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	operation := operationCorrelation(request)
	crossed := testRequest(t, "umpire.local.isolation.permanent-invalidation.crossed")
	crossedPrepare, ok := crossed.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)

	collection := newIsolationCollection(prepare, realize, observe, operation)
	require.EqualError(t, collection.recordOperationCount(crossedPrepare, operation, 1),
		"unsupported isolation operation record")
	require.NoError(t, collection.recordOperationCount(prepare, operation, 1))
	require.NoError(t, collection.recordControlCount(realize, operation, 1))
	require.NoError(t, collection.closeInputs(observe))
	require.Equal(t, isolationDecisionFailed, collection.decision())
}

func TestEnvironmentIsolationOrchestration(t *testing.T) {
	tests := []struct {
		name           string
		configure      func(*testing.T, *environment, *scriptedIsolationProbe, umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command)
		wantStatus     umpireruntime.ReceiptStatus
		wantCode       string
		wantProbeCalls int
		consumed       bool
	}{
		{
			name: "nil context",
			configure: func(_ *testing.T, environment *environment, _ *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				return nil, request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptCanceled, wantCode: runtimeCodeCanceled,
		},
		{
			name: "canceled context",
			configure: func(t *testing.T, _ *environment, _ *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				t.Cleanup(cancel)
				return ctx, request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptCanceled, wantCode: runtimeCodeCanceled,
		},
		{
			name: "unsupported isolation command",
			configure: func(t *testing.T, _ *environment, _ *scriptedIsolationProbe, _ umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				return context.Background(), testRequest(t, "umpire.local.isolation.orchestration.crossed-command").IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptUnsupported, wantCode: runtimeCodeUnsupported,
		},
		{
			name: "missing probe",
			configure: func(_ *testing.T, environment *environment, _ *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				environment.executionProbe = nil
				return context.Background(), request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptCanceled, wantCode: runtimeCodeCanceled, consumed: true,
		},
		{
			name: "failing probe",
			configure: func(_ *testing.T, _ *environment, probe *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				probe.verify = func(context.Context) error { return errors.New("probe failed") }
				return context.Background(), request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptFailed, wantCode: runtimeCodeFailed, wantProbeCalls: 1, consumed: true,
		},
		{
			name: "cancellation during probe",
			configure: func(_ *testing.T, _ *environment, probe *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				ctx, cancel := context.WithCancel(context.Background())
				probe.verify = func(context.Context) error {
					cancel()
					return context.Canceled
				}
				return ctx, request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptCanceled, wantCode: runtimeCodeCanceled, wantProbeCalls: 1, consumed: true,
		},
		{
			name: "crossed command input",
			configure: func(t *testing.T, environment *environment, _ *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				prepare, ok := request.Command(umpireruntime.CommandPrepare)
				require.True(t, ok)
				realize, ok := request.Command(umpireruntime.CommandRealize)
				require.True(t, ok)
				observe, ok := request.Command(umpireruntime.CommandObserve)
				require.True(t, ok)
				operation := operationCorrelation(request)
				environment.isolation = newIsolationCollection(
					prepare, realize, observe, operation,
				)
				crossed := testRequest(t, "umpire.local.isolation.orchestration.crossed-input")
				crossedPrepare, ok := crossed.Command(umpireruntime.CommandPrepare)
				require.True(t, ok)
				require.EqualError(t, environment.RecordOperationCount(crossedPrepare, operation, 1), "unsupported isolation operation record")
				require.NoError(t, environment.RecordOperationCount(
					prepare, operation, 1,
				))
				require.NoError(t, environment.RecordControlCount(
					realize, operation, 1,
				))
				require.NoError(t, environment.CloseIsolationInputs(observe))
				return context.Background(), request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptFailed, wantCode: runtimeCodeFailed, consumed: true,
		},
		{
			name: "crossed correlation input",
			configure: func(t *testing.T, environment *environment, _ *scriptedIsolationProbe, request umpireruntime.CheckedRunRequest) (context.Context, umpireruntime.Command) {
				prepare, ok := request.Command(umpireruntime.CommandPrepare)
				require.True(t, ok)
				realize, ok := request.Command(umpireruntime.CommandRealize)
				require.True(t, ok)
				observe, ok := request.Command(umpireruntime.CommandObserve)
				require.True(t, ok)
				operation := operationCorrelation(request)
				environment.isolation = newIsolationCollection(
					prepare, realize, observe, operation,
				)
				require.EqualError(t, environment.RecordControlCount(realize, operation+"-crossed", 1), "unsupported isolation control record")
				require.NoError(t, environment.RecordOperationCount(
					prepare, operation, 1,
				))
				require.NoError(t, environment.RecordControlCount(
					realize, operation, 1,
				))
				require.NoError(t, environment.CloseIsolationInputs(observe))
				return context.Background(), request.IsolationCommand()
			},
			wantStatus: umpireruntime.ReceiptFailed, wantCode: runtimeCodeFailed, consumed: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest(t, "umpire.local.isolation.orchestration."+strings.ReplaceAll(test.name, " ", "-"))
			environment := newEnvironment(request, &recordingAuthority{})
			prepare, ok := request.Command(umpireruntime.CommandPrepare)
			require.True(t, ok)
			realize, ok := request.Command(umpireruntime.CommandRealize)
			require.True(t, ok)
			observe, ok := request.Command(umpireruntime.CommandObserve)
			require.True(t, ok)
			operation := operationCorrelation(request)
			require.NoError(t, environment.RecordOperationCount(prepare, operation, 1))
			require.NoError(t, environment.RecordControlCount(realize, operation, 1))
			require.NoError(t, environment.CloseIsolationInputs(observe))
			probe := &scriptedIsolationProbe{}
			environment.executionProbe = probe
			ctx, command := test.configure(t, environment, probe, request)

			receipt := environment.Isolate(ctx, command)
			want := lifecycleReceipt(command, lifecycleFactIsolation, test.wantStatus,
				test.wantCode, nil, nil, environment.identities)
			require.Equal(t, want, receipt)
			require.Equal(t, test.wantProbeCalls, probe.calls)

			retryProbe := &scriptedIsolationProbe{}
			environment.executionProbe = retryProbe
			retry := environment.Isolate(context.Background(), request.IsolationCommand())
			if test.consumed {
				require.Equal(t, lifecycleReceipt(request.IsolationCommand(), lifecycleFactIsolation,
					umpireruntime.ReceiptFailed, runtimeCodeFailed, nil, nil, environment.identities), retry)
				require.Zero(t, retryProbe.calls)
			} else {
				require.Equal(t, lifecycleReceipt(request.IsolationCommand(), lifecycleFactIsolation,
					umpireruntime.ReceiptAccepted, "", nil, nil, environment.identities), retry)
				require.Equal(t, 1, retryProbe.calls)
			}
		})
	}
}

func readyIsolationCollection(
	t *testing.T,
	prepare umpireruntime.Command,
	realize umpireruntime.Command,
	observe umpireruntime.Command,
	operation string,
) isolationCollection {
	t.Helper()
	collection := newIsolationCollection(prepare, realize, observe, operation)
	require.NoError(t, collection.recordOperationCount(prepare, operation, 1))
	require.NoError(t, collection.recordControlCount(realize, operation, 1))
	require.NoError(t, collection.closeInputs(observe))
	return collection
}

type scriptedIsolationProbe struct {
	calls  int
	verify func(context.Context) error
}

func (p *scriptedIsolationProbe) Verify(ctx context.Context, _ string) error {
	p.calls++
	if p.verify != nil {
		return p.verify(ctx)
	}
	return nil
}
