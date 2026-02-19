package mixedbrain

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func buildCurrentBinary(t *testing.T, outputPath string) {
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

func downloadAndBuildOmes(t *testing.T, workDir string) {
	t.Helper()

	repoDir := filepath.Join(workDir, "omes")
	omesBinary := filepath.Join(workDir, "omes-bin")

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

	t.Log("Building omes...")
	buildCmd := exec.CommandContext(t.Context(), "go",
		"build",
		"-o", omesBinary,
		"./cmd",
	)
	buildCmd.Dir = repoDir
	out, err = buildCmd.CombinedOutput()
	require.NoError(t, err, "build omes failed:\n%s", out)
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func downloadLatestRelease(t *testing.T, outputPath string) {
	t.Helper()
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		t.Log("Resolving latest release...")

		ctx := t.Context()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/temporalio/temporal/releases/latest", nil)
		require.NoError(collect, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(collect, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(collect, http.StatusOK, resp.StatusCode, "GitHub API returned %d", resp.StatusCode)

		var release ghRelease
		require.NoError(collect, json.NewDecoder(resp.Body).Decode(&release))
		require.NotEmpty(collect, release.TagName, "release tag is empty")

		version := release.TagName
		if len(version) > 0 && version[0] == 'v' {
			version = version[1:]
		}
		archiveName := fmt.Sprintf("temporal_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)

		var downloadURL string
		for _, asset := range release.Assets {
			if asset.Name == archiveName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
		require.NotEmpty(collect, downloadURL, "no release asset found for %s", archiveName)

		t.Logf("Downloading %s (%s)...", archiveName, release.TagName)
		dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		require.NoError(collect, err)
		archiveResp, err := http.DefaultClient.Do(dlReq)
		require.NoError(collect, err)
		defer func() { _ = archiveResp.Body.Close() }()
		require.Equal(collect, http.StatusOK, archiveResp.StatusCode, "download returned %d", archiveResp.StatusCode)

		gzr, err := gzip.NewReader(archiveResp.Body)
		require.NoError(collect, err)
		defer func() { _ = gzr.Close() }()

		tr := tar.NewReader(gzr)
		for {
			header, err := tr.Next()
			if errors.Is(err, io.EOF) {
				require.Fail(collect, "binary \"temporal-server\" not found in archive")
				return
			}
			require.NoError(collect, err)
			if filepath.Base(header.Name) == "temporal-server" && header.Typeflag == tar.TypeReg {
				out, err := os.Create(outputPath)
				require.NoError(collect, err)
				_, err = io.Copy(out, tr)
				closeErr := out.Close()
				require.NoError(collect, err)
				require.NoError(collect, closeErr)
				break
			}
		}

		require.NoError(collect, os.Chmod(outputPath, 0755))
		t.Logf("Release binary saved to %s", outputPath)
	}, retryTimeout, 2*time.Second, "download latest release")
}
