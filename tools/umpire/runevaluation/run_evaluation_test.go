package runevaluation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestCheckWithCheckerConstructsOneAdmittedEvaluationSet(t *testing.T) {
	input := callerClosureExecutionFixture(t)
	inputIdentity := input.Identity()
	inputChecksum := input.Checksum()
	checkingCalls := 0

	output, err := checkWithChecker(context.Background(), input,
		func(_ context.Context, request checkerRequest) (checkerResponse, error) {
			checkingCalls++
			require.Equal(t, expectedCallerClosureRequest(t, input), request)
			return unknownCheckerResponse(request), nil
		})
	require.NoError(t, err)
	require.Equal(t, 1, checkingCalls)
	require.NotEmpty(t, output.Identity())
	require.NotEqual(t, inputIdentity, output.Identity())
	require.NotEqual(t, inputChecksum, output.Checksum())
	require.Equal(t, inputIdentity, input.Identity())
	require.Equal(t, inputChecksum, input.Checksum())
	_, ok := output.Execution()
	require.False(t, ok)
}

func TestCheckWithCheckerRejectsNonExecutionBeforeChecking(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		input func(*testing.T) artifact.AdmittedSet
	}{
		{name: "two members", input: callerClosureExecutableFixture},
		{name: "wrong profile", input: callerClosureExecutionWithWrongProfile},
		{name: "wrong source", input: callerClosureExecutionWithWrongSource},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			checkingCalls := 0
			_, err := checkWithChecker(context.Background(), testCase.input(t),
				func(context.Context, checkerRequest) (checkerResponse, error) {
					checkingCalls++
					return checkerResponse{}, nil
				})
			require.Error(t, err)
			require.Zero(t, checkingCalls)
		})
	}
}

func TestCheckWithCheckerErrorsExposeStableClassification(t *testing.T) {
	type classifiedFailure interface {
		error
		Kind() string
		Phase() string
		Code() string
	}
	for _, testCase := range []struct {
		name string
		run  func(*testing.T) error
		want []string
	}{
		{
			name: "input",
			run: func(t *testing.T) error {
				_, err := checkWithChecker(context.Background(), callerClosureExecutableFixture(t),
					func(context.Context, checkerRequest) (checkerResponse, error) {
						panic("checker reached for invalid input")
					})
				return err
			},
			want: []string{"input", "admission", "umpire.run-evaluation.input.exact-four-member-set"},
		},
		{
			name: "checker",
			run: func(t *testing.T) error {
				_, err := checkWithChecker(context.Background(), callerClosureExecutionFixture(t),
					func(context.Context, checkerRequest) (checkerResponse, error) {
						return checkerResponse{}, errors.New("checker failed")
					})
				return err
			},
			want: []string{"checker", "Observation Evaluation", "umpire.run-evaluation.checker.failed"},
		},
		{
			name: "output invariant",
			run: func(t *testing.T) error {
				_, err := checkWithChecker(context.Background(), callerClosureExecutionFixture(t),
					func(context.Context, checkerRequest) (checkerResponse, error) {
						return checkerResponse{}, nil
					})
				return err
			},
			want: []string{"output-invariant", "evaluation", "umpire.run-evaluation.response.invalid"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.run(t)
			require.Error(t, err)
			var classified classifiedFailure
			require.ErrorAs(t, err, &classified)
			require.Equal(t, testCase.want,
				[]string{classified.Kind(), classified.Phase(), classified.Code()})
		})
	}
}

func callerClosureExecutableFixture(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	members := []artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: readCallerClosureInput(t, "experiment.json")},
		{Path: "artifacts/runtime-configuration.json", Encoded: readCallerClosureInput(t, "runtime-configuration.json")},
	}
	admitted, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	return admitted
}

func callerClosureExecutionFixture(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	return callerClosureExecutionFromExecutable(t, callerClosureExecutableFixture(t), callerClosureSources())
}

