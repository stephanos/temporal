// Package umpire is the framework surface the Model packages author against.
//
// A Model is ordinary Go: entities, actions and machines are package-level values, step functions
// are plain funcs, and the finite state table, the refinement and the Queries are built and answered
// by Check at test time. Lean runs the same checks while elaborating the file; Go has no hook between
// "it compiles" and "it runs", so everything below that is not a type error surfaces from `go test`.
// Bodies here are sketched: the shape is what the Model files bind to.
package umpire

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
)

// ---------------------------------------------------------------------------------------------
// Vocabulary

// Party is who performs an action. System is reserved for timers: a machine owns those under Timers
// rather than declaring them as actions of a party.
type Party string

const (
	Caller   Party = "caller"
	Handler  Party = "handler"
	Worker   Party = "worker"
	Network  Party = "network"
	Operator Party = "operator"
	System   Party = "system"
)

// Entity is what a machine is about. Key names the recorded field that identifies an instance; Refer
// links the entity to others by role.
type Entity struct {
	Name  string
	Key   string
	Refer map[string]*Entity
}

// Observation is a derived read used as evidence where no history event exists.
type Observation struct {
	Name string
	On   *Entity
	Read string
}

// ---------------------------------------------------------------------------------------------
// Finite enumeration

// Finite is what a state, outcome, fact or input type provides so the table can be enumerated: every
// value, in catalog order. `//go:generate go run ../umpire/cmd/finite -type=...` emits it beside
// String for integer enums; a bounded counter writes it by hand. The receiver is unused, the
// framework calls it on the zero value, which is why an interface type cannot satisfy it and sum
// types go through Sum instead.
type Finite[T any] interface {
	Values() []T
}

// Domain is the enumeration of one type in catalog order. Row keys and fixture names come from
// String when the type has it and from %v otherwise.
type Domain[T comparable] []T

// Enum is the domain of a Finite value type: an integer enum, or a struct whose zero value is a member.
func Enum[T interface {
	comparable
	Finite[T]
}]() Domain[T] {
	var zero T
	return zero.Values()
}

// Fields is the domain of a struct state: the cartesian product of its fields, each enumerated
// through the Values method of its type (bool is built in). Reflection finds the methods, so a field
// whose type has none is a Check-time error rather than a compile-time one, and field order fixes
// the catalog order. Nothing stops a step function from constructing a state outside this domain;
// Check rejects the row when it builds the table.
func Fields[S comparable]() Domain[S] {
	var zero S
	return productOfFields[S](reflect.TypeOf(zero)) // walks the fields; sketched
}

// Sum is the domain of a sealed interface. Go cannot find an interface's implementers at run time, so
// the author lists one value per variant, and a variant that carries finite fields contributes one
// class per assignment of them: HandlerError{} stands for both of its classes. go-check-sumtype keeps
// this list and the type switches on the interface honest.
func Sum[T comparable](variants ...T) Domain[T] {
	var out Domain[T]
	for _, v := range variants {
		out = append(out, expandVariant(v)...) // Fields on the variant's struct; sketched
	}
	return out
}

// ---------------------------------------------------------------------------------------------
// Actions and classes

// Action is the erased view of Action0, Action1 and Action3: what Timers, Unobservable, a
// composition's Sync and a Scenario's path name without caring about the input types. A bare Action
// used as a Class stands for every class of it.
type Action interface {
	Class
	ActionName() string
	ActionParty() Party
	inputTypes() []reflect.Type
}

// Class is one action with one assignment of its finite inputs. Scenario paths and Property When
// lines are written in these; With on an action builds one with the inputs checked by the compiler.
type Class interface {
	class() (Action, []any)
}

// Action0 is an action without inputs: a fault, a timer, a poll. Creates or On names its entity;
// Schema the protobuf messages it carries.
type Action0 struct {
	Name    string
	Party   Party
	Creates *Entity
	On      *Entity
	Schema  []string
}

