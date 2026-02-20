package mixedbrain

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testDuration() time.Duration {
	if v := os.Getenv("MIXED_BRAIN_TEST_DURATION"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
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

func TestMixedBrain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping mixed brain test in short mode")
	}

	tmpDir := t.TempDir()

	logRoot := logDir(t)
	root := sourceRoot()
	dynConfigPath := filepath.Join(root, "config", "dynamicconfig", "development-sql.yaml")

	currentBinary := filepath.Join(tmpDir, "temporal-server-current")
	releaseBinary := filepath.Join(tmpDir, "temporal-server-release")
	omesBinary := filepath.Join(tmpDir, "omes-bin")

	t.Run("setup", func(t *testing.T) {
		t.Run("build-current", func(t *testing.T) {
			t.Parallel()
			buildCurrentServer(t, currentBinary)
		})
		t.Run("download-release", func(t *testing.T) {
			t.Parallel()
			downloadAndBuildLatestServer(t, releaseBinary)
		})
		t.Run("build-omes", func(t *testing.T) {
			t.Parallel()
			downloadAndBuildOmes(t, tmpDir)
		})
	})
	if t.Failed() {
		return
	}

	portsCurrent := allocatePortSet()
	portsRelease := allocatePortSet()

	configCurrent := filepath.Join(tmpDir, "config-current")
	configRelease := filepath.Join(tmpDir, "config-release")
	generateConfig(t, configCurrent, portsCurrent, portsCurrent, tmpDir, dynConfigPath)
	generateConfig(t, configRelease, portsRelease, portsCurrent, tmpDir, dynConfigPath)

	// Start current server first and wait for it to initialize the database schema
	// before starting the release server. SQLite schema setup takes an exclusive lock
	// that would cause SQLITE_BUSY if both servers start simultaneously.
	procCurrent := startServerProcess(t, "current", currentBinary, configCurrent, filepath.Join(logRoot, "mixedbrain_process-current.log"))
	t.Cleanup(procCurrent.stop)
	waitForServerHealth(t, portsCurrent.frontendAddr(), 30*time.Second)
	t.Log("Current server is healthy")

	procRelease := startServerProcess(t, "release", releaseBinary, configRelease, filepath.Join(logRoot, "mixedbrain_process-release.log"))
	t.Cleanup(procRelease.stop)
	waitForServerHealth(t, portsRelease.frontendAddr(), 30*time.Second)
	t.Log("Release server is healthy")
	if t.Failed() {
		return
	}

	registerDefaultNamespace(t, portsCurrent.frontendAddr())

	runID := fmt.Sprintf("mixed-brain-%d", time.Now().Unix())
	nexusEndpoint := "mixed-brain-nexus"
	createNexusEndpoint(t, portsCurrent.frontendAddr(), nexusEndpoint, "default", "omes-"+runID)

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

func runOmes(t *testing.T, binary, serverAddr, logPath string, duration time.Duration, runID, nexusEndpoint string) {
	t.Helper()
	t.Logf("Running Omes throughput_stress for %v against %s", duration, serverAddr)

	// Omes registers search attributes on startup.
	// Retry if Omes fails due to search attribute not being ready yet.
	require.Eventually(t, func() bool {
		logFile, err := os.Create(logPath)
		if err != nil {
			return false
		}
		var buf bytes.Buffer
		cmd := exec.CommandContext(t.Context(), binary,
			"run-scenario-with-worker",
			"--scenario", "throughput_stress",
			"--language", "go",
			"--server-address", serverAddr,
			"--duration", duration.String(),
			"--run-id", runID,
			"--max-concurrent", "5",
			"--option", "internal-iterations=10",
			"--option", "nexus-endpoint="+nexusEndpoint,
		)
		cmd.Stdout = logFile
		cmd.Stderr = io.MultiWriter(logFile, &buf)

		err = cmd.Run()
		_ = logFile.Close()
		if err != nil && strings.Contains(buf.String(), "no mapping defined for search attribute") {
			t.Log("Omes failed due to search attributes not ready, retrying...")
			return false
		}
		require.NoError(t, err, "Omes scenario failed, check %s", logPath)
		return true
	}, duration+2*time.Minute, 5*time.Second, "Omes scenario failed to start")
}
