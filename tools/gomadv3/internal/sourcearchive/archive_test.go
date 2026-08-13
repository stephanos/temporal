package sourcearchive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestEnsureDownloadsAndReusesVerifiedArchive(t *testing.T) {
	archive := testArchive(t, archiveEntry{name: "go/VERSION", contents: "go1.26.4\n", mode: 0o644})
	digest := fmt.Sprintf("%x", sha256.Sum256(archive))
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		_, _ = response.Write(archive)
	}))
	defer server.Close()

	config := Config{CacheDir: t.TempDir(), Name: "go1.26.4.src.tar.gz", URL: server.URL, SHA256: digest}
	first, err := Ensure(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Ensure(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || requests.Load() != 1 {
		t.Fatalf("Ensure() paths = %q, %q; requests = %d", first, second, requests.Load())
	}
	contents, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, archive) {
		t.Fatal("cached archive differs from downloaded archive")
	}
}

func TestEnsureReplacesCorruptCacheOnlyAfterVerifiedDownload(t *testing.T) {
	archive := testArchive(t, archiveEntry{name: "go/VERSION", contents: "go1.26.4\n", mode: 0o644})
	digest := fmt.Sprintf("%x", sha256.Sum256(archive))
	cache := t.TempDir()
	path := filepath.Join(cache, "go1.26.4.src.tar.gz")
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write(archive)
	}))
	defer server.Close()

	if _, err := Ensure(context.Background(), Config{CacheDir: cache, Name: filepath.Base(path), URL: server.URL, SHA256: digest}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, archive) {
		t.Fatal("corrupt cache was not replaced")
	}
}

func TestEnsureChecksumFailureDoesNotPublishPartialArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("corrupt"))
	}))
	defer server.Close()
	cache := t.TempDir()
	name := "go1.26.4.src.tar.gz"
	_, err := Ensure(context.Background(), Config{
		CacheDir: cache, Name: name, URL: server.URL,
		SHA256: strings.Repeat("a", 64), Retries: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Ensure() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(cache, name)); !os.IsNotExist(err) {
		t.Fatalf("cache path exists after checksum failure: %v", err)
	}
	entries, err := os.ReadDir(cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("partial cache entries remain: %v", entries)
	}
}

func TestExtractMaterializesRegularGoTree(t *testing.T) {
	archive := testArchive(t,
		archiveEntry{name: "go/", directory: true, mode: 0o755},
		archiveEntry{name: "go/src/", directory: true, mode: 0o755},
		archiveEntry{name: "go/src/make.bash", contents: "#!/bin/bash\n", mode: 0o755},
	)
	archivePath := filepath.Join(t.TempDir(), "go.src.tar.gz")
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "source")
	if err := Extract(context.Background(), archivePath, destination); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(destination, "go", "src", "make.bash"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "#!/bin/bash\n" {
		t.Fatalf("extracted contents = %q", contents)
	}
	info, err := os.Stat(filepath.Join(destination, "go", "src", "make.bash"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("extracted mode = %o", info.Mode().Perm())
	}
}

func TestExtractRejectsUnsafeOrUnsupportedEntries(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry archiveEntry
	}{
		{name: "traversal", entry: archiveEntry{name: "go/../escape", contents: "escape", mode: 0o644}},
		{name: "absolute", entry: archiveEntry{name: "/go/escape", contents: "escape", mode: 0o644}},
		{name: "wrong-root", entry: archiveEntry{name: "other/file", contents: "escape", mode: 0o644}},
		{name: "symlink", entry: archiveEntry{name: "go/link", link: "../escape", mode: 0o777}},
	} {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "go.src.tar.gz")
			if err := os.WriteFile(archivePath, testArchive(t, test.entry), 0o600); err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(t.TempDir(), "source")
			if err := Extract(context.Background(), archivePath, destination); err == nil {
				t.Fatal("Extract() accepted unsafe archive")
			}
		})
	}
}

func TestExtractRejectsDuplicatePathsAndNonemptyDestination(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "go.src.tar.gz")
	archive := testArchive(t,
		archiveEntry{name: "go/VERSION", contents: "first", mode: 0o644},
		archiveEntry{name: "go/VERSION", contents: "second", mode: 0o644},
	)
	if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Extract(context.Background(), archivePath, filepath.Join(t.TempDir(), "duplicate")); err == nil {
		t.Fatal("Extract() accepted duplicate archive paths")
	}

	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(destination, "existing"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Extract(context.Background(), archivePath, destination); err == nil {
		t.Fatal("Extract() accepted a nonempty destination")
	}
}

type archiveEntry struct {
	name      string
	contents  string
	link      string
	mode      int64
	directory bool
}

func testArchive(t *testing.T, entries ...archiveEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	zipper := gzip.NewWriter(&output)
	archive := tar.NewWriter(zipper)
	for _, entry := range entries {
		typeFlag := byte(tar.TypeReg)
		if entry.directory {
			typeFlag = tar.TypeDir
		} else if entry.link != "" {
			typeFlag = tar.TypeSymlink
		}
		header := &tar.Header{Name: entry.name, Linkname: entry.link, Mode: entry.mode, Size: int64(len(entry.contents)), Typeflag: typeFlag}
		if typeFlag != tar.TypeReg {
			header.Size = 0
		}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if typeFlag == tar.TypeReg {
			if _, err := archive.Write([]byte(entry.contents)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
