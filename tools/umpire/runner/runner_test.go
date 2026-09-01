package runner_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/nexus"
)

var expectedCallerClosureInput = runner.InputBinding{
	ArtifactSetIdentity:                     "umpire.artifact-set.ed3605976ba999ec8e166d4309247e2b711fee18f4a421cfb8c6dc037344f1a2",
	ArtifactSetChecksum:                     "sha256:074356889cda0296b13152f87e57d7b980d76125329a9014ceb5321c3f5bda7b",
	ManifestSHA256:                          "sha256:f381da231395b8fec738837535a8bb8da0dd227a08e3d60bf9c2bda620c46b14",
	ExperimentArtifactChecksum:              "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
	ExperimentBehaviorFingerprint:           "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
	RuntimeConfigurationArtifactChecksum:    "sha256:21b4f7d0db2f68f939df901c2c5d146b1be3e45e55ad6cc171445fda5f29c1d5",
	RuntimeConfigurationBehaviorFingerprint: "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67",
	AuthorityRequiredCapabilityDefinitionIDs: []string{
		"umpire.runtime.capability.complete-workflow-history-read",
		"umpire.runtime.capability.ephemeral-server-lifecycle",
		"umpire.runtime.capability.sdk-worker-lifecycle",
	},
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
	requireRunnerFailureClassification(
		t, err, "input-binding", "umpire.runner.input-binding.invalid",
	)
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
	requireRunnerFailureClassification(
		t, err, "input-binding", "umpire.runner.input-binding.drift",
	)
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
	executable, ok := adapter.admitted.Executable()
	require.True(t, ok)
	require.EqualValues(t, "3584", executable.RuntimeConfiguration().PhaseLimits[2].MaxRecords)
}

func TestRunClassifiesAdapterPreflightAsNotStarted(t *testing.T) {
	input := admitCallerClosureSet(t)
	adapter := &recordingAdapter{checkErr: errors.New("checked adapter reached")}

	_, err := runner.Run(
		context.Background(),
		input,
		expectedCallerClosureInput,
		"umpire.generated.runner.preflight-classification-1",
		adapter,
	)

	require.Error(t, err)
	requireExecutionOccurred(t, err, false)
}

func TestRunClassifiesParticipantConstructionAsNotStarted(t *testing.T) {
	input := admitCallerClosureSet(t)

	_, err := runner.Run(
		context.Background(),
		input,
		expectedCallerClosureInput,
		"umpire.generated.runner.participant-classification-1",
		participantFailureAdapter{Binding: nexus.Binding{}},
	)

	require.Error(t, err)
	requireExecutionOccurred(t, err, false)
}

func TestRunRejectsLimitNPlusOneBeforeAdapterConstruction(t *testing.T) {
	adapter := &recordingAdapter{}
	_, err := runner.Run(
		context.Background(),
		mutateCallerClosureConfiguration(t, func(configuration *artifactv2.RuntimeConfiguration) {
			configuration.PhaseLimits[2].MaxRecords = artifactv2.Natural("3585")
		}),
		expectedCallerClosureInput,
		"umpire.generated.runner.limit-rejected-1",
		adapter,
	)

	require.ErrorContains(t, err, "generated input binding")
	requireRunnerFailureClassification(
		t, err, "input-binding", "umpire.runner.input-binding.drift",
	)
	require.Equal(t, 0, adapter.checkCalls)
}

func TestRunRejectsAuthorityLeakBeforeParticipantConstruction(t *testing.T) {
	for _, test := range []struct {
		name         string
		capabilities []string
	}{
		{
			name: "endpoint credential and arbitrary executable",
			capabilities: []string{
				"umpire.runtime.capability.arbitrary-executable",
				"umpire.runtime.capability.credential",
				"umpire.runtime.capability.external-endpoint",
			},
		},
		{
			name: "endpoint plugin and undeclared network",
			capabilities: []string{
				"umpire.runtime.capability.external-endpoint",
				"umpire.runtime.capability.plugin",
				"umpire.runtime.capability.undeclared-network",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			adapter := &authorityLeakAdapter{capabilities: test.capabilities}
			_, err := runner.Run(
				context.Background(),
				admitCallerClosureSet(t),
				expectedCallerClosureInput,
				"umpire.generated.runner.authority-rejected-1",
				adapter,
			)

			require.ErrorContains(t, err, "generated authority binding")
			requireRunnerFailureClassification(
				t, err, "authority-binding", "umpire.runner.authority-binding.unauthorized",
			)
			require.Equal(t, 0, adapter.participantCalls)
			require.Equal(t, 0, adapter.environmentCalls)
		})
	}
}

