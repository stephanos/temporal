package runevaluation

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/internal/runtimeengine"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

const (
	callerClosureExperimentChecksum              = "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8"
	callerClosureExperimentProvenanceChecksum    = "sha256:f7a6ebefca8202c6a7c467fd516e54d162c7d1f254c6c9a1f004a7f0b4135ab8"
	callerClosureConfigurationChecksum           = "sha256:21b4f7d0db2f68f939df901c2c5d146b1be3e45e55ad6cc171445fda5f29c1d5"
	callerClosureConfigurationFingerprint        = "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67"
	callerClosureConfigurationProvenance         = "sha256:3b8bae9ef57fa5f400076af50d01a283d8b928481d654abd0d3c39dd72ea2f6c"
	callerClosureQueryID                         = "workflow-nexus.query.exact-action-caller-closure"
	callerClosureQueryFingerprint                = "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af"
	callerClosureTargetID                        = "workflow-nexus.target.caller-closure"
	callerClosureTargetFingerprint               = "sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd"
	callerClosurePropertyID                      = "workflow-nexus.property.caller-closure"
	callerClosurePropertyFingerprint             = "sha256:b7a6e89d79e40dad31a7f96c281a05ca8af74996fbc2f8a6f302b379d609192f"
	callerClosureConfigurationID                 = "temporal.nexus.runtime-configuration.caller-closure"
	localAuthorityProfileID                      = "temporal.runtime-profile.ephemeral-local"
	localAuthorityProfileFingerprint             = "sha256:dd92f1ee14df101f2ea4abb4439f4722de8c061292a4fdd6b6476c7ca7e09b31"
	callerClosureConfigurationProfileID          = "temporal.nexus.synthetic.basic-lifecycle.profile"
	callerClosureConfigurationProfileFingerprint = "sha256:ac3cf245ad3e4a311eb6372be9caf49301c7e8ad3ee1b1875a53ea69d1ddc105"
	callerClosureObservationProgramID            = "temporal.nexus.observation-program.basic-lifecycle"
	callerClosureObservationProgramFingerprint   = "sha256:1ab36fdcd2978dec901678491646ec67fe0fc1d3bd1883e599bc2c53810b3480"
	callerClosureConfigurationMappingID          = "temporal.nexus.synthetic.basic-lifecycle.mapping"
	callerClosureConfigurationMappingFingerprint = "sha256:608e4db6c3a29d0f953640621ee34d34e16b0090309e85804e21f0cb21be30a2"
	callerClosureCheckedProfileID                = "temporal.system.nexus.caller-closure.profile"
	callerClosureCheckedMappingID                = "temporal.system.nexus.caller-closure.mapping"
	callerClosureCheckedMappingFingerprint       = "sha256:d5d437c89205880d27770b5abdac8aa3eabf07a21e40264ae5601162d70a7f17"
	callerClosureParticipantID                   = "temporal.nexus.participant.caller-closure"
	callerClosureParticipantProgramID            = "temporal.nexus.participant-program.caller-closure"
	callerClosureParticipantProgramFingerprint   = "sha256:f2f1a9a1346576b4d8c6b0b4f7f6c8a138461f90c168ab57747b316807666e56"
)

var callerClosureRequirements = []string{
	"nexus.capability.cancellation",
	"workflow-nexus.capability.ownership",
	"workflow.capability.lifecycle",
}

type checkerCall func(context.Context, checkerRequest) (checkerResponse, error)

// Check evaluates one exact admitted local caller-closure execution set in memory.
func Check(admittedSet artifact.AdmittedSet) (artifact.AdmittedSet, error) {
	ctx, stop := checkerSignalContext()
	defer stop()
	return checkWithChecker(ctx, admittedSet, runFixedChecker)
}

func checkerSignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func checkWithChecker(
	ctx context.Context,
	admittedSet artifact.AdmittedSet,
	checker checkerCall,
) (artifact.AdmittedSet, error) {
	execution, err := checkedCallerClosureExecution(admittedSet)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	request, err := newCheckerRequest(execution)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	response, err := checker(ctx, request)
	if err != nil {
		return artifact.AdmittedSet{}, newEvaluationFailure(
			"checker", "Observation Evaluation", "umpire.run-evaluation.checker.failed", err,
		)
	}
	if err := validateCheckerResponseForRequest(response, request); err != nil {
		return artifact.AdmittedSet{}, newEvaluationFailure(
			"output-invariant", "evaluation", "umpire.run-evaluation.response.invalid", err,
		)
	}
	evidence, result, err := constructEvaluation(execution, request, response)
	if err != nil {
		return artifact.AdmittedSet{}, err
	}
	output, err := execution.AdmitEvaluation(evidence, result)
	if err != nil {
		return artifact.AdmittedSet{}, newEvaluationFailure(
			"output-invariant", "construction", "umpire.run-evaluation.artifact-closure.invalid", err,
		)
	}
	return output, nil
}

