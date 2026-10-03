package deterministicio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestAdapterReplacementUsesExplicitModuleRoot(t *testing.T) {
	root := t.TempDir()
	adapter := BuildAdapter{
		Module: "example.com/adapter", Version: "v1.2.3", Sum: "h1:adapter",
		ReplacementRoot: root, Replacement: filepath.Join(root, "internal", "adapter.go"), PreparedPackage: "example.com/adapter/internal",
	}
	projected := projectAdapterReplacement(Contract{Name: "profile", ImplementationSHA256: "sha256:implementation"}, adapter)
	if projected.ReplacementPath != root || projected.PreparedPackage != adapter.PreparedPackage {
		t.Fatalf("projected replacement = %#v", projected)
	}
}

func TestNewAdapterRegistryAllowsNoAdapters(t *testing.T) {
	registry, err := newAdapterRegistry(nil, []adapterImplementation{{module: "modernc.org/libc"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.definitions) != 0 || len(registry.inventory()) != 0 {
		t.Fatalf("registry = %#v", registry)
	}
}

func TestPrepareBuildAdaptersClassifiesInvalidTargetConfiguration(t *testing.T) {
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.com/target\n\nrequire modernc.org/libc v1.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := Default().PrepareBuildAdapters(target.Spec{WorkingDir: workingDirectory}, t.TempDir())
	if !IsInvalidBuildAdapterConfiguration(err) {
		t.Fatalf("PrepareBuildAdapters() error = %v", err)
	}
	var invalid *InvalidBuildAdapterConfigurationError
	if !errors.As(err, &invalid) || invalid.Err == nil {
		t.Fatalf("PrepareBuildAdapters() error = %#v", err)
	}
}

func TestAdapterRegistryClassifiesMissingTargetModuleSumsAsInvalidConfiguration(t *testing.T) {
	workingDirectory := t.TempDir()
	identity := gomadversion.AdapterIdentity{Module: "example.com/adapter", Version: "v1.2.3", Sum: "h1:adapter"}
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.com/target\n\nrequire example.com/adapter v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	registry := adapterRegistry{definitions: []adapterDefinition{{
		identity: identity,
		implementation: adapterImplementation{module: identity.Module, prepare: func(_, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
			return adapterPreparation{replacement: root, evidence: BuildAdapter{Module: identity.Module, Version: identity.Version, Sum: identity.Sum, ReplacementRoot: root}}, nil
		}},
	}}}
	_, _, err := registry.prepare(target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir()}, t.TempDir())
	if !IsInvalidBuildAdapterConfiguration(err) {
		t.Fatalf("adapterRegistry.prepare() error = %v", err)
	}
}

func TestAdapterRegistryUsesExactVersionReplacement(t *testing.T) {
	workingDirectory := t.TempDir()
	identity := gomadversion.AdapterIdentity{Module: "example.com/adapter", Version: "v1.2.3", Sum: "h1:adapter"}
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.com/target\n\nrequire example.com/adapter v1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), []byte("example.com/adapter v1.2.3 h1:adapter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	registry := adapterRegistry{definitions: []adapterDefinition{{
		identity: identity,
		implementation: adapterImplementation{module: identity.Module, prepare: func(_, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
			replacement := filepath.Join(root, "module")
			if err := os.Mkdir(replacement, 0o700); err != nil {
				return adapterPreparation{}, err
			}
			return adapterPreparation{replacement: replacement, evidence: BuildAdapter{Module: identity.Module, Version: identity.Version, Sum: identity.Sum, ReplacementRoot: replacement}}, nil
		}},
	}}}
	_, adapters, err := registry.prepare(target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir()}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(adapters[0].BuildModFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "replace example.com/adapter v1.2.3 => ") {
		t.Fatalf("build modfile = %s", contents)
	}
}

func TestDetectModuleVersionRejectsDuplicateRequirements(t *testing.T) {
	for _, contents := range []string{
		"require example.com/adapter v1.2.3\nrequire example.com/adapter v1.2.3\n",
		"require (\nexample.com/adapter v1.2.3\n)\nrequire example.com/adapter v1.2.4\n",
	} {
		if _, err := detectModuleVersion([]byte(contents), "example.com/adapter"); err == nil {
			t.Fatalf("detectModuleVersion() accepted duplicate requirements in %q", contents)
		}
	}
}

