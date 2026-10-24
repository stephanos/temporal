package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const gomadModule = "tools/gomad"
const wasmBackend = gomadModule + "/internal/backends/wasm"
const mixedbrainModule = "tests/mixedbrain"

func within(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

type ownership struct {
	fixtures []string
	modules  map[string]bool
}

type source struct {
	path, module, disposition string
}

func main() {
	base := flag.String("base", "main", "Lint comparison revision")
	module := flag.String("module", "", "Check all host packages in tools/gomad or tests/mixedbrain; otherwise check changed packages")
	tags := flag.String("tags", "test_dep", "Go build tags")
	flag.Parse()
	if err := lint(context.Background(), *base, *module, *tags); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func lint(ctx context.Context, base, module, tags string) error {
	if module != "" && module != gomadModule && module != mixedbrainModule {
		return fmt.Errorf("unclassified lint module %q", module)
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := commandOutput(ctx, root, "git", "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return fmt.Errorf("GOLANGCI_LINT_BASE_REV=%s is not a known commit: %w", base, err)
	}
	if module == "" {
		data, err := commandOutput(ctx, root, "git", "merge-base", "HEAD", base)
		if err != nil {
			return err
		}
		base = strings.TrimSpace(string(data))
	}
	policy, err := loadOwnership(root)
	if err != nil {
		return err
	}
	paths, err := gitPaths(ctx, root, "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.go", "go.mod", "**/go.mod")
	if err != nil {
		return err
	}
	modules, err := policy.moduleDirectories(root, paths)
	if err != nil {
		return err
	}
	sources, err := policy.sources(root, paths, modules)
	if err != nil {
		return err
	}
	selected, changedDirs := map[string]bool{}, map[string]bool{}
	if module != "" {
		selected[module] = true
	} else {
		selected, changedDirs, err = policy.selectChanges(ctx, root, base, modules, sources)
		if err != nil {
			return err
		}
	}
	scopes, err := lintScopes(ctx, root, tags, selected, changedDirs, sources)
	if err != nil {
		return err
	}
	for _, scope := range scopes {
		fmt.Printf("Lint module %s: %d host packages\n", scope.module, len(scope.packages))
		if err := runLintScope(ctx, root, base, scope); err != nil {
			return fmt.Errorf("lint module %s: %w", scope.module, err)
		}
	}
	if len(scopes) == 0 {
		fmt.Println("No changed Go packages to lint.")
	}
	return nil
}

func runLintScope(ctx context.Context, root, base string, scope lintScope) (retErr error) {
	args := []string{"lint-code", "GOLANGCI_LINT_BASE_REV=" + base, "ALL_TEST_TAGS=" + scope.tags, "LINT_CODE_DIR=" + filepath.Join(root, scope.module), "LINT_CODE_TARGETS=" + strings.Join(scope.packages, " ")}
	if scope.module != "." {
		patch, err := lintPatch(ctx, root, base, scope.module)
		if err != nil {
			return err
		}
		file, err := os.CreateTemp("", "lintcode-*.patch")
		if err != nil {
			return err
		}
		defer func() {
			if err := os.Remove(file.Name()); err != nil && retErr == nil {
				retErr = err
			}
		}()
		_, writeErr := file.Write(patch)
		closeErr := file.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return err
		}
		args = append(args, "GOLANGCI_LINT_PATCH="+file.Name())
	}
	command := exec.CommandContext(ctx, "make", args...)
	command.Dir = root
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}

func (p ownership) sources(root string, paths, modules []string) ([]source, error) {
	var sources []source
	for _, path := range paths {
		if !strings.HasSuffix(path, ".go") {
			continue
		}
		if _, err := os.Lstat(filepath.Join(root, path)); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		entry, err := p.classify(path, modules)
		if err != nil {
			return nil, err
		}
		sources = append(sources, entry)
	}
	return sources, nil
}

func (p ownership) selectChanges(ctx context.Context, root, base string, modules []string, sources []source) (selected, changedDirs map[string]bool, err error) {
	selected = map[string]bool{}
	changedDirs = map[string]bool{}
	surviving := map[string]source{}
	for _, entry := range sources {
		if entry.disposition == "" {
			surviving[filepath.ToSlash(filepath.Dir(entry.path))] = entry
		}
	}
	changed, err := gitPaths(ctx, root, "diff", "--no-renames", "--name-only", "-z", base, "--", "*.go")
	if err != nil {
		return nil, nil, err
	}
	untracked, err := gitPaths(ctx, root, "ls-files", "-z", "--others", "--exclude-standard", "--", "*.go")
	if err != nil {
		return nil, nil, err
	}
	for _, path := range append(changed, untracked...) {
		directory := filepath.ToSlash(filepath.Dir(path))
		if _, err := os.Lstat(filepath.Join(root, path)); errors.Is(err, os.ErrNotExist) {
			if entry, exists := surviving[directory]; exists {
				selected[entry.module] = true
				changedDirs[directory] = true
			}
			continue
		} else if err != nil {
			return nil, nil, err
		}
		entry, err := p.classify(path, modules)
		if err != nil {
			return nil, nil, err
		}
		if entry.disposition != "" {
			fmt.Printf("%s: %s\n", entry.disposition, path)
			continue
		}
		selected[entry.module] = true
		changedDirs[directory] = true
	}
	return selected, changedDirs, nil
}

type lintScope struct {
	module   string
	packages []string
	tags     string
}

func lintScopes(ctx context.Context, root, tags string, selected, changedDirs map[string]bool, sources []source) ([]lintScope, error) {
	var scopes []lintScope
	for _, owner := range []string{".", gomadModule, mixedbrainModule} {
		if !selected[owner] {
			continue
		}
		var owned []source
		for _, entry := range sources {
			if entry.disposition == "" && entry.module == owner && (owner != "." || changedDirs[filepath.ToSlash(filepath.Dir(entry.path))]) {
				owned = append(owned, entry)
			}
		}
		if len(owned) == 0 {
			if owner != "." {
				return nil, fmt.Errorf("lint module %s has no ordinary host source", owner)
			}
			continue
		}
		packages, err := coveredPackages(ctx, root, owner, tags, owned)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, lintScope{module: owner, packages: packages, tags: tags})
	}
	return scopes, nil
}

func commandOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	var stderr bytes.Buffer
	command.Stderr = &stderr
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w: %s", name, args, err, &stderr)
	}
	return data, nil
}

