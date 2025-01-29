package propcheck

import (
	"fmt"

	"pgregory.net/rapid"
)

type (
	Error interface {
		propcheck_error_message() string
	}
	tryAgainErr struct {
	}
	invalidErr struct {
	}
	failErr struct {
	}
	Var[T any] struct {
		label           string
		defaultValueGen *rapid.Generator[T]
	}
	Model struct {
		label string
	}
	Init struct {
		varLabel string
		value    any
	}
	Transition struct {
		model  *Model
		label  string
		action func(*Run) Error
	}
	Prop struct {
		model *Model
		label string
		check func(*Run) Error
	}
	Set[T comparable] struct {
		label     string
		variants  map[T]int
		sampleGen *rapid.Generator[T]
	}
	Run struct {
		*rapid.T
		state map[string]any
	}
)

func Check(
	t *rapid.T,
	inits []*Init,
	transitions []*Transition,
	props []*Prop,
) {
	r := &Run{
		T:     t,
		state: make(map[string]any),
	}

	// initialize variables
	for _, init := range inits {
		r.state[init.varLabel] = init.value
	}

	// define actions
	actions := map[string]func(*rapid.T){}
	for _, transition := range transitions {
		actions["transition:"+transition.label] = func(t *rapid.T) {
			r.T = t
			transition.action(r)
		}
	}
	for _, prop := range props {
		actions["prop:"+prop.label] = func(t *rapid.T) {
			r.T = t
			prop.check(r)
		}
	}

	rapid.Check(t, func(t *rapid.T) {
		t.Repeat(actions)
	})
}

func NewModel(label string) *Model {
	return &Model{
		label: label,
	}
}

func NewVar[T any](model *Model, label string, defaultValueGen *rapid.Generator[T]) *Var[T] {
	return &Var[T]{
		label:           label,
		defaultValueGen: defaultValueGen,
	}
}

func (v *Var[T]) Get(run *Run) T {
	val, exists := run.state[v.label]
	if !exists {
		val = v.defaultValueGen.Draw(run.T, v.label)
		run.state[v.label] = val
	}
	typedVal, ok := val.(T)
	if !ok {
		panic(fmt.Sprintf("type mismatch: found %T but expected %T", val, typedVal))
	}
	return typedVal
}

func (v *Var[T]) Set(run *Run, val T) {
	run.state[v.label] = val
}

func NewSet[T comparable](label string, variants map[T]int) *Set[T] {
	sum := 0
	for _, prob := range variants {
		sum += prob
	}
	if sum < 99 || sum > 100 {
		panic("sum of probabilities must be between 100 and 99")
	}

	return &Set[T]{
		label:    label,
		variants: variants,
		sampleGen: rapid.Custom(func(t *rapid.T) T {
			rand := rapid.IntRange(0, 99).Draw(t, "")
			for val, prob := range variants {
				if rand < prob {
					return val
				}
			}
			panic("unreachable")
		}),
	}
}

func (s *Set[T]) OneOf(r *Run) T {
	return rapid.OneOf(s.sampleGen).Draw(r.T, s.label)
}

func (r *Run) TryAgain() Error {
	return &tryAgainErr{}
}

func (r *Run) Fail(err error) Error {
	return &failErr{}
}

func (r *Run) Invalid(s string) Error {
	return &invalidErr{}
}

func NewInit[T any](v *Var[T], value T) *Init {
	return &Init{
		varLabel: v.label,
		value:    value,
	}
}

func (e tryAgainErr) propcheck_error_message() string {
	return ""
}

func (f failErr) propcheck_error_message() string {
	return ""
}

func (f invalidErr) propcheck_error_message() string {
	return ""
}

func NewTransition(
	model *Model,
	label string,
	action func(*Run) Error,
) *Transition {
	t := &Transition{
		model:  model,
		label:  label,
		action: action,
	}
	return t
}

func NewProp(
	model *Model,
	label string,
	check func(*Run) Error,
) *Prop {
	return &Prop{
		model: model,
		label: label,
		check: check,
	}
}
