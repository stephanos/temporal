package architecture

import (
	"go/token"
	"go/types"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

type jsonWalkContext struct {
	typ         types.Type
	addressable bool
	key         bool
	projected   bool
}

func (a *effectAnalysis) jsonDecodeCallbacks(destination *abstractValue, pos token.Pos) *abstractValue {
	if jsonNilEmptyInterfaceSlot(destination) {
		a.decoded(destination)
		return nil
	}
	callbacks := a.jsonCallbacks(destination, true, pos)
	a.decoded(destination)
	return callbacks
}

func jsonNilEmptyInterfaceSlot(destination *abstractValue) bool {
	if destination == nil || destination.unknown || destination.builtin || len(destination.types) != 1 ||
		destination.nilness != 1 || !types.Identical(destination.nilType, destination.types[0]) {
		return false
	}
	pointer, ok := destination.types[0].Underlying().(*types.Pointer)
	if !ok {
		return false
	}
	iface, ok := pointer.Elem().Underlying().(*types.Interface)
	if !ok || iface.NumMethods() != 0 {
		return false
	}
	slot := destination.fields["$pointee"]
	return slot != nil && !slot.unknown && !slot.builtin && len(slot.types) == 1 &&
		slot.nilness == -1 && types.Identical(slot.nilType, pointer.Elem()) &&
		types.Identical(slot.types[0], pointer.Elem())
}

func (a *effectAnalysis) jsonCallbacks(v *abstractValue, decoding bool, pos token.Pos) *abstractValue {
	return a.jsonWalkCallbacks(v, decoding, false, false, false, false, pos, map[jsonWalkContext]bool{})
}

func (a *effectAnalysis) jsonWalkCallbacks(v *abstractValue, decoding, addressable, key, unordered, projected bool, pos token.Pos, active map[jsonWalkContext]bool) *abstractValue {
	if v == nil || len(v.types) == 0 {
		a.report("unresolved-effect", pos, "unknown JSON callback receiver")
		return nil
	}
	if v.unknown {
		a.report("unresolved-effect", pos, "unknown dynamic JSON callback receiver")
	}
	concrete := false
	for _, typ := range v.types {
		if typ != nil {
			if _, dynamic := typ.Underlying().(*types.Interface); !dynamic {
				concrete = true
			}
		}
	}
	if !concrete {
		a.report("unresolved-effect", pos, "unknown interface-only JSON callback receiver")
	}
	var errors *abstractValue
	for _, typ := range v.types {
		if typ == nil {
			a.report("unresolved-effect", pos, "unknown JSON receiver type")
			continue
		}
		if _, ok := typ.Underlying().(*types.Interface); ok {
			continue
		}
		context := jsonWalkContext{typ: typ, addressable: addressable, key: key, projected: projected}
		if active[context] {
			a.report("unresolved-effect", pos, "recursive JSON callback binding")
			continue
		}
		active[context] = true
		receiver := *v
		receiver.types = []types.Type{typ}
		methodReceiver := &receiver
		methodType := typ
		if addressable {
			if _, pointer := typ.Underlying().(*types.Pointer); !pointer {
				methodType = types.NewPointer(typ)
				methodReceiver = addressValue(&receiver, methodType, pos)
			}
		}
		names := []string{"MarshalJSONTo", "MarshalJSON", "AppendText", "MarshalText"}
		if decoding {
			names = []string{"UnmarshalJSONFrom", "UnmarshalJSON", "UnmarshalText"}
		}
		if key {
			names = []string{"AppendText", "MarshalText"}
			if decoding {
				names = []string{"UnmarshalText"}
			}
		}
		stream, matched := false, false
		for _, name := range names {
			method := types.NewMethodSet(methodType).Lookup(nil, name)
			if method == nil {
				continue
			}
			fn := method.Obj().(*types.Func)
			if !a.jsonCallbackSignature(fn) {
				continue
			}
			isStream := name == "MarshalJSONTo" || name == "UnmarshalJSONFrom"
			selectedReceiver := a.jsonSelectedReceiver(method, methodReceiver, pos)
			if decoding && projected && !a.jsonNativeMemoryCallback(fn, selectedReceiver) && !jsonScalarReceiver(selectedReceiver) {
				a.report("unresolved-effect", pos, "projected JSON decoder receiver has unproven mutation location "+functionID(fn))
			}
			returned := a.jsonInvokeCallback(fn, selectedReceiver, unordered, pos)
			errors = join(errors, returned)
			matched = true
			if isStream {
				stream = !a.jsonNativeMemoryCallback(fn, selectedReceiver)
			} else {
				break
			}
		}
		if !key && (!matched || stream) {
			switch underlying := typ.Underlying().(type) {
			case *types.Pointer:
				child := receiver.fields["$pointee"]
				childProjected := projected
				if child == nil {
					copy := receiver
					copy.types = []types.Type{underlying.Elem()}
					child = &copy
					childProjected = true
				}
				errors = join(errors, a.jsonWalkCallbacks(child, decoding, jsonAddressable(underlying.Elem(), true, decoding), key, unordered, childProjected, pos, active))
			case *types.Slice:
				child, childProjected := jsonContainerElement(receiver.elements, underlying.Elem())
				errors = join(errors, a.jsonWalkCallbacks(child, decoding, jsonAddressable(underlying.Elem(), true, decoding), false, unordered, projected || childProjected, pos, active))
			case *types.Array:
				child, childProjected := jsonContainerElement(receiver.elements, underlying.Elem())
				errors = join(errors, a.jsonWalkCallbacks(child, decoding, jsonAddressable(underlying.Elem(), addressable, decoding), false, unordered, projected || childProjected, pos, active))
			case *types.Map:
				errors = join(errors, a.jsonWalkCallbacks(valueOf(underlying.Key()), decoding, decoding, true, true, true, pos, active))
				child, childProjected := jsonContainerElement(receiver.elements, underlying.Elem())
				errors = join(errors, a.jsonWalkCallbacks(child, decoding, decoding, false, true, projected || childProjected, pos, active))
			case *types.Struct:
				for index := 0; index < underlying.NumFields(); index++ {
					field := underlying.Field(index)
					tag, found := reflect.StructTag(underlying.Tag(index)).Lookup("json")
					if tag == "-" || (!field.Exported() && !field.Embedded()) {
						continue
					}
					if field.Embedded() && !a.jsonFlatEmbedLayout(underlying) || (!found && strings.Contains(underlying.Tag(index), "json:")) {
						a.report("unresolved-effect", pos, "ambiguous JSON field selection "+field.Name())
					}
					options := strings.Split(tag, ",")
					for _, option := range options[1:] {
						if !slices.Contains([]string{"", "omitempty", "omitzero", "string"}, option) {
							a.report("unresolved-effect", pos, "unsupported JSON field option "+option)
						}
					}
					child := receiver.fields[field.Name()]
					childProjected := projected
					if child == nil {
						child = valueOf(field.Type())
						childProjected = true
					}
					if !decoding && slices.Contains(options[1:], "omitzero") {
						for _, childType := range child.types {
							if childType == nil {
								a.report("unresolved-effect", pos, "unknown JSON IsZero receiver type")
								continue
							}
							methodType := childType
							if _, pointer := childType.Underlying().(*types.Pointer); !pointer {
								methodType = types.NewPointer(childType)
							}
							if method := types.NewMethodSet(methodType).Lookup(nil, "IsZero"); method != nil {
								fn := method.Obj().(*types.Func)
								if a.jsonCallbackSignature(fn) {
									copy := *child
									copy.types = []types.Type{childType}
									receiver := &copy
									if _, pointer := childType.Underlying().(*types.Pointer); !pointer {
										receiver = addressValue(receiver, methodType, pos)
									}
									a.jsonInvokeCallback(fn, a.jsonSelectedReceiver(method, receiver, pos), unordered, pos)
								}
							}
						}
					}
					errors = join(errors, a.jsonWalkCallbacks(child, decoding, jsonAddressable(field.Type(), addressable, decoding), false, unordered, childProjected, pos, active))
				}
			}
		}
		delete(active, context)
	}
	return errors
}

func jsonContainerElement(child *abstractValue, typ types.Type) (*abstractValue, bool) {
	// Empty containers use a bottom element cell, not an unknown element type.
	// Recover only its static shape; represented state still requires analysis.
	if child == nil || isBottomValue(child) {
		return valueOf(typ), true
	}
	return child, false
}

func jsonScalarReceiver(receiver *abstractValue) bool {
	if receiver == nil || receiver.unknown || len(receiver.types) == 0 || len(receiver.functions) != 0 ||
		receiver.elements != nil || receiver.gzipOwned || receiver.nativeWriterOwned {
		return false
	}
	pointerReceiver := false
	for _, typ := range receiver.types {
		if typ == nil {
			return false
		}
		if pointer, ok := typ.Underlying().(*types.Pointer); ok {
			typ = pointer.Elem()
			pointerReceiver = true
			pointee := receiver.fields["$pointee"]
			if pointee == nil || len(pointee.types) == 0 {
				return false
			}
			for _, alternative := range pointee.types {
				if alternative == nil || !types.Identical(alternative, typ) {
					return false
				}
			}
		}
		basic, ok := typ.Underlying().(*types.Basic)
		if !ok || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) == 0 {
			return false
		}
	}
	if !pointerReceiver {
		return len(receiver.fields) == 0
	}
	pointee := receiver.fields["$pointee"]
	if len(receiver.fields) != 1 || pointee == nil || pointee.unknown || len(pointee.fields) != 0 ||
		len(pointee.functions) != 0 || pointee.elements != nil || pointee.gzipOwned || pointee.nativeWriterOwned {
		return false
	}
	return true
}

