package executor

import (
	"errors"
	"slices"
	"strconv"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	"google.golang.org/protobuf/proto"
)

func projectPortableExecution(plan *umpirespb.PortableTestPlan) (artifact.AdmittedSet, error) {
	execution := plan.GetExecution()
	provenance := portableExperimentProvenance(plan)
	experiment, err := artifactv2.SealExperiment(artifactv2.Experiment{
		FormatVersion:            artifactv2.ExperimentFormat,
		QueryBehaviorFingerprint: execution.GetQuery().GetBehaviorFingerprint(),
		Plan: artifactv2.DrivePlan{
			FormatVersion:                      artifactv2.DrivePlanFormat,
			QueryDefinitionID:                  execution.GetQuery().GetDefinitionId(),
			QueryBehaviorFingerprint:           execution.GetQuery().GetBehaviorFingerprint(),
			BehaviorDefinitionID:               execution.GetBehavior().GetDefinitionId(),
			BehaviorFingerprint:                execution.GetBehavior().GetBehaviorFingerprint(),
			TargetDefinitionID:                 execution.GetTarget().GetDefinitionId(),
			TargetBehaviorFingerprint:          execution.GetTarget().GetBehaviorFingerprint(),
			KernelDefinitionID:                 execution.GetKernel().GetDefinitionId(),
			KernelBehaviorFingerprint:          execution.GetKernel().GetBehaviorFingerprint(),
			Bindings:                           portableBindings(execution.GetRoleBindings()),
			SymbolicRoles:                      portableRoles(execution.GetSymbolicRoles()),
			ModelPreconditions:                 portablePreconditions(execution.GetPreconditions()),
			InitialState:                       portableArtifactValue(execution.GetInitialState()),
			RequestedActions:                   portableArtifactValues(execution.GetRequestedActions()),
			ModelOutcomes:                      portableArtifactValues(execution.GetModelOutcomes()),
			ResultingStates:                    portableArtifactValues(execution.GetResultingStates()),
			LinearExtension:                    portableOccurrences(execution.GetOccurrences()),
			SelectedChoices:                    portableArtifactValues(execution.GetSelectedChoices()),
			SelectedVariants:                   portableArtifactValues(execution.GetSelectedVariants()),
			RequestedFaults:                    portableArtifactValues(execution.GetRequestedFaults()),
			CapabilityRequirementDefinitionIDs: portableDefinitionIDs(execution.GetCapabilityRequirements()),
			ExpandedLimits: artifactv2.Limits{
				Behavior: artifactv2.BehaviorLimits{
					Transitions:     artifactv2.Limit{Value: artifactv2.NaturalFromUint64(1), Unit: "semantic-transitions"},
					SelectedActions: artifactv2.Limit{Value: artifactv2.NaturalFromUint64(1), Unit: "selected-actions"},
				},
				Search: artifactv2.Limit{
					Value: artifactv2.NaturalFromUint64(uint64(plan.GetLimits().GetEvaluation().GetMaxWork())),
					Unit:  "candidate-evaluations",
				},
			},
			Checkpoints:     portableCheckpoints(execution.GetCheckpoints()),
			SelectionReason: "behavior-selection",
			Explored: artifactv2.ExploredCounts{
				Setups: artifactv2.NaturalFromUint64(1), Traces: artifactv2.NaturalFromUint64(1),
				Transitions: artifactv2.NaturalFromUint64(1), PropertyEvaluations: artifactv2.NaturalFromUint64(1),
			},
			KnownGaps:  portableKnownGaps(plan.GetKnownGaps()),
			Provenance: provenance,
		},
		Properties:                          portableProperties(plan.GetVerification().GetProperties()),
		ObservationRequirementDefinitionIDs: portableObservationRequirements(plan.GetVerification()),
		Provenance:                          provenance,
	})
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	runtime := execution.GetRuntime()
	runtimeProvenance := portableRuntimeProvenance(plan)
	configuration, err := artifactv2.SealRuntimeConfiguration(artifactv2.RuntimeConfiguration{
		FormatVersion:             artifactv2.RuntimeConfigurationFormat,
		ConfigurationDefinitionID: runtime.GetConfig().GetDefinitionId(),
		BehaviorFingerprint:       runtime.GetConfig().GetBehaviorFingerprint(),
		Experiment:                experimentBinding,
		AuthorityProfile: artifactv2.AuthorityProfile{
			DefinitionID:                    runtime.GetAuthorityProfile().GetDefinitionId(),
			Version:                         artifactv2.NaturalFromUint64(2),
			BehaviorFingerprint:             runtime.GetAuthorityProfile().GetBehaviorFingerprint(),
			RequiredCapabilityDefinitionIDs: portableDefinitionIDs(runtime.GetAuthorityRequiredCapabilities()),
		},
		PhaseLimits: portablePhaseLimits(runtime.GetPhaseLimits()),
		Observation: artifactv2.ObservationConfiguration{
			ProfileDefinitionID:        runtime.GetObservationConfig().GetProfile().GetDefinitionId(),
			ProfileBehaviorFingerprint: runtime.GetObservationConfig().GetProfile().GetBehaviorFingerprint(),
			ProgramDefinitionID:        runtime.GetObservationConfig().GetProgram().GetDefinitionId(),
			ProgramBehaviorFingerprint: runtime.GetObservationConfig().GetProgram().GetBehaviorFingerprint(),
			MappingDefinitionID:        runtime.GetObservationConfig().GetMapping().GetDefinitionId(),
			MappingBehaviorFingerprint: runtime.GetObservationConfig().GetMapping().GetBehaviorFingerprint(),
		},
		ParticipantBindings: portableParticipants(runtime.GetParticipantBindings()),
		KnownGaps:           []artifactv2.KnownGap{},
		Provenance:          runtimeProvenance,
	})
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	experimentBytes, err := artifact.EncodeExperimentV2(experiment)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	configurationBytes, err := artifact.EncodeRuntimeConfigurationV2(configuration)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	return artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: configurationBytes},
	})
}

