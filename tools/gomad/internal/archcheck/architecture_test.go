package archcheck_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/archcheck"
)

const modulePath = "go.temporal.io/server/tools/gomad"

func TestToolPackageLayout(t *testing.T) {
	for _, layout := range []struct {
		root   string
		groups []string
	}{
		{".", []string{"cmd", "internal", "rust", "scripts", "testdata"}},
		{"internal", []string{"archcheck", "backends", "evidence", "model", "runner", "support"}},
		{"cmd", []string{"gomad"}},
	} {
		entries, err := os.ReadDir(layout.root)
		if err != nil {
			t.Fatal(err)
		}
		var groups []string
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if entry.IsDir() {
				groups = append(groups, entry.Name())
			} else if filepath.Ext(entry.Name()) == ".go" {
				t.Errorf("owner group root %s holds Go source %s", layout.root, entry.Name())
			}
		}
		if !slices.Equal(groups, layout.groups) {
			t.Errorf("owner groups under %s = %v, want %v", layout.root, groups, layout.groups)
		}
	}
}

// TestMain runs the module's architecture and CI routing tests from the
// module root, the directory their source, Makefile and workflow paths name.
func TestMain(m *testing.M) {
	if err := os.Chdir(filepath.Join("..", "..")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func backendImportConfig() archcheck.Config {
	return archcheck.Config{
		Module: modulePath,
	}
}

func TestBackendImportDirection(t *testing.T) {
	config := backendImportConfig()
	config.PureFiles = pureFile
	config.PureImports = []string{
		"bufio", "bytes", "cmp", "compress/gzip", "container/heap", "crypto/hmac", "crypto/sha256",
		"embed", "encoding", "encoding/base64", "encoding/binary", "encoding/hex", "encoding/json", "errors", "fmt", "hash", "io", "io/fs",
		"math", "math/bits", "path", "path/filepath", "reflect", "regexp", "slices", "sort", "strconv", "golang.org/x/mod/module",
		"strings", "sync", "sync/atomic", "syscall", "time", "unicode", "unicode/utf8",
		"github.com/go-json-experiment/json", "github.com/go-json-experiment/json/jsontext", "github.com/go-json-experiment/json/v1",
		modulePath + "/internal/model/choice", modulePath + "/internal/backends/target",
	}
	platforms := qualifiedSourcePlatforms()
	host := archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
	if !slices.Contains(platforms, host) {
		platforms = append(platforms, host)
	}
	for _, platform := range platforms {
		started := time.Now()
		findings, err := archcheck.Check(".", sourceGoCommand(), platform, config)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range findings {
			t.Errorf("%s/%s %s %s: %s", platform.OS, platform.Arch, finding.Category, finding.Path, finding.Detail)
		}
		t.Logf("structural checker on %s/%s in %s", platform.OS, platform.Arch, elapsed)
		if platform == host && elapsed >= 5*time.Second {
			t.Errorf("host structural checker took %s (limit 5s)", elapsed)
		}
	}
}
func TestBackendImportFixtures(t *testing.T) {
	const core = "example.invalid/fitness"
	const to = core + "/internal/backends/legacy/toolchain"
	for _, test := range []struct {
		name, from, want string
	}{
		{"shared-legacy-import", core + "/internal/model/world", "forbidden"},
		{"wasm-legacy-import", core + "/internal/backends/wasm", "forbidden"},
		{"legacy-legacy-import", core + "/internal/backends/legacy/build", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"go.mod": "module " + core + "\n\ngo 1.27.1\n",
				strings.TrimPrefix(test.from, core+"/") + "/source.go": "package source\nimport _ \"" + to + "\"\n",
				"internal/backends/legacy/toolchain/toolchain.go":      "package toolchain\n",
			}
			for name, source := range files {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			listed, err := archcheck.List(root, sourceGoCommand(), archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, false)
			if err != nil {
				t.Fatal(err)
			}
			var packages []archcheck.Package
			for _, pkg := range listed {
				if pkg.Error != nil || len(pkg.DepsErrors) != 0 {
					t.Fatalf("fixture did not list: %+v", pkg)
				}
				packages = append(packages, archcheck.Package{ImportPath: pkg.ImportPath, Imports: pkg.Imports})
			}
			findings := archcheck.Config{Module: core}.CheckImports(packages)
			if test.want == "" {
				if len(findings) != 0 {
					t.Fatalf("existing source edge rejected: %v", findings)
				}
				return
			}
			if !slices.ContainsFunc(findings, func(finding string) bool {
				return strings.Contains(finding, test.want) && strings.Contains(finding, test.from) && strings.Contains(finding, to)
			}) {
				t.Fatalf("missing %s diagnostic for listed %s -> %s: %v", test.want, test.from, to, findings)
			}
		})
	}
}

func TestArchitectureInventoryFixtures(t *testing.T) {
	for _, fixture := range []struct{ name, category, path, source string }{
		{"root-production", "ownerless", "main.go", "package fitness\n"},
		{"hidden-production", "uncovered-source", ".hidden/source.go", "package hidden\n"},
		{"underscore-production", "uncovered-source", "_hidden/source.go", "package hidden\n"},
		{"unclassified-testdata", "uncovered-source", "internal/evidence/record/testdata/source.go", "package hidden\n"},
		{"hidden-module", "unclassified-module", ".hidden/go.mod", "module example.invalid/hidden\n\ngo 1.27.1\n"},
		{"included-list-error", "package-error", "internal/evidence/record/source.go", "package record\nimport _ \"example.invalid/missing\"\n"},
		{"owner-edge", "owner-edge", "internal/evidence/record/source.go", "package record\nimport _ \"example.invalid/fitness/internal/runner\"\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"go.mod": "module example.invalid/fitness\n\ngo 1.27.1\n", "internal/runner/source.go": "package runner\n", fixture.path: fixture.source}
			for _, path := range []string{"cmd/gomad/testdata", "internal/runner/testdata", "internal/evidence/qualification/corpus", "internal/backends/wasm/testdata", "internal/model/wasi/testdata", "internal/backends/wasm/toolchain/testdata", "internal/backends/wasm/cluster/testdata", "internal/backends/wasm/qualification/testdata", "internal/backends/wasm/cmd/gomad-wasm/testdata", "testdata/parity/site_scope"} {
				files[path+"/fixture.go"] = "package fixture\n"
			}
			for _, path := range []string{"internal/evidence/qualification/corpus", "internal/backends/wasm/qualification/testdata/fixture"} {
				files[path+"/go.mod"] = "module example.invalid/fixture\n\ngo 1.27.1\n"
				files[path+"/fixture.go"] = "package fixture\n"
			}
			for name, source := range files {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, findings, err := archcheck.Discover(root, sourceGoCommand(), archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
			if err != nil {
				t.Fatal(err)
			}
			for _, finding := range findings {
				if finding.Category == fixture.category {
					return
				}
			}
			t.Fatalf("missing %s for %s: %v", fixture.category, fixture.path, findings)
		})
	}
}

type listedPackage struct {
	ImportPath string
	Imports    []string
}

func TestPackageArchitecture(t *testing.T) {
	packages := listHostPackages(t)
	owners := map[string]bool{}
	logstore := modulePath + "/internal/runner/internal/logstore"
	for _, pkg := range packages {
		if pkg.ImportPath == logstore {
			owners[logstore] = true
			for _, imported := range pkg.Imports {
				if first, _, _ := strings.Cut(imported, "/"); strings.Contains(first, ".") && imported != modulePath+"/internal/support/fs" {
					t.Errorf("shared log owner imports %s; it may use only the standard library and internal/hostfs", imported)
				}
			}
		}
		owner := packageOwner(pkg.ImportPath)
		if owner == "" {
			t.Errorf("package %s has no architectural owner", pkg.ImportPath)
			continue
		}
		owners[owner] = true
		for _, imported := range pkg.Imports {
			if !strings.HasPrefix(imported, modulePath+"/") {
				continue
			}
			importedOwner := packageOwner(imported)
			if importedOwner == "" {
				t.Errorf("package %s imports ownerless package %s", pkg.ImportPath, imported)
				continue
			}
			if !ownerMayImport(owner, importedOwner, pkg.ImportPath, imported) {
				t.Errorf("owner %s package %s imports forbidden owner %s package %s", owner, pkg.ImportPath, importedOwner, imported)
			}
			if !moduleMayImport(pkg.ImportPath, imported) {
				t.Errorf("package %s imports forbidden module edge %s", pkg.ImportPath, imported)
			}
		}
	}

	for imported, allowed := range map[string]bool{modulePath + "/internal/support/fs": true, modulePath + "/internal/model/strictjson": false, modulePath + "/internal/runner/internal/campaign": false, modulePath + "/internal/runner/internal/corpus": false, modulePath + "/internal/evidence/record": false} {
		if moduleMayImport(logstore, imported) != allowed {
			t.Errorf("shared log owner edge to %s admitted = %t", imported, !allowed)
		}
	}
	if moduleMayImport(modulePath+"/internal/runner/internal/exploration", logstore) {
		t.Error("pure exploration engines may import the shared log owner")
	}
	for _, owner := range []string{"developer", "choice", "world", "strictjson", "hostfs"} {
		if !owners[owner] {
			t.Errorf("architectural owner %s has no package", owner)
		}
	}
}

func TestStrictJSONOwnerEdges(t *testing.T) {
	imported := modulePath + "/internal/model/strictjson"
	if got := packageOwner(imported); got != "strictjson" {
		t.Fatalf("strict JSON owner = %q, want strictjson", got)
	}
	for _, test := range []struct {
		name, owner, importing string
	}{
		{"wasi-reference", "wasireference", modulePath + "/internal/model/wasi"},
		{"sqlite-native", "developer", modulePath + "/internal/model/cmd/sqlite-native"},
		{"wasm-qualification", "wasm", modulePath + "/internal/backends/wasm/qualification"},
		{"wasm-installation", "wasm", modulePath + "/internal/backends/wasm/installation"},
		{"wasm-simulation", "wasm", modulePath + "/internal/backends/wasm/cluster"},
		{"cli", "cli", modulePath + "/cmd/gomad/internal/cli"},
		{"runner", "runner", modulePath + "/internal/runner"},
		{"qualification", "qualification", modulePath + "/internal/evidence/qualification"},
		{"target", "target", modulePath + "/internal/backends/target"},
		{"record", "record", modulePath + "/internal/evidence/record"},
		{"world", "world", modulePath + "/internal/model/world"},
		{"simulation", "simulation", modulePath + "/internal/model/clock"},
		{"captured", "captured", modulePath + "/internal/model/capture"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !ownerMayImport(test.owner, "strictjson", test.importing, imported) {
				t.Errorf("owner %s may not import strictjson", test.owner)
			}
		})
	}
}

func TestWASMHostFSOwnerEdges(t *testing.T) {
	for _, test := range []struct {
		name    string
		allowed bool
	}{
		{"backend", true},
		{"wasi", true},
		{"cluster", true},
		{"qualification", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			importing := modulePath + "/internal/backends/wasm/" + test.name
			if got := ownerMayImport("wasm", "internal/support/fs", importing, modulePath+"/internal/support/fs"); got != test.allowed {
				t.Errorf("%s may import internal/hostfs = %t, want %t", importing, got, test.allowed)
			}
		})
	}
	if ownerMayImport("wasm", "hostfs", modulePath+"/internal/model/wasi", modulePath+"/internal/support/fs/unchecked") {
		t.Fatal("WASM host filesystem descendant bypass admitted")
	}
}

