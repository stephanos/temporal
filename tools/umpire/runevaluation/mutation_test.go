package runevaluation

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestRawArtifactMutationFailsAtAdmission(t *testing.T) {
	experiment := append(readCallerClosureInput(t, "experiment.json"), ' ')
	_, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experiment},
		{Path: "artifacts/runtime-configuration.json",
			Encoded: readCallerClosureInput(t, "runtime-configuration.json")},
	})
	require.Error(t, err)
	code, ok := artifact.CodeOf(err)
	require.True(t, ok)
	require.Equal(t, artifact.ErrorNoncanonical, code)
}

func TestCheckerRequestSeparatesRuntimeAndCheckedMappings(t *testing.T) {
	input := callerClosureExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)

	require.Equal(t, "temporal.nexus.synthetic.basic-lifecycle.mapping",
		execution.RuntimeConfiguration().Observation.MappingDefinitionID)
	require.Equal(t, "sha256:608e4db6c3a29d0f953640621ee34d34e16b0090309e85804e21f0cb21be30a2",
		execution.RuntimeConfiguration().Observation.MappingBehaviorFingerprint)
	require.Equal(t, "temporal.system.nexus.caller-closure.mapping", request.Mapping.DefinitionID)
	require.Equal(t, "sha256:150c75ffcdd8b8e6e2ca8807c2c6ac7d924407b3291a0bc1f10ea04469a7df9b",
		request.Mapping.BehaviorFingerprint)
}

func TestCheckerResponseRejectsConsistentCheckedProfileDriftAtTheProtocolBoundary(t *testing.T) {
	input := acceptedCallerClosureExecutionFixture(t, "succeeded")
	output, err := checkWithChecker(context.Background(), input,
		func(_ context.Context, request checkerRequest) (checkerResponse, error) {
			response := acceptedCallerClosureResponse(t, request, "satisfied")
			staleProfileID := "temporal.system.nexus.caller-closure.profile.stale"
			staleProfileVersion := artifactv2.NaturalFromUint64(2)
			response.EvidenceBackedModelTrace.ProfileDefinitionID = staleProfileID
			response.EvidenceBackedModelTrace.ProfileVersion = staleProfileVersion
			mutateLinks := func(links []artifactv2.EvidenceLink) {
				for index := range links {
					links[index].ProfileDefinitionID = staleProfileID
					links[index].ProfileVersion = staleProfileVersion
				}
			}
			mutateLinks(response.EvidenceLinks)
			for verdictIndex := range response.PropertyVerdicts {
				for clauseIndex := range response.PropertyVerdicts[verdictIndex].Clauses {
					mutateLinks(response.PropertyVerdicts[verdictIndex].Clauses[clauseIndex].EvidenceLinks)
				}
			}
			for verdictIndex := range response.QuerySummary.PropertyVerdicts {
				for clauseIndex := range response.QuerySummary.PropertyVerdicts[verdictIndex].Clauses {
					mutateLinks(response.QuerySummary.PropertyVerdicts[verdictIndex].Clauses[clauseIndex].EvidenceLinks)
				}
			}
			return response, nil
		})
	require.Error(t, err)
	require.Empty(t, output.Identity())
	var failure *evaluationFailure
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "evaluation", failure.Phase())
	require.ErrorContains(t, failure.Unwrap(), "profile binding drifted")
}