// A method selection's final index denotes the method, not another field.
// Project its field prefix before binding the declaration's receiver.
func (a *effectAnalysis) jsonSelectedReceiver(selection *types.Selection, outer *abstractValue, pos token.Pos) *abstractValue {
	fn := selection.Obj().(*types.Func)
	typ := selection.Recv()
	value := outer
	addressable := false
	dereference := func(element types.Type) {
		unknown := value == nil || value.unknown
		var pointee *abstractValue
		if value != nil {
			pointee = value.fields["$pointee"]
		}
		if pointee == nil {
			a.report("unresolved-effect", pos, "unknown JSON selected receiver pointee "+functionID(fn))
			pointee = valueOf(element)
			if value != nil {
				pointee.fields = value.fields
				pointee.elements = value.elements
			}
			unknown = true
		}
		copy := *pointee
		copy.unknown = copy.unknown || unknown
		value = &copy
		typ = element
		addressable = true
	}
	indices := selection.Index()
	for _, index := range indices[:len(indices)-1] {
		if pointer, ok := typ.Underlying().(*types.Pointer); ok {
			dereference(pointer.Elem())
		}
		structure, ok := typ.Underlying().(*types.Struct)
		if !ok || index < 0 || index >= structure.NumFields() {
			a.report("unresolved-effect", pos, "invalid JSON selected receiver path "+functionID(fn))
			return &abstractValue{types: []types.Type{fn.Type().(*types.Signature).Recv().Type()}, unknown: true}
		}
		field := structure.Field(index)
		var child *abstractValue
		if value != nil {
			child = value.fields[field.Name()]
		}
		if child == nil {
			a.report("unresolved-effect", pos, "unknown JSON selected receiver field "+field.Name())
			child = valueOf(field.Type())
			child.unknown = true
		}
		copy := *child
		copy.unknown = copy.unknown || value == nil || value.unknown
		for _, alternative := range child.types {
			if alternative == nil || !types.Identical(alternative, field.Type()) {
				a.report("unresolved-effect", pos, "incompatible JSON selected receiver field "+field.Name())
				copy.unknown = true
			}
		}
		if len(child.types) == 0 {
			copy.unknown = true
		}
		value, typ = &copy, field.Type()
	}
	expected := fn.Type().(*types.Signature).Recv().Type()
	if pointer, ok := typ.Underlying().(*types.Pointer); ok && types.Identical(pointer.Elem(), expected) {
		dereference(pointer.Elem())
	} else if pointer, ok := expected.Underlying().(*types.Pointer); ok && types.Identical(pointer.Elem(), typ) {
		if value == nil {
			value = valueOf(typ)
			value.unknown = true
		}
		if !addressable {
			a.report("unresolved-effect", pos, "unaddressable JSON selected receiver "+functionID(fn))
			copy := *value
			copy.unknown = true
			value = &copy
		}
		value = addressValue(value, expected, pos)
	}
	if value == nil || value.unknown || len(value.types) == 0 {
		a.report("unresolved-effect", pos, "unknown JSON selected receiver binding "+functionID(fn))
	}
	return value
}

