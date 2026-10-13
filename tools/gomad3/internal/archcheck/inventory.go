package archcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
)

type HostPackage struct {
	ImportPath, Dir                                                       string
	Imports, GoFiles, CgoFiles, IgnoredGoFiles, TestGoFiles, XTestGoFiles []string
	Error                                                                 *struct{ Err string }
	DepsErrors                                                            []struct{ Err string }
}

type Inventory struct {
	Packages         []HostPackage
	Sources, Modules []string
}

func moduleInfo(root string) (string, string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	for {
		data, err := os.ReadFile(filepath.Join(root, "go.mod"))
		if err == nil {
			module := modfile.ModulePath(data)
			if module == "" {
				return "", "", fmt.Errorf("go.mod has no module directive")
			}
			return module, root, nil
		}
		if !os.IsNotExist(err) || filepath.Dir(root) == root {
			return "", "", err
		}
		root = filepath.Dir(root)
	}
}

func Environment(platform Platform) []string {
	var env []string
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if !slices.Contains([]string{"GOOS", "GOARCH", "GOWORK", "GOTOOLCHAIN", "GOFLAGS", "GOROOT", "GOMADSEED", "GOMAD3_CHILD_SEED"}, key) {
			env = append(env, value)
		}
	}
	return append(env, "GOOS="+platform.OS, "GOARCH="+platform.Arch, "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=")
}

