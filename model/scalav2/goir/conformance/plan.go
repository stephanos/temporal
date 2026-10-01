package conformance

import (
	"errors"
	"fmt"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/scalav2/goir"
)

// plan is everything an assessment reads, fixed when the factory was prepared: the machine's steps
// and hole rows as an index, what each claim reads on each step, and how the Case's evidence names
// the machine's facts. No Assessor changes it.
type plan struct {
	machine     string
	realization string
	limits      Limits
	reader      *reader

	states []string
	ends   []bool
	start  int32
	// steps and holes are per state, in table order.
	steps  [][]step
	holes  [][]string
	claims []*claim
}

// step is one result of one row.
type step struct {
	// index is the step's position among all the machine's steps, which the claims' readings are
	// indexed by.
	index  int
	row    string
	target int32
	// facts is the name the machine's evidence function gives each fact the step records.
	facts []string
}

// reading is what a claim's function says of one step or one monitor state.
type reading uint8

const (
	// notRead is a step the claim is not about.
	notRead reading = iota
	holds
	fails
	// unreadable is a function that reached a hole: the claim neither holds nor fails there.
	unreadable
	// malformed is a function that cannot be read as the IR says it is. Its error is the assessment's
	// when an execution takes the step.
	malformed
)

// claim is the Query's Property or one monitor of its machine, read over every step once.
type claim struct {
	id      string
	monitor bool
	// property is a Property's reading of each step.
	property []reading
	// next is a monitor's state after each step from each state it can come to, or lost. A state is
	// its position among the states found from the initial one, which is zero.
	next [][]int32
	// violated is whether each monitor state violates it.
	violated []reading
	atEnds   bool
	// read is whether the monitor's verdict is read after each step.
	read []reading
	// errs is why a reading is malformed, by the function's argument.
	errs map[[2]int]error
}

// lost is a monitor state a hole left unknown. It is no position of a state.
const lost int32 = -1

func located(at *modelirspb.Position, format string, args ...any) error {
	position := ""
	if at.GetFile() != "" {
		position = fmt.Sprintf("%s:%d", at.GetFile(), at.GetLine())
	}
	return &goir.Error{Position: position, Message: fmt.Sprintf(format, args...)}
}

// compile reads the Query's machine and claims whole, through the Query as goir binds it for a reader
// of recorded steps: the table Check reads, and the Property and monitors as Check declares them.
// Nothing of a claim is decided here. Reading the claims is work, counted against the readings
// ceiling before each reading is made.
func compile(m *modelirspb.Model, key goir.ClaimKey, source *testpilotspb.Case, limits Limits) (*plan, error) {
	realizer, err := goir.NewRealizer(m, goir.DefaultScope)
	if err != nil {
		return nil, err
	}
	declared, err := realizer.Declared(key)
	if err != nil {
		return nil, err
	}
	at := declared.Query.GetPosition()
	bound, err := realizer.Bound(key)
	if err != nil {
		var known *goir.Error
		if errors.As(err, &known) {
			return nil, err
		}
		return nil, located(at, "query %s is not assessed: %v", key.Name, err)
	}
	realization, err := realizationOf(realizer, key, declared)
	if err != nil {
		return nil, err
	}
	p := &plan{machine: bound.Table.Machine, realization: realization.GetId(), limits: limits, states: bound.Table.States}
	if p.reader, err = newReader(realization, source); err != nil {
		return nil, err
	}
	all, err := p.index(bound, at)
	if err != nil {
		return nil, err
	}
	budget := &readings{ceiling: limits.MaxReadings, query: key.Name, at: at}
	c, err := compileProperty(bound.Property, all, budget)
	if err != nil {
		return nil, err
	}
	p.claims = append(p.claims, c)
	for _, mo := range bound.Monitors {
		c, err := compileMonitor(mo, all, budget)
		if err != nil {
			return nil, err
		}
		p.claims = append(p.claims, c)
	}
	if int64(len(p.claims)) > limits.MaxProperties {
		return nil, located(at, "query %s has %d claims, its Property and the monitors of %s, and the assessment's property ceiling is %d", key.Name,
			len(p.claims), p.machine, limits.MaxProperties)
	}
	seen := map[string]bool{}
	for _, c := range p.claims {
		if seen[c.id] {
			return nil, located(at, "query %s has two claims named %s: its Property and a monitor of %s", key.Name, c.id, p.machine)
		}
		seen[c.id] = true
	}
	return p, nil
}

// realizationOf is the one realization that says how the facts of a Query's machine are recorded.
func realizationOf(realizer *goir.Realizer, key goir.ClaimKey, declared *goir.Declared) (*modelirspb.Realization, error) {
	at, machine := declared.Query.GetPosition(), declared.Scenario.GetMachine()
	var realization *modelirspb.Realization
	for _, r := range realizer.Realizations() {
		if r.GetMachine() != machine {
			continue
		}
		if realization != nil {
			return nil, located(at, "query %s runs on %s, which two realizations run; a Run is assessed through one", key.Name, machine)
		}
		realization = r
	}
	if realization == nil {
		return nil, located(at, "query %s runs on %s, which declares no realization: nothing says how its facts are recorded", key.Name, machine)
	}
	return realization, nil
}

