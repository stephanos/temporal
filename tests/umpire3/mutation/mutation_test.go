package mutation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestCoverageCatalogIncludesGeneratedProtobufAndSemanticDimensions(t *testing.T) {
	t.Parallel()

	coverage, err := DefaultCoverageCatalog()
	require.NoError(t, err)
	require.Contains(t, coverage, CoveragePoint{Kind: CoverageProtobuf, Identifier: "presence"})
	require.Contains(t, coverage, CoveragePoint{Kind: CoverageProperty, Identifier: "nexus-operation.closure"})
	require.Contains(t, coverage, CoveragePoint{Kind: CoverageProfile, Identifier: "grpc-only-black-box"})
}

func TestSeededTypedMutationsAreDeterministicAndHonestlyBounded(t *testing.T) {
	t.Parallel()

	experiment := loadMutationExperiment(t)
	reason := "cancel"
	for index := range experiment.Actions {
		if experiment.Actions[index].Kind == string(protocolcatalog.ActionKindRequestCancellation) {
			experiment.Actions[index].Arguments = []protocolexperiment.NamedValue{{
				Name: "reason", Value: protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &reason},
			}}
		}
	}
	replacement := "mutated"
	request := MutationRequest{
		Experiment: experiment, Seed: 42, MaxCandidates: 3,
		Values:        []protocolexperiment.Value{{Type: protocolexperiment.ValueString, Text: &replacement}},
		TopologyKinds: []protocolcatalog.EntityKind{protocolcatalog.EntityKindCallback},
	}
	first, err := Mutate(request)
	require.NoError(t, err)
	second, err := Mutate(request)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first.Selected, 3)
	require.False(t, first.Complete)
	require.Positive(t, first.Omitted)
}

func TestInvalidFaultMutationHasExplicitReason(t *testing.T) {
	t.Parallel()

	report, err := Mutate(MutationRequest{
		Experiment: loadMutationExperiment(t), Seed: 1, MaxCandidates: 20,
		FaultScopes: []protocolexperiment.FaultScope{{}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, report.Rejected)
	require.NotEmpty(t, report.Rejected[0].Reason)
}

func loadMutationExperiment(t *testing.T) protocolexperiment.Experiment {
	t.Helper()
	file, err := os.Open("../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	experiment, err := protocolexperiment.DecodeExperiment(file, protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