func callerClosureExecutionWithWrongProfile(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	experiment, err := artifact.DecodeExperimentV2(readCallerClosureInput(t, "experiment.json"))
	require.NoError(t, err)
	configuration, err := artifact.DecodeRuntimeConfigurationV2(
		readCallerClosureInput(t, "runtime-configuration.json"),
	)
	require.NoError(t, err)
	configuration.AuthorityProfile.DefinitionID = "temporal.runtime-profile.unsupported"
	configuration, err = artifactv2.SealRuntimeConfiguration(configuration)
	require.NoError(t, err)
	experimentBytes, err := artifact.EncodeExperimentV2(experiment)
	require.NoError(t, err)
	configurationBytes, err := artifact.EncodeRuntimeConfigurationV2(configuration)
	require.NoError(t, err)
	admitted, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: configurationBytes},
	})
	require.NoError(t, err)
	return callerClosureExecutionFromExecutable(t, admitted, callerClosureSources())
}

func callerClosureExecutionWithWrongSource(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	sources := callerClosureSources()
	sources[len(sources)-1].SourceDefinitionID = "umpire.evidence.source.unsupported"
	return callerClosureExecutionFromExecutable(t, callerClosureExecutableFixture(t), sources)
}

func callerClosureExecutionFromExecutable(
	t *testing.T,
	admitted artifact.AdmittedSet,
	sources []artifactv2.RawEvidenceSource,
) artifact.AdmittedSet {
	t.Helper()
	executable, ok := admitted.Executable()
	require.True(t, ok)
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	one := artifactv2.NaturalFromUint64(1)
	two := artifactv2.NaturalFromUint64(2)
	run := artifactv2.ExperimentRun{
		FormatVersion:        artifactv2.ExperimentRunFormat,
		RunIdentity:          "umpire.local.caller-closure.evaluation-fixture",
		BehaviorFingerprint:  testDigest("a"),
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Attempt:              one,
		OperationalStatus:    "incomplete",
		PhaseOutcomes: []artifactv2.PhaseOutcome{
			{Phase: "preparation", Status: "succeeded", StartedAtUnixMillis: &one, FinishedAtUnixMillis: &two},
			{Phase: "realization", Status: "succeeded", StartedAtUnixMillis: &one, FinishedAtUnixMillis: &two},
			{Phase: "observation", Status: "succeeded", StartedAtUnixMillis: &one, FinishedAtUnixMillis: &two},
			{Phase: "isolation", Status: "succeeded", StartedAtUnixMillis: &one, FinishedAtUnixMillis: &two},
			{Phase: "cleanup", Status: "succeeded", StartedAtUnixMillis: &one, FinishedAtUnixMillis: &two},
		},
		ControlAttempts: []artifactv2.ControlAttempt{{
			OccurrenceDefinitionID: "workflow-nexus.occurrence.force-close",
			ActionDefinitionID:     "workflow.action.force-close",
			Attempt:                one,
			Status:                 "not-attempted",
		}},
		SourceClosures: callerClosureSourceClosures(sources),
		Cleanup: artifactv2.CleanupOutcome{
			Status: "complete", OpenHandleCount: artifactv2.NaturalFromUint64(0),
		},
		Limits:    configuration.PhaseLimits,
		KnownGaps: []artifactv2.KnownGap{},
		Provenance: artifactv2.Provenance{
			SourceDefinitionIDs: []string{"umpire.runtime.engine"},
			SourceLocations: []artifactv2.SourceLocation{{
				Path: "tools/umpire/internal/runtimeengine/engine.go", Line: one, Column: one,
				Provenance: "runtime-engine",
			}},
		},
	}
	run, err = artifactv2.SealExperimentRun(run)
	require.NoError(t, err)
	rawEvidence := artifactv2.RawEvidence{
		FormatVersion:        artifactv2.RawEvidenceFormat,
		RunIdentity:          run.RunIdentity,
		BehaviorFingerprint:  testDigest("b"),
		Experiment:           experimentBinding,
		RuntimeConfiguration: artifactv2.RuntimeConfigurationArtifactBinding(configuration),
		Run:                  artifactv2.ExperimentRunArtifactBinding(run),
		CaptureStatus:        "partial",
		Sources:              sources,
		Facts:                []artifactv2.RawEvidenceFact{},
		KnownGaps:            []artifactv2.KnownGap{},
		Provenance:           run.Provenance,
	}
	rawEvidence, err = artifactv2.SealRawEvidence(rawEvidence)
	require.NoError(t, err)
	execution, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)
	return execution
}