func TestExecutionSurfaceExposesOnlyTheDigestBoundRunner(t *testing.T) {
	require.NotContains(t, topLevelFunctions(t, "runner.go"), "RunChecked")
	require.NotContains(t, topLevelFunctions(
		t, filepath.Join("..", "runtime", "engine.go"),
	), "Run")
	require.NotContains(t, topLevelFunctions(
		t, filepath.Join("..", "temporal", "nexus", "output.go"),
	), "Run")
	require.Equal(t, []string{
		filepath.Join("runevaluation", "result.go"),
		filepath.Join("runevaluation", "run_evaluation.go"),
		filepath.Join("runner", "runner.go"),
	}, goFilesImporting(t, "..", "go.temporal.io/server/tools/umpire/internal/runtimeengine"))
}

func requireRunnerFailureClassification(
	t *testing.T,
	err error,
	wantKind string,
	wantCode string,
) {
	t.Helper()
	var classified interface {
		error
		Kind() string
		Phase() string
		Code() string
	}
	require.ErrorAs(t, err, &classified)
	require.Equal(t, []string{wantKind, "admission", wantCode}, []string{
		classified.Kind(), classified.Phase(), classified.Code(),
	})
}

type recordingAdapter struct {
	checkCalls int
	admitted   artifact.AdmittedSet
	checkErr   error
}

type authorityLeakAdapter struct {
	capabilities     []string
	participantCalls int
	environmentCalls int
}

type participantFailureAdapter struct {
	nexus.Binding
}

func (participantFailureAdapter) NewParticipant(
	umpireruntime.CheckedRunRequest,
) (umpireruntime.Participant, error) {
	return nil, errors.New("participant construction failed")
}

func requireExecutionOccurred(t *testing.T, err error, want bool) {
	t.Helper()
	var classified interface {
		ExecutionOccurred() bool
	}
	require.ErrorAs(t, err, &classified)
	require.Equal(t, want, classified.ExecutionOccurred())
}

func (a *authorityLeakAdapter) CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	canonical, err := nexus.CheckRequest(admitted, runIdentity)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	authority := canonical.Authority()
	leaked, err := umpireruntime.NewAuthority(
		authority.DefinitionID(),
		authority.Version(),
		authority.BehaviorFingerprint(),
		authority.ConfigurationDefinitionID(),
		authority.ConfigurationBehaviorFingerprint(),
		a.capabilities,
		[]string{},
		authority.PhaseLimits(),
		authority.Seed(),
		authority.Attempt(),
		authority.ParticipantDefinitionID(),
		authority.ProtocolDefinitionID(),
		authority.ProtocolVersion(),
		authority.ParticipantCount(),
		authority.ProgramCount(),
		authority.Program(),
	)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	return umpireruntime.CheckRequest(
		admitted,
		leaked,
		runIdentity,
		canonical.Seed(),
		canonical.Attempt(),
	)
}

func (a *authorityLeakAdapter) EnvironmentFactory() umpireruntime.EnvironmentFactory {
	a.environmentCalls++
	panic("authority leak must fail before environment construction")
}

func (a *authorityLeakAdapter) NewParticipant(
	umpireruntime.CheckedRunRequest,
) (umpireruntime.Participant, error) {
	a.participantCalls++
	return nil, errors.New("authority leak reached participant construction")
}

func (*authorityLeakAdapter) ValidateOutput(
	umpireruntime.CheckedRunRequest,
	umpireruntime.Output,
) error {
	panic("authority leak must not produce output")
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

func mutateCallerClosureConfiguration(
	t *testing.T,
	mutate func(*artifactv2.RuntimeConfiguration),
) artifact.AdmittedSet {
	t.Helper()
	input := admitCallerClosureSet(t)
	executable, ok := input.Executable()
	require.True(t, ok)
	configuration := executable.RuntimeConfiguration()
	configuration.PhaseLimits = append([]artifactv2.PhaseLimit{}, configuration.PhaseLimits...)
	mutate(&configuration)
	configuration, err := artifactv2.SealRuntimeConfiguration(configuration)
	require.NoError(t, err)
	experimentBytes, err := artifact.EncodeExperimentV2(executable.Experiment())
	require.NoError(t, err)
	configurationBytes, err := artifact.EncodeRuntimeConfigurationV2(configuration)
	require.NoError(t, err)
	mutated, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: configurationBytes},
	})
	require.NoError(t, err)
	return mutated
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

func goFilesImporting(t *testing.T, root string, importPath string) []string {
	t.Helper()
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imported := range parsed.Imports {
			if imported.Path.Value == `"`+importPath+`"` {
				relative, err := filepath.Rel(root, path)
				require.NoError(t, err)
				paths = append(paths, relative)
			}
		}
		return nil
	})
	require.NoError(t, err)
	sort.Strings(paths)
	return paths
}
