package gomad3_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/archcheck"
	"go.temporal.io/server/tools/gomad3/internal/puritycheck"
)

const modulePath = "go.temporal.io/server/tools/gomad3"

func backendImportConfig() archcheck.Config {
	const unifiedLauncher = "fn-164-gomad-backend-layout-shared-core-with.7"
	return archcheck.Config{
		Module:         modulePath,
		Exceptions: []archcheck.Exception{
			{From: modulePath + "/backends/wasm/cmd/gomad-wasm", To: modulePath + "/cmd/gomad/api", RemovalTask: unifiedLauncher},
		},
	}
}

func TestBackendImportDirection(t *testing.T) {
	config := backendImportConfig()
	config.PureFiles = nativePureFile
	config.PureImports = []string{
		"bufio", "bytes", "cmp", "compress/gzip", "container/heap", "crypto/hmac", "crypto/sha256",
		"embed", "encoding", "encoding/base64", "encoding/binary", "encoding/hex", "encoding/json", "errors", "fmt", "hash", "io",
		"math", "math/bits", "path", "path/filepath", "reflect", "regexp", "slices", "sort", "strconv", "golang.org/x/mod/module",
		"strings", "sync", "sync/atomic", "syscall", "time", "unicode", "unicode/utf8",
		modulePath + "/choice", modulePath + "/target", modulePath + "/internal/compatibilitypack",
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
	const wasm = "example.invalid/wasm"
	const from = core + "/world"
	const to = core + "/toolchain"
	for _, test := range []struct {
		name, source, want string
		exceptions         []archcheck.Exception
		wasmSource         string
	}{
		{"existing-edge", "package world\nimport _ \"" + to + "\"\n", "", []archcheck.Exception{{From: from, To: to, RemovalTask: "fn-164.3"}}, ""},
		{"new-world-toolchain-import", "package world\nimport _ \"" + to + "\"\n", "forbidden", nil, ""},
		{"deleted-allowlist-entry", "package world\nimport _ \"" + to + "\"\n", "forbidden", nil, ""},
		{"stale-allowlist-entry", "package world\n", "stale", []archcheck.Exception{{From: from, To: to, RemovalTask: "fn-164.3"}}, ""},
		{"wasm-outside-public-import-set", "package world\n", "outside", nil, "package backend\nimport _ \"" + to + "\"\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"go.mod":                 "module " + core + "\n\ngo 1.27.1\n",
				"world/world.go":         test.source,
				"toolchain/toolchain.go": "package toolchain\n",
			}
			if test.wasmSource != "" {
				files["wasm/go.mod"] = "module " + wasm + "\n\ngo 1.27.1\n\nrequire " + core + " v0.0.0\nreplace " + core + " => ..\n"
				files["wasm/backend/backend.go"] = test.wasmSource
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
			roots := []string{root}
			if test.wasmSource != "" {
				roots = append(roots, filepath.Join(root, "wasm"))
			}
			var packages []archcheck.Package
			for _, directory := range roots {
				listed, err := archcheck.List(directory, sourceGoCommand(), archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, false)
				if err != nil {
					t.Fatal(err)
				}
				for _, pkg := range listed {
					if pkg.Error != nil || len(pkg.DepsErrors) != 0 {
						t.Fatalf("fixture did not list: %+v", pkg)
					}
					packages = append(packages, archcheck.Package{ImportPath: pkg.ImportPath, Imports: pkg.Imports})
				}
			}
			config := archcheck.Config{Module: core, WASMModule: wasm, NativePackages: []string{"toolchain"}, Exceptions: test.exceptions}
			findings := config.CheckImports(packages)
			if test.want == "" {
				if len(findings) != 0 {
					t.Fatalf("existing source edge rejected: %v", findings)
				}
				return
			}
			importer := from
			if test.wasmSource != "" {
				importer = wasm + "/backend"
			}
			if !slices.ContainsFunc(findings, func(finding string) bool {
				return strings.Contains(finding, test.want) && strings.Contains(finding, importer) && strings.Contains(finding, to)
			}) {
				t.Fatalf("missing %s diagnostic for listed %s -> %s: %v", test.want, importer, to, findings)
			}
		})
	}
}

