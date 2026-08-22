package veil

import (
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

func TestBindingArtifactValidatesCanonicalMappings(t *testing.T) {
	view := testView()
	binding := BindingArtifact{
		FormatVersion:   BindingFormatVersion,
		BackendRevision: protocolchecker.VeilBackendRevision,
		SourceDigest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ArtifactDigest:  "derived",
		Binding: CompiledBinding{
			View:               view,
			SemanticBinding:    validSemanticBinding(),
			ModuleName:         "NexusCancellationSound",
			ConcreteModuleName: "NexusCancellationSoundConcrete",
			TrustMode:          ReconstructedSMT,
			ActionLabels: []protocolchecker.TraceSource{{
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
	wrongAction.Binding.ActionLabels = []protocolchecker.TraceSource{{
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
		"veil binding artifact digest must be derived")
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
	wrongTrust.SemanticBinding.TrustBadge = protocolcatalog.TrustBadgeKernel
	require.ErrorContains(t, wrongTrust.Validate(),
		"semantic binding trust badge does not match its axiom inventory")
}

func testBinding(t *testing.T, view protocolchecker.FirstOrderView, trustMode SMTTrustMode) BindingArtifact {
	t.Helper()
	binding := BindingArtifact{
		FormatVersion:   BindingFormatVersion,
		BackendRevision: protocolchecker.VeilBackendRevision,
		SourceDigest:    "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ArtifactDigest:  "derived",
		Binding: CompiledBinding{
			View:               view,
			SemanticBinding:    validSemanticBinding(),
			ModuleName:         "NexusCancellationSound",
			ConcreteModuleName: "NexusCancellationSoundConcrete",
			TrustMode:          trustMode,
			ActionLabels: []protocolchecker.TraceSource{{
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
		TrustBadge:  protocolcatalog.TrustBadgeKernelWithDeclaredAxioms,
	}
}

func testView() protocolchecker.FirstOrderView {
	return protocolchecker.FirstOrderView{
		FormatVersion:  protocolchecker.FirstOrderViewFormatVersion,
		Target:         protocolcatalog.TargetIDNexusCancellation,
		Property:       protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess,
		World:          "smoke",
		Variant:        "sound",
		SemanticHash:   "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		CanonicalModel: "Umpire3.Temporal.System.NexusCancellationFencing.behavior",
		Resources: []protocolchecker.FirstOrderResource{{
			Identifier: "operation", Kind: protocolcatalog.EntityKindNexusOperation,
		}},
		LiveOnlyActions:  []protocolcatalog.ActionKind{},
		ActivatingFaults: []protocolcatalog.FaultKind{},
		Relation: protocolchecker.FirstOrderRelation{
			Declaration: "Umpire3.Temporal.Targets.NexusCancellationFencing.firstOrderView",
			Axioms:      []string{},
			TrustBadge:  protocolcatalog.TrustBadgeKernel,
		},
		Bounds:      protocolchecker.FirstOrderBounds{SymbolicDepth: 4, ConcreteStateLimit: 16},
		Sorts:       []protocolchecker.FirstOrderSort{{Identifier: "bit", Kind: protocolchecker.FirstOrderSortEnum, Values: []string{"zero", "one"}}},
		StateFields: []protocolchecker.FirstOrderField{{Identifier: "flag", Sort: "bit"}},
		Initial: protocolchecker.FirstOrderFormula{
			Kind:  protocolchecker.FirstOrderFormulaEqual,
			Left:  &protocolchecker.FirstOrderTerm{Kind: protocolchecker.FirstOrderTermField, Field: "flag"},
			Right: &protocolchecker.FirstOrderTerm{Kind: protocolchecker.FirstOrderTermValue, Sort: "bit", Value: "zero"},
		},
		Actions: []protocolchecker.FirstOrderAction{{
			Identifier: "flip",
			Guard: protocolchecker.FirstOrderFormula{
				Kind:  protocolchecker.FirstOrderFormulaEqual,
				Left:  &protocolchecker.FirstOrderTerm{Kind: protocolchecker.FirstOrderTermField, Field: "flag"},
				Right: &protocolchecker.FirstOrderTerm{Kind: protocolchecker.FirstOrderTermValue, Sort: "bit", Value: "zero"},
			},
			Updates: []protocolchecker.FirstOrderUpdate{{
				Field: "flag",
				Value: protocolchecker.FirstOrderTerm{Kind: protocolchecker.FirstOrderTermValue, Sort: "bit", Value: "one"},
			}},
		}},
		Invariant: protocolchecker.FirstOrderFormula{Kind: protocolchecker.FirstOrderFormulaTrue},
		Oracle: protocolchecker.FirstOrderOracle{
			ResultClass: protocolcatalog.ResultClassFiniteExhaustive,
			TrustBadge:  protocolcatalog.TrustBadgeCheckedCertificate,
			States: []protocolchecker.FirstOrderState{
				{Fields: []protocolchecker.FirstOrderBinding{{Field: "flag", Value: "zero"}}},
				{Fields: []protocolchecker.FirstOrderBinding{{Field: "flag", Value: "one"}}},
			},
		},
	}
}
