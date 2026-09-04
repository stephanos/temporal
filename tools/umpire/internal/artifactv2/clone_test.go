package artifactv2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyArtifactDocumentsPreservesZeroAndEmptyCollections(t *testing.T) {
	tests := []struct {
		name   string
		source any
		copy   func() any
	}{
		{
			name:   "zero Experiment",
			source: Experiment{},
			copy:   func() any { return CopyExperiment(Experiment{}) },
		},
		{
			name:   "empty Experiment collections",
			source: emptyCopyTestExperiment(),
			copy: func() any {
				return CopyExperiment(emptyCopyTestExperiment())
			},
		},
		{
			name:   "zero RuntimeConfiguration",
			source: RuntimeConfiguration{},
			copy: func() any {
				return CopyRuntimeConfiguration(RuntimeConfiguration{})
			},
		},
		{
			name:   "empty RuntimeConfiguration collections",
			source: emptyCopyTestRuntimeConfiguration(),
			copy: func() any {
				return CopyRuntimeConfiguration(emptyCopyTestRuntimeConfiguration())
			},
		},
		{
			name:   "zero ExperimentRun",
			source: ExperimentRun{},
			copy:   func() any { return CopyExperimentRun(ExperimentRun{}) },
		},
		{
			name:   "empty ExperimentRun collections",
			source: emptyCopyTestExperimentRun(),
			copy: func() any {
				return CopyExperimentRun(emptyCopyTestExperimentRun())
			},
		},
		{
			name:   "zero RawEvidence",
			source: RawEvidence{},
			copy:   func() any { return CopyRawEvidence(RawEvidence{}) },
		},
		{
			name:   "empty RawEvidence collections",
			source: emptyCopyTestRawEvidence(),
			copy: func() any {
				return CopyRawEvidence(emptyCopyTestRawEvidence())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.source, test.copy())
		})
	}
}

func TestCopyExperimentIsolatesEveryMutableDescendant(t *testing.T) {
	requireIndependentCopy(t, copyTestExperiment, CopyExperiment, mutateCopyTestExperiment)
}

func TestCopyRuntimeConfigurationIsolatesEveryMutableDescendant(t *testing.T) {
	requireIndependentCopy(
		t,
		copyTestRuntimeConfiguration,
		CopyRuntimeConfiguration,
		mutateCopyTestRuntimeConfiguration,
	)
}

func TestCopyExperimentRunIsolatesEveryMutableDescendant(t *testing.T) {
	requireIndependentCopy(t, copyTestExperimentRun, CopyExperimentRun, mutateCopyTestExperimentRun)
}

func TestCopyRawEvidenceIsolatesEveryMutableDescendantAndPreservesScalarValues(t *testing.T) {
	requireIndependentCopy(t, copyTestRawEvidence, CopyRawEvidence, mutateCopyTestRawEvidence)

	fields := CopyRawEvidence(copyTestRawEvidence()).Facts[1].Fields
	require.Equal(t, []any{true, nil, json.Number("-7"), "value"}, []any{
		fields[0].Value,
		fields[1].Value,
		fields[2].Value,
		fields[3].Value,
	})
}

func requireIndependentCopy[T any](
	t *testing.T,
	fixture func() T,
	copyValue func(T) T,
	mutate func(*T),
) {
	t.Helper()

	source := fixture()
	copied := copyValue(source)
	require.Equal(t, source, copied)
	mutate(&copied)
	require.Equal(t, fixture(), source)

	source = fixture()
	copied = copyValue(source)
	mutate(&source)
	require.Equal(t, fixture(), copied)
}

