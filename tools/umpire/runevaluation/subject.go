package runevaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

// SubjectDefinition binds one definition ID to its admitted behavior.
type SubjectDefinition struct {
	DefinitionID        string
	Kind                string
	BehaviorFingerprint string
}

// SubjectLimit binds one named ExperimentSpec limit.
type SubjectLimit struct {
	Path  string
	Value string
	Unit  string
}

// SubjectKnownGap preserves the nullable fields of an admitted known gap.
type SubjectKnownGap struct {
	Kind           string
	Code           string
	SubjectPresent bool
	Subject        string
	DetailPresent  bool
	Detail         string
}

// SubjectBinding is the generated, inspectable pin for the exact Run Evaluation subject.
type SubjectBinding struct {
	ExperimentSHA256                      string
	ExperimentFormatVersion               string
	DrivePlanFormatVersion                string
	ExperimentArtifactChecksum            string
	DrivePlanArtifactChecksum             string
	DefinitionIDs                         []string
	BehaviorFingerprints                  []string
	Limits                                []SubjectLimit
	KnownGaps                             []SubjectKnownGap
	Query                                 SubjectDefinition
	Properties                            []SubjectDefinition
	ObservationRequirementDefinitionIDs   []string
	ObservationProgram                    SubjectDefinition
	ImplementationLinkID                  string
	ImplementationLinkBehaviorFingerprint string
	ImplementationLinkSourceTarget        SubjectDefinition
	ImplementationLinkDestinationTarget   SubjectDefinition
	ImplementationLinkDiagnosticPresent   bool
}

// PinSubject derives the sole supported Run Evaluation subject from admitted canonical bytes.
func PinSubject(experimentBytes []byte) (SubjectBinding, error) {
	experiment, err := artifact.DecodeExperimentV2(experimentBytes)
	if err != nil {
		return SubjectBinding{}, err
	}
	if !exactCallerClosureExperiment(experiment) {
		return SubjectBinding{}, errors.New("run evaluation subject is not the exact caller-closure profile")
	}

	implementationLink := callerClosureImplementationLink()
	digest := sha256.Sum256(experimentBytes)
	properties := make([]SubjectDefinition, len(experiment.Properties))
	for index, property := range experiment.Properties {
		properties[index] = SubjectDefinition{
			DefinitionID:        property.DefinitionID,
			BehaviorFingerprint: property.BehaviorFingerprint,
		}
	}
	return SubjectBinding{
		ExperimentSHA256:           "sha256:" + hex.EncodeToString(digest[:]),
		ExperimentFormatVersion:    experiment.FormatVersion,
		DrivePlanFormatVersion:     experiment.Plan.FormatVersion,
		ExperimentArtifactChecksum: experiment.ArtifactChecksum,
		DrivePlanArtifactChecksum:  experiment.Plan.ArtifactChecksum,
		DefinitionIDs:              subjectDefinitionIDs(experiment),
		BehaviorFingerprints:       subjectBehaviorFingerprints(experiment),
		Limits:                     subjectLimits(experiment.Plan.ExpandedLimits),
		KnownGaps:                  subjectKnownGaps(experiment.Plan.KnownGaps),
		Query:                      SubjectDefinition{DefinitionID: experiment.Plan.QueryDefinitionID, BehaviorFingerprint: experiment.Plan.QueryBehaviorFingerprint},
		Properties:                 properties,
		ObservationRequirementDefinitionIDs: append(
			[]string(nil), experiment.ObservationRequirementDefinitionIDs...,
		),
		ObservationProgram: SubjectDefinition{
			DefinitionID:        callerClosureObservationProgramID,
			BehaviorFingerprint: callerClosureObservationProgramFingerprint,
		},
		ImplementationLinkID:                  implementationLink.DefinitionID,
		ImplementationLinkBehaviorFingerprint: implementationLink.BehaviorFingerprint,
		ImplementationLinkSourceTarget: SubjectDefinition{
			DefinitionID:        implementationLink.SourceTarget.DefinitionID,
			Kind:                implementationLink.SourceTarget.Kind,
			BehaviorFingerprint: implementationLink.SourceTarget.BehaviorFingerprint,
		},
		ImplementationLinkDestinationTarget: SubjectDefinition{
			DefinitionID:        implementationLink.DestinationTarget.DefinitionID,
			Kind:                implementationLink.DestinationTarget.Kind,
			BehaviorFingerprint: implementationLink.DestinationTarget.BehaviorFingerprint,
		},
		ImplementationLinkDiagnosticPresent: implementationLink.Diagnostic != nil,
	}, nil
}

