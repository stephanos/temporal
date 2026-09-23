package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/campaign"
	"go.temporal.io/server/tools/umpire/replay"
)

func deploymentFlags() []string {
	return []string{"--grpc", "127.0.0.1:7233", "--http", "127.0.0.1:7243", "--namespace", "ns", "--task-queue", "tq"}
}

// subjectFiles writes a Case and a recorded Run to a temporary directory: the Case is not canonical,
// so admission rejects it before anything opens.
func subjectFiles(t *testing.T) (casePath, runPath string) {
	t.Helper()
	directory := t.TempDir()
	casePath = filepath.Join(directory, "case.json")
	runPath = filepath.Join(directory, "run.json")
	require.NoError(t, os.WriteFile(casePath, []byte(`{ "caseId" :  "c" }`), 0o644))
	require.NoError(t, os.WriteFile(runPath, []byte(`{"identity":{},"run":{}}`), 0o644))
	return casePath, runPath
}

// unopened is an environment whose deployment and bridge must never be reached.
func unopened(t *testing.T) environmentFor {
	return func(configuration config, stderr io.Writer) replay.Environment {
		environment := deploymentEnvironment(configuration, stderr)
		environment.StartBridge = func(context.Context) (*replay.Bridge, error) {
			t.Fatal("the bridge was started")
			return nil, nil
		}
		environment.OpenBinder = func(context.Context) (campaign.Binder, func(context.Context) error, error) {
			t.Fatal("the deployment was opened")
			return nil, nil, nil
		}
		return environment
	}
}

// Every refusal of the command line exits 3 before anything is read or opened.
func TestRunRefusesTheCommandLineBeforeOpening(t *testing.T) {
	casePath, runPath := subjectFiles(t)
	subject := []string{"--case", casePath, "--run", runPath, "--set", "s"}
	modelRoot := t.TempDir()
	for name, arguments := range map[string][]string{
		"no subcommand":            {"--case", casePath},
		"missing case":             append([]string{"run", "--run", runPath, "--set", "s", "--query", "q"}, deploymentFlags()...),
		"missing set":              append([]string{"run", "--case", casePath, "--run", runPath, "--query", "q"}, deploymentFlags()...),
		"neither query nor target": append(append([]string{"run"}, subject...), deploymentFlags()...),
		"both query and target":    append(append([]string{"run", "--query", "q", "--target", "t"}, subject...), deploymentFlags()...),
		"missing deployment":       append([]string{"run", "--query", "q"}, subject...),
		"positional argument":      append(append([]string{"run", "--query", "q"}, subject...), append(deploymentFlags(), "extra")...),
		"root under the model": append(append([]string{"run", "--query", "q", "--model-root", modelRoot,
			"--promotion-root", filepath.Join(modelRoot, "proposals")}, subject...), deploymentFlags()...),
		"a limit flag": append(append([]string{"run", "--query", "q", "--max-runs", "20"}, subject...), deploymentFlags()...),
		"unreadable case": append([]string{"run", "--query", "q", "--case", filepath.Join(t.TempDir(), "absent.json"),
			"--run", runPath, "--set", "s"}, deploymentFlags()...),
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(arguments, &stdout, &stderr, func(config, io.Writer) replay.Environment {
				t.Fatal("the environment was built")
				return replay.Environment{}
			})
			require.Equal(t, replay.ExitToolingFailure, code)
			require.Empty(t, stdout.String())
			require.NotEmpty(t, stderr.String())
		})
	}
}

// A rejected subject is reported whole with its reason, exits 3, and opens nothing.
func TestRunReportsARejectedSubjectWithoutOpening(t *testing.T) {
	casePath, runPath := subjectFiles(t)
	var stdout, stderr bytes.Buffer
	code := Run(append([]string{"run", "--case", casePath, "--run", runPath, "--set", "s", "--query", "q"}, deploymentFlags()...),
		&stdout, &stderr, unopened(t))
	require.Equal(t, replay.ExitToolingFailure, code, stderr.String())
	var report replay.Report
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &report))
	require.Equal(t, replay.StatusRejected, report.Admission.Status)
	require.Equal(t, replay.ReasonNoncanonical, report.Admission.Reason)
	require.Equal(t, replay.StatusNotRun, report.SemanticReplay.Status)
	require.Equal(t, replay.StatusNothing, report.Cleanup.Status)
	require.Contains(t, stderr.String(), "rejected (noncanonical)")
	require.Equal(t, byte('\n'), stdout.Bytes()[stdout.Len()-1])
}
