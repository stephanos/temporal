package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
)

func TestRunPublishesSatisfiedSetBeforeWritingExactSummary(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	output := admittedFixtureSet(t, 6)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	destination := filepath.Join(outputRoot, "sets", strings.TrimPrefix(output.ManifestSHA256(), "sha256:"))
	stdout := &publicationWriter{t: t, destination: destination}
	var stderr bytes.Buffer

	code := execute(
		context.Background(),
		[]string{"--set", setRoot, "--output-root", outputRoot},
		stdout,
		&stderr,
		commandDependencies{
			loadSet: func(string) (artifact.AdmittedSet, error) { return input, nil },
			check: func(admitted artifact.AdmittedSet) (artifact.AdmittedSet, error) {
				require.Equal(t, input.Identity(), admitted.Identity())
				return output, nil
			},
			publishSet: artifact.PublishSet,
		},
	)

	require.Equal(t, exitSatisfied, code)
	requireExactBytes(t, fmt.Sprintf(
		"{\"formatVersion\":\"umpire-local-run-evaluation-summary/v2\",\"runIdentity\":\"switch.run.1\",\"operationalStatus\":\"succeeded\",\"observationEvaluationStatus\":\"accepted\",\"semanticStatus\":\"satisfied\",\"evidenceArtifactChecksum\":\"sha256:adbc2ead2d0208951158fa16d558a9502cc9a864094f6eeaebc23203c56ca23b\",\"resultArtifactChecksum\":\"sha256:307e466e3ec21b919528be83313e00fd57c3f5be086b48f37226726fc8f5d4f3\",\"evaluationOutcomeChecksum\":\"sha256:f23cfbb71f27517a9d78cb243764094039439bbfe817e509c037aa8fbc285e6e\",\"artifactSetChecksum\":\"sha256:48a20a42604e2f6d483562fe886df504ca36b6423bccc86b99833210fb0da593\",\"manifestSha256\":\"sha256:cf53d048c8dcdbfe680002ad99e892cb1aebba99ed18bfb12b9d063212160da0\",\"destination\":%q}\n",
		destination,
	), stdout.String())
	require.Empty(t, stderr.String())
	_, err := artifact.LoadSet(destination)
	require.NoError(t, err)
}

func TestRunRejectsEveryNonExactArgumentGrammarBeforeChecking(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		arguments []string
	}{
		{name: "missing arguments"},
		{name: "missing output root", arguments: []string{"--set", "set"}},
		{name: "reversed flags", arguments: []string{"--output-root", "output", "--set", "set"}},
		{name: "equals form", arguments: []string{"--set=set", "--output-root=output"}},
		{name: "empty set", arguments: []string{"--set", "", "--output-root", "output"}},
		{name: "flag as output value", arguments: []string{"--set", "set", "--output-root", "--extra"}},
		{name: "positional argument", arguments: []string{"--set", "set", "--output-root", "output", "extra"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := execute(context.Background(), testCase.arguments, &stdout, &stderr,
				unreachableDependencies(t))

			require.Equal(t, exitToolingError, code)
			require.Empty(t, stdout.String())
			requireExactBytes(t,
				"{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"arguments\",\"phase\":\"admission\",\"subject\":\"arguments\",\"code\":\"umpire.run-evaluation.arguments.invalid\",\"checkingOccurred\":false,\"publicationOccurred\":false,\"runIdentity\":null,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n",
				stderr.String())
		})
	}
}

func TestRunRejectsUnsafeSetAndOutputRootsBeforeChecking(t *testing.T) {
	physicalSet := t.TempDir()
	physicalOutput := t.TempDir()
	setLink := filepath.Join(t.TempDir(), "set")
	outputLink := filepath.Join(t.TempDir(), "output")
	if err := os.Symlink(physicalSet, setLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	require.NoError(t, os.Symlink(physicalOutput, outputLink))

	for _, testCase := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{
			name:      "set symlink",
			arguments: []string{"--set", setLink, "--output-root", physicalOutput},
			want:      "{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"input\",\"phase\":\"admission\",\"subject\":\"set\",\"code\":\"umpire.run-evaluation.input.unsafe-path\",\"checkingOccurred\":false,\"publicationOccurred\":false,\"runIdentity\":null,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n",
		},
		{
			name:      "output symlink",
			arguments: []string{"--set", physicalSet, "--output-root", outputLink},
			want:      "{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"publication\",\"phase\":\"publication\",\"subject\":\"output-root\",\"code\":\"umpire.run-evaluation.publication.unsafe-root\",\"checkingOccurred\":false,\"publicationOccurred\":false,\"runIdentity\":null,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := execute(context.Background(), testCase.arguments, &stdout, &stderr,
				unreachableDependencies(t))

			require.Equal(t, exitToolingError, code)
			require.Empty(t, stdout.String())
			requireExactBytes(t, testCase.want, stderr.String())
		})
	}
}

