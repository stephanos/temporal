package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func TestInstallTemporalAndDiscover(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "schema/sqlite"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "tests/gomadfunctional"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "schema/sqlite/schema.sql"), []byte("CREATE TABLE example(id INTEGER);"), 0600))
	compiler, helper := filepath.Join(root, "go"), filepath.Join(root, "helper")
	for _, path := range []string{compiler, helper} {
		require.NoError(t, os.WriteFile(path, []byte("pinned executable"), 0700))
	}
	output := filepath.Join(root, "installation.json")
	args := []string{"install", "--temporal-checkout", root, "--temporal-package", "./tests/gomadfunctional", "--compiler", compiler, "--helper", helper, "--runtime-root", root, "--cache", filepath.Join(root, "cache"), "--output", output}
	var stdout, stderr bytes.Buffer
	require.Zero(t, run(args, &stdout, &stderr), stderr.String())
	description, err := installation.Resolve(installation.Spec{ExplicitManifest: output})
	require.NoError(t, err)
	config, err := wasi.TemporalConfig(root, "/workspace/tests/gomadfunctional")
	require.NoError(t, err)
	config.NetworkV2 = description.Manifest.Options.Config.NetworkV2
	require.NotNil(t, config.NetworkV2)
	want, err := json.Marshal(config)
	require.NoError(t, err)
	actual, err := json.Marshal(description.Manifest.Options.Config)
	require.NoError(t, err)
	require.Equal(t, string(want), string(actual))
	require.Equal(t, backend.TemporalGuestBuildTags(), description.Manifest.Options.BuildTags)
	stdout.Reset()
	status, handled := Run([]string{"discover"}, &stdout, &stderr, func() (installation.Description, error) { return description, nil })
	require.True(t, handled)
	require.Zero(t, status, stderr.String())
	require.Contains(t, stdout.String(), wasi.CooperativeProfile)
	for _, input := range []struct{ name, path string }{{"compiler", compiler}, {"helper", helper}} {
		require.NoError(t, os.Remove(input.path))
		stderr.Reset()
		require.Equal(t, 2, run(args, &stdout, &stderr))
		require.Contains(t, stderr.String(), "temporal "+input.name)
		require.NoError(t, os.WriteFile(input.path, []byte("pinned executable"), 0700))
	}
	for _, test := range []struct{ name, value string }{{"GOMAD_TEMPORAL_CONFIG", "ignored.json"}, {"GOMAD_WASM_INSTALLATION", "ignored.json"}} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.name, test.value)
			stderr.Reset()
			require.Equal(t, 2, run(args, &stdout, &stderr))
			require.Contains(t, stderr.String(), "unset "+test.name)
		})
	}
}

func TestInstallTemporalRefusesMissingInputs(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		message string
	}{
		{"package", []string{"--temporal-checkout", "/missing"}, "--temporal-package"},
		{"compiler", []string{"--temporal-checkout", "/missing", "--temporal-package", "tests", "--helper", "/helper", "--cache", "/cache"}, "compiler"},
		{"helper", []string{"--temporal-checkout", "/missing", "--temporal-package", "tests", "--compiler", "/go", "--cache", "/cache"}, "helper"},
		{"schema", []string{"--temporal-checkout", "/missing", "--temporal-package", "tests", "--compiler", "/go", "--helper", "/helper", "--cache", "/cache", "--runtime-root", "/runtime"}, "schema/sqlite"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append([]string{"install", "--output", filepath.Join(t.TempDir(), "installation.json")}, test.args...)
			require.Equal(t, 2, run(args, &stdout, &stderr))
			require.Contains(t, stderr.String(), test.message)
			require.Empty(t, stdout.String())
		})
	}
}