func portableModelValue(value *umpirespb.Value) string {
	switch typed := value.GetValue().(type) {
	case *umpirespb.Value_Text:
		return typed.Text
	case *umpirespb.Value_Natural:
		return typed.Natural
	case *umpirespb.Value_BoolValue:
		return strconv.FormatBool(typed.BoolValue)
	default:
		return ""
	}
}

func portableArtifactValue(value *umpirespb.PortableModelValue) artifactv2.ModelValue {
	return artifactv2.ModelValue{DefinitionID: value.GetDefinition().GetDefinitionId(), Value: portableModelValue(value.GetValue())}
}

func portableArtifactValues(values []*umpirespb.PortableModelValue) []artifactv2.ModelValue {
	result := make([]artifactv2.ModelValue, len(values))
	for index, value := range values {
		result[index] = portableArtifactValue(value)
	}
	return result
}

func portableBindings(values []*umpirespb.RoleBinding) []artifactv2.Binding {
	result := make([]artifactv2.Binding, len(values))
	for index, value := range values {
		result[index] = artifactv2.Binding{
			RoleDefinitionID: value.GetRole().GetDefinitionId(), Value: portableArtifactValue(value.GetValue()),
		}
	}
	return result
}

func portableRoles(values []*umpirespb.SymbolicRole) []artifactv2.Role {
	result := make([]artifactv2.Role, len(values))
	for index, value := range values {
		result[index] = artifactv2.Role{
			DefinitionID: value.GetDefinition().GetDefinitionId(), ValueKind: portableDefinitionKind(value.GetKind()),
		}
	}
	return result
}

