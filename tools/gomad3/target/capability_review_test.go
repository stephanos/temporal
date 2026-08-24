package target

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target/internal/livecap"
)

func TestProjectCapabilityReviewCollectsEveryDeniedFinding(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "target.go"), []byte("package main\n\n//go:linkname malformed\nfunc malformed()\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "escape.c"), []byte("int escape;\n"), 0o600))

	review, err := projectCapabilityReview([]listedPackage{
		{
			ImportPath: "example.com/dependency", Name: "dependency", DepOnly: true,
			Imports: []string{"syscall"}, Module: &listedModule{Path: "example.com/dependency", Version: "v1.0.0", Sum: "h1:dependency"},
		},
		{
			ImportPath: "example.com/target", Name: "main", Dir: directory, GoFiles: []string{"target.go"},
			Imports: []string{"example.com/dependency", "os/exec"}, CFiles: []string{"escape.c"}, Module: &listedModule{Path: "example.com/target", Main: true},
		},
	}, nil, []string{"gomad_fixture"})
	requireTestNoError(t, err)
	requireTestEqual(t, []CapabilityPackageReference{{ImportPath: "example.com/target", Name: "main"}}, review.Roots)
	requireTestEqual(t, []string{"gomad_fixture"}, review.BuildTags)
	if len(review.Findings) != 5 {
		t.Fatalf("findings = %#v", review.Findings)
	}
	requireTestEqual(t, []CapabilityFindingKind{
		FindingForbiddenImport,
		FindingNoReviewedGoSource,
		FindingForbiddenImport,
		FindingForeignSource,
		FindingMalformedLinkname,
	}, []CapabilityFindingKind{
		review.Findings[0].Kind,
		review.Findings[1].Kind,
		review.Findings[2].Kind,
		review.Findings[3].Kind,
		review.Findings[4].Kind,
	})
	requireTestEqual(t, "import:syscall", review.Findings[0].Capability)
	requireTestEqual(t, RemediationAddExactPack, review.Findings[0].Remediation)
	requireTestEqual(t, RemediationRemoveDependency, review.Findings[1].Remediation)
	requireTestEqual(t, "target.go", review.Findings[4].SourceName)
	requireTestEqual(t, []string{}, review.Findings[4].Directives)
	requireTestEqual(t, DispositionDenied, review.Findings[4].PolicyDisposition)

	err = validateCapabilityReview(review.Closure)
	var unsupported *UnsupportedCapabilityError
	if !errors.As(err, &unsupported) {
		t.Fatalf("validateCapabilityReview() error = %T %v", err, err)
	}
	requireTestEqual(t, review.Findings[0], unsupported.Finding)
}

func TestProjectCapabilityReviewHashesForeignSources(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "escape.c"), []byte("int escape;\n"), 0o600))

	review, err := projectCapabilityReview([]listedPackage{{
		ImportPath: "example.com/target", Name: "main", Dir: directory,
		GoFiles: []string{"main.go"}, CFiles: []string{"escape.c"}, Module: &listedModule{Path: "example.com/target", Main: true},
	}}, nil, nil)
	requireTestNoError(t, err)
	requireTestEqual(t, []CapabilityForeignSource{{
		Kind: "c", Name: "escape.c", SHA256: "sha256:38e20fd52e198548ec375726cb0095711eb8788e4425035fe3c1ea598ad312a2",
	}}, review.Closure.Packages[0].ForeignSources)
	requireTestEqual(t, "foreign:c:escape.c", review.Findings[0].Capability)
	requireTestEqual(t, "escape.c", review.Findings[0].SourceName)
	requireTestEqual(t, "sha256:38e20fd52e198548ec375726cb0095711eb8788e4425035fe3c1ea598ad312a2", review.Findings[0].SourceSHA256)
	policyPackage := capabilityCompatibilityPackage(review.Closure.Packages[0])
	requireTestEqual(t, []compatibility.ForeignSource{{
		Kind: "c", Name: "escape.c", SHA256: "sha256:38e20fd52e198548ec375726cb0095711eb8788e4425035fe3c1ea598ad312a2",
	}}, policyPackage.ForeignSources)
	if len(policyPackage.GoSources) != 1 || policyPackage.GoSources[0].Name != "main.go" {
		t.Fatalf("policy package = %#v", policyPackage)
	}
}

