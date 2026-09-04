//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

const liveCallerClosureRunIdentity = "umpire.local.caller-closure.live-evaluation-1"

type liveEvaluationSummary struct {
	FormatVersion               string `json:"formatVersion"`
	RunIdentity                 string `json:"runIdentity"`
	OperationalStatus           string `json:"operationalStatus"`
	ObservationEvaluationStatus string `json:"observationEvaluationStatus"`
	SemanticStatus              string `json:"semanticStatus"`
	EvidenceArtifactChecksum    string `json:"evidenceArtifactChecksum"`
	ResultArtifactChecksum      string `json:"resultArtifactChecksum"`
	EvaluationOutcomeChecksum   string `json:"evaluationOutcomeChecksum"`
	ArtifactSetChecksum         string `json:"artifactSetChecksum"`
	ManifestSHA256              string `json:"manifestSha256"`
	Destination                 string `json:"destination"`
}

type liveImplementationTarget struct {
	definitionID        string
	kind                string
	behaviorFingerprint string
}

type liveSemanticSnapshot struct {
	resultArtifactChecksum      string
	resultBehaviorFingerprint   string
	propertyDefinitionID        string
	propertyBehaviorFingerprint string
	implementationDestination   liveImplementationTarget
}

func TestUmpireCallerClosureRunEvaluation(t *testing.T) {
	repositoryRoot, err := filepath.Abs("..")
	require.NoError(t, err)
	requireCorruptionAndAmbiguityControls(t, repositoryRoot)

	input := liveCallerClosureInput(t)
	env, factory := newUmpireTestEnvironment(t)
	binding := newUmpireNexusBinding(t, factory)
	ctx, cancel := context.WithTimeout(env.Context(), 135*time.Second)
	defer cancel()
	operational, err := runner.Run(
		ctx,
		input,
		callerClosureInputBinding,
		liveCallerClosureRunIdentity,
		binding,
	)
	require.NoError(t, err)
	requireLiveOperationalClosure(t, operational)

	executionRoot := filepath.Join(t.TempDir(), "execution")
	writeRunEvaluationExecutionSet(t, executionRoot, operational.AdmittedSet())
	outputRoot := t.TempDir()
	firstBytes := runLiveEvaluationCommand(t, repositoryRoot, executionRoot, outputRoot)
	secondBytes := runLiveEvaluationCommand(t, repositoryRoot, executionRoot, outputRoot)
	require.Equal(t, firstBytes, secondBytes)

	var summary liveEvaluationSummary
	require.NoError(t, json.Unmarshal(firstBytes, &summary))
	require.Equal(t, "umpire-local-run-evaluation-summary/v2", summary.FormatVersion)
	require.Equal(t, liveCallerClosureRunIdentity, summary.RunIdentity)
	require.Equal(t, "succeeded", summary.OperationalStatus)
	require.Equal(t, "accepted", summary.ObservationEvaluationStatus)
	require.Equal(t, "satisfied", summary.SemanticStatus)
	require.Equal(t, filepath.Join(
		outputRoot, "sets", strings.TrimPrefix(summary.ManifestSHA256, "sha256:"),
	), summary.Destination)

	reopened, err := artifact.LoadSet(summary.Destination)
	require.NoError(t, err)
	require.Equal(t, summary.ArtifactSetChecksum, reopened.Checksum())
	require.Equal(t, summary.ManifestSHA256, reopened.ManifestSHA256())
	requireLiveEvaluationMembers(t, executionRoot, summary.Destination)

	evidenceBytes, err := os.ReadFile(filepath.Join(summary.Destination, "artifacts", "evidence.json"))
	require.NoError(t, err)
	resultBytes, err := os.ReadFile(filepath.Join(summary.Destination, "artifacts", "result.json"))
	require.NoError(t, err)
	requireLiveSemanticResult(t, operational, evidenceBytes, resultBytes, summary)
	requireNoNexusEndpoints(t, env.Context(), env.OperatorClient())
}

