package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		"no mode":                nil,
		"another mode":           append([]string{"assess"}, valid...),
		"a reconcile positional": append(append([]string{"reconcile"}, valid...), "extra"),
		"a positional":           append(append([]string{"run"}, valid...), "extra"),
		"no output":              {"run", "--recovery", recovery},
		"no recovery":            {"run", "--output", output},
		"a missing output":       {"run", "--output", filepath.Join(output, "absent"), "--recovery", recovery},
		"a recovery uploaded":    {"run", "--output", output, "--recovery", filepath.Join(output, "recovery.json")},
		"a recovery nowhere":     {"run", "--output", output, "--recovery", filepath.Join(t.TempDir(), "absent", "recovery.json")},
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

// Containment is checked where a path really is, through its symlinks: an output that is a symlink
// into the model, an output under a symlinked model, and a recovery record whose not-yet-existing
// path runs through a symlink into the output are all refused, and nothing is written.
func TestMainResolvesSymlinksBeforeContainment(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "model", "out"), 0o755))
	outside := t.TempDir()
	require.NoError(t, os.Symlink(filepath.Join(root, "model", "out"), filepath.Join(outside, "into-model")))
	output := filepath.Join(outside, "output")
	require.NoError(t, os.Mkdir(output, 0o755))
	require.NoError(t, os.Symlink(output, filepath.Join(outside, "into-output")))
	linkedDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(linkedDir, "out"), 0o755))
	linkedRoot := t.TempDir()
	require.NoError(t, os.Symlink(linkedDir, filepath.Join(linkedRoot, "model")))
	recovery := filepath.Join(t.TempDir(), "recovery.json")
	for name, test := range map[string]struct {
		dir, output, recovery, detail string
	}{
		"an output that links into the model":    {root, filepath.Join(outside, "into-model"), recovery, "under the model"},
		"an output under a linked model":         {linkedRoot, filepath.Join(linkedDir, "out"), recovery, "under the model"},
		"a recovery that links under the output": {root, output, filepath.Join(outside, "into-output", "recovery.json"), "under --output"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(test.dir)
			var stdout, stderr bytes.Buffer
			code := Main([]string{"run", "--output", test.output, "--recovery", test.recovery}, &stdout, &stderr, noEnvironment, controller.ProductionSeams())
			require.Equal(t, controller.ExitFailed, code)
			summary := summaryOf(t, &stdout)
			require.Equal(t, statusUsage, summary.Status)
			require.Contains(t, summary.Detail, test.detail)
			_, err := os.Stat(test.recovery)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

// resolve keeps the part of a path that does not exist yet as written, below its resolved parent.
func TestResolveKeepsTheMissingTail(t *testing.T) {
	target := t.TempDir()
	linked := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(target, linked))
	resolvedReal, err := filepath.EvalSymlinks(target)
	require.NoError(t, err)
	got, err := resolve(filepath.Join(linked, "absent", "recovery.json"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(resolvedReal, "absent", "recovery.json"), got)
	require.True(t, within(resolvedReal, got))
	require.True(t, within(resolvedReal, resolvedReal))
	require.False(t, within(resolvedReal, resolvedReal+"-sibling"))
	require.False(t, within(filepath.Join(resolvedReal, "absent"), resolvedReal))
}

// Reconcile with no recovery record has nothing to reconcile and needs no credential; the report
// is its one document on stdout.
func TestReconcileWithNoRecordIsNothingToReconcile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"reconcile", "--output", t.TempDir(), "--recovery", filepath.Join(t.TempDir(), "recovery.json")},
		&stdout, &stderr, noEnvironment, seams())
	require.Equal(t, controller.ExitAccepted, code)
	var reconciled controller.Report
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &reconciled))
	require.Equal(t, controller.StatusNothingToReconcile, reconciled.Status)
	require.Contains(t, stderr.String(), "umpire-canary reconcile: "+controller.StatusNothingToReconcile)
}

// failingWriter is a stdout that takes nothing, as a closed pipe would.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("stdout is closed") }

// A report stdout cannot take is a reporting failure: exit 3 whatever the mode concluded, with the
// document kept on stderr so nothing it says is lost.
func TestAReportStdoutCannotTakeIsExitThree(t *testing.T) {
	var stderr bytes.Buffer
	code := Main([]string{"reconcile", "--output", t.TempDir(), "--recovery", filepath.Join(t.TempDir(), "recovery.json")},
		failingWriter{}, &stderr, noEnvironment, seams())
	require.Equal(t, controller.ExitFailed, code, "a mode that concluded nothing-to-reconcile still exits 3")
	var reconciled controller.Report
	line, _, _ := strings.Cut(stderr.String(), "\n")
	require.NoError(t, json.Unmarshal([]byte(line), &reconciled), stderr.String())
	require.Equal(t, controller.StatusNothingToReconcile, reconciled.Status)
}