func portableDefinitionKind(kind umpirespb.PortableDefinitionKind) string {
	switch kind {
	case umpirespb.PORTABLE_DEFINITION_KIND_STATE:
		return "state"
	case umpirespb.PORTABLE_DEFINITION_KIND_ACTION:
		return "action"
	case umpirespb.PORTABLE_DEFINITION_KIND_OUTCOME:
		return "outcome"
	case umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION:
		return "observation"
	case umpirespb.PORTABLE_DEFINITION_KIND_RELATION:
		return "relation"
	case umpirespb.PORTABLE_DEFINITION_KIND_CAPABILITY:
		return "capability"
	case umpirespb.PORTABLE_DEFINITION_KIND_PROVIDER:
		return "provider"
	case umpirespb.PORTABLE_DEFINITION_KIND_LAW:
		return "law"
	case umpirespb.PORTABLE_DEFINITION_KIND_CONNECTOR:
		return "connector"
	case umpirespb.PORTABLE_DEFINITION_KIND_TARGET:
		return "target"
	case umpirespb.PORTABLE_DEFINITION_KIND_KERNEL:
		return "kernel"
	default:
		return ""
	}
}

func portablePreconditions(values []*umpirespb.ExecutionPrecondition) []artifactv2.Precondition {
	result := make([]artifactv2.Precondition, len(values))
	for index, value := range values {
		relation := "equal"
		if value.GetOperator() == umpirespb.PRECONDITION_OPERATOR_NOT_EQUALS {
			relation = "different"
		}
		result[index] = artifactv2.Precondition{
			DefinitionID: value.GetDefinition().GetDefinitionId(), Relation: relation,
			Left: portableOperand(value.GetLeft()), Right: portableOperand(value.GetRight()),
		}
	}
	return result
}

func portableOperand(value *umpirespb.ExecutionOperand) artifactv2.Operand {
	switch typed := value.GetOperand().(type) {
	case *umpirespb.ExecutionOperand_Literal:
		literal := portableArtifactValue(typed.Literal)
		return artifactv2.Operand{Kind: "value", Value: &literal}
	case *umpirespb.ExecutionOperand_Role:
		return artifactv2.Operand{Kind: "role", DefinitionID: typed.Role.GetDefinitionId()}
	case *umpirespb.ExecutionOperand_RuntimeBindingSlot:
		return artifactv2.Operand{Kind: "role", DefinitionID: typed.RuntimeBindingSlot.GetDefinitionId()}
	default:
		return artifactv2.Operand{}
	}
}

func portableOccurrences(values []*umpirespb.PlannedOccurrence) []artifactv2.Occurrence {
	result := make([]artifactv2.Occurrence, len(values))
	for index, value := range values {
		authored := value.GetAuthoredDefinitionId()
		result[index] = artifactv2.Occurrence{
			DefinitionID: value.GetDefinition().GetDefinitionId(), ActionDefinitionID: value.GetActionDefinitionId(),
			Position: artifactv2.NaturalFromUint64(uint64(value.GetPosition())), AuthoredDefinitionID: &authored,
		}
	}
	return result
}

func portableCheckpoints(values []*umpirespb.ExecutionCheckpoint) []artifactv2.Checkpoint {
	result := make([]artifactv2.Checkpoint, len(values))
	for index, value := range values {
		result[index] = artifactv2.Checkpoint{
			Transition:   artifactv2.NaturalFromUint64(uint64(value.GetTransition())),
			Observations: portableArtifactValues(value.GetObservations()),
		}
	}
	return result
}

func portableProperties(values []*umpirespb.Property) []artifactv2.Property {
	result := make([]artifactv2.Property, len(values))
	for index, value := range values {
		result[index] = artifactv2.Property{
			DefinitionID:             value.GetDefinition().GetDefinitionId(),
			BehaviorFingerprint:      value.GetDefinition().GetBehaviorFingerprint(),
			RequirementDefinitionIDs: portableDefinitionIDs(value.GetRequirements()),
		}
	}
	return result
}

func portableObservationRequirements(verification *umpirespb.VerificationProgram) []string {
	values := make([]string, 0, len(verification.GetObservation().GetEmits()))
	for _, emit := range verification.GetObservation().GetEmits() {
		values = append(values, emit.GetOutputDefinition().GetDefinitionId())
	}
	slices.Sort(values)
	return slices.Compact(values)
}

func portableDefinitionIDs(values []*umpirespb.DefinitionBinding) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = value.GetDefinitionId()
	}
	return result
}

