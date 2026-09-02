package portableevaluation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"google.golang.org/protobuf/proto"
)

const fixtureDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

func TestEvaluateSatisfiedClosedEvidence(t *testing.T) {
	request := testRequest(t)

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.OPERATIONAL_STATUS_SUCCEEDED, result.GetOperationalStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED, result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.EVALUATION_STATUS_SATISFIED, result.GetSemanticStatus())
	require.Equal(t, umpirespb.CLEANUP_STATUS_COMPLETE, result.GetCleanupStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, result.GetDecision())
	require.Len(t, result.GetProperties(), 1)
	require.Equal(t, umpirespb.SEMANTIC_STATUS_SATISFIED, result.GetProperties()[0].GetStatus())
	protorequire.ProtoEqual(t, &umpirespb.ModelTrace{
		InitialState: testModelValue("feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
		Steps: []*umpirespb.ModelTraceStep{{
			Position:       1,
			PriorState:     testModelValue("feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
			SelectedAction: testModelValue("feature.action", umpirespb.DEFINITION_KIND_ACTION, textValue("close")),
			ModelOutcome:   testModelValue("feature.outcome", umpirespb.DEFINITION_KIND_OUTCOME, textValue("done")),
			ResultingState: testModelValue("feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("closed")),
			Observations: []*umpirespb.ModelValue{
				testModelValue("feature.observation.count", umpirespb.DEFINITION_KIND_OBSERVATION, naturalValue("1")),
				testModelValue("feature.observation.rendered", umpirespb.DEFINITION_KIND_OBSERVATION, textValue("1")),
			},
		}},
	}, withoutTraceID(result.GetImplementationLink().GetTrace()))
	for _, link := range result.GetObservation().GetEvidenceLinks() {
		require.NotEmpty(t, link.GetEvidenceDefinitionIds())
		require.Len(t, link.GetOrderingSupport(), 2)
		require.Len(t, link.GetClosureSupport(), 1)
		require.NotNil(t, link.GetMapping())
	}
	links := result.GetObservation().GetEvidenceLinks()
	require.Same(t, links[0].GetOrderingSupport()[0], links[1].GetOrderingSupport()[0])
	require.Same(t, links[0].GetClosureSupport()[0], links[1].GetClosureSupport()[0])
}

func TestEvaluateViolatedClosedEvidence(t *testing.T) {
	contract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Properties[0].Clauses[1].PerStepImplies.Required = textPattern(
			umpirespb.TRACE_FIELD_OBSERVATION,
			"feature.observation.rendered",
			"2",
		)
	})
	request := requestFor(contract, testRawEvidence(t, contract))

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED, result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.SEMANTIC_STATUS_VIOLATED, result.GetProperties()[0].GetStatus())
	require.Equal(t, umpirespb.EVALUATION_STATUS_VIOLATED, result.GetSemanticStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_FAIL, result.GetDecision())
}

func TestEvaluateIncompleteClosureIsUnknown(t *testing.T) {
	request := testRequest(t)
	request.RawEvidence.Sources[0].Status = "partial"
	request.RawEvidence.CaptureStatus = "partial"
	request.RawEvidence = resealRawEvidence(t, request.RawEvidence)
	request.ExpectedClosures[0].Status = "partial"

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_MISSING_CLOSURE, result.GetObservation().GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED, result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.EVALUATION_STATUS_INCOMPLETE, result.GetSemanticStatus())
	require.Empty(t, result.GetProperties())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateMissingEvidenceFromClosedSourceIsInconclusive(t *testing.T) {
	contract := testContract(t)
	evidence := testRawEvidence(t, contract)
	evidence.Facts = evidence.Facts[:1]
	evidence.Sources[0].FactCount = artifactv2.NaturalFromUint64(1)
	evidence = resealRawEvidence(t, evidence)

	result := Evaluate(context.Background(), requestFor(contract, evidence))

	require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING,
		result.GetObservation().GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.EVALUATION_STATUS_INCOMPLETE, result.GetSemanticStatus())
	require.Empty(t, result.GetProperties())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateConflictingCorrelation(t *testing.T) {
	contract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Observation.Profile.CorrelationSlots = []*umpirespb.CorrelationSlot{{
			DefinitionId: "evidence.correlation.event",
			Kind:         umpirespb.CORRELATION_SLOT_KIND_RUN,
			Fields: []*umpirespb.EvidenceFieldReference{{
				KindDefinitionId:  "evidence.kind.event",
				FieldDefinitionId: "evidence.field.event",
			}},
		}}
	})
	request := requestFor(contract, testRawEvidence(t, contract))

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.OBSERVATION_STATUS_CONFLICT, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_CORRELATION, result.GetObservation().GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.EVALUATION_STATUS_INCOMPLETE, result.GetSemanticStatus())
	require.Empty(t, result.GetProperties())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateUnsupportedEvidenceType(t *testing.T) {
	request := testRequest(t)
	request.RawEvidence.Facts[0].Fields[1].Value = true
	request.RawEvidence = resealRawEvidence(t, request.RawEvidence)

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNSUPPORTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, result.GetObservation().GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.EVALUATION_STATUS_INCOMPLETE, result.GetSemanticStatus())
	require.Empty(t, result.GetProperties())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := Evaluate(ctx, testRequest(t))

	require.Equal(t, umpirespb.TOOLING_STATUS_CANCELED, result.GetToolingStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateWorkLimitExactBoundary(t *testing.T) {
	baseline := Evaluate(context.Background(), testRequest(t))
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, baseline.GetDecision())
	requiredWork := baseline.GetWork().GetTotal()

	exactContract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Limits.MaxEvaluationWork = requiredWork
	})
	exact := Evaluate(context.Background(), requestFor(exactContract, testRawEvidence(t, exactContract)))
	require.Equal(t, requiredWork, exact.GetWork().GetTotal())
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, exact.GetDecision())

	belowContract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Limits.MaxEvaluationWork = requiredWork - 1
	})
	below := Evaluate(context.Background(), requestFor(belowContract, testRawEvidence(t, belowContract)))
	require.Equal(t, requiredWork-1, below.GetWork().GetTotal())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, below.GetDecision())
	require.Contains(t, allDiagnostics(below), umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED)
}

func TestEvaluateInputLimitsExactBoundaries(t *testing.T) {
	baseContract := testContract(t)
	evidence := testRawEvidence(t, baseContract)
	encoded, err := artifactv2.CanonicalRawEvidenceBytes(evidence)
	require.NoError(t, err)

	tests := []struct {
		name       string
		modify     func(*umpirespb.EvaluationContract)
		wantStatus umpirespb.ObservationStatus
		wantLimit  bool
	}{
		{
			name: "input bytes exact N",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Limits.MaxInputBytes = int64(len(encoded))
			},
			wantStatus: umpirespb.OBSERVATION_STATUS_ACCEPTED,
		},
		{
			name: "input bytes N plus one",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Limits.MaxInputBytes = int64(len(encoded)) - 1
			},
			wantStatus: umpirespb.OBSERVATION_STATUS_UNKNOWN,
			wantLimit:  true,
		},
		{
			name: "records exact N",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Limits.MaxEvidenceRecords = int64(len(evidence.Facts))
			},
			wantStatus: umpirespb.OBSERVATION_STATUS_ACCEPTED,
		},
		{
			name: "records N plus one",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Limits.MaxEvidenceRecords = int64(len(evidence.Facts)) - 1
				contract.Observation.Profile.Cardinalities[0].Minimum = 1
				contract.Observation.Profile.Cardinalities[0].Maximum = 1
			},
			wantStatus: umpirespb.OBSERVATION_STATUS_UNKNOWN,
			wantLimit:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contract := testContractWith(t, test.modify)

			result := Evaluate(context.Background(), requestFor(contract, evidence))

			require.Equal(t, test.wantStatus, result.GetObservation().GetStatus())
			if test.wantLimit {
				require.Contains(t, allDiagnostics(result), umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED)
				require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
			}
		})
	}
}