func emptyCopyTestExperiment() Experiment {
	return Experiment{
		Plan: DrivePlan{
			Bindings:                           []Binding{},
			SymbolicRoles:                      []Role{},
			ModelPreconditions:                 []Precondition{},
			RequestedActions:                   []ModelValue{},
			ModelOutcomes:                      []ModelValue{},
			ResultingStates:                    []ModelValue{},
			LinearExtension:                    []Occurrence{},
			SelectedChoices:                    []ModelValue{},
			SelectedVariants:                   []ModelValue{},
			RequestedFaults:                    []ModelValue{},
			CapabilityRequirementDefinitionIDs: []string{},
			Checkpoints:                        []Checkpoint{},
			KnownGaps:                          []KnownGap{},
			Provenance: Provenance{
				SourceDefinitionIDs: []string{},
				SourceLocations:     []SourceLocation{},
			},
		},
		Properties:                          []Property{},
		ObservationRequirementDefinitionIDs: []string{},
		Provenance: Provenance{
			SourceDefinitionIDs: []string{},
			SourceLocations:     []SourceLocation{},
		},
	}
}

func emptyCopyTestRuntimeConfiguration() RuntimeConfiguration {
	return RuntimeConfiguration{
		AuthorityProfile: AuthorityProfile{
			RequiredCapabilityDefinitionIDs: []string{},
		},
		PhaseLimits:         []PhaseLimit{},
		ParticipantBindings: []ParticipantBinding{},
		KnownGaps:           []KnownGap{},
		Provenance: Provenance{
			SourceDefinitionIDs: []string{},
			SourceLocations:     []SourceLocation{},
		},
	}
}

func emptyCopyTestExperimentRun() ExperimentRun {
	return ExperimentRun{
		PhaseOutcomes:   []PhaseOutcome{},
		ControlAttempts: []ControlAttempt{},
		SourceClosures:  []SourceClosure{},
		Limits:          []PhaseLimit{},
		KnownGaps:       []KnownGap{},
		Provenance: Provenance{
			SourceDefinitionIDs: []string{},
			SourceLocations:     []SourceLocation{},
		},
	}
}

func emptyCopyTestRawEvidence() RawEvidence {
	return RawEvidence{
		Sources:   []RawEvidenceSource{},
		Facts:     []RawEvidenceFact{},
		KnownGaps: []KnownGap{},
		Provenance: Provenance{
			SourceDefinitionIDs: []string{},
			SourceLocations:     []SourceLocation{},
		},
	}
}

func copyTestExperiment() Experiment {
	return Experiment{
		Plan: DrivePlan{
			Bindings: []Binding{{
				RoleDefinitionID: "role.original",
				Value:            ModelValue{DefinitionID: "value.binding", Value: "binding"},
			}},
			SymbolicRoles: []Role{{DefinitionID: "role.symbolic", ValueKind: "text"}},
			ModelPreconditions: []Precondition{{
				DefinitionID: "precondition.original",
				Left: Operand{
					Kind:  "literal",
					Value: copyTestModelValuePointer("value.left", "left"),
				},
				Right: Operand{
					Kind:  "literal",
					Value: copyTestModelValuePointer("value.right", "right"),
				},
			}},
			RequestedActions: []ModelValue{{DefinitionID: "action.original", Value: "action"}},
			ModelOutcomes:    []ModelValue{{DefinitionID: "outcome.original", Value: "outcome"}},
			ResultingStates:  []ModelValue{{DefinitionID: "state.original", Value: "state"}},
			LinearExtension: []Occurrence{{
				DefinitionID:         "occurrence.original",
				AuthoredDefinitionID: copyTestStringPointer("occurrence.authored"),
			}},
			SelectedChoices:  []ModelValue{{DefinitionID: "choice.original", Value: "choice"}},
			SelectedVariants: []ModelValue{{DefinitionID: "variant.original", Value: "variant"}},
			RequestedFaults:  []ModelValue{{DefinitionID: "fault.original", Value: "fault"}},
			CapabilityRequirementDefinitionIDs: []string{
				"capability.original",
			},
			Checkpoints: []Checkpoint{{
				Transition: "1",
				Observations: []ModelValue{{
					DefinitionID: "observation.original",
					Value:        "observation",
				}},
			}},
			KnownGaps:  copyTestKnownGaps("plan"),
			Provenance: copyTestProvenance("plan"),
		},
		Properties: []Property{{
			DefinitionID:             "property.original",
			RequirementDefinitionIDs: []string{"requirement.original"},
		}},
		ObservationRequirementDefinitionIDs: []string{"observation.requirement.original"},
		Provenance:                          copyTestProvenance("experiment"),
	}
}

