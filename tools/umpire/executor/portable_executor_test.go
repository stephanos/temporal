package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/portableevaluation"
	"go.temporal.io/server/tools/umpire/runner"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/protobuf/proto"
)

func TestPrepareLeanGeneratedModelPlansRetainExactArtifactBindings(t *testing.T) {
	for _, name := range []string{"normal", "duplicate-delivery", "required-obligation"} {
		t.Run(name, func(t *testing.T) {
			encoded, err := os.ReadFile(filepath.Join(
				"..", "portableevaluation", "testdata", "portable-test-plan-v1", name, "plan.pb",
			))
			require.NoError(t, err)
			plan := new(umpirespb.PortableTestPlan)
			require.NoError(t, proto.Unmarshal(encoded, plan))
			if name != "required-obligation" {
				request := fixtureRequest(t, name)
				wantInput, err := artifact.AdmitSet([]artifact.SetMember{
					{Path: "artifacts/experiment.json", Encoded: request.GetInput().GetExperiment()},
					{Path: "artifacts/runtime-configuration.json", Encoded: request.GetInput().GetRuntimeConfig()},
				})
				require.NoError(t, err)
				gotInput, err := projectPortableExecution(plan)
				require.NoError(t, err)
				wantExecutable, ok := wantInput.Executable()
				require.True(t, ok)
				gotExecutable, ok := gotInput.Executable()
				require.True(t, ok)
				require.Equal(t, wantExecutable.Experiment(), gotExecutable.Experiment())
				require.Equal(t, wantExecutable.RuntimeConfiguration(), gotExecutable.RuntimeConfiguration())
			}

			prepared, err := preparePortableExecution(
				context.Background(), plan,
				func(
					_ context.Context,
					requested testplan.ModelProvenanceBinding,
				) (testplan.ModelProvenanceBinding, error) {
					return requested, nil
				},
			)

			require.NoError(t, err)
			protorequire.ProtoEqual(t, plan, prepared.plan.Plan())
		})
	}
}

func TestPortableExecutorRunsExternalAndModelPlansThroughOnePipeline(t *testing.T) {
	for _, model := range []bool{false, true} {
		name := "external"
		var verifier testplan.ModelProvenanceVerifier
		if model {
			name = "model compiled"
			verifier = func(
				_ context.Context,
				requested testplan.ModelProvenanceBinding,
			) (testplan.ModelProvenanceBinding, error) {
				return requested, nil
			}
		}
		t.Run(name, func(t *testing.T) {
			plan := portableExecutorFixturePlan(t, model)
			var runCalls atomic.Int32
			executor := newPortableExecutor(
				nil, verifier, preparePortableExecution,
				func(
					_ context.Context,
					input artifact.AdmittedSet,
					binding runner.InputBinding,
					runIdentity string,
					_ runner.Adapter,
				) (runOutcome, error) {
					runCalls.Add(1)
					_, executable := input.Executable()
					require.True(t, executable)
					require.Equal(t, input.Identity(), binding.ArtifactSetIdentity)
					return runOutcome{
						rawEvidence:       artifactv2.RawEvidence{RunIdentity: runIdentity},
						operationalStatus: umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
						cleanupStatus:     umpirespb.CLEANUP_STATUS_COMPLETE, reusable: true,
					}, nil
				},
				func(_ context.Context, request portableevaluation.PortableRequest) *umpirespb.ExecutionResult {
					result, err := request.Plan.ScopeResult(&umpirespb.ExecutionResult{
						RunIdentity:       request.ExpectedRunIdentity,
						ToolingStatus:     umpirespb.EXECUTION_TOOLING_STATUS_SUCCEEDED,
						OperationalStatus: umpirespb.EXECUTION_OPERATIONAL_STATUS_SUCCEEDED,
						Observation:       &umpirespb.ObservationEvaluationResult{Status: umpirespb.OBSERVATION_STATUS_ACCEPTED},
						TraceProjection:   &umpirespb.TraceProjectionResult{Status: umpirespb.TRACE_PROJECTION_STATUS_APPLIED},
						SemanticStatus:    umpirespb.EXECUTION_EVALUATION_STATUS_SATISFIED,
						CleanupStatus:     umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE,
						Decision:          umpirespb.EXECUTION_DECISION_PASS,
						Work:              &umpirespb.EvaluationWork{},
					})
					require.NoError(t, err)
					return result
				},
				func() string { return "test.run.portable-executor" },
			)

			result, err := executor.Execute(context.Background(), plan)

			require.NoError(t, err)
			require.Equal(t, int32(1), runCalls.Load())
			require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
			if model {
				require.Equal(t, umpirespb.CLAIM_SCOPE_MODEL_BOUND, result.GetClaimScope())
			} else {
				require.Equal(t, umpirespb.CLAIM_SCOPE_PLAN_LOCAL, result.GetClaimScope())
			}
		})
	}
}

