package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
)

func TestCheckAcceptsEveryRetainedArtifactFamilyWithoutMutatingInput(t *testing.T) {
	for _, test := range []struct {
		family  string
		fixture string
	}{
		{family: "umpire-experiment/v2", fixture: "switch-experiment-v2.json"},
		{family: "umpire-runtime-configuration/v2", fixture: "runtime-configuration-v2.json"},
		{family: "umpire-experiment-run/v2", fixture: "experiment-run-v2.json"},
		{family: "umpire-raw-evidence/v2", fixture: "raw-evidence-v2.json"},
		{family: "umpire-evidence/v2", fixture: "evidence-v2.json"},
		{family: "umpire-result/v2", fixture: "result-v2.json"},
	} {
		t.Run(test.family, func(t *testing.T) {
			path := artifactFixturePath(test.fixture)
			before := snapshotPath(t, path)

			code, stdout, stderr := runCommand(
				"check", "--family", test.family, "--artifact", path,
			)

			require.Equal(t, exitSuccess, code)
			require.Empty(t, stdout)
			require.Empty(t, stderr)
			require.Equal(t, before, snapshotPath(t, path))
		})
	}
}

func TestCheckRejectsNoncanonicalBytesWithoutMutatingInput(t *testing.T) {
	canonical, err := os.ReadFile(artifactFixturePath("switch-experiment-v2.json"))
	require.NoError(t, err)
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, canonical))
	path := filepath.Join(t.TempDir(), "experiment.json")
	require.NoError(t, os.WriteFile(path, compact.Bytes(), 0o640))
	before := snapshotPath(t, path)

	code, stdout, stderr := runCommand(
		"check", "--family", "umpire-experiment/v2", "--artifact", path,
	)

	require.Equal(t, exitFailure, code)
	require.Empty(t, stdout)
	require.Equal(t,
		"umpire-artifact: noncanonical: document is not exact deterministic pretty JSON\n",
		stderr,
	)
	require.Equal(t, before, snapshotPath(t, path))
}

func TestCheckSetAcceptsCompleteSetWithoutMutationOrPublication(t *testing.T) {
	parent := t.TempDir()
	setPath := filepath.Join(parent, "input")
	writeEvaluationSet(t, setPath)
	before := snapshotTree(t, parent)

	code, stdout, stderr := runCommand("check-set", "--set", setPath)

	require.Equal(t, exitSuccess, code)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.Equal(t, before, snapshotTree(t, parent))
}

func TestCheckSetRejectsCompactMemberWithoutMutatingInput(t *testing.T) {
	setPath := filepath.Join(t.TempDir(), "input")
	writeEvaluationSet(t, setPath)
	memberPath := filepath.Join(setPath, "artifacts", "evidence.json")
	canonical, err := os.ReadFile(memberPath)
	require.NoError(t, err)
	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, canonical))
	require.NoError(t, os.WriteFile(memberPath, compact.Bytes(), 0o640))
	before := snapshotTree(t, setPath)

	code, stdout, stderr := runCommand("check-set", "--set", setPath)

	require.Equal(t, exitFailure, code)
	require.Empty(t, stdout)
	require.Equal(t,
		"umpire-artifact: noncanonical: document is not exact deterministic pretty JSON\n",
		stderr,
	)
	require.Equal(t, before, snapshotTree(t, setPath))
}

func TestCheckSetRejectsUnexpectedFilesAndSymlinks(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(*testing.T, string)
		want    string
	}{
		{
			name: "unexpected file",
			prepare: func(t *testing.T, root string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(root, "extra.json"), []byte("{}\n"), 0o600))
			},
			want: "umpire-artifact: closure: artifact set contains unexpected files\n",
		},
		{
			name: "symlinked manifest",
			prepare: func(t *testing.T, root string) {
				t.Helper()
				manifestPath := filepath.Join(root, "manifest.json")
				target := filepath.Join(t.TempDir(), "manifest.json")
				encoded, err := os.ReadFile(manifestPath)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(target, encoded, 0o600))
				require.NoError(t, os.Remove(manifestPath))
				if err := os.Symlink(target, manifestPath); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			},
			want: "umpire-artifact: read Artifact set: \"manifest.json\" is not a regular file\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			setPath := filepath.Join(t.TempDir(), "input")
			writeEvaluationSet(t, setPath)
			test.prepare(t, setPath)
			before := snapshotTree(t, setPath)

			code, stdout, stderr := runCommand("check-set", "--set", setPath)

			require.Equal(t, exitFailure, code)
			require.Empty(t, stdout)
			require.Equal(t, test.want, stderr)
			require.Equal(t, before, snapshotTree(t, setPath))
		})
	}
}

