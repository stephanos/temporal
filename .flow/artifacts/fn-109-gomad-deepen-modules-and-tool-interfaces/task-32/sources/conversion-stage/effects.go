package architecture

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type abstractValue struct {
	types     []types.Type
	functions []*boundFunction
	fields    map[string]*abstractValue
	elements  *abstractValue
	utc       bool
	unknown   bool
	builtin   bool
	text      string
	truth     int
	origin    token.Pos
}
type boundFunction struct {
	fn          *function
	literal     *ast.FuncLit
	pkg         *SourcePackage
	receiver    *abstractValue
	capture     environment
	signature   *types.Signature
	assignments []ast.Expr
}
type environment map[types.Object]*abstractValue
type expressionSite struct {
	expression ast.Expr
	pkg        *SourcePackage
	slot       string
}
type callSite struct {
	call *ast.CallExpr
	pkg  *SourcePackage
}
type effectAnalysis struct {
	program        *Program
	root           string
	chain          []string
	active         map[string]bool
	resolving      map[types.Object]bool
	bindings       map[types.Object][]expressionSite
	fields         map[*types.Var][]expressionSite
	calls          map[string][]callSite
	findings       map[string]Finding
	memo           map[string]*abstractValue
	previous       map[string]*abstractValue
	bindingDepth   int
	typeArguments  map[*types.TypeParam][]types.Type
	recursive      bool
	implicitActive map[*abstractValue]map[string]bool
	checkedMemory  map[string]bool
}

func (a *effectAnalysis) assign(pkg *SourcePackage, target ast.Expr, value *abstractValue, env environment) {
	if target == nil {
		return
	}
	switch target := target.(type) {
	case *ast.ParenExpr:
		a.assign(pkg, target.X, value, env)
	case *ast.Ident:
		if target.Name == "_" {
			return
		}
		obj := pkg.Info.Defs[target]
		if obj == nil {
			obj = pkg.Info.Uses[target]
		}
		if obj != nil {
			env[obj] = join(env[obj], value)
		}
	case *ast.SelectorExpr:
		v := a.eval(pkg, target.X, env)
		if v != nil {
			if v.fields == nil {
				v.fields = map[string]*abstractValue{}
			}
			v.fields[target.Sel.Name] = join(v.fields[target.Sel.Name], value)
		}
	case *ast.IndexExpr:
		v := a.eval(pkg, target.X, env)
		a.eval(pkg, target.Index, env)
		if v != nil {
			v.elements = join(v.elements, value)
		}
	case *ast.StarExpr:
		v := a.eval(pkg, target.X, env)
		if v != nil && value != nil {
			if pointee := v.fields["$pointee"]; pointee != nil {
				v = pointee
			}
			*v = *join(v, value)
		}
	default:
		a.report("unresolved-effect", target.Pos(), fmt.Sprintf("unsupported assignment target %T", target))
	}
}

func valueOf(typ types.Type) *abstractValue {
	v := &abstractValue{}
	if typ != nil {
		v.types = []types.Type{typ}
		_, v.unknown = typ.Underlying().(*types.Interface)
		if _, ok := typ.Underlying().(*types.Signature); ok {
			v.unknown = true
		}
	}
	return v
}
func join(left, right *abstractValue) *abstractValue {
	return joinValues(left, right, map[[2]*abstractValue]*abstractValue{})
}
func joinValues(left, right *abstractValue, seen map[[2]*abstractValue]*abstractValue) *abstractValue {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	if left == right {
		return left
	}
	key := [2]*abstractValue{left, right}
	if value := seen[key]; value != nil {
		return value
	}
	v := left
	if !v.origin.IsValid() {
		v.origin = right.origin
	}
	v.utc = left.utc && right.utc
	v.unknown = left.unknown || right.unknown
	v.builtin = left.builtin && right.builtin
	if left.text != right.text {
		v.text = ""
	}
	seen[key] = v
	for _, typ := range right.types {
		found := false
		for _, existing := range v.types {
			if types.Identical(existing, typ) {
				found = true
				break
			}
		}
		if !found {
			v.types = append(v.types, typ)
		}
	}
	for _, fn := range right.functions {
		found := false
		for _, existing := range v.functions {
			if fn == existing || (fn.fn == existing.fn && fn.literal == existing.literal && bindingKey([]*abstractValue{fn.receiver}) == bindingKey([]*abstractValue{existing.receiver}) && captureKey(fn.capture) == captureKey(existing.capture)) {
				found = true
				break
			}
		}
		if !found {
			v.functions = append(v.functions, fn)
		}
	}
	if v.fields == nil {
		v.fields = map[string]*abstractValue{}
	}
	for key, value := range right.fields {
		v.fields[key] = joinValues(v.fields[key], value, seen)
	}
	v.elements = joinValues(left.elements, right.elements, seen)
	return v
}

