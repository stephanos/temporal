package runevaluation

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
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
)

const liveCallerClosureRunIdentity = "umpire.local.caller-closure.live-evaluation-1"

var liveCallerClosureBinding = runner.InputBinding{
	ArtifactSetIdentity:                     "umpire.artifact-set.ed3605976ba999ec8e166d4309247e2b711fee18f4a421cfb8c6dc037344f1a2",
	ArtifactSetChecksum:                     "sha256:074356889cda0296b13152f87e57d7b980d76125329a9014ceb5321c3f5bda7b",
	ManifestSHA256:                          "sha256:f381da231395b8fec738837535a8bb8da0dd227a08e3d60bf9c2bda620c46b14",
	ExperimentArtifactChecksum:              "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
	ExperimentBehaviorFingerprint:           "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
	RuntimeConfigurationArtifactChecksum:    "sha256:21b4f7d0db2f68f939df901c2c5d146b1be3e45e55ad6cc171445fda5f29c1d5",
	RuntimeConfigurationBehaviorFingerprint: "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67",
}

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

func TestBoundedLiveCallerClosureEvaluation(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	requireCorruptionAndAmbiguityControls(t, repositoryRoot)

	input := liveCallerClosureInput(t)
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()
	operational, err := runner.Run(
		ctx,
		input,
		liveCallerClosureBinding,
		liveCallerClosureRunIdentity,
		nexus.Binding{},
	)
	require.NoError(t, err)
	requireLiveOperationalClosure(t, operational.ExperimentRun(), operational.RawEvidence())

	executionRoot := filepath.Join(t.TempDir(), "execution")
	writeExecutionSet(t, executionRoot, operational.AdmittedSet())
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
	evidence, err := artifact.DecodeEvidenceV2(evidenceBytes)
	require.NoError(t, err)
	resultBytes, err := os.ReadFile(filepath.Join(summary.Destination, "artifacts", "result.json"))
	require.NoError(t, err)
	result, err := artifact.DecodeResultV2(resultBytes)
	require.NoError(t, err)
	require.Equal(t, summary.EvidenceArtifactChecksum, evidence.ArtifactChecksum)
	require.Equal(t, summary.ResultArtifactChecksum, result.ArtifactChecksum)
	require.Equal(t, summary.EvaluationOutcomeChecksum, *result.EvaluationOutcomeChecksum)
	requireLiveSemanticResult(t, operational.ExperimentRun(), operational.RawEvidence(), evidence, result)
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
	root := filepath.Join("..", "temporal", "nexus", "testdata", "caller-closure-input-set")
	files := make(map[string][]byte, 3)
	for _, path := range []string{
		"manifest.json",
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
	} {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		require.NoError(t, err)
		files[path] = encoded
	}
	input, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	return input
}

func requireLiveOperationalClosure(
	t *testing.T,
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
) {
	t.Helper()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, "complete", run.Cleanup.Status)
	require.True(t, run.Cleanup.OpenHandleCount.IsZero())
	require.Equal(t, "closed", rawEvidence.CaptureStatus)
	require.Empty(t, run.KnownGaps)
	require.Empty(t, rawEvidence.KnownGaps)
	require.Equal(t, []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}, sourceDefinitionIDsFromClosures(run.SourceClosures))
	for index, source := range rawEvidence.Sources {
		require.Equal(t, run.SourceClosures[index].SourceDefinitionID, source.SourceDefinitionID)
		require.Equal(t, "closed", run.SourceClosures[index].Status)
		require.Equal(t, "closed", source.Status)
		require.Equal(t, run.SourceClosures[index].RecordCount, source.FactCount)
		require.Equal(t, run.SourceClosures[index].ByteCount, source.ByteCount)
	}
}