func TestEvaluateResultLimitExactBoundary(t *testing.T) {
	baseline := Evaluate(context.Background(), testRequest(t))
	requiredBytes := int64(proto.Size(baseline))
	require.Greater(t, requiredBytes, int64(4096))

	exactContract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Limits.MaxResultBytes = requiredBytes
	})
	exact := Evaluate(context.Background(), requestFor(exactContract, testRawEvidence(t, exactContract)))
	require.Equal(t, requiredBytes, int64(proto.Size(exact)))
	require.Equal(t, umpirespb.CANARY_DECISION_PASS, exact.GetDecision())

	belowContract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Limits.MaxResultBytes = requiredBytes - 1
	})
	below := Evaluate(context.Background(), requestFor(belowContract, testRawEvidence(t, belowContract)))
	require.LessOrEqual(t, int64(proto.Size(below)), requiredBytes-1)
	require.Equal(t, umpirespb.TOOLING_STATUS_INTERNAL_ERROR, below.GetToolingStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, below.GetDecision())
	require.Contains(t, allDiagnostics(below), umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED)
}

func TestNormalizeNaturalExactBoundary(t *testing.T) {
	interpreter := expressionTestInterpreter()

	value, failure := interpreter.normalizeValue(umpirespb.VALUE_KIND_NATURAL, "100", []string{"field"})
	require.Nil(t, failure)
	protorequire.ProtoEqual(t, naturalValue("100"), value)

	_, failure = interpreter.normalizeValue(umpirespb.VALUE_KIND_NATURAL, "101", []string{"field"})
	require.NotNil(t, failure)
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_NATURAL_OUT_OF_RANGE, failure.code)
}

func TestEvaluateDoesNotMutateInputs(t *testing.T) {
	request := testRequest(t)
	contractBefore := proto.CloneOf(request.Contract)
	evidenceBefore := request.RawEvidence

	result := Evaluate(context.Background(), request)
	result.Observation.Trace.InitialState.Definition.DefinitionId = "mutated.result"

	protorequire.ProtoEqual(t, contractBefore, request.Contract)
	require.True(t, reflect.DeepEqual(evidenceBefore, request.RawEvidence))
}

func TestEvaluatePropertyPatternOutcomes(t *testing.T) {
	tests := []struct {
		name           string
		modify         func(*umpirespb.EvaluationContract)
		propertyStatus umpirespb.SemanticStatus
		evaluation     umpirespb.EvaluationStatus
		decision       umpirespb.CanaryDecision
	}{
		{
			name: "natural false is violation",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Properties[0].Clauses[0].PerStepImplies.Required = naturalPattern(
					"feature.observation.count", "0",
				)
			},
			propertyStatus: umpirespb.SEMANTIC_STATUS_VIOLATED,
			evaluation:     umpirespb.EVALUATION_STATUS_VIOLATED,
			decision:       umpirespb.CANARY_DECISION_FAIL,
		},
		{
			name: "missing required value is violation",
			modify: func(contract *umpirespb.EvaluationContract) {
				declareMissingObservationVocabulary(contract, naturalValue("0"))
				contract.Properties[0].Clauses[0].PerStepImplies.Required = naturalPattern(
					"feature.observation.missing", "1",
				)
			},
			propertyStatus: umpirespb.SEMANTIC_STATUS_VIOLATED,
			evaluation:     umpirespb.EVALUATION_STATUS_VIOLATED,
			decision:       umpirespb.CANARY_DECISION_FAIL,
		},
		{
			name: "missing trigger is vacuously satisfied",
			modify: func(contract *umpirespb.EvaluationContract) {
				declareMissingObservationVocabulary(contract, textValue("unused"))
				contract.Properties[0].Clauses[0].PerStepImplies.Trigger = textPattern(
					umpirespb.TRACE_FIELD_OBSERVATION, "feature.observation.missing", "unused",
				)
			},
			propertyStatus: umpirespb.SEMANTIC_STATUS_SATISFIED,
			evaluation:     umpirespb.EVALUATION_STATUS_SATISFIED,
			decision:       umpirespb.CANARY_DECISION_PASS,
		},
		{
			name: "equals text rejects natural",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Properties[0].Clauses[0].PerStepImplies.Required = textPattern(
					umpirespb.TRACE_FIELD_OBSERVATION, "feature.observation.count", "1",
				)
			},
			propertyStatus: umpirespb.SEMANTIC_STATUS_UNSUPPORTED,
			evaluation:     umpirespb.EVALUATION_STATUS_INCOMPLETE,
			decision:       umpirespb.CANARY_DECISION_INCONCLUSIVE,
		},
		{
			name: "natural at most rejects text",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Properties[0].Clauses[1].PerStepImplies.Required = naturalPattern(
					"feature.observation.rendered", "1",
				)
			},
			propertyStatus: umpirespb.SEMANTIC_STATUS_UNSUPPORTED,
			evaluation:     umpirespb.EVALUATION_STATUS_INCOMPLETE,
			decision:       umpirespb.CANARY_DECISION_INCONCLUSIVE,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contract := testContractWith(t, test.modify)
			result := Evaluate(context.Background(), requestFor(contract, testRawEvidence(t, contract)))

			require.Equal(t, test.propertyStatus, result.GetProperties()[0].GetStatus())
			require.Equal(t, test.evaluation, result.GetSemanticStatus())
			require.Equal(t, test.decision, result.GetDecision())
		})
	}
}