func (p *Program) Effects() []Finding {
	a := &effectAnalysis{program: p, active: map[string]bool{}, resolving: map[types.Object]bool{}, bindings: map[types.Object][]expressionSite{}, fields: map[*types.Var][]expressionSite{}, calls: map[string][]callSite{}, findings: map[string]Finding{}, typeArguments: map[*types.TypeParam][]types.Type{}}
	for _, pkg := range p.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for ident, instance := range pkg.Info.Instances {
			obj := pkg.Info.Uses[ident]
			if obj == nil {
				continue
			}
			var parameters *types.TypeParamList
			switch typ := obj.Type().(type) {
			case *types.Signature:
				parameters = typ.TypeParams()
			case *types.Named:
				parameters = typ.TypeParams()
			}
			if parameters != nil {
				for i := 0; i < parameters.Len() && i < instance.TypeArgs.Len(); i++ {
					parameter := parameters.At(i)
					a.typeArguments[parameter] = append(a.typeArguments[parameter], instance.TypeArgs.At(i))
				}
			}
		}
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.CallExpr:
					if fn := calledObject(pkg.Info, node.Fun); fn != nil {
						a.calls[functionID(fn)] = append(a.calls[functionID(fn)], callSite{node, pkg})
					}
				case *ast.ValueSpec:
					for i, name := range node.Names {
						if len(node.Values) == 1 && len(node.Names) > 1 {
							a.bindings[pkg.Info.Defs[name]] = append(a.bindings[pkg.Info.Defs[name]], expressionSite{expression: node.Values[0], pkg: pkg, slot: fmt.Sprint(i)})
						} else if i < len(node.Values) {
							a.bindings[pkg.Info.Defs[name]] = append(a.bindings[pkg.Info.Defs[name]], expressionSite{expression: node.Values[i], pkg: pkg})
						} else if len(node.Values) == 1 {
							a.bindings[pkg.Info.Defs[name]] = append(a.bindings[pkg.Info.Defs[name]], expressionSite{expression: node.Values[0], pkg: pkg, slot: fmt.Sprint(i)})
						}
					}
				case *ast.AssignStmt:
					for i, lhs := range node.Lhs {
						var site expressionSite
						if len(node.Rhs) == 1 && len(node.Lhs) > 1 {
							site = expressionSite{expression: node.Rhs[0], pkg: pkg, slot: fmt.Sprint(i)}
						} else if i < len(node.Rhs) {
							site = expressionSite{expression: node.Rhs[i], pkg: pkg}
						} else {
							continue
						}
						switch lhs := lhs.(type) {
						case *ast.Ident:
							obj := pkg.Info.Defs[lhs]
							if obj == nil {
								obj = pkg.Info.Uses[lhs]
							}
							if obj != nil {
								a.bindings[obj] = append(a.bindings[obj], site)
							}
						case *ast.SelectorExpr:
							if field, ok := pkg.Info.Uses[lhs.Sel].(*types.Var); ok {
								a.fields[field] = append(a.fields[field], site)
							}
						}
					}
				case *ast.CompositeLit:
					typ := pkg.Info.TypeOf(node)
					if pointer, ok := typ.(*types.Pointer); ok {
						typ = pointer.Elem()
					}
					if typ == nil {
						return true
					}
					structure, ok := typ.Underlying().(*types.Struct)
					if !ok {
						return true
					}
					for i, element := range node.Elts {
						if keyed, ok := element.(*ast.KeyValueExpr); ok {
							if key, ok := keyed.Key.(*ast.Ident); ok {
								for j := 0; j < structure.NumFields(); j++ {
									if field := structure.Field(j); field.Name() == key.Name {
										a.fields[field] = append(a.fields[field], expressionSite{expression: keyed.Value, pkg: pkg})
									}
								}
							}
						} else if i < structure.NumFields() {
							field := structure.Field(i)
							a.fields[field] = append(a.fields[field], expressionSite{expression: element, pkg: pkg})
						}
					}
				}
				return true
			})
		}
	}
	for _, pkg := range p.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				for _, site := range a.literalBindings(pkg, call.Fun, map[types.Object]bool{}) {
					literal := site.expression.(*ast.FuncLit)
					signature := site.pkg.Info.TypeOf(literal).(*types.Signature)
					for i := 0; i < signature.Params().Len() && i < len(call.Args); i++ {
						parameter := signature.Params().At(i)
						a.bindings[parameter] = append(a.bindings[parameter], expressionSite{expression: call.Args[i], pkg: pkg})
					}
				}
				return true
			})
		}
	}
	var roots []*function
	for _, fn := range p.Functions {
		if p.IsPureFile(p.Files.Position(fn.declaration.Pos()).Filename) {
			roots = append(roots, fn)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return functionID(roots[i].object) < functionID(roots[j].object) })
	for _, fn := range roots {
		a.root = functionID(fn.object)
		a.analyze(fn.declaration.Pos(), func() {
			a.call(&boundFunction{fn: fn, pkg: fn.pkg}, nil, fn.declaration.Pos())
		})
	}
	a.initializers()
	var findings []Finding
	for _, finding := range a.findings {
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Path+findings[i].Detail < findings[j].Path+findings[j].Detail })
	return findings
}

func (a *effectAnalysis) analyze(pos token.Pos, run func()) {
	a.chain = nil
	a.previous = map[string]*abstractValue{}
	allFindings := a.findings
	for round := 0; round < 256; round++ {
		before := summaryKeys(a.previous)
		a.findings = map[string]Finding{}
		a.memo = map[string]*abstractValue{}
		a.recursive = false
		run()
		if !a.recursive {
			break
		}
		current := map[string]*abstractValue{}
		for key, value := range a.memo {
			current[key] = normalizeAllocations(cloneValue(value))
		}
		if equalSummaryKeys(before, summaryKeys(current)) {
			break
		}
		a.previous = current
		if round == 255 {
			a.report("unresolved-effect", pos, "recursive effect summaries did not converge")
		}
	}
	for key, finding := range a.findings {
		allFindings[key] = finding
	}
	a.findings = allFindings
}

