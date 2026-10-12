package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
)

const canonicalNativeWriterState = "$canonical-native-writer"
const canonicalSHA256Backend = "crypto/internal/fips140/sha256"

// Optional constructor tracking augments ordinary analysis only for exact
// local callsites. Unsupported shapes retain ordinary source/model behavior;
// the canonical boundary independently requires construction-owned facts.
func (a *effectAnalysis) canonicalNativeWriterConstructor(fn *types.Func, args []*abstractValue, pos token.Pos) (*abstractValue, bool) {
	if fn == nil || fn.Pkg() == nil {
		return nil, false
	}
	id := functionID(fn)
	if id != "io.MultiWriter" && id != "crypto/sha256.New" {
		return nil, false
	}
	if a.bindingDepth != 0 || !a.canonicalNativeWriterLocalCall(fn, pos) {
		return nil, false
	}
	fallback := modeledResults(fn)
	reject := func(detail string) (*abstractValue, bool) {
		a.report("unresolved-effect", pos, "native writer constructor: "+detail)
		return fallback, true
	}
	if !a.canonicalNativeWriterFunction(fn, pos) {
		return reject("native declaration/source witness changed")
	}
	call := a.canonicalNativeWriterCall(fn, pos)
	if call == nil || call.Ellipsis.IsValid() || len(call.Args) != len(args) || a.bindingDepth != 0 {
		return reject("missing exact local non-expanded constructor call")
	}
	// State is not an allocation. Giving it the result's origin would let
	// normalizeAllocations merge the witness into the actual pointer location.
	state := &abstractValue{text: id, fields: map[string]*abstractValue{
		"$constructor": {text: strconv.Itoa(int(pos))},
	}}
	var typ types.Type
	if id == "io.MultiWriter" {
		object := a.program.Packages["io"].Types.Scope().Lookup("multiWriter")
		if object == nil {
			return reject("concrete native multiWriter type absent")
		}
		typ = types.NewPointer(object.Type())
		for _, argument := range args {
			if argument == nil || argument.unknown || len(argument.types) == 0 {
				return reject("unknown delegate binding")
			}
			// The pinned constructor flattens an existing native multiWriter.
			// Read its separately indexed graph, never its joined slice element.
			children := []*abstractValue{argument}
			for _, dynamic := range argument.types {
				if types.Identical(dynamic, typ) {
					class, nested, ok := a.canonicalNativeWriterDelegates(argument, pos)
					if !ok || class != "multi" {
						return reject("unproven nested multiWriter state")
					}
					children = nested
					break
				}
			}
			for _, child := range children {
				state.fields[strconv.Itoa(len(state.fields)-1)] = child
			}
		}
	} else {
		if len(args) != 0 || !a.canonicalNativeWriterSource("crypto/internal/boring", pos) ||
			!a.canonicalNativeWriterSource(canonicalSHA256Backend, pos) {
			return reject("unqualified SHA backend selection")
		}
		boring := a.program.Packages["crypto/internal/boring"]
		enabled, ok := boring.Types.Scope().Lookup("Enabled").(*types.Const)
		if !ok || enabled.Val().Kind() != constant.Bool || constant.BoolVal(enabled.Val()) {
			return reject("SHA backend is not source-proven non-BoringCrypto")
		}
		backend := a.program.Packages[canonicalSHA256Backend]
		newFunction, ok := backend.Types.Scope().Lookup("New").(*types.Func)
		digest := backend.Types.Scope().Lookup("Digest")
		if !ok || digest == nil || !a.canonicalNativeWriterFunction(newFunction, pos) {
			return reject("concrete native SHA constructor absent")
		}
		typ = types.NewPointer(digest.Type())
		if signature, ok := newFunction.Type().(*types.Signature); !ok || signature.Params().Len() != 0 ||
			signature.Results().Len() != 1 || !types.Identical(signature.Results().At(0).Type(), typ) {
			return reject("concrete native SHA constructor signature changed")
		}
	}
	pointer := typ.(*types.Pointer)
	result := addressValue(zeroValue(pointer.Elem()), typ, pos)
	if id == "io.MultiWriter" {
		// Preserve the actual native field shape for ordinary callback analysis.
		// Its homogeneous element is deliberately unknown: certificate consumers
		// must analyze EACH state child, not infer one member represents all.
		writers := result.fields["$pointee"].fields["writers"]
		if writers == nil {
			return reject("native multiWriter field layout changed")
		}
		writers.elements = valueOf(a.program.Packages["io"].Types.Scope().Lookup("Writer").Type())
		writers.nilness, writers.nilType = 1, writers.types[0]
		writers.fields = map[string]*abstractValue{}
		for index, child := range state.fields {
			if index != "$constructor" {
				writers.fields[index] = child
			}
		}
	} else {
		// Reset initializes h to nonzero constants. Do not give those values a
		// zero-value fact merely because the constructor allocates a Digest.
		h := result.fields["$pointee"].fields["h"]
		if h == nil {
			return reject("native SHA Digest field layout changed")
		}
		h.elements = valueOf(types.Typ[types.Uint32])
	}
	result.fields[canonicalNativeWriterState] = state
	result.nativeWriterOwned = true
	return result, true
}