func TestCommandUsageErrorsUseExitTwoAndStderrOnly(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing subcommand",
			want: "umpire-artifact: expected check or check-set subcommand\n",
		},
		{
			name: "unknown subcommand",
			args: []string{"publish"},
			want: "umpire-artifact: expected check or check-set subcommand\n",
		},
		{
			name: "missing family",
			args: []string{"check", "--artifact", "artifact.json"},
			want: "umpire-artifact: --family is required\n",
		},
		{
			name: "unsupported family",
			args: []string{"check", "--family", "experiment", "--artifact", "artifact.json"},
			want: "umpire-artifact: unsupported --family \"experiment\"\n",
		},
		{
			name: "missing artifact",
			args: []string{"check", "--family", "umpire-experiment/v2"},
			want: "umpire-artifact: --artifact is required\n",
		},
		{
			name: "missing set",
			args: []string{"check-set"},
			want: "umpire-artifact: --set is required\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runCommand(test.args...)

			require.Equal(t, exitUsage, code)
			require.Empty(t, stdout)
			require.Equal(t, test.want, stderr)
		})
	}
}

type pathSnapshot struct {
	mode    fs.FileMode
	modTime time.Time
	bytes   []byte
}

func runCommand(arguments ...string) (code int, stdoutText string, stderrText string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = run(arguments, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func artifactFixturePath(name string) string {
	return filepath.Join("..", "..", "artifact", "testdata", name)
}

func writeEvaluationSet(t *testing.T, root string) {
	t.Helper()
	fixtures := []struct {
		path string
		name string
	}{
		{path: "artifacts/experiment.json", name: "switch-experiment-v2.json"},
		{path: "artifacts/runtime-configuration.json", name: "runtime-configuration-v2.json"},
		{path: "artifacts/experiment-run.json", name: "experiment-run-v2.json"},
		{path: "artifacts/raw-evidence.json", name: "raw-evidence-v2.json"},
		{path: "artifacts/evidence.json", name: "evidence-v2.json"},
		{path: "artifacts/result.json", name: "result-v2.json"},
	}
	members := make([]artifact.SetMember, len(fixtures))
	for index, fixture := range fixtures {
		encoded, err := os.ReadFile(artifactFixturePath(fixture.name))
		require.NoError(t, err)
		members[index] = artifact.SetMember{Path: fixture.path, Encoded: encoded}
	}
	admitted, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "artifacts"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), admitted.ManifestBytes(), 0o640))
	for _, member := range members {
		require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(member.Path)), member.Encoded, 0o640))
	}
}

func snapshotPath(t *testing.T, path string) pathSnapshot {
	t.Helper()
	info, err := os.Lstat(path)
	require.NoError(t, err)
	snapshot := pathSnapshot{mode: info.Mode(), modTime: info.ModTime()}
	if info.Mode().IsRegular() {
		snapshot.bytes, err = os.ReadFile(path)
		require.NoError(t, err)
	}
	return snapshot
}

func snapshotTree(t *testing.T, root string) map[string]pathSnapshot {
	t.Helper()
	snapshots := make(map[string]pathSnapshot)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		require.NoError(t, err)
		relative, err := filepath.Rel(root, path)
		require.NoError(t, err)
		snapshots[filepath.ToSlash(relative)] = snapshotPath(t, path)
		return nil
	}))
	return snapshots
}
