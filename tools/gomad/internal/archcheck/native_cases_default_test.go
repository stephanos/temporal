//go:build !gomad_native

package archcheck_test

// Without the native selection the native rows of the shared architecture
// tables are empty; see native_cases_test.go.

var (
	nativeCIRoutes             []struct{ path, want string }
	nativePureImports          []string
	nativePureDependencyFiles  []string
	nativePurePrefixes         []string
	nativePureRoots            []string
	nativeStrictJSONOwnerCases []struct{ name, owner, importing string }
	nativeInventoryFixtures    []struct{ name, category, path, source string }
	nativeFixtureExclusions    []string
	nativeFixtureModules       []string
)

func withNativeWireFraming(packages map[string][]string) map[string][]string { return packages }