func portablePhaseLimits(values []*umpirespb.ExecutionPhaseLimit) []artifactv2.PhaseLimit {
	result := make([]artifactv2.PhaseLimit, len(values))
	for index, value := range values {
		result[index] = artifactv2.PhaseLimit{
			Phase:                portablePhase(value.GetPhase()),
			DurationMilliseconds: artifactv2.NaturalFromUint64(uint64(value.GetDurationMilliseconds())),
			MaxAttempts:          artifactv2.NaturalFromUint64(uint64(value.GetMaxAttempts())),
			MaxRecords:           artifactv2.NaturalFromUint64(uint64(value.GetMaxRecords())),
			MaxBytes:             artifactv2.NaturalFromUint64(uint64(value.GetMaxBytes())),
		}
	}
	return result
}

func portablePhase(phase umpirespb.ExecutionPhase) string {
	switch phase {
	case umpirespb.EXECUTION_PHASE_PREPARATION:
		return "preparation"
	case umpirespb.EXECUTION_PHASE_REALIZATION:
		return "realization"
	case umpirespb.EXECUTION_PHASE_OBSERVATION:
		return "observation"
	case umpirespb.EXECUTION_PHASE_ISOLATION:
		return "isolation"
	case umpirespb.EXECUTION_PHASE_CLEANUP:
		return "cleanup"
	default:
		return ""
	}
}

func portableParticipants(values []*umpirespb.PortableParticipantBinding) []artifactv2.ParticipantBinding {
	result := make([]artifactv2.ParticipantBinding, len(values))
	for index, value := range values {
		result[index] = artifactv2.ParticipantBinding{
			ParticipantDefinitionID:    value.GetParticipant().GetDefinitionId(),
			ProtocolDefinitionID:       value.GetProtocol().GetDefinitionId(),
			ProtocolVersion:            artifactv2.NaturalFromUint64(uint64(value.GetProtocolVersion())),
			ProgramDefinitionID:        value.GetProgram().GetDefinitionId(),
			ProgramBehaviorFingerprint: value.GetProgram().GetBehaviorFingerprint(),
			CapabilityDefinitionIDs:    portableDefinitionIDs(value.GetCapabilities()),
		}
	}
	return result
}

func portableKnownGaps(values []*umpirespb.KnownGap) []artifactv2.KnownGap {
	result := make([]artifactv2.KnownGap, len(values))
	for index, value := range values {
		var subject, detail *string
		if value.GetSubject() != "" {
			subjectValue := value.GetSubject()
			subject = &subjectValue
		}
		if value.GetDetail() != "" {
			detailValue := value.GetDetail()
			detail = &detailValue
		}
		result[index] = artifactv2.KnownGap{
			Kind: portableKnownGapKind(value.GetKind()), Code: value.GetCode(), Subject: subject, Detail: detail,
		}
	}
	return result
}

func portableKnownGapKind(kind umpirespb.KnownGapKind) string {
	switch kind {
	case umpirespb.KNOWN_GAP_KIND_CAPABILITY_CONTRACT:
		return "capability-contract"
	case umpirespb.KNOWN_GAP_KIND_INPUT:
		return "input"
	case umpirespb.KNOWN_GAP_KIND_INTERPRETATION:
		return "interpretation"
	case umpirespb.KNOWN_GAP_KIND_CLAIM:
		return "claim"
	default:
		return ""
	}
}

func portableExperimentProvenance(plan *umpirespb.PortableTestPlan) artifactv2.Provenance {
	execution := plan.GetExecution()
	ids := []string{
		execution.GetBehavior().GetDefinitionId(), execution.GetKernel().GetDefinitionId(),
		execution.GetQuery().GetDefinitionId(), execution.GetTarget().GetDefinitionId(),
	}
	for _, property := range plan.GetVerification().GetProperties() {
		ids = append(ids, property.GetDefinition().GetDefinitionId())
	}
	slices.Sort(ids)
	return artifactv2.Provenance{SourceDefinitionIDs: slices.Compact(ids), SourceLocations: portableSourceLocations(plan)}
}