func TestProjectCapabilityReviewHashesHeaderWithoutExecutableFinding(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "api.h"), []byte("#define VALUE 1\n"), 0o600))

	review, err := projectCapabilityReview([]listedPackage{{
		ImportPath: "example.com/target", Name: "main", Dir: directory,
		GoFiles: []string{"main.go"}, HFiles: []string{"api.h"}, Module: &listedModule{Path: "example.com/target", Main: true},
	}}, nil, nil)
	requireTestNoError(t, err)
	requireTestEqual(t, []CapabilityForeignSource{{
		Kind: "header", Name: "api.h", SHA256: "sha256:ac32a92f4af359517c993d6e8583ea2d9b053c2ad171b4c602bc242894bd9696",
	}}, review.Closure.Packages[0].ForeignSources)
	requireTestEqual(t, []CapabilityFinding{}, review.Findings)
}

func TestProjectCapabilityReviewValidatesAdapterReplacementEvidence(t *testing.T) {
	directory := t.TempDir()
	replacement := filepath.Join(directory, "replacement")
	requireTestNoError(t, os.Mkdir(replacement, 0o700))
	requireTestNoError(t, os.WriteFile(filepath.Join(replacement, "main.go"), []byte("package adapter\n"), 0o600))
	moduleSum := "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	adapter := AdapterReplacement{
		Original:        ModuleIdentity{Path: "example.com/adapter", Version: "v1.2.3", Sum: moduleSum},
		ReplacementPath: replacement,
		PreparedPackage: "example.com/adapter",
		ProfileName:     "gomad3-deterministic/v1", ProfileImplementationSHA256: "sha256:" + strings.Repeat("1", 64),
		Adapter:                          ModuleIdentity{Path: "example.com/adapter", Version: "v1.2.3", Sum: moduleSum},
		OriginalSourceInventorySHA256:    "sha256:" + strings.Repeat("2", 64),
		ReplacementSourceInventorySHA256: "sha256:4ddea8f6f238465a2a12e9b32c32a17421d205dbf318d75f49e9fc3378c9b64b",
		PreparedSourceSetSHA256:          "sha256:bb21537d27aaf3797c51cc354bfca8306defdbd42e1757b04fc682000bbeb12f",
	}

	review, err := projectCapabilityReview([]listedPackage{{
		ImportPath: "example.com/adapter", Name: "adapter", Dir: replacement, GoFiles: []string{"main.go"},
		Module: &listedModule{Path: "example.com/adapter", Version: "v1.2.3", Replace: &listedModule{Dir: replacement}},
	}, {
		ImportPath: "example.com/target", Name: "main", Standard: true,
	}}, nil, nil, []AdapterReplacement{adapter})
	requireTestNoError(t, err)
	module := review.Closure.Packages[0].Module
	if module == nil || module.Sum != moduleSum || module.Adapter == nil || module.Adapter.ReplacementSourceInventorySHA256 != adapter.ReplacementSourceInventorySHA256 {
		t.Fatalf("module = %#v", module)
	}

	adapter.ReplacementPath = filepath.Join(directory, "other")
	if _, err := projectCapabilityReview([]listedPackage{{
		ImportPath: "example.com/adapter", Name: "adapter", Dir: replacement, GoFiles: []string{"main.go"},
		Module: &listedModule{Path: "example.com/adapter", Version: "v1.2.3", Sum: moduleSum, Replace: &listedModule{Dir: replacement}},
	}, {ImportPath: "example.com/target", Name: "main", Standard: true}}, nil, nil, []AdapterReplacement{adapter}); err == nil {
		t.Fatal("projectCapabilityReview() accepted the wrong adapter path")
	}
}