func mutateCopyTestExperiment(document *Experiment) {
	document.Properties[0].DefinitionID = "property.changed"
	document.Properties[0].RequirementDefinitionIDs[0] = "requirement.changed"
	document.ObservationRequirementDefinitionIDs[0] = "observation.requirement.changed"
	mutateCopyTestDrivePlan(&document.Plan)
	mutateCopyTestProvenance(&document.Provenance)
}

func mutateCopyTestDrivePlan(plan *DrivePlan) {
	plan.Bindings[0].RoleDefinitionID = "role.changed"
	plan.SymbolicRoles[0].DefinitionID = "role.changed"
	plan.ModelPreconditions[0].DefinitionID = "precondition.changed"
	plan.ModelPreconditions[0].Left.Value.Value = "left changed"
	plan.ModelPreconditions[0].Right.Value.Value = "right changed"
	plan.RequestedActions[0].Value = "action changed"
	plan.ModelOutcomes[0].Value = "outcome changed"
	plan.ResultingStates[0].Value = "state changed"
	plan.LinearExtension[0].DefinitionID = "occurrence.changed"
	*plan.LinearExtension[0].AuthoredDefinitionID = "occurrence.authored.changed"
	plan.SelectedChoices[0].Value = "choice changed"
	plan.SelectedVariants[0].Value = "variant changed"
	plan.RequestedFaults[0].Value = "fault changed"
	plan.CapabilityRequirementDefinitionIDs[0] = "capability.changed"
	plan.Checkpoints[0].Transition = "2"
	plan.Checkpoints[0].Observations[0].Value = "observation changed"
	mutateCopyTestKnownGaps(plan.KnownGaps)
	mutateCopyTestProvenance(&plan.Provenance)
}

func copyTestRuntimeConfiguration() RuntimeConfiguration {
	return RuntimeConfiguration{
		AuthorityProfile: AuthorityProfile{
			DefinitionID:                    "authority.original",
			RequiredCapabilityDefinitionIDs: []string{"authority.capability.original"},
		},
		PhaseLimits: []PhaseLimit{{Phase: "preparation", MaxAttempts: "1"}},
		ParticipantBindings: []ParticipantBinding{{
			ParticipantDefinitionID: "participant.original",
			CapabilityDefinitionIDs: []string{"participant.capability.original"},
		}},
		KnownGaps:  copyTestKnownGaps("runtime"),
		Provenance: copyTestProvenance("runtime"),
	}
}

func mutateCopyTestRuntimeConfiguration(document *RuntimeConfiguration) {
	document.AuthorityProfile.RequiredCapabilityDefinitionIDs[0] = "authority.capability.changed"
	document.PhaseLimits[0].MaxAttempts = "2"
	document.ParticipantBindings[0].ParticipantDefinitionID = "participant.changed"
	document.ParticipantBindings[0].CapabilityDefinitionIDs[0] = "participant.capability.changed"
	mutateCopyTestKnownGaps(document.KnownGaps)
	mutateCopyTestProvenance(&document.Provenance)
}

func copyTestExperimentRun() ExperimentRun {
	return ExperimentRun{
		PhaseOutcomes: []PhaseOutcome{{
			Phase:                "preparation",
			StartedAtUnixMillis:  copyTestNaturalPointer("1"),
			FinishedAtUnixMillis: copyTestNaturalPointer("2"),
			Code:                 copyTestStringPointer("phase.code.original"),
		}},
		ControlAttempts: []ControlAttempt{{
			OccurrenceDefinitionID:  "occurrence.original",
			ReceiptFactDefinitionID: copyTestStringPointer("receipt.original"),
			Code:                    copyTestStringPointer("control.code.original"),
		}},
		SourceClosures: []SourceClosure{{
			SourceDefinitionID: "source.original",
			RecordCount:        "1",
		}},
		Cleanup: CleanupOutcome{
			Status: "complete",
			Code:   copyTestStringPointer("cleanup.code.original"),
		},
		Limits:     []PhaseLimit{{Phase: "preparation", MaxAttempts: "1"}},
		KnownGaps:  copyTestKnownGaps("run"),
		Provenance: copyTestProvenance("run"),
	}
}