// Action1 is an action with one finite input. Classes is required when A is an interface, since the
// framework cannot enumerate it; otherwise Enum[A]() is the default. Examples maps an input class to
// the realization value it stands for; Results names the outcome enum an implementation answers with.
type Action1[A comparable] struct {
	Name     string
	Party    Party
	Creates  *Entity
	On       *Entity
	Schema   []string
	Input    string
	Classes  Domain[A]
	Examples map[A]string
	Results  []string
}

// Action3 is an action with three finite inputs. There is no ActionN: Go has no variadic type
// parameters, so each arity is its own type, and these Models need exactly 0, 1 and 3.
type Action3[A, B, C comparable] struct {
	Name    string
	Party   Party
	Creates *Entity
	On      *Entity
	Schema  []string
	Inputs  [3]string
}

func (a *Action1[A]) With(input A) Class             { return classOf(a, input) }
func (a *Action3[A, B, C]) With(x A, y B, z C) Class { return classOf(a, x, y, z) }

// Schema names the protobuf messages an action carries with the compiler checking they exist: the
// Lean file writes `temporal.api.nexus.v1.HandlerError` as a string, this writes `&nexuspb.HandlerError{}`
// and reads the full name back through protoreflect.
func Schema(messages ...proto.Message) []string {
	names := make([]string, 0, len(messages))
	for _, m := range messages {
		names = append(names, string(proto.MessageName(m)))
	}
	return names
}

// Results is the result enum of an action, by the keys of its members.
func Results[T interface {
	comparable
	Finite[T]
}]() []string {
	return keysOf(Enum[T]())
}

// ---------------------------------------------------------------------------------------------
// Steps and machines

// Step is one row's result: the outcome, the state after, and the facts the step records. A step
// function returns nil when its action is not enabled in the state it was given.
type Step[S, O, F comparable] struct {
	Outcome O
	State   S
	Facts   []F
}

// Records is whether the step records the fact. A method rather than slices.Contains because Go's
// inference will not unify a []ProtocolFact with a NexusOperationCompleted{} argument; a method
// has F fixed already and takes the implicit interface conversion.
func (s Step[S, O, F]) Records(fact F) bool {
	for _, f := range s.Facts {
		if f == fact {
			return true
		}
	}
	return false
}

// Binding ties one action to its step function with the input types checked by the compiler:
// Bind1(handlerReply, step) does not compile if step does not take handlerReply's input type. It is a
// function rather than handlerReply.Bind(step) because Go methods cannot have type parameters.
type Binding[S, O, F comparable] struct {
	action Action
	step   func(S, []any) []Step[S, O, F]
}

func Bind0[S, O, F comparable](action *Action0, step func(S) []Step[S, O, F]) Binding[S, O, F] {
	return Binding[S, O, F]{action, func(s S, _ []any) []Step[S, O, F] { return step(s) }}
}

func Bind1[S, O, F, A comparable](action *Action1[A], step func(S, A) []Step[S, O, F]) Binding[S, O, F] {
	return Binding[S, O, F]{action, func(s S, in []any) []Step[S, O, F] { return step(s, in[0].(A)) }}
}

func Bind3[S, O, F, A, B, C comparable](action *Action3[A, B, C], step func(S, A, B, C) []Step[S, O, F]) Binding[S, O, F] {
	return Binding[S, O, F]{action, func(s S, in []any) []Step[S, O, F] {
		return step(s, in[0].(A), in[1].(B), in[2].(C))
	}}
}

// Steps collects the bindings of one machine. A variadic function rather than a slice literal so the
// type arguments are inferred from the first binding instead of spelled at the Machine field.
func Steps[S, O, F comparable](bindings ...Binding[S, O, F]) []Binding[S, O, F] { return bindings }

