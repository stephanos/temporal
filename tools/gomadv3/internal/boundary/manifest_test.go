package boundary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWritesAndChecksAllArtifacts(t *testing.T) {
	root := t.TempDir()
	writeTestManifest(t, root, validManifest)

	if err := Generate(root, false); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, true); err != nil {
		t.Fatal(err)
	}

	for _, relative := range []string{
		"overlay/src/cmd/compile/internal/gomadintercept/spec_go126.go",
		"expected-intercepts-go1.26.4.txt",
		"boundary/go1.26.4-darwin-arm64.md",
		"internal/ioprofile/boundary_generated.go",
	} {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if len(contents) == 0 {
			t.Fatalf("generated artifact %s is empty", relative)
		}
		if relative == "overlay/src/cmd/compile/internal/gomadintercept/spec_go126.go" {
			if !strings.Contains(string(contents), "ProbeID:") {
				t.Fatalf("generated compiler spec has no semantic probe")
			}
			if !strings.Contains(string(contents), "func qualifiedPlatform") || !strings.Contains(string(contents), `goos == "darwin" && goarch == "arm64"`) {
				t.Fatalf("generated compiler spec has no qualified platform guard")
			}
		}
		if relative == "internal/ioprofile/boundary_generated.go" && !strings.Contains(string(contents), "generatedBoundaryProbes") {
			t.Fatalf("generated host identity has no semantic probe inventory")
		}
	}

	stale := filepath.Join(root, "expected-intercepts-go1.26.4.txt")
	if err := os.WriteFile(stale, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Generate(root, true); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("Generate(check) error = %v", err)
	}
}

func TestLoadRejectsInvalidManifest(t *testing.T) {
	tests := map[string]string{
		"unknown field":    strings.Replace(validManifest, `"schema_version": 1`, `"schema_version": 1, "extra": true`, 1),
		"trailing data":    validManifest + "{}\n",
		"duplicate target": strings.Replace(validManifest, `"compiler_tests": []`, `"intercepts": [`+validEntry+`,`+validEntry+`], "compiler_tests": []`, 1),
		"duplicate probe":  strings.Replace(validManifest, `"compiler_tests": []`, `"intercepts": [`+validEntry+`,`+strings.Replace(validEntry, `"symbol": "OpenFile"`, `"symbol": "Create"`, 1)+`], "compiler_tests": []`, 1),
		"bad disposition":  strings.Replace(validManifest, `"disposition": "model"`, `"disposition": "allow"`, 1),
		"missing delegate": strings.Replace(validManifest, `"disposition": "model"`, `"disposition": "delegate"`, 1),
		"missing signature": strings.Replace(validManifest,
			`"signature": "func(name string, flag int, perm FileMode) (*File, error)"`,
			`"signature": ""`, 1),
		"missing fingerprint": strings.Replace(validManifest,
			`"declaration_sha256": "sha256:d08e5b732697b374f939fb09958c41140fbe086567f00170cd938f53a2758522"`,
			`"declaration_sha256": ""`, 1),
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTestManifest(t, root, contents)
			if _, err := load(filepath.Join(root, "boundary", "manifest.json")); err == nil {
				t.Fatal("load() succeeded")
			}
		})
	}
}

func TestLoadRejectsUnresolvedAndCyclicDelegates(t *testing.T) {
	delegate := strings.Replace(validEntry, `"symbol": "OpenFile"`, `"symbol": "Create"`, 1)
	delegate = strings.Replace(delegate, `"probe": "stdlib.os.open-file"`, `"probe": "stdlib.os.create"`, 1)
	delegate = strings.Replace(delegate, `"disposition": "model"`, `"disposition": "delegate"`, 1)
	for name, boundary := range map[string]string{
		"unresolved": "stdlib.os.missing",
		"cycle":      "stdlib.os.create",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			entry := strings.Replace(delegate, `"delegated_boundary": ""`, `"delegated_boundary": "`+boundary+`"`, 1)
			contents := strings.Replace(validManifest, `"compiler_tests": []`, `"intercepts": [`+validEntry+`,`+entry+`], "compiler_tests": []`, 1)
			writeTestManifest(t, root, contents)
			if _, err := load(filepath.Join(root, "boundary", "manifest.json")); err == nil {
				t.Fatal("load() succeeded")
			}
		})
	}
}

