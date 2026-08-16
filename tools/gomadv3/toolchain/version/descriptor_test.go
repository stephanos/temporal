package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateRendersDescriptorConsumers(t *testing.T) {
	root := writeDescriptorFixture(t, "go1.26.4-darwin-arm64-v1")
	if err := Generate(root, false); err != nil {
		t.Fatal(err)
	}
	upgradeGuide, err := os.ReadFile(filepath.Join(root, upgradeGuideName(Descriptor{BoundaryManifestVersion: "go1.26.4-darwin-arm64-v1"})))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(upgradeGuide), "GOMADV3_APPROVED_BOUNDARY_DIFF_SHA256=<boundary_manifest_diff.sha256>") || !strings.Contains(string(upgradeGuide), "only after reviewing") {
		t.Fatalf("generated upgrade guide omits explicit boundary approval: %s", upgradeGuide)
	}

	if _, err := os.Stat(filepath.Join(root, "toolchain-version.sh")); !os.IsNotExist(err) {
		t.Fatalf("generated shell descriptor exists: %v", err)
	}
	makeConsumer := "include version_generated.mk\n\nprint:\n\t@printf '%s|%s|%s|%s\\n' '$(GOMADV3_GO_VERSION)' '$(GOMADV3_PATCH_FILE)' '$(GOMADV3_EXPECTED_INTERCEPTS)' '$(GOMADV3_BOUNDARY_REPORT)'\n"
	if err := os.WriteFile(filepath.Join(root, "consumer.mk"), []byte(makeConsumer), 0o600); err != nil {
		t.Fatal(err)
	}
	makeCommand := exec.Command("make", "-f", "consumer.mk", "print")
	makeCommand.Dir = root
	if output, err := makeCommand.CombinedOutput(); err != nil || string(output) != "go1.26.4|toolchain/runtime/go1.26.4.patch|expected-intercepts-go1.26.4.txt|deterministicio/boundary/go1.26.4-darwin-arm64.md\n" {
		t.Fatalf("generated Make output = %q, error = %v", output, err)
	}
	goTest := `package version

import "testing"

func TestGeneratedValues(t *testing.T) {
	grpc, grpcFound := AdapterByModule("google.golang.org/grpc")
	libc, libcFound := AdapterByModule("modernc.org/libc")
	if GoVersion != "go1.26.4" || !grpcFound || grpc.Version != "v1.80.0" || !libcFound || libc.Version != "v1.72.3" || BoundaryManifestVersion != "go1.26.4-darwin-arm64-v1" {
		t.Fatalf("generated values = %q, %#v, %#v, %q", GoVersion, grpc, libc, BoundaryManifestVersion)
	}
}
`
	goDirectory := filepath.Join(root, "toolchain", "version")
	if err := os.WriteFile(filepath.Join(goDirectory, "generated_test.go"), []byte(goTest), 0o600); err != nil {
		t.Fatal(err)
	}
	goCommand := exec.Command("go", "test", ".")
	goCommand.Dir = goDirectory
	goCommand.Env = append(os.Environ(), "GO111MODULE=off", "GOWORK=off")
	if output, err := goCommand.CombinedOutput(); err != nil {
		t.Fatalf("generated Go package failed: %v\n%s", err, output)
	}
	if err := Generate(root, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "toolchain-version.sh"), []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, true); err != nil {
		t.Fatalf("Generate(check) considers the obsolete shell descriptor: %v", err)
	}
}

func TestGenerateAllowsNoBuiltInAdapters(t *testing.T) {
	root := writeDescriptorFixture(t, "go1.26.4-darwin-arm64-v1")
	path := filepath.Join(root, descriptorPath)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), `"adapters": [{
    "module": "google.golang.org/grpc",
    "version": "v1.80.0",
    "sum": "h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM="
  }, {
    "module": "modernc.org/libc",
    "version": "v1.72.3",
    "sum": "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="
  }]`, `"adapters": []`, 1))
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, false); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(root, "toolchain", "version", "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "var Adapters = [...]AdapterIdentity{}") {
		t.Fatalf("generated adapters = %s", generated)
	}
}

func TestGenerateRejectsBoundaryVersionMismatch(t *testing.T) {
	root := writeDescriptorFixture(t, "go1.26.4-darwin-arm64-v2")
	if err := Generate(root, false); err == nil || !strings.Contains(err.Error(), "boundary manifest version") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateRejectsPatchAllowlistDrift(t *testing.T) {
	root := writeDescriptorFixture(t, "go1.26.4-darwin-arm64-v1")
	path := filepath.Join(root, descriptorPath)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), `"src/runtime/proc.go"`, `"src/runtime/proc.go", "src/runtime/rand.go"`, 1))
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, false); err == nil || !strings.Contains(err.Error(), "patch allowlist does not match") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateRejectsOverlayAllowlistDrift(t *testing.T) {
	root := writeDescriptorFixture(t, "go1.26.4-darwin-arm64-v1")
	path := filepath.Join(root, "toolchain", "runtime", "overlay", "src", "net", "gomad.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package net\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, false); err == nil || !strings.Contains(err.Error(), "overlay allowlist does not match") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestLoadRejectsDuplicateAllowedPath(t *testing.T) {
	root := writeDescriptorFixture(t, "go1.26.4-darwin-arm64-v1")
	path := filepath.Join(root, descriptorPath)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contents = []byte(strings.Replace(string(contents), `"src/runtime/proc.go"`, `"src/runtime/proc.go", "src/runtime/proc.go"`, 1))
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "patch allowlist path is duplicated") {
		t.Fatalf("Load() error = %v", err)
	}
}

func writeDescriptorFixture(t *testing.T, manifestVersion string) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"deterministicio/boundary", "toolchain/version"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	descriptor := `{
  "schema_version": 1,
  "go_version": "go1.26.4",
  "archive": {
    "name": "go1.26.4.src.tar.gz",
    "url": "https://go.dev/dl/go1.26.4.src.tar.gz",
    "sha256": "4f668a32fbfc1132e6a881fb968c2f1dada631492a339211735fbb255a42602d"
  },
  "supported_platforms": ["darwin/arm64"],
  "boundary_manifest_version": "go1.26.4-darwin-arm64-v1",
  "patch": "toolchain/runtime/go1.26.4.patch",
  "adapters": [{
    "module": "google.golang.org/grpc",
    "version": "v1.80.0",
    "sum": "h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM="
  }, {
    "module": "modernc.org/libc",
    "version": "v1.72.3",
    "sum": "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="
  }],
  "patch_allowlist": ["src/runtime/proc.go"],
  "overlay_allowlist": ["src/os/gomad.go"]
}
`
	if err := os.WriteFile(filepath.Join(root, descriptorPath), []byte(descriptor), 0o600); err != nil {
		t.Fatal(err)
	}
	patch := "diff --git a/src/runtime/proc.go b/src/runtime/proc.go\n"
	if err := os.MkdirAll(filepath.Join(root, "toolchain", "runtime"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "toolchain", "runtime", "go1.26.4.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(root, "toolchain", "runtime", "overlay", "src", "os", "gomad.go")
	if err := os.MkdirAll(filepath.Dir(overlayPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlayPath, []byte("package os\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema_version":1,"manifest_version":"` + manifestVersion + `","go_version":"go1.26.4","platforms":["darwin/arm64"]}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "deterministicio", "boundary", "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