func qualifiedSourcePlatforms() []archcheck.Platform {
	return []archcheck.Platform{{OS: "darwin", Arch: "arm64"}, {OS: "linux", Arch: "amd64"}}
}

func TestHostPackageVet(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	platforms := qualifiedSourcePlatforms()
	host := archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
	if !slices.Contains(platforms, host) {
		platforms = append(platforms, host)
	}
	for _, platform := range platforms {
		t.Run(platform.OS+"/"+platform.Arch, func(t *testing.T) {
			inventory, findings, err := archcheck.Discover(root, sourceGoCommand(), platform)
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != 0 {
				t.Fatalf("invalid host inventory: %+v", findings)
			}
			arguments := []string{"vet", "-p", "1", "-tags", archcheck.SelectionTags()}
			for _, pkg := range inventory.Packages {
				arguments = append(arguments, pkg.ImportPath)
			}
			if len(inventory.Packages) == 0 {
				t.Fatal("empty host inventory")
			}
			encoded, err := json.Marshal(inventory)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("validated source inventory: %s", encoded)
			command := exec.Command(sourceGoCommand(), arguments...)
			command.Dir, command.Env = root, archcheck.Environment(platform)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("vet %d host packages: %v\n%s", len(inventory.Packages), err, output)
			}
			t.Logf("vetted %d complete host packages", len(inventory.Packages))
		})
	}
}