func TestProjectCapabilityReviewValidatesNestedAdapterPreparedPackage(t *testing.T) {
	directory := t.TempDir()
	replacement := filepath.Join(directory, "replacement")
	packageDirectory := filepath.Join(replacement, "internal")
	requireTestNoError(t, os.MkdirAll(packageDirectory, 0o700))
	contents := []byte("package internal\n")
	requireTestNoError(t, os.WriteFile(filepath.Join(packageDirectory, "internal.go"), contents, 0o600))
	moduleSum := "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	sourceSHA256 := fmt.Sprintf("sha256:%x", sha256.Sum256(contents))
	replacementInventory, err := DigestAdapterSourceInventory(replacement)
	requireTestNoError(t, err)
	adapter := AdapterReplacement{
		Original:        ModuleIdentity{Path: "example.com/adapter", Version: "v1.2.3", Sum: moduleSum},
		ReplacementPath: replacement,
		PreparedPackage: "example.com/adapter/internal",
		ProfileName:     "gomad3-deterministic/v1", ProfileImplementationSHA256: "sha256:" + strings.Repeat("1", 64),
		Adapter:                          ModuleIdentity{Path: "example.com/adapter", Version: "v1.2.3", Sum: moduleSum},
		OriginalSourceInventorySHA256:    "sha256:" + strings.Repeat("2", 64),
		ReplacementSourceInventorySHA256: replacementInventory,
		PreparedSourceSetSHA256: compatibility.DigestSources([]compatibility.Source{{
			Name: "internal.go", SHA256: sourceSHA256,
		}}),
	}

	packages := []listedPackage{{
		ImportPath: "example.com/adapter/internal", Name: "internal", Dir: packageDirectory, GoFiles: []string{"internal.go"},
		Module: &listedModule{Path: "example.com/adapter", Version: "v1.2.3", Replace: &listedModule{Dir: replacement}},
	}, {
		ImportPath: "example.com/target", Name: "main", Standard: true,
	}}
	if _, err := projectCapabilityReview(packages, nil, nil, []AdapterReplacement{adapter}); err != nil {
		t.Fatal(err)
	}
	adapter.PreparedSourceSetSHA256 = "sha256:" + strings.Repeat("0", 64)
	if _, err := projectCapabilityReview(packages, nil, nil, []AdapterReplacement{adapter}); err == nil {
		t.Fatal("projectCapabilityReview() accepted the wrong nested prepared source set")
	}

	adapter.PreparedSourceSetSHA256 = compatibility.DigestSources([]compatibility.Source{{
		Name: "internal.go", SHA256: sourceSHA256,
	}})
	packages[0].ImportPath = "example.com/adapter/other"
	if _, err := projectCapabilityReview(packages, nil, nil, []AdapterReplacement{adapter}); err == nil {
		t.Fatal("projectCapabilityReview() accepted adapter evidence without its prepared package")
	}
}

func TestDigestAdapterSourceInventoryReturnsTypedCapacityError(t *testing.T) {
	root := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(root, "one.go"), []byte("1"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(root, "two.go"), []byte("2"), 0o600))

	for _, test := range []struct {
		name         string
		maximumFiles int
		maximumBytes uint64
		resource     string
	}{
		{name: "files", maximumFiles: 1, maximumBytes: 2, resource: "files"},
		{name: "bytes", maximumFiles: 2, maximumBytes: 1, resource: "bytes"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := digestAdapterSourceInventory(root, test.maximumFiles, test.maximumBytes)
			var capacity *AdapterCapacityError
			if !errors.As(err, &capacity) || capacity.Resource != test.resource {
				t.Fatalf("digestAdapterSourceInventory() error = %#v", err)
			}
		})
	}
}

func TestProjectCapabilityReviewTreatsUnreadableSourceAsInfrastructureFailure(t *testing.T) {
	_, err := projectCapabilityReview([]listedPackage{{
		ImportPath: "example.com/target", Name: "main", Dir: t.TempDir(), GoFiles: []string{"missing.go"},
		Module: &listedModule{Path: "example.com/target", Main: true},
	}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unreadable source") {
		t.Fatalf("projectCapabilityReview() error = %v", err)
	}
	var unsupported *UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		t.Fatalf("projectCapabilityReview() error type = %T", err)
	}
}