func gitPaths(ctx context.Context, root string, args ...string) ([]string, error) {
	data, err := commandOutput(ctx, root, "git", args...)
	if err != nil {
		return nil, err
	}
	paths := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
	paths = slices.DeleteFunc(paths, func(path string) bool { return path == "" })
	slices.Sort(paths)
	return slices.Compact(paths), nil
}

func loadOwnership(root string) (ownership, error) {
	policy := ownership{modules: map[string]bool{}}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, gomadModule, "internal/archcheck/inventory.go"), nil, 0)
	if err != nil {
		return policy, fmt.Errorf("read Gomad source classifications: %w", err)
	}
	if policy.fixtures, err = classificationPaths(file, "sourceExclusions"); err != nil {
		return policy, err
	}
	entries, err := classificationPaths(file, "expectedModules")
	if err != nil {
		return policy, err
	}
	for _, entry := range entries {
		if policy.modules[gomadModule+"/"+entry] {
			return policy, fmt.Errorf("gomad module %q is classified twice", entry)
		}
		policy.modules[gomadModule+"/"+entry] = true
	}
	if len(policy.fixtures) == 0 || len(policy.modules) == 0 {
		return policy, errors.New("gomad source classifications need literal sourceExclusions and expectedModules")
	}
	return policy, nil
}

func classificationPaths(file *ast.File, name string) ([]string, error) {
	object := file.Scope.Lookup(name)
	if object == nil {
		return nil, fmt.Errorf("gomad source classification %s is absent", name)
	}
	values, ok := object.Decl.(*ast.ValueSpec)
	if !ok || len(values.Names) != 1 || len(values.Values) != 1 {
		return nil, fmt.Errorf("gomad source classification %s must be a literal", name)
	}
	return literalPaths(name, values.Values[0])
}

