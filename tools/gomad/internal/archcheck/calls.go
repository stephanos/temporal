package archcheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type analysis struct {
	p           *program
	findings    []Finding
	interfaces  map[string]bool
	methodNames map[string]map[string]bool
	streams     map[*sourcePackage]*streamIndex
	err         error
}

func analyze(p *program, now time.Time, logger *log.Logger) []Finding {
	a := &analysis{p: p, interfaces: map[string]bool{}, streams: map[*sourcePackage]*streamIndex{}}
	for _, path := range sortedKeys(p.packages) {
		pkg := p.packages[path]
		var reviewList []string
		for _, file := range pkg.files {
			if !p.pure(p.relative(p.fset.Position(file.Pos()).Filename)) {
				continue
			}
			marks := reviewMarks(p.fset, file)
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.GoStmt:
					a.report("host-effect", node.Pos(), "goroutine start")
				case *ast.CallExpr:
					category, detail := a.call(pkg, node)
					if a.err != nil {
						return false
					}
					if category == "dynamic-call" && useMark(p.fset, node.Pos(), marks) {
						return true
					}
					if category != "" {
						a.report(category, node.Pos(), detail)
					}
				case *ast.RangeStmt:
					if _, ok := pkg.info.TypeOf(node.X).Underlying().(*types.Signature); ok {
						category, detail := a.callable(pkg, node.X)
						if a.err != nil {
							return false
						}
						if category == "dynamic-call" && useMark(p.fset, node.Pos(), marks) {
							return true
						}
						if category != "" {
							a.report(category, node.Pos(), detail)
						}
					}
				}
				return true
			})
			if a.err != nil {
				sortFindings(a.findings)
				return a.findings
			}
			for _, mark := range marks {
				reviewList = append(reviewList, fmt.Sprintf("%s %s", p.fset.Position(mark.pos), mark.text))
				if !mark.valid || !mark.used {
					a.report("stale-review", mark.pos, "review mark requires a dynamic call, date and reason")
				}
				if mark.valid && now.Sub(mark.date) > 365*24*time.Hour {
					logger.Printf("warning: aged dynamic review %s", p.fset.Position(mark.pos))
				}
			}
		}
		slices.Sort(reviewList)
		logger.Printf("%s: %d dynamic review marks", path, len(reviewList))
		for _, mark := range reviewList {
			logger.Print(mark)
		}
	}
	sortFindings(a.findings)
	return a.findings
}

func (a *analysis) call(pkg *sourcePackage, call *ast.CallExpr) (string, string) {
	if value, ok := pkg.info.Types[call.Fun]; ok && value.IsType() {
		return "", ""
	}
	category, detail := a.callable(pkg, call.Fun)
	if category != "" {
		return category, detail
	}
	fn := staticFunction(pkg, call.Fun)
	if fn == nil || fn.Pkg() == nil || !a.p.metadata[fn.Pkg().Path()].Standard {
		return "", ""
	}
	if fn.Pkg().Path() == "time" && fn.Name() == "ParseInLocation" {
		if len(call.Args) != 3 || !exactUTC(pkg, call.Args[2]) {
			return "host-effect", "time.ParseInLocation requires time.UTC"
		}
	}
	if fn.Pkg().Path() == "time" && fn.Name() == "Date" && (len(call.Args) != 8 || !exactUTC(pkg, call.Args[7])) {
		return "host-effect", "time.Date requires time.UTC"
	}
	if fn.Pkg().Path() == "errors" && fn.Name() == "Join" {
		return "", ""
	}
	dynamicDetail := ""
	signature := fn.Type().(*types.Signature)
	for index, arg := range call.Args {
		if signature.Params().Len() == 0 {
			break
		}
		parameter := signature.Params().At(min(index, signature.Params().Len()-1)).Type()
		if signature.Variadic() && index >= signature.Params().Len()-1 {
			parameter = parameter.(*types.Slice).Elem()
		}
		if _, generic := parameter.(*types.TypeParam); generic {
			continue
		}
		if streamWrapper(pkg.info.TypeOf(arg)) {
			if !a.localStream(pkg, arg, map[*types.Var]bool{}) {
				dynamicDetail = "stream wrapper argument not proven constructed in the pure set"
			}
			continue
		}
		switch formal := parameter.Underlying().(type) {
		case *types.Signature:
			if _, literal := unparen(arg).(*ast.FuncLit); literal {
				continue
			}
			if category, detail := a.callable(pkg, arg); category == "host-effect" {
				return category, detail
			} else if category == "dynamic-call" {
				dynamicDetail = detail
			}
		case *types.Interface:
			valueCallbacks := fn.Pkg().Path() == "encoding/json" || fn.Pkg().Path() == "fmt" && !formatTypesOnly(call, fn.Name())
			if (formal.NumMethods() != 0 || valueCallbacks) && (!a.callbackValue(pkg.info.TypeOf(arg), map[types.Type]bool{}) || !a.pureMethods(pkg.info.TypeOf(arg), formal)) {
				if valueCallbacks && containsTime(pkg.info.TypeOf(arg), map[types.Type]bool{}) {
					return "host-effect", "time.Time formatting or JSON callbacks may load local timezone state"
				}
				dynamicDetail = "stdlib callback through " + types.TypeString(pkg.info.TypeOf(arg), nil)
			}
		}
	}
	if dynamicDetail != "" {
		return "dynamic-call", dynamicDetail
	}
	return "", ""
}

