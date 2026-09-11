//go:build test_dep && integration

package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/operatorservice/v1"
)

// TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint is the vision's black-box mode on real
// bytes: a separate process, given nothing but a fixture path and two addresses, runs the Case and
// answers with an exit code. Nothing in the binary knows about the test cluster.
func TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint(t *testing.T) {
	// Build before the cluster starts: the environment bounds the test's own deadline, and a cold
	// build of the CLI is not what that budget is for.
	binary := buildUmpireRun(t)
	env := newTestpilotTestEnvironment(t)
	endpointName := "umpire-run-async-nexus-endpoint"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	command := exec.CommandContext(ctx, binary,
		"--case", filepath.Join("testcore", "testpilot", "testdata", "async-nexus-case.json"),
		"--grpc", env.FrontendGRPCAddress(),
		"--http", env.HttpAPIAddress(),
		"--namespace", "umpire-run-async-nexus",
		"--task-queue", "umpire-run-async-nexus-queue",
		"--nexus-endpoint", endpointName,
		"--create",
		"--timeout", "2m",
	)
	output, err := command.CombinedOutput()

	require.NoError(t, err, "umpire-run exited non-zero: %s", output)
	require.Equal(t, 0, command.ProcessState.ExitCode(), "%s", output)
	require.Contains(t, string(output), "run Completed")
	require.Contains(t, string(output), "verdict Satisfied")

	// `--create` owns what it created, so the Nexus endpoint is gone once the process exits.
	require.NotContains(t, string(output), "delete Nexus endpoint")
	describeCtx, cancelDescribe := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDescribe()
	endpoints, err := env.OperatorClient().ListNexusEndpoints(describeCtx,
		&operatorservice.ListNexusEndpointsRequest{Name: endpointName})
	require.NoError(t, err)
	require.Empty(t, endpoints.GetEndpoints(),
		"the Nexus endpoint umpire-run created was not deleted on exit")

	// The namespace deletion the CLI also issues is a system worker workflow, and this functional
	// cluster does not run that service, so its completion is not asserted here. The cleanup
	// ordering and the deletion call itself are pinned by the provisioning package's unit tests.
}

// TestTestpilotUmpireRunRejectsAnUnreachableEndpoint pins the exit code that separates
// infrastructure from a real inconclusive Verdict.
func TestTestpilotUmpireRunRejectsAnUnreachableEndpoint(t *testing.T) {
	binary := buildUmpireRun(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	command := exec.CommandContext(ctx, binary,
		"--case", filepath.Join("testcore", "testpilot", "testdata", "async-nexus-case.json"),
		"--grpc", "127.0.0.1:1",
		"--http", "127.0.0.1:1",
		"--namespace", "umpire-run-unreachable",
		"--task-queue", "umpire-run-unreachable-queue",
		"--create",
		"--timeout", "20s",
	)
	output, err := command.CombinedOutput()

	require.Error(t, err)
	require.Equal(t, 3, command.ProcessState.ExitCode(), "%s", output)
	require.NotContains(t, string(output), "verdict ")
}

// buildUmpireRun builds the CLI the way a developer would, so the test exercises the real binary
// rather than an in-process call.
func buildUmpireRun(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "umpire-run")
	build := exec.Command("go", "build", "-o", binary, "go.temporal.io/server/tools/umpire/cmd/umpire-run")
	build.Env = os.Environ()
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build umpire-run: %s", output)
	return binary
}
