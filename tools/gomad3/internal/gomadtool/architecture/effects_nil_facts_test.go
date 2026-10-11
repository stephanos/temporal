package architecture

import (
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

// A missing fact is unknown, so these fixtures can observe meaningful RED
// before introducing the unused private fact field.
func nilFactForTest(value *abstractValue) int {
	if value == nil {
		return 0
	}
	field := reflect.ValueOf(value).Elem().FieldByName("nilness")
	if !field.IsValid() {
		return 0
	}
	return int(field.Int())
}

func TestEffectNilFactsGenuineZerosAndUnknowns(t *testing.T) {
	for _, typ := range []types.Type{
		types.NewPointer(types.Typ[types.Int]),
		types.NewInterfaceType(nil, nil),
		types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewSlice(types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
	} {
		if got := nilFactForTest(zeroValue(typ)); got != -1 {
			t.Fatalf("actual zero %s has nil fact %d, want nil(-1)", typ, got)
		}
		if got := nilFactForTest(valueOf(typ)); got != 0 {
			t.Fatalf("unknown parameter-like %s gained fact %d", typ, got)
		}
	}
	if got := nilFactForTest(zeroValue(types.Typ[types.Int])); got != 0 {
		t.Fatalf("nonnilable zero gained nil fact %d", got)
	}
}

func TestEffectNilFactsActualLanguageConstructors(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         int
	}{
		{"nil-literal", "func Root()*int{return nil}", -1},
		{"new", "func Root()*int{return new(int)}", 1},
		{"address", "func Root()*int{var value int;return &value}", 1},
		{"make-map", "func Root()map[string]int{return make(map[string]int)}", 1},
		{"make-slice", "func Root()[]int{return make([]int,0)}", 1},
		{"make-channel", "func Root()chan int{return make(chan int)}", 1},
		{"map-literal", "func Root()map[string]int{return map[string]int{}}", 1},
		{"slice-literal", "func Root()[]int{return []int{}}", 1},
		{"append-nil", "func Root()[]int{var value []int;return append(value,1)}", 0},
		{"variadic-nonempty", "func Keep(values ...int)[]int{return values};func Root()[]int{return Keep(1)}", 1},
		{"variadic-empty", "func Keep(values ...int)[]int{return values};func Root()[]int{return Keep()}", -1},
		{"variadic-expanded", "func Keep(values ...int)[]int{return values};func Root()[]int{return Keep([]int{1}...)}", 1},
		{"variadic-expanded-nil", "func Keep(values ...int)[]int{return values};func Root()[]int{var value []int;return Keep(value...)}", -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\n"+test.source+"\n")
			a := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			value := a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			if got := nilFactForTest(value); got != test.want {
				t.Fatalf("%s nil fact %d, want %d", test.name, got, test.want)
			}
			if value.nilness == 0 && value.nilType != nil {
				t.Fatal("unknown result retained nil type provenance")
			}
		})
	}
}

