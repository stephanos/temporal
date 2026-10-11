package architecture

import (
	"go/types"
	"slices"
	"strings"
	"testing"
)

const zeroConstructorFixture = `package record
type Inner struct{ Callback func() }
type Outer struct{ Inner Inner; Other int }
func (inner *Inner) Set(callback func()){inner.Callback=callback}
func clean(){}
func dirty(){go clean()}
`

func TestEffectZeroInitializationKeepsConstructorFieldBindings(t *testing.T) {
	for _, source := range []struct{ name, body string }{
		{"local", "var outer Outer; outer.Inner.Set(callback); return &outer"},
		{"omitted-literal-field", "outer := Outer{Other:1}; outer.Inner.Set(callback); return &outer"},
		{"alias", "var outer Outer; alias := &outer.Inner; alias.Set(callback); return &outer"},
		{"receiver-copy", "var outer Outer; outer.Inner.Set(callback); copied := Outer(outer); return &copied"},
	} {
		t.Run(source.name, func(t *testing.T) {
			program := effectResolverProgram(t, zeroConstructorFixture+"func Constructor(callback func()) *Outer {"+source.body+"}\n")
			a := effectResolverAnalysis(program)
			constructor := program.Functions["example.invalid/resolver/record.Constructor"]
			dirty := program.Functions["example.invalid/resolver/record.dirty"]
			callback := &abstractValue{types: []types.Type{dirty.object.Type()}, functions: []*boundFunction{{fn: dirty, pkg: dirty.pkg}}}
			result := a.call(&boundFunction{fn: constructor, pkg: constructor.pkg}, []*abstractValue{callback}, constructor.declaration.Pos())
			if result == nil || result.fields["$pointee"] == nil {
				t.Fatalf("constructor lost owned object: %#v", result)
			}
			outer := result.fields["$pointee"]
			inner := outer.fields["Inner"]
			if inner == nil || !slices.Contains(effectResolverFunctionIDs(inner.fields["Callback"]), functionID(dirty.object)) {
				t.Fatalf("constructor lost initialized child callback: outer=%#v inner=%#v", outer, inner)
			}
			a.invoke(inner.fields["Callback"], nil, constructor.declaration.Pos())
			found := false
			for _, finding := range a.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start")
			}
			if !found {
				t.Fatal("initialized callback lost its dirty invocation effects")
			}
		})
	}
}

func TestEffectZeroInitializationPreservesExplicitUnknownLiteralField(t *testing.T) {
	program := effectResolverProgram(t, zeroConstructorFixture+"func Constructor(inner Inner) *Outer{return &Outer{Inner:inner}}\n")
	a := effectResolverAnalysis(program)
	constructor := program.Functions["example.invalid/resolver/record.Constructor"]
	parameter := constructor.object.Type().(*types.Signature).Params().At(0).Type()
	unknown := valueOf(parameter)
	result := a.call(&boundFunction{fn: constructor, pkg: constructor.pkg}, []*abstractValue{unknown}, constructor.declaration.Pos())
	outer := result.fields["$pointee"]
	if outer.fields["Inner"] != unknown || len(outer.fields["Inner"].fields) != 0 {
		t.Fatalf("explicit unknown literal field gained omitted-field zero proof: %#v", outer)
	}
}

func TestEffectZeroInitializationRetainsMixedAliasCallbacks(t *testing.T) {
	program := effectResolverProgram(t, zeroConstructorFixture+`func Constructor() *Outer {
var outer Outer
alias := &outer.Inner
alias.Set(clean)
outer.Inner.Set(dirty)
return &outer
}`)
	a := effectResolverAnalysis(program)
	constructor := program.Functions["example.invalid/resolver/record.Constructor"]
	result := a.call(&boundFunction{fn: constructor, pkg: constructor.pkg}, nil, constructor.declaration.Pos())
	outer := result.fields["$pointee"]
	inner := outer.fields["Inner"]
	if inner == nil {
		t.Fatal("mixed alias initialization lost stable child location")
	}
	got := effectResolverFunctionIDs(inner.fields["Callback"])
	for _, name := range []string{"clean", "dirty"} {
		if !slices.Contains(got, "example.invalid/resolver/record."+name) {
			t.Fatalf("alias join discarded %s: %v", name, got)
		}
	}
}

func TestEffectZeroInitializationDoesNotInitializeUnknownParameters(t *testing.T) {
	program := effectResolverProgram(t, zeroConstructorFixture+"func Constructor(outer Outer) *Outer{return &outer}\n")
	a := effectResolverAnalysis(program)
	constructor := program.Functions["example.invalid/resolver/record.Constructor"]
	parameter := constructor.object.Type().(*types.Signature).Params().At(0).Type()
	result := a.call(&boundFunction{fn: constructor, pkg: constructor.pkg}, []*abstractValue{valueOf(parameter)}, constructor.declaration.Pos())
	outer := result.fields["$pointee"]
	if len(outer.fields) != 0 {
		t.Fatalf("unknown parameter gained zero-initialized child fields: %#v", outer)
	}
}

func TestEffectZeroInitializationDoesNotTreatLocalInterfaceAsCallbackFree(t *testing.T) {
	program := effectResolverProgram(t, "package record\nfunc Constructor() any{var value any;return value}\n")
	a := effectResolverAnalysis(program)
	constructor := program.Functions["example.invalid/resolver/record.Constructor"]
	result := a.call(&boundFunction{fn: constructor, pkg: constructor.pkg}, nil, constructor.declaration.Pos())
	value := result
	if value.builtin {
		t.Fatal("local interface acquired ambiguous callback-free builtin authority")
	}
	a.jsonCallbacks(value, true, constructor.declaration.Pos())
	if len(a.findings) == 0 {
		t.Fatal("static local interface was treated as distinct known-nil proof")
	}
}

func TestEffectZeroInitializationKeepsNilPointerDecodeCallbacks(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Dirty struct{}
func (*Dirty) UnmarshalJSON([]byte)error{go effect();return nil}
func effect(){}
func Constructor()*Dirty{var pointer *Dirty;return pointer}
`)
	a := effectResolverAnalysis(program)
	constructor := program.Functions["example.invalid/resolver/record.Constructor"]
	zeroPointer := a.call(&boundFunction{fn: constructor, pkg: constructor.pkg}, nil, constructor.declaration.Pos())
	pointerType := zeroPointer.types[0]
	destination := addressValue(zeroPointer, types.NewPointer(pointerType), constructor.declaration.Pos())
	a.jsonCallbacks(destination, true, constructor.declaration.Pos())
	found := false
	for _, finding := range a.findings {
		found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "Dirty.UnmarshalJSON") && strings.Contains(finding.Detail, "goroutine start")
	}
	if !found {
		t.Fatalf("zero pointer decode hid allocatable dirty callback: %v", a.findings)
	}
}
