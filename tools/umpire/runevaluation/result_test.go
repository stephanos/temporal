package runevaluation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
)

func TestConstructEvaluationPreservesResponseAndAddsExactTransportClosure(t *testing.T) {
	admitted := callerClosureExecutionFixture(t)
	execution, ok := admitted.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	response := unknownCheckerResponse(request)

	evidence, result, err := constructEvaluation(execution, request, response)
	require.NoError(t, err)
	require.Equal(t, response.ObservationEvaluationStatus, evidence.ObservationEvaluationStatus)
	require.Equal(t, response.Diagnostics, evidence.Diagnostics)
	require.Equal(t, response.ObservationKnownGaps, evidence.KnownGaps)
	require.Equal(t, response.PropertyVerdicts, result.PropertyVerdicts)
	require.Equal(t, response.QuerySummary, result.QuerySummary)
	require.Equal(t, response.ResultKnownGaps, result.KnownGaps)
	require.Equal(t, "incomplete", result.OperationalStatus)
	require.Equal(t, execution.ExperimentRun().Cleanup.Status, result.CleanupStatus)
	require.Equal(t, "not-evaluated", result.ImplementationLinkStatus)
	require.Equal(t, callerClosureImplementationLink(), result.ImplementationLink)
	require.Equal(t, execution.Experiment().Plan.TargetDefinitionID,
		result.ImplementationLink.DestinationTarget.DefinitionID)
	require.NotEqual(t, execution.Experiment().Plan.TargetDefinitionID,
		result.ImplementationLink.SourceTarget.DefinitionID)
	require.Equal(t, callerClosureStagedLimits(), result.Limits)
	require.Equal(t, evaluationProvenance(), evidence.Provenance)
	require.Equal(t, evaluationProvenance(), result.Provenance)
	require.NotEmpty(t, evidence.ArtifactChecksum)
	require.NotEmpty(t, result.ArtifactChecksum)
}

func TestCheckWithCheckerRejectsResponseDriftWithoutASet(t *testing.T) {
	input := callerClosureExecutionFixture(t)
	checkerErr := errors.New("fixture checker failure")
	for _, testCase := range []struct {
		name    string
		checker checkerCall
		cause   error
	}{
		{
			name: "checker failure",
			checker: func(context.Context, checkerRequest) (checkerResponse, error) {
				return checkerResponse{}, checkerErr
			},
			cause: checkerErr,
		},
		{
			name: "crossed binding",
			checker: func(_ context.Context, request checkerRequest) (checkerResponse, error) {
				response := unknownCheckerResponse(request)
				response.RunArtifactChecksum = testDigest("f")
				return response, nil
			},
		},
		{
			name: "invalid known gap union",
			checker: func(_ context.Context, request checkerRequest) (checkerResponse, error) {
				response := unknownCheckerResponse(request)
				response.ResultKnownGaps = []artifactv2.KnownGap{{
					Kind: "input", Code: "umpire.known-gap.unbound",
				}}
				return response, nil
			},
		},
		{
			name: "accepted without verdict",
			checker: func(_ context.Context, request checkerRequest) (checkerResponse, error) {
				response := unknownCheckerResponse(request)
				response.ObservationEvaluationStatus = "accepted"
				response.Diagnostics = []artifactv2.ObservationDiagnostic{}
				response.EvidenceBackedModelTrace = nil
				return response, nil
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := checkWithChecker(context.Background(), input, testCase.checker)
			require.Error(t, err)
			require.Empty(t, output.Identity())
			if testCase.cause != nil {
				require.ErrorIs(t, err, testCase.cause)
			}
			require.NotContains(t, err.Error(), "fixture checker failure")
		})
	}
}