func TestRunCanonicalizesSymlinkedAncestorsToPhysicalRoots(t *testing.T) {
	setParent := t.TempDir()
	outputParent := t.TempDir()
	physicalSet := filepath.Join(setParent, "set")
	physicalOutput := filepath.Join(outputParent, "output")
	require.NoError(t, os.Mkdir(physicalSet, 0o700))
	require.NoError(t, os.Mkdir(physicalOutput, 0o700))
	setAlias := filepath.Join(t.TempDir(), "set-parent")
	outputAlias := filepath.Join(t.TempDir(), "output-parent")
	if err := os.Symlink(setParent, setAlias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	require.NoError(t, os.Symlink(outputParent, outputAlias))
	input := admittedFixtureSet(t, 4)
	output := admittedFixtureSet(t, 6)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := execute(context.Background(), []string{
		"--set", filepath.Join(setAlias, "set"),
		"--output-root", filepath.Join(outputAlias, "output"),
	}, &stdout, &stderr, commandDependencies{
		loadSet: func(root string) (artifact.AdmittedSet, error) {
			require.Equal(t, physicalSet, root)
			return input, nil
		},
		check: func(artifact.AdmittedSet) (artifact.AdmittedSet, error) { return output, nil },
		publishSet: func(root string, admitted artifact.AdmittedSet) (string, error) {
			require.Equal(t, physicalOutput, root)
			return artifact.PublishSet(root, admitted)
		},
	})

	require.Equal(t, exitSatisfied, code)
	require.NotEmpty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestRunRejectsOverlappingSetAndOutputRootsBeforeAdmission(t *testing.T) {
	setRoot := t.TempDir()
	outputInsideSet := filepath.Join(setRoot, "output")
	require.NoError(t, os.Mkdir(outputInsideSet, 0o700))

	for _, arguments := range [][]string{
		{"--set", setRoot, "--output-root", setRoot},
		{"--set", setRoot, "--output-root", outputInsideSet},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := execute(context.Background(), arguments, &stdout, &stderr,
			unreachableDependencies(t))

		require.Equal(t, exitToolingError, code)
		require.Empty(t, stdout.String())
		requireExactBytes(t,
			"{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"publication\",\"phase\":\"publication\",\"subject\":\"output-root\",\"code\":\"umpire.run-evaluation.publication.unsafe-root\",\"checkingOccurred\":false,\"publicationOccurred\":false,\"runIdentity\":null,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n",
			stderr.String())
	}
}

func TestRunRejectsInvalidInputTreesBeforeCheckingOrPublication(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		prepare func(*testing.T, string)
	}{
		{
			name: "six-member descendant",
			prepare: func(t *testing.T, root string) {
				writeAdmittedFixtureSet(t, root, 6)
			},
		},
		{
			name: "orphan file",
			prepare: func(t *testing.T, root string) {
				writeAdmittedFixtureSet(t, root, 4)
				require.NoError(t, os.WriteFile(filepath.Join(root, "orphan.json"), []byte("{}\n"), 0o600))
			},
		},
		{
			name: "symlinked member",
			prepare: func(t *testing.T, root string) {
				writeAdmittedFixtureSet(t, root, 4)
				path := filepath.Join(root, "artifacts", "raw-evidence.json")
				target := filepath.Join(t.TempDir(), "raw-evidence.json")
				encoded, err := os.ReadFile(path)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(target, encoded, 0o600))
				require.NoError(t, os.Remove(path))
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			},
		},
		{
			name: "malformed manifest",
			prepare: func(t *testing.T, root string) {
				writeAdmittedFixtureSet(t, root, 4)
				require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{}\n"), 0o600))
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			setRoot := filepath.Join(t.TempDir(), "set")
			testCase.prepare(t, setRoot)
			outputRoot := t.TempDir()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			dependencies := unreachableDependencies(t)
			dependencies.loadSet = loadInputSet

			code := execute(context.Background(),
				[]string{"--set", setRoot, "--output-root", outputRoot},
				&stdout, &stderr, dependencies)

			require.Equal(t, exitToolingError, code)
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(),
				"\"kind\":\"input\",\"phase\":\"admission\",\"subject\":\"set\"")
			require.Contains(t, stderr.String(), "\"checkingOccurred\":false,\"publicationOccurred\":false")
			require.NoDirExists(t, filepath.Join(outputRoot, "sets"))
		})
	}
}