func callerClosureSourceClosures(sources []artifactv2.RawEvidenceSource) []artifactv2.SourceClosure {
	closures := make([]artifactv2.SourceClosure, len(sources))
	for index, source := range sources {
		closures[index] = artifactv2.SourceClosure{
			SourceDefinitionID: source.SourceDefinitionID,
			Status:             source.Status,
			RecordCount:        source.FactCount,
			ByteCount:          source.ByteCount,
		}
	}
	return closures
}

func callerClosureSources() []artifactv2.RawEvidenceSource {
	zero := artifactv2.NaturalFromUint64(0)
	return []artifactv2.RawEvidenceSource{
		{SourceDefinitionID: umpireruntime.EvidenceSourceCleanup, Status: "partial", FactCount: zero, ByteCount: zero},
		{SourceDefinitionID: umpireruntime.EvidenceSourceControlReceipt, Status: "partial", FactCount: zero, ByteCount: zero},
		{SourceDefinitionID: umpireruntime.EvidenceSourceHistory, Status: "partial", FactCount: zero, ByteCount: zero},
		{SourceDefinitionID: umpireruntime.EvidenceSourceParticipantOutput, Status: "partial", FactCount: zero, ByteCount: zero},
	}
}

func expectedCallerClosureRequest(t *testing.T, admitted artifact.AdmittedSet) checkerRequest {
	t.Helper()
	execution, ok := admitted.Execution()
	require.True(t, ok)
	experiment := execution.Experiment()
	configuration := execution.RuntimeConfiguration()
	run := execution.ExperimentRun()
	rawEvidence := execution.RawEvidence()
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	properties := make([]propertyReference, len(experiment.Properties))
	for index, property := range experiment.Properties {
		properties[index] = propertyReference{
			DefinitionID: property.DefinitionID, BehaviorFingerprint: property.BehaviorFingerprint,
			RequirementDefinitionIDs: property.RequirementDefinitionIDs,
		}
	}
	return checkerRequest{
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
}

func unknownCheckerResponse(request checkerRequest) checkerResponse {
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
		ObservationEvaluationStatus:             "unknown",
		EvidenceLinks:                           []artifactv2.EvidenceLink{},
		Dispositions:                            []artifactv2.FieldDispositionRecord{},
		Diagnostics: []artifactv2.ObservationDiagnostic{{
			Kind: "empty-evidence", ObservationPlanDefinitionID: request.Mapping.DefinitionID,
			RelatedDefinitionIDs: []string{}, Alternatives: []string{},
		}},
		ObservationKnownGaps: []artifactv2.KnownGap{},
		PropertyVerdicts:     []artifactv2.PropertyVerdict{},
		QuerySummary: artifactv2.QuerySummary{
			QueryDefinitionID:               request.Query.DefinitionID,
			Status:                          "incomplete",
			QueryLimits:                     callerClosureQueryLimits(),
			RequiredPropertyDefinitionIDs:   []string{request.Properties[0].DefinitionID},
			PropertyVerdicts:                []artifactv2.PropertyVerdict{},
			MissingPropertyDefinitionIDs:    []string{request.Properties[0].DefinitionID},
			DuplicatePropertyDefinitionIDs:  []string{},
			UnexpectedPropertyDefinitionIDs: []string{},
			DivergentPropertyDefinitionIDs:  []string{},
			WrongQueryResultDefinitionIDs:   []string{},
			TraceIDs:                        []string{},
		},
		SemanticStatus:            "incomplete",
		ResultKnownGaps:           []artifactv2.KnownGap{},
		EvaluationOutcomeChecksum: nil,
	}
}

func callerClosureQueryLimits() artifactv2.Limits {
	return artifactv2.Limits{
		Behavior: artifactv2.BehaviorLimits{
			Transitions: artifactv2.Limit{
				Value: artifactv2.NaturalFromUint64(1), Unit: "semantic-transitions",
			},
			SelectedActions: artifactv2.Limit{
				Value: artifactv2.NaturalFromUint64(1), Unit: "selected-actions",
			},
		},
		Search: artifactv2.Limit{
			Value: artifactv2.NaturalFromUint64(8), Unit: "candidate-evaluations",
		},
	}
}

func readCallerClosureInput(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "temporal", "nexus", "testdata", "caller-closure-input-set", "artifacts", name,
	))
	require.NoError(t, err)
	return encoded
}