func TestCheckWithCheckerAdmitsTheCompleteIndependentStatusMatrix(t *testing.T) {
	for _, operationalStatus := range []string{"succeeded", "failed", "incomplete"} {
		for _, semanticCase := range []struct {
			observation string
			semantic    string
		}{
			{observation: "accepted", semantic: "satisfied"},
			{observation: "accepted", semantic: "violated"},
			{observation: "unknown", semantic: "incomplete"},
			{observation: "conflict", semantic: "incomplete"},
			{observation: "unsupported", semantic: "incomplete"},
		} {
			name := operationalStatus + "/" + semanticCase.observation + "/" + semanticCase.semantic
			t.Run(name, func(t *testing.T) {
				input := acceptedCallerClosureExecutionFixture(t, operationalStatus)
				output, err := checkWithChecker(context.Background(), input,
					func(_ context.Context, request checkerRequest) (checkerResponse, error) {
						if semanticCase.observation == "accepted" {
							return acceptedCallerClosureResponse(t, request, semanticCase.semantic), nil
						}
						return nonAcceptedCallerClosureResponse(t, request, semanticCase.observation), nil
					})
				require.NoError(t, err)
				require.NotEmpty(t, output.Identity())
			})
		}
	}
}

func TestAcceptedOutcomeChecksumIsStableAndSensitiveOnlyToSemanticContent(t *testing.T) {
	checksums := make([]string, 0, 3)
	for _, operationalStatus := range []string{"succeeded", "failed", "incomplete"} {
		input := acceptedCallerClosureExecutionFixture(t, operationalStatus)
		execution, ok := input.Execution()
		require.True(t, ok)
		request, err := newCheckerRequest(execution)
		require.NoError(t, err)
		response := acceptedCallerClosureResponse(t, request, "satisfied")
		checksums = append(checksums, *response.EvaluationOutcomeChecksum)

		first, err := checkWithChecker(context.Background(), input,
			func(context.Context, checkerRequest) (checkerResponse, error) { return response, nil })
		require.NoError(t, err)
		second, err := checkWithChecker(context.Background(), input,
			func(context.Context, checkerRequest) (checkerResponse, error) { return response, nil })
		require.NoError(t, err)
		require.Equal(t, first.Identity(), second.Identity())
		require.Equal(t, first.Checksum(), second.Checksum())
		require.Equal(t, first.ManifestBytes(), second.ManifestBytes())
	}
	require.Equal(t, []string{checksums[0], checksums[0], checksums[0]}, checksums)

	input := acceptedCallerClosureExecutionFixture(t, "succeeded")
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	response := acceptedCallerClosureResponse(t, request, "satisfied")
	response.EvidenceLinks[0].MeaningBehaviorFingerprint = testDigest("d")
	response.PropertyVerdicts[0].Clauses[0].EvidenceLinks = response.EvidenceLinks
	response.QuerySummary.PropertyVerdicts = response.PropertyVerdicts

	output, err := checkWithChecker(context.Background(), input,
		func(context.Context, checkerRequest) (checkerResponse, error) { return response, nil })
	require.Error(t, err)
	require.Empty(t, output.Identity())
}

func TestCheckWithCheckerPreservesExactKnownGapMembershipAndUnion(t *testing.T) {
	runGap := artifactv2.KnownGap{
		Kind: "capability-contract", Code: "umpire.known-gap.run-evaluation.run",
	}
	rawGap := artifactv2.KnownGap{
		Kind: "input", Code: "umpire.known-gap.run-evaluation.raw-evidence",
	}
	observationGap := artifactv2.KnownGap{
		Kind: "interpretation", Code: "umpire.known-gap.run-evaluation.observation",
	}
	input := acceptedCallerClosureExecutionFixtureWithGaps(
		t, "failed", []artifactv2.KnownGap{runGap}, []artifactv2.KnownGap{rawGap},
	)
	execution, ok := input.Execution()
	require.True(t, ok)
	request, err := newCheckerRequest(execution)
	require.NoError(t, err)
	response := acceptedCallerClosureResponse(t, request, "satisfied")
	response.ObservationKnownGaps = []artifactv2.KnownGap{observationGap}
	response.ResultKnownGaps = []artifactv2.KnownGap{runGap, rawGap, observationGap}

	evidence, result, err := constructEvaluation(execution, request, response)
	require.NoError(t, err)
	require.Equal(t, []artifactv2.KnownGap{observationGap}, evidence.KnownGaps)
	require.Equal(t, []artifactv2.KnownGap{runGap}, execution.ExperimentRun().KnownGaps)
	require.Equal(t, []artifactv2.KnownGap{rawGap}, execution.RawEvidence().KnownGaps)
	require.Equal(t, []artifactv2.KnownGap{runGap, rawGap, observationGap}, result.KnownGaps)
	output, err := execution.AdmitEvaluation(evidence, result)
	require.NoError(t, err)
	require.NotEmpty(t, output.Identity())

	response.ResultKnownGaps = []artifactv2.KnownGap{runGap, observationGap}
	output, err = checkWithChecker(context.Background(), input,
		func(context.Context, checkerRequest) (checkerResponse, error) { return response, nil })
	require.Error(t, err)
	require.Empty(t, output.Identity())
}