func List(root, goCommand string, platform Platform, dependencies bool) ([]HostPackage, error) {
	args := []string{"list", "-e", "-mod=readonly", "-json", "-tags", "test_dep"}
	if dependencies {
		args = append(args, "-deps", "-compiled")
	}
	args = append(args, "./...")
	for _, directory := range hostSourcePackages {
		info, err := os.Lstat(filepath.Join(root, directory))
		switch {
		case os.IsNotExist(err):
		case err != nil:
			return nil, err
		case info.IsDir():
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
	var packages []HostPackage
	decoder := json.NewDecoder(&output)
	for {
		var pkg HostPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func Discover(root, goCommand string, platform Platform) (Inventory, []Finding, error) {
	return discover(root, goCommand, platform, true)
}

func discover(root, goCommand string, platform Platform, legacy bool) (Inventory, []Finding, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Inventory{}, nil, err
	}
	module, moduleRoot, err := moduleInfo(root)
	if err != nil {
		return Inventory{}, nil, err
	}
	scope, err := filepath.Rel(moduleRoot, root)
	if err != nil {
		return Inventory{}, nil, err
	}
	packages, err := List(root, goCommand, platform, false)
	if err != nil {
		return Inventory{}, nil, err
	}
	if len(packages) == 0 {
		return Inventory{}, nil, fmt.Errorf("go list selected no packages")
	}
	inventory := Inventory{}
	var findings []Finding
	add := func(category, path, detail string) { findings = append(findings, Finding{category, path, detail}) }
	listed, matched := map[string]HostPackage{}, map[string]bool{}
	for _, pkg := range packages {
		relative, err := filepath.Rel(root, pkg.Dir)
		if err != nil {
			return inventory, findings, err
		}
		relative = filepath.ToSlash(relative)
		moduleRelative := filepath.ToSlash(filepath.Join(scope, relative))
		if Within(moduleRelative, "internal/backends/native/toolchain/runtime/overlay") || Within(moduleRelative, "internal/backends/wasm/wasmhost/target") || !Within(pkg.ImportPath, module) {
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
		if pkg.ImportPath == module {
			add("ownerless", relative, "module root holds no Go files")
		}
	}
	if slices.ContainsFunc(findings, func(f Finding) bool { return f.Category == "package-error" }) {
		return inventory, ordered(findings), nil
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		relative := filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))
		moduleRelative := filepath.ToSlash(filepath.Join(scope, relative))
		if entry.IsDir() && (slices.Contains([]string{".toolchain", ".bin"}, relative) || moduleRelative == "internal/backends/wasm/wasmhost/target") {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			info, err := os.Stat(path)
			if err != nil || info.IsDir() || strings.HasSuffix(relative, ".go") || filepath.Base(relative) == "go.mod" {
				add("uncovered-source", relative, "source symlink is not inventoried")
				return nil
			}
		}
		if filepath.Base(relative) == "go.mod" && relative != "go.mod" {
			inventory.Modules = append(inventory.Modules, relative)
			if !expectedModules[moduleRelative] {
				add("unclassified-module", relative, "nested module needs an explicit classification")
			}
		}
		if !strings.HasSuffix(relative, ".go") {
			return nil
		}
		inventory.Sources = append(inventory.Sources, relative)
		for _, excluded := range sourceExclusions {
			if Within(moduleRelative, excluded) {
				matched[excluded] = true
				return nil
			}
		}
		pkg, found := listed[filepath.ToSlash(filepath.Dir(relative))]
		if !found {
			add("uncovered-source", relative, "Go source was omitted by package discovery")
			return nil
		}
		name := filepath.Base(relative)
		if !slices.Contains(append(append(append(append(slices.Clone(pkg.GoFiles), pkg.CgoFiles...), pkg.IgnoredGoFiles...), pkg.TestGoFiles...), pkg.XTestGoFiles...), name) {
			add("uncovered-source", relative, "Go source is absent from build metadata")
		}
		return nil
	})
	if err != nil {
		return inventory, findings, err
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); legacy && err == nil {
		for _, excluded := range sourceExclusions {
			if !matched[excluded] {
				add("stale-exclusion", excluded, "required excluded source is absent")
			}
		}
		for expected := range expectedModules {
			ownSource := slices.Contains(inventory.Modules, expected) && slices.ContainsFunc(inventory.Sources, func(source string) bool {
				return Within(source, filepath.Dir(expected)) && !slices.ContainsFunc(inventory.Modules, func(nested string) bool {
					return nested != expected && Within(filepath.Dir(nested), filepath.Dir(expected)) && Within(source, filepath.Dir(nested))
				})
			})
			if !ownSource {
				add("stale-exclusion", expected, "required nested module or its own source is absent")
			}
		}
	}
	slices.Sort(inventory.Sources)
	slices.Sort(inventory.Modules)
	slices.SortFunc(inventory.Packages, func(a, b HostPackage) int { return strings.Compare(a.ImportPath, b.ImportPath) })
	if legacy {
		findings = append(findings, PackageEdges(module, inventory.Packages)...)
	}
	return inventory, ordered(findings), nil
}

func Within(path, root string) bool { return within(path, root) }

var sourceExclusions = []string{
	"internal/model/cluster/testdata/simulation_exploration", "internal/model/cluster/testdata/simulation_replay",
	"internal/backends/native/toolchain/runtime/overlay", "internal/backends/native/transcript/testdata", "internal/backends/native/conformance/testdata", "internal/backends/native/qualification/testdata/tagged",
	"cmd/gomad/testdata", "internal/backends/native/compatibilitypack/testdata", "internal/evidence/qualification/corpus",
	"internal/backends/wasm/testdata", "internal/backends/wasm/wasi/testdata", "internal/backends/wasm/toolchain/testdata", "internal/backends/wasm/cluster/testdata", "internal/backends/wasm/qualification/testdata", "internal/backends/wasm/command/testdata",
}

var hostSourcePackages = []string{"internal/backends/native/toolchain/runtime/testdata/vfdpointer", "internal/backends/native/toolchain/runtime/testdata/vfdnative"}

var expectedModules = map[string]bool{
	"internal/backends/native/transcript/testdata/cactusstatsd/go.mod": true, "internal/backends/native/transcript/testdata/hashicorpmetrics/go.mod": true, "internal/backends/native/transcript/testdata/memberlist/go.mod": true, "internal/backends/native/transcript/testdata/pebble/go.mod": true,
	"internal/backends/native/transcript/testdata/sentry/go.mod": true, "internal/backends/native/transcript/testdata/sockaddr/go.mod": true, "internal/backends/native/transcript/testdata/sprig/go.mod": true, "internal/backends/native/transcript/testdata/validator/go.mod": true,
	"internal/backends/native/compatibilitypack/testdata/v041/go.mod": true, "internal/backends/native/compatibilitypack/testdata/xsys/go.mod": true, "internal/backends/native/conformance/testdata/go.mod": true, "internal/backends/native/conformance/testdata/libc_adapter/go.mod": true, "internal/backends/native/conformance/testdata/sqlite_adapter/go.mod": true,
	"internal/evidence/qualification/corpus/go.mod": true, "internal/backends/wasm/qualification/testdata/fixture/go.mod": true,
}
