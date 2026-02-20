package mixedbrain

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	retryTimeout = 30 * time.Second
	omesCommit   = "8e4c1f54f3b0fb5e39d131f859c56fb2236395b1"
)

func sourceRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func buildCurrentServer(t *testing.T, outputPath string) {
	t.Helper()

	t.Log("Building current server binary...")
	cmd := exec.CommandContext(t.Context(), "go",
		"build",
		"-tags", "disable_grpc_modules",
		"-o", outputPath,
		"./cmd/server",
	)
	cmd.Dir = sourceRoot()
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build current binary failed:\n%s", out)
}

// downloadAndBuildReleaseServer finds the highest semver release tag and downloads
// the pre-built binary. We use semver sorting instead of GitHub's "latest" release
// API because the latter returns the most recently published release, not the
// highest version (e.g. a v1.25.3 patch published after v1.26.0 would be "latest").
func downloadAndBuildReleaseServer(t *testing.T, outputPath string) {
	t.Helper()

	t.Log("Resolving release tags...")
	var versions semver.Versions
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		out, err := exec.CommandContext(t.Context(), "git", "ls-remote", "--tags", "--refs", "https://github.com/temporalio/temporal.git").CombinedOutput()
		require.NoError(collect, err, "git ls-remote failed:\n%s", out)

		versions = nil
		for _, line := range strings.Split(string(out), "\n") {
			parts := strings.Fields(line)
			if len(parts) != 2 {
				continue
			}
			tag := strings.TrimPrefix(parts[1], "refs/tags/")
			v, err := semver.ParseTolerant(tag)
			if err != nil || len(v.Pre) > 0 {
				continue
			}
			versions = append(versions, v)
		}
		require.NotEmpty(collect, versions, "no valid release tags found")
	}, retryTimeout, 2*time.Second, "fetch release tags")

	slices.SortFunc(versions, func(a, b semver.Version) int { return b.Compare(a) })
	t.Logf("Found %d release tags, highest: v%s", len(versions), versions[0])

	var archivePath string
	for _, v := range versions {
		version := v.String()
		archiveName := fmt.Sprintf("temporal_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
		downloadURL := fmt.Sprintf("https://github.com/temporalio/temporal/releases/download/v%s/%s", version, archiveName)

		var statusCode int
		t.Logf("Checking GitHub for release asset %s", version)
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			headReq, err := http.NewRequestWithContext(t.Context(), http.MethodHead, downloadURL, nil)
			require.NoError(collect, err)

			resp, err := http.DefaultClient.Do(headReq)
			require.NoError(collect, err, "HEAD request for v%s failed", version)
			_ = resp.Body.Close()

			statusCode = resp.StatusCode
			require.True(collect,
				statusCode == http.StatusOK || statusCode == http.StatusNotFound,
				"HEAD for v%s returned %d", version, statusCode)
		}, retryTimeout, 2*time.Second, "check release asset for v%s", version)

		if statusCode == http.StatusNotFound {
			t.Logf("No release asset for v%s, trying next...", version)
			continue
		}

		t.Logf("Downloading %s (v%s)...", archiveName, version)
		archivePath = filepath.Join(filepath.Dir(outputPath), archiveName)
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			dlReq, err := http.NewRequestWithContext(t.Context(), http.MethodGet, downloadURL, nil)
			require.NoError(collect, err)
			dlResp, err := http.DefaultClient.Do(dlReq)
			require.NoError(collect, err)
			defer func() { _ = dlResp.Body.Close() }()
			require.Equal(collect, http.StatusOK, dlResp.StatusCode, "download returned %d", dlResp.StatusCode)

			f, err := os.Create(archivePath)
			require.NoError(collect, err)
			_, err = io.Copy(f, dlResp.Body)
			closeErr := f.Close()
			require.NoError(collect, err)
			require.NoError(collect, closeErr)
		}, retryTimeout, 2*time.Second, "download release binary")
		break
	}
	require.NotEmpty(t, archivePath, "no downloadable release found for %s/%s", runtime.GOOS, runtime.GOARCH)

	t.Log("Extracting server binary...")
	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			require.Fail(t, "binary \"temporal-server\" not found in archive")
			return
		}
		require.NoError(t, err)
		if filepath.Base(header.Name) == "temporal-server" && header.Typeflag == tar.TypeReg {
			out, err := os.Create(outputPath)
			require.NoError(t, err)
			_, err = io.Copy(out, tr)
			closeErr := out.Close()
			require.NoError(t, err)
			require.NoError(t, closeErr)
			break
		}
	}

	require.NoError(t, os.Chmod(outputPath, 0755))
	t.Logf("Release binary saved to %s", outputPath)
}

func downloadAndBuildOmes(t *testing.T, workDir string) {
	t.Helper()

	repoDir := filepath.Join(workDir, "omes")
	omesBinary := filepath.Join(workDir, "omes-bin")

	t.Logf("Cloning Omes at %s...", omesCommit[:12])
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		_ = os.RemoveAll(repoDir)
		cmd := exec.CommandContext(t.Context(), "git",
			"clone",
			"--filter=blob:none",
			"https://github.com/temporalio/omes",
			repoDir,
		)
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		require.NoError(collect, err, "git clone failed:\n%s", out)
	}, retryTimeout, 2*time.Second, "git clone omes")

	cmd := exec.CommandContext(t.Context(), "git", "checkout", omesCommit)
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git checkout %s failed:\n%s", omesCommit, out)

	t.Log("Building Omes...")
	buildCmd := exec.CommandContext(t.Context(), "go",
		"build",
		"-o", omesBinary,
		"./cmd",
	)
	buildCmd.Dir = repoDir
	out, err = buildCmd.CombinedOutput()
	require.NoError(t, err, "build Omes failed:\n%s", out)
}