func requireCorruptionAndAmbiguityControls(t *testing.T, repositoryRoot string) {
	t.Helper()
	modelRoot := filepath.Join(repositoryRoot, "model")
	runBoundedCommand(t, modelRoot, "mise", "exec", "--", "lake", "build",
		"Temporal.Tool.RunEvaluationMutationTests")
	const controls = "^(TestRawArtifactMutationFailsAtAdmission|" +
		"TestCheckerRequestSeparatesRuntimeAndCheckedMappings|" +
		"TestCheckerResponseRejectsConsistentCheckedProfileDriftAtTheProtocolBoundary|" +
		"TestRealCheckerObservationMutationMatrix|" +
		"TestRealCheckerMisboundParticipantCancellationEvidenceIsSemanticConflict|" +
		"TestRealCheckerPartialEvidencePublishesAnInMemoryResult|" +
		"TestRealCheckerSiblingIsDeterministic|" +
		"TestRealCheckerSiblingAdmitsExactAcceptedSet|" +
		"TestRealCheckerCancellationPublishesNoPartialSet)$"
	runBoundedCommand(t, filepath.Join(repositoryRoot, "tools", "umpire", "runevaluation"),
		"go", "test", "-count=1", "-run", controls, ".")
}

func runBoundedCommand(t *testing.T, directory string, name string, arguments ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NoError(t, ctx.Err())
	return output
}

func liveCallerClosureInput(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	return loadUmpireCallerClosureInputSet(t, "caller-closure-input-set")
}

func writeRunEvaluationExecutionSet(t *testing.T, root string, admitted artifact.AdmittedSet) {
	t.Helper()
	execution, ok := admitted.Execution()
	require.True(t, ok)
	experiment, err := artifact.EncodeExperimentV2(execution.Experiment())
	require.NoError(t, err)
	configuration, err := artifact.EncodeRuntimeConfigurationV2(execution.RuntimeConfiguration())
	require.NoError(t, err)
	run, err := artifact.EncodeExperimentRunV2(execution.ExperimentRun())
	require.NoError(t, err)
	rawEvidence, err := artifact.EncodeRawEvidenceV2(execution.RawEvidence())
	require.NoError(t, err)
	files := map[string][]byte{
		"manifest.json":                        admitted.ManifestBytes(),
		"artifacts/experiment.json":            experiment,
		"artifacts/runtime-configuration.json": configuration,
		"artifacts/experiment-run.json":        run,
		"artifacts/raw-evidence.json":          rawEvidence,
	}
	for path, encoded := range files {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0o700))
		require.NoError(t, os.WriteFile(absolute, encoded, 0o600))
	}
}

func requireLiveOperationalClosure(
	t *testing.T,
	operational umpireruntime.Output,
) {
	t.Helper()
	run := operational.ExperimentRun()
	rawEvidence := operational.RawEvidence()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, "complete", run.Cleanup.Status)
	require.True(t, run.Cleanup.OpenHandleCount.IsZero())
	require.Equal(t, "closed", rawEvidence.CaptureStatus)
	require.Empty(t, run.KnownGaps)
	require.Empty(t, rawEvidence.KnownGaps)
	sourceDefinitionIDs := make([]string, len(run.SourceClosures))
	for index, closure := range run.SourceClosures {
		sourceDefinitionIDs[index] = closure.SourceDefinitionID
	}
	require.Equal(t, []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}, sourceDefinitionIDs)
	for index, source := range rawEvidence.Sources {
		require.Equal(t, run.SourceClosures[index].SourceDefinitionID, source.SourceDefinitionID)
		require.Equal(t, "closed", run.SourceClosures[index].Status)
		require.Equal(t, "closed", source.Status)
		require.Equal(t, run.SourceClosures[index].RecordCount, source.FactCount)
		require.Equal(t, run.SourceClosures[index].ByteCount, source.ByteCount)
	}
}

func runLiveEvaluationCommand(
	t *testing.T,
	repositoryRoot string,
	executionRoot string,
	outputRoot string,
) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx,
		"make", "-C", repositoryRoot, "--no-print-directory",
		"umpire-check-local-run-evaluation", "SET="+executionRoot, "OUTPUT_ROOT="+outputRoot,
	)
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Run(), stdout.String(), stderr.String())
	require.NoError(t, ctx.Err())
	require.Empty(t, stderr.String())
	require.Equal(t, 1, bytes.Count(stdout.Bytes(), []byte{'\n'}))
	return stdout.Bytes()
}