func TestMakeTargetsMatchTheirOwnership(t *testing.T) {
	contents, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	makefile := string(contents)
	for _, target := range []string{"generate:", "validate:", "validate-protocol:", "validate-replay-plan:", "validate-wasm-determinism:", "build:", "lint-code:", "test-harness:", "world-test:", "test-wasm:", "test:"} {
		if !strings.Contains(makefile, "\n"+target) {
			t.Errorf("Makefile is missing %s", target)
		}
	}
	for _, obsolete := range []string{"VERSION_OUTPUTS", "COMPATIBILITY_OUTPUTS", "IOWIRE_INPUTS", "IOWIRE_OUTPUTS", "patch-test:", "runner-test:", "\ntoolchain: validate "} {
		if strings.Contains(makefile, obsolete) {
			t.Errorf("Makefile retains obsolete ownership %q", obsolete)
		}
	}
}

func TestDomainModulesDoNotExportWireFraming(t *testing.T) {
	for directory, forbidden := range map[string][]string{
		"internal/model/choice": {
			"Header", "TapeHeader", "Terminal", "EncodeHeader", "DecodeHeader", "PublishHeader",
			"EncodeRecord", "DecodeRecord", "EncodeTapeHeader", "DecodeTapeHeader", "EncodeTerminal", "DecodeTerminal",
			"HeaderBytes", "RecordBytes", "TapeHeaderBytes", "TapeRecordBytes", "TapeChecksumOffset",
			"TerminalFrameBytes", "TerminalChecksumOffset", "DigestBytes", "Hash",
		},
	} {
		exported := packageExports(t, directory)
		for _, name := range forbidden {
			if exported[name] {
				t.Errorf("%s exports raw wire implementation %s", directory, name)
			}
		}
	}
}