func TestPortableExecutorRejectsTenCallBurstWithoutQueueing(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	executor := newPortableExecutor(
		nil, nil,
		func(context.Context, *umpirespb.PortableTestPlan, testplan.ModelProvenanceVerifier) (portablePreparation, error) {
			return portablePreparation{executionTimeout: time.Minute}, nil
		},
		func(
			_ context.Context, _ artifact.AdmittedSet, _ runner.InputBinding, runIdentity string, _ runner.Adapter,
		) (runOutcome, error) {
			calls.Add(1)
			close(entered)
			<-release
			return runOutcome{
				rawEvidence: artifactv2.RawEvidence{RunIdentity: runIdentity}, reusable: true,
			}, nil
		},
		func(context.Context, portableevaluation.PortableRequest) *umpirespb.ExecutionResult {
			return &umpirespb.ExecutionResult{Decision: umpirespb.EXECUTION_DECISION_PASS}
		},
		func() string { return "test.run.portable" },
	)
	plan := portableExecutionRequest()
	firstDone := make(chan error, 1)
	go func() {
		_, err := executor.Execute(context.Background(), plan)
		firstDone <- err
	}()
	<-entered

	var wait sync.WaitGroup
	errorsSeen := make(chan error, 9)
	for range 9 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := executor.Execute(context.Background(), plan)
			errorsSeen <- err
		}()
	}
	wait.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		requirePortableExecutorError(t, err, PortableErrorResourceExhausted)
	}
	require.Equal(t, int32(1), calls.Load())

	close(release)
	require.NoError(t, <-firstDone)
}

func TestPortableExecutorCancellationAndCleanupPoisoningRemainAtTheExecutionSeam(t *testing.T) {
	var prepares atomic.Int32
	executor := newPortableExecutor(
		nil, nil,
		func(context.Context, *umpirespb.PortableTestPlan, testplan.ModelProvenanceVerifier) (portablePreparation, error) {
			prepares.Add(1)
			return portablePreparation{executionTimeout: time.Minute}, nil
		},
		func(
			_ context.Context, _ artifact.AdmittedSet, _ runner.InputBinding, runIdentity string, _ runner.Adapter,
		) (runOutcome, error) {
			return runOutcome{
				rawEvidence:   artifactv2.RawEvidence{RunIdentity: runIdentity},
				cleanupStatus: umpirespb.CLEANUP_STATUS_INCOMPLETE, reusable: false,
			}, nil
		},
		func(context.Context, portableevaluation.PortableRequest) *umpirespb.ExecutionResult {
			return &umpirespb.ExecutionResult{
				CleanupStatus: umpirespb.EXECUTION_CLEANUP_STATUS_INCOMPLETE,
				Decision:      umpirespb.EXECUTION_DECISION_INCONCLUSIVE,
			}
		},
		func() string { return "test.run.portable" },
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := executor.Execute(ctx, portableExecutionRequest())
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(0), prepares.Load())

	result, err := executor.Execute(context.Background(), portableExecutionRequest())
	require.NoError(t, err)
	require.Equal(t, umpirespb.EXECUTION_DECISION_INCONCLUSIVE, result.GetDecision())

	_, err = executor.Execute(context.Background(), portableExecutionRequest())
	requirePortableExecutorError(t, err, PortableErrorFailedPrecondition)
	require.Equal(t, int32(1), prepares.Load())
}