// CheckSubject rejects generated-pin drift before an execution environment is opened.
func CheckSubject(
	experimentBytes []byte,
	expected SubjectBinding,
) error {
	actual, err := PinSubject(experimentBytes)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return errors.New("run evaluation subject binding drifted")
	}
	return nil
}

func subjectLimits(limits artifactv2.Limits) []SubjectLimit {
	return []SubjectLimit{
		{Path: "behavior.transitions", Value: limits.Behavior.Transitions.Value.String(), Unit: limits.Behavior.Transitions.Unit},
		{Path: "behavior.selectedActions", Value: limits.Behavior.SelectedActions.Value.String(), Unit: limits.Behavior.SelectedActions.Unit},
		{Path: "search", Value: limits.Search.Value.String(), Unit: limits.Search.Unit},
	}
}

func subjectKnownGaps(knownGaps []artifactv2.KnownGap) []SubjectKnownGap {
	result := make([]SubjectKnownGap, len(knownGaps))
	for index, knownGap := range knownGaps {
		result[index] = SubjectKnownGap{Kind: knownGap.Kind, Code: knownGap.Code}
		if knownGap.Subject != nil {
			result[index].SubjectPresent = true
			result[index].Subject = *knownGap.Subject
		}
		if knownGap.Detail != nil {
			result[index].DetailPresent = true
			result[index].Detail = *knownGap.Detail
		}
	}
	return result
}

func subjectBehaviorFingerprints(experiment artifactv2.Experiment) []string {
	result := []string{
		experiment.QueryBehaviorFingerprint,
		experiment.Plan.QueryBehaviorFingerprint,
		experiment.Plan.BehaviorFingerprint,
		experiment.Plan.TargetBehaviorFingerprint,
		experiment.Plan.KernelBehaviorFingerprint,
	}
	for _, property := range experiment.Properties {
		result = append(result, property.BehaviorFingerprint)
	}
	return result
}

func subjectDefinitionIDs(experiment artifactv2.Experiment) []string {
	plan := experiment.Plan
	result := []string{
		plan.QueryDefinitionID,
		plan.BehaviorDefinitionID,
		plan.TargetDefinitionID,
		plan.KernelDefinitionID,
	}
	for _, binding := range plan.Bindings {
		result = append(result, binding.RoleDefinitionID, binding.Value.DefinitionID)
	}
	for _, role := range plan.SymbolicRoles {
		result = append(result, role.DefinitionID)
	}
	for _, precondition := range plan.ModelPreconditions {
		result = append(result, precondition.DefinitionID)
		result = appendSubjectOperandDefinitionIDs(result, precondition.Left)
		result = appendSubjectOperandDefinitionIDs(result, precondition.Right)
	}
	result = append(result, plan.InitialState.DefinitionID)
	result = appendSubjectModelValueDefinitionIDs(result, plan.RequestedActions)
	result = appendSubjectModelValueDefinitionIDs(result, plan.ModelOutcomes)
	result = appendSubjectModelValueDefinitionIDs(result, plan.ResultingStates)
	for _, occurrence := range plan.LinearExtension {
		result = append(result, occurrence.DefinitionID, occurrence.ActionDefinitionID)
		if occurrence.AuthoredDefinitionID != nil {
			result = append(result, *occurrence.AuthoredDefinitionID)
		}
	}
	result = appendSubjectModelValueDefinitionIDs(result, plan.SelectedChoices)
	result = appendSubjectModelValueDefinitionIDs(result, plan.SelectedVariants)
	result = appendSubjectModelValueDefinitionIDs(result, plan.RequestedFaults)
	result = append(result, plan.CapabilityRequirementDefinitionIDs...)
	for _, checkpoint := range plan.Checkpoints {
		result = appendSubjectModelValueDefinitionIDs(result, checkpoint.Observations)
	}
	for _, property := range experiment.Properties {
		result = append(result, property.DefinitionID)
		result = append(result, property.RequirementDefinitionIDs...)
	}
	result = append(result, experiment.ObservationRequirementDefinitionIDs...)
	result = append(result, experiment.Provenance.SourceDefinitionIDs...)
	result = append(result, plan.Provenance.SourceDefinitionIDs...)
	return result
}

func appendSubjectOperandDefinitionIDs(result []string, operand artifactv2.Operand) []string {
	if operand.DefinitionID != "" {
		result = append(result, operand.DefinitionID)
	}
	if operand.Value != nil {
		result = append(result, operand.Value.DefinitionID)
	}
	return result
}

func appendSubjectModelValueDefinitionIDs(
	result []string,
	values []artifactv2.ModelValue,
) []string {
	for _, value := range values {
		result = append(result, value.DefinitionID)
	}
	return result
}