func TestRunReportsToolingFailuresAtTheirOwningBoundary(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	runIdentity := "switch.run.1"

	for _, testCase := range []struct {
		name       string
		configure  func(*commandDependencies)
		wantStderr string
	}{
		{
			name: "input admission",
			configure: func(dependencies *commandDependencies) {
				dependencies.loadSet = func(string) (artifact.AdmittedSet, error) {
					return artifact.AdmittedSet{}, artifact.ErrClosure
				}
			},
			wantStderr: "{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"input\",\"phase\":\"admission\",\"subject\":\"set\",\"code\":\"umpire.run-evaluation.input.closure\",\"checkingOccurred\":false,\"publicationOccurred\":false,\"runIdentity\":null,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n",
		},
		{
			name: "checker",
			configure: func(dependencies *commandDependencies) {
				dependencies.check = func(artifact.AdmittedSet) (artifact.AdmittedSet, error) {
					return artifact.AdmittedSet{}, classifiedTestError{
						kind: "checker", phase: "Observation Evaluation",
						code: "umpire.run-evaluation.checker.failed",
					}
				}
			},
			wantStderr: fmt.Sprintf("{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"checker\",\"phase\":\"Observation Evaluation\",\"subject\":\"set\",\"code\":\"umpire.run-evaluation.checker.failed\",\"checkingOccurred\":true,\"publicationOccurred\":false,\"runIdentity\":%q,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n", runIdentity),
		},
		{
			name: "output invariant",
			configure: func(dependencies *commandDependencies) {
				dependencies.check = func(artifact.AdmittedSet) (artifact.AdmittedSet, error) {
					return artifact.AdmittedSet{}, classifiedTestError{
						kind: "output-invariant", phase: "construction",
						code: "umpire.run-evaluation.result.invalid",
					}
				}
			},
			wantStderr: fmt.Sprintf("{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"output-invariant\",\"phase\":\"construction\",\"subject\":\"set\",\"code\":\"umpire.run-evaluation.result.invalid\",\"checkingOccurred\":true,\"publicationOccurred\":false,\"runIdentity\":%q,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n", runIdentity),
		},
		{
			name: "publication",
			configure: func(dependencies *commandDependencies) {
				dependencies.publishSet = func(string, artifact.AdmittedSet) (string, error) {
					return "", errors.New("read-only")
				}
			},
			wantStderr: fmt.Sprintf("{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"publication\",\"phase\":\"publication\",\"subject\":\"output-root\",\"code\":\"umpire.run-evaluation.publication.failed\",\"checkingOccurred\":true,\"publicationOccurred\":false,\"runIdentity\":%q,\"artifactSetChecksum\":null,\"manifestSha256\":null,\"destination\":null}\n", runIdentity),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			dependencies := commandDependencies{
				loadSet: func(string) (artifact.AdmittedSet, error) { return input, nil },
				check: func(artifact.AdmittedSet) (artifact.AdmittedSet, error) {
					return admittedFixtureSet(t, 6), nil
				},
				publishSet: artifact.PublishSet,
			}
			testCase.configure(&dependencies)
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := execute(context.Background(),
				[]string{"--set", setRoot, "--output-root", outputRoot},
				&stdout, &stderr, dependencies)

			require.Equal(t, exitToolingError, code)
			require.Empty(t, stdout.String())
			requireExactBytes(t, testCase.wantStderr, stderr.String())
			require.NoDirExists(t, filepath.Join(outputRoot, "sets"))
		})
	}
}

func TestRunReportsSemanticInputRejectionBeforeInstalledPairResolution(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := execute(context.Background(),
		[]string{"--set", setRoot, "--output-root", outputRoot},
		&stdout, &stderr, commandDependencies{
			loadSet: func(string) (artifact.AdmittedSet, error) { return input, nil },
			check: func(artifact.AdmittedSet) (artifact.AdmittedSet, error) {
				return artifact.AdmittedSet{}, classifiedTestError{
					kind: "input", phase: "generated-view",
					code: "umpire.run-evaluation.input.unsupported-profile",
				}
			},
			publishSet: artifact.PublishSet,
		})

	require.Equal(t, exitToolingError, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(),
		`"kind":"input","phase":"generated-view","subject":"set"`)
	require.Contains(t, stderr.String(),
		`"code":"umpire.run-evaluation.input.unsupported-profile"`)
}

