package gomad3_test

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const modulePath = "go.temporal.io/server/tools/gomad3"

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
	for _, owner := range []string{"cli", "developer", "runner", "qualification", "target", "record", "artifact", "choice", "deterministicio", "world", "toolchain", "upgrade", "compatibility", "canonicaljson", "hostexec", "hostfs"} {
		if !owners[owner] {
			t.Errorf("architectural owner %s has no package", owner)
		}
	}
}

func TestPublicPackagesDoNotExportTypeAliases(t *testing.T) {
	for _, directory := range []string{"artifact", "choice", "deterministicio", "qualification", "record", "runner", "target", "toolchain", "upgrade", "world"} {
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

func TestRunnerRequestsCompileInExternalModule(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("find Gomad module root: %v", err)
	}
	fixture, err := os.ReadFile("internal/gomadtool/conformance/testdata/runner_external/consumer.go")
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
	// test-builder runs ./toolchain under stock Go through gomadtool, so the
	// Makefile names the package once, for the patched toolchain.
	if recipes := regexp.MustCompile(`(?m)\s\./toolchain(\s|$)`).FindAllString(makefile, -1); len(recipes) != 1 {
		t.Errorf("Makefile tests ./toolchain in %d recipes, want only test-toolchain", len(recipes))
	}
}

func TestExactModuleEdges(t *testing.T) {
	packages := listHostPackages(t)
	imports := make(map[string][]string, len(packages))
	for _, pkg := range packages {
		imports[pkg.ImportPath] = pkg.Imports
	}
	for _, required := range []string{
		modulePath + "/target/internal/build",
		modulePath + "/target/internal/capabilityreview",
		modulePath + "/target/internal/provenance",
	} {
		if !slices.Contains(imports[modulePath+"/target"], required) {
			t.Errorf("target facade does not delegate to %s", required)
		}
	}
	for _, forbidden := range []string{modulePath + "/qualification", modulePath + "/upgrade"} {
		if slices.Contains(imports[modulePath+"/toolchain"], forbidden) {
			t.Errorf("toolchain imports orchestration package %s", forbidden)
		}
	}
	for _, required := range []string{modulePath + "/qualification/set", modulePath + "/toolchain/version"} {
		if !slices.Contains(imports[modulePath+"/upgrade"], required) {
			t.Errorf("upgrade orchestration does not depend on %s", required)
		}
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
		"deterministicio": {
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
	arguments := []string{
		"list", "-json", "-tags", "test_dep",
		"./cmd/...", "./runner/...", "./qualification/...", "./target/...",
		"./record/...", "./artifact/...", "./choice/...", "./deterministicio/...", "./world/...",
		"./upgrade/...",
		"./toolchain", "./toolchain/version", "./internal/...",
	}
	command := exec.Command("go", arguments...)
	command.Env = append(command.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list Gomad v3 host packages: %v\n%s", err, output)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	packages := []listedPackage{}
	for decoder.More() {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("decode listed package: %v", err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

func packageOwner(importPath string) string {
	relative := strings.TrimPrefix(importPath, modulePath+"/")
	switch {
	case relative == "cmd/gomad" || strings.HasPrefix(relative, "cmd/gomad/internal/cli"):
		return "cli"
	case relative == "cmd/gomadtool" || strings.HasPrefix(relative, "cmd/gomadtool/") || relative == "internal/gomadtool" || strings.HasPrefix(relative, "internal/gomadtool/"):
		return "developer"
	case relative == "internal/compatibilitypack" || strings.HasPrefix(relative, "internal/compatibilitypack/"):
		return "compatibility"
	case relative == "upgrade" || strings.HasPrefix(relative, "upgrade/"):
		return "upgrade"
	case relative == "runner" || strings.HasPrefix(relative, "runner/"):
		return "runner"
	case relative == "qualification" || strings.HasPrefix(relative, "qualification/"):
		return "qualification"
	case relative == "target" || strings.HasPrefix(relative, "target/"):
		return "target"
	case relative == "record" || strings.HasPrefix(relative, "record/"):
		return "record"
	case relative == "artifact" || strings.HasPrefix(relative, "artifact/"):
		return "artifact"
	case relative == "choice" || strings.HasPrefix(relative, "choice/"):
		return "choice"
	case relative == "deterministicio" || strings.HasPrefix(relative, "deterministicio/"):
		return "deterministicio"
	case relative == "world" || strings.HasPrefix(relative, "world/"):
		return "world"
	case relative == "simulation" || strings.HasPrefix(relative, "simulation/"):
		return "simulation"
	case relative == "toolchain" || strings.HasPrefix(relative, "toolchain/"):
		return "toolchain"
	case relative == "internal/canonicaljson" || strings.HasPrefix(relative, "internal/canonicaljson/"):
		return "canonicaljson"
	case relative == "internal/hostexec" || strings.HasPrefix(relative, "internal/hostexec/"):
		return "hostexec"
	case relative == "internal/hostfs" || strings.HasPrefix(relative, "internal/hostfs/"):
		return "hostfs"
	case relative == "internal/preparation" || strings.HasPrefix(relative, "internal/preparation/"):
		return "preparation"
	default:
		return ""
	}
}

func ownerMayImport(owner, importedOwner, importing, imported string) bool {
	if owner == importedOwner {
		return true
	}
	allowed := map[string][]string{
		"cli":             {"runner", "qualification", "target", "record", "artifact", "deterministicio", "preparation", "toolchain", "canonicaljson"},
		"developer":       {"choice", "compatibility", "qualification", "simulation", "toolchain", "upgrade", "hostexec", "hostfs"},
		"runner":          {"target", "record", "artifact", "choice", "deterministicio", "preparation", "world", "canonicaljson", "hostexec", "hostfs"},
		"qualification":   {"runner", "target", "record", "artifact", "choice", "deterministicio", "preparation", "canonicaljson", "hostexec", "hostfs"},
		"target":          {"compatibility", "record", "toolchain", "canonicaljson", "hostexec", "hostfs"},
		"record":          {"canonicaljson"},
		"artifact":        {"choice", "deterministicio", "target", "record", "hostfs"},
		"compatibility":   {"target", "record", "canonicaljson", "hostfs"},
		"deterministicio": {"target", "record", "toolchain", "canonicaljson", "hostfs"},
		"preparation":     {"target", "deterministicio", "record"},
		"world":           {"canonicaljson"},
		"simulation":      {"record", "canonicaljson"},
		"toolchain":       {"canonicaljson", "hostexec", "hostfs"},
		"upgrade":         {"qualification", "toolchain", "deterministicio", "compatibility", "canonicaljson", "hostexec", "hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/toolchain/version"
	}
	// Only the maintenance engines read adapters and compatibility packs.
	// The public compatibility facade also serializes its report as JSON.
	if owner == "upgrade" && (importedOwner == "compatibility" || importedOwner == "deterministicio" || importedOwner == "canonicaljson") {
		return importing == modulePath+"/upgrade/pinimpact" || importing == modulePath+"/upgrade/adapterregen" ||
			(importing == modulePath+"/upgrade" && importedOwner == "canonicaljson")
	}
	return true
}

func moduleMayImport(importing, imported string) bool {
	if !strings.HasPrefix(imported, modulePath+"/") {
		return true
	}
	forbidden := map[string][]string{
		modulePath + "/artifact": {modulePath + "/runner"},
		modulePath + "/record":   {modulePath + "/runner"},
		modulePath + "/runner/internal/campaign": {
			modulePath + "/runner/internal/execution", modulePath + "/runner/internal/corpus", modulePath + "/runner/internal/minimizer",
		},
		modulePath + "/runner/internal/execution": {
			modulePath + "/runner/internal/campaign", modulePath + "/runner/internal/corpus", modulePath + "/runner/internal/exploration",
		},
		modulePath + "/runner/internal/corpus": {
			modulePath + "/runner/internal/campaign", modulePath + "/runner/internal/execution", modulePath + "/runner/internal/exploration",
		},
	}
	for module, denied := range forbidden {
		if importing == module || strings.HasPrefix(importing, module+"/") {
			for _, prefix := range denied {
				if imported == prefix || strings.HasPrefix(imported, prefix+"/") {
					return false
				}
			}
		}
	}
	return true
}