func TestProjectCapabilityReviewRejectsUnsafeSourceFiles(t *testing.T) {
	for _, test := range []struct {
		name string
		make func(*testing.T, string)
	}{
		{name: "symbolic link", make: func(t *testing.T, path string) {
			t.Helper()
			target := filepath.Join(filepath.Dir(path), "target.go")
			requireTestNoError(t, os.WriteFile(target, []byte("package main\n"), 0o600))
			requireTestNoError(t, os.Symlink(target, path))
		}},
		{name: "oversized", make: func(t *testing.T, path string) {
			t.Helper()
			requireTestNoError(t, os.WriteFile(path, make([]byte, maximumCapabilitySourceBytes+1), 0o600))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			test.make(t, filepath.Join(directory, "input.go"))
			_, err := projectCapabilityReview([]listedPackage{{
				ImportPath: "example.com/target", Name: "main", Dir: directory, GoFiles: []string{"input.go"},
				Module: &listedModule{Path: "example.com/target", Main: true},
			}}, nil, nil)
			if err == nil || !strings.Contains(err.Error(), "unreadable source") {
				t.Fatalf("projectCapabilityReview() error = %v", err)
			}
		})
	}
}

func TestReviewGoCapabilityReviewRejectsBoundedOutputOverflow(t *testing.T) {
	directory := t.TempDir()
	command := filepath.Join(directory, "go")
	requireTestNoError(t, os.WriteFile(command, []byte(fmt.Sprintf("#!/bin/sh\nyes x | head -c %d\n", maximumCapabilityReviewOutputBytes+1)), 0o700))

	_, err := reviewGoCapabilityReview(context.Background(), command, Spec{Kind: KindGoRun}, nil, directory, ".")
	if err == nil || !strings.Contains(err.Error(), "output exceeds") {
		t.Fatalf("reviewGoCapabilityReview() error = %v", err)
	}
	if IsInvalidCapabilityReview(err) {
		t.Fatalf("output overflow was classified as invalid input: %v", err)
	}
}

func TestReviewGoCapabilityReviewClassifiesGoListRejectionAsInvalidInput(t *testing.T) {
	for _, diagnostic := range []string{
		"no required module provides package missing",
		"updates to go.mod needed, disabled by -mod=readonly",
		"missing go.sum entry for module providing package example.com/dependency",
		"cannot find module providing package example.com/dependency: import lookup disabled by -mod=readonly",
	} {
		t.Run(diagnostic, func(t *testing.T) {
			directory := t.TempDir()
			command := filepath.Join(directory, "go")
			requireTestNoError(t, os.WriteFile(command, []byte("#!/bin/sh\necho '"+diagnostic+"' >&2\nexit 1\n"), 0o700))

			_, err := reviewGoCapabilityReview(context.Background(), command, Spec{Kind: KindGoRun}, nil, directory, "missing")
			if err == nil || !IsInvalidCapabilityReview(err) {
				t.Fatalf("reviewGoCapabilityReview() error = %T %v", err, err)
			}
		})
	}
}

func TestReviewGoCapabilityReviewClassifiesUnknownGoListFailureAsInfrastructure(t *testing.T) {
	directory := t.TempDir()
	command := filepath.Join(directory, "go")
	requireTestNoError(t, os.WriteFile(command, []byte("#!/bin/sh\necho internal-toolchain-failure >&2\nexit 1\n"), 0o700))

	_, err := reviewGoCapabilityReview(context.Background(), command, Spec{Kind: KindGoRun}, nil, directory, ".")
	if err == nil || IsInvalidCapabilityReview(err) {
		t.Fatalf("reviewGoCapabilityReview() error = %T %v", err, err)
	}
}