func TestCheckWithCheckerRejectsEverySemanticOutputInvariantClass(t *testing.T) {
	input := acceptedCallerClosureExecutionFixture(t, "succeeded")
	for _, testCase := range []struct {
		name   string
		mutate func(*checkerResponse)
	}{
		{name: "missing verdict", mutate: func(response *checkerResponse) {
			response.PropertyVerdicts = []artifactv2.PropertyVerdict{}
			response.QuerySummary.Status = "incomplete"
			response.QuerySummary.PropertyVerdicts = []artifactv2.PropertyVerdict{}
			response.QuerySummary.MissingPropertyDefinitionIDs = response.QuerySummary.RequiredPropertyDefinitionIDs
			response.QuerySummary.TraceIDs = []string{}
			response.SemanticStatus = "incomplete"
			response.EvaluationOutcomeChecksum = nil
		}},
		{name: "duplicate verdict", mutate: func(response *checkerResponse) {
			response.PropertyVerdicts = append(response.PropertyVerdicts, response.PropertyVerdicts[0])
			response.QuerySummary.Status = "incomplete"
			response.QuerySummary.PropertyVerdicts = response.PropertyVerdicts
			response.QuerySummary.DuplicatePropertyDefinitionIDs =
				response.QuerySummary.RequiredPropertyDefinitionIDs
			response.SemanticStatus = "incomplete"
			response.EvaluationOutcomeChecksum = nil
		}},
		{name: "unexpected verdict", mutate: func(response *checkerResponse) {
			unexpected := "workflow-nexus.property.caller-closure-unexpected"
			response.PropertyVerdicts[0].PropertyDefinitionID = unexpected
			response.QuerySummary.Status = "incomplete"
			response.QuerySummary.PropertyVerdicts = response.PropertyVerdicts
			response.QuerySummary.MissingPropertyDefinitionIDs =
				response.QuerySummary.RequiredPropertyDefinitionIDs
			response.QuerySummary.UnexpectedPropertyDefinitionIDs = []string{unexpected}
			response.SemanticStatus = "incomplete"
			response.EvaluationOutcomeChecksum = nil
		}},
		{name: "query partition", mutate: func(response *checkerResponse) {
			response.QuerySummary.MissingPropertyDefinitionIDs =
				response.QuerySummary.RequiredPropertyDefinitionIDs
		}},
		{name: "evidence link bijection", mutate: func(response *checkerResponse) {
			response.EvidenceLinks = []artifactv2.EvidenceLink{}
		}},
		{name: "disposition", mutate: func(response *checkerResponse) {
			response.Dispositions[0].Disposition = "hash"
		}},
		{name: "diagnostic matrix", mutate: func(response *checkerResponse) {
			response.Diagnostics = []artifactv2.ObservationDiagnostic{{
				Kind:                        "empty-evidence",
				ObservationPlanDefinitionID: callerClosureObservationProgramID,
				RelatedDefinitionIDs:        []string{}, Alternatives: []string{},
			}}
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := checkWithChecker(context.Background(), input,
				func(_ context.Context, request checkerRequest) (checkerResponse, error) {
					response := acceptedCallerClosureResponse(t, request, "satisfied")
					testCase.mutate(&response)
					return response, nil
				})
			require.Error(t, err)
			require.Empty(t, output.Identity())
		})
	}
}

func acceptedCallerClosureExecutionFixture(t *testing.T, operationalStatus string) artifact.AdmittedSet {
	return acceptedCallerClosureExecutionFixtureWithGaps(t, operationalStatus, nil, nil)
}