func checkedCallerClosureExecution(admittedSet artifact.AdmittedSet) (artifact.ExecutionSet, error) {
	execution, ok := admittedSet.Execution()
	if !ok {
		return artifact.ExecutionSet{}, newEvaluationFailure(
			"input", "admission", "umpire.run-evaluation.input.exact-four-member-set", nil,
		)
	}
	experiment := execution.Experiment()
	configuration := execution.RuntimeConfiguration()
	run := execution.ExperimentRun()
	rawEvidence := execution.RawEvidence()
	if !exactCallerClosureExperiment(experiment) || !exactCallerClosureConfiguration(configuration) {
		return artifact.ExecutionSet{}, newEvaluationFailure(
			"input", "generated-view", "umpire.run-evaluation.input.unsupported-profile", nil,
		)
	}
	if !exactCallerClosureSources(run.SourceClosures, rawEvidence.Sources) {
		return artifact.ExecutionSet{}, newEvaluationFailure(
			"input", "generated-view", "umpire.run-evaluation.input.unsupported-source", nil,
		)
	}
	if runtimeengine.OperationalStatus(run) != run.OperationalStatus {
		return artifact.ExecutionSet{}, newEvaluationFailure(
			"input", "generated-view", "umpire.run-evaluation.input.operational-status", nil,
		)
	}
	return execution, nil
}

func exactCallerClosureExperiment(experiment artifactv2.Experiment) bool {
	binding, err := artifactv2.ExperimentArtifactBinding(experiment)
	if err != nil {
		return false
	}
	return binding.ArtifactChecksum == callerClosureExperimentChecksum &&
		binding.BehaviorFingerprint == callerClosureQueryFingerprint &&
		binding.ProvenanceChecksum == callerClosureExperimentProvenanceChecksum &&
		experiment.Plan.QueryDefinitionID == callerClosureQueryID &&
		experiment.Plan.QueryBehaviorFingerprint == callerClosureQueryFingerprint &&
		experiment.Plan.TargetDefinitionID == callerClosureTargetID &&
		experiment.Plan.TargetBehaviorFingerprint == callerClosureTargetFingerprint &&
		len(experiment.Properties) == 1 &&
		experiment.Properties[0].DefinitionID == callerClosurePropertyID &&
		experiment.Properties[0].BehaviorFingerprint == callerClosurePropertyFingerprint &&
		slices.Equal(experiment.Properties[0].RequirementDefinitionIDs, callerClosureRequirements)
}

func exactCallerClosureConfiguration(configuration artifactv2.RuntimeConfiguration) bool {
	binding := artifactv2.RuntimeConfigurationArtifactBinding(configuration)
	if binding.ArtifactChecksum != callerClosureConfigurationChecksum ||
		binding.BehaviorFingerprint != callerClosureConfigurationFingerprint ||
		binding.ProvenanceChecksum != callerClosureConfigurationProvenance ||
		configuration.ConfigurationDefinitionID != callerClosureConfigurationID ||
		configuration.AuthorityProfile.DefinitionID != localAuthorityProfileID ||
		configuration.AuthorityProfile.Version != artifactv2.NaturalFromUint64(2) ||
		configuration.AuthorityProfile.BehaviorFingerprint != localAuthorityProfileFingerprint ||
		len(configuration.AuthorityProfile.RequiredCapabilityDefinitionIDs) != 0 ||
		configuration.Observation.ProfileDefinitionID != callerClosureConfigurationProfileID ||
		configuration.Observation.ProfileBehaviorFingerprint != callerClosureConfigurationProfileFingerprint ||
		configuration.Observation.ProgramDefinitionID != callerClosureObservationProgramID ||
		configuration.Observation.ProgramBehaviorFingerprint != callerClosureObservationProgramFingerprint ||
		configuration.Observation.MappingDefinitionID != callerClosureConfigurationMappingID ||
		configuration.Observation.MappingBehaviorFingerprint != callerClosureConfigurationMappingFingerprint ||
		len(configuration.ParticipantBindings) != 1 {
		return false
	}
	participant := configuration.ParticipantBindings[0]
	return participant.ParticipantDefinitionID == callerClosureParticipantID &&
		participant.ProtocolDefinitionID == "umpire.participant-protocol.v2" &&
		participant.ProtocolVersion == artifactv2.NaturalFromUint64(2) &&
		participant.ProgramDefinitionID == callerClosureParticipantProgramID &&
		participant.ProgramBehaviorFingerprint == callerClosureParticipantProgramFingerprint &&
		slices.Equal(participant.CapabilityDefinitionIDs, callerClosureRequirements)
}