func TestProfileVerifiesSelectedAdapterIdentities(t *testing.T) {
	selected := []Adapter{
		{Module: "google.golang.org/grpc", Version: "v1.83.2", Sum: "h1:EManeRomTObA0BU7I8vXgg/78uE5MJ9M8B39EX2WscU="},
		{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="},
	}
	if err := Default().VerifyAdapters(selected); err != nil {
		t.Fatal(err)
	}
	selected[0].Version = "v1.80.1"
	if err := Default().VerifyAdapters(selected); err == nil {
		t.Fatal("VerifyAdapters() accepted a modified identity")
	}
}

func TestNewAdapterRegistryRequiresAnImplementationForEveryIdentity(t *testing.T) {
	_, err := newAdapterRegistry([]gomadversion.AdapterIdentity{{
		Module: "example.com/runtime", Version: "v1.2.3", Sum: "h1:identity",
	}}, nil)
	if err == nil {
		t.Fatal("newAdapterRegistry() succeeded")
	}
}

func TestNewAdapterRegistryRejectsDuplicateImplementations(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: "example.com/runtime", Version: "v1.2.3", Sum: "h1:identity"}
	implementation := adapterImplementation{module: identity.Module}
	_, err := newAdapterRegistry([]gomadversion.AdapterIdentity{identity}, []adapterImplementation{implementation, implementation})
	if err == nil {
		t.Fatal("newAdapterRegistry() succeeded")
	}
}

func TestPrepareTargetBuildAdaptersRejectsMissingSumBeforeDownloading(t *testing.T) {
	workingDirectory := t.TempDir()
	moduleFile := []byte("module example.test\n\ngo 1.26.4\n\nrequire google.golang.org/grpc v1.83.2\n")
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), moduleFile, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// A toolchain root without a Go command fails any download attempt with a
	// different error, so the rejection must come before one.
	_, _, err := Default().PrepareTargetBuildAdapters(context.Background(), target.Spec{
		PreparationRoot: t.TempDir(), WorkingDir: workingDirectory, ToolchainRoot: t.TempDir(),
	})
	if !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), "module sum") {
		t.Fatalf("PrepareTargetBuildAdapters() error = %v", err)
	}
	for name, want := range map[string][]byte{"go.mod": moduleFile, "go.sum": {}} {
		got, readErr := os.ReadFile(filepath.Join(workingDirectory, name))
		if readErr != nil || string(got) != string(want) {
			t.Fatalf("%s = %q, %v; want unchanged %q", name, got, readErr, want)
		}
	}
}

// The go command records a directory replacement's path in the target's
// module information, so the published location is part of target identity.
func TestAdapterRegistryPublishesReplacementAtStableToolchainLocation(t *testing.T) {
	identity := gomadversion.AdapterIdentity{Module: "example.com/adapter", Version: "v1.2.3", Sum: "h1:adapter"}
	inventory := "sha256:0123456789abcdef" + strings.Repeat("0", 48)
	registry := adapterRegistry{definitions: []adapterDefinition{{
		identity: identity,
		implementation: adapterImplementation{module: identity.Module, prepare: func(_, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
			replacement := filepath.Join(root, "adapter")
			if err := os.Mkdir(replacement, 0o700); err != nil {
				return adapterPreparation{}, err
			}
			return adapterPreparation{replacement: replacement, evidence: BuildAdapter{
				Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
				ReplacementRoot: replacement, Replacement: filepath.Join(replacement, "adapter.go"), ReplacementSourceInventorySHA256: inventory,
			}}, nil
		}},
	}}}
	for _, relative := range []bool{false, true} {
		parent := t.TempDir()
		toolchainRoot := filepath.Join(parent, "toolchain")
		specRoot := toolchainRoot
		if relative {
			t.Chdir(parent)
			specRoot = "toolchain"
		}
		workingDirectory := t.TempDir()
		if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.com/target\n\nrequire example.com/adapter v1.2.3\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), []byte("example.com/adapter v1.2.3 h1:adapter\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, adapters, err := registry.prepare(target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir(), ToolchainRoot: specRoot}, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		published := filepath.Join(toolchainRoot, "adapters", "adapter@v1.2.3-0123456789abcdef")
		if len(adapters) != 1 || adapters[0].ReplacementRoot != published || adapters[0].Replacement != filepath.Join(published, "adapter.go") {
			t.Fatalf("relative=%t adapters = %#v, want replacement root %s", relative, adapters, published)
		}
		contents, err := os.ReadFile(adapters[0].BuildModFile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(string(contents), "\nreplace example.com/adapter v1.2.3 => "+published+"\n") {
			t.Fatalf("relative=%t build modfile = %s", relative, contents)
		}
	}
}
