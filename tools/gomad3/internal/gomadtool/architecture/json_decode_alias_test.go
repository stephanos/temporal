package architecture

import (
	"go/types"
	"os"
	"strings"
	"testing"
)

func TestJSONDecodeEmptyContainerRetainsLaterCallbackMutation(t *testing.T) {
	const module = "example.invalid/jsondecodealias"
	program := stockSummaryProgram(t, module, `package record
import("encoding/json";"time")
type Dirty struct { F func() time.Time }
func (value *Dirty) UnmarshalJSON([]byte) error { value.F=time.Now;return nil }
func clean() time.Time { return time.Time{} }
func Root() {
 var items []Dirty
 _=json.Unmarshal([]byte("[{}]"), &items)
 items=append(items, Dirty{F:clean})
 _=items[0].F()
}
`)
	findings := program.Effects()
	for _, finding := range findings {
		if !strings.Contains(finding.Detail, module+"/record.Root ->") {
			continue
		}
		if finding.Category == "unresolved-effect" || finding.Category == "host-effect" && strings.Contains(finding.Detail, "time.Now") {
			return
		}
	}
	t.Fatalf("decode receiver mutation disappeared behind a later clean append: %v", findings)
}

func TestJSONProjectedDecoderCompositeReceiverRetainsCallbacks(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Dirty struct { F func() }
type Failure int
func (*Dirty) UnmarshalJSON([]byte) error { go sideEffect(); return Failure(1) }
func (Failure) Error() string { go sideEffect(); return "failure" }
func sideEffect() {}
type Slice []Dirty
type PointerSlice []*Dirty
type ArraySlice [][1]Dirty
type NestedSlice []struct { Value Dirty }
type MapSlice []map[string]Dirty
type Promoted struct { Dirty }
type PromotedSlice []Promoted
type Scalar int
func (*Scalar) UnmarshalJSON([]byte) error { return Failure(1) }
type ScalarSlice []Scalar
func Root() {}
`)
	root := program.Functions["example.invalid/resolver/record.Root"]
	for _, name := range []string{"Slice", "PointerSlice", "ArraySlice", "NestedSlice", "MapSlice", "PromotedSlice", "ScalarSlice"} {
		t.Run(name, func(t *testing.T) {
			value := zeroValue(root.pkg.Types.Scope().Lookup(name).Type())
			unchanged := bindingKeyInputSnapshot([]*abstractValue{value})
			analysis := effectResolverAnalysis(program)
			analysis.jsonCallbacks(value, true, root.declaration.Pos())
			host, callbackHost, projected := false, false, false
			for _, finding := range analysis.findings {
				host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Failure.Error -> goroutine start")
				callbackHost = callbackHost || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.*Dirty.UnmarshalJSON -> goroutine start")
				projected = projected || finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, "projected JSON decoder receiver")
			}
			if !host || callbackHost != (name != "ScalarSlice") || projected != (name != "ScalarSlice") {
				t.Fatalf("error host=%v callback host=%v projected=%v: %v", host, callbackHost, projected, analysis.findings)
			}
			if !unchanged() {
				t.Fatal("projected callback walk mutated original container graph")
			}
		})
	}
}

func TestJSONProjectedScalarDecoderPreservesOrdinaryJoinedState(t *testing.T) {
	source, err := os.ReadFile("../../../record/uint64_string.go")
	if err != nil {
		t.Fatal(err)
	}
	const module = "example.invalid/jsondecodescalar"
	program := stockSummaryProgram(t, module, string(source)+`
func Root() []Uint64String {
 var items []Uint64String
 _=json.Unmarshal([]byte("[\"7\"]"), &items)
 items=append(items, Uint64String(0))
 return items
}
`)
	root := program.Functions[module+"/record.Root"]
	analysis := effectResolverAnalysis(program)
	returned := analysis.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	if len(analysis.findings) != 0 {
		t.Fatalf("actual scalar decoder is unsupported: %v", analysis.findings)
	}
	if returned == nil || returned.elements == nil || returned.elements.builtin || len(returned.elements.functions) != 0 ||
		len(returned.elements.types) != 1 || !types.Identical(returned.elements.types[0], root.pkg.Types.Scope().Lookup("Uint64String").Type()) {
		t.Fatalf("decode+append invented callback-free authority or lost the actual scalar type: %s", bindingKey([]*abstractValue{returned}))
	}
}