func exactCallerClosureSources(
	closures []artifactv2.SourceClosure,
	sources []artifactv2.RawEvidenceSource,
) bool {
	expected := []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}
	if len(closures) != len(expected) || len(sources) != len(expected) {
		return false
	}
	for index, definitionID := range expected {
		if closures[index].SourceDefinitionID != definitionID ||
			sources[index].SourceDefinitionID != definitionID {
			return false
		}
	}
	return true
}

func newCheckerRequest(execution artifact.ExecutionSet) (checkerRequest, error) {
	experiment := execution.Experiment()
	configuration := execution.RuntimeConfiguration()
	run := execution.ExperimentRun()
	rawEvidence := execution.RawEvidence()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	if err != nil {
		return checkerRequest{}, newEvaluationFailure(
			"input", "generated-view", "umpire.run-evaluation.request.binding", err,
		)
	}
	properties := make([]propertyReference, len(experiment.Properties))
	for index, property := range experiment.Properties {
		properties[index] = propertyReference{
			DefinitionID:             property.DefinitionID,
			BehaviorFingerprint:      property.BehaviorFingerprint,
			RequirementDefinitionIDs: slices.Clone(property.RequirementDefinitionIDs),
		}
	}
	request := checkerRequest{
		FormatVersion:              checkerRequestFormat,
		CheckerIdentity:            checkerIdentity,
		CheckerVersion:             artifactv2.NaturalFromUint64(2),
		CheckerBehaviorFingerprint: checkerBehaviorFingerprint,
		Experiment:                 experimentBinding,
		RuntimeConfiguration:       artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Run:                        artifactv2.ExperimentRunArtifactBinding(run),
		RawEvidence:                artifactv2.RawEvidenceArtifactBinding(rawEvidence),
		RunIdentity:                run.RunIdentity,
		Query: definitionReference{
			DefinitionID: experiment.Plan.QueryDefinitionID, BehaviorFingerprint: experiment.Plan.QueryBehaviorFingerprint,
		},
		Properties: properties,
		ObservationProgram: definitionReference{
			DefinitionID:        configuration.Observation.ProgramDefinitionID,
			BehaviorFingerprint: configuration.Observation.ProgramBehaviorFingerprint,
		},
		Mapping: definitionReference{
			DefinitionID:        callerClosureCheckedMappingID,
			BehaviorFingerprint: callerClosureCheckedMappingFingerprint,
		},
		PhaseOutcomes:        run.PhaseOutcomes,
		ControlAttempts:      run.ControlAttempts,
		SourceClosures:       run.SourceClosures,
		CaptureStatus:        rawEvidence.CaptureStatus,
		Sources:              rawEvidence.Sources,
		Facts:                rawEvidence.Facts,
		RunKnownGaps:         run.KnownGaps,
		RawEvidenceKnownGaps: rawEvidence.KnownGaps,
	}
	if err := validateCheckerRequest(request); err != nil {
		return checkerRequest{}, newEvaluationFailure(
			"input", "generated-view", "umpire.run-evaluation.request.invalid", err,
		)
	}
	return request, nil
}

func validateCheckerResponseForRequest(response checkerResponse, request checkerRequest) error {
	if err := validateCheckerResponse(response); err != nil {
		return err
	}
	if response.CheckerIdentity != request.CheckerIdentity ||
		response.CheckerVersion != request.CheckerVersion ||
		response.CheckerBehaviorFingerprint != request.CheckerBehaviorFingerprint {
		return errors.New("checker response handshake drifted")
	}
	if response.ExperimentArtifactChecksum != request.Experiment.ArtifactChecksum ||
		response.RuntimeConfigurationArtifactChecksum != request.RuntimeConfiguration.ArtifactChecksum ||
		response.RunArtifactChecksum != request.Run.ArtifactChecksum ||
		response.RawEvidenceArtifactChecksum != request.RawEvidence.ArtifactChecksum ||
		response.ExperimentBehaviorFingerprint != request.Experiment.BehaviorFingerprint ||
		response.RuntimeConfigurationBehaviorFingerprint != request.RuntimeConfiguration.BehaviorFingerprint ||
		response.RunIdentity != request.RunIdentity {
		return errors.New("checker response binding drifted")
	}
	return validateCheckerResponseProjection(response, request)
}