func TestPortableExecutorRejectsPostDispatchInvariantWithoutAResult(t *testing.T) {
	plan := portableExecutorFixturePlan(t, false)
	executor := newPortableExecutor(
		nil, nil, preparePortableExecution,
		func(
			context.Context, artifact.AdmittedSet, runner.InputBinding, string, runner.Adapter,
		) (runOutcome, error) {
			return runOutcome{reusable: false}, portableExecutionError{
				err: errors.New("runtime invariant failed"), occurred: true,
			}
		},
		portableevaluation.EvaluatePortable,
		func() string { return "test.run.portable-invariant" },
	)

	result, err := executor.Execute(context.Background(), plan)

	require.Nil(t, result)
	requirePortableExecutorError(t, err, PortableErrorInternal)
}

func TestPortableExecutorPreservesCancellationDuringRuntimeAdmission(t *testing.T) {
	plan := portableExecutorFixturePlan(t, false)
	executor := newPortableExecutor(
		nil, nil, preparePortableExecution,
		func(
			context.Context, artifact.AdmittedSet, runner.InputBinding, string, runner.Adapter,
		) (runOutcome, error) {
			return runOutcome{reusable: true}, context.Canceled
		},
		portableevaluation.EvaluatePortable,
		func() string { return "test.run.portable-canceled-admission" },
	)

	result, err := executor.Execute(context.Background(), plan)

	require.Nil(t, result)
	require.ErrorIs(t, err, context.Canceled)
}

type portableExecutionError struct {
	err      error
	occurred bool
}

func (e portableExecutionError) Error() string { return e.err.Error() }
func (e portableExecutionError) Unwrap() error { return e.err }
func (e portableExecutionError) ExecutionOccurred() bool {
	return e.occurred
}

func portableExecutionRequest() *umpirespb.PortableTestPlan {
	return &umpirespb.PortableTestPlan{Limits: &umpirespb.PortableTestPlanLimits{
		Execution: &umpirespb.ExecutionLimits{MaxTotalDurationMilliseconds: 60_000},
	}}
}