func packageExports(t *testing.T, directory string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read package directory %s: %v", directory, err)
	}
	exported := map[string]bool{}
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, filepath.Join(directory, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse %s/%s: %v", directory, entry.Name(), err)
		}
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				exported[declaration.Name.Name] = declaration.Name.IsExported()
			case *ast.GenDecl:
				for _, specification := range declaration.Specs {
					switch specification := specification.(type) {
					case *ast.TypeSpec:
						exported[specification.Name.Name] = specification.Name.IsExported()
					case *ast.ValueSpec:
						for _, name := range specification.Names {
							exported[name.Name] = name.IsExported()
						}
					default:
					}
				}
			default:
			}
		}
	}
	return exported
}

func listHostPackages(t *testing.T) []listedPackage {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var packages []listedPackage
	for _, platform := range qualifiedSourcePlatforms() {
		inventory, findings, err := archcheck.Discover(root, sourceGoCommand(), platform)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range findings {
			t.Errorf("%s/%s %s %s: %s", platform.OS, platform.Arch, finding.Category, finding.Path, finding.Detail)
		}
		for _, pkg := range inventory.Packages {
			packages = append(packages, listedPackage{pkg.ImportPath, pkg.Imports})
		}
	}
	return packages
}

func packageOwner(importPath string) string { return archcheck.Owner(modulePath, importPath) }

func ownerMayImport(owner, importedOwner, importing, imported string) bool {
	return archcheck.OwnerMayImport(modulePath, owner, importedOwner, importing, imported)
}

func moduleMayImport(importing, imported string) bool {
	return archcheck.ModuleMayImport(modulePath, importing, imported)
}

var pureDependencyFiles = []string{
	"internal/model/choice/tape.go", "internal/model/choice/divergence.go", "internal/model/choice/no_op_select.go",
	"internal/model/choice/wire.go", "internal/model/choice/trace.go", "internal/model/choice/interrupted.go",
	"internal/model/choice/limits.go", "internal/model/choice/internal/wire/wire_generated.go",
	"internal/model/strictjson/decode.go", "internal/model/strictjson/identity.go",
}

func TestStrictJSONPureDependencyInventoryIsExact(t *testing.T) {
	for _, path := range []string{"internal/model/strictjson/decode.go", "internal/model/strictjson/identity.go"} {
		if !pureFile(path) {
			t.Errorf("strict JSON memory-only source is not pure: %s", path)
		}
	}
	for _, path := range []string{"internal/model/strictjson/identity_test.go", "internal/model/strictjson/host.go"} {
		if pureFile(path) {
			t.Errorf("strict JSON source was broadly classified pure: %s", path)
		}
	}
}

func pureFile(path string) bool {
	for _, dependency := range pureDependencyFiles {
		if path == dependency {
			return true
		}
	}
	for _, prefix := range []string{"internal/model/world/mailbox/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if filepath.ToSlash(filepath.Dir(path)) == "internal/model/world" {
		return true
	}
	return path == "internal/model/clock/simulation_progress.go"
}

