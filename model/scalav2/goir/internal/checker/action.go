package checker

import (
	"fmt"
	"reflect"
	"strings"
)

// Party is who performs an action. System is reserved for timers, which a machine owns.
type Party string

const System Party = "system"

// Entity is what a machine keeps state for. Key names the recorded field that identifies an
// instance; Refer names the entities it refers to, by role.
type Entity struct {
	Name  string
	Key   string
	Refer map[string]*Entity
}

// Observation is a derived read used as evidence where no history event exists: the Go form of
// the Lean `observation` command.
type Observation struct {
	Name string
	On   *Entity
	Read string
}

// ActionDecl is one declared action: a party's side effect, or a timer the system owns. Its
// inputs are finite domains; each assignment of them is one class.
type ActionDecl struct {
	Name     string
	Party    Party
	On       *Entity
	Creates  *Entity
	Schemas  []string
	Inputs   []string
	Results  string
	Examples []ClassExample
	timer    bool
	types    []reflect.Type
	err      error
}

// Timer declares a timer: an action with no input that the system performs.
func Timer(name string) *Action0 {
	return &Action0{&ActionDecl{Name: name, Party: System, timer: true}}
}

// Action0 is an action with no input. It has one class, named by the action.
type Action0 struct{ *ActionDecl }

// Action1 is an action with one input of type A.
type Action1[A any] struct{ *ActionDecl }

// Action2 is an action with two inputs.
type Action2[A, B any] struct{ *ActionDecl }

// Action3 is an action with three inputs.
type Action3[A, B, C any] struct{ *ActionDecl }

// Option sets one optional part of an action declaration.
type Option func(*ActionDecl)

// On names the entity the action acts on.
func On(e *Entity) Option { return func(d *ActionDecl) { d.On = e } }

// Creates names the entity the action creates.
func Creates(e *Entity) Option { return func(d *ActionDecl) { d.Creates = e } }

// Schema names the protobuf messages the action carries, as the Lean `schema:` line does.
func Schema(names ...string) Option { return func(d *ActionDecl) { d.Schemas = names } }

// Results names the domain of results the action reports, as the Lean `results:` line does.
func Results(name string) Option { return func(d *ActionDecl) { d.Results = name } }

// ClassExample is an Abstraction Claim on one input class: the author's claim that every realized
// value of the class behaves alike, with the example the functional Case runs.
type ClassExample struct {
	Value   any
	Example string
}

// Example records an Abstraction Claim, as a Lean `examples:` line does. Claims keep their
// declaration order, which is the order exploration targets list them in.
func Example(value any, example string) Option {
	return func(d *ActionDecl) { d.Examples = append(d.Examples, ClassExample{value, example}) }
}

func declare(d *ActionDecl, opts []Option) *ActionDecl {
	for _, o := range opts {
		o(d)
	}
	// An example names a class of the action's input, so its value must be one. A wrong type fails
	// when the package initializes, before any test runs.
	for _, ex := range d.Examples {
		switch {
		case len(d.types) != 1:
			d.err = fmt.Errorf("action %s: an example names a class of a one-input action", d.Name)
		case !reflect.TypeOf(ex.Value).AssignableTo(d.types[0]):
			d.err = fmt.Errorf("action %s: example %v is a %T, not a %s", d.Name, ex.Value, ex.Value, d.types[0])
		default:
		}
	}
	return d
}

// NewAction0 declares an action with no input.
func NewAction0(name string, party Party, opts ...Option) *Action0 {
	return &Action0{declare(&ActionDecl{Name: name, Party: party}, opts)}
}

// NewAction1 declares an action with one input, named as the Lean `input:` line names it.
func NewAction1[A any](name string, party Party, input string, opts ...Option) *Action1[A] {
	return &Action1[A]{declare(&ActionDecl{Name: name, Party: party, Inputs: []string{input},
		types: []reflect.Type{reflect.TypeFor[A]()}}, opts)}
}

// NewAction2 declares an action with two inputs.
func NewAction2[A, B any](name string, party Party, a, b string, opts ...Option) *Action2[A, B] {
	return &Action2[A, B]{declare(&ActionDecl{Name: name, Party: party, Inputs: []string{a, b},
		types: []reflect.Type{reflect.TypeFor[A](), reflect.TypeFor[B]()}}, opts)}
}

// NewAction3 declares an action with three inputs.
func NewAction3[A, B, C any](name string, party Party, a, b, c string, opts ...Option) *Action3[A, B, C] {
	return &Action3[A, B, C]{declare(&ActionDecl{Name: name, Party: party, Inputs: []string{a, b, c},
		types: []reflect.Type{reflect.TypeFor[A](), reflect.TypeFor[B](), reflect.TypeFor[C]()}}, opts)}
}

// Class is one class of an action: the action with one assignment of its inputs.
type Class struct {
	Decl   *ActionDecl
	values []reflect.Value
}

// Key is the class key: the action name followed by its input keys, joined by "-".
func (c Class) Key() string {
	parts := []string{c.Decl.Name}
	for i, v := range c.values {
		parts = append(parts, keyOf(v, c.Decl.types[i]))
	}
	return strings.Join(parts, "-")
}

// With selects the one class of an action with no input.
func (a *Action0) With() Class { return Class{Decl: a.ActionDecl} }

// With selects the class of this input value.
func (a *Action1[A]) With(v A) Class {
	return Class{Decl: a.ActionDecl, values: []reflect.Value{reflect.ValueOf(&v).Elem()}}
}

// With selects the class of these input values.
func (a *Action2[A, B]) With(x A, y B) Class {
	return Class{Decl: a.ActionDecl, values: []reflect.Value{
		reflect.ValueOf(&x).Elem(), reflect.ValueOf(&y).Elem()}}
}

// With selects the class of these input values.
func (a *Action3[A, B, C]) With(x A, y B, z C) Class {
	return Class{Decl: a.ActionDecl, values: []reflect.Value{
		reflect.ValueOf(&x).Elem(), reflect.ValueOf(&y).Elem(), reflect.ValueOf(&z).Elem()}}
}

// classes enumerates every class of an action: the product of its input domains, with the last
// input varying fastest.
func (d *ActionDecl) classes() ([]Class, error) {
	if d.err != nil {
		return nil, d.err
	}
	out := []Class{{Decl: d}}
	for i, t := range d.types {
		members, err := domainOf(t)
		if err != nil {
			return nil, fmt.Errorf("action %s input %s: %w", d.Name, d.Inputs[i], err)
		}
		next := make([]Class, 0, len(out)*len(members))
		for _, prefix := range out {
			for _, m := range members {
				values := append(append([]reflect.Value{}, prefix.values...), m)
				next = append(next, Class{Decl: d, values: values})
			}
		}
		out = next
	}
	return out, nil
}