func acceptedCallerClosureExecutionFixtureWithGaps(
	t *testing.T,
	operationalStatus string,
	runGaps []artifactv2.KnownGap,
	rawEvidenceGaps []artifactv2.KnownGap,
) artifact.AdmittedSet {
	t.Helper()
	admitted := callerClosureExecutableFixture(t)
	executable, ok := admitted.Executable()
	require.True(t, ok)
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	run, err := artifact.DecodeExperimentRunV2(readEvaluationArtifactFixture(t, "experiment-run.json"))
	require.NoError(t, err)
	rawEvidence, err := artifact.DecodeRawEvidenceV2(readEvaluationArtifactFixture(t, "raw-evidence.json"))
	require.NoError(t, err)

	run.RunIdentity = "umpire.local.caller-closure.accepted-fixture"
	run.Experiment = experimentBinding
	run.RuntimeConfiguration = artifactv2.RuntimeConfigurationArtifactBinding(configuration)
	run.Limits = configuration.PhaseLimits
	run.ControlAttempts[0].OccurrenceDefinitionID = "workflow-nexus.occurrence.force-close"
	run.ControlAttempts[0].ActionDefinitionID = "workflow.action.force-close"
	run.ControlAttempts[0].Status = "accepted"
	run.ControlAttempts[0].Code = nil
	if operationalStatus == "incomplete" {
		code := "umpire.runtime.control.canceled"
		run.ControlAttempts[0].Status = "canceled"
		run.ControlAttempts[0].Code = &code
	}
	if operationalStatus == "failed" {
		code := "umpire.runtime.control.rejected"
		run.ControlAttempts[0].Status = "rejected"
		run.ControlAttempts[0].Code = &code
	}
	run.OperationalStatus = operationalStatus
	run.KnownGaps = append([]artifactv2.KnownGap{}, runGaps...)
	run.Provenance = evaluationFixtureRuntimeProvenance()

	rawEvidence.RunIdentity = run.RunIdentity
	rawEvidence.Experiment = experimentBinding
	rawEvidence.RuntimeConfiguration = artifactv2.RuntimeConfigurationArtifactBinding(configuration)
	rawEvidence.KnownGaps = append([]artifactv2.KnownGap{}, rawEvidenceGaps...)
	rawEvidence.Provenance = run.Provenance
	for factIndex := range rawEvidence.Facts {
		fact := &rawEvidence.Facts[factIndex]
		if fact.FactDefinitionID != *run.ControlAttempts[0].ReceiptFactDefinitionID {
			continue
		}
		for fieldIndex := range fact.Fields {
			field := &fact.Fields[fieldIndex]
			switch field.FieldDefinitionID {
			case artifactv2.ControlReceiptActionFieldDefinitionID:
				field.Value = run.ControlAttempts[0].ActionDefinitionID
			case artifactv2.ControlReceiptOccurrenceFieldDefinitionID:
				field.Value = run.ControlAttempts[0].OccurrenceDefinitionID
			case artifactv2.ControlReceiptStatusFieldDefinitionID:
				field.Value = run.ControlAttempts[0].Status
			case artifactv2.ControlReceiptAttemptFieldDefinitionID:
				field.Value = json.Number(run.ControlAttempts[0].Attempt.String())
			default:
				continue
			}
		}
	}
	recomputeEvaluationFixtureByteCounts(t, &run, &rawEvidence)
	run, err = artifactv2.SealExperimentRun(run)
	require.NoError(t, err)
	rawEvidence.Run = artifactv2.ExperimentRunArtifactBinding(run)
	rawEvidence, err = artifactv2.SealRawEvidence(rawEvidence)
	require.NoError(t, err)
	execution, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)
	return execution
}

