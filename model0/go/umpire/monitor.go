package umpire

import (
	"fmt"
	"strings"
)

// Evaluation is where a Monitor's verdict is read.
type Evaluation struct {
	kind     evaluationKind
	after    func(Result) bool
	afterKey func(Result) (bool, error)
}

type evaluationKind int

const (
	everyStep evaluationKind = iota
	atEnds
	afterSteps
)

// EveryStep reads a Monitor's verdict after every step.
func EveryStep() Evaluation { return Evaluation{kind: everyStep} }

// AtEnds reads a Monitor's verdict after a step into a state the machine may end in.
func AtEnds() Evaluation { return Evaluation{kind: atEnds} }

// After reads a Monitor's verdict after each step whose result f accepts.
func After(f func(Result) bool) Evaluation { return Evaluation{kind: afterSteps, after: f} }

// Monitor is a passive observer of a machine's steps over a finite state, as the IR's `Monitor`:
// Next turns its state, the typed state before a step and the step's result into its state after
// the step, and Violated says whether a state violates it where At reads it. It never disables a
// step; the search keeps its state in the product state, so two histories that leave it in
// different states stay apart.
type Monitor struct {
	Name     string
	Initial  string
	Next     func(monitor string, before any, step Result) (string, error)
	Violated func(monitor string) bool
	At       Evaluation
	err      error
	// keyNext and keyViolated are the functions of a Monitor declared over a table's keys.
	keyNext     func(monitor, before string, step Result) (string, error)
	keyViolated func(monitor string) (bool, error)
}

// next is the Monitor's state after a step from the state keyed state, whose typed value is before.
func (m *Monitor) next(key, state string, before any, res Result) (string, error) {
	if m.keyNext != nil {
		return m.keyNext(key, state, res)
	}
	return m.Next(key, before, res)
}

func (m *Monitor) violated(key string) (bool, error) {
	if m.keyViolated != nil {
		return m.keyViolated(key)
	}
	return m.Violated(key), nil
}

// NewMonitor declares a Monitor with a finite state type M over a machine whose steps have type
// Step[S, O, F].
func NewMonitor[M, S, O, F any](name string, initial M, next func(M, S, Step[S, O, F]) M, violated func(M) bool,
	at Evaluation) *Monitor {
	decl := "monitor " + name
	states, err := DomainOf[M]()
	if err != nil {
		return &Monitor{Name: name, err: errorf(decl, "state type: %v", err)}
	}
	byKey := make(map[string]M, len(states))
	for _, s := range states {
		byKey[KeyOf(s)] = s
	}
	if _, ok := byKey[KeyOf(initial)]; !ok {
		return &Monitor{Name: name, err: errorf(decl, "the initial state %s is outside the state domain", KeyOf(initial))}
	}
	return &Monitor{Name: name, Initial: KeyOf(initial), At: at,
		Next: func(key string, before any, res Result) (string, error) {
			s, isState := before.(S)
			step, isStep := res.Step.(Step[S, O, F])
			if !isState || !isStep {
				return "", errorf(decl, "the step into %s is not a step of a %T", res.State, s)
			}
			out := KeyOf(next(byKey[key], s, step))
			if _, ok := byKey[out]; !ok {
				return "", errorf(decl, "the step into %s moves it to %s, which is outside the state domain", res.State, out)
			}
			return out, nil
		},
		Violated: func(key string) bool { return violated(byKey[key]) },
	}
}

func (m *Monitor) check() error {
	if m.err != nil {
		return m.err
	}
	if m.keyNext != nil && m.keyViolated != nil {
		return nil
	}
	if m.Next == nil || m.Violated == nil {
		return errorf("monitor "+m.Name, "a Monitor names its next and violated functions")
	}
	return nil
}

// reads reports whether the verdict is read after a step into res from a table whose ends are
// ends.
func (e Evaluation) reads(res Result, ends map[string]bool) (bool, error) {
	switch e.kind {
	case atEnds:
		return ends[res.State], nil
	case afterSteps:
		if e.afterKey != nil {
			return e.afterKey(res)
		}
		return e.after(res), nil
	default:
		return true, nil
	}
}

// Verdict is what a Monitor says of a path.
type Verdict string

const (
	MonitorHeld     Verdict = "held"
	MonitorViolated Verdict = "violated"
	// MonitorUnread is a Monitor whose evaluation point the path never reaches: no verdict.
	MonitorUnread Verdict = "unread"
	// MonitorUnknown is a Monitor whose functions could not be read on the witness's last step.
	MonitorUnknown Verdict = "unknown"
)

// MonitorVerdict is one Monitor's verdict on a witness, with the state the witness leaves it in. A
// Query verified within its limits has no witness; its verdict is held when some explored step read
// the Monitor and unread otherwise, with no state.
type MonitorVerdict struct {
	Name    string
	State   string
	Verdict Verdict
}

// monitorState is one Monitor's part of a product state.
type monitorState struct {
	key            string
	read, violated bool
	// unknown marks a Monitor whose functions could not be read on the step into this state: key is
	// its state before that step.
	unknown bool
}

func (m monitorState) verdict() Verdict {
	switch {
	case m.violated:
		return MonitorViolated
	case m.unknown:
		return MonitorUnknown
	case m.read:
		return MonitorHeld
	default:
		return MonitorUnread
	}
}

// identity spells the Monitors' part of a product state. Each key is prefixed with its length, so no
// spelling of one Monitor's state reads as part of another's.
func identity(mons []monitorState) string {
	var b strings.Builder
	for _, m := range mons {
		fmt.Fprintf(&b, "%d:%s%s%s", len(m.key), m.key, bit(m.read), bit(m.violated))
		if m.unknown {
			b.WriteString("?")
		}
	}
	return b.String()
}

func bit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Watch adds Monitors to the Query. In a verify, a Monitor violated where it is read is a
// counterexample; in a find, it takes nothing away from the witness and reports its verdict.
func (q *Query) Watch(monitors ...*Monitor) *Query {
	q.monitors = append(q.monitors, monitors...)
	return q
}

func (q *Query) checkMonitors() error {
	seen := map[string]bool{}
	for _, m := range q.monitors {
		if m.Name == "" {
			return errorf(q.decl(), "a monitor has no name")
		}
		if seen[m.Name] {
			return errorf(q.decl(), "two monitors are named %s", m.Name)
		}
		seen[m.Name] = true
		if err := m.check(); err != nil {
			return wrapError(q.decl(), err)
		}
	}
	return nil
}
