package deterministicio

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/target"
	gomadversion "go.temporal.io/server/tools/gomadv3/toolchain/version"
)

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
			return adapterPreparation{replacement: root, evidence: BuildAdapter{Module: identity.Module, Version: identity.Version, Sum: identity.Sum}}, nil
		}},
	}}}
	_, _, err := registry.prepare(target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir()}, t.TempDir())
	if !IsInvalidBuildAdapterConfiguration(err) {
		t.Fatalf("adapterRegistry.prepare() error = %v", err)
	}
}

func TestProfileVerifiesSelectedAdapterIdentities(t *testing.T) {
	selected := []Adapter{{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="}}
	if err := Default().VerifyAdapters(selected); err != nil {
		t.Fatal(err)
	}
	selected[0].Version = "v1.72.4"
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