// This provides graph provenance only, not callback purity or short-write
// authority. The certificate consumer retains ordinary Write/StringWrite and
// concrete error analyses for every returned child and for the native hash.
func (a *effectAnalysis) canonicalNativeWriterDelegates(v *abstractValue, pos token.Pos) (class string, children []*abstractValue, ok bool) {
	if v == nil || v.unknown || !v.nativeWriterOwned || !v.origin.IsValid() || len(v.types) != 1 {
		return "", nil, false
	}
	state := v.fields[canonicalNativeWriterState]
	if state == nil || state.unknown || state.origin.IsValid() || len(state.functions) != 0 || state.elements != nil ||
		state.fields["$constructor"] == nil || state.fields["$constructor"].text != strconv.Itoa(int(v.origin)) {
		return "", nil, false
	}
	path, name, concrete := "io", "MultiWriter", "multiWriter"
	class = "multi"
	if state.text == "crypto/sha256.New" {
		path, name, concrete, class = "crypto/sha256", "New", "Digest", "hash"
	} else if state.text != "io.MultiWriter" {
		return "", nil, false
	}
	pkg := a.program.Packages[path]
	if pkg == nil {
		return "", nil, false
	}
	fn, found := pkg.Types.Scope().Lookup(name).(*types.Func)
	if !found || !a.canonicalNativeWriterFunction(fn, pos) || a.canonicalNativeWriterCall(fn, v.origin) == nil {
		return "", nil, false
	}
	concretePackage := pkg
	if class == "hash" {
		if len(state.fields) != 1 || !a.canonicalNativeWriterSource(canonicalSHA256Backend, pos) ||
			!a.canonicalNativeWriterSource("crypto/internal/boring", pos) {
			return "", nil, false
		}
		concretePackage = a.program.Packages[canonicalSHA256Backend]
	}
	object := concretePackage.Types.Scope().Lookup(concrete)
	if object == nil || !types.Identical(v.types[0], types.NewPointer(object.Type())) || v.fields["$pointee"] == nil {
		return "", nil, false
	}
	if class == "multi" {
		writers := v.fields["$pointee"].fields["writers"]
		if writers == nil || writers.unknown || len(writers.fields) != len(state.fields)-1 {
			return "", nil, false
		}
		for index := 0; index < len(state.fields)-1; index++ {
			key := strconv.Itoa(index)
			child := state.fields[key]
			if child == nil || child.unknown || len(child.types) == 0 || writers.fields[key] != child {
				return "", nil, false
			}
			children = append(children, child)
		}
	}
	return class, children, true
}