func (a *analysis) pureMethods(typ types.Type, formal *types.Interface) bool {
	if typ == nil {
		return false
	}
	if _, ok := typ.Underlying().(*types.Interface); ok {
		return a.pureInterface(typ)
	}
	methods := types.NewMethodSet(typ)
	for i := 0; i < formal.NumMethods(); i++ {
		required := formal.Method(i)
		selected := methods.Lookup(required.Pkg(), required.Name())
		if selected == nil {
			return false
		}
		fn := selected.Obj().(*types.Func)
		if fn.Pkg() == nil {
			return false
		}
		path := fn.Pkg().Path()
		if a.p.metadata[path].Standard {
			if !allowed(path, symbol(fn)) {
				return false
			}
		} else {
			source := a.p.functions[functionID(fn)]
			if source == "" || !a.p.pure(source) {
				return false
			}
		}
	}
	return true
}

func containsTime(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	switch typ := types.Unalias(typ).(type) {
	case *types.Named:
		if obj := typ.Obj(); obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "Time" {
			return true
		}
		return containsTime(typ.Underlying(), seen)
	case *types.Pointer:
		return containsTime(typ.Elem(), seen)
	case *types.Slice:
		return containsTime(typ.Elem(), seen)
	case *types.Array:
		return containsTime(typ.Elem(), seen)
	case *types.Map:
		return containsTime(typ.Key(), seen) || containsTime(typ.Elem(), seen)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if containsTime(typ.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func formatTypesOnly(call *ast.CallExpr, name string) bool {
	index := 0
	switch name {
	case "Sprintf", "Errorf":
	case "Fprintf", "Appendf":
		index = 1
	default:
		return false
	}
	if len(call.Args) <= index {
		return false
	}
	literal, ok := call.Args[index].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}
	format, err := strconv.Unquote(literal.Value)
	if err != nil || !strings.Contains(format, "%") {
		return false
	}
	for _, verb := range []string{"%%", "%T", "%p"} {
		format = strings.ReplaceAll(format, verb, "")
	}
	return !strings.Contains(format, "%")
}

func (a *analysis) callbackValue(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return true
	}
	seen[typ] = true
	switch typ := types.Unalias(typ).(type) {
	case *types.Named:
		if _, ok := typ.Underlying().(*types.Interface); ok {
			return a.pureInterface(typ)
		}
		obj := typ.Obj()
		if obj.Pkg() != nil {
			path := obj.Pkg().Path()
			standard := a.p.metadata[path].Standard
			if !standard && !a.p.pure(a.p.relative(a.p.fset.Position(obj.Pos()).Filename)) {
				return false
			}
			methods := types.NewMethodSet(types.NewPointer(typ))
			for i := 0; i < methods.Len(); i++ {
				fn := methods.At(i).Obj().(*types.Func)
				if !slices.Contains(callbackMethods, fn.Name()) {
					continue
				}
				if standard && !allowed(path, symbol(fn)) || !standard && !a.p.pure(a.p.functions[functionID(fn)]) {
					return false
				}
			}
		}
		return a.callbackValue(typ.Underlying(), seen)
	case *types.Interface:
		return a.pureInterface(typ)
	case *types.Pointer:
		return a.callbackValue(typ.Elem(), seen)
	case *types.Slice:
		return a.callbackValue(typ.Elem(), seen)
	case *types.Array:
		return a.callbackValue(typ.Elem(), seen)
	case *types.Map:
		return a.callbackValue(typ.Key(), seen) && a.callbackValue(typ.Elem(), seen)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if !a.callbackValue(typ.Field(i).Type(), seen) {
				return false
			}
		}
	}
	return true
}

