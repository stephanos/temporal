//go:build gomad_native

package archcheck_test

import (
	"slices"
	"strings"
	"testing"
)

func TestExactModuleEdges(t *testing.T) {
	packages := listHostPackages(t)
	imports := make(map[string][]string, len(packages))
	for _, pkg := range packages {
		imports[pkg.ImportPath] = pkg.Imports
	}
	for _, required := range []string{
		modulePath + "/internal/backends/native/build/internal/build",
		modulePath + "/internal/backends/native/build/internal/capabilityreview",
		modulePath + "/internal/backends/native/build/internal/provenance",
	} {
		if !slices.Contains(imports[modulePath+"/internal/backends/native/build"], required) {
			t.Errorf("native build facade does not delegate to %s", required)
		}
	}
	for _, forbidden := range []string{modulePath + "/internal/evidence/qualification", modulePath + "/internal/backends/native/qualification", modulePath + "/internal/backends/native/upgrade"} {
		if slices.Contains(imports[modulePath+"/internal/backends/native/toolchain"], forbidden) {
			t.Errorf("toolchain imports orchestration package %s", forbidden)
		}
	}
	for _, required := range []string{modulePath + "/internal/backends/native/qualification/set", modulePath + "/internal/backends/native/toolchain/version"} {
		if !slices.Contains(imports[modulePath+"/internal/backends/native/upgrade"], required) {
			t.Errorf("upgrade orchestration does not depend on %s", required)
		}
	}
	// One installation description supplies build, cache and adapter
	// locations to the builder and to their ordinary consumers.
	for _, consumer := range []string{modulePath + "/internal/backends/native/toolchain", modulePath + "/internal/backends/native/build", modulePath + "/internal/backends/native/transcript"} {
		if !slices.Contains(imports[consumer], modulePath+"/internal/backends/native/toolchain/installation") {
			t.Errorf("%s does not read installation locations from internal/backends/native/toolchain/installation", consumer)
		}
	}
	for _, imported := range imports[modulePath+"/internal/backends/native/toolchain/installation"] {
		if strings.HasPrefix(imported, modulePath+"/") {
			t.Errorf("internal/backends/native/toolchain/installation imports module package %s", imported)
		}
	}
	// One neutral owner digests adapter source inventories for capability
	// review and adapter preparation; target no longer exports the digest.
	for _, consumer := range []string{modulePath + "/internal/backends/native/build", modulePath + "/internal/backends/native/transcript"} {
		if !slices.Contains(imports[consumer], modulePath+"/internal/backends/native/sourceinventory") {
			t.Errorf("%s does not digest adapter inventories through internal/backends/native/sourceinventory", consumer)
		}
	}
	for _, imported := range imports[modulePath+"/internal/backends/native/sourceinventory"] {
		if strings.HasPrefix(imported, modulePath+"/") && imported != modulePath+"/internal/host/fs" {
			t.Errorf("internal/backends/native/sourceinventory imports module package %s", imported)
		}
	}
	if _, found := packageExports(t, "internal/backends/target")["DigestAdapterSourceInventory"]; found {
		t.Error("target still owns adapter source-inventory hashing")
	}
}