func acceptedCallerClosureResponse(t *testing.T, request checkerRequest, semanticStatus string) checkerResponse {
	t.Helper()
	evidence, err := artifact.DecodeEvidenceV2(readEvaluationArtifactFixture(t, "evidence.json"))
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(readEvaluationArtifactFixture(t, "result.json"))
	require.NoError(t, err)

	trace := evidence.EvidenceBackedModelTrace
	require.NotNil(t, trace)
	trace.ObservationPlan = artifactv2.DefinitionReference{
		DefinitionID:        request.Mapping.DefinitionID,
		BehaviorFingerprint: request.Mapping.BehaviorFingerprint,
	}
	trace.MappingDefinitionID = request.Mapping.DefinitionID
	trace.MappingBehaviorFingerprint = request.Mapping.BehaviorFingerprint
	trace.ProfileDefinitionID = callerClosureCheckedProfileID
	trace.AppliedLimit = artifactv2.Limit{
		Value: artifactv2.NaturalFromUint64(4096), Unit: "evidence-records",
	}
	trace.Source = evaluationProvenance().SourceLocations[0]
	for index := range evidence.EvidenceLinks {
		adaptCallerClosureEvidenceLink(&evidence.EvidenceLinks[index])
	}

	verdict := result.PropertyVerdicts[0]
	verdict.QueryDefinitionID = request.Query.DefinitionID
	verdict.PropertyDefinitionID = request.Properties[0].DefinitionID
	verdict.PropertyBehaviorFingerprint = request.Properties[0].BehaviorFingerprint
	verdict.QueryLimits = callerClosureQueryLimits()
	verdict.EvidenceLimit = &trace.AppliedLimit
	verdict.ProvenanceDefinitionIDs = []string{request.Properties[0].DefinitionID, request.Query.DefinitionID}
	for index := range verdict.Clauses {
		clause := &verdict.Clauses[index]
		clause.PropertyDefinitionID = request.Properties[0].DefinitionID
		clause.ClauseDefinitionID = request.Properties[0].DefinitionID + ".clause"
		clause.QueryLimits = callerClosureQueryLimits()
		clause.EvidenceLimit = trace.AppliedLimit
		clause.ProvenanceDefinitionIDs = []string{request.Properties[0].DefinitionID}
		clause.EvidenceLinks = evidence.EvidenceLinks
		if semanticStatus == "violated" {
			clause.Status = "violated"
		}
	}
	verdict.Status = semanticStatus
	result.QuerySummary.QueryDefinitionID = request.Query.DefinitionID
	result.QuerySummary.Status = semanticStatus
	result.QuerySummary.QueryLimits = callerClosureQueryLimits()
	result.QuerySummary.RequiredPropertyDefinitionIDs = []string{request.Properties[0].DefinitionID}
	result.QuerySummary.PropertyVerdicts = []artifactv2.PropertyVerdict{verdict}
	result.QuerySummary.MissingPropertyDefinitionIDs = []string{}
	result.QuerySummary.DuplicatePropertyDefinitionIDs = []string{}
	result.QuerySummary.UnexpectedPropertyDefinitionIDs = []string{}
	result.QuerySummary.DivergentPropertyDefinitionIDs = []string{}
	result.QuerySummary.WrongQueryResultDefinitionIDs = []string{}
	result.QuerySummary.TraceIDs = []string{*verdict.TraceID}

	outcomeChecksum := map[string]string{
		"satisfied": "sha256:aaa003026ed096fc9d2f435e6acf1806a81dd4597a32b86b0a39580fe9b74950",
		"violated":  "sha256:c9ddf0938c9bc484e8ee5c1e9a27834a94a756347c7902c9bdac1aa3251cc883",
	}[semanticStatus]
	return checkerResponse{
		FormatVersion:                           checkerResponseFormat,
		CheckerIdentity:                         checkerIdentity,
		CheckerVersion:                          artifactv2.NaturalFromUint64(2),
		CheckerBehaviorFingerprint:              checkerBehaviorFingerprint,
		ExperimentArtifactChecksum:              request.Experiment.ArtifactChecksum,
		RuntimeConfigurationArtifactChecksum:    request.RuntimeConfiguration.ArtifactChecksum,
		RunArtifactChecksum:                     request.Run.ArtifactChecksum,
		RawEvidenceArtifactChecksum:             request.RawEvidence.ArtifactChecksum,
		ExperimentBehaviorFingerprint:           request.Experiment.BehaviorFingerprint,
		RuntimeConfigurationBehaviorFingerprint: request.RuntimeConfiguration.BehaviorFingerprint,
		RunIdentity:                             request.RunIdentity,
		ObservationEvaluationStatus:             "accepted",
		EvidenceBackedModelTrace:                trace,
		EvidenceLinks:                           evidence.EvidenceLinks,
		Dispositions:                            evidence.Dispositions,
		Diagnostics:                             []artifactv2.ObservationDiagnostic{},
		ObservationKnownGaps:                    []artifactv2.KnownGap{},
		PropertyVerdicts:                        []artifactv2.PropertyVerdict{verdict},
		QuerySummary:                            result.QuerySummary,
		SemanticStatus:                          semanticStatus,
		ResultKnownGaps:                         []artifactv2.KnownGap{},
		EvaluationOutcomeChecksum:               &outcomeChecksum,
	}
}

