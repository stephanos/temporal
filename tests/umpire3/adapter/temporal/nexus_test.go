package temporal

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/execution/observation"
	"go.temporal.io/server/tests/umpire3/mutation"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
	"go.temporal.io/server/tests/umpire3/regression"
	"go.temporal.io/server/tests/umpire3/scenario"
	scenarionexus "go.temporal.io/server/tests/umpire3/scenario/nexus"
)

func TestNexusFactoryPreparationFailure(t *testing.T) {
	factory := newNexusFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{}, errors.New("cluster unavailable")
	}, nexusOptions{})

	_, err := factory.Prepare(context.Background(), loadNexusExperiment(t))
	require.ErrorContains(t, err, "cluster unavailable")
}

func TestNexusFactoryLearnsMintedIdentity(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Actions[1].Bindings = []protocolexperiment.Binding{{
		Symbol: "learned-operation", Type: "identity", Projection: "operation-id",
	}}
	factory := newNexusFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "server-operation"}, nil
	}, nexusOptions{})
	prepared, err := factory.Prepare(context.Background(), experiment)
	require.NoError(t, err)
	session := prepared.Session

	_, err = session.Realize(context.Background(), experiment.Actions[0], execution.Bindings{})
	require.NoError(t, err)
	evidence, err := session.Realize(context.Background(), experiment.Actions[1], execution.Bindings{})
	require.NoError(t, err)
	require.Equal(t, "server-operation", evidence.GroundedBindings["learned-operation"])
}

func TestEveryNexusRealizerHonorsCancellation(t *testing.T) {
	experiment := loadNexusExperiment(t)
	factory := newNexusFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, nexusOptions{})
	for _, action := range experiment.Actions {
		prepared, err := factory.Prepare(context.Background(), experiment)
		require.NoError(t, err)
		session := prepared.Session
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = session.Realize(ctx, action, execution.Bindings{})
		require.ErrorIs(t, err, context.Canceled, action.Kind)
	}
}

func TestNexusNegativeControlChangesImplementationBehavior(t *testing.T) {
	experiment := loadNexusExperiment(t)
	probe := func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "server-operation"}, nil
	}

	sound, err := execution.Run(context.Background(), execution.Request{
		Experiment:  experiment,
		Environment: newNexusFactory(probe, nexusOptions{}),
	})
	require.NoError(t, err)
	require.Equal(t, execution.ClaimConforming, sound.Claim.Kind)
	require.Equal(t, "build", sound.Environment.BuildID)

	faulty, err := execution.Run(context.Background(), execution.Request{
		Experiment:  experiment,
		Environment: newNexusFactory(probe, nexusOptions{AllowStaleSuccess: true}),
	})
	require.NoError(t, err)
	require.Equal(t, execution.ClaimViolating, faulty.Claim.Kind)
}

func TestTypedNexusRegressionFacadeRuns(t *testing.T) {
	operation := scenarionexus.Operation("operation")
	authored := scenarionexus.Scenario("typed-nexus-cancellation", operation,
		scenario.OnePath(operation.CancelWithRetry(), operation.CancellationSafety()))
	factory := newNexusFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, nexusOptions{})

	regression.RequireRegression(t, authored, regression.WithEnvironment(factory))
}

func TestNexusTaskTransportDeterminesVisibleCompletion(t *testing.T) {
	experiment := loadNexusExperiment(t)
	probe := func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace"}, nil
	}

	for _, testCase := range []struct {
		name              string
		allowStaleSuccess bool
		wantClaim         execution.ClaimKind
		wantReportSuccess bool
	}{
		{name: "sound worker", wantClaim: execution.ClaimConforming},
		{
			name:              "faulty worker",
			allowStaleSuccess: true,
			wantClaim:         execution.ClaimViolating,
			wantReportSuccess: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			transport := &fakeNexusTaskTransport{}
			result, err := execution.Run(context.Background(), execution.Request{
				Experiment: experiment,
				Environment: newNexusFactory(probe, nexusOptions{
					AllowStaleSuccess: testCase.allowStaleSuccess,
					TaskTransport:     transport,
				}),
			})
			require.NoError(t, err)
			require.Equal(t, testCase.wantClaim, result.Claim.Kind)
			require.Equal(t, "server-minted-operation", result.Bindings["operation"])
			require.Equal(t, []bool{testCase.wantReportSuccess}, transport.reportedSuccess)
			require.Equal(t, "fake-matching", result.Observations[len(result.Observations)-1].Source)
			require.True(t, transport.cleaned)
		})
	}
}

