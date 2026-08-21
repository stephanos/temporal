package veil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestBindingArtifactValidatesCanonicalMappings(t *testing.T) {
	view := testView()
	binding := BindingArtifact{
		FormatVersion:   BindingFormatVersion,
		BackendRevision: protocol.VeilBackendRevision,
		SourceDigest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ArtifactDigest:  "derived",
		Binding: CompiledBinding{
			View:               view,
			SemanticBinding:    validSemanticBinding(),
			ModuleName:         "NexusCancellationSound",
			ConcreteModuleName: "NexusCancellationSoundConcrete",
			TrustMode:          ReconstructedSMT,
			ActionLabels: []protocol.TraceSource{{
				Action: "flip", BackendAction: "Flip",
			}},
			FieldLabels: []NameBinding{{Identifier: "flag", BackendIdentifier: "flag"}},
			EnumLabels: []EnumBinding{{
				Identifier: "bit", BackendIdentifier: "Bit",
				Values: []NameBinding{
					{Identifier: "zero", BackendIdentifier: "Zero"},
					{Identifier: "one", BackendIdentifier: "One"},
				},
			}},
			PropertyLabel: "NexusCancellationWonExcludesSuccess",
		},
	}
	require.NoError(t, binding.DeriveArtifactDigest())
	require.NoError(t, binding.Validate())
	require.NoError(t, binding.ValidateAgainst(view))

	wrongAction := binding
	wrongAction.Binding.ActionLabels = []protocol.TraceSource{{
		Action: "missing", BackendAction: "Flip",
	}}
	require.ErrorContains(t, wrongAction.Validate(), `action label 0 maps "missing"; expected "flip"`)

	wrongView := view
	wrongView.World = "other"
	require.ErrorContains(t, binding.ValidateAgainst(wrongView),
		"veil binding does not match the first-order view")
}

func TestBindingArtifactRejectsAuthoredDigest(t *testing.T) {
	binding := BindingArtifact{ArtifactDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}
	require.ErrorContains(t, binding.DeriveArtifactDigest(),
		"Veil binding artifact digest must be derived")
}

func TestCompiledBindingRequiresKernelCheckedSemanticBinding(t *testing.T) {
	binding := testBinding(t, testView(), ReconstructedSMT)

	missing := binding.Binding
	missing.SemanticBinding = SemanticBinding{}
	require.ErrorContains(t, missing.Validate(), "semantic binding declaration is required")

	forbidden := binding.Binding
	forbidden.SemanticBinding.Axioms = append(forbidden.SemanticBinding.Axioms, "sorryAx")
	require.ErrorContains(t, forbidden.Validate(), `semantic binding has invalid axiom "sorryAx"`)

	wrongTrust := binding.Binding
	wrongTrust.SemanticBinding.TrustBadge = protocol.TrustBadgeKernel
	require.ErrorContains(t, wrongTrust.Validate(),
		"semantic binding trust badge does not match its axiom inventory")
}

func testBinding(t *testing.T, view protocol.FirstOrderView, trustMode SMTTrustMode) BindingArtifact {
	t.Helper()
	binding := BindingArtifact{
		FormatVersion:   BindingFormatVersion,
		BackendRevision: protocol.VeilBackendRevision,
		SourceDigest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ArtifactDigest:  "derived",
		Binding: CompiledBinding{
			View:               view,
			SemanticBinding:    validSemanticBinding(),
			ModuleName:         "NexusCancellationSound",
			ConcreteModuleName: "NexusCancellationSoundConcrete",
			TrustMode:          trustMode,
			ActionLabels: []protocol.TraceSource{{
				Action: "flip", BackendAction: "Flip",
			}},
			FieldLabels: []NameBinding{{Identifier: "flag", BackendIdentifier: "flag"}},
			EnumLabels: []EnumBinding{{
				Identifier: "bit", BackendIdentifier: "Bit",
				Values: []NameBinding{
					{Identifier: "zero", BackendIdentifier: "Zero"},
					{Identifier: "one", BackendIdentifier: "One"},
				},
			}},
			PropertyLabel: "NexusCancellationWonExcludesSuccess",
		},
	}
	require.NoError(t, binding.DeriveArtifactDigest())
	return binding
}

func validSemanticBinding() SemanticBinding {
	return SemanticBinding{
		Declaration: "Umpire3.Temporal.Veil.NexusCancellationFencing.soundSemanticBinding",
		Axioms:      []string{"propext", "Classical.choice", "Quot.sound"},
		TrustBadge:  protocol.TrustBadgeKernelWithDeclaredAxioms,
	}
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
		Resources: []protocol.FirstOrderResource{{
			Identifier: "operation", Kind: protocol.EntityKindNexusOperation,
		}},
		LiveOnlyActions:  []protocol.ActionKind{},
		ActivatingFaults: []protocol.FaultKind{},
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
