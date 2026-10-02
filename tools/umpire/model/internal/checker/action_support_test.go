package checker

import (
	"fmt"
	"reflect"
	"strings"
)

// Party is who performs an action. System is reserved for timers, which a machine owns.
type Party string

// Entity is what a machine keeps state for. Key names the recorded field that identifies an
// instance; Refer names the entities it refers to, by role.
type Entity struct {
	Name  string
	Key   string
	Refer map[string]*Entity
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

// ClassExample is an Abstraction Claim on one input class: the author's claim that every realized
// value of the class behaves alike, with the example the functional Case runs.
type ClassExample struct {
	Value   any
	Example string
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

// claimsOf lists the Abstraction Claims of the actions a machine binds, in the order the actions'
// classes first appear and then in declaration order.
func claimsOf(t *Table, classes []Class) []Claim {
	var out []Claim
	seen := map[*ActionDecl]bool{}
	for _, c := range classes {
		if seen[c.Decl] {
			continue
		}
		seen[c.Decl] = true
		for _, ex := range c.Decl.Examples {
			v := reflect.ValueOf(ex.Value)
			key := c.Decl.Name + "-" + keyOf(v, c.Decl.types[0])
			out = append(out, Claim{Member: t.Family.ID("action", t.owner(), key),
				Action: string(t.Family) + ".action." + c.Decl.Name, Field: c.Decl.Inputs[0],
				ClassName: classSpelling(v, c.Decl.types[0]), Example: ex.Example})
		}
	}
	return out
}

// classSpelling spells a class the way a Lean `examples:` line does: a variant by its constructor
// and named fields, `handlerError (retryable := true)`, and anything else by its key.
func classSpelling(v reflect.Value, static reflect.Type) string {
	if static.Kind() != reflect.Interface {
		return keyOf(v, static)
	}
	name := lowerFirst(v.Type().Name())
	var fields []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if f.IsExported() {
			fields = append(fields, fieldName(f)+" := "+keyOf(v.Field(i), f.Type))
		}
	}
	if len(fields) == 0 {
		return name
	}
	return name + " (" + strings.Join(fields, ", ") + ")"
}

const System Party = "system"

// Action0 is an action with no input. It has one class, named by the action.
type Action0 struct{ *ActionDecl }

// Action1 is an action with one input of type A.
type Action1[A any] struct{ *ActionDecl }

// Option sets one optional part of an action declaration.
type Option func(*ActionDecl)

// WithExample records an Abstraction Claim, as a Lean `examples:` line does. Claims keep their
// declaration order, which is the order exploration targets list them in.
func WithExample(value any, example string) Option {
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

// With selects the one class of an action with no input.
func (a *Action0) With() Class { return Class{Decl: a.ActionDecl} }

// With selects the class of this input value.
func (a *Action1[A]) With(v A) Class {
	return Class{Decl: a.ActionDecl, values: []reflect.Value{reflect.ValueOf(&v).Elem()}}
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
