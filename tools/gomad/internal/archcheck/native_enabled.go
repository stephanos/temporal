//go:build gomad_native

package archcheck

// The native backend's ownership, inventory and edge tables. They exist only
// under the native selection; the default run names no native path and finds
// the nested native module generically through the parent go.mod replace.

const NativeSelected = true

const (
	nativeModulePath = "go.temporal.io/server/tools/gomad/internal/backends/native"
	nativeDirectory  = "internal/backends/native"
	nativeOverlay    = "internal/backends/native/toolchain/runtime/overlay"
)

var nativeSourceExclusions = []string{
	"internal/backends/native/toolchain/runtime/overlay", "internal/backends/native/transcript/testdata", "internal/backends/native/conformance/testdata", "internal/backends/native/qualification/testdata/tagged",
	"internal/backends/native/compatibilitypack/testdata",
}

var nativeHostSourcePackages = []string{"internal/backends/native/toolchain/runtime/testdata/vfdpointer", "internal/backends/native/toolchain/runtime/testdata/vfdnative"}

var nativeExpectedModules = map[string]bool{
	"internal/backends/native/go.mod": true, "internal/backends/native/transcript/testdata/cactusstatsd/go.mod": true, "internal/backends/native/transcript/testdata/hashicorpmetrics/go.mod": true, "internal/backends/native/transcript/testdata/memberlist/go.mod": true, "internal/backends/native/transcript/testdata/pebble/go.mod": true,
	"internal/backends/native/transcript/testdata/sentry/go.mod": true, "internal/backends/native/transcript/testdata/sockaddr/go.mod": true, "internal/backends/native/transcript/testdata/sprig/go.mod": true, "internal/backends/native/transcript/testdata/validator/go.mod": true,
	"internal/backends/native/compatibilitypack/testdata/v041/go.mod": true, "internal/backends/native/compatibilitypack/testdata/xsys/go.mod": true, "internal/backends/native/conformance/testdata/go.mod": true, "internal/backends/native/conformance/testdata/libc_adapter/go.mod": true, "internal/backends/native/conformance/testdata/sqlite_adapter/go.mod": true,
}

func nativeOwner(relative string) (string, bool) {
	if relative == "internal/backends/native" || Within(relative, "internal/backends/native/execution") {
		return "native", true
	}
	for path, owner := range map[string]string{
		"internal/backends/native/cmd/gomadtool": "developer", "internal/backends/native/toolchain": "toolchain", "internal/backends/native/build": "target",
		"internal/backends/native/transcript": "deterministicio", "internal/backends/native/upgrade": "upgrade", "internal/backends/native/conformance": "developer",
		"internal/backends/native/generation": "developer", "internal/backends/native/qualification": "qualification", "internal/backends/native/preparation": "preparation",
		"internal/backends/native/compatibilitypack": "compatibility", "internal/backends/native/sourceinventory": "sourceinventory",
	} {
		if Within(relative, path) {
			return owner, true
		}
	}
	return "", false
}

// Module-relative packages the native-only edge rules name.
var (
	nativeFixtureImports  = []string{"internal/backends/native", "internal/backends/native/build", "internal/backends/native/transcript"}
	nativeCLIImports      = []string{"internal/backends/native"}
	nativeVersionImports  = []string{"internal/backends/native/toolchain/version", "internal/backends/native/toolchain/installation"}
	nativeUpgradeAdapters = []string{"internal/backends/native/upgrade/pinimpact", "internal/backends/native/upgrade/adapterregen"}
	nativeRequiredEdges   = map[string][]string{
		"internal/backends/native/build": {"internal/backends/native/build/internal/build", "internal/backends/native/build/internal/capabilityreview", "internal/backends/native/build/internal/provenance", "internal/backends/native/toolchain/installation", "internal/backends/native/sourceinventory"}, "internal/backends/native/upgrade": {"internal/backends/native/qualification/set", "internal/backends/native/toolchain/version"},
		"internal/backends/native/toolchain": {"internal/backends/native/toolchain/installation"}, "internal/backends/native/transcript": {"internal/backends/native/toolchain/installation", "internal/backends/native/sourceinventory"},
	}
)

func nativeModuleEdge(module, importing, imported string) bool {
	return importing == module+"/internal/backends/native/toolchain/installation" || (importing == module+"/internal/backends/native/sourceinventory" && imported != module+"/internal/host/fs") || (importing == module+"/internal/backends/native/toolchain" && (Within(imported, module+"/internal/evidence/qualification") || Within(imported, module+"/internal/backends/native/qualification") || Within(imported, module+"/internal/backends/native/upgrade")))
}
