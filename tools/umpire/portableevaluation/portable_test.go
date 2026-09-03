package portableevaluation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/protobuf/proto"
)

func TestEvaluatePortableUsesTheExistingEvaluatorForPlanLocalResults(t *testing.T) {
	contract := testContract(t)
	authorized := testAuthorizedPortablePlan(t, contract, false)
	evidence := testRawEvidence(t, contract)
	request := requestFor(contract, evidence)

	result := EvaluatePortable(context.Background(), PortableRequest{
		Plan:                         authorized,
		RawEvidence:                  evidence,
		ExpectedRunIdentity:          request.ExpectedRunIdentity,
		ExpectedExperiment:           portableArtifactBinding(contract.GetExperiment()),
		ExpectedRuntimeConfiguration: portableArtifactBinding(contract.GetRuntimeConfig()),
		ExpectedRun:                  request.ExpectedRun,
		ExpectedClosures:             request.ExpectedClosures,
		OperationalStatus:            umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		CleanupStatus:                umpirespb.CLEANUP_STATUS_COMPLETE,
	})

	require.Equal(t, umpirespb.EXECUTION_TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.EXECUTION_OPERATIONAL_STATUS_SUCCEEDED, result.GetOperationalStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.TRACE_PROJECTION_STATUS_APPLIED, result.GetTraceProjection().GetStatus())
	require.Equal(t, umpirespb.EXECUTION_EVALUATION_STATUS_SATISFIED, result.GetSemanticStatus())
	require.Equal(t, umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE, result.GetCleanupStatus())
	require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
	require.Equal(t, umpirespb.CLAIM_SCOPE_PLAN_LOCAL, result.GetClaimScope())
	require.True(t, proto.Equal(
		&umpirespb.ExecutionResult{EvidenceLinks: result.GetObservation().GetEvidenceLinks()},
		&umpirespb.ExecutionResult{EvidenceLinks: result.GetEvidenceLinks()},
	))
}

func TestEvaluatePortableAppliesDirectPlanTraceWithoutARenameLink(t *testing.T) {
	contract := proto.CloneOf(testContract(t))
	for _, property := range contract.GetProperties() {
		for _, clause := range property.GetClauses() {
			for _, pattern := range []*umpirespb.Pattern{
				clause.GetPerStepImplies().GetTrigger(),
				clause.GetPerStepImplies().GetRequired(),
			} {
				pattern.Definition.DefinitionId = "system." + pattern.GetDefinition().GetDefinitionId()[len("feature."):]
			}
		}
	}
	authorized := testAuthorizedPortablePlan(t, contract, true)
	evidence := testRawEvidence(t, contract)
	request := requestFor(contract, evidence)

	result := EvaluatePortable(context.Background(), PortableRequest{
		Plan:                         authorized,
		RawEvidence:                  evidence,
		ExpectedRunIdentity:          request.ExpectedRunIdentity,
		ExpectedExperiment:           portableArtifactBinding(contract.GetExperiment()),
		ExpectedRuntimeConfiguration: portableArtifactBinding(contract.GetRuntimeConfig()),
		ExpectedRun:                  request.ExpectedRun,
		ExpectedClosures:             request.ExpectedClosures,
		OperationalStatus:            umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		CleanupStatus:                umpirespb.CLEANUP_STATUS_COMPLETE,
	})

	require.Equal(t, umpirespb.TRACE_PROJECTION_STATUS_DIRECT, result.GetTraceProjection().GetStatus())
	require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
	require.Empty(t, result.GetTraceProjection().GetApplications())
}