func (a *effectAnalysis) canonicalNativeWriterOperation(fn *types.Func, receiver *abstractValue, args []*abstractValue, pos token.Pos) (*abstractValue, bool) {
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != "io" {
		return nil, false
	}
	id := functionID(fn)
	if id != "io.*multiWriter.Write" && id != "io.*multiWriter.WriteString" {
		return nil, false
	}
	if receiver == nil || !receiver.nativeWriterOwned {
		return nil, false
	}
	result := modeledResults(fn)
	class, children, owned := a.canonicalNativeWriterDelegates(receiver, pos)
	if !a.canonicalNativeWriterFunction(fn, pos) || !owned || class != "multi" || len(args) != 1 || args[0] == nil {
		a.report("unresolved-effect", pos, "native multiWriter operation has unproven source/receiver/delegate binding")
		return result, true
	}
	state := receiver.fields[canonicalNativeWriterState]
	if a.compressionActive == nil {
		a.compressionActive = map[*abstractValue]bool{}
	}
	if a.compressionActive[state] {
		a.report("unresolved-effect", pos, "recursive native multiWriter callback mutation")
		return result, true
	}
	a.compressionActive[state] = true
	defer delete(a.compressionActive, state)
	ioPackage := a.program.Packages["io"]
	writerInterface := ioPackage.Types.Scope().Lookup("Writer").Type().Underlying().(*types.Interface)
	stringInterface := ioPackage.Types.Scope().Lookup("StringWriter").Type().Underlying().(*types.Interface)
	var errors *abstractValue
	for _, child := range children {
		for _, typ := range child.types {
			if typ == nil || !types.Implements(typ, writerInterface) {
				a.report("unresolved-effect", pos, "native multiWriter has unproven concrete Writer callback")
				continue
			}
			if _, dynamic := typ.Underlying().(*types.Interface); dynamic {
				a.report("unresolved-effect", pos, "native multiWriter has unresolved dynamic Writer callback")
				continue
			}
			copy := *child
			copy.types = []types.Type{typ}
			name, argument := "Write", args[0]
			if fn.Name() == "WriteString" {
				if types.Implements(typ, stringInterface) {
					name = "WriteString"
				} else {
					argument = valueOf(types.NewSlice(types.Typ[types.Byte]))
					argument.elements = valueOf(types.Typ[types.Byte])
				}
			}
			// Invoke the real callback for EACH member. Joining returned errors
			// is conservative; never join the member receiver graphs themselves.
			errors = join(errors, a.compressionInvokeWriter(&copy, name, argument, pos))
		}
	}
	if _, _, stillOwned := a.canonicalNativeWriterDelegates(receiver, pos); !stillOwned {
		a.report("unresolved-effect", pos, "native multiWriter delegate binding changed during callbacks")
	}
	errors = join(errors, a.canonicalNativeShortWriteError(pos))
	result.fields["1"] = join(result.fields["1"], errors)
	return result, true
}

func (a *effectAnalysis) canonicalNativeShortWriteError(pos token.Pos) *abstractValue {
	unknown := func() *abstractValue {
		a.report("unresolved-effect", pos, "native multiWriter short-write initializer has unproven source binding")
		return valueOf(types.Universe.Lookup("error").Type())
	}
	ioPackage, errorsPackage := a.program.Packages["io"], a.program.Packages["errors"]
	if ioPackage == nil || errorsPackage == nil || !a.canonicalNativeWriterSource("io", pos) {
		return unknown()
	}
	sentinel, ok := ioPackage.Types.Scope().Lookup("ErrShortWrite").(*types.Var)
	constructor, known := errorsPackage.Types.Scope().Lookup("New").(*types.Func)
	if !ok || !known || !a.canonicalNativeWriterFunction(constructor, pos) {
		return unknown()
	}
	for _, initializer := range a.program.canonicalSources["io"].initializers {
		if len(initializer.variables) != 1 || initializer.variables[0] != sentinel {
			continue
		}
		call, ok := initializer.expression.(*ast.CallExpr)
		if !ok || call.Ellipsis.IsValid() || len(call.Args) != 1 || calledObject(ioPackage.Info, call.Fun) != constructor {
			return unknown()
		}
		value := a.eval(ioPackage, initializer.expression, environment{})
		if value == nil || value.unknown {
			return unknown()
		}
		if a.sourceObjectMutation(sentinel) {
			a.report("unresolved-effect", pos, "native multiWriter exported short-write error has unproven mutable provenance")
			value = join(value, a.resolve(sentinel))
		}
		return value
	}
	return unknown()
}