// Machine is one entity's behavior over a finite state type.
type Machine[S, O, F comparable] struct {
	Name   string
	For    *Entity
	States Domain[S]
	Starts []S
	// Ends is the predicate the design ends on. Lean lists phases; a predicate is how Go names the
	// subset of a struct state without a phase-only view of it.
	Ends func(S) bool
	// Timers are the system actions the machine owns; Unobservable the timers that record nothing.
	Timers       []Action
	Unobservable []Action
	// Evidence maps a fact to the recorded event or observation name. A key of a variant with fields,
	// such as NexusOperationTimedOut{}, stands for every class of it.
	Evidence map[F]string
	Steps    []Binding[S, O, F]
	// Refines, built by Refines(target, mapFn), is checked by Check, by mapped states: a row (s, a, s')
	// passes if map s == map s' (a target stutter) or the target has any row from map s to map s',
	// under any action class. Typed in S so the map cannot be attached to the wrong machine.
	Refines Refinement[S]
}

// Check builds the finite table and runs every semantic check the compiler could not: every step
// lands in States, every recorded fact has an evidence name, every action bound in Steps is declared
// once with the arity Bind gave it, Timers are system actions and Unobservable ⊆ Timers, and Refines
// holds. It fails the test at the first violation and returns the table for the pins.
func (m *Machine[S, O, F]) Check(t testing.TB) *Table[S, O, F] {
	t.Helper()
	return buildTable(t, m) // sketched
}

// Refinement re-runs only the refinement walk and returns its report.
func (m *Machine[S, O, F]) Refinement(t testing.TB) RefinementReport {
	t.Helper()
	return m.Refines.check(t, m.Check(t))
}

// Table is what Search, the Behavior Fingerprint and Contract lowering read.
type Table[S, O, F comparable] struct {
	States     []S
	Ends       []S
	ActionKeys []string // one per action class, in catalog order
	Rows       []Row[S, O, F]
	Reachable  []S
	Stuck      *S // a reachable non-end state no action leaves, if any
}

// Row is one (state, action class) pair and its step.
type Row[S, O, F comparable] struct {
	From   S
	Action string
	Step   Step[S, O, F]
}

// Refinement is the erased target of a Refines line; only Refines builds one.
type Refinement[S comparable] interface {
	check(t testing.TB, rows any) RefinementReport
}

// Refines declares that a machine over S refines target through mapState. The type parameters tie
// mapState's argument to the refining machine and its result to the target's state type.
func Refines[S, T, O2, F2 comparable](target *Machine[T, O2, F2], mapState func(S) T) Refinement[S] {
	return refinement[S, T, O2, F2]{target, mapState}
}

// RefinementReport is the derived step mapping: each row key of the refining machine to the target
// row it is, or nil for a stutter, and the first row that is neither.
type RefinementReport struct {
	Rows     map[string]*string
	Rejected *string
}

// ---------------------------------------------------------------------------------------------
// Properties, scenarios, limits, queries

// Model is what a Property, a Scenario and an exploratory Set name: a Machine or a Compose.
type Model[S, O, F comparable] interface {
	AnyModel
	table(t testing.TB) *Table[S, O, F]
}

// AnyModel is a Model with the type parameters erased, for a Set or a composition member.
type AnyModel interface {
	ModelName() string
	check(t testing.TB)
}

// Property is a claim about a machine.
type Property[S, O, F comparable] struct {
	Name    string
	Machine Model[S, O, F]
	// When names the action class the claim is about, and Holds is then a same-step claim of that
	// action's step. Without When, Transition is a claim of the step before and the step after.
	// Check rejects a Property with both or neither; the struct cannot.
	When       Class
	Holds      func(step Step[S, O, F]) bool
	Transition func(before, after Step[S, O, F]) bool
}

// Scenario is one path: a start and the classed actions in order.
type Scenario[S, O, F comparable] struct {
	Name    string
	Model   Model[S, O, F]
	Starts  S
	Actions []Class
}

type Limits struct {
	Name    string
	Steps   int
	Actions int
	Search  int
}