func TestArchitectureInventoryFixtures(t *testing.T) {
	for _, fixture := range []struct{ name, category, path, source string }{
		{"root-production", "ownerless", "main.go", "package fitness\n"},
		{"hidden-production", "uncovered-source", ".hidden/source.go", "package hidden\n"},
		{"underscore-production", "uncovered-source", "_hidden/source.go", "package hidden\n"},
		{"unclassified-testdata", "uncovered-source", "record/testdata/source.go", "package hidden\n"},
		{"overlay-prefix", "package-error", "backends/native/toolchain/runtime/overlayextra/source.go", "package overlayextra\nimport _ \"example.invalid/missing\"\n"},
		{"hidden-module", "unclassified-module", ".hidden/go.mod", "module example.invalid/hidden\n\ngo 1.27.1\n"},
		{"excluded-root-module", "unclassified-module", "backends/native/transcript/testdata/extra/go.mod", "module example.invalid/extra\n\ngo 1.27.1\n"},
		{"included-list-error", "package-error", "record/source.go", "package record\nimport _ \"example.invalid/missing\"\n"},
		{"owner-edge", "owner-edge", "record/source.go", "package record\nimport _ \"example.invalid/fitness/runner\"\n"},
		{"required-facade-edge", "required-edge", "backends/native/build/source.go", "package build\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{"go.mod": "module example.invalid/fitness\n\ngo 1.27.1\n", "runner/source.go": "package runner\n", fixture.path: fixture.source}
			for _, path := range []string{"backends/native/toolchain/runtime/overlay", "cmd/gomad/testdata", "backends/native/transcript/testdata", "internal/compatibilitypack/testdata", "backends/native/conformance/testdata", "testdata", "qualification/corpus", "backends/wasm/testdata", "backends/wasm/wasi/testdata", "backends/wasm/toolchain/testdata", "backends/wasm/simulation/testdata", "backends/wasm/qualification/testdata", "backends/wasm/cmd/gomad-wasm/testdata"} {
				files[path+"/fixture.go"] = "package fixture\n"
			}
			for _, path := range []string{"backends/native/transcript/testdata/cactusstatsd", "backends/native/transcript/testdata/hashicorpmetrics", "backends/native/transcript/testdata/memberlist", "backends/native/transcript/testdata/pebble", "backends/native/transcript/testdata/sentry", "backends/native/transcript/testdata/sockaddr", "backends/native/transcript/testdata/sprig", "backends/native/transcript/testdata/validator", "internal/compatibilitypack/testdata/v041", "internal/compatibilitypack/testdata/xsys", "backends/native/conformance/testdata", "backends/native/conformance/testdata/libc_adapter", "backends/native/conformance/testdata/sqlite_adapter", "qualification/corpus", "backends/wasm/qualification/testdata/fixture"} {
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

func TestArchitecturePublicSignatureFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name, source string
		leaks        bool
	}{
		{"nested-container", "type Report struct{ Values map[string][]*leaf.Value }", true},
		{"alias", "type Report = leaf.Value", true},
		{"defined-rhs", "type Report leaf.Value", true},
		{"interface-result", "type Report interface{ Value() leaf.Value }", true},
		{"generic-constraint", "type Report[T interface{~[]leaf.Value}] struct{ Value T }", true},
		{"foreign-generic-argument", "type Report struct{ Value atomic.Pointer[leaf.Value] }", true},
		{"promoted-method", "type hidden struct{};func(hidden)Value()leaf.Value{return leaf.Value{}};type Report struct{hidden}", true},
		{"private-storage", "type Report struct{value leaf.Value}", false},
		{"private-defined-graph", "type hidden struct{Text string};type Report hidden", false},
		{"recursive-public-graph", "type Report struct{Next *Report;Values []string}", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"go.mod":                 "module example.invalid/fitness\n\ngo 1.27.1\n",
				"internal/leaf/value.go": "package leaf\ntype Value struct{Child Child};type Child struct{Text string}\n",
				"report/value.go":        "package report\nimport(\"example.invalid/fitness/internal/leaf\";\"sync/atomic\");var _ leaf.Value;var _ atomic.Pointer[int]\n" + fixture.source,
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
			platform := archcheck.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
			findings, err := puritycheck.PublicSignatures(root, sourceGoCommand(), platform, "example.invalid/outside-consumer")
			if err != nil {
				t.Fatal(err)
			}
			if (len(findings) != 0) != fixture.leaks {
				t.Fatalf("leaks=%t findings=%v", fixture.leaks, findings)
			}
			if legal, err := puritycheck.PublicSignatures(root, sourceGoCommand(), platform, "example.invalid/fitness/consumer"); err != nil || len(legal) != 0 {
				t.Fatalf("same-parent consumer rejected: %v (%v)", legal, err)
			}
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
	for _, pkg := range packages {
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
	for _, owner := range []string{"cli", "developer", "runner", "qualification", "target", "record", "artifact", "choice", "deterministicio", "world", "toolchain", "upgrade", "compatibility", "canonicaljson", "hostexec", "hostfs", "sourceinventory"} {
		if !owners[owner] {
			t.Errorf("architectural owner %s has no package", owner)
		}
	}
}

func TestPublicPackagesDoNotExportTypeAliases(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, platform := range typedSourcePlatforms() {
		findings, err := puritycheck.PublicSignatures(root, sourceGoCommand(), platform, "example.com/gomad-runner-consumer")
		if err != nil {
			t.Fatal(err)
		}
		for _, finding := range findings {
			t.Errorf("%s/%s %s %s: %s", platform.OS, platform.Arch, finding.Category, finding.Path, finding.Detail)
		}
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
			arguments := []string{"vet", "-p", "1", "-tags", "test_dep"}
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

func TestRunnerExecutionInjectionIsPrivate(t *testing.T) {
	for _, name := range []string{"runner.go", "campaign_shard_execution.go", "replay_operation.go", "minimize_operation.go", "resume.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("runner", name), nil, 0)
		if err != nil {
			t.Fatalf("parse runner source %s: %v", name, err)
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name == "Executor" || typeSpec.Name.Name == "ReplayExecutor" {
					t.Errorf("runner exposes inaccessible execution interface %s", typeSpec.Name.Name)
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok || !strings.HasSuffix(typeSpec.Name.Name, "Spec") {
					continue
				}
				for _, field := range structure.Fields.List {
					for _, fieldName := range field.Names {
						if fieldName.IsExported() && fieldName.Name == "Executor" {
							t.Errorf("runner.%s exposes inaccessible execution field", typeSpec.Name.Name)
						}
					}
				}
			}
		}
	}
}

func TestInspectionSourceOwnershipDelegation(t *testing.T) {
	for _, consumer := range []struct {
		file, function, required string
	}{
		{"cmd/gomad/internal/cli/analyze.go", "runAnalyze", "preparation.Inspect"},
		{"qualification/analysis/prepared_review.go", "PrepareCapabilityReview", "preparation.Inspect"},
		{"cmd/gomadtool/compatibility_pack.go", "runCompatibilityPackDiscover", "capabilityanalysis.PrepareCapabilityReview"},
		{"cmd/gomadtool/compatibility_pack.go", "qualifyCompatibilityPackRequest", "capabilityanalysis.PrepareCapabilityReview"},
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
					expected := modulePath + "/internal/preparation"
					if qualifier == "capabilityanalysis" {
						expected = modulePath + "/qualification/analysis"
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

func TestRunnerRequestsCompileInExternalModule(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("find Gomad module root: %v", err)
	}
	fixture, err := os.ReadFile("backends/native/conformance/testdata/runner_external/consumer.go")
	if err != nil {
		t.Fatalf("read external Runner fixture: %v", err)
	}
	directory := t.TempDir()
	module := "module example.com/gomad-runner-consumer\n\ngo 1.27.1\n\nrequire go.temporal.io/server/tools/gomad3 v0.0.0\nreplace go.temporal.io/server/tools/gomad3 => " + root + "\n"
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte(module), 0o600); err != nil {
		t.Fatalf("write external module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "consumer.go"), fixture, 0o600); err != nil {
		t.Fatalf("write external Runner fixture: %v", err)
	}
	goExecutable := os.Getenv("GOMAD3_STOCK_GO")
	if goExecutable == "" {
		goExecutable = "go"
	}
	command := exec.Command(goExecutable, "test", "-mod=mod", "-tags", "test_dep", ".")
	command.Dir = directory
	for _, variable := range os.Environ() {
		if strings.HasPrefix(variable, "GOROOT=") || strings.HasPrefix(variable, "GOBIN=") || strings.HasPrefix(variable, "GOMADSEED=") || strings.HasPrefix(variable, "GOMAD3_CHILD_SEED=") {
			continue
		}
		command.Env = append(command.Env, variable)
	}
	command.Env = append(command.Env, "GOWORK=off", "GOTOOLCHAIN=local", "GOEXPERIMENT=nogreenteagc")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compile external Runner fixture: %v\n%s", err, output)
	}
}

func TestCurrentVocabularyHasNoLegacyCampaignBoundary(t *testing.T) {
	for _, root := range []string{"cmd/gomad", "qualification", "runner"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, visitErr error) error {
			if visitErr != nil {
				return visitErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, obsolete := range []string{"partial_runs", "run journal", "batch plan", "batch lifecycle", "batch path", "batch directory", "batch record", "batch summary", "batch run", "batch_", "run_evidence", "max runs", "run timeout"} {
				if strings.Contains(string(contents), obsolete) {
					t.Errorf("current production file %s retains legacy boundary vocabulary %q", path, obsolete)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan current vocabulary under %s: %v", root, err)
		}
	}
}

func TestMakeTargetsMatchTheirOwnership(t *testing.T) {
	contents, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	makefile := string(contents)
	for _, target := range []string{"clean:", "prune-cache:", "clean-qualifications:", "test-toolchain:", "test-host:", "overlay-test:", "test-simulation:", "validate-toolchain:", "validate-compatibility:"} {
		if !strings.Contains(makefile, "\n"+target) {
			t.Errorf("Makefile is missing %s", target)
		}
	}
	for _, obsolete := range []string{"VERSION_OUTPUTS", "COMPATIBILITY_OUTPUTS", "IOWIRE_INPUTS", "IOWIRE_OUTPUTS", "patch-test:", "runner-test:", "\ntoolchain: validate "} {
		if strings.Contains(makefile, obsolete) {
			t.Errorf("Makefile retains obsolete ownership %q", obsolete)
		}
	}
	// test-builder runs the native builder under stock Go through gomadtool, so the
	// Makefile names the package once, for the patched toolchain.
	if recipes := regexp.MustCompile(`(?m)\s\./backends/native/toolchain(\s|$)`).FindAllString(makefile, -1); len(recipes) != 1 {
		t.Errorf("Makefile tests native toolchain in %d recipes, want only test-toolchain", len(recipes))
	}
}

func TestExactModuleEdges(t *testing.T) {
	packages := listHostPackages(t)
	imports := make(map[string][]string, len(packages))
	for _, pkg := range packages {
		imports[pkg.ImportPath] = pkg.Imports
	}
	for _, required := range []string{
		modulePath + "/backends/native/build/internal/build",
		modulePath + "/backends/native/build/internal/capabilityreview",
		modulePath + "/backends/native/build/internal/provenance",
	} {
		if !slices.Contains(imports[modulePath+"/backends/native/build"], required) {
			t.Errorf("native build facade does not delegate to %s", required)
		}
	}
	for _, forbidden := range []string{modulePath + "/qualification", modulePath + "/backends/native/qualification", modulePath + "/backends/native/upgrade"} {
		if slices.Contains(imports[modulePath+"/backends/native/toolchain"], forbidden) {
			t.Errorf("toolchain imports orchestration package %s", forbidden)
		}
	}
	for _, required := range []string{modulePath + "/backends/native/qualification/set", modulePath + "/backends/native/toolchain/version"} {
		if !slices.Contains(imports[modulePath+"/backends/native/upgrade"], required) {
			t.Errorf("upgrade orchestration does not depend on %s", required)
		}
	}
	// One installation description supplies build, cache and adapter
	// locations to the builder and to their ordinary consumers.
	for _, consumer := range []string{modulePath + "/backends/native/toolchain", modulePath + "/backends/native/build", modulePath + "/backends/native/transcript"} {
		if !slices.Contains(imports[consumer], modulePath+"/backends/native/toolchain/installation") {
			t.Errorf("%s does not read installation locations from backends/native/toolchain/installation", consumer)
		}
	}
	for _, imported := range imports[modulePath+"/backends/native/toolchain/installation"] {
		if strings.HasPrefix(imported, modulePath+"/") {
			t.Errorf("backends/native/toolchain/installation imports module package %s", imported)
		}
	}
	// One neutral owner digests adapter source inventories for capability
	// review and adapter preparation; target no longer exports the digest.
	for _, consumer := range []string{modulePath + "/backends/native/build", modulePath + "/backends/native/transcript"} {
		if !slices.Contains(imports[consumer], modulePath+"/internal/sourceinventory") {
			t.Errorf("%s does not digest adapter inventories through internal/sourceinventory", consumer)
		}
	}
	for _, imported := range imports[modulePath+"/internal/sourceinventory"] {
		if strings.HasPrefix(imported, modulePath+"/") && imported != modulePath+"/internal/hostfs" {
			t.Errorf("internal/sourceinventory imports module package %s", imported)
		}
	}
	if _, found := packageExports(t, "target")["DigestAdapterSourceInventory"]; found {
		t.Error("target still owns adapter source-inventory hashing")
	}
}

func TestDomainModulesDoNotExportWireFraming(t *testing.T) {
	for directory, forbidden := range map[string][]string{
		"choice": {
			"Header", "TapeHeader", "Terminal", "EncodeHeader", "DecodeHeader", "PublishHeader",
			"EncodeRecord", "DecodeRecord", "EncodeTapeHeader", "DecodeTapeHeader", "EncodeTerminal", "DecodeTerminal",
			"HeaderBytes", "RecordBytes", "TapeHeaderBytes", "TapeRecordBytes", "TapeChecksumOffset",
			"TerminalFrameBytes", "TerminalChecksumOffset", "DigestBytes", "Hash",
		},
		"backends/native/transcript": {
			"ProducedTranscriptHeader", "ExpectedTranscriptHeader", "TranscriptRecord", "Terminal",
			"MountLimits", "MountRequest", "MountChild", "MountEntry", "MountResponse",
			"EncodeProducedTranscriptHeader", "DecodeProducedTranscriptHeader", "PublishProducedTranscript",
			"EncodeExpectedTranscriptHeader", "DecodeExpectedTranscriptHeader", "EncodeTranscriptRecord",
			"DecodeTranscriptRecord", "EncodeTerminal", "DecodeTerminal", "WriteMountLookupRequest",
			"ReadMountLookupRequest", "WriteMountResponse", "ReadMountResponse", "Hash",
			"BootstrapFrameBytes", "TranscriptHeaderBytes", "TranscriptRecordBytes", "TranscriptOperationBytes",
			"TerminalFrameBytes", "MountRequestHeaderBytes", "MountResponseHeaderBytes", "DigestBytes",
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

func TestPublicPackagesDoNotExportForwardingAliases(t *testing.T) {
	for _, directory := range []string{"artifact", "choice", "captured", "volume", "backends/native/transcript", "qualification", "record", "runner", "target", "backends/native/toolchain", "backends/native/upgrade", "world"} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatalf("read package directory %s: %v", directory, err)
		}
		files := token.NewFileSet()
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(directory, entry.Name())
			file, err := parser.ParseFile(files, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, declaration := range file.Decls {
				generic, ok := declaration.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, specification := range generic.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if ok && typeSpec.Name.IsExported() && typeSpec.Assign.IsValid() {
						t.Errorf("public package %s exports forwarding alias %s", directory, typeSpec.Name.Name)
					}
				}
			}
		}
	}
}
