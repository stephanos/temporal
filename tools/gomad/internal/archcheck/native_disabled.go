//go:build !gomad_native

package archcheck

// Without the native selection the native tables are empty; see
// native_enabled.go.

const NativeSelected = false

const (
	nativeModulePath = ""
	nativeDirectory  = ""
	nativeOverlay    = ""
)

var (
	nativeSourceExclusions   []string
	nativeHostSourcePackages []string
	nativeExpectedModules    map[string]bool
	nativeFixtureImports     []string
	nativeCLIImports         []string
	nativeVersionImports     []string
	nativeUpgradeAdapters    []string
	nativeRequiredEdges      map[string][]string
)

func nativeOwner(string) (string, bool) { return "", false }

func nativeModuleEdge(string, string, string) bool { return false }