func nonAcceptedCallerClosureResponse(
	t *testing.T,
	request checkerRequest,
	observationStatus string,
) checkerResponse {
	t.Helper()
	response := acceptedCallerClosureResponse(t, request, "satisfied")
	diagnosticKind := map[string]string{
		"unknown":     "empty-evidence",
		"conflict":    "contradictory-fact",
		"unsupported": "profile-mismatch",
	}[observationStatus]
	response.ObservationEvaluationStatus = observationStatus
	response.EvidenceBackedModelTrace = nil
	response.EvidenceLinks = []artifactv2.EvidenceLink{}
	response.Diagnostics = []artifactv2.ObservationDiagnostic{{
		Kind: diagnosticKind, ObservationPlanDefinitionID: request.Mapping.DefinitionID,
		RelatedDefinitionIDs: []string{}, Alternatives: []string{},
	}}
	response.PropertyVerdicts = []artifactv2.PropertyVerdict{}
	response.QuerySummary.Status = "incomplete"
	response.QuerySummary.PropertyVerdicts = []artifactv2.PropertyVerdict{}
	response.QuerySummary.MissingPropertyDefinitionIDs = []string{request.Properties[0].DefinitionID}
	response.QuerySummary.TraceIDs = []string{}
	response.SemanticStatus = "incomplete"
	response.EvaluationOutcomeChecksum = nil
	return response
}

func adaptCallerClosureEvidenceLink(link *artifactv2.EvidenceLink) {
	link.MappingDefinitionID = callerClosureCheckedMappingID
	link.MappingBehaviorFingerprint = callerClosureCheckedMappingFingerprint
	link.ProfileDefinitionID = callerClosureCheckedProfileID
	link.AppliedLimit = artifactv2.Limit{
		Value: artifactv2.NaturalFromUint64(4096), Unit: "evidence-records",
	}
}

func recomputeEvaluationFixtureByteCounts(
	t *testing.T,
	run *artifactv2.ExperimentRun,
	rawEvidence *artifactv2.RawEvidence,
) {
	t.Helper()
	byteCounts := make(map[string]uint64, len(rawEvidence.Sources))
	for _, fact := range rawEvidence.Facts {
		encoded, err := artifact.CanonicalPretty(fact)
		require.NoError(t, err)
		byteCounts[fact.SourceDefinitionID] += uint64(len(encoded))
	}
	for index := range rawEvidence.Sources {
		count := artifactv2.NaturalFromUint64(byteCounts[rawEvidence.Sources[index].SourceDefinitionID])
		rawEvidence.Sources[index].ByteCount = count
		run.SourceClosures[index].ByteCount = count
	}
}

func evaluationFixtureRuntimeProvenance() artifactv2.Provenance {
	one := artifactv2.NaturalFromUint64(1)
	return artifactv2.Provenance{
		SourceDefinitionIDs: []string{"umpire.runtime.engine"},
		SourceLocations: []artifactv2.SourceLocation{{
			Path: "tools/umpire/internal/runtimeengine/engine.go", Line: one, Column: one,
			Provenance: "runtime-engine",
		}},
	}
}

func readEvaluationArtifactFixture(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "artifact", "testdata", "valid-run-evaluation-set", "artifacts", name,
	))
	require.NoError(t, err)
	return encoded
}