func TestRealCheckerObservationMutationMatrix(t *testing.T) {
	process := realCheckerProcess(t)
	baselineRequest := exactCallerClosureMutationRequest(t)
	baseline, err := process.run(context.Background(), baselineRequest)
	require.NoError(t, err)

	for _, testCase := range []struct {
		name           string
		mutate         func(*checkerRequest)
		status         string
		diagnosticKind string
		clearValue     string
	}{
		{
			name: "missing order",
			mutate: func(request *checkerRequest) {
				request.Facts[7].Ordinal = artifactv2.NaturalFromUint64(6)
			},
			status: "unknown", diagnosticKind: "sequence-gap",
		},
		{
			name: "conflicting duplicate",
			mutate: func(request *checkerRequest) {
				request.Facts[7] = request.Facts[6]
			},
			status: "conflict", diagnosticKind: "duplicate-evidence-identity",
		},
		{
			name: "clear endpoint",
			mutate: func(request *checkerRequest) {
				request.Facts[8].Fields[2].Disposition = "plain"
				request.Facts[8].Fields[2].Value = "clear-endpoint.example"
			},
			status: "unsupported", diagnosticKind: "digest-policy-mismatch",
			clearValue: "clear-endpoint.example",
		},
		{
			name: "redacted endpoint",
			mutate: func(request *checkerRequest) {
				request.Facts[8].Fields[2].Disposition = "redacted"
				request.Facts[8].Fields[2].Value = nil
			},
			status: "unsupported", diagnosticKind: "digest-policy-mismatch",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := exactCallerClosureMutationRequest(t)
			testCase.mutate(&request)

			stdout, stderr, runErr := runRealCheckerOutput(t, process, request)
			require.NoError(t, runErr, string(stderr))
			response, err := decodeCheckerResponse(stdout, request)
			require.NoError(t, err)
			require.Equal(t, testCase.status, response.ObservationEvaluationStatus)
			require.Equal(t, "incomplete", response.SemanticStatus)
			require.Empty(t, response.PropertyVerdicts)
			require.Nil(t, response.EvidenceBackedModelTrace)
			require.Len(t, response.Diagnostics, 1)
			require.Equal(t, testCase.diagnosticKind, response.Diagnostics[0].Kind)
			if testCase.clearValue != "" {
				encoded, err := artifact.CanonicalPretty(response)
				require.NoError(t, err)
				require.False(t, bytes.Contains(encoded, []byte(testCase.clearValue)))
			}
		})
	}

	permutedRequest := exactCallerClosureMutationRequest(t)
	slices.Reverse(permutedRequest.Facts)
	permuted, err := process.run(context.Background(), permutedRequest)
	require.NoError(t, err)
	require.Equal(t, baseline, permuted)
}

func TestRealCheckerDuplicateDeliveryMutationMatrix(t *testing.T) {
	process := realCheckerProcess(t)

	for _, testCase := range []struct {
		name           string
		mutate         func(*testing.T, *checkerRequest)
		status         string
		diagnosticKind string
	}{
		{
			name: "missing marker",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.participant.synthetic-duplicate.fixture")
				fact.Fields = slices.DeleteFunc(fact.Fields, func(field artifactv2.RawEvidenceField) bool {
					return field.FieldDefinitionID ==
						"umpire.evidence.field.synthetic-contribution-marker"
				})
			},
			status: "unknown", diagnosticKind: "unresolved-binding",
		},
		{
			name: "synthetic count two",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.participant.synthetic-duplicate.fixture")
				setMutationField(t, fact, "umpire.evidence.field.synthetic-contribution-count",
					artifactv2.NaturalFromUint64(2))
			},
			status: "conflict", diagnosticKind: "contradictory-fact",
		},
		{
			name: "correlation drift",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.participant.synthetic-duplicate.fixture")
				setMutationField(t, fact, umpireruntime.EvidenceFieldOperationCorrelationID,
					"runtime.correlation.operation.drifted")
			},
			status: "conflict", diagnosticKind: "contradictory-binding",
		},
		{
			name: "redacted marker",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.participant.synthetic-duplicate.fixture")
				for index := range fact.Fields {
					if fact.Fields[index].FieldDefinitionID ==
						"umpire.evidence.field.synthetic-contribution-marker" {
						fact.Fields[index].Disposition = "redacted"
						fact.Fields[index].Value = nil
						return
					}
				}
				require.Fail(t, "synthetic marker field not found")
			},
			status: "unsupported", diagnosticKind: "field-mismatch",
		},
		{
			name: "missing participant causal parent",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.participant.synthetic-duplicate.fixture")
				fact.CausalFactDefinitionIDs = []string{}
			},
			status: "unknown", diagnosticKind: "missing-causal-parent",
		},
		{
			name: "incompatible history order",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.history.00000000000000000004.fixture")
				fact.CausalFactDefinitionIDs = []string{
					"umpire.runtime.fact.history.00000000000000000001.fixture",
				}
			},
			status: "conflict", diagnosticKind: "contradictory-fact",
		},
		{
			name: "missing history causal parent",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.history.00000000000000000004.fixture")
				fact.CausalFactDefinitionIDs = []string{}
			},
			status: "unknown", diagnosticKind: "missing-causal-parent",
		},
		{
			name: "missing completed event",
			mutate: func(t *testing.T, request *checkerRequest) {
				fact := duplicateDeliveryFact(t, request,
					"umpire.runtime.fact.history.00000000000000000005.fixture")
				setMutationField(t, fact, umpireruntime.EvidenceFieldEventType,
					"temporal.history.NexusOperationStarted")
			},
			status: "unknown", diagnosticKind: "unresolved-binding",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			input := duplicateDeliveryExecutionFixture(t)
			execution, ok := input.Execution()
			require.True(t, ok)
			request, err := newCheckerRequest(execution)
			require.NoError(t, err)
			testCase.mutate(t, &request)

			stdout, stderr, runErr := runRealCheckerOutput(t, process, request)
			require.NoError(t, runErr, string(stderr))
			response, err := decodeCheckerResponse(stdout, request)
			require.NoError(t, err)
			require.Equal(t, testCase.status, response.ObservationEvaluationStatus)
			require.Equal(t, "incomplete", response.SemanticStatus)
			require.Empty(t, response.PropertyVerdicts)
			require.Nil(t, response.EvidenceBackedModelTrace)
			require.Len(t, response.Diagnostics, 1)
			require.Equal(t, testCase.diagnosticKind, response.Diagnostics[0].Kind)
		})
	}
}

