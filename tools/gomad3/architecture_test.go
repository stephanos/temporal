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
	for _, owner := range []string{"cli", "developer", "runner", "qualification", "target", "record", "artifact", "choice", "deterministicio", "world", "simulation", "toolchain", "upgrade", "compatibility", "canonicaljson", "hostexec", "hostfs"} {
		if !owners[owner] {
			t.Errorf("architectural owner %s has no package", owner)
		}
	}
}

func TestCleanupRemovesSupersededFiles(t *testing.T) {
	for _, path := range []string{
		"build.sh",
		"clock_audit_test.sh",
		"compiler_test_exec.sh",
		"exec.sh",
		"regenerate-patch.sh",
		"test.sh",
		"runner/internal/execution/output.go",
		"runner/internal/execution/output_test.go",
		"internal/compatibilitypack/migration_baseline.json",
		"toolchain/cmd/gomadtool",
		"toolchain/internal/conformance",
		"toolchain/internal/generate",
		"toolchain/internal/validation",
	} {
		if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
			t.Errorf("superseded file %s still exists or cannot be checked: %v", path, err)
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

func TestCurrentVocabularyHasNoLegacyCampaignBoundary(t *testing.T) {
	for _, root := range []string{"cmd/gomad", "qualification", "runner"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
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
	for _, target := range []string{"clean:", "prune-cache:", "clean-qualifications:", "test-toolchain:", "test-host:", "validate-toolchain:", "validate-compatibility:"} {
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

func TestCanonicalJSONHasOnePrivateOwner(t *testing.T) {
	if _, err := os.Stat("internal/canonicaljson/canonical.go"); err != nil {
		t.Errorf("private canonical JSON module is missing: %v", err)
	}
	if _, err := os.Stat("evidence/canonical.go"); err == nil || !os.IsNotExist(err) {
		t.Errorf("evidence still owns canonical JSON or cannot be checked: %v", err)
	}
	worldCodec, err := os.ReadFile("world/codec.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(worldCodec), "func canonicalJSON(") {
		t.Error("World still owns a duplicate canonical JSON implementation")
	}
}

func TestRecordAndArtifactHaveSeparateOwners(t *testing.T) {
	for _, path := range []string{"record/record.go", "artifact/store.go", "artifact/publication.go"} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("deep module %s is missing: %v", path, err)
		}
	}
	if _, err := os.Stat("evidence"); err == nil || !os.IsNotExist(err) {
		t.Errorf("superseded evidence package still exists or cannot be checked: %v", err)
	}
	if _, err := os.Stat("runner/internal/campaign/artifact.go"); err == nil || !os.IsNotExist(err) {
		t.Errorf("campaign storage still owns artifact publication or cannot be checked: %v", err)
	}
}

func TestReadOnlyMountHasOneDeepOwner(t *testing.T) {
	if _, err := os.Stat("deterministicio/readonlymount/capture.go"); err != nil {
		t.Errorf("read-only mount module is missing: %v", err)
	}
	entries, err := filepath.Glob("deterministicio/readonly_*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("deterministic I/O root still owns read-only mount files: %v", entries)
	}
}

func TestDeveloperToolingIsNotOwnedByToolchain(t *testing.T) {
	for _, path := range []string{"cmd/gomadtool/main.go", "internal/gomadtool/generation/boundary", "internal/gomadtool/generation/protocol"} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("developer tooling owner %s is missing: %v", path, err)
		}
	}
	for _, path := range []string{"toolchain/cmd/gomadtool", "toolchain/internal/generate", "toolchain/internal/conformance", "toolchain/internal/validation"} {
		if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
			t.Errorf("toolchain still owns developer tooling %s or cannot be checked: %v", path, err)
		}
	}
}

func TestWorldProcessSessionHasOneDeepOwner(t *testing.T) {
	if _, err := os.Stat("world/process/session.go"); err != nil {
		t.Errorf("World process-session module is missing: %v", err)
	}
	for _, path := range []string{"world/host", "world/target", "world/internal/transport"} {
		if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
			t.Errorf("superseded World process-session owner %s still exists or cannot be checked: %v", path, err)
		}
	}
}