func TestEvaluatePropertyDestinationVocabulary(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*umpirespb.EvaluationContract)
		wantStatus umpirespb.SemanticStatus
		wantCode   umpirespb.DiagnosticCode
		decision   umpirespb.CanaryDecision
	}{
		{
			name: "undeclared pattern",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Properties[0].Clauses[0].PerStepImplies.Trigger.Definition =
					testBinding("feature.action.undeclared")
			},
			wantStatus: umpirespb.SEMANTIC_STATUS_UNSUPPORTED,
			wantCode:   umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING,
			decision:   umpirespb.CANARY_DECISION_INCONCLUSIVE,
		},
		{
			name: "undeclared requirement",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.Properties[0].Requirements = []*umpirespb.DefinitionBinding{
					testBinding("feature.capability"),
				}
			},
			wantStatus: umpirespb.SEMANTIC_STATUS_UNSUPPORTED,
			wantCode:   umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING,
			decision:   umpirespb.CANARY_DECISION_INCONCLUSIVE,
		},
		{
			name: "crossed requirement kind",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.ImplementationLink.DefinitionEntries = []*umpirespb.DefinitionRenameEntry{{
					Source: testBinding("system.capability"), Kind: umpirespb.DEFINITION_KIND_RELATION,
					Destination: testBinding("feature.capability"),
				}}
				contract.Properties[0].Requirements = []*umpirespb.DefinitionBinding{
					testBinding("feature.capability"),
				}
			},
			wantStatus: umpirespb.SEMANTIC_STATUS_UNSUPPORTED,
			wantCode:   umpirespb.DIAGNOSTIC_CODE_MISSING_BINDING,
			decision:   umpirespb.CANARY_DECISION_INCONCLUSIVE,
		},
		{
			name: "declared requirement",
			modify: func(contract *umpirespb.EvaluationContract) {
				contract.ImplementationLink.DefinitionEntries = []*umpirespb.DefinitionRenameEntry{{
					Source: testBinding("system.capability"), Kind: umpirespb.DEFINITION_KIND_CAPABILITY,
					Destination: testBinding("feature.capability"),
				}}
				contract.Properties[0].Requirements = []*umpirespb.DefinitionBinding{
					testBinding("feature.capability"),
				}
			},
			wantStatus: umpirespb.SEMANTIC_STATUS_SATISFIED,
			decision:   umpirespb.CANARY_DECISION_PASS,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contract := testContractWith(t, test.modify)

			result := Evaluate(context.Background(), requestFor(contract, testRawEvidence(t, contract)))

			require.Equal(t, test.wantStatus, result.GetProperties()[0].GetStatus())
			require.Equal(t, test.decision, result.GetDecision())
			if test.wantCode != umpirespb.DIAGNOSTIC_CODE_UNSPECIFIED {
				require.Contains(t, allDiagnostics(result), test.wantCode)
			}
		})
	}
}

func TestDestinationPatternRequiresExactFingerprint(t *testing.T) {
	contract := testContract(t)
	pattern := proto.CloneOf(contract.GetProperties()[0].GetClauses()[0].GetPerStepImplies().GetTrigger())
	pattern.Definition.BehaviorFingerprint =
		"sha256:1111111111111111111111111111111111111111111111111111111111111111"

	require.False(t, (&interpreter{contract: contract}).destinationDefinesPattern(pattern))
}

func TestEvaluateRetainsExactBindingsAndClauseEvidence(t *testing.T) {
	request := testRequest(t)

	result := Evaluate(context.Background(), request)

	require.Len(t, result.GetImplementationLink().GetApplications(), 7)
	for _, application := range result.GetImplementationLink().GetApplications() {
		require.NotNil(t, application.GetEntry())
		require.NotNil(t, application.GetSourceEvidenceLink())
		found := false
		for _, entry := range request.Contract.GetImplementationLink().GetEntries() {
			found = found || proto.Equal(entry, application.GetEntry())
		}
		require.True(t, found)
	}
	property := result.GetProperties()[0]
	protorequire.ProtoEqual(t, request.Contract.GetProperties()[0].GetDefinition(), property.GetProperty())
	for _, clause := range property.GetClauses() {
		require.NotEmpty(t, clause.GetCoordinates())
		require.NotEmpty(t, clause.GetEvidenceLinks())
		for _, link := range clause.GetEvidenceLinks() {
			protorequire.ProtoEqual(t, request.Contract.GetObservation().GetMapping(), link.GetMapping())
		}
	}
}

func TestEvaluateMissingRenameExactMapping(t *testing.T) {
	contract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.ImplementationLink.Entries = contract.ImplementationLink.Entries[1:]
	})

	result := Evaluate(context.Background(), requestFor(contract, testRawEvidence(t, contract)))

	require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_UNKNOWN, result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_MISSING_LINK_MAPPING, result.GetImplementationLink().GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateRejectsCrossedArtifactBinding(t *testing.T) {
	request := testRequest(t)
	request.RawEvidence.Experiment.ArtifactChecksum = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	request.RawEvidence = resealRawEvidence(t, request.RawEvidence)

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_INPUT, result.GetToolingStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_CORRELATION, result.GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateRejectsStaleRunAndClosure(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{
			name: "run identity",
			mutate: func(request *Request) {
				request.RawEvidence.RunIdentity = "test.run.stale"
				request.RawEvidence = resealRawEvidence(t, request.RawEvidence)
			},
		},
		{
			name: "run binding",
			mutate: func(request *Request) {
				request.RawEvidence.Run.ArtifactChecksum =
					"sha256:1111111111111111111111111111111111111111111111111111111111111111"
				request.RawEvidence = resealRawEvidence(t, request.RawEvidence)
			},
		},
		{
			name: "post closure evidence",
			mutate: func(request *Request) {
				postClosure := request.RawEvidence.Facts[1]
				postClosure.FactDefinitionID = "evidence.fact.post-closure"
				postClosure.Ordinal = artifactv2.NaturalFromUint64(2)
				postClosure.CausalFactDefinitionIDs = []string{"evidence.fact.finish"}
				request.RawEvidence.Facts = append(request.RawEvidence.Facts, postClosure)
				request.RawEvidence.Sources[0].FactCount = artifactv2.NaturalFromUint64(3)
				request.RawEvidence = resealRawEvidence(t, request.RawEvidence)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest(t)
			test.mutate(&request)

			result := Evaluate(context.Background(), request)

			require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_INPUT, result.GetToolingStatus())
			require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
			require.Equal(t, umpirespb.DIAGNOSTIC_CLASS_CONFLICT, result.GetDiagnostics()[0].GetDiagnosticClass())
			require.Equal(t, umpirespb.DIAGNOSTIC_CODE_CORRELATION, result.GetDiagnostics()[0].GetCode())
			require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
		})
	}
}

func TestEvaluateRejectsEvidenceKindCrossedWithAnotherSource(t *testing.T) {
	contract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Observation.Profile.Sources = []*umpirespb.EvidenceSourceDeclaration{
			{SourceDefinitionId: "evidence.source.other"},
			{SourceDefinitionId: "evidence.source.runtime"},
		}
	})
	evidence := testRawEvidence(t, contract)
	evidence.Sources = []artifactv2.RawEvidenceSource{
		{
			SourceDefinitionID: "evidence.source.other", Status: "closed",
			FactCount: artifactv2.NaturalFromUint64(1), ByteCount: artifactv2.NaturalFromUint64(0),
		},
		{
			SourceDefinitionID: "evidence.source.runtime", Status: "closed",
			FactCount: artifactv2.NaturalFromUint64(1), ByteCount: artifactv2.NaturalFromUint64(0),
		},
	}
	evidence.Facts[0].SourceDefinitionID = "evidence.source.other"
	evidence.Facts[1].Ordinal = artifactv2.NaturalFromUint64(0)
	evidence = resealRawEvidence(t, evidence)

	result := Evaluate(context.Background(), requestFor(contract, evidence))

	require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_CONFLICT, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_SOURCE_IDENTITY,
		result.GetObservation().GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.EVALUATION_STATUS_INCOMPLETE, result.GetSemanticStatus())
	require.Empty(t, result.GetProperties())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateRejectsMalformedAndMisorderedEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*artifactv2.RawEvidence)
	}{
		{
			name: "malformed causal list",
			mutate: func(evidence *artifactv2.RawEvidence) {
				evidence.Facts[0].CausalFactDefinitionIDs = nil
			},
		},
		{
			name: "source order",
			mutate: func(evidence *artifactv2.RawEvidence) {
				evidence.Facts[0], evidence.Facts[1] = evidence.Facts[1], evidence.Facts[0]
			},
		},
		{
			name: "causal order",
			mutate: func(evidence *artifactv2.RawEvidence) {
				evidence.Facts[1].CausalFactDefinitionIDs = []string{"evidence.fact.dangling"}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest(t)
			test.mutate(&request.RawEvidence)
			request.RawEvidence = sealRawEvidenceUnchecked(t, request.RawEvidence)

			result := Evaluate(context.Background(), request)

			require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_INPUT, result.GetToolingStatus())
			require.Equal(t, umpirespb.DIAGNOSTIC_CLASS_INVALID, result.GetDiagnostics()[0].GetDiagnosticClass())
			require.Equal(t, umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, result.GetDiagnostics()[0].GetCode())
			require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
		})
	}
}

