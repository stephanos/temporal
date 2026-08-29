package runner_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

var expectedCallerClosureInput = runner.InputBinding{
	ArtifactSetIdentity:                     "umpire.artifact-set.ed3605976ba999ec8e166d4309247e2b711fee18f4a421cfb8c6dc037344f1a2",
	ArtifactSetChecksum:                     "sha256:074356889cda0296b13152f87e57d7b980d76125329a9014ceb5321c3f5bda7b",
	ManifestSHA256:                          "sha256:f381da231395b8fec738837535a8bb8da0dd227a08e3d60bf9c2bda620c46b14",
	ExperimentArtifactChecksum:              "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
	ExperimentBehaviorFingerprint:           "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
	RuntimeConfigurationArtifactChecksum:    "sha256:21b4f7d0db2f68f939df901c2c5d146b1be3e45e55ad6cc171445fda5f29c1d5",
	RuntimeConfigurationBehaviorFingerprint: "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67",
}

func TestRunRejectsIncompleteInputBeforeAdapterConstruction(t *testing.T) {
	adapter := &recordingAdapter{}

	_, err := runner.Run(
		context.Background(),
		artifact.AdmittedSet{},
		expectedCallerClosureInput,
		"umpire.generated.runner.incomplete-1",
		adapter,
	)

	require.ErrorContains(t, err, "exact two-member executable set")
	require.Equal(t, 0, adapter.checkCalls)
}

func TestRunRejectsGeneratedDigestDriftBeforeAdapterConstruction(t *testing.T) {
	input := admitCallerClosureSet(t)
	binding := expectedCallerClosureInput
	binding.ExperimentArtifactChecksum = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	adapter := &recordingAdapter{}

	_, err := runner.Run(
		context.Background(),
		input,
		binding,
		"umpire.generated.runner.digest-drift-1",
		adapter,
	)

	require.ErrorContains(t, err, "generated input binding")
	require.Equal(t, 0, adapter.checkCalls)
}

func TestRunPassesTheExactAdmittedSetToTheAdapter(t *testing.T) {
	input := admitCallerClosureSet(t)
	adapter := &recordingAdapter{checkErr: errors.New("checked adapter reached")}

	_, err := runner.Run(
		context.Background(),
		input,
		expectedCallerClosureInput,
		"umpire.generated.runner.bound-1",
		adapter,
	)

	require.ErrorContains(t, err, "checked adapter reached")
	require.Equal(t, 1, adapter.checkCalls)
	require.Equal(t, input.Identity(), adapter.admitted.Identity())
}

func TestExecutionSurfaceExposesOnlyTheDigestBoundRunner(t *testing.T) {
	require.NotContains(t, topLevelFunctions(t, "runner.go"), "RunChecked")
	require.NotContains(t, topLevelFunctions(
		t, filepath.Join("..", "temporal", "nexus", "output.go"),
	), "Run")
}

type recordingAdapter struct {
	checkCalls int
	admitted   artifact.AdmittedSet
	checkErr   error
}

func (a *recordingAdapter) CheckRequest(
	admitted artifact.AdmittedSet,
	_ string,
) (umpireruntime.CheckedRunRequest, error) {
	a.checkCalls++
	a.admitted = admitted
	return umpireruntime.CheckedRunRequest{}, a.checkErr
}

func (*recordingAdapter) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	panic("environment factory must not be constructed")
}

func (*recordingAdapter) NewParticipant(
	umpireruntime.CheckedRunRequest,
) (umpireruntime.Participant, error) {
	panic("participant must not be constructed")
}

func (*recordingAdapter) ValidateOutput(
	umpireruntime.CheckedRunRequest,
	umpireruntime.Output,
) error {
	panic("output must not be validated")
}

func admitCallerClosureSet(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	root := filepath.Join("..", "temporal", "nexus", "testdata", "caller-closure-input-set")
	files := make(map[string][]byte, 3)
	for _, path := range []string{
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"manifest.json",
	} {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		require.NoError(t, err)
		files[path] = encoded
	}
	admitted, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	return admitted
}

func topLevelFunctions(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	require.NoError(t, err)
	names := []string{}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil {
			names = append(names, function.Name.Name)
		}
	}
	return names
}