func TestCompatibilityPackHasOneOwner(t *testing.T) {
	for _, path := range []string{"internal/compatibilitypack/policy.go", "internal/compatibilitypack/authoring/generate.go"} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("compatibility-pack owner %s is missing: %v", path, err)
		}
	}
	for _, path := range []string{"target/internal/compatibility", "target/packdev"} {
		if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
			t.Errorf("target still owns compatibility-pack implementation %s or cannot be checked: %v", path, err)
		}
	}
}

func TestUpgradeOrchestrationIsAboveToolchain(t *testing.T) {
	if _, err := os.Stat("upgrade/upgrade.go"); err != nil {
		t.Errorf("upgrade orchestration module is missing: %v", err)
	}
	if _, err := os.Stat("toolchain/upgrade.go"); err == nil || !os.IsNotExist(err) {
		t.Errorf("toolchain still owns upgrade orchestration or cannot be checked: %v", err)
	}
}

func TestQualificationUseCasesHaveExplicitOwners(t *testing.T) {
	for _, path := range []string{
		"qualification/analysis/analysis.go",
		"qualification/comparison/comparison.go",
		"qualification/set/set.go",
		"qualification/workload/workload.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("qualification use-case owner %s is missing: %v", path, err)
		}
	}
	for _, path := range []string{"qualification/suite_legacy.go", "qualification/suite_previous.go"} {
		if _, err := os.Stat(path); err == nil || !os.IsNotExist(err) {
			t.Errorf("legacy qualification-set codec %s still exists or cannot be checked: %v", path, err)
		}
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

func TestConformanceRuntimeIsGroupedByBehavior(t *testing.T) {
	if _, err := os.Stat("internal/gomadtool/conformance/runtime.go"); err == nil || !os.IsNotExist(err) {
		t.Errorf("conformance runtime monolith still exists or cannot be checked: %v", err)
	}
	for _, path := range []string{
		"runtime_campaign.go", "runtime_clocks.go", "runtime_compatibility.go", "runtime_linking.go",
		"runtime_load.go", "runtime_repeatability.go", "runtime_scheduling.go",
	} {
		if _, err := os.Stat(filepath.Join("internal/gomadtool/conformance", path)); err != nil {
			t.Errorf("behavior-local conformance runtime file %s is missing: %v", path, err)
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
		"./simulation/...", "./upgrade/...",
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
	default:
		return ""
	}
}

func ownerMayImport(owner, importedOwner, importing, imported string) bool {
	if owner == importedOwner {
		return true
	}
	allowed := map[string][]string{
		"cli":             {"runner", "qualification", "target", "record", "artifact", "deterministicio", "toolchain", "canonicaljson"},
		"developer":       {"compatibility", "qualification", "simulation", "toolchain", "upgrade", "hostexec", "hostfs"},
		"runner":          {"target", "record", "artifact", "choice", "deterministicio", "world", "canonicaljson", "hostexec", "hostfs"},
		"qualification":   {"runner", "target", "record", "artifact", "choice", "deterministicio", "canonicaljson", "hostexec", "hostfs"},
		"target":          {"compatibility", "record", "toolchain", "canonicaljson", "hostexec", "hostfs"},
		"record":          {"canonicaljson"},
		"artifact":        {"choice", "deterministicio", "target", "record", "hostfs"},
		"compatibility":   {"target", "record", "canonicaljson", "hostfs"},
		"deterministicio": {"target", "record", "toolchain", "canonicaljson", "hostfs"},
		"world":           {"canonicaljson"},
		"simulation":      {"record", "canonicaljson"},
		"toolchain":       {"canonicaljson", "hostexec", "hostfs"},
		"upgrade":         {"qualification", "toolchain", "hostexec"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/toolchain/version"
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
