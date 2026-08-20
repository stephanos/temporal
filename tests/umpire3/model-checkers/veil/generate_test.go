package veil

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestCompareReachableStatesAcceptsExactOracle(t *testing.T) {
	require.NoError(t, CompareReachableStates(testView()))
}

func TestCompareReachableStatesRejectsOmittedTransition(t *testing.T) {
	view := testView()
	view.Actions[0].Guard = protocol.FirstOrderFormula{
		Kind:    protocol.FirstOrderFormulaNot,
		Operand: &protocol.FirstOrderFormula{Kind: protocol.FirstOrderFormulaTrue},
	}

	err := CompareReachableStates(view)
	require.ErrorContains(t, err, "missing reachable state flag=one")
}

func TestGenerateProducesDeterministicCompleteSourceMap(t *testing.T) {
	view := testView()
	first, err := Generate(view, Interactive)
	require.NoError(t, err)
	second, err := Generate(view, Interactive)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, "NexusCancellationSound", first.Module)
	require.Equal(t, map[string]string{"flip": "Flip"}, first.ActionLabels)
	require.NotEmpty(t, first.Source)

	identifiers := make([]string, 0, len(first.ActionLabels))
	for identifier := range first.ActionLabels {
		identifiers = append(identifiers, identifier)
	}
	slices.Sort(identifiers)
	require.Equal(t, []string{"flip"}, identifiers)
	require.NotContains(t, string(first.Source), "let preFlag")
	require.Contains(t, string(first.Source), "invariant [CanonicalReachableEnvelope]")
	require.Contains(t, string(first.Source), "(sequential := true)")
	require.NotContains(t, string(first.Source), "maxDepth := 0,")
}

func TestGenerateConcreteModuleExportsCallableChecker(t *testing.T) {
	generated, err := Generate(testView(), Concrete)
	require.NoError(t, err)
	require.Equal(t, "NexusCancellationSoundConcrete", generated.Module)
	require.True(t, generated.ExportsModelChecker)
	require.NotEmpty(t, generated.Source)
}

func TestGenerateRecordsSMTTrustMode(t *testing.T) {
	reconstructed, err := GenerateWithTrust(testView(), Interactive, ReconstructedSMT)
	require.NoError(t, err)
	require.Equal(t, ReconstructedSMT, reconstructed.TrustMode)
	require.Contains(t, string(reconstructed.Source), "set_option veil.smt.trust false")

	trusted, err := GenerateWithTrust(testView(), Interactive, TrustedSMT)
	require.NoError(t, err)
	require.Equal(t, TrustedSMT, trusted.TrustMode)
	require.Contains(t, string(trusted.Source), "set_option veil.smt.trust true")
	require.NotEqual(t, reconstructed.Source, trusted.Source)
	require.Equal(t,
		strings.Replace(string(reconstructed.Source), "veil.smt.trust false", "veil.smt.trust true", 1),
		string(trusted.Source),
	)
}

func testView() protocol.FirstOrderView {
	return protocol.FirstOrderView{
		FormatVersion:  protocol.FirstOrderViewFormatVersion,
		Target:         protocol.TargetIDNexusCancellation,
		Property:       protocol.PropertyIDNexusCancellationWonExcludesSuccess,
		World:          "smoke",
		Variant:        "sound",
		SemanticHash:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		CanonicalModel: "Umpire3.Temporal.System.NexusCancellationFencing.behavior",
		Relation: protocol.FirstOrderRelation{
			Declaration: "Umpire3.Temporal.Targets.NexusCancellationFencing.firstOrderView",
			Axioms:      []string{},
			TrustBadge:  protocol.TrustBadgeKernel,
		},
		Bounds:      protocol.FirstOrderBounds{SymbolicDepth: 4, ConcreteStateLimit: 16},
		Sorts:       []protocol.FirstOrderSort{{Identifier: "bit", Kind: protocol.FirstOrderSortEnum, Values: []string{"zero", "one"}}},
		StateFields: []protocol.FirstOrderField{{Identifier: "flag", Sort: "bit"}},
		Initial: protocol.FirstOrderFormula{
			Kind:  protocol.FirstOrderFormulaEqual,
			Left:  &protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermField, Field: "flag"},
			Right: &protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermValue, Sort: "bit", Value: "zero"},
		},
		Actions: []protocol.FirstOrderAction{{
			Identifier: "flip",
			Guard: protocol.FirstOrderFormula{
				Kind:  protocol.FirstOrderFormulaEqual,
				Left:  &protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermField, Field: "flag"},
				Right: &protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermValue, Sort: "bit", Value: "zero"},
			},
			Updates: []protocol.FirstOrderUpdate{{
				Field: "flag",
				Value: protocol.FirstOrderTerm{Kind: protocol.FirstOrderTermValue, Sort: "bit", Value: "one"},
			}},
		}},
		Invariant: protocol.FirstOrderFormula{Kind: protocol.FirstOrderFormulaTrue},
		Oracle: protocol.FirstOrderOracle{
			ResultClass: protocol.ResultClassFiniteExhaustive,
			TrustBadge:  protocol.TrustBadgeCheckedCertificate,
			States: []protocol.FirstOrderState{
				{Fields: []protocol.FirstOrderBinding{{Field: "flag", Value: "zero"}}},
				{Fields: []protocol.FirstOrderBinding{{Field: "flag", Value: "one"}}},
			},
		},
	}
}