func mutateCopyTestExperimentRun(document *ExperimentRun) {
	document.PhaseOutcomes[0].Phase = "realization"
	*document.PhaseOutcomes[0].StartedAtUnixMillis = "3"
	*document.PhaseOutcomes[0].FinishedAtUnixMillis = "4"
	*document.PhaseOutcomes[0].Code = "phase.code.changed"
	document.ControlAttempts[0].OccurrenceDefinitionID = "occurrence.changed"
	*document.ControlAttempts[0].ReceiptFactDefinitionID = "receipt.changed"
	*document.ControlAttempts[0].Code = "control.code.changed"
	document.SourceClosures[0].RecordCount = "2"
	*document.Cleanup.Code = "cleanup.code.changed"
	document.Limits[0].MaxAttempts = "2"
	mutateCopyTestKnownGaps(document.KnownGaps)
	mutateCopyTestProvenance(&document.Provenance)
}

func copyTestRawEvidence() RawEvidence {
	return RawEvidence{
		Sources: []RawEvidenceSource{{
			SourceDefinitionID: "source.original",
			Status:             "closed",
			FactCount:          "2",
		}},
		Facts: []RawEvidenceFact{
			{
				FactDefinitionID:        "fact.cause",
				SourceDefinitionID:      "source.original",
				CausalFactDefinitionIDs: []string{},
				Fields:                  []RawEvidenceField{},
			},
			{
				FactDefinitionID:        "fact.original",
				SourceDefinitionID:      "source.original",
				Ordinal:                 "1",
				CausalFactDefinitionIDs: []string{"fact.cause"},
				Fields: []RawEvidenceField{
					{FieldDefinitionID: "field.boolean", Disposition: "plain", Value: true},
					{FieldDefinitionID: "field.nil", Disposition: "plain", Value: nil},
					{FieldDefinitionID: "field.number", Disposition: "plain", Value: json.Number("-7")},
					{FieldDefinitionID: "field.string", Disposition: "plain", Value: "value"},
				},
			},
		},
		KnownGaps:  copyTestKnownGaps("evidence"),
		Provenance: copyTestProvenance("evidence"),
	}
}

func mutateCopyTestRawEvidence(document *RawEvidence) {
	document.Sources[0].Status = "partial"
	document.Facts[0].FactDefinitionID = "fact.cause.changed"
	document.Facts[1].CausalFactDefinitionIDs[0] = "fact.cause.changed"
	document.Facts[1].Fields[0].FieldDefinitionID = "field.nil.changed"
	mutateCopyTestKnownGaps(document.KnownGaps)
	mutateCopyTestProvenance(&document.Provenance)
}

func copyTestProvenance(prefix string) Provenance {
	return Provenance{
		SourceDefinitionIDs: []string{prefix + ".source.original"},
		SourceLocations: []SourceLocation{{
			Path: prefix + "/source/original",
		}},
	}
}

func mutateCopyTestProvenance(provenance *Provenance) {
	provenance.SourceDefinitionIDs[0] = "source.changed"
	provenance.SourceLocations[0].Path = "source/changed"
}

func copyTestKnownGaps(prefix string) []KnownGap {
	return []KnownGap{{
		Kind:    "unsupported",
		Code:    prefix + ".gap.original",
		Subject: copyTestStringPointer(prefix + ".subject.original"),
		Detail:  copyTestStringPointer(prefix + ".detail.original"),
	}}
}

func mutateCopyTestKnownGaps(gaps []KnownGap) {
	gaps[0].Code = "gap.changed"
	*gaps[0].Subject = "subject.changed"
	*gaps[0].Detail = "detail.changed"
}

func copyTestModelValuePointer(definitionID string, value string) *ModelValue {
	return &ModelValue{DefinitionID: definitionID, Value: value}
}

func copyTestNaturalPointer(value Natural) *Natural {
	return &value
}

func copyTestStringPointer(value string) *string {
	return &value
}