func (p *Program) IsPureFile(path string) bool {
	for _, pkg := range p.Packages {
		if !Within(pkg.Types.Path(), p.Module) {
			continue
		}
		relative := strings.TrimPrefix(pkg.Types.Path(), p.Module+"/")
		if filepath.Dir(path) != pkg.Metadata.Dir {
			continue
		}
		switch {
		case relative == "world", relative == "record", Within(relative, "world/mailbox"), Within(relative, "runner/internal/exploration"), Within(relative, "target/internal/capabilitypolicy"):
			return true
		case relative == "runner/internal/campaign":
			return filepath.Base(path) == "controller.go"
		case relative == "target":
			return filepath.Base(path) == "capability_evaluation.go"
		case relative == "runner/internal/execution":
			return filepath.Base(path) == "simulation_progress.go"
		}
	}
	return false
}

func (a *effectAnalysis) report(category string, pos token.Pos, detail string) {
	if a.bindingDepth != 0 {
		return
	}
	path := a.program.Files.Position(pos).String()
	detail = a.root + " -> " + strings.Join(a.chain, " -> ") + " -> " + detail
	key := category + path + detail
	a.findings[key] = Finding{Category: category, Path: path, Detail: detail}
}

func (a *effectAnalysis) resolve(obj types.Object) *abstractValue {
	if obj == nil {
		return nil
	}
	if obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "UTC" {
		v := valueOf(obj.Type())
		v.utc = true
		return v
	}
	if _, generic := obj.Type().(*types.TypeParam); generic {
		return a.concreteValue(obj.Type(), map[*types.TypeParam]bool{})
	}
	if !needsBindings(obj.Type(), map[types.Type]bool{}) {
		return valueOf(obj.Type())
	}
	if fn, ok := obj.(*types.Func); ok {
		return &abstractValue{functions: []*boundFunction{{fn: a.program.Functions[functionID(fn)], pkg: a.program.Packages[fn.Pkg().Path()]}}}
	}
	if a.resolving[obj] {
		return valueOf(obj.Type())
	}
	a.resolving[obj] = true
	defer delete(a.resolving, obj)
	a.bindingDepth++
	defer func() { a.bindingDepth-- }()
	var result *abstractValue
	for _, site := range a.bindings[obj] {
		result = join(result, a.bindingValue(site))
	}
	if field, ok := obj.(*types.Var); ok && field.IsField() {
		for _, site := range a.fields[field] {
			result = join(result, a.bindingValue(site))
		}
	}
	if result != nil {
		return result
	}
	for _, fn := range a.program.Functions {
		if fn.pkg.Types != obj.Pkg() {
			continue
		}
		signature := fn.object.Type().(*types.Signature)
		for i := 0; i < signature.Params().Len(); i++ {
			if signature.Params().At(i) == obj {
				for _, site := range a.calls[functionID(fn.object)] {
					if i < len(site.call.Args) {
						result = join(result, a.eval(site.pkg, site.call.Args[i], environment{}))
					}
				}
			}
		}
	}
	if result != nil {
		return result
	}
	return valueOf(obj.Type())
}

func (a *effectAnalysis) bindingValue(site expressionSite) *abstractValue {
	v := a.eval(site.pkg, site.expression, environment{})
	if site.slot != "" && v != nil {
		return v.fields[site.slot]
	}
	return v
}

func (a *effectAnalysis) literalBindings(pkg *SourcePackage, expression ast.Expr, seen map[types.Object]bool) []expressionSite {
	switch expression := expression.(type) {
	case *ast.FuncLit:
		return []expressionSite{{expression: expression, pkg: pkg}}
	case *ast.ParenExpr:
		return a.literalBindings(pkg, expression.X, seen)
	case *ast.Ident:
		object := pkg.Info.Uses[expression]
		if object == nil {
			object = pkg.Info.Defs[expression]
		}
		if object == nil || seen[object] {
			return nil
		}
		seen[object] = true
		var result []expressionSite
		for _, site := range a.bindings[object] {
			result = append(result, a.literalBindings(site.pkg, site.expression, seen)...)
		}
		return result
	case *ast.SelectorExpr:
		field, ok := pkg.Info.Uses[expression.Sel].(*types.Var)
		if !ok || seen[field] {
			return nil
		}
		seen[field] = true
		var result []expressionSite
		for _, site := range a.fields[field] {
			result = append(result, a.literalBindings(site.pkg, site.expression, seen)...)
		}
		return result
	}
	return nil
}

func (a *effectAnalysis) concreteValue(typ types.Type, seen map[*types.TypeParam]bool) *abstractValue {
	if parameter, ok := typ.(*types.TypeParam); ok {
		if seen[parameter] {
			return nil
		}
		seen[parameter] = true
		var result *abstractValue
		for _, actual := range a.typeArguments[parameter] {
			result = join(result, a.concreteValue(actual, seen))
		}
		delete(seen, parameter)
		if result != nil {
			return result
		}
	}
	return valueOf(typ)
}

