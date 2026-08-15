package deterministicio

import (
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/target"
)

func TestRequirementsProjectsKnownInventoryDomainsConservatively(t *testing.T) {
	closure := target.CapabilityClosure{Packages: []target.CapabilityPackage{
		{ImportPath: "example.com/dependency", Name: "dependency", Imports: []string{"path/filepath"}},
		{ImportPath: "example.com/target", Name: "main", Root: true, Imports: []string{"crypto/rand", "net", "os", "time"}},
		{ImportPath: "modernc.org/libc", Name: "libc", Module: &target.CapabilityModule{Path: "modernc.org/libc", Version: "v1.72.3"}},
	}}
	adapters := []Adapter{{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="}}

	requirements, err := Default().Requirements(closure, adapters)
	requireTestNoError(t, err)
	requireTestEqual(t, []string{"adapter:modernc.org/libc", "entropy", "filesystem", "loopback_tcp", "time"}, requirementNames(requirements))
	for _, requirement := range requirements {
		if !requirement.Modeled || len(requirement.Packages) == 0 {
			t.Fatalf("requirement = %#v", requirement)
		}
	}
	requireTestEqual(t, []target.CapabilityPackageReference{{ImportPath: "example.com/dependency", Name: "dependency"}, {ImportPath: "example.com/target", Name: "main"}}, requirements[2].Packages)
}

func TestRequirementsRejectsUnselectedAdapterIdentity(t *testing.T) {
	_, err := Default().Requirements(target.CapabilityClosure{Packages: []target.CapabilityPackage{}}, []Adapter{{Module: "modernc.org/libc", Version: "changed", Sum: "h1:changed"}})
	if err == nil || !strings.Contains(err.Error(), "unavailable or modified") {
		t.Fatalf("Requirements() error = %v", err)
	}
}

func TestRequirementsIgnoresSelectedAdapterOutsideTargetClosure(t *testing.T) {
	closure := target.CapabilityClosure{Packages: []target.CapabilityPackage{{
		ImportPath: "example.com/target", Name: "target", Root: true, Imports: []string{"time"},
	}}}
	adapters := []Adapter{{Module: "modernc.org/libc", Version: "v1.72.3", Sum: "h1:ZnDF4tXn4NBXFutMMQC4vtbTFSXhhKzR73fv0beZEAU="}}
	requirements, err := Default().Requirements(closure, adapters)
	requireTestNoError(t, err)
	requireTestEqual(t, []string{"time"}, requirementNames(requirements))
}

func requirementNames(requirements []Requirement) []string {
	names := make([]string, len(requirements))
	for index, requirement := range requirements {
		names[index] = requirement.Feature
	}
	return names
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