func literalPaths(name string, expression ast.Expr) ([]string, error) {
	list, ok := expression.(*ast.CompositeLit)
	if !ok {
		return nil, fmt.Errorf("gomad %s must remain a literal path classification", name)
	}
	var paths []string
	for _, expression := range list.Elts {
		if name == "expectedModules" {
			pair, ok := expression.(*ast.KeyValueExpr)
			if !ok {
				return nil, fmt.Errorf("gomad %s has a nonliteral entry", name)
			}
			value, ok := pair.Value.(*ast.Ident)
			if !ok || value.Name != "true" {
				return nil, fmt.Errorf("gomad %s has an inactive classification", name)
			}
			expression = pair.Key
		}
		literal, ok := expression.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return nil, fmt.Errorf("gomad %s has a nonliteral path", name)
		}
		path, err := strconv.Unquote(literal.Value)
		if err != nil {
			return nil, err
		}
		if path == "." || filepath.IsAbs(path) || filepath.ToSlash(filepath.Clean(path)) != path || strings.HasPrefix(path, "../") {
			return nil, fmt.Errorf("gomad %s has an invalid path %q", name, path)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func (p ownership) moduleDirectories(root string, paths []string) ([]string, error) {
	var modules []string
	for _, path := range paths {
		if filepath.Base(path) != "go.mod" || within(path, ".flow") {
			continue
		}
		info, err := os.Lstat(filepath.Join(root, path))
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("uncovered module symlink %s", path)
		}
		directory := filepath.ToSlash(filepath.Dir(path))
		if directory != "." && directory != gomadModule && directory != wasmBackend+"/internal/evidence/qualification/testdata/fixture" && directory != mixedbrainModule && !p.modules[path] {
			return nil, fmt.Errorf("unclassified module %s", path)
		}
		modules = append(modules, directory)
	}
	for _, module := range []string{".", gomadModule, mixedbrainModule} {
		if !slices.Contains(modules, module) {
			return nil, fmt.Errorf("required lint module is missing: %s", filepath.ToSlash(filepath.Join(module, "go.mod")))
		}
	}
	slices.SortFunc(modules, func(a, b string) int { return len(b) - len(a) })
	return modules, nil
}

func (p ownership) classify(path string, modules []string) (source, error) {
	entry := source{path: path}
	switch {
	case within(path, ".flow"):
		entry.disposition = "retained evidence"
	case within(path, "common/gomaddebug/testdata/determinism"), within(path, "common/gomaddebug/testdata/fixture"):
		entry.disposition = "Gomad debug qualification fixture"
	case within(path, "tests/gomadfunctional/testdata/softassert"):
		entry.disposition = "WASM soft assertion fixture"
	case within(path, "tools/gomad/internal/model/cluster/testdata/simulation_exploration"):
		entry.disposition = "simulation fixture"
	case within(path, wasmBackend+"/testdata/environment"):
		entry.disposition = "WASM environment fixture"
	case within(path, wasmBackend+"/toolchain/testdata/choice_divergence"):
		entry.disposition = "WASM helper-role divergence fixture"
	case within(path, wasmBackend+"/toolchain/testdata/choices"), within(path, wasmBackend+"/testdata/runtime_probes"):
		entry.disposition = "WASM runtime qualification fixture"
	case within(path, wasmBackend+"/testdata/frontend_namespace"), within(path, wasmBackend+"/testdata/sqlite_contention"):
		entry.disposition = "WASM Temporal diagnostic fixture"
	case within(path, wasmBackend+"/testdata/time_probes"), within(path, wasmBackend+"/testdata/unsupported_clock"), within(path, wasmBackend+"/testdata/network_probes"), within(path, wasmBackend+"/testdata/prototype_temporal"):
		entry.disposition = "WASM determinism prototype fixture"
	case within(path, wasmBackend+"/testdata/storage_sqlite"), within(path, wasmBackend+"/testdata/network_streams"), within(path, wasmBackend+"/testdata/temporal_storage"), within(path, wasmBackend+"/testdata/temporal_broker"), within(path, wasmBackend+"/wasi/testdata/runtime_cost"), within(path, wasmBackend+"/wasi/testdata/namespace_owner"), within(path, wasmBackend+"/cluster/testdata/service"), within(path, wasmBackend+"/command/testdata"):
		entry.disposition = "WASM productionization fixture"
	case within(path, wasmBackend+"/qualification/testdata/fixture"):
		entry.disposition = "WASM support and cost qualification fixture"
	case within(path, wasmBackend+"/testdata/io_failure_module"):
		entry.disposition = "WASM guest admission failure fixture"
	case within(path, wasmBackend+"/testdata/ringpop"):
		entry.disposition = "WASM ringpop scenario fixture"
	case within(path, gomadModule+"/internal/model/wasi/testdata/runtime_cost"), within(path, gomadModule+"/internal/model/wasi/testdata/namespace_owner"):
		entry.disposition = "WASM productionization fixture"
	case within(path, wasmBackend):
		relative := strings.TrimPrefix(path, wasmBackend+"/")
		if err := p.validateWASMHostSource(path, relative); err != nil {
			return entry, err
		}
	case within(path, gomadModule+"/internal/model/wasi"):
		relative := strings.TrimPrefix(path, gomadModule+"/")
		if err := p.validateGomadHostSource(path, relative); err != nil {
			return entry, err
		}
	case within(path, gomadModule):
		relative := strings.TrimPrefix(path, gomadModule+"/")
		for _, fixture := range p.fixtures {
			if within(relative, fixture) {
				entry.disposition = "Gomad qualification fixture"
				return entry, nil
			}
		}
		if err := p.validateGomadHostSource(path, relative); err != nil {
			return entry, err
		}
	default:
	}
	if entry.disposition != "" {
		return entry, nil
	}
	for _, module := range modules {
		if module == "." || within(path, module) {
			entry.module = module
			return entry, nil
		}
	}
	return entry, fmt.Errorf("source has no module owner: %s", path)
}

func (p ownership) validateWASMHostSource(path, relative string) error {
	for part := range strings.SplitSeq(filepath.ToSlash(filepath.Dir(relative)), "/") {
		if part != "." && (part == "testdata" || strings.HasPrefix(part, ".") || strings.HasPrefix(part, "_")) {
			return fmt.Errorf("uncovered WASM host source %s", path)
		}
	}
	return nil
}

func (p ownership) validateGomadHostSource(path, relative string) error {
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(relative)), "/") {
		if part != "." && (part == "testdata" || strings.HasPrefix(part, ".") || strings.HasPrefix(part, "_")) {
			return fmt.Errorf("uncovered Gomad host source %s", path)
		}
	}
	return nil
}

