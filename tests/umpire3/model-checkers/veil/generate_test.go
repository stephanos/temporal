package veil

import (
	"bytes"
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
	reconstructedModel := strings.SplitN(string(reconstructed.Source),
		"\nnamespace Umpire3Veil.Generated", 2)[0]
	trustedModel := strings.SplitN(string(trusted.Source),
		"\nnamespace Umpire3Veil.Generated", 2)[0]
	require.Equal(t,
		strings.Replace(reconstructedModel, "veil.smt.trust false", "veil.smt.trust true", 1),
		trustedModel,
	)
	require.Contains(t, string(reconstructed.Source), "trustMode := .reconstructed")
	require.Contains(t, string(trusted.Source), "trustMode := .trusted")
	require.NotEqual(t, reconstructed.ModelHash, trusted.ModelHash)
}

func TestValidateGeneratedSemanticsRejectsRendererMutation(t *testing.T) {
	view := testView()
	generated, err := Generate(view, Interactive)
	require.NoError(t, err)

	mutatedGuard := bytes.Replace(generated.Source,
		[]byte("  require (flag = Zero)"), []byte("  require (flag = One)"), 1)
	require.NotEqual(t, generated.Source, mutatedGuard)
	require.ErrorContains(t, validateGeneratedSemantics(view, mutatedGuard),
		`generated action "flip" guard`)

	mutatedUpdate := bytes.Replace(generated.Source,
		[]byte("  flag := One"), []byte("  flag := Zero"), 1)
	require.NotEqual(t, generated.Source, mutatedUpdate)
	require.ErrorContains(t, validateGeneratedSemantics(view, mutatedUpdate),
		`generated action "flip" successor`)
}

func TestGenerateNamesFiniteMembersOfUninterpretedSorts(t *testing.T) {
	view := testView()
	view.Sorts = []protocol.FirstOrderSort{{
		Identifier: "node", Kind: protocol.FirstOrderSortUninterpreted,
		Values: []string{}, Cardinality: 2,
	}}
	view.StateFields[0].Sort = "node"
	view.Initial.Right = &protocol.FirstOrderTerm{
		Kind: protocol.FirstOrderTermValue, Sort: "node", Value: "member-0",
	}
	view.Actions[0].Guard.Right = &protocol.FirstOrderTerm{
		Kind: protocol.FirstOrderTermValue, Sort: "node", Value: "member-0",
	}
	view.Actions[0].Updates[0].Value = protocol.FirstOrderTerm{
		Kind: protocol.FirstOrderTermValue, Sort: "node", Value: "member-1",
	}
	view.Oracle.States = []protocol.FirstOrderState{
		{Fields: []protocol.FirstOrderBinding{{Field: "flag", Value: "member-0"}}},
		{Fields: []protocol.FirstOrderBinding{{Field: "flag", Value: "member-1"}}},
	}

	generated, err := Generate(view, Interactive)
	require.NoError(t, err)
	require.Contains(t, string(generated.Source), "type node")
	require.Contains(t, string(generated.Source), "immutable individual NodeMember0 : node")
	require.Contains(t, string(generated.Source),
		"assumption [NodeMembersDistinct] (NodeMember0 ≠ NodeMember1)")
	require.Contains(t, string(generated.Source),
		"assumption [NodeMembersExhaustive] ∀ value : node, (value = NodeMember0) ∨ (value = NodeMember1)")
	require.Contains(t, string(generated.Source), "{ node := Fin 2 }")
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