// Admit only a finite, collision-free subset of the pinned field dominance
// rules. More complex embeddings continue through the conservative owner.
func (a *effectAnalysis) jsonFlatEmbedLayout(structure *types.Struct) bool {
	var names []string
	var addFields func(*types.Struct, bool) bool
	addFields = func(current *types.Struct, embedded bool) bool {
		for index := 0; index < current.NumFields(); index++ {
			field := current.Field(index)
			tag, found := reflect.StructTag(current.Tag(index)).Lookup("json")
			if tag == "-" || !field.Exported() && !field.Embedded() {
				continue
			}
			if !found && strings.Contains(current.Tag(index), "json:") {
				return false
			}
			if field.Embedded() {
				child, ok := field.Type().Underlying().(*types.Struct)
				if embedded || !field.Exported() || current.Tag(index) != "" || !ok {
					return false
				}
				for _, typ := range []types.Type{field.Type(), types.NewPointer(field.Type())} {
					methods := types.NewMethodSet(typ)
					for methodIndex := 0; methodIndex < methods.Len(); methodIndex++ {
						fn := methods.At(methodIndex).Obj().(*types.Func)
						if fn.Name() != "IsZero" && a.jsonCallbackSignature(fn) {
							return false
						}
					}
				}
				if !addFields(child, true) {
					return false
				}
				continue
			}
			options := strings.Split(tag, ",")
			name := options[0]
			if name == "" {
				name = field.Name()
			}
			for _, character := range name {
				if !unicode.IsLetter(character) && !unicode.IsDigit(character) && !strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", character) {
					return false
				}
			}
			for _, option := range options[1:] {
				if !slices.Contains([]string{"", "omitempty", "omitzero", "string"}, option) {
					return false
				}
			}
			for _, previous := range names {
				if strings.EqualFold(previous, name) {
					return false
				}
			}
			names = append(names, name)
		}
		return true
	}
	return addFields(structure, false)
}

