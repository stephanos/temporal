package architecture

import (
	"go/types"
	"strings"
	"testing"
)

func TestEffectRangeEmptyContainerRetainsStaticValueShape(t *testing.T) {
	for _, test := range []struct{ name, declaration, loop string }{
		{"slice-value", "var order []string", "for _, id := range order"},
		{"array-value", "var order [0]string", "for _, id := range order"},
		{"map-value", "var order map[uint64]string", "for _, id := range order"},
		{"map-key-control", "var order map[string]uint64", "for id := range order"},
	} {
		t.Run(test.name, func(t *testing.T) {
			program := stockSummaryProgram(t, "example.invalid/rangejsonshape", `package record
import "encoding/json"
func Root() {
 `+test.declaration+`
 type participant struct { Node string }
 var participants []participant
 `+test.loop+` { participants=append(participants, participant{Node:id}) }
 _,_=json.Marshal(participants)
}

`)
			if findings := program.Effects(); len(findings) != 0 {
				t.Fatalf("statically string range value lost its type before participant.Node encoding: %v", findings)
			}
		})
	}
}

func TestEffectRangeShapeRecoveryPreservesNonbottomState(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Scalar int
func (Scalar) MarshalJSON() ([]byte,error) { go sideEffect(); return nil,nil }
func sideEffect() {}
func Collect(values []Scalar) Scalar {
 for _, value := range values { return value }
 panic("empty")
}
`)
	root := program.Functions["example.invalid/resolver/record.Collect"]
	for _, test := range []struct {
		name    string
		builtin bool
	}{
		{"bottom", false}, {"builtin-container-bottom", true}, {"unknown", false},
		{"function", false}, {"field", false}, {"element", false}, {"origin", false}, {"nil-fact", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			values := zeroValue(root.object.Type().(*types.Signature).Params().At(0).Type())
			values.builtin = test.builtin
			child := values.elements
			switch test.name {
			case "unknown":
				child.unknown = true
			case "function":
				child.functions = []*boundFunction{{fn: root, pkg: root.pkg}}
			case "field":
				child.fields = map[string]*abstractValue{"State": valueOf(types.Typ[types.Int])}
			case "element":
				child.elements = valueOf(types.Typ[types.Int])
			case "origin":
				child.origin = root.declaration.Pos()
			case "nil-fact":
				child.nilness = -1
			}
			unchanged := bindingKeyInputSnapshot([]*abstractValue{values})
			analysis := effectResolverAnalysis(program)
			returned := analysis.call(&boundFunction{fn: root, pkg: root.pkg}, []*abstractValue{values}, root.declaration.Pos())
			if !unchanged() {
				t.Fatal("range shape recovery mutated the original element graph")
			}
			if returned == nil || returned.builtin {
				t.Fatalf("range invented builtin authority: %s", bindingKey([]*abstractValue{returned}))
			}
			bottom := test.name == "bottom" || test.name == "builtin-container-bottom"
			if (len(returned.types) != 0) != bottom {
				t.Fatalf("nonbottom state was retyped, or bottom lost its static type: %s", bindingKey([]*abstractValue{returned}))
			}
			if !bottom && bindingKey([]*abstractValue{returned}) != bindingKey([]*abstractValue{child}) {
				t.Fatal("range erased represented nonbottom facts")
			}
			analysis.jsonCallbacks(returned, false, root.declaration.Pos())
			host, unresolved := false, false
			for _, finding := range analysis.findings {
				host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Scalar.MarshalJSON -> goroutine start")
				unresolved = unresolved || finding.Category == "unresolved-effect"
			}
			if host != bottom || unresolved == bottom {
				t.Fatalf("range callback proof changed: host=%v unresolved=%v findings=%v", host, unresolved, analysis.findings)
			}
		})
	}
}
