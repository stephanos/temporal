package engine

import (
	"fmt"
	"strings"
)

// Evaluation is where a Monitor's verdict is read.
type Evaluation struct {
	kind     evaluationKind
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

// Monitor is a passive observer of a machine's steps over a finite state, as the IR's `Monitor`:
// keyNext turns its state, the key of the state before a step and the step's result into its state
// after the step, and keyViolated says whether a state violates it where At reads it. It never
// disables a step; the search keeps its state in the product state, so two histories that leave it
// in different states stay apart.
type Monitor struct {
	Name    string
	Initial string
	At      Evaluation
	// keyNext and keyViolated are the functions of a Monitor declared over a table's keys.
	keyNext     func(monitor, before string, step Result) (string, error)
	keyViolated func(monitor string) (bool, error)
}

func (m *Monitor) check() error {
	if m.keyNext == nil || m.keyViolated == nil {
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
		return e.afterKey(res)
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
