package runevaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/tools/umpire/artifact"
)

func TestLocalRunEvaluationMakeTargetBuildsVerifiedPairAndPublishes(t *testing.T) {
	input := realAcceptedCallerClosureExecutionFixture(t)
	setRoot := filepath.Join(t.TempDir(), "input")
	writeExecutionSet(t, setRoot, input)
	outputRoot := t.TempDir()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	command := exec.Command(
		"make", "-C", repositoryRoot, "--no-print-directory",
		"umpire-check-local-run-evaluation", "SET="+setRoot, "OUTPUT_ROOT="+outputRoot,
	)
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	require.NoError(t, command.Run(), stderr.String())
	require.Empty(t, stderr.String())
	require.True(t, bytes.HasSuffix(stdout.Bytes(), []byte{'\n'}))
	require.Equal(t, 1, bytes.Count(stdout.Bytes(), []byte{'\n'}))

	var summary struct {
		FormatVersion               string  `json:"formatVersion"`
		RunIdentity                 string  `json:"runIdentity"`
		OperationalStatus           string  `json:"operationalStatus"`
		ObservationEvaluationStatus string  `json:"observationEvaluationStatus"`
		SemanticStatus              string  `json:"semanticStatus"`
		EvidenceArtifactChecksum    string  `json:"evidenceArtifactChecksum"`
		ResultArtifactChecksum      string  `json:"resultArtifactChecksum"`
		EvaluationOutcomeChecksum   *string `json:"evaluationOutcomeChecksum"`
		ArtifactSetChecksum         string  `json:"artifactSetChecksum"`
		ManifestSHA256              string  `json:"manifestSha256"`
		Destination                 string  `json:"destination"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &summary))
	require.Equal(t, "umpire-local-run-evaluation-summary/v2", summary.FormatVersion)
	require.Equal(t, "umpire.local.caller-closure.evaluation-fixture", summary.RunIdentity)
	require.Equal(t, "succeeded", summary.OperationalStatus)
	require.Equal(t, "accepted", summary.ObservationEvaluationStatus)
	require.Equal(t, "satisfied", summary.SemanticStatus)
	require.NotEmpty(t, summary.EvidenceArtifactChecksum)
	require.NotEmpty(t, summary.ResultArtifactChecksum)
	require.NotNil(t, summary.EvaluationOutcomeChecksum)
	require.NotEmpty(t, summary.ArtifactSetChecksum)
	require.NotEmpty(t, summary.ManifestSHA256)
	require.Equal(t, filepath.Join(
		outputRoot, "sets", strings.TrimPrefix(summary.ManifestSHA256, "sha256:"),
	), summary.Destination)
	requireExactCommandBytes(t, fmt.Sprintf(
		"{\"formatVersion\":\"umpire-local-run-evaluation-summary/v2\",\"runIdentity\":\"umpire.local.caller-closure.evaluation-fixture\",\"operationalStatus\":\"succeeded\",\"observationEvaluationStatus\":\"accepted\",\"semanticStatus\":\"satisfied\",\"evidenceArtifactChecksum\":\"sha256:5d059e9d42d5d43e92dc21a03a1270414813f37218dbb2a740e4b1b72525c77c\",\"resultArtifactChecksum\":\"sha256:941c450c55bda954fa8a449d81f8cd9820b99d488f4cc3e9ec32f48004bae8f1\",\"evaluationOutcomeChecksum\":\"sha256:873119596ec56716da7711690ff1cd63d5a3a055667facb3921616cf5f87cd35\",\"artifactSetChecksum\":\"sha256:3f312bb6224d75d92e5259492b9b2732e3e0968c111206db8ae821a308756213\",\"manifestSha256\":\"sha256:b30d42fab09e39715edd7fc8d8eb2b510c83a1709a4f3cb99ce00fd9c8693905\",\"destination\":%q}\n",
		summary.Destination,
	), stdout.String())
	loaded, err := artifact.LoadSet(summary.Destination)
	require.NoError(t, err)
	require.Equal(t, summary.ArtifactSetChecksum, loaded.Checksum())
	require.Equal(t, summary.ManifestSHA256, loaded.ManifestSHA256())
}

func TestCheckerSignalContextCancelsOnTermination(t *testing.T) {
	ctx, stop := checkerSignalContext()
	defer stop()

	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
	await.RequireTrue(t, func() bool {
		return ctx.Err() != nil
	}, time.Second, 10*time.Millisecond)
}

func TestLocalRunEvaluationMakeTargetValidatesInputsBeforeBuilding(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	directory := t.TempDir()
	file := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(file, []byte("not a directory\n"), 0o600))

	for _, testCase := range []struct {
		name      string
		variables []string
		want      string
	}{
		{name: "missing set", variables: []string{"OUTPUT_ROOT=" + directory}, want: "SET is required"},
		{name: "missing output root", variables: []string{"SET=" + directory}, want: "OUTPUT_ROOT is required"},
		{name: "set is not a directory", variables: []string{"SET=" + file, "OUTPUT_ROOT=" + directory}, want: "SET must be a directory"},
		{name: "output is not a directory", variables: []string{"SET=" + directory, "OUTPUT_ROOT=" + file}, want: "OUTPUT_ROOT must be a directory"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			arguments := []string{"-C", repositoryRoot, "--no-print-directory", "umpire-check-local-run-evaluation"}
			arguments = append(arguments, testCase.variables...)
			command := exec.Command("make", arguments...)
			command.Env = os.Environ()
			output, err := command.CombinedOutput()

			require.Error(t, err)
			require.Contains(t, string(output), testCase.want)
			require.NotContains(t, string(output), "Build completed successfully")
		})
	}
}

func TestLocalRunEvaluationMakeTargetDoesNotExposePairNameOverrides(t *testing.T) {
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	directory := t.TempDir()
	command := exec.Command(
		"make", "-C", repositoryRoot, "--no-print-directory", "-n",
		"umpire-check-local-run-evaluation", "SET="+directory, "OUTPUT_ROOT="+directory,
		"UMPIRE_RUN_EVALUATION_CHECKER=adversary-checker",
		"UMPIRE_LOCAL_RUN_EVALUATION_COMMAND=adversary-controller",
	)
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotContains(t, string(output), "adversary")
	require.Contains(t, string(output), "temporal-run-evaluation-checker")
	require.Contains(t, string(output), "umpire-local-run-evaluation")
}

func writeExecutionSet(t *testing.T, root string, admitted artifact.AdmittedSet) {
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

func requireExactCommandBytes(t *testing.T, expected string, actual string) {
	t.Helper()
	require.Equal(t, expected, actual)
}