// callbackMethods are the methods admitted json, fmt, errors, io and gzip
// functions invoke on caller values, including the json v2 and omitzero hooks.
var callbackMethods = []string{"MarshalJSON", "UnmarshalJSON", "MarshalJSONTo", "UnmarshalJSONFrom", "MarshalText", "UnmarshalText", "AppendText", "IsZero", "String", "GoString", "Format", "Error", "Unwrap", "Is", "As", "Read", "ReadByte", "Write", "WriteString", "ReadFrom", "WriteTo"}

func (a *analysis) callable(pkg *sourcePackage, expr ast.Expr) (string, string) {
	expr = unparen(expr)
	if _, ok := expr.(*ast.FuncLit); ok {
		return "", ""
	}
	if ident, ok := expr.(*ast.Ident); ok {
		if _, ok := pkg.info.Uses[ident].(*types.Builtin); ok {
			return "", ""
		}
	}
	selector, _ := expr.(*ast.SelectorExpr)
	if selector != nil {
		if selection := pkg.info.Selections[selector]; selection != nil {
			receiver := selection.Recv()
			// A method promoted from an embedded interface is still an interface call.
			if fn, ok := selection.Obj().(*types.Func); ok && !types.IsInterface(receiver) && fn.Signature().Recv() != nil {
				receiver = fn.Signature().Recv().Type()
			}
			if types.IsInterface(receiver) {
				if a.pureInterface(receiver) {
					return "", ""
				}
				return "dynamic-call", "interface method " + selection.Obj().Name()
			}
		}
	}
	fn := staticFunction(pkg, expr)
	if fn == nil {
		return "dynamic-call", "function or method value requires review"
	}
	path, name := fn.Pkg().Path(), symbol(fn)
	if a.p.metadata[path].Standard {
		if !allowed(path, name) {
			return "host-effect", path + "." + name + " is absent from the pure stdlib allowlist"
		}
		if recv := fn.Signature().Recv(); recv != nil && streamWrapper(recv.Type()) && (selector == nil || !a.localStream(pkg, selector.X, map[*types.Var]bool{})) {
			return "dynamic-call", "stream wrapper " + name + " on a stream not proven constructed in the pure set"
		}
		return "", ""
	}
	if path == "golang.org/x/mod/module" {
		if err := reviewedModule(a.p.metadata[path], name); err != nil {
			return "host-effect", err.Error()
		}
		return "", ""
	}
	if source := a.p.functions[functionID(fn)]; source != "" && a.p.pure(source) {
		return "", ""
	}
	return "host-effect", path + "." + name + " is declared outside the pure set"
}

