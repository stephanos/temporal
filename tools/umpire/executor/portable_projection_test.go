package executor

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

const portableProjectionDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestProjectPortableExecutionProducesTheExistingRunnerInput(t *testing.T) {
	plan := portableProjectionPlan()

	input, err := projectPortableExecution(plan)

	require.NoError(t, err)
	executable, ok := input.Executable()
	require.True(t, ok)
	require.Equal(t, plan.GetExecution().GetQuery().GetBehaviorFingerprint(),
		executable.Experiment().QueryBehaviorFingerprint)
	require.Equal(t, plan.GetExecution().GetRuntime().GetConfig().GetDefinitionId(),
		executable.RuntimeConfiguration().ConfigurationDefinitionID)
	require.Equal(t, []string{"test.capability.runtime"},
		executable.RuntimeConfiguration().AuthorityProfile.RequiredCapabilityDefinitionIDs)
}

func TestPortableModelBindingsMustMatchTheProjectedRunnerInput(t *testing.T) {
	plan := portableProjectionPlan()
	input, err := projectPortableExecution(plan)
	require.NoError(t, err)
	executable, ok := input.Executable()
	require.True(t, ok)
	experiment, err := artifactv2.ExperimentArtifactBinding(executable.Experiment())
	require.NoError(t, err)
	runtime := artifactv2.RuntimeConfigurationArtifactBinding(executable.RuntimeConfiguration())
	plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: &umpirespb.ModelCompiledPlanProvenance{
		Experiment: portableProtoArtifactBinding(experiment), RuntimeConfig: portableProtoArtifactBinding(runtime),
	}}
	require.True(t, portableModelBindingsMatch(plan, input))

	plan.GetModelCompiled().Experiment.ArtifactChecksum = portableProjectionDigest
	require.False(t, portableModelBindingsMatch(plan, input))
}

func portableProjectionPlan() *umpirespb.PortableTestPlan {
	capability := portableProjectionBinding("test.capability.runtime")
	return &umpirespb.PortableTestPlan{
		Provenance: &umpirespb.PortableTestPlan_External{External: &umpirespb.ExternalPlanProvenance{
			Sources: []*umpirespb.SourceLocation{{Path: "test/portable.plan", Line: 1, Column: 1, Provenance: "test"}},
		}},
		Execution: &umpirespb.ExecutionProgram{
			Query:        portableProjectionBinding("test.query.portable"),
			Behavior:     portableProjectionBinding("test.behavior.portable"),
			Target:       portableProjectionBinding("test.target.portable"),
			Kernel:       portableProjectionBinding("test.kernel.portable"),
			RoleBindings: []*umpirespb.RoleBinding{}, SymbolicRoles: []*umpirespb.SymbolicRole{},
			Preconditions:    []*umpirespb.ExecutionPrecondition{},
			InitialState:     portableProjectionValue("test.state.portable", "open"),
			RequestedActions: []*umpirespb.PortableModelValue{portableProjectionValue("test.action.portable", "close")},
			ModelOutcomes:    []*umpirespb.PortableModelValue{portableProjectionValue("test.outcome.portable", "done")},
			ResultingStates:  []*umpirespb.PortableModelValue{portableProjectionValue("test.state.portable", "closed")},
			Occurrences: []*umpirespb.PlannedOccurrence{{
				Definition:         portableProjectionBinding("test.occurrence.portable"),
				ActionDefinitionId: "test.action.portable", Position: 1,
				AuthoredDefinitionId: "test.occurrence.portable",
			}},
			SelectedChoices: []*umpirespb.PortableModelValue{}, SelectedVariants: []*umpirespb.PortableModelValue{},
			RequestedFaults: []*umpirespb.PortableModelValue{}, CapabilityRequirements: []*umpirespb.DefinitionBinding{capability},
			Checkpoints: []*umpirespb.ExecutionCheckpoint{{Transition: 1, Observations: []*umpirespb.PortableModelValue{}}},
			Runtime: &umpirespb.RuntimeProgram{
				AuthorityProfile: portableProjectionBinding("test.authority.portable"),
				Config:           portableProjectionBinding("test.runtime.portable"),
				ParticipantBindings: []*umpirespb.PortableParticipantBinding{{
					Participant: portableProjectionBinding("test.participant.portable"),
					Protocol:    portableProjectionBinding("test.protocol.portable"), ProtocolVersion: 2,
					Program: portableProjectionBinding("test.program.portable"), Capabilities: []*umpirespb.DefinitionBinding{},
				}},
				ObservationConfig: &umpirespb.PortableObservationConfig{
					Profile: portableProjectionBinding("test.observation-profile.portable"),
					Program: portableProjectionBinding("test.observation-program.portable"),
					Mapping: portableProjectionBinding("test.observation-mapping.portable"),
				},
				PhaseLimits:                   portableProjectionPhaseLimits(),
				AuthorityRequiredCapabilities: []*umpirespb.DefinitionBinding{capability},
			},
		},
		Verification: &umpirespb.VerificationProgram{
			Observation: &umpirespb.ObservationProgram{Emits: []*umpirespb.Emit{}},
			Properties: []*umpirespb.Property{{
				Definition:   portableProjectionBinding("test.property.portable"),
				Requirements: []*umpirespb.DefinitionBinding{},
			}},
		},
		Limits: &umpirespb.PortableTestPlanLimits{
			Evaluation: &umpirespb.PortableEvaluationLimits{MaxWork: 10_000},
		},
		KnownGaps: []*umpirespb.KnownGap{},
	}
}

func portableProjectionBinding(id string) *umpirespb.DefinitionBinding {
	return &umpirespb.DefinitionBinding{DefinitionId: id, BehaviorFingerprint: portableProjectionDigest}
}

func portableProjectionValue(id, value string) *umpirespb.PortableModelValue {
	return &umpirespb.PortableModelValue{
		Definition: portableProjectionBinding(id),
		Value:      &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value}},
	}
}

func portableProjectionPhaseLimits() []*umpirespb.ExecutionPhaseLimit {
	return []*umpirespb.ExecutionPhaseLimit{
		{Phase: umpirespb.EXECUTION_PHASE_PREPARATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_REALIZATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_OBSERVATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 3_584, MaxBytes: 12 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_ISOLATION, DurationMilliseconds: 15_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_CLEANUP, DurationMilliseconds: 15_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
	}
}

func portableProtoArtifactBinding(binding artifactv2.ArtifactBinding) *umpirespb.ArtifactBinding {
	return &umpirespb.ArtifactBinding{
		FormatVersion: binding.FormatVersion, ArtifactChecksum: binding.ArtifactChecksum,
		BehaviorFingerprint: binding.BehaviorFingerprint, ProvenanceChecksum: binding.ProvenanceChecksum,
	}
}