//nolint:staticcheck // Tests execute on the compiling host and require that exact reviewed Go toolchain.
func sourceGoCommand() string { return filepath.Join(runtime.GOROOT(), "bin", "go") }

func typedSourcePlatforms() []archcheck.Platform {
	platforms := []archcheck.Platform{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	if os.Getenv("GOMAD_ARCH_ALL_PLATFORMS") == "1" || os.Getenv("CI") != "" {
		for _, platform := range []archcheck.Platform{{OS: "darwin", Arch: "arm64"}, {OS: "linux", Arch: "amd64"}} {
			if platform != platforms[0] {
				platforms = append(platforms, platform)
			}
		}
	}
	return platforms
}

func TestPureModulesHaveNoHostEffects(t *testing.T) {
	paths := append([]string{"internal/model/clock/simulation_progress.go"}, pureDependencyFiles...)
	if err := pureFileInventory(".", paths); err != nil {
		t.Fatal(err)
	}
	for _, platform := range typedSourcePlatforms() {
		started := time.Now()
		findings, err := archcheck.CheckPurity(".", sourceGoCommand(), platform, pureFile)
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range findings {
			t.Errorf("%s/%s %s %s: %s", platform.OS, platform.Arch, finding.Category, finding.Path, finding.Detail)
		}
		elapsed := time.Since(started)
		t.Logf("typed pure files on %s/%s in %s", platform.OS, platform.Arch, elapsed)
		if platform.OS == runtime.GOOS && platform.Arch == runtime.GOARCH && elapsed >= 30*time.Second {
			t.Errorf("host pure checker took %s (limit 30s)", elapsed)
		}
	}
}

func pureFileInventory(root string, paths []string) error {
	for _, path := range paths {
		info, err := os.Stat(filepath.Join(root, path))
		if err != nil {
			return fmt.Errorf("required pure file %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("required pure file %s is not regular source", path)
		}
	}
	return nil
}

func TestPureFileInventoryRejectsMissingAndRenamed(t *testing.T) {
	for _, name := range []string{"missing", "renamed"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			paths := []string{"original.go", "dependency.go"}
			for _, path := range paths {
				if err := os.WriteFile(filepath.Join(root, path), []byte("package fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := pureFileInventory(root, paths); err != nil {
				t.Fatal(err)
			}
			if name == "missing" {
				if err := os.Remove(filepath.Join(root, "dependency.go")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Rename(filepath.Join(root, "dependency.go"), filepath.Join(root, "renamed.go")); err != nil {
					t.Fatal(err)
				}
			}
			if err := pureFileInventory(root, paths); err == nil || !strings.Contains(err.Error(), "dependency.go") {
				t.Fatalf("missing exact dependency admitted: %v", err)
			}
		})
	}
}

func checkerBudget(root string) (production, tests int, err error) {
	for _, budget := range []struct {
		directory string
		limit     int
	}{{"structure", 700}, {"purity", 2000}} {
		directory, before := budget.directory, production
		err = filepath.WalkDir(filepath.Join(root, "internal/archcheck"), func(path string, entry os.DirEntry, visitErr error) error {
			if visitErr != nil {
				return visitErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || moduleArchitectureTest(root, path) {
				return nil
			}
			structural := slices.Contains([]string{"check.go", "check_test.go", "edges.go", "findings.go", "imports.go", "imports_test.go", "inventory.go"}, filepath.Base(path))
			if structural != (directory == "structure") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lines := bytes.Count(data, []byte{'\n'})
			if len(data) != 0 && data[len(data)-1] != '\n' {
				lines++
			}
			if strings.HasSuffix(path, "_test.go") {
				tests += lines
			} else {
				production += lines
			}
			return nil
		})
		if err != nil {
			return production, tests, err
		}
		if production-before > budget.limit {
			return production, tests, fmt.Errorf("architecture checker budget exceeded: %s has %d production lines (limit %d)", directory, production-before, budget.limit)
		}
	}
	if production > 2500 || tests > 1500 {
		err = fmt.Errorf("architecture checker budget exceeded: %d production (limit 2500), %d tests (limit 1500)", production, tests)
	}
	return production, tests, err
}

// moduleArchitectureTest reports the module's own architecture and CI routing
// tests, which live beside the checker but are not part of its budget.
func moduleArchitectureTest(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && slices.Contains([]string{
		"internal/archcheck/architecture_test.go",
		"internal/archcheck/integration_test.go",
	}, filepath.ToSlash(relative))
}

func TestCheckerBudgetExcludesModuleContractTests(t *testing.T) {
	for _, name := range []string{"architecture_test.go", "integration_test.go"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			directory := filepath.Join(root, "internal/archcheck")
			if err := os.MkdirAll(directory, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, name), []byte(strings.Repeat("\n", 1501)), 0600); err != nil {
				t.Fatal(err)
			}
			production, tests, err := checkerBudget(root)
			if err != nil || production != 0 || tests != 0 {
				t.Fatalf("module contract test counted as checker implementation: production=%d tests=%d err=%v", production, tests, err)
			}
		})
	}
}

func TestArchitectureCheckerBudget(t *testing.T) {
	production, tests, err := checkerBudget(".")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("checker lines: %d production (limit 2500), %d tests (limit 1500)", production, tests)
}

func TestArchitectureCheckerBudgetRejectsOversize(t *testing.T) {
	for _, test := range []struct {
		name, directory, file string
		lines                 int
	}{
		{"production", "internal/archcheck", "purity.go", 1999}, {"tests", "internal/archcheck", "purity_test.go", 1501},
		{"structure", "internal/archcheck", "check.go", 701}, {"purity", "internal/archcheck", "purity.go", 2001},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for _, directory := range []string{"internal/archcheck"} {
				if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
					t.Fatal(err)
				}
			}
			source := "package archcheck\n" + strings.Repeat("// oversized\n", test.lines-1)
			if err := os.WriteFile(filepath.Join(root, test.directory, test.file), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			if test.name == "production" {
				if err := os.WriteFile(filepath.Join(root, "internal/archcheck", "check.go"), []byte(strings.Repeat("\n", 600)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			production, tests, err := checkerBudget(root)
			if err == nil || !strings.Contains(err.Error(), "budget exceeded") {
				t.Fatalf("oversized %s accepted: %d production, %d tests", test.name, production, tests)
			}
			if test.name == "production" && production != 2599 || test.name == "tests" && tests != 1501 {
				t.Fatalf("wrong line observation: %d production, %d tests", production, tests)
			}
		})
	}
}

func TestArchitectureEffectFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/architecture/effects.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name, Category, Detail string
		Files                  map[string]string
		PureFiles              []string `json:"pure_files"`
		Reviewed               bool
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 36 {
		t.Fatalf("effect fixtures=%d want 36", len(fixtures))
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/fitness\n\ngo 1.27.1\n"), 0600); err != nil {
				t.Fatal(err)
			}
			for name, path := range fixture.Files {
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, source, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, path := range fixture.PureFiles {
				if _, exists := fixture.Files[path]; !exists {
					t.Fatalf("unknown pure fixture file %s", path)
				}
			}
			check := func() []archcheck.Finding {
				findings, err := archcheck.CheckPurity(root, sourceGoCommand(),
					archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH},
					func(path string) bool { return slices.Contains(fixture.PureFiles, path) })
				if err != nil {
					t.Fatal(err)
				}
				return findings
			}
			findings := check()
			category := fixture.Category
			if fixture.Reviewed {
				category = "dynamic-call"
			}
			if category != "" && len(findings) == 0 {
				t.Fatalf("missing %s (%s)", category, fixture.Detail)
			}
			for _, finding := range findings {
				if finding.Category != category || !strings.Contains(finding.Detail, fixture.Detail) {
					t.Fatalf("want %s (%s): %+v", category, fixture.Detail, findings)
				}
			}
			if category != "dynamic-call" {
				return
			}
			linesByFile := map[string]map[int]bool{}
			for _, finding := range findings {
				parts := strings.Split(finding.Path, ":")
				if len(parts) != 3 {
					t.Fatalf("invalid finding position %s", finding.Path)
				}
				line, err := strconv.Atoi(parts[1])
				if err != nil {
					t.Fatal(err)
				}
				if linesByFile[parts[0]] == nil {
					linesByFile[parts[0]] = map[int]bool{}
				}
				linesByFile[parts[0]][line-1] = true
			}
			for name, marked := range linesByFile {
				path := filepath.Join(root, name)
				source, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				lines := strings.Split(string(source), "\n")
				for line := range marked {
					lines[line] += " //gomad:dynamic reviewed 2026-10-12 callback boundary is explicitly reviewed by this fixture"
				}
				if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if marked := check(); len(marked) != 0 {
				t.Fatalf("reviewed callback rejected: %+v", marked)
			}
		})
	}
}