func TestRunReportsBrokenStdoutAfterKeepingAuthoritativePublication(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	output := admittedFixtureSet(t, 6)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	destination := filepath.Join(outputRoot, "sets", strings.TrimPrefix(output.ManifestSHA256(), "sha256:"))
	var stderr bytes.Buffer

	code := execute(context.Background(),
		[]string{"--set", setRoot, "--output-root", outputRoot},
		errorWriter{}, &stderr, commandDependencies{
			loadSet:    func(string) (artifact.AdmittedSet, error) { return input, nil },
			check:      func(artifact.AdmittedSet) (artifact.AdmittedSet, error) { return output, nil },
			publishSet: artifact.PublishSet,
		})

	require.Equal(t, exitToolingError, code)
	requireExactBytes(t, fmt.Sprintf(
		"{\"formatVersion\":\"umpire-local-run-evaluation-error/v2\",\"kind\":\"reporting\",\"phase\":\"reporting\",\"subject\":\"stdout\",\"code\":\"umpire.run-evaluation.reporting.failed\",\"checkingOccurred\":true,\"publicationOccurred\":true,\"runIdentity\":\"switch.run.1\",\"artifactSetChecksum\":\"sha256:48a20a42604e2f6d483562fe886df504ca36b6423bccc86b99833210fb0da593\",\"manifestSha256\":\"sha256:cf53d048c8dcdbfe680002ad99e892cb1aebba99ed18bfb12b9d063212160da0\",\"destination\":%q}\n",
		destination,
	), stderr.String())
	_, err := artifact.LoadSet(destination)
	require.NoError(t, err)
}

func TestRunReturnsTheSameRevalidatedDestinationForAnIdenticalRetry(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	output := admittedFixtureSet(t, 6)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	dependencies := commandDependencies{
		loadSet:    func(string) (artifact.AdmittedSet, error) { return input, nil },
		check:      func(artifact.AdmittedSet) (artifact.AdmittedSet, error) { return output, nil },
		publishSet: artifact.PublishSet,
	}
	arguments := []string{"--set", setRoot, "--output-root", outputRoot}

	var first bytes.Buffer
	var firstError bytes.Buffer
	require.Equal(t, exitSatisfied,
		execute(context.Background(), arguments, &first, &firstError, dependencies))
	var second bytes.Buffer
	var secondError bytes.Buffer
	require.Equal(t, exitSatisfied,
		execute(context.Background(), arguments, &second, &secondError, dependencies))

	require.Equal(t, first.String(), second.String())
	require.Empty(t, firstError.String())
	require.Empty(t, secondError.String())
	require.Len(t, directoryEntries(t, filepath.Join(outputRoot, "sets")), 1)
}

func TestRunDoesNotRepairAConflictingImmutableDestination(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	output := admittedFixtureSet(t, 6)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	dependencies := commandDependencies{
		loadSet:    func(string) (artifact.AdmittedSet, error) { return input, nil },
		check:      func(artifact.AdmittedSet) (artifact.AdmittedSet, error) { return output, nil },
		publishSet: artifact.PublishSet,
	}
	arguments := []string{"--set", setRoot, "--output-root", outputRoot}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	require.Equal(t, exitSatisfied,
		execute(context.Background(), arguments, &stdout, &stderr, dependencies))
	destination := filepath.Join(outputRoot, "sets", strings.TrimPrefix(output.ManifestSHA256(), "sha256:"))
	conflictingPath := filepath.Join(destination, "artifacts", "result.json")
	conflict := []byte("conflicting bytes\n")
	require.NoError(t, os.WriteFile(conflictingPath, conflict, 0o600))
	stdout.Reset()
	stderr.Reset()

	code := execute(context.Background(), arguments, &stdout, &stderr, dependencies)

	require.Equal(t, exitToolingError, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "\"kind\":\"publication\"")
	got, err := os.ReadFile(conflictingPath)
	require.NoError(t, err)
	require.Equal(t, conflict, got)
}

func TestRunReportsActualOutputPermissionFailureWithoutPartialDestination(t *testing.T) {
	input := admittedFixtureSet(t, 4)
	output := admittedFixtureSet(t, 6)
	setRoot := t.TempDir()
	outputRoot := t.TempDir()
	require.NoError(t, os.Chmod(outputRoot, 0o500))
	t.Cleanup(func() { _ = os.Chmod(outputRoot, 0o700) })
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := execute(context.Background(),
		[]string{"--set", setRoot, "--output-root", outputRoot},
		&stdout, &stderr, commandDependencies{
			loadSet:    func(string) (artifact.AdmittedSet, error) { return input, nil },
			check:      func(artifact.AdmittedSet) (artifact.AdmittedSet, error) { return output, nil },
			publishSet: artifact.PublishSet,
		})

	require.Equal(t, exitToolingError, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "\"kind\":\"publication\"")
	require.NoDirExists(t, filepath.Join(outputRoot, "sets"))
}

