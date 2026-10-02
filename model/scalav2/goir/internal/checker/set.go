package checker

import (
	"errors"
	"fmt"
	"strings"
)

// Purpose is what a set of Queries is for.
type Purpose string

const (
	Functional  Purpose = "functional"
	Canary      Purpose = "canary"
	Exploratory Purpose = "exploratory"
)

// Binding is how a set binds a party: the Case performs the party's actions, or the world does and
// the verifier reads which class occurred.
type Binding string

const (
	Driven   Binding = "driven"
	Observed Binding = "observed"
)

// Set is a named group of Queries by purpose, binding every party but system: the Go form of the
// Lean `set` command. An exploratory set names the machine it covers, its goals and its budget
// instead of Queries.
type Set struct {
	Name     string
	Purpose  Purpose
	Bindings map[Party]Binding
	Repeat   string
	Queries  []*Query
	Machine  Model
	Cover    []CoverageGoal
	Budget   Limits
}

// Targets enumerates an exploratory set's coverage targets.
func (s *Set) Targets() ([]CoverageTarget, error) {
	if s.Purpose != Exploratory {
		return nil, errorf("set "+s.Name, "only an exploratory set enumerates targets")
	}
	return CoverageTargets(s.Machine, s.Cover, s.Budget)
}

// Check runs every semantic check the Lean elaborator runs over these declarations and returns
// every failure, each naming its declaration: machine tables (domain membership, stuck states,
// evidence for every recorded fact, Property and Scenario names declared once), refinements, compositions, Query answers, and set rules.
func Check(decls ...any) error {
	var errs []error
	for _, d := range decls {
		errs = append(errs, checkOne(d)...)
	}
	return errors.Join(errs...)
}

type evidenced interface {
	EvidenceFor(fact string) (string, bool)
}

type refining interface {
	Refinement() (*Refinement, error)
	HasRefinement() bool
}

func checkOne(d any) []error {
	switch v := d.(type) {
	case *Set:
		return checkSet(v)
	case *Query:
		return checkQuery(v)
	case Model:
		return checkModel(v)
	default:
		return []error{fmt.Errorf("umpire.Check: %T is not a declaration", d)}
	}
}

func checkQuery(q *Query) []error {
	a, err := q.Answer()
	if err != nil {
		return []error{err}
	}
	want := Found
	if q.Form == VerifyForm {
		want = VerifiedWithinLimits
	}
	if a.Outcome != want {
		return []error{errorf(q.decl(), "%s", a)}
	}
	if a.Incomplete() {
		first := a.Unknown[0]
		return []error{errorf(q.decl(), "%s is incomplete: the search explored %d unknown, the first the %s '%s'",
			a, len(a.Unknown), first.Kind, first.Row)}
	}
	return nil
}

func checkModel(m Model) []error {
	t, err := m.Table()
	if err != nil {
		return []error{err}
	}
	var errs []error
	if n, ok := m.(claimNamer); ok {
		errs = append(errs, n.claimNames().duplicates(t.Machine)...)
	}
	if t.Stuck != "" {
		errs = append(errs, errorf("machine "+t.Machine,
			"the machine reaches '%s', does not end there, and can take no step from it; either a "+
				"step is missing or '%s' belongs under Ends", t.Stuck, t.Stuck))
	}
	if e, ok := m.(evidenced); ok {
		errs = append(errs, checkEvidence(t, e)...)
	}
	if r, ok := m.(refining); ok && r.HasRefinement() {
		if _, err := r.Refinement(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// checkEvidence rejects a recorded fact whose constructor no Evidence line names.
func checkEvidence(t *Table, e evidenced) []error {
	var errs []error
	missing := map[string]bool{}
	for _, r := range t.Rows {
		for _, res := range r.Results {
			for _, f := range res.Facts {
				if _, ok := e.EvidenceFor(actionName(f)); ok || missing[f] {
					continue
				}
				missing[f] = true
				errs = append(errs, errorf("machine "+t.Machine,
					"row %s records %s, and no Evidence line names what confirms it", r.Key, f))
			}
		}
	}
	return errs
}

func checkSet(s *Set) []error {
	decl := "set " + s.Name
	var errs []error
	if s.Purpose == Exploratory {
		if s.Machine == nil || len(s.Cover) == 0 {
			errs = append(errs, errorf(decl, "an exploratory set names its machine, its goals and its budget"))
		}
		return errs
	}
	if len(s.Queries) == 0 {
		errs = append(errs, errorf(decl, "a %s set lists Queries", s.Purpose))
	}
	for _, q := range s.Queries {
		if q.Form != FindForm {
			errs = append(errs, errorf(decl, "%s verifies, and a %s set realizes only find Queries", q.Name, s.Purpose))
		}
		if s.Purpose != Canary {
			continue
		}
		// A canary runs against a deployment that performs the handler's part itself; a step on its
		// path that records nothing is a capability gap no deployment closes.
		a, err := q.Answer()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		t, _ := q.Scenario.Machine.Table()
		for _, rk := range a.Rows {
			row := t.Rows[t.rowIndex(rk)]
			silent := true
			for _, res := range row.Results {
				if len(res.Facts) > 0 {
					silent = false
				}
			}
			if silent {
				errs = append(errs, errorf(decl, "%s takes the silent step %s, a gap no deployment closes",
					q.Name, row.Action))
			}
		}
	}
	return errs
}

// HasRefinement reports whether the machine declares a refinement.
func (m *Machine[S, O, F]) HasRefinement() bool { return m.refinement != nil }

func (s *Set) String() string {
	var names []string
	for _, q := range s.Queries {
		names = append(names, q.Name)
	}
	return fmt.Sprintf("%s set %s [%s]", s.Purpose, s.Name, strings.Join(names, ", "))
}
