//go:build gomad_native

package archcheck_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Native rows of the shared architecture tables; empty without the native
// selection (native_cases_default_test.go).

// Native paths no longer feed the WASM job (fn-219): the models and the
// protocol generator live in the parent module.
var nativeCIRoutes = []struct{ path, want string }{
	{"tools/gomad/internal/backends/native/build/build.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/fs.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/handles.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/local_handles.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadfs/volume.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/hash.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/simulation_wire.go", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/qualification/tests.json", "native=true\nwasm=false\n"},
	{"tools/gomad/internal/backends/native/transcript/schema/iowire_runtime.go.tmpl", "native=true\nwasm=false\n"},
}

var nativePureImports = []string{modulePath + "/internal/backends/native/compatibilitypack"}

var nativePureDependencyFiles = []string{
	"internal/backends/native/compatibilitypack/evaluation.go", "internal/backends/native/compatibilitypack/selection_pure.go",
	"internal/backends/native/compatibilitypack/schema.go",
}

var nativePurePrefixes = []string{"internal/backends/native/build/internal/capabilitypolicy/"}

var nativePureRoots = []string{"internal/backends/native/build/capability_evaluation.go"}

var nativeStrictJSONOwnerCases = []struct{ name, owner, importing string }{
	{"native", "native", modulePath + "/internal/backends/native"},
	{"compatibility", "compatibility", modulePath + "/internal/backends/native/compatibilitypack"},
	{"deterministicio", "deterministicio", modulePath + "/internal/backends/native/transcript"},
	{"toolchain", "toolchain", modulePath + "/internal/backends/native/toolchain"},
	{"upgrade", "upgrade", modulePath + "/internal/backends/native/upgrade"},
}

var nativeInventoryFixtures = []struct{ name, category, path, source string }{
	{"overlay-prefix", "package-error", "internal/backends/native/toolchain/runtime/overlayextra/source.go", "package overlayextra\nimport _ \"example.invalid/missing\"\n"},
	{"excluded-root-module", "unclassified-module", "internal/backends/native/transcript/testdata/extra/go.mod", "module example.invalid/extra\n\ngo 1.27.1\n"},
	{"required-facade-edge", "required-edge", "internal/backends/native/build/source.go", "package build\n"},
}

var nativeFixtureExclusions = []string{"internal/backends/native/toolchain/runtime/overlay", "internal/backends/native/transcript/testdata", "internal/backends/native/compatibilitypack/testdata", "internal/backends/native/conformance/testdata"}

var nativeFixtureModules = []string{"internal/backends/native/transcript/testdata/cactusstatsd", "internal/backends/native/transcript/testdata/hashicorpmetrics", "internal/backends/native/transcript/testdata/memberlist", "internal/backends/native/transcript/testdata/pebble", "internal/backends/native/transcript/testdata/sentry", "internal/backends/native/transcript/testdata/sockaddr", "internal/backends/native/transcript/testdata/sprig", "internal/backends/native/transcript/testdata/validator", "internal/backends/native/compatibilitypack/testdata/v041", "internal/backends/native/compatibilitypack/testdata/xsys", "internal/backends/native/conformance/testdata", "internal/backends/native/conformance/testdata/libc_adapter", "internal/backends/native/conformance/testdata/sqlite_adapter"}

func withNativeWireFraming(packages map[string][]string) map[string][]string {
	for directory, forbidden := range map[string][]string{
		"internal/backends/native/transcript": {
			"ProducedTranscriptHeader", "ExpectedTranscriptHeader", "TranscriptRecord", "Terminal",
			"MountLimits", "MountRequest", "MountChild", "MountEntry", "MountResponse",
			"EncodeProducedTranscriptHeader", "DecodeProducedTranscriptHeader", "PublishProducedTranscript",
			"EncodeExpectedTranscriptHeader", "DecodeExpectedTranscriptHeader", "EncodeTranscriptRecord",
			"DecodeTranscriptRecord", "EncodeTerminal", "DecodeTerminal", "WriteMountLookupRequest",
			"ReadMountLookupRequest", "WriteMountResponse", "ReadMountResponse", "Hash",
			"BootstrapFrameBytes", "TranscriptHeaderBytes", "TranscriptRecordBytes", "TranscriptOperationBytes",
			"TerminalFrameBytes", "MountRequestHeaderBytes", "MountResponseHeaderBytes", "DigestBytes",
		}} {
		packages[directory] = forbidden
	}
	return packages
}

func TestInspectionSourceOwnershipDelegation(t *testing.T) {
	for _, consumer := range []struct {
		file, function, required string
	}{
		{"cmd/gomad/internal/cli/native_enabled_analyze.go", "runAnalyze", "preparation.Inspect"},
		{"internal/backends/native/qualification/analysis/prepared_review.go", "PrepareCapabilityReview", "preparation.Inspect"},
		{"internal/backends/native/cmd/gomadtool/compatibility_pack.go", "runCompatibilityPackDiscover", "capabilityanalysis.PrepareCapabilityReview"},
		{"internal/backends/native/cmd/gomadtool/compatibility_pack.go", "qualifyCompatibilityPackRequest", "capabilityanalysis.PrepareCapabilityReview"},
	} {
		t.Run(consumer.function, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), consumer.file, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			var body *ast.BlockStmt
			for _, declaration := range file.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == consumer.function {
					body = function.Body
				}
			}
			if body == nil {
				t.Fatalf("missing consumer body %s", consumer.function)
			}
			qualifier, _, _ := strings.Cut(consumer.required, ".")
			imported := false
			for _, specification := range file.Imports {
				if specification.Name != nil && specification.Name.Name == qualifier || specification.Name == nil && strings.HasSuffix(strings.Trim(specification.Path.Value, "\""), "/"+qualifier) {
					expected := modulePath + "/internal/backends/native/preparation"
					if qualifier == "capabilityanalysis" {
						expected = modulePath + "/internal/backends/native/qualification/analysis"
					}
					imported = specification.Path.Value == "\""+expected+"\""
				}
			}
			if !imported {
				t.Fatalf("consumer %s has no owner import for %s", consumer.function, consumer.required)
			}
			found := false
			ast.Inspect(body, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if qualifier, ok := selector.X.(*ast.Ident); ok && qualifier.Name+"."+selector.Sel.Name == consumer.required {
					found = true
				}
				if slices.Contains([]string{"MkdirTemp", "Chmod", "RemoveAll", "PrepareTargetBuildAdapters", "ReviewCapabilities"}, selector.Sel.Name) {
					t.Errorf("consumer %s owns inspection operation %s", consumer.function, selector.Sel.Name)
				}
				return true
			})
			if !found {
				t.Errorf("consumer %s does not delegate through %s", consumer.function, consumer.required)
			}
		})
	}
}

// test-builder runs the native builder under stock Go through gomadtool, so the
// Makefile names the native toolchain package once, for the patched toolchain.
func TestMakefileTestsNativeToolchainOnce(t *testing.T) {
	data, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if recipes := regexp.MustCompile(`(?m)\sgo\.temporal\.io/server/tools/gomad/internal/backends/native/toolchain(\s|$)`).FindAllString(string(data), -1); len(recipes) != 1 {
		t.Errorf("Makefile tests native toolchain in %d recipes, want only test-toolchain", len(recipes))
	}
}