func staticFunction(pkg *sourcePackage, expr ast.Expr) *types.Func {
	switch expr := unparen(expr).(type) {
	case *ast.IndexExpr:
		return staticFunction(pkg, expr.X)
	case *ast.IndexListExpr:
		return staticFunction(pkg, expr.X)
	case *ast.Ident:
		fn, _ := pkg.info.Uses[expr].(*types.Func)
		return fn
	case *ast.SelectorExpr:
		if selection := pkg.info.Selections[expr]; selection != nil {
			fn, _ := selection.Obj().(*types.Func)
			return fn
		}
		fn, _ := pkg.info.Uses[expr.Sel].(*types.Func)
		return fn
	}
	return nil
}

func unparen(expr ast.Expr) ast.Expr {
	for {
		if paren, ok := expr.(*ast.ParenExpr); ok {
			expr = paren.X
		} else {
			return expr
		}
	}
}

func exactUTC(pkg *sourcePackage, expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	imported, ok := pkg.info.Uses[ident].(*types.PkgName)
	return ok && imported.Imported().Path() == "time" && selector.Sel.Name == "UTC"
}

func (a *analysis) pureInterface(typ types.Type) bool {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	obj := named.Obj()
	if !within(obj.Pkg().Path(), a.p.module) || !a.p.pure(a.p.relative(a.p.fset.Position(obj.Pos()).Filename)) {
		return false
	}
	key := types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() })
	if result, found := a.interfaces[key]; found {
		return result
	}
	iface := typ.Underlying().(*types.Interface)
	if !a.moduleBound(iface) {
		a.interfaces[key] = false
		return false
	}
	foreign, err := a.exportedInterface(named)
	if err != nil {
		a.fail(err)
		return false
	}
	if !a.otherImplementations(foreign) {
		a.interfaces[key] = false
		return false
	}
	found := false
	for _, pkg := range a.p.packages {
		for _, name := range pkg.types.Scope().Names() {
			obj, ok := pkg.types.Scope().Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			matches, pure := a.pureImplementation(obj, iface, foreign)
			if matches && !pure {
				a.interfaces[key] = false
				return false
			}
			found = found || matches
		}
	}
	a.interfaces[key] = found
	return found
}

// moduleBound reports interfaces only module types can implement: one method
// is unexported or names a module type, which no standard-library or
// third-party type can declare. Others may hold values from outside the module.
func (a *analysis) moduleBound(iface *types.Interface) bool {
	for i := 0; i < iface.NumMethods(); i++ {
		if method := iface.Method(i); !method.Exported() || a.namesModuleType(method.Type(), map[types.Type]bool{}) {
			return true
		}
	}
	return false
}