// Claim and Path are the erased Property and Scenario. A Query cannot be generic over both: a verify
// Query reads a product Property along a protocol Scenario, and the type system cannot say "a
// Property of a machine this Scenario's machine refines". Check confirms the pair instead.
type Claim interface {
	ClaimName() string
	claim()
}

type Path interface {
	PathName() string
	path()
}

// Query asks Search about one Property on one Scenario. Find is realized by a Set; Verify is searched
// over every trace of the path and never realized. Exactly one of the two is set.
type Query struct {
	Name   string
	Find   Claim
	Verify Claim
	In     Path
	Limits *Limits
}

// Answer is what Search returns. Explanation is the text a failing pin prints.
type Answer struct {
	Found          bool
	Witness        []string
	Verified       bool
	Counterexample []string
	Candidates     int
	Explanation    string
}

func (q *Query) Answer(t testing.TB) Answer {
	t.Helper()
	return search(t, q) // sketched
}

// ---------------------------------------------------------------------------------------------
// Sets

type Purpose string

const (
	Functional  Purpose = "functional"
	Canary      Purpose = "canary"
	Exploratory Purpose = "exploratory"
)

// Role is how a Set binds a party: the Case performs a driven party's actions and reads an observed
// party's.
type Role string

const (
	Driven   Role = "driven"
	Observed Role = "observed"
)

// Repeat is the switch a functional Set runs once per value of.
type Repeat string

const Implementation Repeat = "implementation"

// Cover is what an exploratory Set targets; the flags combine.
type Cover uint8

const (
	Rows Cover = 1 << iota
	Results
	ClassMembers
)

// Set is what Testpilot registers: a functional or canary list of Queries, or an exploration of one
// machine within a budget.
type Set struct {
	Name    string
	Purpose Purpose
	Bind    map[Party]Role
	Repeat  Repeat
	Queries []*Query
	Machine AnyModel
	Cover   Cover
	Budget  *Limits
}

// Check confirms every party the Queries' actions name is bound, a canary names no silent step, and
// each Query answers.
func (s *Set) Check(t testing.TB) {
	t.Helper()
	checkSet(t, s) // sketched
}

// ---------------------------------------------------------------------------------------------
// Composition

// Compose is the product of machines about different entities. Members are named, and the composed
// state is a struct whose fields carry the member name in an `umpire:"..."` tag. Sync names the
// composed actions: each fires one action of each named member as one step. A member action no Sync
// line names stays executable on its own.
type Compose[S comparable] struct {
	Name    string
	For     []*Entity
	Members Members
	Sync    Sync
	Starts  []S
	Ends    func(S) bool
}

type Members map[string]AnyModel

// Sync maps a composed action to the member actions it fires as one, by member name.
type Sync map[Action]map[string]Action

// Joint is a member's outcome or fact under the member's name. A composed step has one per member,
// and Go cannot type "the tuple of the members' types", so a Compose is a Model[S, Joint, Joint]: its
// Properties read State directly and match Facts through Joint.
type Joint struct {
	Member string
	Value  any
}

// Composed is the step type of a composition over S.
type Composed[S comparable] = Step[S, Joint, Joint]

// Check builds the product table, confirming each member's field and tag, that every Sync pair names
// actions the members bind, and that Starts and Ends fall in the product.
func (c *Compose[S]) Check(t testing.TB) *Table[S, Joint, Joint] {
	t.Helper()
	return buildProduct(t, c) // sketched
}

// At qualifies a member's own action in a composed path, as `operation.scheduleToStart` does in Lean.
func At(member string, class Class) Class { return memberClass{member, class} }

// Restrict cuts a machine to the actions a composition names.
func Restrict[S, O, F comparable](name string, m *Machine[S, O, F], actions ...Action) *Machine[S, O, F] {
	cut := *m
	cut.Name = name
	cut.Steps = keepBindings(m.Steps, actions)
	return &cut
}

// ---------------------------------------------------------------------------------------------
// Checking everything at once

// Checkable is any declaration with a Check. Testpilot calls Check on each Set it registers; the pins
// call it on everything.
type Checkable interface {
	check(t testing.TB)
}