func TestEvaluatePortablePreservesTrustworthyDecisionsAndInconclusiveEvidence(t *testing.T) {
	tests := []struct {
		name         string
		contract     func(testing.TB) *umpirespb.EvaluationContract
		mutate       func(testing.TB, *artifactv2.RawEvidence, *[]artifactv2.SourceClosure)
		wantDecision umpirespb.ExecutionDecision
	}{
		{
			name: "trustworthy violation",
			contract: func(t testing.TB) *umpirespb.EvaluationContract {
				return testContractWith(t, func(contract *umpirespb.EvaluationContract) {
					contract.Properties[0].Clauses[1].PerStepImplies.Required = textPattern(
						umpirespb.TRACE_FIELD_OBSERVATION, "feature.observation.rendered", "2",
					)
				})
			},
			wantDecision: umpirespb.EXECUTION_DECISION_FAIL,
		},
		{
			name:     "unclosed evidence",
			contract: testContract,
			mutate: func(t testing.TB, evidence *artifactv2.RawEvidence, closures *[]artifactv2.SourceClosure) {
				evidence.Sources[0].Status = "partial"
				evidence.CaptureStatus = "partial"
				*evidence = resealRawEvidence(t, *evidence)
				(*closures)[0].Status = "partial"
			},
			wantDecision: umpirespb.EXECUTION_DECISION_INCONCLUSIVE,
		},
		{
			name:     "crossed run",
			contract: testContract,
			mutate: func(t testing.TB, evidence *artifactv2.RawEvidence, _ *[]artifactv2.SourceClosure) {
				evidence.RunIdentity = "test.run.crossed"
				*evidence = resealRawEvidence(t, *evidence)
			},
			wantDecision: umpirespb.EXECUTION_DECISION_INCONCLUSIVE,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contract := test.contract(t)
			evidence := testRawEvidence(t, contract)
			request := requestFor(contract, evidence)
			if test.mutate != nil {
				test.mutate(t, &evidence, &request.ExpectedClosures)
			}

			result := EvaluatePortable(context.Background(), portableRequestFor(
				testAuthorizedPortablePlan(t, contract, false), contract, evidence, request,
			))

			require.Equal(t, test.wantDecision, result.GetDecision())
			require.NotEqual(t, umpirespb.EXECUTION_TOOLING_STATUS_INTERNAL_ERROR, result.GetToolingStatus())
		})
	}
}

func TestEvaluatePortableEnforcesWorkAndResultBoundaries(t *testing.T) {
	contract := testContract(t)
	evidence := testRawEvidence(t, contract)
	request := requestFor(contract, evidence)
	baseline := EvaluatePortable(context.Background(), portableRequestFor(
		testAuthorizedPortablePlan(t, contract, false), contract, evidence, request,
	))
	require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, baseline.GetDecision())

	t.Run("work N and N plus one", func(t *testing.T) {
		for _, test := range []struct {
			name string
			max  int64
			want umpirespb.ExecutionDecision
		}{
			{name: "exact N", max: baseline.GetWork().GetTotal(), want: umpirespb.EXECUTION_DECISION_PASS},
			{name: "N plus one", max: baseline.GetWork().GetTotal() - 1, want: umpirespb.EXECUTION_DECISION_INCONCLUSIVE},
		} {
			t.Run(test.name, func(t *testing.T) {
				authorized := testAuthorizedPortablePlanWith(t, contract, false, func(plan *umpirespb.PortableTestPlan) {
					plan.Limits.Evaluation.MaxWork = test.max
				})
				result := EvaluatePortable(context.Background(), portableRequestFor(authorized, contract, evidence, request))
				require.Equal(t, test.want, result.GetDecision())
			})
		}
	})

	t.Run("result bytes N and N plus one", func(t *testing.T) {
		required := int64(proto.Size(baseline))
		exact := testAuthorizedPortablePlanWith(t, contract, false, func(plan *umpirespb.PortableTestPlan) {
			plan.Limits.Output.MaxResultBytes = required
			plan.Limits.Output.MaxDiagnosticBytes = 1024
		})
		exactResult := EvaluatePortable(context.Background(), portableRequestFor(exact, contract, evidence, request))
		require.Equal(t, required, int64(proto.Size(exactResult)))
		require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, exactResult.GetDecision())

		below := testAuthorizedPortablePlanWith(t, contract, false, func(plan *umpirespb.PortableTestPlan) {
			plan.Limits.Output.MaxResultBytes = required - 1
			plan.Limits.Output.MaxDiagnosticBytes = 1024
		})
		belowResult := EvaluatePortable(context.Background(), portableRequestFor(below, contract, evidence, request))
		require.LessOrEqual(t, int64(proto.Size(belowResult)), required-1)
		require.Equal(t, umpirespb.EXECUTION_DECISION_INCONCLUSIVE, belowResult.GetDecision())
		require.NotEqual(t, umpirespb.EXECUTION_TOOLING_STATUS_INTERNAL_ERROR, belowResult.GetToolingStatus())
		require.Empty(t, belowResult.GetProperties())
		require.Empty(t, belowResult.GetEvidenceLinks())
		require.Equal(t, umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED, belowResult.GetDiagnostics()[0].GetCode())
	})
}