func jsonAddressable(typ types.Type, addressable, decoding bool) bool {
	_, dynamic := typ.Underlying().(*types.Interface)
	return addressable && (decoding || !dynamic)
}

func (a *effectAnalysis) jsonCallbackSignature(fn *types.Func) bool {
	signature := fn.Type().(*types.Signature)
	errorType := types.Universe.Lookup("error").Type()
	bytesType := types.NewSlice(types.Typ[types.Byte])
	params, results := signature.Params(), signature.Results()
	if signature.Variadic() || signature.TypeParams().Len() != 0 {
		return false
	}
	switch fn.Name() {
	case "IsZero":
		return params.Len() == 0 && results.Len() == 1 && types.Identical(results.At(0).Type(), types.Typ[types.Bool])
	case "MarshalJSON", "MarshalText", "AppendText":
		parameter := params.Len() == 0
		if fn.Name() == "AppendText" {
			parameter = params.Len() == 1 && types.Identical(params.At(0).Type(), bytesType)
		}
		return parameter && results.Len() == 2 && types.Identical(results.At(0).Type(), bytesType) && types.Identical(results.At(1).Type(), errorType)
	case "UnmarshalJSON", "UnmarshalText":
		return params.Len() == 1 && types.Identical(params.At(0).Type(), bytesType) && results.Len() == 1 && types.Identical(results.At(0).Type(), errorType)
	case "MarshalJSONTo", "UnmarshalJSONFrom":
		coder := a.jsonStreamType(fn.Name() == "UnmarshalJSONFrom")
		return coder != nil && params.Len() == 1 && types.Identical(params.At(0).Type(), coder) && results.Len() == 1 && types.Identical(results.At(0).Type(), errorType)
	}
	return false
}