func TestRealCheckerRejectsCrossedDuplicateDeliverySemanticClosure(t *testing.T) {
	process := realCheckerProcess(t)
	input := duplicateDeliveryExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	request.Query = definitionReference{
		DefinitionID: callerClosureQueryID, BehaviorFingerprint: callerClosureQueryFingerprint,
	}
	request.ObservationProgram = definitionReference{
		DefinitionID:        callerClosureObservationProgramID,
		BehaviorFingerprint: callerClosureObservationProgramFingerprint,
	}
	request.Mapping = definitionReference{
		DefinitionID:        callerClosureCheckedMappingID,
		BehaviorFingerprint: callerClosureCheckedMappingFingerprint,
	}

	stdout, stderr, runErr := runRealCheckerOutput(t, process, request)
	require.Error(t, runErr)
	require.Empty(t, stdout)
	require.JSONEq(t, `{"field":"semantics","kind":"closure-drift"}`, string(stderr))
}

func TestDuplicateDeliveryResponseRejectsStrictNormalSemanticBindings(t *testing.T) {
	process := realCheckerProcess(t)
	input := duplicateDeliveryExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	response, err := process.run(context.Background(), request)
	require.NoError(t, err)

	response.ImplementationLink = callerClosureImplementationLink()
	response.EvidenceBackedModelTrace.ProfileDefinitionID = callerClosureCheckedProfileID
	mutateProfile := func(links []artifactv2.EvidenceLink) {
		for index := range links {
			links[index].ProfileDefinitionID = callerClosureCheckedProfileID
		}
	}
	mutateProfile(response.EvidenceLinks)
	for verdictIndex := range response.PropertyVerdicts {
		for clauseIndex := range response.PropertyVerdicts[verdictIndex].Clauses {
			mutateProfile(response.PropertyVerdicts[verdictIndex].Clauses[clauseIndex].EvidenceLinks)
		}
	}
	for verdictIndex := range response.QuerySummary.PropertyVerdicts {
		for clauseIndex := range response.QuerySummary.PropertyVerdicts[verdictIndex].Clauses {
			mutateProfile(response.QuerySummary.PropertyVerdicts[verdictIndex].Clauses[clauseIndex].EvidenceLinks)
		}
	}

	output, err := checkWithChecker(context.Background(), input,
		func(context.Context, checkerRequest) (checkerResponse, error) {
			return response, nil
		})
	require.Error(t, err)
	require.Empty(t, output.Identity())
	var failure *evaluationFailure
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "evaluation", failure.Phase())
	require.ErrorContains(t, failure.Unwrap(), "Implementation Link binding drifted")
}