func portableRequestFor(
	plan testplan.AuthorizedPlan,
	contract *umpirespb.EvaluationContract,
	evidence artifactv2.RawEvidence,
	request Request,
) PortableRequest {
	return PortableRequest{
		Plan: plan, RawEvidence: evidence, ExpectedRunIdentity: request.ExpectedRunIdentity,
		ExpectedExperiment:           portableArtifactBinding(contract.GetExperiment()),
		ExpectedRuntimeConfiguration: portableArtifactBinding(contract.GetRuntimeConfig()),
		ExpectedRun:                  request.ExpectedRun, ExpectedClosures: request.ExpectedClosures,
		OperationalStatus: umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		CleanupStatus:     umpirespb.CLEANUP_STATUS_COMPLETE,
	}
}

func testAuthorizedPortablePlan(
	t testing.TB,
	contract *umpirespb.EvaluationContract,
	direct bool,
) testplan.AuthorizedPlan {
	return testAuthorizedPortablePlanWith(t, contract, direct, nil)
}

func testAuthorizedPortablePlanWith(
	t testing.TB,
	contract *umpirespb.EvaluationContract,
	direct bool,
	mutate func(*umpirespb.PortableTestPlan),
) testplan.AuthorizedPlan {
	t.Helper()
	verification := &umpirespb.VerificationProgram{
		Evidence:    proto.CloneOf(contract.GetObservation().GetProfile()),
		Observation: proto.CloneOf(contract.GetObservation()),
		Properties:  clonePortableProperties(contract.GetProperties()),
		Decision:    &umpirespb.DecisionPolicy{Kind: umpirespb.DECISION_POLICY_KIND_STRICT_V1},
	}
	if direct {
		verification.TraceProjection = &umpirespb.VerificationProgram_DirectPlanTrace{
			DirectPlanTrace: &umpirespb.DirectPlanTrace{},
		}
	} else {
		verification.TraceProjection = &umpirespb.VerificationProgram_RenameExactLink{
			RenameExactLink: proto.CloneOf(contract.GetImplementationLink()),
		}
	}
	plan := &umpirespb.PortableTestPlan{
		Version: &umpirespb.FormatVersion{Major: 1},
		PlanId:  "test.plan.portable-evaluation",
		Provenance: &umpirespb.PortableTestPlan_External{External: &umpirespb.ExternalPlanProvenance{
			Sources: []*umpirespb.SourceLocation{testLocation()},
		}},
		Execution:    portableExecutionProgram(contract),
		Verification: verification,
		Limits: &umpirespb.PortableTestPlanLimits{
			Structural: &umpirespb.StructuralLimits{
				MaxPlanBytes: 1 << 20, MaxNestingDepth: 256,
				MaxCollectionItems: 10_000, MaxOperatorCount: 100_000,
			},
			Execution: &umpirespb.ExecutionLimits{
				MaxActions: 1, MaxFaults: 1, MaxPhaseAttempts: 1,
				MaxPhaseDurationMilliseconds: 30_000,
				MaxTotalDurationMilliseconds: 120_000,
			},
			Evidence: &umpirespb.EvidenceLimits{
				MaxRecords: 100_000, MaxBytes: 16 << 20, MaxSources: 10_000,
			},
			Evaluation: &umpirespb.PortableEvaluationLimits{
				MaxExpressionDepth: 64, MaxNatural: "18446744073709551615",
				MaxWork: 10_000_000,
			},
			Output: &umpirespb.OutputLimits{
				MaxDiagnosticBytes: 64 << 10, MaxResultBytes: 4 << 20,
			},
		},
		KnownGaps:           []*umpirespb.KnownGap{},
		ExternalObligations: []*umpirespb.ExternalVerificationObligation{},
	}
	if mutate != nil {
		mutate(plan)
	}
	sealed, err := testplan.Seal(plan)
	require.NoError(t, err)
	admitted, err := testplan.Admit(sealed)
	require.NoError(t, err)
	authorized, err := testplan.Authorize(context.Background(), admitted, nil)
	require.NoError(t, err)
	return authorized
}

