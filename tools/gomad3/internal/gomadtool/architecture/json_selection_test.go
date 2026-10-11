package architecture

import (
	"go/types"
	"strings"
	"testing"
)

func TestJSONCallbacksPromotedReceiverPreservesCapturedState(t *testing.T) {
	for _, tc := range []struct {
		name, embedding, receiver string
		pointerRoot               bool
	}{
		{"value", "Leaf", "Leaf", false},
		{"addressed-value", "Leaf", "*Leaf", true},
		{"embedded-pointer", "*Leaf", "*Leaf", false},
		{"pointer-to-value", "*Leaf", "Leaf", false},
		{"multiple-pointer-path", "*Middle", "*Leaf", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access := "v.Callback()"
			if tc.receiver == "*Leaf" {
				access = "(*v).Callback()"
			}
			program := effectResolverProgram(t, "package record\ntype Leaf struct{Callback func()}\ntype Middle struct{*Leaf}\ntype Dirty struct{"+tc.embedding+"; Callback func()}\nfunc(v "+tc.receiver+")MarshalJSON()([]byte,error){"+access+";return nil,nil}\nfunc captured(){go sideEffect()}\nfunc decoy(){}\nfunc sideEffect(){}\nfunc Root(){}\n")
			root := program.Functions["example.invalid/resolver/record.Root"]
			captured := program.Functions["example.invalid/resolver/record.captured"]
			decoy := program.Functions["example.invalid/resolver/record.decoy"]
			leaf := valueOf(root.pkg.Types.Scope().Lookup("Leaf").Type())
			leaf.fields = map[string]*abstractValue{"Callback": {functions: []*boundFunction{{fn: captured, pkg: captured.pkg}}}}
			selected := leaf
			field := "Leaf"
			if strings.HasPrefix(tc.embedding, "*") {
				selected = addressValue(leaf, types.NewPointer(leaf.types[0]), root.declaration.Pos())
			}
			if tc.embedding == "*Middle" {
				middle := valueOf(root.pkg.Types.Scope().Lookup("Middle").Type())
				middle.fields = map[string]*abstractValue{"Leaf": selected}
				selected = addressValue(middle, types.NewPointer(middle.types[0]), root.declaration.Pos())
				field = "Middle"
			}
			outer := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
			outer.fields = map[string]*abstractValue{field: selected, "Callback": {functions: []*boundFunction{{fn: decoy, pkg: decoy.pkg}}}}
			if tc.pointerRoot {
				outer = addressValue(outer, types.NewPointer(outer.types[0]), root.declaration.Pos())
			}
			analysis := effectResolverAnalysis(program)
			analysis.jsonCallbacks(outer, false, root.declaration.Pos())
			found := false
			for _, finding := range analysis.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.captured -> goroutine start")
			}
			if !found {
				t.Fatalf("promoted receiver lost its captured callback: %v", analysis.findings)
			}
			if outer.fields[field] != selected || leaf.fields["Callback"].functions[0].fn != captured {
				t.Fatal("receiver projection mutated original field bindings")
			}
		})
	}
}

func TestJSONCallbacksPromotedReceiverShadowing(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Leaf struct{}\ntype Dirty struct{Leaf}\nfunc(Leaf)MarshalJSON()([]byte,error){go sideEffect();return nil,nil}\nfunc(Dirty)MarshalJSON()([]byte,error){return nil,nil}\nfunc sideEffect(){}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	analysis := effectResolverAnalysis(program)
	analysis.jsonCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type()), false, root.declaration.Pos())
	if len(analysis.findings) != 0 {
		t.Fatalf("outer codec failed to shadow the embedded codec: %v", analysis.findings)
	}
}

func TestJSONCallbacksPromotedReceiverOtherCallbackKinds(t *testing.T) {
	for _, tc := range []struct {
		name, method string
		decoding     bool
	}{
		{"marshal-text", "func(v *Leaf)MarshalText()([]byte,error){(*v).Callback();return nil,nil}", false},
		{"append-text", "func(v *Leaf)AppendText(b []byte)([]byte,error){(*v).Callback();return b,nil}", false},
		{"unmarshal-json", "func(v *Leaf)UnmarshalJSON([]byte)error{(*v).Callback();return nil}", true},
		{"unmarshal-text", "func(v *Leaf)UnmarshalText([]byte)error{(*v).Callback();return nil}", true},
		{"is-zero", "func(v *Leaf)IsZero()bool{(*v).Callback();return false}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, "package record\ntype Leaf struct{Callback func()}\ntype Dirty struct{*Leaf; Callback func()}\ntype Holder struct{Value Dirty `json:\"value,omitzero\"`}\n"+tc.method+"\nfunc captured(){go sideEffect()}\nfunc decoy(){}\nfunc sideEffect(){}\nfunc Root(){}\n")
			root := program.Functions["example.invalid/resolver/record.Root"]
			captured := program.Functions["example.invalid/resolver/record.captured"]
			decoy := program.Functions["example.invalid/resolver/record.decoy"]
			leaf := valueOf(root.pkg.Types.Scope().Lookup("Leaf").Type())
			leaf.fields = map[string]*abstractValue{"Callback": {functions: []*boundFunction{{fn: captured, pkg: captured.pkg}}}}
			outer := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
			outer.fields = map[string]*abstractValue{"Leaf": addressValue(leaf, types.NewPointer(leaf.types[0]), root.declaration.Pos()), "Callback": {functions: []*boundFunction{{fn: decoy, pkg: decoy.pkg}}}}
			if tc.name == "is-zero" {
				holder := valueOf(root.pkg.Types.Scope().Lookup("Holder").Type())
				holder.fields = map[string]*abstractValue{"Value": outer}
				outer = holder
			} else if tc.decoding {
				outer = addressValue(outer, types.NewPointer(outer.types[0]), root.declaration.Pos())
			}
			analysis := effectResolverAnalysis(program)
			analysis.jsonCallbacks(outer, tc.decoding, root.declaration.Pos())
			found := false
			for _, finding := range analysis.findings {
				found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.captured -> goroutine start")
			}
			if !found {
				t.Fatalf("promoted %s lost captured receiver state: %v", tc.name, analysis.findings)
			}
		})
	}
}

func TestJSONCallbacksPromotedUnknownBindingRemainsConservative(t *testing.T) {
	program := effectResolverProgram(t, "package record\ntype Leaf struct{}\ntype Dirty struct{Leaf}\nfunc(Leaf)MarshalJSON()([]byte,error){return nil,nil}\nfunc Root(){}\n")
	root := program.Functions["example.invalid/resolver/record.Root"]
	for _, unknown := range []bool{false, true} {
		analysis := effectResolverAnalysis(program)
		outer := valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
		if unknown {
			outer.fields = map[string]*abstractValue{"Leaf": {types: []types.Type{root.pkg.Types.Scope().Lookup("Leaf").Type()}, unknown: true}}
		}
		analysis.jsonCallbacks(outer, false, root.declaration.Pos())
		if len(analysis.findings) == 0 {
			t.Fatalf("missing/unknown promoted child unknown=%v was silently accepted", unknown)
		}
	}
}