func duplicateDeliveryFact(
	t *testing.T,
	request *checkerRequest,
	definitionID string,
) *artifactv2.RawEvidenceFact {
	t.Helper()
	for index := range request.Facts {
		if request.Facts[index].FactDefinitionID == definitionID {
			return &request.Facts[index]
		}
	}
	require.Failf(t, "duplicate-delivery fact not found", "definitionID=%q", definitionID)
	return nil
}

func TestRealCheckerMisboundParticipantCancellationEvidenceIsSemanticConflict(t *testing.T) {
	process := realCheckerProcess(t)
	input := callerClosureExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)

	for _, testCase := range []struct {
		name   string
		mutate func(*testing.T, *checkerRequest)
	}{
		{
			name: "prepare command",
			mutate: func(t *testing.T, request *checkerRequest) {
				setMutationField(t, &request.Facts[8], umpireruntime.EvidenceFieldCommandKind, "prepare")
			},
		},
		{
			name: "observe command",
			mutate: func(t *testing.T, request *checkerRequest) {
				setMutationField(t, &request.Facts[8], umpireruntime.EvidenceFieldCommandKind, "observe")
			},
		},
		{
			name: "nonaccepted status",
			mutate: func(t *testing.T, request *checkerRequest) {
				setMutationField(t, &request.Facts[8], umpireruntime.EvidenceFieldStatus, "rejected")
			},
		},
		{
			name: "run correlation drift",
			mutate: func(t *testing.T, request *checkerRequest) {
				setMutationField(t, &request.Facts[8], umpireruntime.EvidenceFieldRunCorrelationID,
					"umpire.local.caller-closure.drifted")
			},
		},
		{
			name: "workflow correlation drift",
			mutate: func(t *testing.T, request *checkerRequest) {
				setMutationField(t, &request.Facts[8], umpireruntime.EvidenceFieldWorkflowCorrelationID,
					"temporal.workflow.caller-closure.drifted")
			},
		},
		{
			name: "operation correlation drift",
			mutate: func(t *testing.T, request *checkerRequest) {
				setMutationField(t, &request.Facts[8], umpireruntime.EvidenceFieldOperationCorrelationID,
					"temporal.operation.caller-closure.drifted")
			},
		},
		{
			name: "duplicate cancellation candidate",
			mutate: func(_ *testing.T, request *checkerRequest) {
				duplicate := request.Facts[8]
				duplicate.FactDefinitionID = "umpire.runtime.fact.participant.duplicate"
				duplicate.Ordinal = artifactv2.NaturalFromUint64(1)
				request.Facts = append(request.Facts, duplicate)
				request.Sources[3].FactCount = artifactv2.NaturalFromUint64(2)
				request.SourceClosures[3].RecordCount = artifactv2.NaturalFromUint64(2)
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := exactCallerClosureMutationRequest(t)
			testCase.mutate(t, &request)

			response, err := process.run(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, "conflict", response.ObservationEvaluationStatus)
			require.Equal(t, "incomplete", response.SemanticStatus)
			require.Empty(t, response.PropertyVerdicts)
			require.Nil(t, response.EvidenceBackedModelTrace)
			require.Len(t, response.Diagnostics, 1)
			require.Equal(t, "contradictory-fact", response.Diagnostics[0].Kind)
			evidence, result, err := constructEvaluation(execution, request, response)
			require.NoError(t, err)
			published, err := execution.AdmitEvaluation(evidence, result)
			require.NoError(t, err)
			require.NotEmpty(t, published.Identity())
		})
	}
}

