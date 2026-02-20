package mixedbrain

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func testDuration() time.Duration {
	if v := os.Getenv("MIXED_BRAIN_TEST_DURATION"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			panic(fmt.Sprintf("invalid MIXED_BRAIN_TEST_DURATION %q: %v", v, err))
		}
		return d
	}
	return 30 * time.Second // locally we want only a smoke test to ensure it works
}

func logDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("TEST_OUTPUT_ROOT")
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "temporal-test-output")
	}
	require.NoError(t, os.MkdirAll(dir, 0755))
	return dir
}

// TestMixedBrain starts two servers in parallel, one using the current branch's binary
// and the other using the latest release binary. It then runs Omes throughput_stress
// to ensure that the mixed brain works correctly.
// Uses SQLite locally; and a dedicated database in CI for better concurrency.
func TestMixedBrain(t *testing.T) {
	tmpDir := t.TempDir()
	logRoot := logDir(t)

	currentBinary := filepath.Join(tmpDir, "temporal-server-current")
	releaseBinary := filepath.Join(tmpDir, "temporal-server-release")
	omesBinary := filepath.Join(tmpDir, "omes-bin")

	t.Run("setup", func(t *testing.T) {
		t.Run("build current server", func(t *testing.T) {
			t.Parallel()
			buildServer(t, sourceRoot(), currentBinary)
		})
		t.Run("download and build release server", func(t *testing.T) {
			t.Parallel()
			downloadAndBuildReleaseServer(t, releaseBinary)
		})
		t.Run("download and build Omes", func(t *testing.T) {
			t.Parallel()
			downloadAndBuildOmes(t, tmpDir)
		})
	})
	if t.Failed() {
		return
	}

	var portsCurrent, portsRelease portSet
	if os.Getenv("CI") != "" {
		portsCurrent = portSetA
		portsRelease = portSetB
	} else {
		portsCurrent = newRandPortSet()
		portsRelease = newRandPortSet()
	}

	configCurrent := generateConfig(t, tmpDir, portsCurrent, portsCurrent)
	configRelease := generateConfig(t, tmpDir, portsRelease, portsCurrent)

	procCurrent := startServerProcess(t, "current", currentBinary, configCurrent, filepath.Join(logRoot, "mixedbrain_process-current.log"))
	t.Cleanup(procCurrent.stop)

	conn, err := grpc.NewClient(portsCurrent.frontendAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// This ensures the current server is fully booted before starting the release server.
	registerDefaultNamespace(t, conn)

	procRelease := startServerProcess(t, "release", releaseBinary, configRelease, filepath.Join(logRoot, "mixedbrain_process-release.log"))
	t.Cleanup(procRelease.stop)

	waitForClusterFormation(t, conn, 90*time.Second, portsCurrent, portsRelease)
	t.Log("Cluster formed with both servers")

	runID := fmt.Sprintf("mixed-brain-%d", time.Now().Unix())
	nexusEndpoint := "mixed-brain-nexus"
	createNexusEndpoint(t, conn, nexusEndpoint, "default", "omes-"+runID)

	proxy := startFrontendProxy(t, portsCurrent.frontendAddr(), portsRelease.frontendAddr())
	t.Cleanup(proxy.stop)

	runOmes(t, omesBinary, proxy.addr(), filepath.Join(logRoot, "mixedbrain_omes.log"), testDuration(), runID, nexusEndpoint)

	procCurrent.requireAlive(t)
	procRelease.requireAlive(t)

	for i, backend := range []string{"current", "release"} {
		count := proxy.connCount[i].Load()
		t.Logf("Proxy connections to %s: %d", backend, count)
		require.Positive(t, count, "expected proxy to route traffic to %s server", backend)
	}

	t.Log("Mixed brain test passed")
}

// runOmes runs Omes throughput stress scenario.
// Retries if Omes fails due to search attribute not being ready yet.
// Deducts elapsed time from duration on retry so total wall time stays bounded.
func runOmes(t *testing.T, binary, serverAddr, logPath string, duration time.Duration, runID, nexusEndpoint string) {
	t.Helper()
	t.Logf("Running Omes throughput_stress for %v against %s", duration, serverAddr)

	started := time.Now()
	for {
		remaining := duration - time.Since(started)
		require.Greater(t, remaining, 10*time.Second, "Omes never started successfully, check %s", logPath)

		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		require.NoError(t, err)
		var buf bytes.Buffer
		cmd := exec.CommandContext(t.Context(), binary,
			"run-scenario-with-worker",
			"--scenario", "throughput_stress",
			"--language", "go",
			"--server-address", serverAddr,
			"--duration", remaining.String(),
			"--run-id", runID,
			"--max-concurrent", "5",
			"--option", "internal-iterations=10",
			"--option", "nexus-endpoint="+nexusEndpoint,
		)
		cmd.Stdout = logFile
		cmd.Stderr = io.MultiWriter(logFile, &buf)
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = 15 * time.Second

		err = cmd.Run()
		_ = logFile.Close()
		if err != nil && strings.Contains(buf.String(), "no mapping defined for search attribute") {
			t.Log("Omes failed due to search attributes not ready, retrying...")
			continue
		}
		require.NoError(t, err, "Omes scenario failed, check %s", logPath)
		return
	}
}