// taken is one step as the claims' functions read it: the class and the state it leaves, and the
// result.
type taken struct {
	action, source string
	result         umpire.Result
}

// index lays the bound table's rows and unknown pairs out by state, from the Query's start, and
// returns every step in the order the claims' readings are indexed by.
func (p *plan) index(bound *goir.Bound, at *modelirspb.Position) ([]taken, error) {
	table := bound.Table
	index := make(map[string]int32, len(p.states))
	for i, state := range p.states {
		index[state] = int32(i)
	}
	p.ends = make([]bool, len(p.states))
	for _, end := range table.Ends {
		p.ends[index[end]] = true
	}
	var known bool
	if p.start, known = index[bound.Start]; !known {
		return nil, located(at, "the Query starts in %s, which is no state of %s", bound.Start, p.machine)
	}
	p.steps, p.holes = make([][]step, len(p.states)), make([][]string, len(p.states))
	for _, hole := range table.Unknown {
		from, known := index[hole.Source]
		if !known {
			return nil, located(at, "%s has a hole row %s at %s, which is no state of it", p.machine, hole.Row, hole.Source)
		}
		p.holes[from] = append(p.holes[from], hole.Row)
	}
	// A fact is recorded under the name the machine's evidence gives its constructor.
	names := map[string]string{}
	for _, line := range table.Evidence {
		names[line[0]] = line[1]
	}
	var all []taken
	for _, row := range table.Rows {
		from, known := index[row.Source]
		if !known {
			return nil, located(at, "%s has a row %s from %s, which is no state of it", p.machine, row.Key, row.Source)
		}
		for _, result := range row.Results {
			target, known := index[result.State]
			record, isRecord := result.Step.(goir.Value)
			if !known || !isRecord || record.Kind != goir.RecordValue || len(record.Fields) != 4 || record.Fields[2].Kind != goir.ListValue {
				return nil, located(at, "row %s of %s has a result that is no step into a state of it", row.Key, p.machine)
			}
			s := step{index: len(all), row: row.Key, target: target}
			for _, fact := range record.Fields[2].Items {
				s.facts = append(s.facts, names[fact.Case])
			}
			p.steps[from] = append(p.steps[from], s)
			all = append(all, taken{action: row.Action, source: row.Source, result: result})
		}
	}
	return all, nil
}

// readings counts the claim readings preparing an assessment makes, against their ceiling.
type readings struct {
	made, ceiling int
	query         string
	at            *modelirspb.Position
}

// charge counts one reading before it is made.
func (r *readings) charge() error {
	if r.made >= r.ceiling {
		return located(r.at, "reading the claims of query %s on every step takes more than the readings ceiling of %d", r.query, r.ceiling)
	}
	r.made++
	return nil
}

// decided is a bound function's answer as a reading. An error that is no hole is kept, and is the
// assessment's when an execution reaches what it was read on.
func (c *claim) decided(answer bool, err error, where [2]int) reading {
	switch {
	case err != nil && goir.Unknown(err):
		return unreadable
	case err != nil:
		c.errs[where] = err
		return malformed
	case answer:
		return holds
	default:
		return fails
	}
}

func compileProperty(p goir.BoundProperty, steps []taken, budget *readings) (*claim, error) {
	c := &claim{id: p.Name, property: make([]reading, len(steps)), errs: map[[2]int]error{}}
	for i, s := range steps {
		if !p.About(s.action) {
			continue
		}
		if err := budget.charge(); err != nil {
			return nil, err
		}
		answer, err := p.Holds(s.source, s.result)
		c.property[i] = c.decided(answer, err, [2]int{i, 0})
	}
	return c, nil
}

// compileMonitor reads a monitor from its initial state through every state a step can take it to.
func compileMonitor(mo goir.BoundMonitor, steps []taken, budget *readings) (*claim, error) {
	c := &claim{id: mo.Name, monitor: true, atEnds: mo.AtEnds, errs: map[[2]int]error{}, read: make([]reading, len(steps))}
	for j, s := range steps {
		if mo.AtEnds {
			continue
		}
		if err := budget.charge(); err != nil {
			return nil, err
		}
		answer, err := mo.Read(s.result)
		c.read[j] = c.decided(answer, err, [2]int{j, 0})
	}
	found, position := []string{mo.Initial}, map[string]int32{mo.Initial: 0}
	for i := 0; i < len(found); i++ {
		if err := budget.charge(); err != nil {
			return nil, err
		}
		answer, err := mo.Violated(found[i])
		c.violated = append(c.violated, c.decided(answer, err, [2]int{-1, i}))
		after := make([]int32, len(steps))
		for j, s := range steps {
			if err := budget.charge(); err != nil {
				return nil, err
			}
			next, err := mo.Next(found[i], s.source, s.result)
			switch {
			case err != nil && goir.Unknown(err):
				after[j] = lost
			case err != nil:
				c.errs[[2]int{j, i + 1}], after[j] = err, lost
			default:
				at, known := position[next]
				if !known {
					at = int32(len(found))
					found, position[next] = append(found, next), at
				}
				after[j] = at
			}
		}
		c.next = append(c.next, after)
	}
	return c, nil
}