func sourceDefinitionIDsFromClosures(closures []artifactv2.SourceClosure) []string {
	result := make([]string, len(closures))
	for index, closure := range closures {
		result[index] = closure.SourceDefinitionID
	}
	return result
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
	run artifactv2.ExperimentRun,
	rawEvidence artifactv2.RawEvidence,
	evidence artifactv2.Evidence,
	result artifactv2.Result,
) {
	t.Helper()
	require.Equal(t, "accepted", evidence.ObservationEvaluationStatus)
	require.NotNil(t, evidence.EvidenceBackedModelTrace)
	require.True(t, evidence.EvidenceBackedModelTrace.SourceClosed)
	require.Empty(t, evidence.Diagnostics)
	require.Empty(t, evidence.KnownGaps)
	require.Equal(t, artifactv2.ExperimentRunArtifactBinding(run), evidence.Run)
	require.Equal(t, artifactv2.RawEvidenceArtifactBinding(rawEvidence), evidence.RawEvidence)
	require.Equal(t, "succeeded", result.OperationalStatus)
	require.Equal(t, "accepted", result.ObservationEvaluationStatus)
	require.Equal(t, "applied", result.ImplementationLinkStatus)
	require.Equal(t, "satisfied", result.SemanticStatus)
	require.Equal(t, "complete", result.CleanupStatus)
	require.Equal(t, artifactv2.ExperimentRunArtifactBinding(run), result.Run)
	require.Equal(t, artifactv2.RawEvidenceArtifactBinding(rawEvidence), result.RawEvidence)
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
	require.Equal(t, []string{
		"workflow-nexus.property.clause.delivery",
		"workflow-nexus.property.clause.ownership",
		"workflow-nexus.property.clause.uniqueness",
	}, clauseDefinitionIDs(verdict.Clauses))
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

	cleanupFact, controlFact, terminalHistoryFact, participantFact := liveEvidenceFacts(t, rawEvidence)
	require.Contains(t, evidence.EvidenceBackedModelTrace.EvidenceDefinitionIDs, cleanupFact)
	propertyEvidence := propertyEvidenceDefinitionIDs(verdict.Clauses)
	require.Contains(t, propertyEvidence, controlFact)
	require.Contains(t, propertyEvidence, terminalHistoryFact)
	require.Contains(t, propertyEvidence, participantFact)
	require.NotContains(t, propertyEvidence, cleanupFact)
}

func clauseDefinitionIDs(clauses []artifactv2.SemanticClauseVerdict) []string {
	result := make([]string, len(clauses))
	for index, clause := range clauses {
		result[index] = clause.ClauseDefinitionID
	}
	return result
}

func liveEvidenceFacts(
	t *testing.T,
	rawEvidence artifactv2.RawEvidence,
) (cleanup string, control string, terminalHistory string, participant string) {
	t.Helper()
	for _, fact := range rawEvidence.Facts {
		switch fact.SourceDefinitionID {
		case umpireruntime.EvidenceSourceCleanup:
			if rawEvidenceFactHasField(fact, umpireruntime.EvidenceFieldOpenHandleCount) {
				cleanup = fact.FactDefinitionID
			}
		case umpireruntime.EvidenceSourceControlReceipt:
			control = fact.FactDefinitionID
		case umpireruntime.EvidenceSourceParticipantOutput:
			if rawEvidenceFactHasField(fact, umpireruntime.EvidenceFieldCancellationCallbackCount) {
				participant = fact.FactDefinitionID
			}
		case umpireruntime.EvidenceSourceHistory:
			if rawEvidenceStringField(fact, umpireruntime.EvidenceFieldEventType) ==
				"temporal.history.WorkflowExecutionCanceled" {
				terminalHistory = fact.FactDefinitionID
			}
		}
	}
	require.NotEmpty(t, cleanup)
	require.NotEmpty(t, control)
	require.NotEmpty(t, terminalHistory)
	require.NotEmpty(t, participant)
	return cleanup, control, terminalHistory, participant
}

func rawEvidenceFactHasField(fact artifactv2.RawEvidenceFact, definitionID string) bool {
	return slices.ContainsFunc(fact.Fields, func(field artifactv2.RawEvidenceField) bool {
		return field.FieldDefinitionID == definitionID
	})
}

func rawEvidenceStringField(fact artifactv2.RawEvidenceFact, definitionID string) string {
	for _, field := range fact.Fields {
		if field.FieldDefinitionID == definitionID {
			value, _ := field.Value.(string)
			return value
		}
	}
	return ""
}

func propertyEvidenceDefinitionIDs(clauses []artifactv2.SemanticClauseVerdict) []string {
	var result []string
	for _, clause := range clauses {
		for _, link := range clause.EvidenceLinks {
			result = append(result, link.EvidenceDefinitionIDs...)
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}