func TestEffectNilFactsJoinCloneAndBindingKeys(t *testing.T) {
	typ := types.NewPointer(types.Typ[types.Int])
	nilPointer := zeroValue(typ)
	nonnilPointer := addressValue(zeroValue(types.Typ[types.Int]), typ, token.NoPos)
	unknownPointer := valueOf(typ)
	for _, pair := range [][2]*abstractValue{
		{nilPointer, nonnilPointer}, {nonnilPointer, nilPointer},
		{nilPointer, unknownPointer}, {unknownPointer, nilPointer},
	} {
		if got := nilFactForTest(join(cloneValue(pair[0]), cloneValue(pair[1]))); got != 0 {
			t.Fatalf("mixed nil facts retained certainty %d", got)
		}
	}
	if got := nilFactForTest(join(cloneValue(nilPointer), cloneValue(nilPointer))); got != -1 {
		t.Fatalf("agreed nil facts lost: %d", got)
	}
	if nilFactForTest(copyValue(nilPointer, typ)) != -1 || nilFactForTest(cloneValue(nilPointer)) != -1 {
		t.Fatal("scalar fact was not preserved by value/graph copy")
	}
	if bindingKey([]*abstractValue{nilPointer}) == bindingKey([]*abstractValue{unknownPointer}) {
		t.Fatal("nil fact omitted from recursive binding/convergence key")
	}
	if got := nilFactForTest(join(cloneValue(nilPointer), zeroValue(types.NewInterfaceType(nil, nil)))); got != 0 {
		t.Fatalf("pointer payload and empty-interface facts were conflated: %d", got)
	}
	if mixed := join(cloneValue(nilPointer), unknownPointer); mixed.nilType != nil {
		t.Fatal("unknown joined fact retained type provenance")
	}
	if copied := copyValue(nilPointer, types.NewInterfaceType(nil, nil)); copied.nilness != 0 || copied.nilType != nil {
		t.Fatal("pointer payload gained an interface-slot nil fact")
	}
	debris := &abstractValue{nilType: typ}
	if copyValue(debris, typ).nilType != nil || cloneValue(debris).nilType != nil {
		t.Fatal("unknown copied fact retained representational debris")
	}
	if bindingKey([]*abstractValue{debris}) != bindingKey([]*abstractValue{{}}) {
		t.Fatal("unknown provenance debris changed convergence key")
	}
}

func TestEffectNilFactsFallbackDerefAndRetypingStayUnknown(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Named *int
func Deref(value **int)*int{return *value}
func Retype(value *int)Named{return Named(value)}
func Recursive(value *int)*int{return Recursive(value)}
`)
	for _, name := range []string{"Deref", "Retype", "Recursive"} {
		a := effectResolverAnalysis(program)
		fn := program.Functions["example.invalid/resolver/record."+name]
		parameter := fn.object.Type().(*types.Signature).Params().At(0).Type()
		value := valueOf(parameter)
		if name == "Deref" {
			pointer := parameter.Underlying().(*types.Pointer)
			value = addressValue(zeroValue(pointer.Elem()), parameter, token.NoPos)
			delete(value.fields, "$pointee")
		} else if name == "Retype" {
			value = addressValue(zeroValue(types.Typ[types.Int]), parameter, token.NoPos)
		}
		result := a.call(&boundFunction{fn: fn, pkg: fn.pkg}, []*abstractValue{value}, fn.declaration.Pos())
		if got := nilFactForTest(result); got != 0 {
			t.Fatalf("%s fallback/retyping gained fact %d", name, got)
		}
	}
}

func TestEffectNilFactsWeakAliasMutationWidens(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Holder struct{ Value *int }
func Root()*int{
 var holder Holder
 alias:=&holder
 alias.Value=new(int)
 return holder.Value
}
`)
	a := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	value := a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	if value.nilness != 0 || value.nilType != nil {
		t.Fatal("weak alias assignment retained nil certainty")
	}
}

func TestEffectNilFactsRecursiveSnapshotDropsFactsNotProvenance(t *testing.T) {
	typ := types.NewPointer(types.Typ[types.Int])
	callback := &boundFunction{capture: environment{}}
	value := addressValue(zeroValue(types.Typ[types.Int]), typ, token.NoPos)
	value.fields["alias"] = value
	value.elements = zeroValue(typ)
	callback.receiver = value.elements
	value.functions = []*boundFunction{callback}
	copy := cloneValue(value)
	forgetNilFacts(copy)
	if copy.nilness != 0 || copy.nilType != nil || copy.elements.nilness != 0 || copy.elements.nilType != nil || copy.functions[0].receiver.nilness != 0 {
		t.Fatal("provisional recursive graph retained stale nil facts")
	}
	if copy.fields["alias"] != copy || len(copy.functions) != 1 || copy.functions[0].receiver != copy.elements {
		t.Fatal("discarding nil facts destroyed alias/callback provenance")
	}
	if value.nilness != 1 || value.elements.nilness != -1 {
		t.Fatal("discarding cloned provisional facts mutated original graph")
	}
}
