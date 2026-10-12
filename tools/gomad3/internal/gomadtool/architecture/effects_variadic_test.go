package architecture

import (
	"go/types"
	"reflect"
	"strings"
	"testing"
)

func TestEffectVariadicASTPreservesEveryTailCallback(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantHost   bool
	}{
		{"clean-tail", "Invoke(clean, clean)", false},
		{"dirty-second", "Invoke(clean, dirty)", true},
		{"dirty-third", "Invoke(clean, clean, dirty)", true},
		{"joined-tail-alternatives", "var flag bool; Invoke(clean, choose(flag))", true},
		{"dirty-fixed-prefix", "WithPrefix(dirty, clean, clean)", true},
		{"dirty-after-fixed-prefix", "WithPrefix(clean, clean, dirty)", true},
		{"function-value", "f := Invoke; f(clean, dirty)", true},
		{"function-literal", "f := func(values ...func()) { Invoke(values...) }; f(clean, dirty)", true},
		{"method-receiver", "Consumer{}.Invoke(clean, dirty)", true},
		{"method-value", "f := Consumer{}.Invoke; f(clean, dirty)", true},
		{"method-expression", "Consumer.Invoke(Consumer{}, clean, dirty)", true},
		{"method-expression-function-value", "f := Consumer.Invoke; f(Consumer{}, clean, dirty)", true},
		{"method-expression-fixed-prefix", "Consumer.WithPrefix(Consumer{}, clean, dirty)", true},
		{"method-expression-function-value-fixed-prefix", "f := Consumer.WithPrefix; f(Consumer{}, clean, dirty)", true},
		{"method-expression-expanded", "Consumer.WithPrefix(Consumer{}, clean, []func(){clean, dirty}...)", true},
		{"method-expression-function-value-expanded", "f := Consumer.WithPrefix; f(Consumer{}, clean, []func(){clean, dirty}...)", true},
		{"interface-method", "var consumer Runner = Consumer{}; consumer.Invoke(clean, dirty)", true},
		{"interface-expanded", "var consumer Runner = Consumer{}; consumer.Invoke([]func(){clean, dirty}...)", true},
		{"expanded", "Invoke([]func(){clean, dirty}...)", true},
		{"expanded-clean", "Invoke([]func(){clean, clean}...)", false},
		{"unknown-tail", "var unknown func(); Invoke(clean, unknown)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, `package record
type Consumer struct{}
type Runner interface{Invoke(...func())}
func Root(){`+tc.body+`}
func Invoke(values ...func()){ for _, value := range values { value() } }
func WithPrefix(first func(), values ...func()){ first(); Invoke(values...) }
func (Consumer) Invoke(values ...func()){ Invoke(values...) }
func (Consumer) WithPrefix(first func(), values ...func()){first(); Invoke(values...)}
func clean(){}
func dirty(){go clean()}
func choose(flag bool)func(){if flag{return clean};return dirty}
`)
			a := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			var host, unresolved bool
			for _, finding := range a.findings {
				host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start")
				unresolved = unresolved || finding.Category == "unresolved-effect"
			}
			if host != tc.wantHost {
				t.Fatalf("host effect=%v want=%v findings=%v", host, tc.wantHost, a.findings)
			}
			if tc.name == "unknown-tail" || tc.name == "function-literal" {
				// Pure-file function literals are also independent roots with
				// unknown incoming arguments. Keep that conservative finding even
				// when the separate actual invocation binds a known dirty tail.
				if !unresolved {
					t.Fatalf("unknown variadic root was accepted: %v", a.findings)
				}
			} else if unresolved {
				t.Fatalf("known variadic binding became unresolved: %v", a.findings)
			}
		})
	}
}

func TestEffectVariadicUninitializedLocalCallbacksRemainUnresolved(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantHost   bool
	}{
		{"reversed", "Invoke(unknown, clean)", false},
		{"expanded", "Invoke([]func(){clean, unknown}...)", false},
		{"dirty-first", "Invoke(dirty, unknown)", true},
		{"dirty-last", "Invoke(unknown, dirty)", true},
		{"expanded-dirty", "Invoke([]func(){dirty, unknown}...)", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, `package record
func Root(){var unknown func(); `+tc.body+`}
func Invoke(values ...func()){for _, value := range values {value()}}
func clean(){}
func dirty(){go clean()}
`)
			a := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			var host, unresolved bool
			for _, finding := range a.findings {
				host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start")
				unresolved = unresolved || finding.Category == "unresolved-effect"
			}
			if host != tc.wantHost || !unresolved {
				t.Fatalf("host=%v want=%v unresolved=%v findings=%v", host, tc.wantHost, unresolved, a.findings)
			}
		})
	}
}

func TestEffectVariadicNamedSliceDistinguishesEllipsis(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantHost   bool
	}{
		{"slice-is-one-writer", "Invoke(Named{Clean{}})", true},
		{"expanded-slice-elements", "Invoke(Named{Clean{}}...)", false},
		{"expanded-dirty-element", "Invoke(Named{Clean{}, Dirty{}}...)", true},
		{"interface-slice-is-one-writer", "var consumer Runner = Consumer{}; consumer.Invoke(Named{Clean{}})", true},
		{"interface-expanded-slice-elements", "var consumer Runner = Consumer{}; consumer.Invoke(Named{Clean{}}...)", false},
		{"interface-expanded-dirty-element", "var consumer Runner = Consumer{}; consumer.Invoke(Named{Clean{}, Dirty{}}...)", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := effectResolverProgram(t, `package record
type Writer interface{ Write() }
type Clean struct{}
type Dirty struct{}
type Named []Writer
type Runner interface{Invoke(...Writer)}
type Consumer struct{}
func (Clean) Write(){}
func (Dirty) Write(){go sideEffect()}
func (Named) Write(){go sideEffect()}
func sideEffect(){}
func Invoke(values ...Writer){for _, value := range values {value.Write()}}
func (Consumer) Invoke(values ...Writer){Invoke(values...)}
func Root(){`+tc.body+`}
`)
			a := effectResolverAnalysis(program)
			root := program.Functions["example.invalid/resolver/record.Root"]
			a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
			var host bool
			for _, finding := range a.findings {
				host = host || finding.Category == "host-effect"
				if finding.Category == "unresolved-effect" {
					t.Fatalf("known named slice binding unresolved: %v", a.findings)
				}
			}
			if host != tc.wantHost {
				t.Fatalf("host effect=%v want=%v findings=%v", host, tc.wantHost, a.findings)
			}
		})
	}
}

func TestEffectVariadicFormalResolverRetainsEveryCallAndTail(t *testing.T) {
	program := effectResolverProgram(t, `package record
func Root(){Invoke(clean, dirty); Invoke([]func(){other}...)}
func Invoke(values ...func()){return}
func clean(){}
func dirty(){go clean()}
func other(){}
`)
	fn := program.Functions["example.invalid/resolver/record.Invoke"]
	param := fn.object.Type().(*types.Signature).Params().At(0)
	a := effectResolverAnalysis(program)
	value := a.resolve(param)
	want := []string{"example.invalid/resolver/record.clean", "example.invalid/resolver/record.dirty", "example.invalid/resolver/record.other"}
	if got := effectResolverFunctionIDs(elementValue(value)); !reflect.DeepEqual(got, want) {
		t.Fatalf("variadic formal callbacks=%v want=%v", got, want)
	}
	if a.bindingDepth != 0 || len(a.resolving) != 0 {
		t.Fatal("variadic formal resolver leaked recursion guards")
	}
}

func TestEffectVariadicActualZeroArgumentsDoNotReplaceUnknownRoot(t *testing.T) {
	program := effectResolverProgram(t, `package record
func Root() []func(){return Collect()}
func Collect(values ...func()) []func(){return values}
`)
	a := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	value := a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	if value == nil || value.elements == nil || value.elements.unknown || len(value.elements.functions) != 0 || len(value.elements.types) != 0 {
		t.Fatalf("actual zero arguments not represented as a known empty variadic slice: %#v", value)
	}
	// A root with no actual AST call site has unknown incoming parameters. A nil
	// call argument vector must not silently declare that its variadic tail is empty.
	unbound := effectResolverProgram(t, `package record
func Root(values ...func()) []func(){return values}
`)
	b := effectResolverAnalysis(unbound)
	fn := unbound.Functions["example.invalid/resolver/record.Root"]
	unknown := b.call(&boundFunction{fn: fn, pkg: fn.pkg}, nil, fn.declaration.Pos())
	if callback := elementValue(unknown); callback == nil || !callback.unknown {
		t.Fatalf("unknown root tail was converted to empty: %#v", unknown)
	}
}

func TestEffectVariadicUnknownRootUsesActualCallerBindings(t *testing.T) {
	program := effectResolverProgram(t, `package record
func Caller(){Invoke(clean, dirty)}
func Invoke(values ...func()){for _, value := range values {value()}}
func clean(){}
func dirty(){go clean()}
`)
	a := effectResolverAnalysis(program)
	fn := program.Functions["example.invalid/resolver/record.Invoke"]
	a.call(&boundFunction{fn: fn, pkg: fn.pkg}, nil, fn.declaration.Pos())
	var host bool
	for _, finding := range a.findings {
		host = host || finding.Category == "host-effect" && strings.Contains(finding.Detail, "goroutine start")
		if finding.Category == "unresolved-effect" {
			t.Fatalf("known caller tail was lost during root resolution: %v", a.findings)
		}
	}
	if !host {
		t.Fatalf("nil root arguments hid actual dirty caller tail: %v", a.findings)
	}
}

func TestEffectNonvariadicASTKeepsPositionalBindings(t *testing.T) {
	program := effectResolverProgram(t, `package record
func Root(){Invoke(clean, dirty)}
func Invoke(first, second func()){first()}
func clean(){}
func dirty(){go clean()}
`)
	a := effectResolverAnalysis(program)
	root := program.Functions["example.invalid/resolver/record.Root"]
	a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	if len(a.findings) != 0 {
		t.Fatalf("nonvariadic positional binding changed: %v", a.findings)
	}
}

func TestEffectVariadicASTKeepsFlatStandardSummaryArguments(t *testing.T) {
	a := reflectSummaryAnalysis(t, `package record
type Clean struct{}
type Dirty struct{}
func (Clean) String()string{return "clean"}
func (Dirty) String()string{go sideEffect();return "dirty"}
func sideEffect(){}
func Format(format string, values ...any)string{return ""}
func Root(){Format("%v %v", Clean{}, Dirty{})}
`)
	root := a.program.Functions["example.invalid/resolver/record.Root"]
	format := a.program.Functions["example.invalid/resolver/record.Format"]
	standard := stockSummaryFunction(t, a, "fmt", "", "Sprintf")
	for ident, object := range root.pkg.Info.Uses {
		if object == format.object {
			root.pkg.Info.Uses[ident] = standard
		}
	}
	a.call(&boundFunction{fn: root, pkg: root.pkg}, nil, root.declaration.Pos())
	if len(a.findings) != 1 {
		t.Fatalf("flat standard summary lost dirty tail: %v", a.findings)
	}
	for _, finding := range a.findings {
		if finding.Category != "host-effect" || !strings.Contains(finding.Detail, "record.Dirty.String") || !strings.Contains(finding.Detail, "goroutine start") {
			t.Fatalf("standard summary arguments incorrectly packed: %v", finding)
		}
	}
}

func TestEffectMethodExpressionFormalResolverKeepsReceiverOffset(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Consumer struct{}
func Caller(){Consumer.Invoke(Consumer{}, clean, dirty); Consumer.Invoke(Consumer{}, other, []func(){dirty}...)}
func (Consumer) Invoke(first func(), values ...func()){}
func clean(){}
func dirty(){go clean()}
func other(){}
`)
	a := effectResolverAnalysis(program)
	fn := program.Functions["example.invalid/resolver/record.Consumer.Invoke"]
	params := fn.object.Type().(*types.Signature).Params()
	first := effectResolverFunctionIDs(a.resolve(params.At(0)))
	want := []string{"example.invalid/resolver/record.clean", "example.invalid/resolver/record.other"}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("receiver shifted fixed parameter: got=%v want=%v", first, want)
	}
	tail := effectResolverFunctionIDs(elementValue(a.resolve(params.At(1))))
	if !reflect.DeepEqual(tail, []string{"example.invalid/resolver/record.dirty"}) {
		t.Fatalf("receiver shifted variadic tail: %v", tail)
	}
}

