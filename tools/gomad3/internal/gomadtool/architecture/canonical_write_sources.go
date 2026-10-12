package architecture

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

const canonicalSummaryPath = "go.temporal.io/server/tools/gomad3/internal/canonicaljson"

var canonicalSummaryFiles = []string{"canonical.go", "implementation.go", "streaming.go"}

var canonicalSummaryDependencies = []string{"bufio", "bytes", "crypto/sha256", "embed", "encoding", "encoding/base64", "encoding/json", "errors", "fmt", "io", "reflect", "sort", "strconv", "strings", "sync", "unicode", "unicode/utf8"}

type canonicalInitializer struct {
	variables  []*types.Var
	expression ast.Expr
}

type canonicalPackageSourceSnapshot struct {
	native       *jsonPackageSourceSnapshot
	initializers []canonicalInitializer
	wholeSource  string
	startupPin   string
	astDigest    string
	definitions  map[*ast.Ident]types.Object
	uses         map[*ast.Ident]types.Object
	selections   map[*ast.SelectorExpr]*types.Selection
	typeValues   map[ast.Expr]types.TypeAndValue
}

func canonicalSourcePackage(path string) bool {
	return path == canonicalSummaryPath || slices.Contains(canonicalSummaryDependencies, path) || path == canonicalSHA256Backend || path == "crypto/internal/boring"
}

func canonicalCandidatePackage(pkg *SourcePackage) bool {
	if pkg.Metadata.Standard {
		return false
	}
	for _, file := range pkg.Files {
		constructor := false
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if fn := calledObject(pkg.Info, call.Fun); fn != nil && (functionID(fn) == "io.MultiWriter" || functionID(fn) == "crypto/sha256.New") {
					constructor = true
				}
			}
			return !constructor
		})
		if constructor {
			return true
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			object, ok := pkg.Info.Defs[fn.Name].(*types.Func)
			if ok && (object.Name() == "MarshalJSON" || object.Name() == "MarshalText" || object.Name() == "IsZero") {
				if receiver := object.Type().(*types.Signature).Recv(); receiver != nil && canonicalInteger(receiver.Type()) {
					return true
				}
			}
		}
		for _, imported := range file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err == nil && path == canonicalSummaryPath {
				return true
			}
		}
	}
	return false
}

func bindCanonicalPackageSource(pkg *SourcePackage, digests map[string]string) (*canonicalPackageSourceSnapshot, error) {
	snapshot := &canonicalPackageSourceSnapshot{native: bindJSONPackageSource(pkg, digests), startupPin: startupSources[pkg.Metadata.ImportPath]}
	if !pkg.Metadata.Standard || pkg.Metadata.ImportPath == "bufio" || pkg.Metadata.ImportPath == "bytes" || pkg.Metadata.ImportPath == "strconv" || pkg.Metadata.ImportPath == "io" || pkg.Metadata.ImportPath == "errors" || pkg.Metadata.ImportPath == "crypto/sha256" || pkg.Metadata.ImportPath == canonicalSHA256Backend || pkg.Metadata.ImportPath == "crypto/internal/boring" {
		var err error
		snapshot.astDigest, err = canonicalASTDigest(pkg.Files)
		if err != nil {
			return nil, err
		}
		snapshot.definitions = maps.Clone(pkg.Info.Defs)
		snapshot.uses = maps.Clone(pkg.Info.Uses)
		snapshot.selections = maps.Clone(pkg.Info.Selections)
		snapshot.typeValues = maps.Clone(pkg.Info.Types)
	}
	for _, initializer := range pkg.Info.InitOrder {
		snapshot.initializers = append(snapshot.initializers, canonicalInitializer{slices.Clone(initializer.Lhs), initializer.Rhs})
	}
	if pkg.Metadata.Standard {
		var err error
		snapshot.wholeSource, err = compressionWholeGoDigest(pkg.Metadata)
		if err != nil {
			return nil, err
		}
	}
	return snapshot, nil
}

func canonicalASTDigest(files []*ast.File) (string, error) {
	var buffer bytes.Buffer
	for _, file := range files {
		if err := printer.Fprint(&buffer, token.NewFileSet(), file); err != nil {
			return "", err
		}
		buffer.WriteByte(0)
	}
	return fmt.Sprintf("%x", sha256.Sum256(buffer.Bytes())), nil
}

