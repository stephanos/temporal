package artifactv2

import "slices"

// CopyExperiment returns an Experiment with independent mutable storage.
func CopyExperiment(document Experiment) Experiment {
	copied := document
	copied.Properties = slices.Clone(document.Properties)
	for index := range copied.Properties {
		copied.Properties[index].RequirementDefinitionIDs = slices.Clone(
			document.Properties[index].RequirementDefinitionIDs,
		)
	}
	copied.ObservationRequirementDefinitionIDs = slices.Clone(
		document.ObservationRequirementDefinitionIDs,
	)
	copied.Provenance = copyProvenance(document.Provenance)
	copied.Plan = copyDrivePlan(document.Plan)
	return copied
}

// CopyRuntimeConfiguration returns a RuntimeConfiguration with independent mutable storage.
func CopyRuntimeConfiguration(document RuntimeConfiguration) RuntimeConfiguration {
	copied := document
	copied.AuthorityProfile.RequiredCapabilityDefinitionIDs = slices.Clone(
		document.AuthorityProfile.RequiredCapabilityDefinitionIDs,
	)
	copied.PhaseLimits = slices.Clone(document.PhaseLimits)
	copied.ParticipantBindings = slices.Clone(document.ParticipantBindings)
	for index := range copied.ParticipantBindings {
		copied.ParticipantBindings[index].CapabilityDefinitionIDs = slices.Clone(
			document.ParticipantBindings[index].CapabilityDefinitionIDs,
		)
	}
	copied.KnownGaps = copyKnownGaps(document.KnownGaps)
	copied.Provenance = copyProvenance(document.Provenance)
	return copied
}

// CopyExperimentRun returns an ExperimentRun with independent mutable storage.
func CopyExperimentRun(document ExperimentRun) ExperimentRun {
	copied := document
	copied.PhaseOutcomes = slices.Clone(document.PhaseOutcomes)
	for index := range copied.PhaseOutcomes {
		copied.PhaseOutcomes[index].StartedAtUnixMillis = copyNaturalPointer(
			document.PhaseOutcomes[index].StartedAtUnixMillis,
		)
		copied.PhaseOutcomes[index].FinishedAtUnixMillis = copyNaturalPointer(
			document.PhaseOutcomes[index].FinishedAtUnixMillis,
		)
		copied.PhaseOutcomes[index].Code = copyStringPointer(document.PhaseOutcomes[index].Code)
	}
	copied.ControlAttempts = slices.Clone(document.ControlAttempts)
	for index := range copied.ControlAttempts {
		copied.ControlAttempts[index].ReceiptFactDefinitionID = copyStringPointer(
			document.ControlAttempts[index].ReceiptFactDefinitionID,
		)
		copied.ControlAttempts[index].Code = copyStringPointer(document.ControlAttempts[index].Code)
	}
	copied.SourceClosures = slices.Clone(document.SourceClosures)
	copied.Cleanup.Code = copyStringPointer(document.Cleanup.Code)
	copied.Limits = slices.Clone(document.Limits)
	copied.KnownGaps = copyKnownGaps(document.KnownGaps)
	copied.Provenance = copyProvenance(document.Provenance)
	return copied
}

// CopyRawEvidence returns RawEvidence with independent mutable storage.
func CopyRawEvidence(document RawEvidence) RawEvidence {
	copied := document
	copied.Sources = slices.Clone(document.Sources)
	copied.Facts = slices.Clone(document.Facts)
	for index := range copied.Facts {
		copied.Facts[index].CausalFactDefinitionIDs = slices.Clone(
			document.Facts[index].CausalFactDefinitionIDs,
		)
		copied.Facts[index].Fields = slices.Clone(document.Facts[index].Fields)
	}
	copied.KnownGaps = copyKnownGaps(document.KnownGaps)
	copied.Provenance = copyProvenance(document.Provenance)
	return copied
}

func copyDrivePlan(plan DrivePlan) DrivePlan {
	copied := plan
	copied.Bindings = slices.Clone(plan.Bindings)
	copied.SymbolicRoles = slices.Clone(plan.SymbolicRoles)
	copied.ModelPreconditions = slices.Clone(plan.ModelPreconditions)
	for index := range copied.ModelPreconditions {
		copied.ModelPreconditions[index].Left.Value = copyModelValuePointer(
			plan.ModelPreconditions[index].Left.Value,
		)
		copied.ModelPreconditions[index].Right.Value = copyModelValuePointer(
			plan.ModelPreconditions[index].Right.Value,
		)
	}
	copied.RequestedActions = slices.Clone(plan.RequestedActions)
	copied.ModelOutcomes = slices.Clone(plan.ModelOutcomes)
	copied.ResultingStates = slices.Clone(plan.ResultingStates)
	copied.LinearExtension = slices.Clone(plan.LinearExtension)
	for index := range copied.LinearExtension {
		copied.LinearExtension[index].AuthoredDefinitionID = copyStringPointer(
			plan.LinearExtension[index].AuthoredDefinitionID,
		)
	}
	copied.SelectedChoices = slices.Clone(plan.SelectedChoices)
	copied.SelectedVariants = slices.Clone(plan.SelectedVariants)
	copied.RequestedFaults = slices.Clone(plan.RequestedFaults)
	copied.CapabilityRequirementDefinitionIDs = slices.Clone(
		plan.CapabilityRequirementDefinitionIDs,
	)
	copied.Checkpoints = slices.Clone(plan.Checkpoints)
	for index := range copied.Checkpoints {
		copied.Checkpoints[index].Observations = slices.Clone(plan.Checkpoints[index].Observations)
	}
	copied.KnownGaps = copyKnownGaps(plan.KnownGaps)
	copied.Provenance = copyProvenance(plan.Provenance)
	return copied
}

func copyProvenance(provenance Provenance) Provenance {
	return Provenance{
		SourceDefinitionIDs: slices.Clone(provenance.SourceDefinitionIDs),
		SourceLocations:     slices.Clone(provenance.SourceLocations),
	}
}

func copyKnownGaps(gaps []KnownGap) []KnownGap {
	copied := slices.Clone(gaps)
	for index := range copied {
		copied[index].Subject = copyStringPointer(gaps[index].Subject)
		copied[index].Detail = copyStringPointer(gaps[index].Detail)
	}
	return copied
}

func copyModelValuePointer(value *ModelValue) *ModelValue {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyNaturalPointer(value *Natural) *Natural {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