func portableExecutionProgram(contract *umpirespb.EvaluationContract) *umpirespb.ExecutionProgram {
	capability := testBinding("test.capability.runtime")
	return &umpirespb.ExecutionProgram{
		Setup:               testBinding("test.setup.portable"),
		Query:               proto.CloneOf(contract.GetQuery()),
		Behavior:            testBinding("test.behavior.portable"),
		Target:              proto.CloneOf(contract.GetImplementationLink().GetSourceTarget()),
		Kernel:              testBinding("test.kernel.portable"),
		RoleBindings:        []*umpirespb.RoleBinding{},
		SymbolicRoles:       []*umpirespb.SymbolicRole{},
		RuntimeBindingSlots: []*umpirespb.RuntimeBindingSlot{},
		Preconditions:       []*umpirespb.ExecutionPrecondition{},
		InitialState: portableModelValue(
			"system.state", umpirespb.PORTABLE_DEFINITION_KIND_STATE, textValue("open"),
		),
		RequestedActions: []*umpirespb.PortableModelValue{portableModelValue(
			"system.action", umpirespb.PORTABLE_DEFINITION_KIND_ACTION, textValue("close"),
		)},
		ModelOutcomes: []*umpirespb.PortableModelValue{portableModelValue(
			"system.outcome", umpirespb.PORTABLE_DEFINITION_KIND_OUTCOME, textValue("done"),
		)},
		ResultingStates: []*umpirespb.PortableModelValue{portableModelValue(
			"system.state", umpirespb.PORTABLE_DEFINITION_KIND_STATE, textValue("closed"),
		)},
		Occurrences: []*umpirespb.PlannedOccurrence{{
			Definition:         testBinding("test.occurrence.close"),
			ActionDefinitionId: "system.action", Position: 1,
			AuthoredDefinitionId: "test.occurrence.close",
		}},
		SelectedChoices:        []*umpirespb.PortableModelValue{},
		SelectedVariants:       []*umpirespb.PortableModelValue{},
		RequestedFaults:        []*umpirespb.PortableModelValue{},
		CapabilityRequirements: []*umpirespb.DefinitionBinding{capability},
		Checkpoints: []*umpirespb.ExecutionCheckpoint{{
			Transition: 1,
			Observations: []*umpirespb.PortableModelValue{
				portableModelValue("system.observation.count", umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION, naturalValue("1")),
				portableModelValue("system.observation.rendered", umpirespb.PORTABLE_DEFINITION_KIND_OBSERVATION, textValue("1")),
			},
		}},
		Runtime: &umpirespb.RuntimeProgram{
			AuthorityProfile: testBinding("test.authority.portable"),
			Config:           testBinding("test.runtime-config.portable"),
			ParticipantBindings: []*umpirespb.PortableParticipantBinding{{
				Participant: testBinding("test.participant.portable"),
				Protocol:    testBinding("test.protocol.portable"), ProtocolVersion: 2,
				Program:      testBinding("test.program.portable"),
				Capabilities: []*umpirespb.DefinitionBinding{capability},
			}},
			ObservationConfig: &umpirespb.PortableObservationConfig{
				Profile: proto.CloneOf(contract.GetObservation().GetProfile().GetDefinition()),
				Program: proto.CloneOf(contract.GetObservation().GetDefinition()),
				Mapping: proto.CloneOf(contract.GetObservation().GetMapping()),
			},
			PhaseLimits:                   portablePhaseLimits(),
			Termination:                   &umpirespb.TerminationObligation{Definition: testBinding("test.termination.portable")},
			Cleanup:                       &umpirespb.CleanupObligation{Definition: testBinding("test.cleanup.portable")},
			AuthorityRequiredCapabilities: []*umpirespb.DefinitionBinding{capability},
		},
	}
}

func portableModelValue(
	id string,
	kind umpirespb.PortableDefinitionKind,
	value *umpirespb.Value,
) *umpirespb.PortableModelValue {
	return &umpirespb.PortableModelValue{Definition: testBinding(id), Kind: kind, Value: value}
}

func clonePortableProperties(properties []*umpirespb.Property) []*umpirespb.Property {
	return proto.CloneOf(&umpirespb.VerificationProgram{Properties: properties}).GetProperties()
}

func portablePhaseLimits() []*umpirespb.ExecutionPhaseLimit {
	return []*umpirespb.ExecutionPhaseLimit{
		{Phase: umpirespb.EXECUTION_PHASE_PREPARATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_REALIZATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_OBSERVATION, DurationMilliseconds: 30_000, MaxAttempts: 1, MaxRecords: 3_584, MaxBytes: 12 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_ISOLATION, DurationMilliseconds: 15_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
		{Phase: umpirespb.EXECUTION_PHASE_CLEANUP, DurationMilliseconds: 15_000, MaxAttempts: 1, MaxRecords: 128, MaxBytes: 1 << 20},
	}
}

func portableArtifactBinding(binding *umpirespb.ArtifactBinding) artifactv2.ArtifactBinding {
	return artifactv2.ArtifactBinding{
		FormatVersion: binding.GetFormatVersion(), ArtifactChecksum: binding.GetArtifactChecksum(),
		BehaviorFingerprint: binding.GetBehaviorFingerprint(), ProvenanceChecksum: binding.GetProvenanceChecksum(),
	}
}
