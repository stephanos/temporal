package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/canary/controller"
)

func noEnvironment(string) (string, bool) { return "", false }

func summaryOf(t *testing.T, stdout *bytes.Buffer) controller.Summary {
	t.Helper()
	var summary controller.Summary
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &summary), stdout.String())
	return summary
}

// The command line is closed: one mode and two flags. Anything else, and every option that would
// name a Case, target, Driver, checker, retry, executable, endpoint, credential or release, is a
// usage refusal with exit 3 before anything is read.
func TestMainRefusesEveryOtherCommandLine(t *testing.T) {
	output := t.TempDir()
	recovery := filepath.Join(t.TempDir(), "recovery.json")
	valid := []string{"--output", output, "--recovery", recovery}
	cases := map[string][]string{
		"no mode":             nil,
		"another mode":        append([]string{"assess"}, valid...),
		"a positional":        append(append([]string{"run"}, valid...), "extra"),
		"no output":           {"run", "--recovery", recovery},
		"no recovery":         {"run", "--output", output},
		"a missing output":    {"run", "--output", filepath.Join(output, "absent"), "--recovery", recovery},
		"a recovery uploaded": {"run", "--output", output, "--recovery", filepath.Join(output, "recovery.json")},
		"a recovery nowhere":  {"run", "--output", output, "--recovery", filepath.Join(t.TempDir(), "absent", "recovery.json")},
	}
	for _, forbidden := range []string{"case", "target", "driver", "checker", "retry", "executable", "endpoint", "credential", "release", "profile", "policy", "grpc", "namespace"} {
		cases["--"+forbidden] = append(append([]string{"run"}, valid...), "--"+forbidden, "x")
	}
	for name, arguments := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Main(arguments, &stdout, &stderr, noEnvironment, controller.ProductionSeams())
			require.Equal(t, controller.ExitFailed, code)
			require.Equal(t, statusUsage, summaryOf(t, &stdout).Status)
			_, err := os.Stat(recovery)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

// An output under the model is refused: the model never receives the canary's output.
func TestMainRefusesAnOutputUnderTheModel(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "model", "out"), 0o755))
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "--output", filepath.Join("model", "out"), "--recovery", filepath.Join(root, "recovery.json")},
		&stdout, &stderr, noEnvironment, controller.ProductionSeams())
	require.Equal(t, controller.ExitFailed, code)
	summary := summaryOf(t, &stdout)
	require.Equal(t, statusUsage, summary.Status)
	require.Contains(t, summary.Detail, "under the model")
}

// The untagged build with no credential in its environment refuses as authority-unavailable before
// it reads the target or writes a record, and says so on stdout and stderr.
func TestTheUntaggedBuildNeedsACredential(t *testing.T) {
	recovery := filepath.Join(t.TempDir(), "recovery.json")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"run", "--output", t.TempDir(), "--recovery", recovery}, &stdout, &stderr, noEnvironment, seams())
	require.Equal(t, controller.ExitFailed, code)
	summary := summaryOf(t, &stdout)
	require.Equal(t, controller.StatusAuthorityUnavailable, summary.Status)
	require.Empty(t, summary.Iterations)
	require.Contains(t, stderr.String(), controller.StatusAuthorityUnavailable)
	_, err := os.Stat(recovery)
	require.ErrorIs(t, err, os.ErrNotExist)
}