func TestEvaluateRejectsContractMutation(t *testing.T) {
	request := testRequest(t)
	request.Contract.ContractId = "test.contract.mutated"

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_CONTRACT, result.GetToolingStatus())
	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateRetainsRawEvidenceKnownGaps(t *testing.T) {
	request := testRequest(t)
	subject := "evidence.source.runtime"
	detail := "collector reported a bounded omission"
	request.RawEvidence.KnownGaps = []artifactv2.KnownGap{{
		Kind: "input", Code: "evidence.gap.bounded-omission", Subject: &subject, Detail: &detail,
	}}
	request.RawEvidence = resealRawEvidence(t, request.RawEvidence)

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.OBSERVATION_STATUS_UNKNOWN, result.GetObservation().GetStatus())
	require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
		result.GetImplementationLink().GetStatus())
	require.Equal(t, umpirespb.EVALUATION_STATUS_INCOMPLETE, result.GetSemanticStatus())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
	protorequire.ProtoEqual(t, &umpirespb.KnownGap{
		Kind: umpirespb.KNOWN_GAP_KIND_INPUT, Code: "evidence.gap.bounded-omission",
		Subject: subject, Detail: detail,
	}, result.GetKnownGaps()[0])
}

func TestEvaluateRejectsInvalidRawEvidenceChecksum(t *testing.T) {
	request := testRequest(t)
	request.RawEvidence.ArtifactChecksum = fixtureDigest

	result := Evaluate(context.Background(), request)

	require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_INPUT, result.GetToolingStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH, result.GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateRejectsResultLimitBelowMinimum(t *testing.T) {
	contract := testContractWith(t, func(contract *umpirespb.EvaluationContract) {
		contract.Limits.MaxDiagnosticBytes = 1
		contract.Limits.MaxResultBytes = 1
	})

	result := Evaluate(context.Background(), requestFor(contract, testRawEvidence(t, contract)))

	require.Equal(t, umpirespb.TOOLING_STATUS_INVALID_CONTRACT, result.GetToolingStatus())
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_MALFORMED_CONTRACT, result.GetDiagnostics()[0].GetCode())
	require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
}

func TestEvaluateRequiresEveryIndependentSuccessStatus(t *testing.T) {
	tests := []struct {
		name        string
		operational umpirespb.OperationalStatus
		cleanup     umpirespb.CleanupStatus
	}{
		{name: "operational incomplete", operational: umpirespb.OPERATIONAL_STATUS_INCOMPLETE, cleanup: umpirespb.CLEANUP_STATUS_COMPLETE},
		{name: "operational failed", operational: umpirespb.OPERATIONAL_STATUS_FAILED, cleanup: umpirespb.CLEANUP_STATUS_COMPLETE},
		{name: "cleanup incomplete", operational: umpirespb.OPERATIONAL_STATUS_SUCCEEDED, cleanup: umpirespb.CLEANUP_STATUS_INCOMPLETE},
		{name: "cleanup failed", operational: umpirespb.OPERATIONAL_STATUS_SUCCEEDED, cleanup: umpirespb.CLEANUP_STATUS_FAILED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := testRequest(t)
			request.OperationalStatus = test.operational
			request.CleanupStatus = test.cleanup

			result := Evaluate(context.Background(), request)

			require.Equal(t, umpirespb.EVALUATION_STATUS_SATISFIED, result.GetSemanticStatus())
			require.Equal(t, umpirespb.CANARY_DECISION_INCONCLUSIVE, result.GetDecision())
		})
	}
}

func TestObservationExpressionOperators(t *testing.T) {
	eventField := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.event",
	}
	countField := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.count",
	}
	record := &normalizedRecord{
		fact: artifactv2.RawEvidenceFact{
			FactDefinitionID: "evidence.fact", KindDefinitionID: "evidence.kind.event",
		},
		fields: []*normalizedField{
			{reference: eventField, value: textValue("finish")},
			{reference: countField, value: naturalValue("1")},
		},
	}
	tests := []struct {
		name       string
		expression *umpirespb.ObservationExpression
		want       *umpirespb.Value
	}{
		{name: "literal text", expression: literalText("finish"), want: textValue("finish")},
		{name: "literal natural", expression: literalNatural("1"), want: naturalValue("1")},
		{name: "field", expression: fieldExpression(countField), want: naturalValue("1")},
		{name: "natural render v1", expression: naturalRender(fieldExpression(countField)), want: textValue("1")},
		{name: "present true", expression: present(fieldExpression(eventField)), want: boolValue(true)},
		{name: "present false", expression: present(fieldExpression(&umpirespb.EvidenceFieldReference{
			KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.missing",
		})), want: boolValue(false)},
		{name: "equals true", expression: equals(fieldExpression(eventField), literalText("finish")), want: boolValue(true)},
		{name: "equals false", expression: equals(fieldExpression(eventField), literalText("other")), want: boolValue(false)},
		{name: "all false", expression: all(
			equals(fieldExpression(eventField), literalText("finish")),
			equals(fieldExpression(countField), literalNatural("2")),
		), want: boolValue(false)},
		{name: "any true", expression: anyExpression(
			equals(fieldExpression(eventField), literalText("other")),
			equals(fieldExpression(countField), literalNatural("1")),
		), want: boolValue(true)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interpreter := expressionTestInterpreter()

			got, failure := interpreter.evaluateExpression(test.expression, record)

			require.Nil(t, failure)
			protorequire.ProtoEqual(t, test.want, got)
		})
	}
}