func TestReviewGoCapabilityReviewClassifiesTimeoutAsInfrastructureFailure(t *testing.T) {
	directory := t.TempDir()
	command := filepath.Join(directory, "go")
	requireTestNoError(t, os.WriteFile(command, []byte("#!/bin/sh\nwhile :; do :; done\n"), 0o700))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := reviewGoCapabilityReview(ctx, command, Spec{Kind: KindGoRun}, nil, directory, ".")
	if err == nil || !errors.Is(err, context.DeadlineExceeded) || IsInvalidCapabilityReview(err) {
		t.Fatalf("reviewGoCapabilityReview() error = %T %v", err, err)
	}
}

func TestReviewGoCapabilityReviewUsesReadonlyModuleResolution(t *testing.T) {
	directory := t.TempDir()
	command := filepath.Join(directory, "go")
	requireTestNoError(t, os.WriteFile(command, []byte(`#!/bin/sh
case " $* " in
  *" -mod=readonly "*) ;;
  *) echo missing-readonly >&2; exit 1 ;;
esac
printf '%s' '{"ImportPath":"example.com/target","Name":"main","Standard":true}'
`), 0o700))

	_, err := reviewGoCapabilityReview(context.Background(), command, Spec{Kind: KindGoRun}, nil, directory, ".")
	requireTestNoError(t, err)
}

func TestReviewCapabilitiesClassifiesRealReadonlyModuleFailureAsInvalidInput(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/target\n\ngo 1.26.4\n\nrequire github.com/stretchr/testify v1.11.1\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\n\nimport _ \"github.com/stretchr/testify/require\"\n\nfunc main() {}\n"), 0o600))

	_, err := ReviewCapabilities(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: directory, ToolchainRoot: toolchainRoot(t),
	})
	if err == nil || !IsInvalidCapabilityReview(err) {
		t.Fatalf("ReviewCapabilities() error = %T %v", err, err)
	}
	if _, statErr := os.Stat(filepath.Join(directory, "go.sum")); !os.IsNotExist(statErr) {
		t.Fatalf("read-only capability review wrote go.sum: %v", statErr)
	}
}

func TestReviewCapabilitiesClassifiesInvalidTargetInputs(t *testing.T) {
	tests := map[string]Spec{
		"kind":      {Kind: KindExec},
		"build tag": {Kind: KindGoRun, BuildTags: []string{"invalid,tag"}},
		"source":    {Kind: KindGoRun, Source: "-option", WorkingDir: t.TempDir(), ToolchainRoot: t.TempDir()},
	}
	for name, spec := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ReviewCapabilities(context.Background(), spec)
			if err == nil || !IsInvalidCapabilityReview(err) {
				t.Fatalf("ReviewCapabilities() error = %T %v", err, err)
			}
		})
	}
}

func TestReviewCapabilitiesLinkedModeBuildsAndProjectsManifest(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/linkedreview\n\ngo 1.26.4\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte(`package main

import "os/exec"

func dead() { _ = exec.Command("true") }
func main() {}
`), 0o600))
	review, err := ReviewCapabilities(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: directory, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
		CapabilityMode: CapabilityModeLinked,
	})
	requireTestNoError(t, err)
	if review.Schema != CapabilityReviewSchema || review.CapabilityMode != CapabilityModeLinked || review.CapabilityManifest == nil {
		t.Fatalf("linked review identity = %#v", review)
	}
	if len(review.Findings) != 0 || len(review.EliminatedFindings) != 1 || review.EliminatedFindings[0].Capability != "import:os/exec" {
		t.Fatalf("linked review findings = active %#v eliminated %#v", review.Findings, review.EliminatedFindings)
	}
}

func TestReviewCapabilitiesLinkedModeReturnsActiveFindings(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/linkedunsupported\n\ngo 1.26.4\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\nimport \"os/exec\"\nfunc main() { _ = exec.Command(\"true\") }\n"), 0o600))
	review, err := ReviewCapabilities(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: directory, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
		CapabilityMode: CapabilityModeLinked,
	})
	requireTestNoError(t, err)
	if len(review.Findings) != 1 || review.Findings[0].Capability != "import:os/exec" || len(review.EliminatedFindings) != 0 {
		t.Fatalf("linked review findings = active %#v eliminated %#v", review.Findings, review.EliminatedFindings)
	}
}