type listedPackage struct {
	Dir                                                          string
	GoFiles, CgoFiles, IgnoredGoFiles, TestGoFiles, XTestGoFiles []string
	Error                                                        *struct{ Err string }
	DepsErrors                                                   []struct{ Err string }
}

func coveredPackages(ctx context.Context, root, module, tags string, sources []source) ([]string, error) {
	var packages []string
	for _, entry := range sources {
		path := filepath.Join(root, entry.path)
		if err := regularSource(root, entry.path); err != nil {
			return nil, err
		}
		directory, err := filepath.Rel(filepath.Join(root, module), filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		for _, char := range directory {
			if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_./-", char) {
				return nil, fmt.Errorf("unsupported lint package directory for %s", entry.path)
			}
		}
		if directory != "." {
			directory = "./" + filepath.ToSlash(directory)
		}
		packages = append(packages, directory)
	}
	slices.Sort(packages)
	packages = slices.Compact(packages)
	args := append([]string{"list", "-e", "-mod=readonly", "-json", "-tags", tags}, packages...)
	data, err := commandOutput(ctx, filepath.Join(root, module), "go", args...)
	if err != nil {
		return nil, err
	}
	covered, err := packageSourceCoverage(data)
	if err != nil {
		return nil, err
	}
	for _, entry := range sources {
		if !covered[filepath.Join(root, entry.path)] {
			return nil, fmt.Errorf("uncovered host source %s: absent from Go package metadata", entry.path)
		}
	}
	return packages, nil
}

func packageSourceCoverage(data []byte) (map[string]bool, error) {
	covered := map[string]bool{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg listedPackage
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if pkg.Error != nil {
			return nil, fmt.Errorf("go package %s: %s", pkg.Dir, pkg.Error.Err)
		}
		if len(pkg.DepsErrors) != 0 {
			return nil, fmt.Errorf("go package %s: %s", pkg.Dir, pkg.DepsErrors[0].Err)
		}
		for _, name := range slices.Concat(pkg.GoFiles, pkg.CgoFiles, pkg.IgnoredGoFiles, pkg.TestGoFiles, pkg.XTestGoFiles) {
			covered[filepath.Join(pkg.Dir, name)] = true
		}
	}
	return covered, nil
}

func regularSource(root, path string) error {
	for candidate := filepath.Join(root, path); candidate != root; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("uncovered source symlink %s", path)
		}
	}
	return nil
}
