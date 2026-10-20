package archcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type metadata struct {
	ImportPath, Dir, Export string
	Standard                bool
	GoFiles, CgoFiles       []string
	ImportMap               map[string]string
	Module                  *struct {
		Path, Dir, Version string
		Replace            *struct{ Path, Version string }
		Main               bool
	}
	Error      *struct{ Err string }
	DepsErrors []struct{ Err string }
}

type sourcePackage struct {
	types *types.Package
	info  *types.Info
	files []*ast.File
}

type program struct {
	root, module string
	fset         *token.FileSet
	metadata     map[string]metadata
	packages     map[string]*sourcePackage
	functions    map[string]string
	pure         func(string) bool
	importer     types.Importer
	goCommand    string
	platform     Platform
}

func environment(platform Platform) []string {
	var result []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "GOROOT", "GOOS", "GOARCH", "GOFLAGS", "GOWORK", "GOENV", "GOTOOLCHAIN", "GOEXPERIMENT", "CGO_ENABLED", "GOMADSEED", "GOMAD_CHILD_SEED":
			continue
		}
		result = append(result, entry)
	}
	return append(result, "GOOS="+platform.OS, "GOARCH="+platform.Arch, "GOFLAGS=", "GOWORK=off", "GOENV=off", "GOTOOLCHAIN=local", "GOEXPERIMENT=", "CGO_ENABLED=0")
}

func runGo(root, command string, platform Platform, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir, cmd.Env = root, environment(platform)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return out.Bytes(), nil
}

func list(root, command string, platform Platform, export bool, patterns []string) ([]metadata, error) {
	args := []string{"list", "-e", "-json", "-mod=readonly", "-tags=" + SelectionTags()}
	if export {
		args = append(args, "-export", "-deps")
	}
	data, err := runGo(root, command, platform, append(args, patterns...)...)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var result []metadata
	for {
		var pkg metadata
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if pkg.Error != nil {
			return nil, fmt.Errorf("package %s: %s", pkg.ImportPath, pkg.Error.Err)
		}
		if len(pkg.DepsErrors) > 0 {
			return nil, fmt.Errorf("package %s: %s", pkg.ImportPath, pkg.DepsErrors[0].Err)
		}
		result = append(result, pkg)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("go list selected no packages")
	}
	return result, nil
}

func load(root, goCommand string, platform Platform, pure func(string) bool) (*program, error) {
	root, err := canonicalRoot(root)
	if err != nil {
		return nil, err
	}
	version, err := runGo(root, goCommand, platform, "version")
	if err != nil {
		return nil, err
	}
	if err := checkVersion(string(version)); err != nil {
		return nil, err
	}
	packages, err := list(root, goCommand, platform, false, []string{"./..."})
	if err != nil {
		return nil, err
	}
	p := &program{root: root, fset: token.NewFileSet(), metadata: map[string]metadata{}, packages: map[string]*sourcePackage{}, functions: map[string]string{}, pure: pure, goCommand: goCommand, platform: platform}
	var selected []string
	for _, pkg := range packages {
		if pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		if pkg.Module.Main {
			p.module = pkg.Module.Path
		}
		p.metadata[pkg.ImportPath] = pkg
		for _, name := range append(slices.Clone(pkg.GoFiles), pkg.CgoFiles...) {
			if pure(p.relative(filepath.Join(pkg.Dir, name))) {
				selected = append(selected, pkg.ImportPath)
				break
			}
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("pure set matches no file on %s/%s", platform.OS, platform.Arch)
	}
	if err := p.exports(selected); err != nil {
		return nil, err
	}
	p.importer = importer.ForCompiler(p.fset, "gc", func(path string) (io.ReadCloser, error) {
		pkg, ok := p.metadata[path]
		if !ok || pkg.Export == "" {
			return nil, fmt.Errorf("package %s has no export data", path)
		}
		return os.Open(pkg.Export)
	})
	for _, path := range selected {
		if err := p.loadSource(path); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *program) loadSource(path string) error {
	meta := p.metadata[path]
	if len(meta.CgoFiles) > 0 {
		return fmt.Errorf("cannot type-check pure source in cgo package %s", path)
	}
	pkg := &sourcePackage{info: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}, Instances: map[*ast.Ident]types.Instance{}}}
	for _, name := range meta.GoFiles {
		file, err := parser.ParseFile(p.fset, filepath.Join(meta.Dir, name), nil, parser.ParseComments)
		if err != nil {
			return err
		}
		pkg.files = append(pkg.files, file)
	}
	config := types.Config{Importer: mappedImporter{p.importer, meta.ImportMap}, GoVersion: reviewedGoMinor, Sizes: types.SizesFor("gc", p.platform.Arch)}
	checked, err := config.Check(path, p.fset, pkg.files, pkg.info)
	if err != nil {
		return fmt.Errorf("type-check %s: %w", path, err)
	}
	pkg.types = checked
	p.packages[path] = pkg
	for ident, obj := range pkg.info.Defs {
		if fn, ok := obj.(*types.Func); ok {
			p.functions[functionID(fn)] = p.relative(p.fset.Position(ident.Pos()).Filename)
		}
	}
	return nil
}

func (p *program) exports(paths []string) error {
	packages, err := list(p.root, p.goCommand, p.platform, true, paths)
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		p.metadata[pkg.ImportPath] = pkg
	}
	return nil
}

func (p *program) relative(path string) string {
	rel, err := filepath.Rel(p.root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

type mappedImporter struct {
	types.Importer
	imports map[string]string
}

func (i mappedImporter) Import(path string) (*types.Package, error) {
	if mapped := i.imports[path]; mapped != "" {
		path = mapped
	}
	return i.Importer.Import(path)
}

func functionID(fn *types.Func) string {
	fn = fn.Origin()
	return fn.Pkg().Path() + "." + symbol(fn)
}

func symbol(fn *types.Func) string {
	if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
		typ := types.Unalias(recv.Type())
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = types.Unalias(ptr.Elem())
		}
		if named, ok := typ.(*types.Named); ok {
			return named.Obj().Name() + "." + fn.Name()
		}
	}
	return fn.Name()
}