func TestValidateCandidateCoverageRequiresExplicitClassification(t *testing.T) {
	root := t.TempDir()
	writeTestManifest(t, root, validManifest)
	definition, err := load(filepath.Join(root, "boundary", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCandidateCoverage(definition, []string{"os.OpenFile"}); err != nil {
		t.Fatal(err)
	}
	if err := validateCandidateCoverage(definition, []string{"os.Open", "os.OpenFile"}); err == nil || !strings.Contains(err.Error(), "os.Open") {
		t.Fatalf("validateCandidateCoverage() error = %v", err)
	}

	definition.ReviewedCandidates = []reviewedCandidate{{
		Target: "os.Open", Disposition: "delegate", Boundaries: []string{"stdlib.os.open-file"},
	}}
	if err := validateCandidateCoverage(definition, []string{"os.Open", "os.OpenFile"}); err != nil {
		t.Fatal(err)
	}
	if err := validateCandidateCoverage(definition, []string{"os.OpenFile"}); err == nil || !strings.Contains(err.Error(), "no longer discovered") {
		t.Fatalf("validateCandidateCoverage(stale) error = %v", err)
	}
}

func TestValidateDelegateReachabilityRejectsBypassedBoundary(t *testing.T) {
	root := t.TempDir()
	writeTestManifest(t, root, validManifest)
	definition, err := load(filepath.Join(root, "boundary", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	definition.ReviewedCandidates = []reviewedCandidate{{
		Target: "os.Open", Disposition: "delegate", Boundaries: []string{"stdlib.os.open-file"},
	}}
	functions := map[string]*discoveredFunction{
		"os.Open":     {callKeys: []string{"os.OpenFile"}},
		"os.OpenFile": {},
	}
	if err := validateDelegateReachability(definition, functions); err != nil {
		t.Fatal(err)
	}
	functions["os.Open"].callKeys = nil
	if err := validateDelegateReachability(definition, functions); err == nil || !strings.Contains(err.Error(), "does not reach") {
		t.Fatalf("validateDelegateReachability() error = %v", err)
	}
}

func TestQualifyChecksPinnedStandardLibrarySignatures(t *testing.T) {
	root := t.TempDir()
	writeTestManifest(t, root, validManifest)
	if err := Qualify(root); err != nil {
		t.Fatal(err)
	}

	invalid := strings.Replace(validManifest,
		`func(name string, flag int, perm FileMode) (*File, error)`,
		`func(name string) (*File, error)`, 1)
	writeTestManifest(t, root, invalid)
	if err := Qualify(root); err == nil || !strings.Contains(err.Error(), "signature mismatch") {
		t.Fatalf("Qualify() error = %v", err)
	}

	stale := strings.Replace(validManifest,
		`sha256:d08e5b732697b374f939fb09958c41140fbe086567f00170cd938f53a2758522`,
		`sha256:0000000000000000000000000000000000000000000000000000000000000000`, 1)
	writeTestManifest(t, root, stale)
	if err := Qualify(root); err == nil || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("Qualify(stale source) error = %v", err)
	}
}

func TestRefreshFingerprintsRepairsMissingSourceIdentity(t *testing.T) {
	root := t.TempDir()
	incomplete := strings.Replace(validManifest,
		`"source": "os/file.go",`,
		`"source": "",`, 1)
	writeTestManifest(t, root, incomplete)
	if err := RefreshFingerprints(root); err != nil {
		t.Fatal(err)
	}
	if err := Qualify(root); err != nil {
		t.Fatal(err)
	}
}

func TestManifestIdentityIgnoresCompilerFixtures(t *testing.T) {
	definition := manifest{
		SchemaVersion: 1, ManifestVersion: "go1.26.4-darwin-arm64-v1", GoVersion: "go1.26.4",
		Platforms: []string{"darwin/arm64"}, Intercepts: []intercept{{Package: "os", Symbol: "Open", Hook: "gomadInterceptOpen"}},
		CompilerTests: []compilerTest{{Case: "first", Package: "example.test/first", Symbol: "Target", Hook: "Hook"}},
	}
	first, err := manifestIdentity(definition)
	if err != nil {
		t.Fatal(err)
	}
	definition.CompilerTests = []compilerTest{{Case: "second", Package: "example.test/second", Symbol: "Other", Hook: "OtherHook"}}
	second, err := manifestIdentity(definition)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("compiler fixture changed manifest identity: %s != %s", first, second)
	}
}

func writeTestManifest(t *testing.T, root, contents string) {
	t.Helper()
	path := filepath.Join(root, "boundary", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

const validEntry = `{
  "package": "os",
  "symbol": "OpenFile",
  "signature": "func(name string, flag int, perm FileMode) (*File, error)",
  "source": "os/file.go",
  "declaration_sha256": "sha256:d08e5b732697b374f939fb09958c41140fbe086567f00170cd938f53a2758522",
  "package_sha256": "sha256:f3919bd2e90d342f0528ec9032dacf27cd1d6cba49b25c6bf7bf9b6dff7c7a5d",
  "operation": "filesystem.open",
  "probe": "stdlib.os.open-file",
  "disposition": "model",
  "hook": "gomadInterceptOpenFile",
  "delegated_boundary": "",
  "adapters": ["internal/gomadfs"],
  "conformance_fixtures": ["internal/ioprofile.TestProfileFilesystemStaysInMemory"],
  "negative_fixtures": [],
  "escape_fixtures": ["testdata/io_filesystem.host-escape"]
}`

const validManifest = `{
  "schema_version": 1,
  "manifest_version": "go1.26.4-darwin-arm64-v1",
  "go_version": "go1.26.4",
  "platforms": ["darwin/arm64"],
  "intercepts": [` + validEntry + `],
  "compiler_tests": []
}
`