func TestRunKeepsToolingStatusWhenStderrIsUnavailable(t *testing.T) {
	var stdout bytes.Buffer

	code := execute(context.Background(), nil, &stdout, errorWriter{}, unreachableDependencies(t))

	require.Equal(t, exitToolingError, code)
	require.Empty(t, stdout.String())
}

func TestSummaryExitStatusRequiresAllThreeSuccessDimensions(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		operational  string
		observation  string
		semantic     string
		wantExitCode int
	}{
		{name: "satisfied", operational: "succeeded", observation: "accepted", semantic: "satisfied", wantExitCode: exitSatisfied},
		{name: "violated", operational: "succeeded", observation: "accepted", semantic: "violated", wantExitCode: exitNotSatisfied},
		{name: "nonaccepted", operational: "succeeded", observation: "unknown", semantic: "incomplete", wantExitCode: exitNotSatisfied},
		{name: "operational failed", operational: "failed", observation: "accepted", semantic: "satisfied", wantExitCode: exitNotSatisfied},
		{name: "operational incomplete", operational: "incomplete", observation: "accepted", semantic: "satisfied", wantExitCode: exitNotSatisfied},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.wantExitCode, summaryExitStatus(commandSummary{
				OperationalStatus:           testCase.operational,
				ObservationEvaluationStatus: testCase.observation,
				SemanticStatus:              testCase.semantic,
			}))
		})
	}
}

type publicationWriter struct {
	t           *testing.T
	destination string
	bytes.Buffer
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("broken writer")
}

type classifiedTestError struct {
	kind  string
	phase string
	code  string
}

func (failure classifiedTestError) Error() string { return failure.code }
func (failure classifiedTestError) Kind() string  { return failure.kind }
func (failure classifiedTestError) Phase() string { return failure.phase }
func (failure classifiedTestError) Code() string  { return failure.code }

func (writer *publicationWriter) Write(value []byte) (int, error) {
	writer.t.Helper()
	_, err := artifact.LoadSet(writer.destination)
	require.NoError(writer.t, err)
	return writer.Buffer.Write(value)
}

func admittedFixtureSet(t *testing.T, memberCount int) artifact.AdmittedSet {
	t.Helper()
	root := filepath.Join("..", "..", "artifact", "testdata", "valid-run-evaluation-set")
	paths := []string{
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"artifacts/experiment-run.json",
		"artifacts/raw-evidence.json",
		"artifacts/evidence.json",
		"artifacts/result.json",
	}
	members := make([]artifact.SetMember, memberCount)
	for index, path := range paths[:memberCount] {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		require.NoError(t, err)
		members[index] = artifact.SetMember{Path: path, Encoded: encoded}
	}
	admitted, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	return admitted
}

func writeAdmittedFixtureSet(t *testing.T, root string, memberCount int) {
	t.Helper()
	admitted := admittedFixtureSet(t, memberCount)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "artifacts"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), admitted.ManifestBytes(), 0o600))
	fixtureRoot := filepath.Join("..", "..", "artifact", "testdata", "valid-run-evaluation-set")
	paths := []string{
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"artifacts/experiment-run.json",
		"artifacts/raw-evidence.json",
		"artifacts/evidence.json",
		"artifacts/result.json",
	}
	for _, path := range paths[:memberCount] {
		encoded, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(path)))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), encoded, 0o600))
	}
}

func directoryEntries(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	require.NoError(t, err)
	return entries
}

func unreachableDependencies(t *testing.T) commandDependencies {
	t.Helper()
	unreachable := func() {
		t.Helper()
		require.FailNow(t, "command dependency reached")
	}
	return commandDependencies{
		loadSet: func(string) (artifact.AdmittedSet, error) {
			unreachable()
			return artifact.AdmittedSet{}, nil
		},
		check: func(artifact.AdmittedSet) (artifact.AdmittedSet, error) {
			unreachable()
			return artifact.AdmittedSet{}, nil
		},
		publishSet: func(string, artifact.AdmittedSet) (string, error) {
			unreachable()
			return "", nil
		},
	}
}

func requireExactBytes(t *testing.T, expected string, actual string) {
	t.Helper()
	require.Equal(t, expected, actual)
}

var _ io.Writer = (*publicationWriter)(nil)
