package architecture

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestJSONErrorCallbacksUseEligibleSignatures(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Wrong struct{}
func (Wrong) Error()string{return "wrong"}
func (Wrong) Is(string)bool{go sideEffect();return false}
func (Wrong) Unwrap()string{go sideEffect();return "wrong"}
func sideEffect(){}
func Root(){}
`)
	analysis := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	analysis.jsonErrorCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Wrong").Type()), root.declaration.Pos())
	if len(analysis.findings) != 0 {
		t.Fatalf("JSON invoked methods not implementing an error callback interface: %v", analysis.findings)
	}
}

func TestJSONErrorCallbacksRetainUnwrappedErrors(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Failure struct{}
type Wrap struct{}
type Multi struct{}
func (Failure) Error()string{go sideEffect();return "failure"}
func (Failure) Is(error)bool{go sideEffect();return false}
func (Wrap) Error()string{return "wrap"}
func (Wrap) Unwrap()error{return Failure{}}
func (Multi) Error()string{return "multi"}
func (Multi) Unwrap()[]error{return []error{Failure{}}}
func sideEffect(){}
func Root(){}
`)
	for _, name := range []string{"Wrap", "Multi"} {
		t.Run(name, func(t *testing.T) {
			analysis := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			analysis.jsonErrorCallbacks(valueOf(root.pkg.Types.Scope().Lookup(name).Type()), root.declaration.Pos())
			for _, callback := range []string{"Error", "Is"} {
				found := false
				for _, finding := range analysis.findings {
					found = found || finding.Category == "host-effect" && strings.Contains(finding.Detail, "/record.Failure."+callback+" -> goroutine start")
				}
				if !found {
					t.Fatalf("JSON lost unwrapped %s callback: %v", callback, analysis.findings)
				}
			}
		})
	}
}

func TestJSONErrorCallbacksNilAndUnknownRemainDistinct(t *testing.T) {
	program := effectResolverProgram(t, "package record\nfunc Root(){}\n")
	knownNil := effectResolverAnalysis(program)
	knownNil.jsonErrorCallbacks(valueOf(types.Typ[types.UntypedNil]), token.NoPos)
	if len(knownNil.findings) != 0 {
		t.Fatalf("known nil acquired error callbacks: %v", knownNil.findings)
	}
	for _, value := range []*abstractValue{nil, valueOf(types.Universe.Lookup("error").Type()), {unknown: true}} {
		analysis := effectResolverAnalysis(program)
		analysis.jsonErrorCallbacks(value, token.NoPos)
		if len(analysis.findings) == 0 {
			t.Fatal("unknown error binding was treated as known nil")
		}
	}
}

func TestJSONErrorCallbacksNilMultiUnwrapHasNoChildCallback(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Empty struct{}
func (Empty) Error()string{return "empty"}
func (Empty) Unwrap()[]error{return nil}
func Root(){}
`)
	analysis := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	analysis.jsonErrorCallbacks(valueOf(root.pkg.Types.Scope().Lookup("Empty").Type()), root.declaration.Pos())
	if len(analysis.findings) != 0 {
		t.Fatalf("known nil unwrap acquired an unknown child error: %v", analysis.findings)
	}
}