func portableExecutorFixturePlan(t *testing.T, model bool) *umpirespb.PortableTestPlan {
	t.Helper()
	request := fixtureRequest(t, "normal")
	contract, err := evaluationcontract.Admit(request.GetEvaluationContract())
	require.NoError(t, err)
	input, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: request.GetInput().GetExperiment()},
		{Path: "artifacts/runtime-configuration.json", Encoded: request.GetInput().GetRuntimeConfig()},
	})
	require.NoError(t, err)
	executable, ok := input.Executable()
	require.True(t, ok)
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	execution := &umpirespb.ExecutionProgram{
		Setup: portableProjectionBinding("test.setup.portable-executor"),
		Query: proto.CloneOf(contract.GetQuery()),
		Behavior: &umpirespb.DefinitionBinding{
			DefinitionId:        experiment.Plan.BehaviorDefinitionID,
			BehaviorFingerprint: experiment.Plan.BehaviorFingerprint,
		},
		Target: proto.CloneOf(contract.GetImplementationLink().GetDestinationTarget()),
		Kernel: &umpirespb.DefinitionBinding{
			DefinitionId:        experiment.Plan.KernelDefinitionID,
			BehaviorFingerprint: experiment.Plan.KernelBehaviorFingerprint,
		},
		RoleBindings: []*umpirespb.RoleBinding{}, SymbolicRoles: []*umpirespb.SymbolicRole{},
		RuntimeBindingSlots: []*umpirespb.RuntimeBindingSlot{}, Preconditions: []*umpirespb.ExecutionPrecondition{},
		InitialState:           portableExecutorValue(contract, experiment.Plan.InitialState, umpirespb.PORTABLE_DEFINITION_KIND_STATE),
		RequestedActions:       portableExecutorValues(contract, experiment.Plan.RequestedActions, umpirespb.PORTABLE_DEFINITION_KIND_ACTION),
		ModelOutcomes:          portableExecutorValues(contract, experiment.Plan.ModelOutcomes, umpirespb.PORTABLE_DEFINITION_KIND_OUTCOME),
		ResultingStates:        portableExecutorValues(contract, experiment.Plan.ResultingStates, umpirespb.PORTABLE_DEFINITION_KIND_STATE),
		SelectedChoices:        portableExecutorValues(contract, experiment.Plan.SelectedChoices, umpirespb.PORTABLE_DEFINITION_KIND_STATE),
		SelectedVariants:       portableExecutorValues(contract, experiment.Plan.SelectedVariants, umpirespb.PORTABLE_DEFINITION_KIND_STATE),
		RequestedFaults:        portableExecutorValues(contract, experiment.Plan.RequestedFaults, umpirespb.PORTABLE_DEFINITION_KIND_ACTION),
		CapabilityRequirements: portableExecutorCapabilities(contract, experiment.Plan.CapabilityRequirementDefinitionIDs),
		Checkpoints:            portableExecutorCheckpoints(contract, experiment.Plan.Checkpoints),
		Runtime: &umpirespb.RuntimeProgram{
			AuthorityProfile: &umpirespb.DefinitionBinding{
				DefinitionId:        configuration.AuthorityProfile.DefinitionID,
				BehaviorFingerprint: configuration.AuthorityProfile.BehaviorFingerprint,
			},
			Config: &umpirespb.DefinitionBinding{
				DefinitionId:        configuration.ConfigurationDefinitionID,
				BehaviorFingerprint: configuration.BehaviorFingerprint,
			},
			ObservationConfig: &umpirespb.PortableObservationConfig{
				Profile: proto.CloneOf(contract.GetObservation().GetProfile().GetDefinition()),
				Program: proto.CloneOf(contract.GetObservation().GetDefinition()),
				Mapping: proto.CloneOf(contract.GetObservation().GetMapping()),
			},
			ParticipantBindings: portableExecutorParticipants(contract, configuration.ParticipantBindings),
			PhaseLimits:         portableExecutorPhaseLimits(configuration.PhaseLimits),
			Termination:         &umpirespb.TerminationObligation{Definition: portableProjectionBinding("test.termination.portable-executor")},
			Cleanup:             &umpirespb.CleanupObligation{Definition: portableProjectionBinding("test.cleanup.portable-executor")},
			AuthorityRequiredCapabilities: portableExecutorCapabilities(
				contract, configuration.AuthorityProfile.RequiredCapabilityDefinitionIDs,
			),
		},
		ArtifactProjection: &umpirespb.PlanArtifactProjection{
			ExpandedLimits: &umpirespb.PlanSearchLimits{
				MaxSemanticTransitions:  portableExecutorNatural(t, experiment.Plan.ExpandedLimits.Behavior.Transitions.Value),
				MaxSelectedActions:      portableExecutorNatural(t, experiment.Plan.ExpandedLimits.Behavior.SelectedActions.Value),
				MaxCandidateEvaluations: portableExecutorNatural(t, experiment.Plan.ExpandedLimits.Search.Value),
			},
			SelectionReason: portableExecutorSelectionReason(experiment.Plan.SelectionReason),
			Explored: &umpirespb.PlanExploredCounts{
				Setups:              portableExecutorNatural(t, experiment.Plan.Explored.Setups),
				Traces:              portableExecutorNatural(t, experiment.Plan.Explored.Traces),
				Transitions:         portableExecutorNatural(t, experiment.Plan.Explored.Transitions),
				PropertyEvaluations: portableExecutorNatural(t, experiment.Plan.Explored.PropertyEvaluations),
			},
			ExperimentKnownGaps:  portableExecutorKnownGaps(experiment.Plan.KnownGaps),
			ExperimentProvenance: portableExecutorArtifactProvenance(experiment.Plan.Provenance),
			RuntimeKnownGaps:     portableExecutorKnownGaps(configuration.KnownGaps),
			RuntimeProvenance:    portableExecutorArtifactProvenance(configuration.Provenance),
			ExperimentObservationRequirementDefinitionIds: append(
				[]string{}, experiment.ObservationRequirementDefinitionIDs...,
			),
			RuntimeObservationConfig: &umpirespb.PortableObservationConfig{
				Profile: portableExecutorBinding(
					configuration.Observation.ProfileDefinitionID,
					configuration.Observation.ProfileBehaviorFingerprint,
				),
				Program: portableExecutorBinding(
					configuration.Observation.ProgramDefinitionID,
					configuration.Observation.ProgramBehaviorFingerprint,
				),
				Mapping: portableExecutorBinding(
					configuration.Observation.MappingDefinitionID,
					configuration.Observation.MappingBehaviorFingerprint,
				),
			},
		},
	}
	for _, occurrence := range experiment.Plan.LinearExtension {
		execution.Occurrences = append(execution.Occurrences, &umpirespb.PlannedOccurrence{
			Definition:           portableProjectionBinding(occurrence.DefinitionID),
			ActionDefinitionId:   occurrence.ActionDefinitionID,
			Position:             portableExecutorNatural(t, occurrence.Position),
			AuthoredDefinitionId: occurrence.DefinitionID,
		})
	}
	plan := &umpirespb.PortableTestPlan{
		Version: &umpirespb.FormatVersion{Major: 1}, PlanId: "test.plan.portable-executor",
		Provenance: &umpirespb.PortableTestPlan_External{External: &umpirespb.ExternalPlanProvenance{
			Sources: portableExecutorLocations(experiment.Provenance.SourceLocations),
		}},
		Execution: execution,
		Verification: &umpirespb.VerificationProgram{
			Evidence:    proto.CloneOf(contract.GetObservation().GetProfile()),
			Observation: proto.CloneOf(contract.GetObservation()),
			TraceProjection: &umpirespb.VerificationProgram_RenameExactLink{
				RenameExactLink: proto.CloneOf(contract.GetImplementationLink()),
			},
			Properties: proto.CloneOf(&umpirespb.VerificationProgram{Properties: contract.GetProperties()}).GetProperties(),
			Decision:   &umpirespb.DecisionPolicy{Kind: umpirespb.DECISION_POLICY_KIND_STRICT_V1},
		},
		Limits: &umpirespb.PortableTestPlanLimits{
			Structural: &umpirespb.StructuralLimits{
				MaxPlanBytes: 1 << 20, MaxNestingDepth: 256, MaxCollectionItems: 10_000, MaxOperatorCount: 100_000,
			},
			Execution: &umpirespb.ExecutionLimits{
				MaxActions: 1, MaxFaults: 1, MaxPhaseAttempts: 1,
				MaxPhaseDurationMilliseconds: 30_000, MaxTotalDurationMilliseconds: 120_000,
			},
			Evidence: &umpirespb.EvidenceLimits{MaxRecords: 100_000, MaxBytes: 16 << 20, MaxSources: 10_000},
			Evaluation: &umpirespb.PortableEvaluationLimits{
				MaxExpressionDepth: 64, MaxNatural: "18446744073709551615", MaxWork: 10_000_000,
			},
			Output: &umpirespb.OutputLimits{MaxDiagnosticBytes: 64 << 10, MaxResultBytes: 4 << 20},
		},
		KnownGaps: append(
			portableExecutorKnownGaps(experiment.Plan.KnownGaps), portableExecutorKnownGaps(configuration.KnownGaps)...,
		),
		ExternalObligations: []*umpirespb.ExternalVerificationObligation{},
	}
	if model {
		projected, projectErr := projectPortableExecution(plan)
		require.NoError(t, projectErr)
		projectedExecutable, projectedOK := projected.Executable()
		require.True(t, projectedOK)
		experimentBinding, bindingErr := artifactv2.ExperimentArtifactBinding(projectedExecutable.Experiment())
		require.NoError(t, bindingErr)
		propertyBindings := make([]*umpirespb.DefinitionBinding, len(plan.GetVerification().GetProperties()))
		for index, property := range plan.GetVerification().GetProperties() {
			propertyBindings[index] = proto.CloneOf(property.GetDefinition())
		}
		plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: &umpirespb.ModelCompiledPlanProvenance{
			Test: portableProjectionBinding("test.definition.portable-executor"), Query: proto.CloneOf(plan.GetExecution().GetQuery()),
			Experiment: portableProtoArtifactBinding(experimentBinding),
			RuntimeConfig: portableProtoArtifactBinding(
				artifactv2.RuntimeConfigurationArtifactBinding(projectedExecutable.RuntimeConfiguration()),
			),
			Properties:       propertyBindings,
			CompilerContract: portableProjectionBinding("test.compiler.portable-executor"),
			Sources:          portableExecutorLocations(experiment.Provenance.SourceLocations),
		}}
	}
	sealed, err := testplan.Seal(plan)
	require.NoError(t, err)
	return sealed
}