func TestObservationExpressionFailures(t *testing.T) {
	eventField := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.event",
	}
	missingField := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.missing",
	}
	record := &normalizedRecord{
		fact: artifactv2.RawEvidenceFact{
			FactDefinitionID: "evidence.fact", KindDefinitionID: "evidence.kind.event",
		},
		fields: []*normalizedField{{reference: eventField, value: textValue("finish")}},
	}
	tests := []struct {
		name       string
		expression *umpirespb.ObservationExpression
		class      umpirespb.DiagnosticClass
		code       umpirespb.DiagnosticCode
	}{
		{name: "field missing", expression: fieldExpression(missingField), class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_FIELD},
		{name: "natural render type", expression: naturalRender(literalText("1")), class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH},
		{name: "equals type", expression: equals(literalText("1"), literalNatural("1")), class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH},
		{name: "all type", expression: all(literalText("true")), class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH},
		{name: "any type", expression: anyExpression(literalNatural("1")), class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH},
		{name: "unspecified operator", expression: &umpirespb.ObservationExpression{}, class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_UNSUPPORTED_OPERATOR},
		{name: "all validates after false", expression: all(
			equals(literalText("a"), literalText("b")), fieldExpression(missingField),
		), class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_FIELD},
		{name: "any validates after true", expression: anyExpression(
			equals(literalText("a"), literalText("a")), fieldExpression(missingField),
		), class: umpirespb.DIAGNOSTIC_CLASS_UNKNOWN, code: umpirespb.DIAGNOSTIC_CODE_MISSING_FIELD},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			interpreter := expressionTestInterpreter()

			_, failure := interpreter.evaluateExpression(test.expression, record)

			require.NotNil(t, failure)
			require.Equal(t, test.class, failure.class)
			require.Equal(t, test.code, failure.code)
		})
	}
}

func TestPresentRetainsConflictAndUnsupported(t *testing.T) {
	field := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.event",
	}
	tests := []struct {
		name   string
		fields []*normalizedField
		class  umpirespb.DiagnosticClass
		code   umpirespb.DiagnosticCode
	}{
		{
			name: "duplicate", fields: []*normalizedField{
				{reference: field, value: textValue("first")},
				{reference: field, value: textValue("second")},
			},
			class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT, code: umpirespb.DIAGNOSTIC_CODE_DUPLICATE_FIELD,
		},
		{
			name: "unexposed", fields: []*normalizedField{{reference: field}},
			class: umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, code: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := &normalizedRecord{
				fact: artifactv2.RawEvidenceFact{
					FactDefinitionID: "evidence.fact", KindDefinitionID: "evidence.kind.event",
				},
				fields: test.fields,
			}

			_, failure := expressionTestInterpreter().evaluateExpression(present(fieldExpression(field)), record)

			require.NotNil(t, failure)
			require.Equal(t, test.class, failure.class)
			require.Equal(t, test.code, failure.code)
		})
	}
}

func TestNormalizeFieldDispositionMatrix(t *testing.T) {
	fact := artifactv2.RawEvidenceFact{
		FactDefinitionID: "evidence.fact", KindDefinitionID: "evidence.kind.event",
	}
	tests := []struct {
		name      string
		raw       artifactv2.RawEvidenceField
		kind      umpirespb.FieldDispositionKind
		wantValue *umpirespb.Value
		wantToken string
		wantCode  umpirespb.DiagnosticCode
	}{
		{
			name: "retain", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "plain", Value: "visible",
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_RETAIN, wantValue: textValue("visible"),
		},
		{
			name: "retain rejects hidden", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "redacted",
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_RETAIN, wantCode: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
		},
		{
			name: "redact", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "redacted",
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_REDACT,
		},
		{
			name: "redact rejects exposed", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "plain", Value: "secret",
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_REDACT, wantCode: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
		},
		{
			name: "hash", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "sha256", Value: fixtureDigest,
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_HASH, wantToken: fixtureDigest,
		},
		{
			name: "hash rejects malformed", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "sha256", Value: "not-a-digest",
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_HASH, wantCode: umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
		},
		{
			name: "reject", raw: artifactv2.RawEvidenceField{
				FieldDefinitionID: "evidence.field", Disposition: "plain", Value: "forbidden",
			},
			kind: umpirespb.FIELD_DISPOSITION_KIND_REJECT, wantCode: umpirespb.DIAGNOSTIC_CODE_UNDECLARED_FIELD,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declaration := &umpirespb.EvidenceFieldDeclaration{
				FieldDefinitionId: "evidence.field", ValueKind: umpirespb.VALUE_KIND_TEXT,
				Disposition: test.kind,
			}

			field, failure := expressionTestInterpreter().normalizeField(fact, test.raw, declaration)

			if test.wantCode != umpirespb.DIAGNOSTIC_CODE_UNSPECIFIED {
				require.Nil(t, field)
				require.NotNil(t, failure)
				require.Equal(t, umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, failure.class)
				require.Equal(t, test.wantCode, failure.code)
				return
			}
			require.Nil(t, failure)
			require.True(t, proto.Equal(test.wantValue, field.value))
			require.Equal(t, test.wantToken, field.digestToken)
		})
	}
}