func TestRealCheckerPartialEvidencePublishesAnInMemoryResult(t *testing.T) {
	process := realCheckerProcess(t)
	input := callerClosureExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)

	stdout, stderr, runErr := runRealCheckerOutput(t, process, request)
	require.NoError(t, runErr, string(stderr))
	require.Empty(t, stderr)
	response, err := decodeCheckerResponse(stdout, request)
	require.NoError(t, err)
	require.Equal(t, "unknown", response.ObservationEvaluationStatus)
	require.Equal(t, "incomplete", response.SemanticStatus)
	require.Empty(t, response.PropertyVerdicts)
	require.Nil(t, response.EvidenceBackedModelTrace)
	partialEvidence := projectCheckerEvidence(response, request)
	require.NoError(t, artifactv2.ValidateEvidence(partialEvidence))
	crossPlanEvidence := partialEvidence
	crossPlanEvidence.Diagnostics = append(
		[]artifactv2.ObservationDiagnostic(nil), partialEvidence.Diagnostics...,
	)
	crossPlanEvidence.Diagnostics[0].ObservationPlanDefinitionID = request.ObservationProgram.DefinitionID
	require.ErrorContains(t, artifactv2.ValidateEvidence(crossPlanEvidence),
		"does not match mapping")
	evidence, result, err := constructEvaluation(execution, request, response)
	require.NoError(t, err)
	_, err = execution.AdmitEvaluation(evidence, result)
	require.NoError(t, err)

	output, err := checkWithChecker(context.Background(), input, process.run)
	require.NoError(t, err)
	require.NotEmpty(t, output.Identity())
}

func exactCallerClosureMutationRequest(t *testing.T) checkerRequest {
	t.Helper()
	input := callerClosureExecutionFixture(t)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	one := artifactv2.NaturalFromUint64(1)
	receiptID := "umpire.runtime.fact.control.fixture"
	request.ControlAttempts = []artifactv2.ControlAttempt{{
		OccurrenceDefinitionID:  "workflow-nexus.occurrence.force-close",
		ActionDefinitionID:      "workflow.action.force-close",
		Attempt:                 one,
		ReceiptFactDefinitionID: &receiptID,
		Status:                  "accepted",
	}}
	request.CaptureStatus = "closed"
	request.Sources = mutationSources("closed")
	request.SourceClosures = mutationSourceClosures("closed")
	request.Facts = exactCallerClosureFacts(request.RunIdentity)
	return request
}

func mutationSources(status string) []artifactv2.RawEvidenceSource {
	counts := []uint64{1, 1, 6, 1}
	definitions := []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}
	result := make([]artifactv2.RawEvidenceSource, len(definitions))
	for index, definitionID := range definitions {
		result[index] = artifactv2.RawEvidenceSource{
			SourceDefinitionID: definitionID,
			Status:             status,
			FactCount:          artifactv2.NaturalFromUint64(counts[index]),
			ByteCount:          artifactv2.NaturalFromUint64(0),
		}
	}
	return result
}

func mutationSourceClosures(status string) []artifactv2.SourceClosure {
	sources := mutationSources(status)
	result := make([]artifactv2.SourceClosure, len(sources))
	for index, source := range sources {
		result[index] = artifactv2.SourceClosure{
			SourceDefinitionID: source.SourceDefinitionID,
			Status:             source.Status,
			RecordCount:        source.FactCount,
			ByteCount:          source.ByteCount,
		}
	}
	return result
}