func portableExecutorBinding(id, fingerprint string) *umpirespb.DefinitionBinding {
	return &umpirespb.DefinitionBinding{DefinitionId: id, BehaviorFingerprint: fingerprint}
}

func portableExecutorValue(
	contract *umpirespb.EvaluationContract,
	value artifactv2.ModelValue,
	kind umpirespb.PortableDefinitionKind,
) *umpirespb.PortableModelValue {
	binding := portableProjectionBinding(value.DefinitionID)
	modelValue := &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value.Value}}
	for _, entry := range contract.GetImplementationLink().GetEntries() {
		if entry.GetDestination().GetDefinition().GetDefinitionId() == value.DefinitionID &&
			portableModelValue(entry.GetDestination().GetValue()) == value.Value {
			binding = proto.CloneOf(entry.GetDestination().GetDefinition())
			modelValue = proto.CloneOf(entry.GetDestination().GetValue())
			break
		}
	}
	return &umpirespb.PortableModelValue{
		Definition: binding, Kind: kind,
		Value: modelValue,
	}
}

func portableExecutorValues(
	contract *umpirespb.EvaluationContract,
	values []artifactv2.ModelValue,
	kind umpirespb.PortableDefinitionKind,
) []*umpirespb.PortableModelValue {
	result := make([]*umpirespb.PortableModelValue, len(values))
	for index, value := range values {
		result[index] = portableExecutorValue(contract, value, kind)
	}
	return result
}