func portableRuntimeProvenance(plan *umpirespb.PortableTestPlan) artifactv2.Provenance {
	runtime := plan.GetExecution().GetRuntime()
	ids := []string{
		runtime.GetAuthorityProfile().GetDefinitionId(), runtime.GetConfig().GetDefinitionId(),
		runtime.GetObservationConfig().GetProfile().GetDefinitionId(),
		runtime.GetObservationConfig().GetProgram().GetDefinitionId(),
		runtime.GetObservationConfig().GetMapping().GetDefinitionId(),
	}
	for _, participant := range runtime.GetParticipantBindings() {
		ids = append(ids, participant.GetProgram().GetDefinitionId())
	}
	slices.Sort(ids)
	return artifactv2.Provenance{SourceDefinitionIDs: slices.Compact(ids), SourceLocations: portableSourceLocations(plan)}
}

func portableSourceLocations(plan *umpirespb.PortableTestPlan) []artifactv2.SourceLocation {
	var values []*umpirespb.SourceLocation
	if plan.GetExternal() != nil {
		values = plan.GetExternal().GetSources()
	} else if plan.GetModelCompiled() != nil {
		values = plan.GetModelCompiled().GetSources()
	}
	result := make([]artifactv2.SourceLocation, len(values))
	for index, value := range values {
		result[index] = artifactv2.SourceLocation{
			Path: value.GetPath(), Line: artifactv2.NaturalFromUint64(uint64(value.GetLine())),
			Column: artifactv2.NaturalFromUint64(uint64(value.GetColumn())), Provenance: value.GetProvenance(),
		}
	}
	return result
}

func portableModelBindingsMatch(plan *umpirespb.PortableTestPlan, input artifact.AdmittedSet) bool {
	model := plan.GetModelCompiled()
	if model == nil {
		return true
	}
	executable, ok := input.Executable()
	if !ok {
		return false
	}
	experiment, err := artifactv2.ExperimentArtifactBinding(executable.Experiment())
	if err != nil {
		return false
	}
	runtime := artifactv2.RuntimeConfigurationArtifactBinding(executable.RuntimeConfiguration())
	return protoBindingMatches(model.GetExperiment(), experiment) && protoBindingMatches(model.GetRuntimeConfig(), runtime)
}

func portableInputBindings(
	input artifact.AdmittedSet,
	runtimeBindingSlots []*umpirespb.RuntimeBindingSlot,
	requiredCapabilities []string,
) (runnerBindingResult, error) {
	executable, ok := input.Executable()
	if !ok {
		return runnerBindingResult{}, errors.New("portable projection did not produce an executable set")
	}
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	if err != nil {
		return runnerBindingResult{}, err
	}
	return runnerBindingResult{
		binding: runner.InputBinding{
			ArtifactSetIdentity: input.Identity(), ArtifactSetChecksum: input.Checksum(), ManifestSHA256: input.ManifestSHA256(),
			ExperimentArtifactChecksum:               experiment.ArtifactChecksum,
			ExperimentBehaviorFingerprint:            experiment.QueryBehaviorFingerprint,
			RuntimeConfigurationArtifactChecksum:     configuration.ArtifactChecksum,
			RuntimeConfigurationBehaviorFingerprint:  configuration.BehaviorFingerprint,
			AuthorityRequiredCapabilityDefinitionIDs: slices.Clone(requiredCapabilities),
			RuntimeBindingSlots:                      cloneRuntimeBindingSlots(runtimeBindingSlots),
		},
		experiment: experimentBinding, runtime: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
	}, nil
}

func cloneRuntimeBindingSlots(slots []*umpirespb.RuntimeBindingSlot) []*umpirespb.RuntimeBindingSlot {
	cloned := make([]*umpirespb.RuntimeBindingSlot, len(slots))
	for index, slot := range slots {
		cloned[index] = proto.CloneOf(slot)
	}
	return cloned
}

type runnerBindingResult struct {
	binding    runner.InputBinding
	experiment artifactv2.ArtifactBinding
	runtime    artifactv2.ArtifactBinding
}