func exactCallerClosureFacts(runIdentity string) []artifactv2.RawEvidenceFact {
	facts := []artifactv2.RawEvidenceFact{
		mutationFact("umpire.runtime.fact.cleanup.fixture", umpireruntime.EvidenceSourceCleanup,
			"umpire.evidence.kind.cleanup", 0, nil, []artifactv2.RawEvidenceField{
				mutationField("umpire.evidence.field.open-handle-count", json.Number("0")),
				mutationField("umpire.evidence.field.status", "complete"),
			}),
		mutationFact("umpire.runtime.fact.control.fixture", umpireruntime.EvidenceSourceControlReceipt,
			"umpire.evidence.kind.control-receipt", 0, nil, []artifactv2.RawEvidenceField{
				mutationField("umpire.evidence.field.action-definition-id", "workflow.action.force-close"),
				mutationField("umpire.evidence.field.attempt", json.Number("1")),
				mutationField("umpire.evidence.field.occurrence-definition-id",
					"workflow-nexus.occurrence.force-close"),
				mutationField("umpire.evidence.field.status", "accepted"),
			}),
	}
	events := []string{
		"temporal.history.WorkflowExecutionStarted",
		"temporal.history.NexusOperationScheduled",
		"temporal.history.NexusOperationStarted",
		"temporal.history.NexusOperationCancelRequested",
		"temporal.history.NexusOperationCancelRequestCompleted",
		"temporal.history.WorkflowExecutionCanceled",
	}
	for index, event := range events {
		definitionID := "umpire.runtime.fact.history." + strconv.Itoa(index+1)
		parents := []string{}
		if index != 0 {
			parents = []string{"umpire.runtime.fact.history." + strconv.Itoa(index)}
		}
		facts = append(facts, mutationFact(definitionID, umpireruntime.EvidenceSourceHistory,
			"umpire.evidence.kind.workflow-history-event", uint64(index), parents,
			[]artifactv2.RawEvidenceField{
				mutationField("umpire.evidence.field.event-id", json.Number(strconv.Itoa(index+1))),
				mutationField("umpire.evidence.field.event-type", event),
				mutationField("umpire.evidence.field.operation-correlation-id",
					"temporal.operation.caller-closure.fixture"),
				mutationField("umpire.evidence.field.run-correlation-id",
					runIdentity),
				mutationField("umpire.evidence.field.workflow-correlation-id",
					"temporal.workflow.caller-closure.fixture"),
			}))
	}
	facts = append(facts, mutationFact(
		"umpire.runtime.fact.participant.fixture", umpireruntime.EvidenceSourceParticipantOutput,
		"umpire.evidence.kind.participant-command", 0, nil, []artifactv2.RawEvidenceField{
			mutationField("umpire.evidence.field.cancellation-callback-count", json.Number("1")),
			mutationField("umpire.evidence.field.command-kind", "realize"),
			{
				FieldDefinitionID: "umpire.evidence.field.endpoint-identity",
				Disposition:       "sha256",
				Value:             "sha256:d86e4da201f1fdd1e116376d712fe630f7bbf8d98cc08fa3ed1b2c087a7aac1c",
			},
			{
				FieldDefinitionID: "umpire.evidence.field.namespace-identity",
				Disposition:       "sha256",
				Value:             "sha256:d86e4da201f1fdd1e116376d712fe630f7bbf8d98cc08fa3ed1b2c087a7aac1c",
			},
			mutationField("umpire.evidence.field.operation-correlation-id",
				"temporal.operation.caller-closure.fixture"),
			mutationField("umpire.evidence.field.run-correlation-id", runIdentity),
			mutationField("umpire.evidence.field.status", "accepted"),
			{
				FieldDefinitionID: "umpire.evidence.field.task-queue-identity",
				Disposition:       "sha256",
				Value:             "sha256:d86e4da201f1fdd1e116376d712fe630f7bbf8d98cc08fa3ed1b2c087a7aac1c",
			},
			mutationField("umpire.evidence.field.workflow-correlation-id",
				"temporal.workflow.caller-closure.fixture"),
		}))
	return facts
}

func setMutationField(
	t *testing.T,
	fact *artifactv2.RawEvidenceFact,
	definitionID string,
	value any,
) {
	t.Helper()
	for index := range fact.Fields {
		if fact.Fields[index].FieldDefinitionID == definitionID {
			fact.Fields[index].Value = value
			return
		}
	}
	require.Failf(t, "mutation field not found", "field=%q", definitionID)
}

func mutationFact(
	definitionID string,
	sourceDefinitionID string,
	kindDefinitionID string,
	ordinal uint64,
	parents []string,
	fields []artifactv2.RawEvidenceField,
) artifactv2.RawEvidenceFact {
	if parents == nil {
		parents = []string{}
	}
	return artifactv2.RawEvidenceFact{
		FactDefinitionID: definitionID, SourceDefinitionID: sourceDefinitionID,
		Ordinal: artifactv2.NaturalFromUint64(ordinal), KindDefinitionID: kindDefinitionID,
		CausalFactDefinitionIDs: parents, Fields: fields,
	}
}

func mutationField(definitionID string, value any) artifactv2.RawEvidenceField {
	return artifactv2.RawEvidenceField{
		FieldDefinitionID: definitionID,
		Disposition:       "plain",
		Value:             value,
	}
}
