package architecture

import (
	"go/token"
	"go/types"
)

// JSON wrappers can inspect returned errors before the caller observes them.
// This JSON-only traversal preserves errorCallbacks' graph/call behavior while
// restricting each invocation to the actual error interface signature.
func (a *effectAnalysis) jsonErrorCallbacks(value *abstractValue, pos token.Pos) {
	a.jsonErrorGraph(value, pos, map[string]bool{})
}

func (a *effectAnalysis) jsonErrorGraph(value *abstractValue, pos token.Pos, seen map[string]bool) {
	if value == nil {
		a.report("unresolved-effect", pos, "unknown JSON error callback binding")
		return
	}
	key := bindingKey([]*abstractValue{value})
	if seen[key] {
		return
	}
	seen[key] = true
	if value.builtin {
		if value.elements != nil {
			a.jsonErrorGraph(value.elements, pos, seen)
		}
		return
	}
	if value.unknown || len(value.types) == 0 {
		a.report("unresolved-effect", pos, "unknown dynamic JSON error callback receiver")
	}
	errorType := types.Universe.Lookup("error").Type()
	for _, typ := range value.types {
		if typ == nil {
			a.report("unresolved-effect", pos, "missing JSON error callback type")
			continue
		}
		if basic, ok := typ.Underlying().(*types.Basic); ok && basic.Kind() == types.UntypedNil {
			continue
		}
		if _, ok := typ.Underlying().(*types.Interface); ok {
			a.report("unresolved-effect", pos, "unbound JSON error callback interface")
			continue
		}
		if !types.Implements(typ, errorType.Underlying().(*types.Interface)) {
			a.report("unresolved-effect", pos, "JSON error binding does not implement error")
			continue
		}
		receiver := *value
		receiver.types = []types.Type{typ}
		methods := types.NewMethodSet(typ)
		for index := 0; index < methods.Len(); index++ {
			method := methods.At(index).Obj().(*types.Func)
			signature := method.Type().(*types.Signature)
			if !jsonErrorSignature(method.Name(), signature) {
				continue
			}
			var arguments []*abstractValue
			if method.Name() == "Is" {
				// The concrete sentinel is not needed to accept a callback; its
				// unknown error interface must not manufacture a host-free target.
				arguments = []*abstractValue{valueOf(errorType)}
			}
			returned := a.implicit(&receiver, []string{method.Name()}, arguments, pos, map[types.Type]bool{})
			if method.Name() != "Unwrap" {
				continue
			}
			if jsonKnownNilError(returned) {
				continue
			}
			if types.Identical(signature.Results().At(0).Type(), types.NewSlice(errorType)) {
				returned = elementValue(returned)
			}
			a.jsonErrorGraph(returned, pos, seen)
		}
	}
}

func jsonKnownNilError(value *abstractValue) bool {
	if value == nil || value.unknown || len(value.types) == 0 {
		return false
	}
	for _, typ := range value.types {
		if typ == nil {
			return false
		}
		basic, ok := typ.Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.UntypedNil {
			return false
		}
	}
	return true
}

func jsonErrorSignature(name string, signature *types.Signature) bool {
	if signature.Variadic() || signature.Results().Len() != 1 {
		return false
	}
	errorType := types.Universe.Lookup("error").Type()
	result := signature.Results().At(0).Type()
	switch name {
	case "Error":
		return signature.Params().Len() == 0 && types.Identical(result, types.Typ[types.String])
	case "Is":
		return signature.Params().Len() == 1 && types.Identical(signature.Params().At(0).Type(), errorType) && types.Identical(result, types.Typ[types.Bool])
	case "Unwrap":
		return signature.Params().Len() == 0 && (types.Identical(result, errorType) || types.Identical(result, types.NewSlice(errorType)))
	default:
		return false
	}
}