func TestReviewCapabilitiesGuardedModeReturnsGuardedFindings(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/guardedreview\n\ngo 1.26.4\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\nimport \"os/exec\"\nfunc main() { _ = exec.Command(\"true\") }\n"), 0o600))
	review, err := ReviewCapabilities(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: directory, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
		CapabilityMode: CapabilityModeGuarded,
	})
	requireTestNoError(t, err)
	if review.CapabilityMode != CapabilityModeGuarded || review.CapabilityManifest == nil || review.CapabilityManifest.Schema != "gomad3.live-capability-manifest/v2" || review.CapabilityManifest.GuardImplementationSHA256 == "" {
		t.Fatalf("guarded review identity = %#v", review)
	}
	if len(review.Findings) != 0 || len(review.EliminatedFindings) != 0 || len(review.GuardedFindings) != 1 || review.GuardedFindings[0].Capability != "import:os/exec" {
		t.Fatalf("guarded review findings = active %#v guarded %#v eliminated %#v", review.Findings, review.GuardedFindings, review.EliminatedFindings)
	}
}

func TestReviewCapabilitiesLinkedModeRejectsLiveDeniedBoundary(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/linkedboundary\n\ngo 1.26.4\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\nimport \"os\"\nfunc main() { _, _ = os.Readlink(\"target\") }\n"), 0o600))
	review, err := ReviewCapabilities(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: directory, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
		CapabilityMode: CapabilityModeLinked,
	})
	requireTestNoError(t, err)
	if len(review.Findings) != 1 || review.Findings[0].Kind != FindingDeniedBoundary || review.Findings[0].Capability != "filesystem.readlink" {
		t.Fatalf("linked denied-boundary findings = %#v", review.Findings)
	}
}

func TestReviewCapabilitiesGuardedModeGuardsLiveDeniedBoundary(t *testing.T) {
	directory := t.TempDir()
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.com/guardedboundary\n\ngo 1.26.4\n"), 0o600))
	requireTestNoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main\nimport \"os\"\nfunc main() { _, _ = os.Readlink(\"target\") }\n"), 0o600))
	review, err := ReviewCapabilities(context.Background(), Spec{
		Kind: KindGoRun, Source: ".", WorkingDir: directory, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t),
		CapabilityMode: CapabilityModeGuarded,
	})
	requireTestNoError(t, err)
	if len(review.Findings) != 0 || len(review.GuardedFindings) != 1 || review.GuardedFindings[0].Kind != FindingDeniedBoundary || review.GuardedFindings[0].Capability != "filesystem.readlink" {
		t.Fatalf("guarded denied-boundary findings = active %#v guarded %#v", review.Findings, review.GuardedFindings)
	}
}

func TestLinkedCapabilityCapacityErrorIsUnsupported(t *testing.T) {
	err := linkedCapabilityError(&livecap.CapacityError{Resource: "fact count", Required: 100001, Maximum: 100000})
	var capacity *UnsupportedCapabilityCapacityError
	if !errors.As(err, &capacity) {
		t.Fatalf("linkedCapabilityError() = %T %v", err, err)
	}
	if capacity.Resource != "fact count" || capacity.Required != 100001 || capacity.Maximum != 100000 || !IsUnsupportedCapability(err) {
		t.Fatalf("capacity error = %#v", capacity)
	}
}

func TestLinkedCapabilityBuildCapacityDiagnosticIsUnsupported(t *testing.T) {
	err := linkedCapabilityBuildError(errors.New("exit status 1"), []byte("link: -gomadcap: live capability facts requires 100001, maximum is 100000\n"))
	var capacity *UnsupportedCapabilityCapacityError
	if !errors.As(err, &capacity) || capacity.Resource != "facts" || capacity.Required != 100001 || capacity.Maximum != 100000 {
		t.Fatalf("linkedCapabilityBuildError() = %T %v", err, err)
	}
}

func requireTestNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireTestEqual(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