func (a *analysis) namesModuleType(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	switch typ := types.Unalias(typ).(type) {
	case *types.Named:
		if typ.Obj().Pkg() != nil && within(typ.Obj().Pkg().Path(), a.p.module) {
			return true
		}
		for i := 0; i < typ.TypeArgs().Len(); i++ {
			if a.namesModuleType(typ.TypeArgs().At(i), seen) {
				return true
			}
		}
	case *types.Pointer:
		return a.namesModuleType(typ.Elem(), seen)
	case *types.Slice:
		return a.namesModuleType(typ.Elem(), seen)
	case *types.Array:
		return a.namesModuleType(typ.Elem(), seen)
	case *types.Chan:
		return a.namesModuleType(typ.Elem(), seen)
	case *types.Map:
		return a.namesModuleType(typ.Key(), seen) || a.namesModuleType(typ.Elem(), seen)
	case *types.Signature:
		for _, tuple := range []*types.Tuple{typ.Params(), typ.Results()} {
			for i := 0; i < tuple.Len(); i++ {
				if a.namesModuleType(tuple.At(i).Type(), seen) {
					return true
				}
			}
		}
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if a.namesModuleType(typ.Field(i).Type(), seen) {
				return true
			}
		}
	case *types.Interface:
		for i := 0; i < typ.NumMethods(); i++ {
			if a.namesModuleType(typ.Method(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func (a *analysis) pureImplementation(obj *types.TypeName, iface, foreign *types.Interface) (bool, bool) {
	candidate, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok {
		return false, false
	}
	if _, ok := candidate.Underlying().(*types.Interface); ok {
		return false, false
	}
	for _, receiver := range []types.Type{candidate, types.NewPointer(candidate)} {
		if !types.Implements(receiver, iface) && !types.Implements(receiver, foreign) {
			continue
		}
		if !a.p.pure(a.p.relative(a.p.fset.Position(obj.Pos()).Filename)) {
			return true, false
		}
		set := types.NewMethodSet(receiver)
		for i := 0; i < set.Len(); i++ {
			fn, _ := set.At(i).Obj().(*types.Func)
			if source := a.p.functions[functionID(fn)]; source == "" || !a.p.pure(source) {
				return true, false
			}
		}
		return true, true
	}
	return false, false
}

func (a *analysis) exportedInterface(named *types.Named) (*types.Interface, error) {
	obj := named.Obj()
	pkg, err := a.p.importer.Import(obj.Pkg().Path())
	if err != nil {
		return nil, err
	}
	exported := pkg.Scope().Lookup(obj.Name())
	if exported == nil {
		iface := named.Underlying().(*types.Interface)
		methods := make([]*types.Func, iface.NumMethods())
		for i := range methods {
			method := iface.Method(i)
			methods[i] = types.NewFunc(method.Pos(), pkg, method.Name(), a.exportType(method.Type()).(*types.Signature))
		}
		return types.NewInterfaceType(methods, nil).Complete(), a.err
	}
	typ := exported.Type()
	if args := named.TypeArgs(); args.Len() > 0 {
		arguments := make([]types.Type, args.Len())
		for i := range arguments {
			arguments[i] = args.At(i)
		}
		typ, err = types.Instantiate(nil, typ, arguments, true)
		if err != nil {
			return nil, err
		}
	}
	return typ.Underlying().(*types.Interface), nil
}

func (a *analysis) exportType(typ types.Type) types.Type {
	unalias := types.Unalias(typ)
	switch typ := unalias.(type) {
	case *types.Named:
		if typ.Obj().Pkg() == nil {
			return typ
		}
		pkg, err := a.p.importer.Import(typ.Obj().Pkg().Path())
		if err != nil {
			a.fail(err)
			return typ
		}
		obj := pkg.Scope().Lookup(typ.Obj().Name())
		if obj == nil {
			return typ
		}
		if args := typ.TypeArgs(); args.Len() > 0 {
			values := make([]types.Type, args.Len())
			for i := range values {
				values[i] = a.exportType(args.At(i))
			}
			instantiated, err := types.Instantiate(nil, obj.Type(), values, true)
			if err != nil {
				a.fail(err)
				return typ
			}
			return instantiated
		}
		return obj.Type()
	case *types.Pointer:
		return types.NewPointer(a.exportType(typ.Elem()))
	case *types.Slice:
		return types.NewSlice(a.exportType(typ.Elem()))
	case *types.Array:
		return types.NewArray(a.exportType(typ.Elem()), typ.Len())
	case *types.Map:
		return types.NewMap(a.exportType(typ.Key()), a.exportType(typ.Elem()))
	case *types.Chan:
		return types.NewChan(typ.Dir(), a.exportType(typ.Elem()))
	case *types.Signature:
		return types.NewSignatureType(nil, nil, nil, a.exportTuple(typ.Params()), a.exportTuple(typ.Results()), typ.Variadic())
	}
	return typ
}

func (a *analysis) exportTuple(tuple *types.Tuple) *types.Tuple {
	if tuple == nil {
		return nil
	}
	values := make([]*types.Var, tuple.Len())
	for i := range values {
		value := tuple.At(i)
		values[i] = types.NewVar(value.Pos(), value.Pkg(), value.Name(), a.exportType(value.Type()))
	}
	return types.NewTuple(values...)
}

func (a *analysis) otherImplementations(iface *types.Interface) bool {
	if a.methodNames == nil {
		if err := a.indexMethodNames(); err != nil {
			a.fail(err)
			return false
		}
	}
	for _, path := range sortedKeys(a.methodNames) {
		names := a.methodNames[path]
		if !possibleMethods(names, "", iface) {
			continue
		}
		if a.p.metadata[path].Export == "" {
			if err := a.p.exports([]string{path}); err != nil {
				a.fail(err)
				return false
			}
		}
		pkg, err := a.p.importer.Import(path)
		if err != nil {
			a.fail(err)
			return false
		}
		if hasImplementation(pkg, iface) || unexportedImplementation(pkg, names, iface) {
			return false
		}
	}
	for _, pkg := range a.p.packages {
		for _, value := range pkg.info.Types {
			typ := types.Unalias(value.Type)
			if typ == nil {
				continue
			}
			if ptr, ok := typ.(*types.Pointer); ok {
				typ = types.Unalias(ptr.Elem())
			}
			if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != nil && !within(named.Obj().Pkg().Path(), a.p.module) {
				if _, ok := named.Underlying().(*types.Interface); !ok && (types.Implements(named, iface) || types.Implements(types.NewPointer(named), iface)) {
					return false
				}
			}
		}
	}
	return true
}

func (a *analysis) indexMethodNames() error {
	a.methodNames = map[string]map[string]bool{}
	for path, meta := range a.p.metadata {
		if !within(path, a.p.module) || a.p.packages[path] != nil {
			continue
		}
		names := map[string]bool{}
		for _, name := range meta.GoFiles {
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(meta.Dir, name), nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			indexDeclaredMethods(file, names)
		}
		a.methodNames[path] = names
	}
	return nil
}

func indexDeclaredMethods(file *ast.File, names map[string]bool) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if node.Recv != nil {
				names[node.Name.Name] = true
				names[receiverName(node.Recv.List[0].Type)+"\x00"+node.Name.Name] = true
			}
			return false
		case *ast.TypeSpec:
			if st, ok := node.Type.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						names["*"] = true
						names[node.Name.Name+"\x00*"] = true
					}
				}
			}
		}
		return true
	})
}

