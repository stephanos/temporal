package model

import (
	"errors"
	"fmt"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Total is a Query's static combination count with the factors it is the product of, counted before
// any state is reached or any row is evaluated (model/SEMANTICS.md, Query totals). A pinned Scenario
// counts the Scenario machine's states times the scheduled slots within the step limit; a free one
// counts its states times its action classes, every input assignment of each, times the step limit.
// A composition counts its own state catalog and composed classes.
type Total struct {
	Free bool
	// States is the Scenario machine's or composition's whole state catalog.
	States count
	// Classes is a free Scenario's action classes.
	Classes count
	// Steps is the Query's step limit, and Schedule a pinned Scenario's scheduled actions.
	Steps, Schedule int64
	Count           count
}

// N is the count, and whether it fits an int64.
func (t Total) N() (int64, bool) { return t.Count.n, !t.Count.overflow }

func (t Total) String() string {
	if t.Free {
		return fmt.Sprintf("%s states × %s classes × %d steps = %s", t.States, t.Classes, t.Steps, t.Count)
	}
	return fmt.Sprintf("%s states × %d scheduled slots (the least of %d steps and %d scheduled actions) = %s",
		t.States, min(t.Steps, t.Schedule), t.Steps, t.Schedule, t.Count)
}

func (c count) String() string {
	if c.overflow {
		return fmt.Sprintf("more than %d", c.n)
	}
	return fmt.Sprint(c.n)
}

// QueryTotal is the static combination count of one Query of a Model Validate admits.
func QueryTotal(m *umpirespb.Model, q *umpirespb.Query) (Total, error) {
	return newValidator(m).total(m, q)
}

// RequireTotals refuses each Query of an admitted Model that asserts no total, with the count it
// would assert. IR lifted before the assertion existed omits it; the lifter requires it of every
// Query a current Model declares, and so does whoever generates Cases from that Model.
func RequireTotals(m *umpirespb.Model) error {
	v := newValidator(m)
	var errs []error
	for _, q := range m.GetQueries() {
		if q.GetTotal() != nil {
			continue
		}
		t, err := v.total(m, q)
		if err != nil {
			errs = append(errs, errorAt(q.GetPosition(), "query %s declares no total, and its count is unknown: %v", q.GetName(), err))
			continue
		}
		errs = append(errs, errorAt(q.GetPosition(), "query %s declares no total: its static combination count is %s", q.GetName(), t))
	}
	return errors.Join(errs...)
}

// WithTotals is the Model with each Query that asserts a total asserting its count instead, as an
// exploration recounts the candidate it derives by changing a Scenario's schedule. The source Query's
// assertion is the author's and is never rewritten; only the derived candidate's is.
func WithTotals(m *umpirespb.Model) (*umpirespb.Model, error) {
	out := proto.CloneOf(m)
	v := newValidator(out)
	for _, q := range out.GetQueries() {
		if q.GetTotal() == nil {
			continue
		}
		t, err := v.total(out, q)
		if err != nil {
			return nil, err
		}
		n, ok := t.N()
		if !ok {
			return nil, errorAt(q.GetPosition(), "query %s: its static combination count is %s, more than an int64 holds", q.GetName(), t)
		}
		q.Total = wrapperspb.Int64(n)
	}
	return out, nil
}

// WithoutTotals is the Model as an identity reads it: with no Query's total, which is metadata no
// fingerprint, answer, lowering or exploration identity reads, so a corrected total changes none.
func WithoutTotals(m *umpirespb.Model) *umpirespb.Model {
	out := proto.CloneOf(m)
	for _, q := range out.GetQueries() {
		q.Total = nil
	}
	return out
}

// totals checks every Query that asserts a total against its count. It runs once the rest of the
// Model admits, so the Scenario and its machine exist.
func (v *validator) totals(m *umpirespb.Model) {
	for _, q := range m.GetQueries() {
		if q.GetTotal() == nil {
			continue
		}
		at, declared := q.GetPosition(), q.GetTotal().GetValue()
		t, err := v.total(m, q)
		switch {
		case err != nil:
			v.report(at, "query %s declares a total of %d, and its count is unknown: %v", q.GetName(), declared, err)
		case declared < 0:
			v.report(at, "query %s declares a total of %d, below 0: its static combination count is %s", q.GetName(), declared, t)
		case t.Count.overflow:
			v.report(at, "query %s declares a total of %d, and its static combination count is %s, more than an int64 holds", q.GetName(), declared, t)
		case declared != t.Count.n:
			v.report(at, "query %s declares a total of %d, and its static combination count is %s", q.GetName(), declared, t)
		default:
		}
	}
}

func (v *validator) total(m *umpirespb.Model, q *umpirespb.Query) (Total, error) {
	ref := q.GetScenario()
	var s *umpirespb.Scenario
	for _, candidate := range m.GetScenarios() {
		if candidate.GetMachine() == ref.GetMachine() && candidate.GetName() == ref.GetName() {
			s = candidate
		}
	}
	if s == nil {
		return Total{}, fmt.Errorf("no Scenario %s of %s", ref.GetName(), ref.GetMachine())
	}
	t := Total{Free: s.GetFree(), Steps: max(int64(q.GetLimits().GetSteps()), 0)}
	var state string
	var classes func() (count, error)
	switch c, mm := v.compositions[s.GetMachine()], v.machines[s.GetMachine()]; {
	case c != nil:
		state, classes = c.GetStateType(), func() (count, error) { return v.composedClassCount(c) }
		t.Schedule = int64(len(s.GetActions()) + len(s.GetKeys()))
	case mm != nil:
		state, classes = mm.GetStateType(), func() (count, error) { return v.in.boundClasses(mm, v.actions) }
		t.Schedule = int64(len(s.GetActions()))
	default:
		return Total{}, fmt.Errorf("no machine or composition %s", s.GetMachine())
	}
	states, err := v.in.size(named(state))
	if err != nil {
		return Total{}, err
	}
	t.States = states
	if !t.Free {
		t.Count = states.times(count{n: min(t.Steps, t.Schedule)})
		return t, nil
	}
	if t.Classes, err = classes(); err != nil {
		return Total{}, err
	}
	t.Count = states.times(t.Classes).times(count{n: t.Steps})
	return t, nil
}