func portableExecutorCapabilities(
	contract *umpirespb.EvaluationContract,
	ids []string,
) []*umpirespb.DefinitionBinding {
	result := make([]*umpirespb.DefinitionBinding, len(ids))
	for index, id := range ids {
		result[index] = portableProjectionBinding(id)
		for _, entry := range contract.GetImplementationLink().GetDefinitionEntries() {
			if entry.GetDestination().GetDefinitionId() == id {
				result[index] = proto.CloneOf(entry.GetDestination())
				break
			}
		}
	}
	return result
}

func portableExecutorCheckpoints(
	contract *umpirespb.EvaluationContract,
	values []artifactv2.Checkpoint,
) []*umpirespb.ExecutionCheckpoint {
	result := make([]*umpirespb.ExecutionCheckpoint, len(values))
	for index, value := range values {
		result[index] = &umpirespb.ExecutionCheckpoint{
			Transition:   portableExecutorNatural(nil, value.Transition),
			Observations: portableExecutorValues(contract, value.Observations, umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION),
		}
	}
	return result
}

func portableExecutorParticipants(
	contract *umpirespb.EvaluationContract,
	values []artifactv2.ParticipantBinding,
) []*umpirespb.PortableParticipantBinding {
	result := make([]*umpirespb.PortableParticipantBinding, len(values))
	for index, value := range values {
		result[index] = &umpirespb.PortableParticipantBinding{
			Participant:     portableProjectionBinding(value.ParticipantDefinitionID),
			Protocol:        portableProjectionBinding(value.ProtocolDefinitionID),
			ProtocolVersion: portableExecutorNatural(nil, value.ProtocolVersion),
			Program:         portableExecutorBinding(value.ProgramDefinitionID, value.ProgramBehaviorFingerprint),
			Capabilities:    portableExecutorCapabilities(contract, value.CapabilityDefinitionIDs),
		}
	}
	return result
}