func receiverName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return receiverName(expr.X)
	case *ast.IndexExpr:
		return receiverName(expr.X)
	case *ast.IndexListExpr:
		return receiverName(expr.X)
	}
	return ""
}

func possibleMethods(names map[string]bool, prefix string, iface *types.Interface) bool {
	for i := 0; i < iface.NumMethods(); i++ {
		if !names[prefix+iface.Method(i).Name()] && !names[prefix+"*"] {
			return false
		}
	}
	return true
}

func hasImplementation(pkg *types.Package, iface *types.Interface) bool {
	for _, name := range pkg.Scope().Names() {
		obj, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		typ := types.Unalias(obj.Type())
		if _, ok := typ.Underlying().(*types.Interface); ok {
			continue
		}
		if types.Implements(typ, iface) || types.Implements(types.NewPointer(typ), iface) {
			return true
		}
	}
	return false
}

func unexportedImplementation(pkg *types.Package, names map[string]bool, iface *types.Interface) bool {
	for name := range names {
		receiver, _, ok := strings.Cut(name, "\x00")
		if ok && pkg.Scope().Lookup(receiver) == nil && possibleMethods(names, receiver+"\x00", iface) {
			return true
		}
	}
	return false
}

func (a *analysis) fail(err error) {
	a.err = err
	a.findings = append(a.findings, Finding{Category: "package-error", Path: a.p.root, Detail: err.Error()})
}

func (a *analysis) report(category string, pos token.Pos, detail string) {
	position := a.p.fset.Position(pos)
	position.Filename = a.p.relative(position.Filename)
	a.findings = append(a.findings, Finding{Category: category, Path: position.String(), Detail: detail})
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func sortFindings(findings []Finding) {
	slices.SortFunc(findings, func(a, b Finding) int {
		return strings.Compare(a.Category+"\x00"+a.Path+"\x00"+a.Detail, b.Category+"\x00"+b.Path+"\x00"+b.Detail)
	})
}