func TestSelectEmissionRejectsDuplicateAndContradictoryValues(t *testing.T) {
	emit := testEmit(
		"emit.state", "system.state", umpirespb.DEFINITION_KIND_STATE,
		initialState(), literalText("unused"), literalText("unused"),
	)
	tests := []struct {
		name   string
		values []*umpirespb.ModelValue
		code   umpirespb.DiagnosticCode
	}{
		{
			name: "duplicate", values: []*umpirespb.ModelValue{
				testModelValue("system.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
				testModelValue("system.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
			},
			code: umpirespb.DIAGNOSTIC_CODE_DUPLICATE_COORDINATE,
		},
		{
			name: "contradictory", values: []*umpirespb.ModelValue{
				testModelValue("system.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
				testModelValue("system.state", umpirespb.DEFINITION_KIND_STATE, textValue("closed")),
			},
			code: umpirespb.DIAGNOSTIC_CODE_CONTRADICTORY_COORDINATE,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidates := make([]*emission, len(test.values))
			for index, value := range test.values {
				candidates[index] = &emission{emit: emit, value: value}
			}

			_, failure := selectEmission(emit, candidates)

			require.NotNil(t, failure)
			require.Equal(t, umpirespb.DIAGNOSTIC_CLASS_CONFLICT, failure.class)
			require.Equal(t, test.code, failure.code)
		})
	}
}

func TestApplyLinkRejectsDuplicateAndContradictoryMappings(t *testing.T) {
	source := testModelValue("system.state", umpirespb.DEFINITION_KIND_STATE, textValue("open"))
	coordinate := initialState()
	tests := []struct {
		name         string
		destinations []*umpirespb.ModelValue
		code         umpirespb.DiagnosticCode
	}{
		{
			name: "duplicate", destinations: []*umpirespb.ModelValue{
				testModelValue("feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
				testModelValue("feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
			},
			code: umpirespb.DIAGNOSTIC_CODE_DUPLICATE_LINK_MAPPING,
		},
		{
			name: "contradictory", destinations: []*umpirespb.ModelValue{
				testModelValue("feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
				testModelValue("feature.other-state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
			},
			code: umpirespb.DIAGNOSTIC_CODE_CONTRADICTORY_LINK_MAPPING,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries := make([]*umpirespb.RenameExactEntry, len(test.destinations))
			for index, destination := range test.destinations {
				entries[index] = &umpirespb.RenameExactEntry{
					Source: proto.CloneOf(source), Destination: destination,
				}
			}
			interpreter := newInterpreter(context.Background(), Request{})
			interpreter.contract = &umpirespb.EvaluationContract{
				Limits: &umpirespb.EvaluationLimits{MaxResultBytes: 1 << 20},
				ImplementationLink: &umpirespb.RenameExactLink{
					Entries: entries, ApplicationLimit: &umpirespb.Limit{Value: 1, Unit: "semantic-transitions"},
				},
			}
			interpreter.work.limit = 10

			_, failure := interpreter.applyLink(
				&umpirespb.ModelTrace{InitialState: source},
				[]*umpirespb.EvidenceLink{{Coordinate: coordinate}},
			)

			require.NotNil(t, failure)
			require.Equal(t, umpirespb.DIAGNOSTIC_CLASS_CONFLICT, failure.class)
			require.Equal(t, test.code, failure.code)
		})
	}
}

func TestImplementationLinkApplicationLimitExactBoundary(t *testing.T) {
	limit := &umpirespb.Limit{Value: 1, Unit: "semantic-transitions"}

	require.Nil(t, validateApplicationLimit(&umpirespb.ModelTrace{
		Steps: []*umpirespb.ModelTraceStep{{Position: 1}},
	}, limit))
	failure := validateApplicationLimit(&umpirespb.ModelTrace{
		Steps: []*umpirespb.ModelTraceStep{{Position: 1}, {Position: 2}},
	}, limit)
	require.NotNil(t, failure)
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED, failure.code)
	require.Equal(t, int64(1), failure.limit.GetValue())
	require.Equal(t, int64(2), failure.observed)
}

func TestEvidenceLinkPreflightsResultGrowth(t *testing.T) {
	request := testRequest(t)
	interpreter := newInterpreter(context.Background(), request)
	interpreter.contract = &umpirespb.EvaluationContract{
		Limits:      &umpirespb.EvaluationLimits{MaxResultBytes: 1},
		Observation: &umpirespb.ObservationProgram{Mapping: testBinding("test.mapping")},
	}
	candidate := &emission{
		emit:   &umpirespb.Emit{DefinitionId: "emit.value", Coordinate: initialState()},
		record: &normalizedRecord{fact: request.RawEvidence.Facts[0]},
	}

	link, failure := interpreter.evidenceLink(candidate)

	require.Nil(t, link)
	require.NotNil(t, failure)
	require.Equal(t, umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED, failure.code)
	require.Zero(t, interpreter.resultBytesReserved)
	require.Len(t, interpreter.orderingSupport, len(request.RawEvidence.Facts))
	require.Len(t, interpreter.closureSupport, len(request.RawEvidence.Sources))
}

func expressionTestInterpreter() *interpreter {
	return &interpreter{
		ctx:      context.Background(),
		contract: &umpirespb.EvaluationContract{Limits: &umpirespb.EvaluationLimits{MaxNatural: "100"}},
		work:     workTracker{limit: 100},
	}
}

func FuzzEvaluateFailsClosed(f *testing.F) {
	contract := testContract(f)
	f.Add([]byte{0, 0})
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, mutation []byte) {
		evidence := testRawEvidence(t, contract)
		if len(mutation) > 0 {
			evidence.Facts[0].Fields[1].Value = fmt.Sprintf("mutated-%x", mutation)
		}
		if len(mutation) > 1 {
			evidence.Facts[1].Fields[0].Value = fmt.Sprintf("%d", mutation[1])
		}
		evidence = resealRawEvidence(t, evidence)

		result := Evaluate(context.Background(), requestFor(contract, evidence))

		if result.GetDecision() == umpirespb.CANARY_DECISION_PASS {
			require.Equal(t, umpirespb.TOOLING_STATUS_SUCCEEDED, result.GetToolingStatus())
			require.Equal(t, umpirespb.OBSERVATION_STATUS_ACCEPTED, result.GetObservation().GetStatus())
			require.Equal(t, umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED, result.GetImplementationLink().GetStatus())
			require.Equal(t, umpirespb.EVALUATION_STATUS_SATISFIED, result.GetSemanticStatus())
			require.Equal(t, umpirespb.CLEANUP_STATUS_COMPLETE, result.GetCleanupStatus())
		}
	})
}

func resealRawEvidence(t testing.TB, evidence artifactv2.RawEvidence) artifactv2.RawEvidence {
	t.Helper()
	sealed, err := artifactv2.SealRawEvidence(evidence)
	require.NoError(t, err)
	require.NoError(t, artifactv2.ValidateRawEvidence(sealed))
	return sealed
}

func sealRawEvidenceUnchecked(t testing.TB, evidence artifactv2.RawEvidence) artifactv2.RawEvidence {
	t.Helper()
	sealed, err := artifactv2.SealRawEvidence(evidence)
	require.NoError(t, err)
	return sealed
}

func allDiagnostics(result *umpirespb.EvaluationResult) []umpirespb.DiagnosticCode {
	var codes []umpirespb.DiagnosticCode
	for _, diagnostic := range result.GetDiagnostics() {
		codes = append(codes, diagnostic.GetCode())
	}
	for _, diagnostic := range result.GetObservation().GetDiagnostics() {
		codes = append(codes, diagnostic.GetCode())
	}
	for _, diagnostic := range result.GetImplementationLink().GetDiagnostics() {
		codes = append(codes, diagnostic.GetCode())
	}
	for _, property := range result.GetProperties() {
		for _, diagnostic := range property.GetDiagnostics() {
			codes = append(codes, diagnostic.GetCode())
		}
	}
	return codes
}

func withoutTraceID(trace *umpirespb.ModelTrace) *umpirespb.ModelTrace {
	if trace == nil {
		return nil
	}
	cloned := proto.CloneOf(trace)
	cloned.TraceId = ""
	return cloned
}

func testRequest(t testing.TB) Request {
	t.Helper()
	contract := testContract(t)
	evidence := testRawEvidence(t, contract)
	return requestFor(contract, evidence)
}

func requestFor(contract *umpirespb.EvaluationContract, evidence artifactv2.RawEvidence) Request {
	closures := make([]artifactv2.SourceClosure, len(evidence.Sources))
	for index, source := range evidence.Sources {
		closures[index] = artifactv2.SourceClosure{
			SourceDefinitionID: source.SourceDefinitionID, Status: source.Status,
			RecordCount: source.FactCount, ByteCount: source.ByteCount,
		}
	}
	return Request{
		Contract: contract, RawEvidence: evidence, ExpectedClosures: closures,
		ExpectedRunIdentity: evidence.RunIdentity, ExpectedRun: evidence.Run,
		OperationalStatus: umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		CleanupStatus:     umpirespb.CLEANUP_STATUS_COMPLETE,
	}
}

func testContract(t testing.TB) *umpirespb.EvaluationContract {
	return testContractWith(t, nil)
}

func testContractWith(t testing.TB, modify func(*umpirespb.EvaluationContract)) *umpirespb.EvaluationContract {
	t.Helper()
	fieldCount := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.count",
	}
	fieldEvent := &umpirespb.EvidenceFieldReference{
		KindDefinitionId: "evidence.kind.event", FieldDefinitionId: "evidence.field.event",
	}
	finish := equals(fieldExpression(fieldEvent), literalText("finish"))
	contract := &umpirespb.EvaluationContract{
		Version:       &umpirespb.FormatVersion{Major: 1},
		ContractId:    "test.contract.portable-evaluation",
		Experiment:    testArtifactBinding(artifactv2.ExperimentFormat),
		RuntimeConfig: testArtifactBinding(artifactv2.RuntimeConfigurationFormat),
		Test:          testBinding("test.definition.test"),
		Query:         testBinding("test.definition.query"),
		Limits: &umpirespb.EvaluationLimits{
			MaxContractBytes: 1 << 20, MaxInputBytes: 1 << 20, MaxEvidenceRecords: 16,
			MaxExpressionDepth: 16, MaxCollectionItems: 64, MaxOperatorCount: 100, MaxNatural: "100",
			MaxEvaluationWork: 1000, MaxDiagnosticBytes: 4096, MaxResultBytes: 1 << 20,
			MaxTotalDurationMilliseconds: 10000,
		},
		Observation: &umpirespb.ObservationProgram{
			Definition:     testBinding("test.observation.program"),
			Source:         testLocation(),
			Mapping:        testBinding("test.observation.mapping"),
			MappingVersion: 1,
			Profile: &umpirespb.EvidenceProfile{
				Definition: testBinding("test.evidence.profile"), Version: 1,
				Sources: []*umpirespb.EvidenceSourceDeclaration{{SourceDefinitionId: "evidence.source.runtime"}},
				Kinds: []*umpirespb.EvidenceKindDeclaration{{
					KindDefinitionId: "evidence.kind.event", SourceDefinitionId: "evidence.source.runtime",
					Fields: []*umpirespb.EvidenceFieldDeclaration{
						{FieldDefinitionId: "evidence.field.count", ValueKind: umpirespb.VALUE_KIND_NATURAL, Disposition: umpirespb.FIELD_DISPOSITION_KIND_RETAIN},
						{FieldDefinitionId: "evidence.field.event", ValueKind: umpirespb.VALUE_KIND_TEXT, Disposition: umpirespb.FIELD_DISPOSITION_KIND_RETAIN},
					},
				}},
				Cardinalities: []*umpirespb.EvidenceCardinality{{KindDefinitionId: "evidence.kind.event", Minimum: 2, Maximum: 2}},
			},
			Emits: []*umpirespb.Emit{
				testEmit("emit.action", "system.action", umpirespb.DEFINITION_KIND_ACTION, selectedAction(1),
					all(present(fieldExpression(fieldEvent)), finish), literalText("close")),
				testEmit("emit.initial", "system.state", umpirespb.DEFINITION_KIND_STATE, initialState(),
					equals(fieldExpression(fieldEvent), literalText("start")), literalText("open")),
				testEmit("emit.observation-count", "system.observation.count", umpirespb.DEFINITION_KIND_OBSERVATION,
					observation(1, 1), equals(fieldExpression(fieldCount), literalNatural("1")), fieldExpression(fieldCount)),
				testEmit("emit.observation-rendered", "system.observation.rendered", umpirespb.DEFINITION_KIND_OBSERVATION,
					observation(1, 2), finish, naturalRender(fieldExpression(fieldCount))),
				testEmit("emit.outcome", "system.outcome", umpirespb.DEFINITION_KIND_OUTCOME, modelOutcome(1),
					anyExpression(equals(fieldExpression(fieldEvent), literalText("other")), finish), literalText("done")),
				testEmit("emit.resulting", "system.state", umpirespb.DEFINITION_KIND_STATE, resultingState(1),
					finish, literalText("closed")),
			},
			Ordering: []*umpirespb.EmitOrdering{
				{PredecessorEmitDefinitionId: "emit.action", SuccessorEmitDefinitionId: "emit.outcome"},
				{PredecessorEmitDefinitionId: "emit.initial", SuccessorEmitDefinitionId: "emit.action"},
			},
		},
		ImplementationLink: &umpirespb.RenameExactLink{
			Definition: testBinding("test.implementation-link"), Source: testLocation(),
			SourceTarget: testBinding("system.target"), DestinationTarget: testBinding("feature.target"),
			ApplicationLimit: &umpirespb.Limit{Value: 16, Unit: "semantic-transitions"},
			Entries: []*umpirespb.RenameExactEntry{
				testRename("system.action", "feature.action", umpirespb.DEFINITION_KIND_ACTION, textValue("close")),
				testRename("system.observation.count", "feature.observation.count", umpirespb.DEFINITION_KIND_OBSERVATION, naturalValue("1")),
				testRename("system.observation.rendered", "feature.observation.rendered", umpirespb.DEFINITION_KIND_OBSERVATION, textValue("1")),
				testRename("system.outcome", "feature.outcome", umpirespb.DEFINITION_KIND_OUTCOME, textValue("done")),
				testRename("system.state", "feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("closed")),
				testRename("system.state", "feature.state", umpirespb.DEFINITION_KIND_STATE, textValue("open")),
			},
		},
		Properties: []*umpirespb.Property{{
			Definition: testBinding("feature.property"), Source: testLocation(),
			Clauses: []*umpirespb.PropertyClause{
				testClause("feature.property.clause.count", selectedActionPattern("feature.action", "close"),
					naturalPattern("feature.observation.count", "1")),
				testClause("feature.property.clause.rendered", selectedActionPattern("feature.action", "close"),
					textPattern(umpirespb.TRACE_FIELD_OBSERVATION, "feature.observation.rendered", "1")),
			},
		}},
		KnownGaps:  []*umpirespb.KnownGap{},
		Provenance: []*umpirespb.SourceLocation{testLocation()},
	}
	if modify != nil {
		modify(contract)
	}
	encodedJSON, err := evaluationcontract.CanonicalProtoJSON(contract)
	require.NoError(t, err)
	encoded, err := evaluationcontract.Pack(encodedJSON)
	require.NoError(t, err)
	admitted, err := evaluationcontract.Admit(encoded)
	require.NoError(t, err)
	return admitted
}

func testRawEvidence(t testing.TB, contract *umpirespb.EvaluationContract) artifactv2.RawEvidence {
	t.Helper()
	evidence := artifactv2.RawEvidence{
		FormatVersion: artifactv2.RawEvidenceFormat,
		RunIdentity:   "test.run.fixture", BehaviorFingerprint: fixtureDigest,
		Experiment: artifactv2.ArtifactBinding{
			FormatVersion:       contract.GetExperiment().GetFormatVersion(),
			ArtifactChecksum:    contract.GetExperiment().GetArtifactChecksum(),
			BehaviorFingerprint: contract.GetExperiment().GetBehaviorFingerprint(),
			ProvenanceChecksum:  contract.GetExperiment().GetProvenanceChecksum(),
		},
		RuntimeConfiguration: artifactv2.ArtifactBinding{
			FormatVersion:       contract.GetRuntimeConfig().GetFormatVersion(),
			ArtifactChecksum:    contract.GetRuntimeConfig().GetArtifactChecksum(),
			BehaviorFingerprint: contract.GetRuntimeConfig().GetBehaviorFingerprint(),
			ProvenanceChecksum:  contract.GetRuntimeConfig().GetProvenanceChecksum(),
		},
		Run: artifactv2.ArtifactBinding{
			FormatVersion: artifactv2.ExperimentRunFormat, ArtifactChecksum: fixtureDigest,
			BehaviorFingerprint: fixtureDigest, ProvenanceChecksum: fixtureDigest,
		},
		CaptureStatus: "closed",
		Sources: []artifactv2.RawEvidenceSource{{
			SourceDefinitionID: "evidence.source.runtime", Status: "closed",
			FactCount: artifactv2.NaturalFromUint64(2), ByteCount: artifactv2.NaturalFromUint64(0),
		}},
		Facts: []artifactv2.RawEvidenceFact{
			{
				FactDefinitionID: "evidence.fact.start", SourceDefinitionID: "evidence.source.runtime",
				Ordinal: artifactv2.NaturalFromUint64(0), KindDefinitionID: "evidence.kind.event",
				CausalFactDefinitionIDs: []string{},
				Fields: []artifactv2.RawEvidenceField{
					{FieldDefinitionID: "evidence.field.count", Disposition: "plain", Value: json.Number("0")},
					{FieldDefinitionID: "evidence.field.event", Disposition: "plain", Value: "start"},
				},
			},
			{
				FactDefinitionID: "evidence.fact.finish", SourceDefinitionID: "evidence.source.runtime",
				Ordinal: artifactv2.NaturalFromUint64(1), KindDefinitionID: "evidence.kind.event",
				CausalFactDefinitionIDs: []string{"evidence.fact.start"},
				Fields: []artifactv2.RawEvidenceField{
					{FieldDefinitionID: "evidence.field.count", Disposition: "plain", Value: json.Number("1")},
					{FieldDefinitionID: "evidence.field.event", Disposition: "plain", Value: "finish"},
				},
			},
		},
		KnownGaps: []artifactv2.KnownGap{},
		Provenance: artifactv2.Provenance{
			SourceDefinitionIDs: []string{},
			SourceLocations: []artifactv2.SourceLocation{{
				Path: "portableevaluation/fixture", Line: artifactv2.NaturalFromUint64(1),
				Column: artifactv2.NaturalFromUint64(1), Provenance: "generated",
			}},
		},
	}
	sealed, err := artifactv2.SealRawEvidence(evidence)
	require.NoError(t, err)
	require.NoError(t, artifactv2.ValidateRawEvidence(sealed))
	return sealed
}

func testArtifactBinding(format string) *umpirespb.ArtifactBinding {
	return &umpirespb.ArtifactBinding{
		FormatVersion: format, ArtifactChecksum: fixtureDigest,
		BehaviorFingerprint: fixtureDigest, ProvenanceChecksum: fixtureDigest,
	}
}

func testBinding(id string) *umpirespb.DefinitionBinding {
	return &umpirespb.DefinitionBinding{DefinitionId: id, BehaviorFingerprint: fixtureDigest}
}

func testLocation() *umpirespb.SourceLocation {
	return &umpirespb.SourceLocation{Path: "portableevaluation/fixture", Line: 1, Column: 1, Provenance: "lean-model"}
}

func testEmit(id, output string, kind umpirespb.DefinitionKind, coordinate *umpirespb.ModelCoordinate, condition, value *umpirespb.ObservationExpression) *umpirespb.Emit {
	return &umpirespb.Emit{
		DefinitionId: id, SourceKindDefinitionId: "evidence.kind.event",
		OutputDefinition: testBinding(output), OutputKind: kind, Coordinate: coordinate,
		Condition: condition, Value: value,
	}
}

func testRename(source, destination string, kind umpirespb.DefinitionKind, value *umpirespb.Value) *umpirespb.RenameExactEntry {
	return &umpirespb.RenameExactEntry{
		Source: testModelValue(source, kind, value), Destination: testModelValue(destination, kind, value),
	}
}

func declareMissingObservationVocabulary(contract *umpirespb.EvaluationContract, value *umpirespb.Value) {
	entry := testRename(
		"system.observation.missing", "feature.observation.missing",
		umpirespb.DEFINITION_KIND_OBSERVATION, value,
	)
	entries := contract.GetImplementationLink().GetEntries()
	contract.ImplementationLink.Entries = append(entries, nil)
	copy(contract.ImplementationLink.Entries[3:], contract.ImplementationLink.Entries[2:])
	contract.ImplementationLink.Entries[2] = entry
}

func testModelValue(id string, kind umpirespb.DefinitionKind, value *umpirespb.Value) *umpirespb.ModelValue {
	return &umpirespb.ModelValue{Definition: testBinding(id), Kind: kind, Value: value}
}

func testClause(id string, trigger, required *umpirespb.Pattern) *umpirespb.PropertyClause {
	return &umpirespb.PropertyClause{
		DefinitionId: id, Provenance: umpirespb.PROPERTY_CLAUSE_PROVENANCE_TRANSITION_CONTRACT,
		PerStepImplies: &umpirespb.PerStepImplies{Trigger: trigger, Required: required},
	}
}

func selectedActionPattern(id, value string) *umpirespb.Pattern {
	return textPattern(umpirespb.TRACE_FIELD_SELECTED_ACTION, id, value)
}

func textPattern(field umpirespb.TraceField, id, value string) *umpirespb.Pattern {
	return &umpirespb.Pattern{
		Field: field, Definition: testBinding(id),
		Operator: &umpirespb.Pattern_EqualsText{EqualsText: &umpirespb.EqualsText{Value: value}},
	}
}

func naturalPattern(id, bound string) *umpirespb.Pattern {
	return &umpirespb.Pattern{
		Field: umpirespb.TRACE_FIELD_OBSERVATION, Definition: testBinding(id),
		Operator: &umpirespb.Pattern_NaturalAtMost{NaturalAtMost: &umpirespb.NaturalAtMost{Bound: bound}},
	}
}

func literalText(value string) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_LiteralText{LiteralText: &umpirespb.LiteralText{Value: value}}}
}