func portableExecutorPhaseLimits(values []artifactv2.PhaseLimit) []*umpirespb.ExecutionPhaseLimit {
	result := make([]*umpirespb.ExecutionPhaseLimit, len(values))
	for index, value := range values {
		result[index] = &umpirespb.ExecutionPhaseLimit{
			Phase: []umpirespb.ExecutionPhase{
				umpirespb.EXECUTION_PHASE_PREPARATION, umpirespb.EXECUTION_PHASE_REALIZATION,
				umpirespb.EXECUTION_PHASE_OBSERVATION, umpirespb.EXECUTION_PHASE_ISOLATION,
				umpirespb.EXECUTION_PHASE_CLEANUP,
			}[index],
			DurationMilliseconds: portableExecutorNatural(nil, value.DurationMilliseconds),
			MaxAttempts:          portableExecutorNatural(nil, value.MaxAttempts),
			MaxRecords:           portableExecutorNatural(nil, value.MaxRecords),
			MaxBytes:             portableExecutorNatural(nil, value.MaxBytes),
		}
	}
	return result
}

func portableExecutorLocations(values []artifactv2.SourceLocation) []*umpirespb.SourceLocation {
	result := make([]*umpirespb.SourceLocation, len(values))
	for index, value := range values {
		result[index] = &umpirespb.SourceLocation{
			Path: value.Path, Line: portableExecutorNatural(nil, value.Line),
			Column: portableExecutorNatural(nil, value.Column), Provenance: value.Provenance,
		}
	}
	return result
}

func portableExecutorArtifactProvenance(value artifactv2.Provenance) *umpirespb.PlanArtifactProvenance {
	return &umpirespb.PlanArtifactProvenance{
		SourceDefinitionIds: append([]string{}, value.SourceDefinitionIDs...),
		SourceLocations:     portableExecutorLocations(value.SourceLocations),
	}
}

func portableExecutorKnownGaps(values []artifactv2.KnownGap) []*umpirespb.KnownGap {
	result := make([]*umpirespb.KnownGap, len(values))
	for index, value := range values {
		result[index] = &umpirespb.KnownGap{Kind: portableExecutorKnownGapKind(value.Kind), Code: value.Code}
		if value.Subject != nil {
			result[index].Subject = *value.Subject
		}
		if value.Detail != nil {
			result[index].Detail = *value.Detail
		}
	}
	return result
}

func portableExecutorKnownGapKind(kind string) umpirespb.KnownGapKind {
	switch kind {
	case "capability-contract":
		return umpirespb.KNOWN_GAP_KIND_CAPABILITY_CONTRACT
	case "input":
		return umpirespb.KNOWN_GAP_KIND_INPUT
	case "interpretation":
		return umpirespb.KNOWN_GAP_KIND_INTERPRETATION
	case "claim":
		return umpirespb.KNOWN_GAP_KIND_CLAIM
	default:
		return umpirespb.KNOWN_GAP_KIND_UNSPECIFIED
	}
}

func portableExecutorSelectionReason(reason string) umpirespb.PlanSelectionReason {
	switch reason {
	case "satisfying-witness":
		return umpirespb.PLAN_SELECTION_REASON_SATISFYING_WITNESS
	case "violating-counterexample":
		return umpirespb.PLAN_SELECTION_REASON_VIOLATING_COUNTEREXAMPLE
	case "behavior-selection":
		return umpirespb.PLAN_SELECTION_REASON_BEHAVIOR_SELECTION
	default:
		return umpirespb.PLAN_SELECTION_REASON_UNSPECIFIED
	}
}

func portableExecutorNatural(t testing.TB, value artifactv2.Natural) int64 {
	parsed, err := strconv.ParseInt(value.String(), 10, 64)
	if t != nil {
		require.NoError(t, err)
	}
	return parsed
}

func requirePortableExecutorError(t *testing.T, err error, want PortableErrorCode) {
	t.Helper()
	code, ok := PortableCodeOf(err)
	require.True(t, ok)
	require.Equal(t, want, code)
}
