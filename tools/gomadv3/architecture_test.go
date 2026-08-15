package gomadv3_test

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

const modulePath = "go.temporal.io/server/tools/gomadv3"

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
		}
	}
	for _, owner := range []string{"cli", "runner", "qualification", "target", "evidence", "choice", "deterministicio", "world", "toolchain", "hostexec", "hostfs"} {
		if !owners[owner] {
			t.Errorf("architectural owner %s has no package", owner)
		}
	}
}

func TestOwnerMayImportCompatibilityPackCommand(t *testing.T) {
	packdev := modulePath + "/target/packdev"
	if !ownerMayImport("toolchain", "target", modulePath+"/toolchain/cmd/gomadtool", packdev) {
		t.Fatal("gomadtool must be able to import the compatibility-pack development kit")
	}
	for _, importing := range []string{
		modulePath + "/toolchain",
		modulePath + "/toolchain/version",
		modulePath + "/toolchain/cmd/other",
	} {
		if ownerMayImport("toolchain", "target", importing, packdev) {
			t.Fatalf("toolchain package %s may import the compatibility-pack development kit", importing)
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
		"./evidence/...", "./choice/...", "./deterministicio/...", "./world/...",
		"./toolchain", "./toolchain/version", "./toolchain/cmd/...", "./toolchain/internal/...", "./internal/...",
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
	case relative == "runner" || strings.HasPrefix(relative, "runner/"):
		return "runner"
	case relative == "qualification" || strings.HasPrefix(relative, "qualification/"):
		return "qualification"
	case relative == "target" || strings.HasPrefix(relative, "target/"):
		return "target"
	case relative == "evidence" || strings.HasPrefix(relative, "evidence/"):
		return "evidence"
	case relative == "choice" || strings.HasPrefix(relative, "choice/"):
		return "choice"
	case relative == "deterministicio" || strings.HasPrefix(relative, "deterministicio/"):
		return "deterministicio"
	case relative == "world" || strings.HasPrefix(relative, "world/"):
		return "world"
	case relative == "toolchain" || strings.HasPrefix(relative, "toolchain/"):
		return "toolchain"
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
	if owner == "toolchain" && importedOwner == "target" {
		return importing == modulePath+"/toolchain/cmd/gomadtool" && imported == modulePath+"/target/packdev"
	}
	allowed := map[string][]string{
		"cli":             {"runner", "qualification", "target", "evidence", "deterministicio", "toolchain"},
		"runner":          {"target", "evidence", "choice", "deterministicio", "world", "hostexec", "hostfs"},
		"qualification":   {"runner", "target", "evidence", "choice", "deterministicio", "hostexec", "hostfs"},
		"target":          {"evidence", "toolchain", "hostexec", "hostfs"},
		"evidence":        {"hostfs"},
		"deterministicio": {"target", "toolchain", "hostfs"},
		"toolchain":       {"qualification", "hostexec", "hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/toolchain/version"
	}
	return true
}