func requireLiveEvaluationMembers(t *testing.T, executionRoot string, destination string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(destination, "artifacts"))
	require.NoError(t, err)
	names := make([]string, len(entries))
	for index, entry := range entries {
		require.False(t, entry.IsDir())
		names[index] = entry.Name()
	}
	require.Equal(t, []string{
		"evidence.json",
		"experiment-run.json",
		"experiment.json",
		"raw-evidence.json",
		"result.json",
		"runtime-configuration.json",
	}, names)
	for _, name := range []string{
		"experiment.json", "runtime-configuration.json", "experiment-run.json", "raw-evidence.json",
	} {
		input, err := os.ReadFile(filepath.Join(executionRoot, "artifacts", name))
		require.NoError(t, err)
		published, err := os.ReadFile(filepath.Join(destination, "artifacts", name))
		require.NoError(t, err)
		require.Equal(t, input, published)
	}
}

func requireLiveSemanticResult(
	t *testing.T,
	operational umpireruntime.Output,
	evidenceBytes []byte,
	resultBytes []byte,
	summary liveEvaluationSummary,
) liveSemanticSnapshot {
	t.Helper()
	run := operational.ExperimentRun()
	rawEvidence := operational.RawEvidence()
	evidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	require.Equal(t, summary.EvidenceArtifactChecksum, evidence.ArtifactChecksum)
	require.Equal(t, summary.ResultArtifactChecksum, result.ArtifactChecksum)
	require.Equal(t, summary.EvaluationOutcomeChecksum, *result.EvaluationOutcomeChecksum)
	require.Equal(t, "accepted", evidence.ObservationEvaluationStatus)
	require.NotNil(t, evidence.EvidenceBackedModelTrace)
	require.True(t, evidence.EvidenceBackedModelTrace.SourceClosed)
	require.Empty(t, evidence.Diagnostics)
	require.Empty(t, evidence.KnownGaps)
	require.Equal(t, rawEvidence.Run, evidence.Run)
	require.Equal(t, run.FormatVersion, evidence.Run.FormatVersion)
	require.Equal(t, run.ArtifactChecksum, evidence.Run.ArtifactChecksum)
	require.Equal(t, run.BehaviorFingerprint, evidence.Run.BehaviorFingerprint)
	require.Equal(t, run.ProvenanceChecksum, evidence.Run.ProvenanceChecksum)
	require.Equal(t, rawEvidence.FormatVersion, evidence.RawEvidence.FormatVersion)
	require.Equal(t, rawEvidence.ArtifactChecksum, evidence.RawEvidence.ArtifactChecksum)
	require.Equal(t, rawEvidence.BehaviorFingerprint, evidence.RawEvidence.BehaviorFingerprint)
	require.Equal(t, rawEvidence.ProvenanceChecksum, evidence.RawEvidence.ProvenanceChecksum)
	require.Equal(t, "succeeded", result.OperationalStatus)
	require.Equal(t, "accepted", result.ObservationEvaluationStatus)
	require.Equal(t, "applied", result.ImplementationLinkStatus)
	require.Equal(t, "satisfied", result.SemanticStatus)
	require.Equal(t, "complete", result.CleanupStatus)
	require.Equal(t, evidence.Run, result.Run)
	require.Equal(t, evidence.RawEvidence, result.RawEvidence)
	require.Empty(t, result.KnownGaps)
	require.Equal(t, "temporal.system.nexus.caller-closure.implementation-link",
		result.ImplementationLink.DefinitionID)
	require.Equal(t, "workflow-nexus.target.caller-closure",
		result.ImplementationLink.DestinationTarget.DefinitionID)
	require.Len(t, result.PropertyVerdicts, 1)
	verdict := result.PropertyVerdicts[0]
	require.Equal(t, "workflow-nexus.property.caller-closure", verdict.PropertyDefinitionID)
	require.Equal(t, "satisfied", verdict.Status)
	require.Nil(t, verdict.Diagnostic)
	clauseDefinitionIDs := make([]string, len(verdict.Clauses))
	for index, clause := range verdict.Clauses {
		clauseDefinitionIDs[index] = clause.ClauseDefinitionID
	}
	require.Equal(t, []string{
		"workflow-nexus.property.clause.delivery",
		"workflow-nexus.property.clause.ownership",
		"workflow-nexus.property.clause.uniqueness",
	}, clauseDefinitionIDs)
	for _, clause := range verdict.Clauses {
		require.Equal(t, "satisfied", clause.Status)
		require.NotEmpty(t, clause.EvidenceLinks)
	}
	require.Equal(t, "satisfied", result.QuerySummary.Status)
	require.Empty(t, result.QuerySummary.MissingPropertyDefinitionIDs)
	require.Empty(t, result.QuerySummary.DuplicatePropertyDefinitionIDs)
	require.Empty(t, result.QuerySummary.UnexpectedPropertyDefinitionIDs)
	require.Empty(t, result.QuerySummary.DivergentPropertyDefinitionIDs)
	require.Empty(t, result.QuerySummary.WrongQueryResultDefinitionIDs)

	var cleanupFact string
	var controlFact string
	var terminalHistoryFact string
	var participantFact string
	for _, fact := range rawEvidence.Facts {
		var hasOpenHandleCount bool
		var hasCancellationCallbackCount bool
		var eventType string
		for _, field := range fact.Fields {
			switch field.FieldDefinitionID {
			case umpireruntime.EvidenceFieldOpenHandleCount:
				hasOpenHandleCount = true
			case umpireruntime.EvidenceFieldCancellationCallbackCount:
				hasCancellationCallbackCount = true
			case umpireruntime.EvidenceFieldEventType:
				eventType, _ = field.Value.(string)
			default:
			}
		}
		switch fact.SourceDefinitionID {
		case umpireruntime.EvidenceSourceCleanup:
			if hasOpenHandleCount {
				cleanupFact = fact.FactDefinitionID
			}
		case umpireruntime.EvidenceSourceControlReceipt:
			controlFact = fact.FactDefinitionID
		case umpireruntime.EvidenceSourceParticipantOutput:
			if hasCancellationCallbackCount {
				participantFact = fact.FactDefinitionID
			}
		case umpireruntime.EvidenceSourceHistory:
			if eventType == "temporal.history.WorkflowExecutionCanceled" {
				terminalHistoryFact = fact.FactDefinitionID
			}
		default:
			require.Failf(t, "unexpected evidence source", "source=%q", fact.SourceDefinitionID)
		}
	}
	require.NotEmpty(t, cleanupFact)
	require.NotEmpty(t, controlFact)
	require.NotEmpty(t, terminalHistoryFact)
	require.NotEmpty(t, participantFact)
	require.Contains(t, evidence.EvidenceBackedModelTrace.EvidenceDefinitionIDs, cleanupFact)
	var propertyEvidence []string
	for _, clause := range verdict.Clauses {
		for _, link := range clause.EvidenceLinks {
			propertyEvidence = append(propertyEvidence, link.EvidenceDefinitionIDs...)
		}
	}
	slices.Sort(propertyEvidence)
	propertyEvidence = slices.Compact(propertyEvidence)
	require.Contains(t, propertyEvidence, controlFact)
	require.Contains(t, propertyEvidence, terminalHistoryFact)
	require.Contains(t, propertyEvidence, participantFact)
	require.NotContains(t, propertyEvidence, cleanupFact)
	return liveSemanticSnapshot{
		resultArtifactChecksum:      result.ArtifactChecksum,
		resultBehaviorFingerprint:   result.BehaviorFingerprint,
		propertyDefinitionID:        verdict.PropertyDefinitionID,
		propertyBehaviorFingerprint: verdict.PropertyBehaviorFingerprint,
		implementationDestination: liveImplementationTarget{
			definitionID:        result.ImplementationLink.DestinationTarget.DefinitionID,
			kind:                result.ImplementationLink.DestinationTarget.Kind,
			behaviorFingerprint: result.ImplementationLink.DestinationTarget.BehaviorFingerprint,
		},
	}
}
