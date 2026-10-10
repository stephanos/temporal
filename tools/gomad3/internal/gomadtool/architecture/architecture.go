package architecture

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
)

type Finding struct {
	Category string
	Platform string
	Path     string
	Detail   string
}

type Platform struct{ OS, Arch string }

type Package struct {
	ImportPath                                                                    string
	Dir                                                                           string
	Imports                                                                       []string
	ImportMap                                                                     map[string]string
	GoFiles, CgoFiles, CompiledGoFiles, IgnoredGoFiles, TestGoFiles, XTestGoFiles []string
	Standard                                                                      bool
	Error                                                                         *struct{ Err string }
	DepsErrors                                                                    []struct{ Err string }
}

type Inventory struct {
	Packages         []Package
	Sources, Modules []string
}

func Discover(root, goCommand string, platform Platform) (Inventory, []Finding, error) {
	module, err := Module(root)
	if err != nil {
		return Inventory{}, nil, err
	}
	packages, err := List(root, goCommand, platform, false)
	if err != nil {
		return Inventory{}, nil, err
	}
	inventory := Inventory{}
	var findings []Finding
	add := func(category, path, detail string) {
		findings = append(findings, Finding{category, platform.OS + "/" + platform.Arch, path, detail})
	}
	sourceMatches := map[string]bool{}
	moduleMatches := map[string]bool{}
	listed := map[string]Package{}
	for _, pkg := range packages {
		relative, err := filepath.Rel(root, pkg.Dir)
		if err != nil {
			return inventory, findings, err
		}
		relative = filepath.ToSlash(relative)
		if Within(relative, "toolchain/runtime/overlay") {
			continue
		}
		if pkg.ImportPath != module && !Within(pkg.ImportPath, module) {
			continue
		}
		inventory.Packages = append(inventory.Packages, pkg)
		listed[relative] = pkg
		if pkg.Error != nil {
			add("package-error", relative, pkg.Error.Err)
		}
		for _, issue := range pkg.DepsErrors {
			add("package-error", relative, issue.Err)
		}
		if pkg.ImportPath == module && len(pkg.GoFiles)+len(pkg.CgoFiles) != 0 {
			add("ownerless", relative, "module root owns only the test harness")
		} else if Owner(module, pkg.ImportPath) == "" {
			add("ownerless", relative, pkg.ImportPath)
		}
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == ".toolchain" || relative == ".bin" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if relative == ".toolchain" || relative == ".bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				add("uncovered-source", relative, "source inventory cannot follow a directory symlink")
				return nil
			}
		}
		if entry.Type()&os.ModeSymlink != 0 && (strings.HasSuffix(relative, ".go") || filepath.Base(relative) == "go.mod") {
			add("uncovered-source", relative, "source symlink is not inventoried")
			return nil
		}
		if filepath.Base(relative) == "go.mod" && relative != "go.mod" {
			inventory.Modules = append(inventory.Modules, relative)
			if expectedModules[relative] {
				moduleMatches[relative] = true
			} else {
				add("unclassified-module", relative, "nested module needs an explicit classification")
			}
		}
		if !strings.HasSuffix(relative, ".go") {
			return nil
		}
		inventory.Sources = append(inventory.Sources, relative)
		for _, excluded := range sourceExclusions {
			if Within(relative, excluded) {
				sourceMatches[excluded] = true
				return nil
			}
		}
		directory := filepath.ToSlash(filepath.Dir(relative))
		pkg, found := listed[directory]
		if !found {
			add("uncovered-source", relative, "Go source was omitted by package discovery")
			return nil
		}
		name := filepath.Base(relative)
		if !contains(pkg.GoFiles, name) && !contains(pkg.CgoFiles, name) && !contains(pkg.IgnoredGoFiles, name) && !contains(pkg.TestGoFiles, name) && !contains(pkg.XTestGoFiles, name) {
			add("uncovered-source", relative, "Go source is absent from build metadata")
		}
		return nil
	})
	if err != nil {
		return inventory, findings, err
	}
	for _, excluded := range sourceExclusions {
		if !sourceMatches[excluded] {
			add("stale-exclusion", excluded, "required excluded source is absent")
		}
	}
	for expected := range expectedModules {
		if !moduleMatches[expected] {
			add("stale-exclusion", expected, "required nested module is absent")
			continue
		}
		directory := filepath.ToSlash(filepath.Dir(expected))
		matched := false
		for _, source := range inventory.Sources {
			if !Within(source, directory) {
				continue
			}
			belongs := true
			for _, nested := range inventory.Modules {
				nestedDirectory := filepath.ToSlash(filepath.Dir(nested))
				if nested != expected && Within(nestedDirectory, directory) && Within(source, nestedDirectory) {
					belongs = false
					break
				}
			}
			if belongs {
				matched = true
				break
			}
		}
		if !matched {
			add("stale-exclusion", expected, "required nested module has no source of its own")
		}
	}
	sort.Strings(inventory.Sources)
	findings = append(findings, PackageEdges(module, inventory.Packages)...)
	sort.Strings(inventory.Modules)
	sort.Slice(inventory.Packages, func(i, j int) bool { return inventory.Packages[i].ImportPath < inventory.Packages[j].ImportPath })
	return inventory, findings, nil
}