func (a *effectAnalysis) canonicalNativeWriterFunction(fn *types.Func, pos token.Pos) bool {
	if fn == nil || fn.Pkg() == nil || !a.canonicalNativeWriterSource(fn.Pkg().Path(), pos) {
		return false
	}
	fn = fn.Origin()
	pkg := a.program.Packages[fn.Pkg().Path()]
	decl := a.program.Functions[functionID(fn)]
	bound, exists := a.program.canonicalSources[fn.Pkg().Path()].native.functions[functionID(fn)]
	signature, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	if receiver := signature.Recv(); receiver != nil {
		selection := types.NewMethodSet(receiver.Type()).Lookup(nil, fn.Name())
		if selection == nil || len(selection.Index()) != 1 || selection.Obj() != fn {
			return false
		}
	} else if pkg.Types.Scope().Lookup(fn.Name()) != fn {
		return false
	}
	return exists && decl != nil && decl.declaration != nil && decl.declaration.Body != nil &&
		decl.pkg == pkg && decl.object == fn && fn.Pkg() == pkg.Types &&
		pkg.Info.Defs[decl.declaration.Name] == fn &&
		bound.object == fn && bound.declaration == decl.declaration
}

func (a *effectAnalysis) canonicalNativeWriterCall(fn *types.Func, pos token.Pos) *ast.CallExpr {
	if !pos.IsValid() {
		return nil
	}
	for path, pkg := range a.program.Packages {
		if pkg.Metadata.Standard {
			continue
		}
		for _, file := range pkg.Files {
			if pos < file.Pos() || pos > file.End() {
				continue
			}
			if !a.canonicalScalarSourceBinding(path) {
				return nil
			}
			var call *ast.CallExpr
			ast.Inspect(file, func(node ast.Node) bool {
				if candidate, ok := node.(*ast.CallExpr); ok && candidate.Pos() == pos {
					if calledObject(pkg.Info, candidate.Fun) == fn.Origin() && !candidate.Ellipsis.IsValid() {
						call = candidate
					}
					return false
				}
				return call == nil
			})
			return call
		}
	}
	return nil
}

func (a *effectAnalysis) canonicalNativeWriterLocalCall(fn *types.Func, pos token.Pos) bool {
	if !pos.IsValid() {
		return false
	}
	for path, pkg := range a.program.Packages {
		if pkg.Metadata.Standard || a.program.canonicalSources[path] == nil {
			continue
		}
		for _, file := range pkg.Files {
			if pos < file.Pos() || pos > file.End() {
				continue
			}
			var body *ast.BlockStmt
			for _, declaration := range file.Decls {
				if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Body != nil && pos >= fn.Body.Pos() && pos <= fn.Body.End() {
					body = fn.Body
					break
				}
			}
			if body == nil {
				return false
			}
			local := false
			closure := false
			ast.Inspect(body, func(node ast.Node) bool {
				if literal, ok := node.(*ast.FuncLit); ok && pos >= literal.Pos() && pos <= literal.End() {
					closure = true
					return false
				}
				if call, ok := node.(*ast.CallExpr); ok && call.Pos() == pos {
					local = !call.Ellipsis.IsValid() && functionID(calledObject(pkg.Info, call.Fun)) == functionID(fn)
					return false
				}
				return !local
			})
			return local && !closure
		}
	}
	return false
}

func (a *effectAnalysis) canonicalNativeWriterSource(path string, pos token.Pos) bool {
	snapshot := a.program.canonicalSources[path]
	// Declaration authority includes the captured AST and type-info bindings;
	// a native name or startup byte pin alone cannot qualify a mutated program.
	if snapshot == nil || snapshot.astDigest == "" || !a.canonicalSourceBinding(path) {
		return false
	}
	if memorySummarySources[path] != "" && !a.checkMemorySource(path, pos) {
		return false
	}
	for name, digest := range snapshot.native.digests {
		data, err := os.ReadFile(filepath.Join(snapshot.native.metadata.Dir, name))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != digest {
			return false
		}
	}
	digest, err := compressionWholeGoDigest(snapshot.native.metadata)
	return err == nil && digest == snapshot.wholeSource
}