func canonicalImplementationDigest(digests map[string]string) string {
	data := []byte("gomad3-canonicaljson-implementation/v1\n")
	for _, name := range canonicalSummaryFiles {
		data = fmt.Appendf(data, "%s:%s\n", name, digests[name])
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func (a *effectAnalysis) canonicalParsedBinding(path string) bool {
	snapshot := a.program.canonicalSources[path]
	if snapshot == nil {
		return false
	}
	native, pkg := snapshot.native, a.program.Packages[path]
	if pkg == nil || pkg != native.pkg || pkg.Types != native.types || pkg.Info != native.info ||
		!slices.Equal(pkg.Files, native.files) || !slices.Equal(pkg.Types.Imports(), native.imports) ||
		len(pkg.Info.InitOrder) != len(snapshot.initializers) {
		return false
	}
	if snapshot.astDigest != "" {
		fingerprint, err := canonicalASTDigest(pkg.Files)
		if err != nil || fingerprint != snapshot.astDigest || !maps.Equal(pkg.Info.Defs, snapshot.definitions) ||
			!maps.Equal(pkg.Info.Uses, snapshot.uses) || !maps.Equal(pkg.Info.Selections, snapshot.selections) || !maps.Equal(pkg.Info.Types, snapshot.typeValues) {
			return false
		}
	}
	for index, initializer := range pkg.Info.InitOrder {
		bound := snapshot.initializers[index]
		if !slices.Equal(initializer.Lhs, bound.variables) || initializer.Rhs != bound.expression {
			return false
		}
	}
	for _, current := range []Package{a.program.Metadata[path], pkg.Metadata} {
		if current.ImportPath != path || current.ImportPath != native.metadata.ImportPath || current.Dir != native.metadata.Dir ||
			current.Standard != native.metadata.Standard || current.Error != nil || len(current.DepsErrors) != 0 ||
			!slices.Equal(current.GoFiles, native.metadata.GoFiles) || !slices.Equal(current.CgoFiles, native.metadata.CgoFiles) ||
			!slices.Equal(current.CompiledGoFiles, native.metadata.CompiledGoFiles) || !slices.Equal(current.Imports, native.metadata.Imports) ||
			!maps.Equal(current.ImportMap, native.metadata.ImportMap) || len(current.CgoFiles) != 0 {
			return false
		}
	}
	return true
}

func (a *effectAnalysis) canonicalSourceBinding(path string) bool {
	if !a.canonicalParsedBinding(path) {
		return false
	}
	snapshot := a.program.canonicalSources[path]
	native := snapshot.native
	if path == canonicalSummaryPath {
		if native.metadata.Standard || !compressionSameNames(native.metadata.GoFiles, canonicalSummaryFiles) ||
			len(native.digests) != len(canonicalSummaryFiles) || canonicalImplementationDigest(native.digests) != canonicaljson.ImplementationSHA256() ||
			!compressionSameNames(native.metadata.Imports, canonicalSummaryDependencies) {
			return false
		}
	} else if !native.metadata.Standard || snapshot.startupPin == "" || snapshot.startupPin != startupSources[path] || snapshot.wholeSource != snapshot.startupPin {
		return false
	}
	if len(native.digests) != len(native.metadata.GoFiles) || !compressionCompiledSelection(native.metadata) {
		return false
	}
	for _, name := range native.metadata.GoFiles {
		if filepath.Base(name) != name || native.digests[name] == "" {
			return false
		}
	}
	for _, dependency := range native.imports {
		if dependency == types.Unsafe {
			continue
		}
		imported := a.program.Packages[dependency.Path()]
		if imported == nil || imported.Types != dependency || !imported.Metadata.Standard || !a.program.Metadata[dependency.Path()].Standard {
			return false
		}
	}
	for _, file := range native.files {
		for _, dependency := range file.Imports {
			name, err := strconv.Unquote(dependency.Path.Value)
			resolved := native.metadata.ImportMap[name]
			if err != nil || resolved != "" && resolved != name {
				return false
			}
		}
	}
	return true
}

func (a *effectAnalysis) canonicalScalarSourceBinding(path string) bool {
	if !a.canonicalParsedBinding(path) {
		return false
	}
	source := a.program.canonicalSources[path].native
	if source.metadata.Standard || len(source.metadata.CgoFiles) != 0 || len(source.digests) != len(source.metadata.GoFiles) {
		return false
	}
	for name, digest := range source.digests {
		if filepath.Base(name) != name {
			return false
		}
		data, err := os.ReadFile(filepath.Join(source.metadata.Dir, name))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			return false
		}
	}
	return true
}

func (a *effectAnalysis) checkCanonicalSummarySource(fn *types.Func, pos token.Pos) bool {
	reject := func(detail string) bool {
		a.report("unresolved-effect", pos, "canonical write source witness: "+detail)
		return false
	}
	if fn == nil || fn.Pkg() == nil {
		return reject("missing function identity")
	}
	path := canonicalSummaryPath
	pkg := a.program.Packages[path]
	snapshot := a.program.canonicalSources[path]
	decl := a.program.Functions[functionID(fn)]
	if pkg == nil || snapshot == nil || fn.Origin().Pkg() != pkg.Types ||
		fn.Origin() != pkg.Types.Scope().Lookup("WriteCanonicalJSON") || decl == nil || decl.pkg != pkg || decl.object.Origin() != fn.Origin() ||
		snapshot.native.functions[functionID(fn)].object != fn.Origin() || snapshot.native.functions[functionID(fn)].declaration != decl.declaration ||
		pkg.Info.Defs[decl.declaration.Name] != fn.Origin() {
		return reject("native parsed declaration identity changed")
	}
	paths := append([]string{path}, canonicalSummaryDependencies...)
	for _, dependency := range paths {
		if !a.canonicalSourceBinding(dependency) {
			return reject("package binding changed: " + dependency)
		}
	}
	if a.checkedCanonical != nil && maps.Equal(a.checkedCanonical, a.program.canonicalSources) {
		return true
	}
	for _, dependency := range paths {
		source := a.program.canonicalSources[dependency]
		for name, digest := range source.native.digests {
			data, err := os.ReadFile(filepath.Join(source.native.metadata.Dir, name))
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
				return reject("parsed source bytes changed: " + dependency + "/" + name)
			}
		}
		if dependency != path {
			digest, err := compressionWholeGoDigest(source.native.metadata)
			if err != nil || digest != source.wholeSource {
				return reject("native source inventory changed: " + dependency)
			}
		}
	}
	json := a.program.Packages["encoding/json"]
	if json == nil || !a.checkJSONSummarySource(json.Types.Scope().Lookup("Marshal").(*types.Func), pos) {
		return reject("native JSON dependency authority changed")
	}
	a.checkedCanonical = maps.Clone(a.program.canonicalSources)
	return true
}
