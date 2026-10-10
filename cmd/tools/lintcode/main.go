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

const gomadModule = "tools/gomad3"
const mixedbrainModule = "tests/mixedbrain"

func within(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+"/")
}

type ownership struct {
	fixtures     []string
	modules      map[string]bool
	overlays     map[string]bool
	hostPackages map[string]bool
}

type source struct {
	path, module, disposition string
}

func main() {
	base := flag.String("base", "main", "Lint comparison revision")
	module := flag.String("module", "", "Check all host packages in tools/gomad3 or tests/mixedbrain; otherwise check changed packages")
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
		selected, changedDirs, err = policy.selectChanges(ctx, root, base, modules)
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
		command := exec.CommandContext(ctx, "make", "lint-code", "GOLANGCI_LINT_BASE_REV="+base, "ALL_TEST_TAGS="+scope.tags, "LINT_CODE_DIR="+filepath.Join(root, scope.module), "LINT_CODE_TARGETS="+strings.Join(scope.packages, " "))
		command.Dir = root
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("lint module %s: %w", scope.module, err)
		}
	}
	if len(scopes) == 0 {
		fmt.Println("No changed Go packages to lint.")
	}
	return nil
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

func (p ownership) selectChanges(ctx context.Context, root, base string, modules []string) (selected, changedDirs map[string]bool, err error) {
	selected = map[string]bool{}
	changedDirs = map[string]bool{}
	changed, err := gitPaths(ctx, root, "diff", "--no-renames", "--name-only", "-z", base, "--", "*.go")
	if err != nil {
		return nil, nil, err
	}
	untracked, err := gitPaths(ctx, root, "ls-files", "-z", "--others", "--exclude-standard", "--", "*.go")
	if err != nil {
		return nil, nil, err
	}
	for _, path := range append(changed, untracked...) {
		entry, err := p.classify(path, modules)
		if err != nil {
			return nil, nil, err
		}
		if entry.disposition != "" {
			fmt.Printf("%s: %s\n", entry.disposition, path)
			continue
		}
		selected[entry.module] = true
		changedDirs[filepath.ToSlash(filepath.Dir(path))] = true
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
		moduleScopes, err := moduleScopes(ctx, root, owner, tags, owned)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, moduleScopes...)
	}
	return scopes, nil
}

func moduleScopes(ctx context.Context, root, owner, tags string, sources []source) ([]lintScope, error) {
	var ordinary, integration []source
	for _, entry := range sources {
		if owner == "." && filepath.ToSlash(filepath.Dir(entry.path)) == "tools/gomad3integration" {
			integration = append(integration, entry)
		} else {
			ordinary = append(ordinary, entry)
		}
	}
	var scopes []lintScope
	scope, err := sourceScope(ctx, root, owner, tags, ordinary)
	if err != nil {
		return nil, err
	}
	if scope != nil {
		scopes = append(scopes, *scope)
	}
	if len(integration) != 0 {
		scope, err := sourceScope(ctx, root, owner, tags+",gomad3_integration", integration)
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, *scope)
	}
	return scopes, nil
}

func sourceScope(ctx context.Context, root, owner, tags string, sources []source) (*lintScope, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	packages, err := coveredPackages(ctx, root, owner, tags, sources)
	if err != nil {
		return nil, err
	}
	return &lintScope{module: owner, packages: packages, tags: tags}, nil
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
	policy := ownership{modules: map[string]bool{}, overlays: map[string]bool{}, hostPackages: map[string]bool{}}
	path := filepath.Join(root, gomadModule, "internal/gomadtool/architecture/architecture.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return policy, fmt.Errorf("read Gomad source classifications: %w", err)
	}
	for _, name := range []string{"sourceExclusions", "expectedModules", "hostSourcePackages"} {
		entries, err := classificationPaths(file, name)
		if err != nil {
			return policy, err
		}
		switch name {
		case "sourceExclusions":
			policy.fixtures = entries
		case "expectedModules":
			for _, entry := range entries {
				policy.modules[gomadModule+"/"+entry] = true
			}
		case "hostSourcePackages":
			if err := policy.registerHostPackages(root, entries); err != nil {
				return policy, err
			}
		default:
			return policy, fmt.Errorf("unknown Gomad source classification %q", name)
		}
	}
	if len(policy.fixtures) == 0 || len(policy.modules) == 0 || len(policy.hostPackages) == 0 {
		return policy, errors.New("gomad source classifications need literal sourceExclusions, expectedModules and hostSourcePackages")
	}
	data, err := os.ReadFile(filepath.Join(root, gomadModule, "toolchain/version/version.json"))
	if err != nil {
		return policy, err
	}
	var descriptor struct {
		OverlayAllowlist []string `json:"overlay_allowlist"`
	}
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return policy, err
	}
	if len(descriptor.OverlayAllowlist) == 0 {
		return policy, errors.New("gomad overlay allowlist is empty")
	}
	for _, entry := range descriptor.OverlayAllowlist {
		if !within(entry, "src") || filepath.ToSlash(filepath.Clean(entry)) != entry || policy.overlays[entry] {
			return policy, fmt.Errorf("invalid Gomad overlay allowlist entry %q", entry)
		}
		policy.overlays[entry] = true
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

func (p ownership) registerHostPackages(root string, entries []string) error {
	for _, entry := range entries {
		base := filepath.Base(entry)
		if filepath.ToSlash(filepath.Dir(entry)) != "toolchain/runtime/testdata" || strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") || strings.ContainsAny(base, "*?[]") || p.hostPackages[entry] {
			return fmt.Errorf("invalid Gomad host source package %q", entry)
		}
		if err := regularHostDirectory(root, gomadModule+"/"+entry); err != nil {
			return err
		}
		p.hostPackages[entry] = true
	}
	return nil
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
		if directory != "." && directory != gomadModule && directory != mixedbrainModule && !p.modules[path] {
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
	case within(path, "tools/gomad3sim/testdata/simulation_exploration"):
		entry.disposition = "simulation fixture"
	case within(path, "tools/gomad3integration/testdata/tagged"):
		entry.disposition = "integration fixture"
	case within(path, gomadModule+"/toolchain/runtime/overlay"):
		relative := strings.TrimPrefix(path, gomadModule+"/toolchain/runtime/overlay/")
		if !p.overlays[relative] {
			return entry, fmt.Errorf("unclassified runtime overlay source %s", path)
		}
		entry.disposition = "runtime overlay"
	case within(path, gomadModule):
		relative := strings.TrimPrefix(path, gomadModule+"/")
		for _, fixture := range p.fixtures {
			if fixture != "toolchain/runtime/overlay" && within(relative, fixture) {
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

func (p ownership) validateGomadHostSource(path, relative string) error {
	if !p.hostPackages[filepath.ToSlash(filepath.Dir(relative))] {
		for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(relative)), "/") {
			if part != "." && (part == "testdata" || strings.HasPrefix(part, ".") || strings.HasPrefix(part, "_")) {
				return fmt.Errorf("uncovered Gomad host source %s", path)
			}
		}
		if within(relative, "toolchain/runtime") {
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

func regularHostDirectory(root, path string) error {
	for candidate := filepath.Join(root, path); candidate != root; candidate = filepath.Dir(candidate) {
		info, err := os.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("uncovered source symlink %s", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("gomad host package directory is not a directory: %s", path)
		}
	}
	return nil
}