func TestEffectMethodExpressionBindingIdentityAndUnknownReceiver(t *testing.T) {
	program := effectResolverProgram(t, `package record
type Consumer struct{}
func (Consumer) Invoke(first func(), values ...func()){first()}
`)
	fn := program.Functions["example.invalid/resolver/record.Consumer.Invoke"]
	methodExpression := &boundFunction{fn: fn, pkg: fn.pkg, methodExpression: true}
	methodValue := &boundFunction{fn: fn, pkg: fn.pkg}
	expression := &abstractValue{functions: []*boundFunction{methodExpression}}
	value := &abstractValue{functions: []*boundFunction{methodValue}}
	if bindingKey([]*abstractValue{expression}) == bindingKey([]*abstractValue{value}) {
		t.Fatal("method-expression and method-value argument conventions share a binding key")
	}
	for _, combined := range []*abstractValue{join(expression, value), normalizeAllocations(join(expression, value)), cloneValue(join(expression, value))} {
		if combined == nil || len(combined.functions) != 2 || combined.functions[0].methodExpression == combined.functions[1].methodExpression {
			t.Fatalf("mixed function alternatives dropped an argument convention: %#v", combined)
		}
	}
	for _, tc := range []struct {
		name string
		args []*abstractValue
	}{
		{"missing", nil},
		{"unknown", []*abstractValue{{unknown: true}, {unknown: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := effectResolverAnalysis(program)
			b.sourceCall(methodExpression, tc.args, false, fn.declaration.Pos())
			found := false
			for _, finding := range b.findings {
				found = found || finding.Category == "unresolved-effect" && strings.Contains(finding.Detail, tc.name+" method-expression receiver")
			}
			if !found {
				t.Fatalf("%s actual receiver accepted: %v", tc.name, b.findings)
			}
		})
	}
	b := effectResolverAnalysis(program)
	b.call(methodExpression, nil, fn.declaration.Pos())
	if len(b.findings) == 0 {
		t.Fatal("unknown external root accepted as an empty actual call")
	}
	for _, finding := range b.findings {
		if strings.Contains(finding.Detail, "method-expression receiver") {
			t.Fatalf("non-AST root received actual-call normalization: %v", finding)
		}
	}
}