func literalNatural(value string) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_LiteralNatural{LiteralNatural: &umpirespb.LiteralNatural{Value: value}}}
}

func fieldExpression(field *umpirespb.EvidenceFieldReference) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Field{Field: field}}
}

func naturalRender(operand *umpirespb.ObservationExpression) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_NaturalRenderV1{NaturalRenderV1: &umpirespb.NaturalRenderV1{Operand: operand}}}
}

func present(operand *umpirespb.ObservationExpression) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Present{Present: &umpirespb.Present{Operand: operand}}}
}

func equals(left, right *umpirespb.ObservationExpression) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Equals{Equals: &umpirespb.Equals{Left: left, Right: right}}}
}

func all(operands ...*umpirespb.ObservationExpression) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_All{All: &umpirespb.All{Operands: operands}}}
}

func anyExpression(operands ...*umpirespb.ObservationExpression) *umpirespb.ObservationExpression {
	return &umpirespb.ObservationExpression{Operator: &umpirespb.ObservationExpression_Any{Any: &umpirespb.Any{Operands: operands}}}
}

func textValue(value string) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_Text{Text: value}}
}

func naturalValue(value string) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: value}}
}

func boolValue(value bool) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: value}}
}

func initialState() *umpirespb.ModelCoordinate {
	return &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_INITIAL_STATE}
}

func selectedAction(step int64) *umpirespb.ModelCoordinate {
	return &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_SELECTED_ACTION, Step: step}
}

func modelOutcome(step int64) *umpirespb.ModelCoordinate {
	return &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_MODEL_OUTCOME, Step: step}
}

func resultingState(step int64) *umpirespb.ModelCoordinate {
	return &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_RESULTING_STATE, Step: step}
}

func observation(step, position int64) *umpirespb.ModelCoordinate {
	return &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_OBSERVATION, Step: step, Position: position}
}
