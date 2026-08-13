package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPublishesCheckedUpgradeEvidence(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	baseline := []byte(`{"manifest_version":"go1.26.3-darwin-arm64-v1","intercepts":[{"package":"os","symbol":"Open","signature":"func(name string) (*File, error)","hook":"oldHook"}]}`)
	err := Run(context.Background(), Options{
		Root: root, Output: output, BaselineManifest: baseline,
		Gates: []Gate{{Name: "unit", Command: []string{"/usr/bin/printf", "gate passed\n"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var dossier Dossier
	if err := json.Unmarshal(contents, &dossier); err != nil {
		t.Fatal(err)
	}
	if dossier.Schema != "gomadv3.upgrade-dossier/v1" || !dossier.Qualified || dossier.Version.GoVersion != "go1.26.4" {
		t.Fatalf("dossier identity = %#v", dossier)
	}
	if !strings.Contains(dossier.UpstreamPatch.Diff, "src/runtime/proc.go") || dossier.UpstreamPatch.SHA256 == "" {
		t.Fatalf("upstream patch evidence = %#v", dossier.UpstreamPatch)
	}
	if len(dossier.BoundaryDiff.Added) != 1 || dossier.BoundaryDiff.Added[0] != "os.OpenFile" || len(dossier.BoundaryDiff.Removed) != 1 || dossier.BoundaryDiff.Removed[0] != "os.Open" {
		t.Fatalf("boundary diff = %#v", dossier.BoundaryDiff)
	}
	if !dossier.OverlayCollision.Checked || len(dossier.OverlayCollision.Collisions) != 0 {
		t.Fatalf("overlay collision evidence = %#v", dossier.OverlayCollision)
	}
	if len(dossier.Gates) != 1 || dossier.Gates[0].Status != "passed" || dossier.Gates[0].Output != "gate passed\n" {
		t.Fatalf("gate evidence = %#v", dossier.Gates)
	}
	if dossier.RetainedCorpus.Status != "not-configured" {
		t.Fatalf("retained corpus evidence = %#v", dossier.RetainedCorpus)
	}
}

func TestRunPublishesFailedGateEvidence(t *testing.T) {
	root := writeUpgradeFixture(t, false)
	output := filepath.Join(root, ".toolchain", "upgrade-dossier.json")
	err := Run(context.Background(), Options{
		Root: root, Output: output,
		Gates: []Gate{{Name: "failure", Command: []string{"/bin/sh", "-c", "printf failed-output; exit 23"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "qualification gate failure") {
		t.Fatalf("Run() error = %v", err)
	}
	contents, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var dossier Dossier
	if unmarshalErr := json.Unmarshal(contents, &dossier); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	if dossier.Qualified || len(dossier.Gates) != 1 || dossier.Gates[0].Status != "failed" || dossier.Gates[0].ExitCode != 23 || dossier.Gates[0].Output != "failed-output" {
		t.Fatalf("failed dossier = %#v", dossier)
	}
}

func TestRunRejectsOverlayCollisionBeforeGates(t *testing.T) {
	root := writeUpgradeFixture(t, true)
	err := Run(context.Background(), Options{Root: root, Output: filepath.Join(root, "dossier.json")})
	if err == nil || !strings.Contains(err.Error(), "overlay collides with upstream source") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestLoadCorpusRejectsUncheckedOrUnqualifiedJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "corpus.json")
	if err := os.WriteFile(path, []byte("{\"qualified\":false,\"schema\":\"gomadv3.qualification-set-report/v1\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCorpus(root, path); err == nil || !strings.Contains(err.Error(), "qualification set") {
		t.Fatalf("loadCorpus() error = %v", err)
	}
}

func writeUpgradeFixture(t *testing.T, collide bool) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"boundary", "overlay/src/os", ".toolchain/downloads"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	patch := "diff --git a/src/runtime/proc.go b/src/runtime/proc.go\n--- a/src/runtime/proc.go\n+++ b/src/runtime/proc.go\n"
	if err := os.WriteFile(filepath.Join(root, "go1.26.4.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "overlay", "src", "os", "gomad.go"), []byte("package os\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema_version":1,"manifest_version":"go1.26.4-darwin-arm64-v1","go_version":"go1.26.4","platforms":["darwin/arm64"],"intercepts":[{"package":"os","symbol":"OpenFile","signature":"func(name string, flag int, perm FileMode) (*File, error)","hook":"gomadInterceptOpenFile"}]}`
	if err := os.WriteFile(filepath.Join(root, "boundary", "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "expected-intercepts-go1.26.4.txt"), []byte("os.OpenFile -> os.gomadInterceptOpenFile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, ".toolchain", "downloads", "go1.26.4.src.tar.gz")
	archive := createArchive(t, archivePath, collide)
	digest := sha256.Sum256(archive)
	descriptor := fmt.Sprintf(`{
  "schema_version": 1,
  "go_version": "go1.26.4",
  "archive": {"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"%x"},
  "supported_platforms": ["darwin/arm64"],
  "boundary_manifest_version": "go1.26.4-darwin-arm64-v1",
  "patch": "go1.26.4.patch",
  "adapters": [{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],
  "patch_allowlist": ["src/runtime/proc.go"],
  "overlay_allowlist": ["src/os/gomad.go"]
}
`, digest)
	if err := os.WriteFile(filepath.Join(root, "version.json"), []byte(descriptor), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func createArchive(t *testing.T, path string, collide bool) []byte {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zipper := gzip.NewWriter(file)
	archive := tar.NewWriter(zipper)
	paths := []string{"go/src/runtime/proc.go"}
	if collide {
		paths = append(paths, "go/src/os/gomad.go")
	}
	for _, name := range paths {
		contents := []byte("package fixture\n")
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write(contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}
