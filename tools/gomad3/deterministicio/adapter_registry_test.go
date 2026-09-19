package deterministicio

import (
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
		{Module: "google.golang.org/grpc", Version: "v1.80.0", Sum: "h1:Xr6m2WmWZLETvUNvIUmeD5OAagMw3FiKmMlTdViWsHM="},
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
