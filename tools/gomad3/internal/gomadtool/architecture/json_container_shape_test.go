package architecture

import (
	"strings"
	"testing"
)

func TestJSONCallbacksEmptyContainerStaticShape(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Slice []int
type Map map[string]int
type EmptyArray [0]int
type Nested struct { Values []map[string][]int }
type Dirty int
type Failure int
func (Dirty) MarshalJSON() ([]byte,error) { return nil, Failure(1) }
func (*Dirty) UnmarshalJSON([]byte) error { return Failure(1) }
func (Failure) Error() string { go sideEffect(); return "failure" }
type DirtySlice []Dirty
type DirtyMap map[string]Dirty
type InterfaceSlice []any
type InterfaceMap map[string]any
func sideEffect() {}
func Root() {}
`)
	root := program.Functions["example.invalid/resolver/record.Root"]
	for _, test := range []struct {
		name, typ, represented, taint string
		unknown, host, unresolved     bool
	}{
		{name: "slice", typ: "Slice"},
		{name: "map", typ: "Map"},
		{name: "zero-length-array", typ: "EmptyArray"},
		{name: "nested-struct", typ: "Nested"},
		{name: "concrete-element-callback-error", typ: "DirtySlice", host: true},
		{name: "unordered-concrete-element-callback", typ: "DirtyMap", host: true, unresolved: true},
		{name: "interface-element", typ: "InterfaceSlice", unresolved: true},
		{name: "interface-map-element", typ: "InterfaceMap", unresolved: true},
		{name: "unknown-element", typ: "Slice", unknown: true, unresolved: true},
		{name: "typeless-field-state", typ: "Slice", taint: "field", unresolved: true},
		{name: "typeless-function-state", typ: "Slice", taint: "function", unresolved: true},
		{name: "typeless-element-state", typ: "Slice", taint: "element", unresolved: true},
		{name: "represented-interface-element", typ: "InterfaceSlice", represented: "Dirty", host: true},
	} {
		for _, decoding := range []bool{false, true} {
			name := test.name + map[bool]string{false: "/marshal", true: "/unmarshal"}[decoding]
			t.Run(name, func(t *testing.T) {
				value := zeroValue(root.pkg.Types.Scope().Lookup(test.typ).Type())
				if test.unknown {
					value.elements.unknown = true
				}
				switch test.taint {
				case "field":
					value.elements.fields = map[string]*abstractValue{"State": valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())}
				case "function":
					value.elements.functions = []*boundFunction{{fn: root, pkg: root.pkg}}
				case "element":
					value.elements.elements = valueOf(root.pkg.Types.Scope().Lookup("Dirty").Type())
				}
				if test.represented != "" {
					value.elements = valueOf(root.pkg.Types.Scope().Lookup(test.represented).Type())
				}
				unchanged := bindingKeyInputSnapshot([]*abstractValue{value})
				analysis := effectResolverAnalysis(program)
				analysis.jsonCallbacks(value, decoding, root.declaration.Pos())
				host, unresolved := false, false
				for _, finding := range analysis.findings {
					host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "record.Failure.Error -> goroutine start")
					unresolved = unresolved || finding.Category == "unresolved-effect"
				}
				if host != test.host || unresolved != test.unresolved {
					t.Errorf("host=%v unresolved=%v want host=%v unresolved=%v: %v", host, unresolved, test.host, test.unresolved, analysis.findings)
				}
				if !unchanged() {
					t.Fatal("container shape traversal changed the original element graph or aliases")
				}
			})
		}
	}
}

func TestJSONSummaryZeroContainerSourceCalls(t *testing.T) {
	const module = "example.invalid/jsoncontainershape"
	program := stockSummaryProgram(t, module, `package record
import ("bytes"; "encoding/json")
func RootMap() { var fields map[string]json.RawMessage; _ = json.Unmarshal([]byte("{}"), &fields) }
func decode(target any) { decoder := json.NewDecoder(bytes.NewReader([]byte("{}"))); _ = decoder.Decode(target) }
func RootForwarded() { var value struct { Subscriptions []json.RawMessage; Labels map[string]string }; decode(&value) }
func RootMarshal() { var value struct { Bytes []byte; Values map[string][]uint64 }; _, _ = json.Marshal(value) }
`)
	for _, name := range []string{"RootMap", "RootForwarded", "RootMarshal"} {
		t.Run(name, func(t *testing.T) {
			analysis := effectResolverAnalysis(program)
			root := program.Functions[module+"/record."+name]
			analysis.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			if len(analysis.findings) != 0 {
				t.Fatalf("source-bound concrete empty container received callback findings: %v", analysis.findings)
			}
		})
	}
}