func (a *effectAnalysis) jsonStreamType(decoding bool) types.Type {
	pkg := a.program.Packages["encoding/json/jsontext"]
	if pkg == nil || pkg.Types == nil {
		return nil
	}
	name := "Encoder"
	if decoding {
		name = "Decoder"
	}
	object := pkg.Types.Scope().Lookup(name)
	if object == nil {
		return nil
	}
	return types.NewPointer(object.Type())
}

func (a *effectAnalysis) jsonInvokeCallback(fn *types.Func, receiver *abstractValue, unordered bool, pos token.Pos) *abstractValue {
	declaration := a.program.Functions[functionID(fn)]
	if declaration == nil || declaration.object.Origin() != fn.Origin() {
		a.report("unresolved-effect", pos, "unknown JSON callback declaration "+functionID(fn))
		return nil
	}
	if a.jsonNativeMemoryCallback(fn, receiver) && a.checkJSONSummarySource(fn, pos) {
		returned := modeledResults(fn)
		if fn.Type().(*types.Signature).Results().Len() == 2 {
			returned = returned.fields["1"]
		}
		return returned
	}
	if unordered {
		a.report("unresolved-effect", pos, "JSON map callback order "+functionID(fn))
	}
	var args []*abstractValue
	switch fn.Name() {
	case "AppendText", "UnmarshalJSON", "UnmarshalText":
		args = []*abstractValue{valueOf(types.NewSlice(types.Typ[types.Byte]))}
	case "MarshalJSONTo", "UnmarshalJSONFrom":
		stream := valueOf(a.jsonStreamType(fn.Name() == "UnmarshalJSONFrom"))
		stream.unknown = true
		args = []*abstractValue{stream}
		a.report("unresolved-effect", pos, "unsupported custom JSONtext stream context "+functionID(fn))
	}
	returned := a.call(&boundFunction{fn: declaration, pkg: declaration.pkg, receiver: receiver}, args, pos)
	if fn.Name() == "IsZero" {
		return nil
	}
	if fn.Type().(*types.Signature).Results().Len() == 2 {
		if returned != nil {
			returned = returned.fields["1"]
		}
	}
	a.jsonErrorCallbacks(returned, pos)
	return returned
}

func (a *effectAnalysis) jsonNativeMemoryCallback(fn *types.Func, receiver *abstractValue) bool {
	if a.checkedJSON == nil || receiver == nil || receiver.unknown || len(receiver.types) == 0 {
		return false
	}
	switch functionID(fn) {
	case "encoding/json.Number.MarshalJSONTo", "encoding/json.*Number.UnmarshalJSONFrom", "encoding/json/jsontext.Value.MarshalJSON", "encoding/json/jsontext.*Value.UnmarshalJSON":
	default:
		return false
	}
	imported := a.program.Packages[fn.Pkg().Path()]
	snapshot := a.program.jsonSources[fn.Pkg().Path()]
	if imported == nil || imported.Types != fn.Origin().Pkg() || snapshot == nil || snapshot.functions[functionID(fn)].object != fn.Origin() {
		return false
	}
	expected := fn.Type().(*types.Signature).Recv().Type()
	for _, typ := range receiver.types {
		if types.Identical(typ, expected) {
			continue
		}
		pointer, ok := typ.(*types.Pointer)
		if !ok || !types.Identical(pointer.Elem(), expected) {
			return false
		}
	}
	return true
}
