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

	shell := exec.Command("bash", "-c", `source "$1"; printf '%s|%s|%s|%s|%s\n' "$go_version" "$patch_name" "$adapter_modernc_org_libc_version" "${patch_allowed_paths[0]}" "${qualified_platforms[0]}"`, "bash", filepath.Join(root, "toolchain-version.sh"))
	if output, err := shell.CombinedOutput(); err != nil || string(output) != "go1.26.4|go1.26.4.patch|v1.72.3|src/runtime/proc.go|darwin/arm64\n" {
		t.Fatalf("generated shell output = %q, error = %v", output, err)
	}
	makeConsumer := "include version_generated.mk\n\nprint:\n\t@printf '%s|%s|%s|%s\\n' '$(GOMADV3_GO_VERSION)' '$(GOMADV3_PATCH_FILE)' '$(GOMADV3_EXPECTED_INTERCEPTS)' '$(GOMADV3_BOUNDARY_REPORT)'\n"
	if err := os.WriteFile(filepath.Join(root, "consumer.mk"), []byte(makeConsumer), 0o600); err != nil {
		t.Fatal(err)
	}
	makeCommand := exec.Command("make", "-f", "consumer.mk", "print")
	makeCommand.Dir = root
	if output, err := makeCommand.CombinedOutput(); err != nil || string(output) != "go1.26.4|go1.26.4.patch|expected-intercepts-go1.26.4.txt|boundary/go1.26.4-darwin-arm64.md\n" {
		t.Fatalf("generated Make output = %q, error = %v", output, err)
	}
	goTest := `package version

import "testing"

func TestGeneratedValues(t *testing.T) {
	if GoVersion != "go1.26.4" || ModerncLibcVersion != "v1.72.3" || BoundaryManifestVersion != "go1.26.4-darwin-arm64-v1" {
		t.Fatalf("generated values = %q, %q, %q", GoVersion, ModerncLibcVersion, BoundaryManifestVersion)
	}
}
`
	goDirectory := filepath.Join(root, "internal", "version")
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
	if err := Generate(root, true); err == nil || !strings.Contains(err.Error(), "generated version artifact is stale: toolchain-version.sh") {
		t.Fatalf("Generate(check) error = %v", err)
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
	path := filepath.Join(root, "overlay", "src", "net", "gomad.go")
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
	if err := os.MkdirAll(filepath.Join(root, "boundary"), 0o700); err != nil {
		t.Fatal(err)
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
  "patch": "go1.26.4.patch",
  "adapters": [{
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
	if err := os.WriteFile(filepath.Join(root, "go1.26.4.patch"), []byte(patch), 0o600); err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(root, "overlay", "src", "os", "gomad.go")
	if err := os.MkdirAll(filepath.Dir(overlayPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlayPath, []byte("package os\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema_version":1,"manifest_version":"` + manifestVersion + `","go_version":"go1.26.4","platforms":["darwin/arm64"]}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "boundary", "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