func Check(t testing.TB, decls ...Checkable) {
	t.Helper()
	for _, d := range decls {
		d.check(t)
	}
}

// --- the interfaces above, satisfied ---------------------------------------------------------

func (m *Machine[S, O, F]) ModelName() string                  { return m.Name }
func (m *Machine[S, O, F]) check(t testing.TB)                 { m.Check(t) }
func (m *Machine[S, O, F]) table(t testing.TB) *Table[S, O, F] { return m.Check(t) }

func (c *Compose[S]) ModelName() string                          { return c.Name }
func (c *Compose[S]) check(t testing.TB)                         { c.Check(t) }
func (c *Compose[S]) table(t testing.TB) *Table[S, Joint, Joint] { return c.Check(t) }

func (p *Property[S, O, F]) ClaimName() string { return p.Name }
func (p *Property[S, O, F]) claim()            {}
func (p *Property[S, O, F]) check(t testing.TB) { checkProperty(t, p) }

func (s *Scenario[S, O, F]) PathName() string { return s.Name }
func (s *Scenario[S, O, F]) path()            {}
func (s *Scenario[S, O, F]) check(t testing.TB) { checkScenario(t, s) }

func (q *Query) check(t testing.TB) { q.Answer(t) }
func (s *Set) check(t testing.TB)   { s.Check(t) }

func (a *Action0) ActionName() string                { return a.Name }
func (a *Action0) ActionParty() Party                { return a.Party }
func (a *Action0) inputTypes() []reflect.Type        { return nil }
func (a *Action0) class() (Action, []any)            { return a, nil }
func (a *Action1[A]) ActionName() string             { return a.Name }
func (a *Action1[A]) ActionParty() Party             { return a.Party }
func (a *Action1[A]) inputTypes() []reflect.Type     { return typesOf[A]() }
func (a *Action1[A]) class() (Action, []any)         { return a, nil }
func (a *Action3[A, B, C]) ActionName() string       { return a.Name }
func (a *Action3[A, B, C]) ActionParty() Party       { return a.Party }
func (a *Action3[A, B, C]) inputTypes() []reflect.Type {
	return append(append(typesOf[A](), typesOf[B]()...), typesOf[C]()...)
}
func (a *Action3[A, B, C]) class() (Action, []any) { return a, nil }

// --- unexported plumbing, elided -------------------------------------------------------------

func typesOf[T any]() []reflect.Type                           { return []reflect.Type{reflect.TypeFor[T]()} }
func checkProperty[S, O, F comparable](testing.TB, *Property[S, O, F]) { panic("sketched") }
func checkScenario[S, O, F comparable](testing.TB, *Scenario[S, O, F]) { panic("sketched") }

func productOfFields[S comparable](reflect.Type) Domain[S]     { panic("sketched") }
func expandVariant[T comparable](T) []T                        { panic("sketched") }
func keysOf[T comparable](Domain[T]) []string                  { panic("sketched") }
func classOf(Action, ...any) Class                             { panic("sketched") }
func buildTable[S, O, F comparable](testing.TB, *Machine[S, O, F]) *Table[S, O, F] { panic("sketched") }
func buildProduct[S comparable](testing.TB, *Compose[S]) *Table[S, Joint, Joint]   { panic("sketched") }
func search(testing.TB, *Query) Answer                         { panic("sketched") }
func checkSet(testing.TB, *Set)                                { panic("sketched") }
func keepBindings[S, O, F comparable](in []Binding[S, O, F], _ []Action) []Binding[S, O, F] { return in }

type refinement[S, T, O2, F2 comparable] struct {
	target   *Machine[T, O2, F2]
	mapState func(S) T
}

func (refinement[S, T, O2, F2]) check(testing.TB, any) RefinementReport { panic("sketched") }

type memberClass struct {
	member string
	class  Class
}

func (m memberClass) class() (Action, []any) { return m.class.class() }