func needsBindings(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	switch typ := typ.Underlying().(type) {
	case *types.Signature, *types.Interface:
		return true
	case *types.Tuple:
		for i := 0; i < typ.Len(); i++ {
			if needsBindings(typ.At(i).Type(), seen) {
				return true
			}
		}
	case *types.Pointer:
		return needsBindings(typ.Elem(), seen)
	case *types.Slice:
		return needsBindings(typ.Elem(), seen)
	case *types.Array:
		return needsBindings(typ.Elem(), seen)
	case *types.Map:
		return needsBindings(typ.Elem(), seen)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if needsBindings(typ.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func (a *effectAnalysis) eval(pkg *SourcePackage, expr ast.Expr, env environment) *abstractValue {
	if expr == nil {
		return nil
	}
	typ := pkg.Info.TypeOf(expr)
	if a.bindingDepth > 0 && typ != nil && !needsBindings(typ, map[types.Type]bool{}) && !containsLocation(typ, map[types.Type]bool{}) && pkg.Info.Types[expr].Value == nil {
		return valueOf(typ)
	}
	switch expr := expr.(type) {
	case *ast.BasicLit:
		v := valueOf(typ)
		if expr.Kind == token.STRING {
			v.text, _ = strconv.Unquote(expr.Value)
		}
		return v
	case *ast.Ident:
		obj := pkg.Info.Uses[expr]
		if obj == nil {
			obj = pkg.Info.Defs[expr]
		}
		if obj != nil {
			if value := env[obj]; value != nil {
				return value
			}
			if c, ok := obj.(*types.Const); ok && c.Val().Kind() == constant.Bool {
				v := valueOf(typ)
				v.truth = -1
				if constant.BoolVal(c.Val()) {
					v.truth = 1
				}
				return v
			}
		}
		return a.resolve(obj)
	case *ast.SelectorExpr:
		obj := pkg.Info.Uses[expr.Sel]
		if fn, ok := obj.(*types.Func); ok {
			if fn.Pkg() == nil {
				return valueOf(fn.Type())
			}
			bound := &boundFunction{fn: a.program.Functions[functionID(fn)], pkg: a.program.Packages[fn.Pkg().Path()]}
			if selection := pkg.Info.Selections[expr]; selection != nil && selection.Kind() == types.MethodVal {
				bound.receiver = a.eval(pkg, expr.X, env)
			}
			return &abstractValue{functions: []*boundFunction{bound}}
		}
		if selection := pkg.Info.Selections[expr]; selection != nil {
			receiver := a.eval(pkg, expr.X, env)
			if receiver != nil && receiver.fields[expr.Sel.Name] != nil {
				return receiver.fields[expr.Sel.Name]
			}
		}
		return a.resolve(obj)
	case *ast.FuncLit:
		capture := environment{}
		ast.Inspect(expr.Body, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok {
				obj := pkg.Info.Uses[ident]
				if obj != nil && env[obj] != nil {
					capture[obj] = env[obj]
				}
			}
			return true
		})
		closure := &boundFunction{literal: expr, pkg: pkg, capture: capture}
		if a.program.IsPureFile(a.program.Files.Position(expr.Pos()).Filename) {
			a.call(closure, nil, expr.Pos())
		}
		return &abstractValue{origin: expr.Pos(), functions: []*boundFunction{closure}}
	case *ast.CompositeLit:
		v := valueOf(typ)
		v.origin = expr.Pos()
		v.fields = map[string]*abstractValue{}
		structure, _ := typ.Underlying().(*types.Struct)
		for i, element := range expr.Elts {
			if pair, ok := element.(*ast.KeyValueExpr); ok {
				if key, ok := pair.Key.(*ast.Ident); ok && structure != nil {
					v.fields[key.Name] = a.eval(pkg, pair.Value, env)
				} else {
					a.eval(pkg, pair.Key, env)
					v.elements = join(v.elements, a.eval(pkg, pair.Value, env))
				}
			} else if structure != nil && i < structure.NumFields() {
				v.fields[structure.Field(i).Name()] = a.eval(pkg, element, env)
			} else {
				v.elements = join(v.elements, a.eval(pkg, element, env))
			}
		}
		return v
	case *ast.UnaryExpr:
		v := a.eval(pkg, expr.X, env)
		if v != nil && expr.Op == token.AND {
			copy := *v
			copy.origin = expr.Pos()
			copy.types = []types.Type{typ}
			copy.fields = map[string]*abstractValue{}
			for key, value := range v.fields {
				copy.fields[key] = value
			}
			copy.fields["$pointee"] = v
			return &copy
		}
		return v
	case *ast.ParenExpr:
		return a.eval(pkg, expr.X, env)
	case *ast.StarExpr:
		v := a.eval(pkg, expr.X, env)
		if v == nil {
			return valueOf(typ)
		}
		if pointee := v.fields["$pointee"]; pointee != nil {
			v = pointee
		}
		copy := *v
		copy.types = []types.Type{typ}
		copy.fields = map[string]*abstractValue{}
		for name, child := range v.fields {
			if name != "$pointee" {
				copy.fields[name] = child
			}
		}
		return &copy
	case *ast.IndexExpr:
		v := a.eval(pkg, expr.X, env)
		a.eval(pkg, expr.Index, env)
		if v != nil && len(v.functions) > 0 {
			return v
		}
		if v != nil && v.elements != nil {
			return v.elements
		}
		child := a.concreteValue(typ, map[*types.TypeParam]bool{})
		if v != nil && v.builtin {
			child.builtin = true
			child.unknown = false
		}
		return child
	case *ast.IndexListExpr:
		return a.eval(pkg, expr.X, env)
	case *ast.SliceExpr:
		v := a.eval(pkg, expr.X, env)
		a.eval(pkg, expr.Low, env)
		a.eval(pkg, expr.High, env)
		a.eval(pkg, expr.Max, env)
		return v
	case *ast.TypeAssertExpr:
		v := a.eval(pkg, expr.X, env)
		if v != nil && !v.unknown {
			return v
		}
		return valueOf(typ)
	case *ast.BinaryExpr:
		a.eval(pkg, expr.X, env)
		a.eval(pkg, expr.Y, env)
		return valueOf(typ)
	case *ast.CallExpr:
		return a.evaluateCall(pkg, expr, env)
	default:
		return valueOf(typ)
	}
}

func containsLocation(typ types.Type, seen map[types.Type]bool) bool {
	if typ == nil || seen[typ] {
		return false
	}
	seen[typ] = true
	if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "time" && named.Obj().Name() == "Location" {
		return true
	}
	switch typ := typ.Underlying().(type) {
	case *types.Pointer:
		return containsLocation(typ.Elem(), seen)
	case *types.Slice:
		return containsLocation(typ.Elem(), seen)
	case *types.Array:
		return containsLocation(typ.Elem(), seen)
	case *types.Map:
		return containsLocation(typ.Elem(), seen)
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			if containsLocation(typ.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func cloneEnvironment(env environment) environment {
	result := environment{}
	for obj, value := range env {
		if obj != nil {
			result[obj] = value
		}
	}
	return result
}

func (a *effectAnalysis) evaluateCall(pkg *SourcePackage, call *ast.CallExpr, env environment) *abstractValue {
	var args []*abstractValue
	for _, arg := range call.Args {
		args = append(args, a.eval(pkg, arg, env))
	}
	if typ, ok := pkg.Info.Types[call.Fun]; ok && typ.IsType() {
		destination := pkg.Info.TypeOf(call)
		if len(args) > 0 && args[0] != nil {
			if _, dynamic := destination.Underlying().(*types.Interface); dynamic {
				return args[0]
			}
			converted := *args[0]
			converted.types = []types.Type{destination}
			converted.builtin = false
			converted.origin = call.Pos()
			return &converted
		}
		return valueOf(destination)
	}
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if builtin, ok := pkg.Info.Uses[ident].(*types.Builtin); ok {
			switch builtin.Name() {
			case "copy":
				if len(args) == 2 && args[0] != nil {
					args[0].elements = join(args[0].elements, elementValue(args[1]))
				}
				return valueOf(pkg.Info.TypeOf(call))
			case "append":
				v := valueOf(pkg.Info.TypeOf(call))
				if len(args) > 0 {
					v = args[0]
				}
				if v == nil {
					v = valueOf(pkg.Info.TypeOf(call))
				}
				for _, arg := range args[1:] {
					if call.Ellipsis.IsValid() {
						arg = elementValue(arg)
					}
					v.elements = join(v.elements, arg)
				}
				return v
			case "new":
				return valueOf(pkg.Info.TypeOf(call))
			case "make":
				return valueOf(pkg.Info.TypeOf(call))
			default:
				return valueOf(pkg.Info.TypeOf(call))
			}
		}
	}
	fn := calledObject(pkg.Info, call.Fun)
	var receiver *abstractValue
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok && pkg.Info.Selections[selector] != nil {
		receiver = a.eval(pkg, selector.X, env)
	}
	if fn != nil {
		if signature := fn.Type().(*types.Signature); signature.Recv() != nil && receiver != nil {
			if _, dynamic := signature.Recv().Type().Underlying().(*types.Interface); dynamic {
				result := a.implicit(receiver, []string{fn.Name()}, args, call.Pos(), map[types.Type]bool{})
				if result != nil {
					return result
				}
				return tupleValue(signature.Results())
			}
		}
		if result, handled := a.standard(fn, receiver, args, call.Pos()); handled {
			return result
		}
		if declaration := a.program.Functions[functionID(fn)]; declaration != nil {
			return a.call(&boundFunction{fn: declaration, pkg: declaration.pkg, receiver: receiver}, args, call.Pos())
		}
	}
	value := a.eval(pkg, call.Fun, env)
	var result *abstractValue
	if value != nil {
		for _, bound := range value.functions {
			result = join(result, a.call(bound, args, call.Pos()))
		}
		if len(value.functions) > 0 && !value.unknown {
			return result
		}
	}
	if fn != nil {
		a.report("unresolved-effect", call.Pos(), "unresolved call "+functionID(fn))
	} else {
		a.report("unresolved-effect", call.Pos(), "unresolved callback")
	}
	return valueOf(pkg.Info.TypeOf(call))
}

func (a *effectAnalysis) call(bound *boundFunction, args []*abstractValue, pos token.Pos) *abstractValue {
	if bound == nil || bound.pkg == nil || (bound.fn == nil && bound.literal == nil) {
		a.report("unresolved-effect", pos, "function has no analyzable source")
		return nil
	}
	if bound.fn != nil {
		if result, handled := a.standard(bound.fn.object, bound.receiver, args, pos); handled {
			return result
		}
	}
	var body *ast.BlockStmt
	var signature *types.Signature
	id := ""
	if bound.fn != nil {
		body = bound.fn.declaration.Body
		signature = bound.fn.object.Type().(*types.Signature)
		id = functionID(bound.fn.object)
	} else {
		body = bound.literal.Body
		signature = bound.signature
		if signature == nil {
			signature = bound.pkg.Info.TypeOf(bound.literal).(*types.Signature)
		}
		id = a.program.Files.Position(bound.literal.Pos()).String()
	}
	if body == nil {
		a.report("unresolved-effect", pos, "bodyless call "+id)
		return nil
	}
	key := fmt.Sprint(a.bindingDepth != 0) + id + bindingGraphKey(args, bound.receiver, bound.capture)
	if a.active[key] {
		a.recursive = true
		if previous, ok := a.previous[key]; ok {
			return cloneValue(previous)
		}
		return bottomResults(signature.Results())
	}
	a.active[key] = true
	defer delete(a.active, key)
	a.chain = append(a.chain, id)
	defer func() { a.chain = a.chain[:len(a.chain)-1] }()
	env := cloneEnvironment(bound.capture)
	for i := 0; i < signature.Params().Len(); i++ {
		param := signature.Params().At(i)
		if i < len(args) && args[i] != nil {
			env[param] = args[i]
		} else {
			env[param] = a.resolve(param)
		}
	}
	for i, target := range bound.assignments {
		if i < len(args) {
			a.assign(bound.pkg, target, args[i], env)
		}
	}
	if receiver := signature.Recv(); receiver != nil {
		if bound.receiver != nil {
			env[receiver] = bound.receiver
		} else {
			env[receiver] = valueOf(receiver.Type())
		}
	}
	var result *abstractValue
	var inspect func(ast.Node) bool
	inspect = func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			if a.program.IsPureFile(a.program.Files.Position(node.Pos()).Filename) {
				a.eval(bound.pkg, node, env)
			}
			return false
		case *ast.IfStmt:
			if node.Init != nil {
				ast.Inspect(node.Init, inspect)
			}
			condition := a.eval(bound.pkg, node.Cond, env)
			if condition == nil || condition.truth >= 0 {
				ast.Inspect(node.Body, inspect)
			}
			if node.Else != nil && (condition == nil || condition.truth <= 0) {
				ast.Inspect(node.Else, inspect)
			}
			return false
		case *ast.TypeSwitchStmt:
			if node.Init != nil {
				ast.Inspect(node.Init, inspect)
			}
			var expression ast.Expr
			switch assignment := node.Assign.(type) {
			case *ast.AssignStmt:
				if len(assignment.Rhs) > 0 {
					expression = assignment.Rhs[0]
				}
			case *ast.ExprStmt:
				expression = assignment.X
			}
			if assertion, ok := expression.(*ast.TypeAssertExpr); ok {
				expression = assertion.X
			}
			value := a.eval(bound.pkg, expression, env)
			for _, statement := range node.Body.List {
				clause := statement.(*ast.CaseClause)
				if obj := bound.pkg.Info.Implicits[clause]; obj != nil {
					narrowed := valueOf(obj.Type())
					if value != nil {
						copy := *value
						narrowed = &copy
						narrowed.types = []types.Type{obj.Type()}
						_, narrowed.unknown = obj.Type().Underlying().(*types.Interface)
						if len(clause.List) == 0 {
							narrowed.types = value.types
							narrowed.unknown = value.unknown
						}
						if value.builtin {
							narrowed.unknown = false
						}
					}
					env[obj] = narrowed
				}
				for _, statement := range clause.Body {
					ast.Inspect(statement, inspect)
				}
			}
			return false
		case *ast.GoStmt:
			a.report("host-effect", node.Pos(), "goroutine start")
			a.eval(bound.pkg, node.Call, env)
			return false
		case *ast.AssignStmt:
			var values []*abstractValue
			for _, expr := range node.Rhs {
				values = append(values, a.eval(bound.pkg, expr, env))
			}
			if len(node.Rhs) == 1 && len(node.Lhs) > 1 && len(values) > 0 && values[0] != nil {
				if assertion, ok := node.Rhs[0].(*ast.TypeAssertExpr); ok {
					source := a.eval(bound.pkg, assertion.X, env)
					target := bound.pkg.Info.TypeOf(assertion)
					value := valueOf(target)
					if source != nil {
						copy := *source
						value = &copy
						value.types = []types.Type{target}
						value.unknown = false
					}
					okValue := valueOf(types.Typ[types.Bool])
					if source != nil && !source.unknown {
						all, anyMatch := true, false
						for _, actual := range source.types {
							if _, dynamic := actual.Underlying().(*types.Interface); dynamic {
								all = false
								continue
							}
							matches := types.AssignableTo(actual, target)
							all = all && matches
							anyMatch = anyMatch || matches
						}
						if all {
							okValue.truth = 1
						} else if !anyMatch {
							okValue.truth = -1
						}
					}
					values = []*abstractValue{value, okValue}
				} else {
					v := values[0]
					values = nil
					for i := range node.Lhs {
						values = append(values, v.fields[fmt.Sprint(i)])
					}
				}
			}
			for i, lhs := range node.Lhs {
				if i < len(values) {
					a.assign(bound.pkg, lhs, values[i], env)
				}
			}
			return false
		case *ast.ValueSpec:
			var tuple *abstractValue
			if len(node.Values) == 1 && len(node.Names) > 1 {
				tuple = a.eval(bound.pkg, node.Values[0], env)
			}
			for i, name := range node.Names {
				obj := bound.pkg.Info.Defs[name]
				if obj == nil {
					continue
				}
				if tuple != nil {
					env[obj] = tuple.fields[fmt.Sprint(i)]
				} else if i < len(node.Values) {
					env[obj] = a.eval(bound.pkg, node.Values[i], env)
				} else {
					env[obj] = valueOf(obj.Type())
					if _, ok := obj.Type().Underlying().(*types.Interface); ok {
						env[obj].builtin = true
						env[obj].unknown = false
					}
				}
			}
			return false
		case *ast.RangeStmt:
			v := a.eval(bound.pkg, node.X, env)
			if iterator, ok := bound.pkg.Info.TypeOf(node.X).Underlying().(*types.Signature); ok {
				yieldSignature := iterator.Params().At(0).Type().Underlying().(*types.Signature)
				yield := &abstractValue{origin: node.For, functions: []*boundFunction{{
					literal: &ast.FuncLit{Type: &ast.FuncType{Func: node.For}, Body: node.Body},
					pkg:     bound.pkg, capture: cloneEnvironment(env),
					signature: yieldSignature, assignments: []ast.Expr{node.Key, node.Value},
				}}}
				a.invoke(v, []*abstractValue{yield}, node.Pos())
				return false
			}
			if node.Value != nil {
				value := elementValue(v)
				if value == nil {
					value = a.concreteValue(bound.pkg.Info.TypeOf(node.Value), map[*types.TypeParam]bool{})
					if v != nil && v.builtin {
						value.builtin = true
						value.unknown = false
					}
				}
				a.assign(bound.pkg, node.Value, value, env)
			}
		case *ast.ReturnStmt:
			v := &abstractValue{fields: map[string]*abstractValue{}}
			if len(node.Results) == 1 && signature.Results().Len() > 1 {
				if returned := a.eval(bound.pkg, node.Results[0], env); returned != nil {
					v.fields = returned.fields
				}
			} else {
				for i, expr := range node.Results {
					v.fields[fmt.Sprint(i)] = a.eval(bound.pkg, expr, env)
				}
				if len(node.Results) == 0 {
					for i := 0; i < signature.Results().Len(); i++ {
						obj := signature.Results().At(i)
						v.fields[fmt.Sprint(i)] = env[obj]
					}
				}
			}
			result = join(result, v)
			return false
		case *ast.CallExpr:
			a.eval(bound.pkg, node, env)
			return false
		}
		return true
	}
	ast.Inspect(body, inspect)
	if result == nil {
		result = tupleValue(signature.Results())
	} else if signature.Results().Len() == 1 {
		result = result.fields["0"]
	}
	compactFields(result, map[*abstractValue]bool{})
	if a.memo != nil {
		a.memo[key] = result
	}
	return result
}

func compactFields(v *abstractValue, seen map[*abstractValue]bool) {
	if v == nil || seen[v] {
		return
	}
	seen[v] = true
	for _, typ := range v.types {
		for {
			pointer, ok := typ.(*types.Pointer)
			if !ok {
				break
			}
			typ = pointer.Elem()
		}
		structure, ok := typ.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for i := 0; i < structure.NumFields(); i++ {
			field := structure.Field(i)
			child := v.fields[field.Name()]
			if child == nil {
				continue
			}
			if !needsBindings(field.Type(), map[types.Type]bool{}) && !child.utc && child.text == "" && len(child.functions) == 0 {
				delete(v.fields, field.Name())
			}
		}
	}
	for _, child := range v.fields {
		compactFields(child, seen)
	}
	compactFields(v.elements, seen)
	for _, fn := range v.functions {
		compactFields(fn.receiver, seen)
		for _, child := range fn.capture {
			compactFields(child, seen)
		}
	}
}

func normalizeAllocations(value *abstractValue) *abstractValue {
	allocated := map[token.Pos]*abstractValue{}
	seen := map[*abstractValue]bool{}
	var nodes []*abstractValue
	var collect func(*abstractValue)
	collect = func(v *abstractValue) {
		if v == nil || seen[v] {
			return
		}
		seen[v] = true
		nodes = append(nodes, v)
		if v.origin.IsValid() && allocated[v.origin] == nil {
			allocated[v.origin] = v
		}
		var names []string
		for name := range v.fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			collect(v.fields[name])
		}
		collect(v.elements)
		for _, fn := range v.functions {
			collect(fn.receiver)
			for _, child := range fn.capture {
				collect(child)
			}
		}
	}
	collect(value)
	for _, v := range nodes {
		if representative := allocated[v.origin]; v.origin.IsValid() && representative != v {
			join(representative, v)
		}
	}
	representative := func(v *abstractValue) *abstractValue {
		if v != nil && v.origin.IsValid() {
			return allocated[v.origin]
		}
		return v
	}
	for _, v := range nodes {
		for name, child := range v.fields {
			v.fields[name] = representative(child)
		}
		v.elements = representative(v.elements)
		for _, fn := range v.functions {
			fn.receiver = representative(fn.receiver)
			for obj, child := range fn.capture {
				fn.capture[obj] = representative(child)
			}
		}
	}
	for _, v := range nodes {
		var functions []*boundFunction
		for _, fn := range v.functions {
			duplicate := false
			for _, existing := range functions {
				if fn.fn == existing.fn && fn.literal == existing.literal && bindingKey([]*abstractValue{fn.receiver}) == bindingKey([]*abstractValue{existing.receiver}) && captureKey(fn.capture) == captureKey(existing.capture) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				functions = append(functions, fn)
			}
		}
		v.functions = functions
	}
	return representative(value)
}

func tupleValue(tuple *types.Tuple) *abstractValue {
	if tuple == nil {
		return nil
	}
	if tuple.Len() == 1 {
		return valueOf(tuple.At(0).Type())
	}
	v := &abstractValue{fields: map[string]*abstractValue{}}
	for i := 0; i < tuple.Len(); i++ {
		v.fields[fmt.Sprint(i)] = valueOf(tuple.At(i).Type())
	}
	return v
}
func bottomResults(tuple *types.Tuple) *abstractValue {
	v := tupleValue(tuple)
	var mark func(*abstractValue)
	mark = func(v *abstractValue) {
		if v == nil {
			return
		}
		v.unknown = false
		v.builtin = true
		for _, child := range v.fields {
			mark(child)
		}
	}
	mark(v)
	return v
}
func cloneValue(value *abstractValue) *abstractValue {
	seen := map[*abstractValue]*abstractValue{}
	functions := map[*boundFunction]*boundFunction{}
	var clone func(*abstractValue) *abstractValue
	clone = func(v *abstractValue) *abstractValue {
		if v == nil {
			return nil
		}
		if existing := seen[v]; existing != nil {
			return existing
		}
		copy := *v
		result := &copy
		seen[v] = result
		result.types = append([]types.Type(nil), v.types...)
		result.fields = map[string]*abstractValue{}
		for name, child := range v.fields {
			result.fields[name] = clone(child)
		}
		result.elements = clone(v.elements)
		result.functions = nil
		for _, fn := range v.functions {
			copied := functions[fn]
			if copied == nil {
				copy := *fn
				copied = &copy
				functions[fn] = copied
				copied.receiver = clone(fn.receiver)
				copied.capture = environment{}
				for obj, value := range fn.capture {
					copied.capture[obj] = clone(value)
				}
			}
			result.functions = append(result.functions, copied)
		}
		return result
	}
	return clone(value)
}
func summaryKeys(values map[string]*abstractValue) map[string]string {
	keys := map[string]string{}
	for key, value := range values {
		keys[key] = bindingKey([]*abstractValue{value})
	}
	return keys
}
func equalSummaryKeys(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}
func bindingKey(values []*abstractValue) string {
	var out strings.Builder
	seen := map[*abstractValue]int{}
	var write func(*abstractValue)
	write = func(v *abstractValue) {
		if v == nil {
			out.WriteString("nil;")
			return
		}
		if n, ok := seen[v]; ok {
			fmt.Fprintf(&out, "ref%d;", n)
			return
		}
		seen[v] = len(seen)
		fmt.Fprintf(&out, "{%t,%t,%t,%q,%d;", v.utc, v.unknown, v.builtin, v.text, v.truth)
		var names []string
		for _, typ := range v.types {
			names = append(names, types.TypeString(typ, func(p *types.Package) string { return p.Path() }))
		}
		sort.Strings(names)
		fmt.Fprintf(&out, "types%q;", names)
		for _, fn := range v.functions {
			if fn.fn != nil {
				out.WriteString(functionID(fn.fn.object))
			} else if fn.literal != nil {
				fmt.Fprintf(&out, "literal%d", fn.literal.Pos())
			}
			write(fn.receiver)
			var objects []types.Object
			for obj := range fn.capture {
				if obj != nil {
					objects = append(objects, obj)
				}
			}
			sort.Slice(objects, func(i, j int) bool { return objects[i].Pos() < objects[j].Pos() })
			for _, obj := range objects {
				fmt.Fprintf(&out, "capture%d:", obj.Pos())
				write(fn.capture[obj])
			}
		}
		var fields []string
		for name := range v.fields {
			fields = append(fields, name)
		}
		sort.Strings(fields)
		for _, name := range fields {
			fmt.Fprintf(&out, "field%q:", name)
			write(v.fields[name])
		}
		out.WriteString("elements:")
		write(v.elements)
		out.WriteString("};")
	}
	for i, v := range values {
		fmt.Fprintf(&out, "arg%d:", i)
		write(v)
	}
	return out.String()
}

func captureKey(env environment) string {
	var objects []types.Object
	for obj := range env {
		if obj != nil {
			objects = append(objects, obj)
		}
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].Pos() < objects[j].Pos() })
	var out strings.Builder
	for _, obj := range objects {
		fmt.Fprintf(&out, "%d:%s", obj.Pos(), bindingKey([]*abstractValue{env[obj]}))
	}
	return out.String()
}

func bindingGraphKey(args []*abstractValue, receiver *abstractValue, capture environment) string {
	graph := &abstractValue{fields: map[string]*abstractValue{"receiver": receiver}}
	for i, value := range args {
		graph.fields[fmt.Sprintf("arg%06d", i)] = value
	}
	for obj, value := range capture {
		if obj != nil {
			graph.fields[fmt.Sprintf("capture%012d", obj.Pos())] = value
		}
	}
	return bindingKey([]*abstractValue{normalizeAllocations(cloneValue(graph))})
}

func calledObject(info *types.Info, expression ast.Expr) *types.Func {
	switch expression := expression.(type) {
	case *ast.Ident:
		fn, _ := info.Uses[expression].(*types.Func)
		return fn
	case *ast.SelectorExpr:
		fn, _ := info.Uses[expression.Sel].(*types.Func)
		return fn
	case *ast.IndexExpr:
		return calledObject(info, expression.X)
	case *ast.IndexListExpr:
		return calledObject(info, expression.X)
	case *ast.ParenExpr:
		return calledObject(info, expression.X)
	}
	return nil
}