var hostSourcePackages = []string{"toolchain/runtime/testdata/vfdpointer", "toolchain/runtime/testdata/vfdnative"}
var sourceExclusions = []string{"toolchain/runtime/overlay", "cmd/gomad/testdata", "deterministicio/testdata", "internal/compatibilitypack/testdata", "internal/gomadtool/conformance/testdata", "testdata", "qualification/corpus"}
var expectedModules = map[string]bool{
	"deterministicio/testdata/cactusstatsd/go.mod": true, "deterministicio/testdata/hashicorpmetrics/go.mod": true,
	"deterministicio/testdata/memberlist/go.mod": true, "deterministicio/testdata/pebble/go.mod": true,
	"deterministicio/testdata/sentry/go.mod": true, "deterministicio/testdata/sockaddr/go.mod": true,
	"deterministicio/testdata/sprig/go.mod": true, "deterministicio/testdata/validator/go.mod": true,
	"internal/compatibilitypack/testdata/v041/go.mod": true, "internal/compatibilitypack/testdata/xsys/go.mod": true, "internal/gomadtool/conformance/testdata/go.mod": true,
	"internal/gomadtool/conformance/testdata/libc_adapter/go.mod": true, "internal/gomadtool/conformance/testdata/sqlite_adapter/go.mod": true,
	"qualification/corpus/go.mod": true,
}

func Within(path, root string) bool { return path == root || strings.HasPrefix(path, root+"/") }
func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func Module(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", err
	}
	if file.Module == nil {
		return "", fmt.Errorf("go.mod has no module directive")
	}
	return file.Module.Mod.Path, nil
}

func Environment(platform Platform) []string {
	var env []string
	for _, value := range os.Environ() {
		key := strings.SplitN(value, "=", 2)[0]
		switch key {
		case "GOOS", "GOARCH", "GOWORK", "GOTOOLCHAIN", "GOFLAGS", "GOROOT", "GOMADSEED", "GOMAD3_CHILD_SEED":
			continue
		}
		env = append(env, value)
	}
	return append(env, "GOOS="+platform.OS, "GOARCH="+platform.Arch, "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=")
}

func List(root, goCommand string, platform Platform, dependencies bool) ([]Package, error) {
	args := []string{"list", "-e", "-mod=readonly", "-json", "-tags", "test_dep"}
	if dependencies {
		args = append(args, "-deps", "-compiled")
	}
	args = append(args, "./...")
	for _, directory := range hostSourcePackages {
		info, err := os.Lstat(filepath.Join(root, directory))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			args = append(args, "./"+directory)
		}
	}
	command := exec.Command(goCommand, args...)
	command.Dir, command.Env = root, Environment(platform)
	var output, stderr bytes.Buffer
	command.Stdout, command.Stderr = &output, &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, &stderr)
	}
	decoder := json.NewDecoder(&output)
	var packages []Package
	for {
		var pkg Package
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func Owner(module, importPath string) string {
	if importPath == module {
		return "test-harness"
	}
	if !Within(importPath, module) {
		return ""
	}
	relative := strings.TrimPrefix(importPath, module+"/")
	if relative == "cmd/gomad" || Within(relative, "cmd/gomad/internal/cli") {
		return "cli"
	}
	if Within(relative, "cmd/gomadtool") || Within(relative, "internal/gomadtool") {
		return "developer"
	}
	for path, owner := range map[string]string{"internal/compatibilitypack": "compatibility", "internal/canonicaljson": "canonicaljson", "internal/hostexec": "hostexec", "hostexec": "hostexec", "internal/hostfs": "hostfs", "hostfs": "hostfs", "internal/preparation": "preparation", "internal/sourceinventory": "sourceinventory"} {
		if Within(relative, path) {
			return owner
		}
	}
	for _, owner := range []string{"upgrade", "runner", "qualification", "target", "record", "artifact", "choice", "deterministicio", "world", "simulation", "toolchain"} {
		if Within(relative, owner) {
			return owner
		}
	}
	return ""
}
