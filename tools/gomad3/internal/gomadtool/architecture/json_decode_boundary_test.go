package architecture

import (
	"go/types"
	"strings"
	"testing"
)

func TestJSONDecodeBoundaryPreservesCallbacks(t *testing.T) {
	const module = "example.invalid/jsondecodeboundary"
	tests := []struct {
		name, parameter, body string
		host, unresolved      bool
	}{
		{"NilEmpty", "", `var value any; _ = decoder.Decode(&value)`, false, false},
		{"NilNamedEmpty", "", `var value Empty; _ = decoder.Decode(&value)`, false, false},
		{"UnmarshalNilEmpty", "", `var value any; _ = json.Unmarshal([]byte("{}"), &value)`, false, false},
		{"Prepopulated", "", `var value any = Dirty{}; _ = decoder.Decode(&value)`, true, false},
		{"TypedNilDirty", "", `var value *Dirty; _ = decoder.Decode(&value)`, true, true},
		{"NonemptyInterface", "", `var value interface{ Marker() }; _ = decoder.Decode(&value)`, false, true},
		{"AliasMutation", "", `var value any; alias := &value; set(alias); _ = decoder.Decode(alias)`, true, false},
		{"SecondDecode", "", `var value any; _ = decoder.Decode(&value); _ = decoder.Decode(&value)`, false, true},
		{"PostDecodeBranch", "", `var value any; _ = decoder.Decode(&value); if value != nil { go sideEffect() }`, true, false},
		{"NilOuterPointer", "", `var destination *any; _ = decoder.Decode(destination)`, false, true},
		{"UnknownPointer", "destination *any", `_ = decoder.Decode(destination)`, false, true},
		{"ReaderError", "", `var value any; decoder = json.NewDecoder(Reader{}); _ = decoder.Decode(&value)`, true, true},
	}
	source := `package record
import ("bytes"; "encoding/json")
type Empty interface{}
type Dirty struct{}
func (Dirty) UnmarshalJSON([]byte) error { go sideEffect(); return nil }
func set(destination *any) { *destination = Dirty{} }
func sideEffect() {}
type Reader struct{}
type Failure struct{}
func (Reader) Read([]byte) (int, error) { return 0, Failure{} }
func (Failure) Error() string { go sideEffect(); return "failure" }
`
	for _, test := range tests {
		source += "func Root" + test.name + "(" + test.parameter + ") { decoder := json.NewDecoder(bytes.NewReader([]byte(\"{}\"))); _ = decoder; " + test.body + " }\n"
	}
	program := stockSummaryProgram(t, module, source)
	t.Run("MutationClearsNilFacts", func(t *testing.T) {
		analysis := effectResolverAnalysis(program)
		root := program.Functions[module+"/record.RootNilEmpty"]
		reader := valueOf(root.pkg.Types.Scope().Lookup("Reader").Type())
		constructor := stockSummaryFunction(t, analysis, "encoding/json", "", "NewDecoder")
		decoder, _ := analysis.standard(constructor, nil, []*abstractValue{reader}, root.declaration.Pos())
		slot := zeroValue(types.Universe.Lookup("any").Type())
		destination := addressValue(slot, types.NewPointer(slot.types[0]), root.declaration.Pos())
		decode := stockSummaryFunction(t, analysis, "encoding/json", "*Decoder", "Decode")
		analysis.standard(decode, decoder, []*abstractValue{destination}, root.declaration.Pos())
		if slot.nilness != 0 || slot.nilType != nil {
			t.Fatalf("Decode mutation retained stale slot nil facts: nilness=%d nilType=%v", slot.nilness, slot.nilType)
		}
	})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis := effectResolverAnalysis(program)
			root := program.Functions[module+"/record.Root"+test.name]
			analysis.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			host, unresolved := false, false
			for _, finding := range analysis.findings {
				host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start")
				unresolved = unresolved || finding.Category == "unresolved-effect"
			}
			if host != test.host || unresolved != test.unresolved {
				t.Fatalf("host=%v unresolved=%v want host=%v unresolved=%v: %v", host, unresolved, test.host, test.unresolved, analysis.findings)
			}
		})
	}
	t.Run("GenericNilWalkerStillConservative", func(t *testing.T) {
		analysis := effectResolverAnalysis(program)
		root := program.Functions[module+"/record.RootNilEmpty"]
		analysis.jsonCallbacks(zeroValue(types.Universe.Lookup("any").Type()), true, root.declaration.Pos())
		if len(analysis.findings) == 0 {
			t.Fatal("generic callback walker accepted a local zero interface without a decode boundary")
		}
	})
}