func TestNexusObservationsCarryCausalEvidence(t *testing.T) {
	experiment := loadNexusExperiment(t)
	factory := newNexusFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, nexusOptions{})
	result, err := execution.Run(context.Background(), execution.Request{
		Experiment:  experiment,
		Environment: factory,
	})
	require.NoError(t, err)
	for _, observed := range result.Observations {
		require.NotEmpty(t, observed.Source)
		require.NotEmpty(t, observed.CausalReference)
	}
}

func TestNexusAdapterEmitsRawFactsWithoutPropertyTruth(t *testing.T) {
	experiment := loadNexusExperiment(t)
	factory := newNexusFactory(func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, nexusOptions{})
	prepared, err := factory.Prepare(context.Background(), experiment)
	require.NoError(t, err)
	_, emitsFacts := prepared.Session.(execution.FactSession)
	require.True(t, emitsFacts)

	result, err := execution.Run(context.Background(), execution.Request{
		Experiment:  experiment,
		Environment: factory,
	})
	require.NoError(t, err)
	require.NotEmpty(t, result.Facts)
	require.Contains(t, factKinds(result.Facts), observation.NexusCancellationWindow)
	require.Equal(t, execution.ClaimConforming, result.Claim.Kind)
}

func TestNexusViolationMinimizesWithoutChangingCheckpoint(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Scope.Bounds.MaxDepth = 10
	experiment.Actions = append(experiment.Actions[:5], append([]protocolexperiment.Action{
		{Identifier: "irrelevant-retry", Kind: "retry-task",
			AllowedOutcomes:      []protocolexperiment.ActionOutcome{protocolexperiment.ActionOutcomeApplied},
			RequiredCapabilities: []string{"nexus-worker-control"}},
	}, experiment.Actions[5:]...)...)
	probe := func(context.Context) (clusterInfo, error) {
		return clusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}
	original, err := execution.Run(context.Background(), execution.Request{
		Experiment:  experiment,
		Environment: newNexusFactory(probe, nexusOptions{AllowStaleSuccess: true}),
	})
	require.NoError(t, err)
	require.Equal(t, execution.ClaimViolating, original.Claim.Kind, original.Claim.Reason)

	minimized, err := mutation.MinimizeActions(context.Background(), experiment,
		func(ctx context.Context, candidate protocolexperiment.Experiment) (execution.Result, error) {
			return execution.Run(ctx, execution.Request{
				Experiment:  candidate,
				Environment: newNexusFactory(probe, nexusOptions{AllowStaleSuccess: true}),
			})
		})
	require.NoError(t, err)
	for _, action := range minimized.Actions {
		require.NotEqual(t, "irrelevant-retry", action.Identifier)
	}
	require.Less(t, len(minimized.Actions), len(experiment.Actions))
}

func loadNexusExperiment(t *testing.T) protocolexperiment.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}

type fakeNexusTaskTransport struct {
	reportedSuccess []bool
	cleaned         bool
}

func (f *fakeNexusTaskTransport) Dispatch(context.Context) (NexusTask, error) {
	return NexusTask{
		OperationID: "server-minted-operation",
		Source:      "fake-matching",
		Reference:   "fake/task",
	}, nil
}

func (f *fakeNexusTaskTransport) Complete(_ context.Context, completion NexusTaskCompletion) (NexusTaskOutcome, error) {
	f.reportedSuccess = append(f.reportedSuccess, completion.ReportSuccess)
	return NexusTaskOutcome{
		SuccessVisible: completion.ReportSuccess,
		Source:         "fake-matching",
		Reference:      "fake/result",
	}, nil
}

func (f *fakeNexusTaskTransport) Cleanup(context.Context) error {
	f.cleaned = true
	return nil
}

func factKinds(facts []observation.Fact) []string {
	kinds := make([]string, 0, len(facts))
	for _, fact := range facts {
		switch {
		case fact.History != nil:
			kinds = append(kinds, fact.History.EventType)
		case fact.Mechanism != nil:
			kinds = append(kinds, fact.Mechanism.Action)
		case fact.Window != nil:
			kinds = append(kinds, fact.Window.Purpose)
		default:
		}
	}
	return kinds
}
