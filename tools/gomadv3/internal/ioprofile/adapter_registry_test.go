package ioprofile

import (
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
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

func TestProfileVerifiesSelectedAdapterIdentities(t *testing.T) {
	selected := []record.TargetAdapter{{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="}}
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
