//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
)

// TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint is the vision's black-box mode on real
// bytes: a separate process, given nothing but a fixture path and two addresses, runs the Case and
// answers with an exit code. Nothing in the binary knows about the test cluster.
func TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint(t *testing.T) {
	// Build before the cluster starts: the environment bounds the test's own deadline, and a cold
	// build of the CLI is not what that budget is for.
	binary := buildUmpireRun(t)
	env := newTestpilotTestEnvironment(t,
		testcore.WithWorkerService("umpire-run --create deletes its namespace through the system delete-namespace workflow"),
		testcore.WithDynamicConfig(dynamicconfig.TransferProcessorUpdateAckInterval, 1*time.Second),
		testcore.WithDynamicConfig(dynamicconfig.VisibilityProcessorUpdateAckInterval, 1*time.Second),
	)
	namespaceName := "umpire-run-nexus-caller"
	endpointName := "umpire-run-nexus-caller-endpoint"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	arguments := []string{
		"--case", filepath.Join("..", "model", "cases", "nexus-caller-asyncCompletion-case.json"),
		"--grpc", env.FrontendGRPCAddress(),
		"--http", env.HttpAPIAddress(),
		"--namespace", namespaceName,
		"--task-queue", "umpire-run-nexus-caller-queue",
		"--nexus-endpoint", endpointName,
		"--create",
		"--timeout", "2m",
	}
	if dir := os.Getenv(umpireRepeatRunDirVariable); dir != "" {
		arguments = append(arguments, "--record", capturePath(t, dir))
	}
	command := exec.CommandContext(ctx, binary, arguments...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	started := time.Now()
	err := command.Run()
	// The process's fate is said beside every failure, never in the signature: it is what tells a
	// teardown kill from an output mismatch, and its elapsed time differs on every run.
	process := fmt.Sprintf("exit status %d, killed by the context %t, elapsed %s",
		command.ProcessState.ExitCode(), ctx.Err() != nil, time.Since(started))
	output := stdout.String() + stderr.String()
	signed := func(assertion string, check func()) {
		t.Helper()
		requireSigned(t, assertion, func(assertion string) recordedrun.Signature {
			return recordedrun.ReportSignature(t.Name(), assertion, stdout.String(), stderr.String())
		}, check)
	}

	signed("exit status", func() {
		require.NoError(t, err, "umpire-run exited non-zero (%s): %s", process, output)
		require.Equal(t, 0, command.ProcessState.ExitCode(), "%s: %s", process, output)
	})
	signed("run report", func() {
		require.Contains(t, output, "run Completed", process)
		require.Contains(t, output, "verdict Satisfied", process)
	})

	// `--create` owns what it created, so the Nexus endpoint and the namespace are gone once the
	// process exits.
	signed("endpoint deleted", func() {
		require.NotContains(t, output, "delete Nexus endpoint", process)
		describeCtx, cancelDescribe := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelDescribe()
		endpoints, err := env.OperatorClient().ListNexusEndpoints(describeCtx,
			&operatorservice.ListNexusEndpointsRequest{Name: endpointName})
		require.NoError(t, err)
		require.Empty(t, endpoints.GetEndpoints(),
			"the Nexus endpoint %s umpire-run created was not deleted on exit", endpointName)
	})

	// The delete returns once the namespace is renamed to `<name>-deleted-<id>`; only reclaiming it
	// runs on afterwards. The bounded wait keeps a slow rename a named failure, not a hang.
	signed("namespace deleted", func() {
		require.NotContains(t, output, "delete namespace", process)
		await.Requiref(t.Context(), t, func(t *await.T) {
			_, err := env.FrontendClient().DescribeNamespace(t.Context(),
				&workflowservice.DescribeNamespaceRequest{Namespace: namespaceName})
			var notFound *serviceerror.NamespaceNotFound
			require.ErrorAs(t, err, &notFound)
		}, 30*time.Second, 100*time.Millisecond,
			"the namespace %s umpire-run created was not deleted on exit", namespaceName)
	})
}

// TestTestpilotUmpireRunRejectsAnUnreachableEndpoint pins the exit code that separates
// infrastructure from a real inconclusive Verdict.
func TestTestpilotUmpireRunRejectsAnUnreachableEndpoint(t *testing.T) {
	binary := buildUmpireRun(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	command := exec.CommandContext(ctx, binary,
		"--case", testpilotFixturePath(testpilotcore.NexusCallerAsyncCompletionFixture),
		"--grpc", "127.0.0.1:1",
		"--http", "127.0.0.1:1",
		"--namespace", "umpire-run-unreachable",
		"--task-queue", "umpire-run-unreachable-queue",
		"--nexus-endpoint", "umpire-run-unreachable-endpoint",
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
	return buildUmpireCommand(t, "umpire-run")
}

// buildUmpireCommand builds one command under tools/umpire/cmd into the test's temporary
// directory.
func buildUmpireCommand(t *testing.T, name string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, "go.temporal.io/server/tools/umpire/cmd/"+name)
	build.Env = os.Environ()
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build %s: %s", name, output)
	return binary
}
