// Package lower lowers a find Query of an admitted IR Model into a Testpilot Case, through the
// realization the Model declares for the Query's machine.
//
// A Case is built here and nowhere else for a Model the IR carries: the front end declares the
// realization, and this package places a Query's witness in it. It decides nothing about the feature.
// Which commands a Case carries follows from the classes the Scenario pins; what its Contract requires
// follows from the Property; what a declaration needs that Testpilot cannot run yet is reported, with
// the task that owns it, instead of a Case.
package lower

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Standing is what lowering a Query came to.
type Standing string

const (
	// Lowered is a find Query whose witness became a Case.
	Lowered Standing = "lowered"
	// NothingToRealize is a verify Query: it is searched and verified, and realizes nothing.
	NothingToRealize Standing = "nothing-to-realize"
	// NoRealization is a find Query whose machine declares no realization.
	NoRealization Standing = "no-realization"
	// NotSupported is a find Query whose realization declares what Testpilot cannot run yet.
	NotSupported Standing = "unsupported"
)

// Unsupported is one declaration of a realization that Testpilot has no primitive for: what it is,
// where it was written, and the task that owns the primitive, or that none does.
type Unsupported struct {
	Construct string
	ID        string
	Position  string
	Owner     string
	Why       string
}

// What this package has nothing to lower into is owned by no task: it is a limit of the prototype,
// and is said to be one.
const ownerNone = "none: a recorded limit of the prototype"

// Disposition is what became of one declaration of a realization in one Case.
type Disposition string

const (
	// InCase is a declaration the Case carries, in the parts As names.
	InCase Disposition = "in-case"
	// OffPath is a command the Query's path does not perform, or a kind of evidence the Case does not
	// carry: one that confirms no step of the path and is not exhaustive.
	OffPath Disposition = "off-path"
	// Names is what identifies the realization in the IR, which no part of a Case repeats.
	Names Disposition = "names"
	// Unread is a hint of the API behavior, or a server step, which shapes only how a Case waits, that
	// no wait of the Case reads: a read that writes its own interval, or one whose window it is not in.
	Unread Disposition = "unread"
)

// Entry is one declaration of a realization and what became of it. Kind is the field of the
// realization that declares it, `realization` for a field of the realization itself, or `command`;
// ID is the declaration's id, or the field's name. As names the parts of the Case that carry it.
type Entry struct {
	Kind        string
	ID          string
	Position    string
	Disposition Disposition
	As          []string
}

// Lowering is what lowering one Query came to: its standing, the Case of a lowered one with every
// declaration accounted for, and the gaps of an unsupported one. OffPath is what the realization
// declares that Testpilot cannot run and this Query's path does not take: it stands in the way of
// other Queries, and of this one not at all.
type Lowering struct {
	Standing    Standing
	Case        *testpilotspb.Case
	Unsupported []Unsupported
	OffPath     []Unsupported
	Inventory   []Entry
}

// Producer lowers the Queries of one admitted, checked Model.
type Producer struct {
	realizer *check.Realizer
	// found is each Query's receipt by its name, which a Model gives one Query.
	found map[string]check.Receipt
	// realizations is the realizations the Model declares.
	realizations []*umpirespb.Realization
}

// NewProducer admits a Model, binds it and answers its Queries once, for every Case lowered from it.
func NewProducer(m *umpirespb.Model) (*Producer, error) {
	realizer, err := check.NewRealizer(m, check.DefaultScope)
	if err != nil {
		return nil, err
	}
	p := &Producer{realizer: realizer, found: map[string]check.Receipt{}, realizations: realizer.Realizations()}
	for _, r := range check.Check(m, check.DefaultScope).Receipts {
		if r.Subject == check.QuerySubject {
			p.found[r.Key.Name] = r
		}
	}
	return p, nil
}

// asked is one Query with the declarations it names.
type asked struct {
	key      check.ClaimKey
	q        *umpirespb.Query
	property *umpirespb.Property
	scenario *umpirespb.Scenario
	r        *umpirespb.Realization
}

// ask finds a Query and what it is before anything of it is checked: a verify Query, a find Query
// with no realization, or a find Query with the one realization of its machine. The Query is the one
// Check gave a receipt, resolved by that receipt's key.
func (p *Producer) ask(query string) (*asked, Standing, error) {
	receipt, ok := p.found[query]
	if !ok {
		// The Realizer's refusal names the Model the Query is missing from.
		if _, err := p.realizer.Declared(check.ClaimKey{Name: query}); err != nil {
			return nil, "", err
		}
		return nil, "", &interp.Error{Message: "no Query " + query}
	}
	declared, err := p.realizer.Declared(receipt.Key)
	if err != nil {
		return nil, "", err
	}
	a := &asked{key: receipt.Key, q: declared.Query, property: declared.Property, scenario: declared.Scenario}
	standing, r, err := Realizable(a.q, a.scenario, p.realizations)
	if err != nil {
		return nil, "", err
	}
	a.r = r
	return a, standing, nil
}

// Realizable is what a Query is before anything of it is checked: a verify Query realizes nothing, a
// find Query whose Scenario's machine no realization runs has no realization, and a find Query with
// the one realization of its machine is lowered through it. It is the one place that is decided, for
// the manifest and for model lint alike.
func Realizable(q *umpirespb.Query, scenario *umpirespb.Scenario, realizations []*umpirespb.Realization) (Standing, *umpirespb.Realization, error) {
	if q.GetForm() != umpirespb.Query_FORM_FIND {
		return NothingToRealize, nil, nil
	}
	var running []*umpirespb.Realization
	for _, r := range realizations {
		if r.GetMachine() == scenario.GetMachine() {
			running = append(running, r)
		}
	}
	switch len(running) {
	case 0:
		return NoRealization, nil, nil
	case 1:
		return Lowered, running[0], nil
	default:
		return "", nil, errorAt(q.GetPosition(), "query %s runs on %s, which %d realizations run; a Query is lowered through one",
			q.GetName(), scenario.GetMachine(), len(running))
	}
}

// standingOf is the one place the standing of a Query is decided. An error of the realization or of
// the Query's path comes first, every one of them, whatever else is true of the Query. Then what
// Testpilot cannot run: a gap is reported only of a realization that is otherwise sound. Then what
// the Query is: one that realizes nothing or has no realization has that standing, and any other is
// lowered.
func standingOf(problems []error, gaps []Unsupported, realizable Standing) (Standing, error) {
	switch {
	case len(problems) > 0:
		return "", errors.Join(problems...)
	case len(gaps) > 0:
		return NotSupported, nil
	default:
		return realizable, nil
	}
}

// Lower lowers one Query under a Case identity. A Query that realizes nothing and one whose machine
// declares no realization have that standing and no Case. One whose realization needs what Testpilot
// cannot run lists every such declaration and has no Case: nothing is lowered around a gap. The
// realization and the Query's path are checked whole before either is said.
func (p *Producer) Lower(query string, identity Identity) (*Lowering, error) {
	a, realizable, err := p.ask(query)
	if err != nil {
		return nil, err
	}
	var l *lowering
	var problems []error
	var gaps, off []Unsupported
	if realizable == Lowered {
		// A realization or a path that cannot be read at all is an error, and has no gaps to list.
		if l, problems = p.check(a, identity); l != nil {
			gaps, off = p.gaps(a.r, l.takes)
			gaps = append(append(gaps, l.unanswered()...), l.late()...)
		}
	}
	standing, err := standingOf(problems, gaps, realizable)
	if err != nil {
		return nil, err
	}
	if standing != Lowered {
		return &Lowering{Standing: standing, Unsupported: gaps, OffPath: off}, nil
	}
	produced, err := cp.Produce(l.query, identity, l.realization, p.source(a.q))
	if err != nil {
		return nil, errorAt(a.q.GetPosition(), "query %s: %v", query, err)
	}
	if err := l.ordered(produced); err != nil {
		return nil, err
	}
	inventory, err := l.inventory(produced)
	if err != nil {
		return nil, err
	}
	return &Lowering{Standing: Lowered, Case: produced, OffPath: off, Inventory: inventory}, nil
}

// gaps lists every declaration of a realization that Testpilot has nothing for, in the order the IR
// lists them: the ones in a Query's way, and the ones off its path. A kind of evidence and a control
// are the realization's, and stand in the way of every Query of it. An authored monitor of the
// machine is no gap: a Case's Contract carries none, and the prepared assessment reads each beside
// the Contract (tools/umpire/conformance). The record of an attempt stands in the way of the Queries whose Case runs two
// activities: the Run's record names neither. A command stands in the way of the Queries whose Case would carry it,
// which takes says, and is off the path of the others.
func (p *Producer) gaps(r *umpirespb.Realization, takes func(*umpirespb.Item, *umpirespb.Performance) bool) (out, off []Unsupported) {
	for _, e := range r.GetEvidence() {
		out = append(out, evidenceGaps(e)...)
	}
	out = append(out, indistinct(r, takes)...)
	for _, c := range r.GetControls() {
		if !heldByDriver(c) {
			out = append(out, Unsupported{Construct: "hold-delivery control", ID: c.GetId(), Position: locate(c.GetPosition()), Owner: ownerNone,
				Why: "a Driver holds what a step dispatched to a task queue; none holds the deliveries of a channel"})
		}
	}
	command := func(s *umpirespb.Script, item *umpirespb.Item, performance *umpirespb.Performance) {
		c := item.GetCommand()
		if performance != nil {
			c = performance.GetCommand()
		}
		gap, unsupported := commandGap(r, s, c)
		switch {
		case !unsupported:
		case takes(item, performance):
			out = append(out, gap)
		default:
			off = append(off, gap)
		}
	}
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			if item.GetCommand() != nil {
				command(s, item, nil)
			}
			for _, performance := range item.GetPerforms() {
				command(s, item, performance)
			}
		}
	}
	return out, off
}

// indistinct is the records of attempts a Case of a Query cannot tell apart: every one, where the Case
// runs two activities. A Run records an attempt at the command that carries it, and the one carrier of
// a Case reserves every activity entrypoint the Case gives an instruction, so the records of all of
// them are that command's, and none names its script.
func indistinct(r *umpirespb.Realization, takes func(*umpirespb.Item, *umpirespb.Performance) bool) (out []Unsupported) {
	var activities []string
	for _, s := range r.GetScripts() {
		if s.GetActivity() == nil {
			continue
		}
		if slices.ContainsFunc(s.GetItems(), func(item *umpirespb.Item) bool {
			return item.GetCommand() != nil && takes(item, nil) ||
				slices.ContainsFunc(item.GetPerforms(), func(performance *umpirespb.Performance) bool { return takes(item, performance) })
		}) {
			activities = append(activities, s.GetId())
		}
	}
	if len(activities) < 2 {
		return nil
	}
	for _, e := range r.GetEvidence() {
		of := e.GetRunEvent().GetAttempt()
		if of == nil {
			continue
		}
		at := of.GetPosition()
		if at.GetFile() == "" {
			at = e.GetPosition()
		}
		out = append(out, Unsupported{Construct: "attempt record among several activities", ID: e.GetId(), Position: locate(at), Owner: ownerNone,
			Why: fmt.Sprintf("a Run records an attempt at the command that carries it, by its number and under no script's name, and one command of a Case "+
				"carries every activity the Case runs: the attempts of scripts %s are not told apart", strings.Join(activities, " and "))})
	}
	return out
}

// heldByDriver is whether a control is one a Driver realizes: the hold of what a step dispatched to
// a task queue.
func heldByDriver(c *umpirespb.Control) bool {
	return c.GetHoldDispatched() != nil && c.GetRole() != ""
}

// controlOf is the control of a realization a hold or a release names, by id.
func controlOf(r *umpirespb.Realization, c *umpirespb.Command) *umpirespb.Control {
	id := c.GetHold()
	if _, releases := c.GetInstruction().(*umpirespb.Command_Release); releases {
		id = c.GetRelease()
	}
	for _, declared := range r.GetControls() {
		if declared.GetId() == id {
			return declared
		}
	}
	return nil
}

// commandGap is what a Case has no instruction for in a command, if anything: the hold or the
// release of a control no Driver realizes.
func commandGap(r *umpirespb.Realization, s *umpirespb.Script, c *umpirespb.Command) (Unsupported, bool) {
	switch c.GetInstruction().(type) {
	case *umpirespb.Command_Hold, *umpirespb.Command_Release:
		if heldByDriver(controlOf(r, c)) {
			return Unsupported{}, false
		}
		return Unsupported{Construct: "hold-delivery command", ID: s.GetId() + "/" + c.GetId(), Position: locate(c.GetPosition()),
			Owner: ownerNone, Why: "no instruction holds or releases the deliveries of a channel"}, true
	default:
		return Unsupported{}, false
	}
}

// evidenceGaps is what a Case cannot carry of one kind of evidence, in the order it is declared.
func evidenceGaps(e *umpirespb.Evidence) (out []Unsupported) {
	// A Run records a durable commit of the receiver only as the record of the instruction that
	// observed it, a release that delivered to it: no RPC a caller makes and no history reports one.
	if e.GetCommitment() == umpirespb.Evidence_COMMITMENT_DURABLE && e.GetRunEvent() == nil {
		out = append(out, Unsupported{Construct: "durable-commit observation", ID: e.GetId(), Position: locate(e.GetPosition()), Owner: ownerNone,
			Why: "what an RPC returned and what history holds report no durable commit of the receiver: a Run records one only as the record of the release that observed it"})
	}
	for _, f := range e.GetFields() {
		if !f.GetRedacted() {
			continue
		}
		at := f.GetPosition()
		if at.GetFile() == "" {
			at = e.GetPosition()
		}
		out = append(out, Unsupported{Construct: "redacted evidence field", ID: e.GetId() + "/" + f.GetId(), Position: locate(at), Owner: ownerNone,
			Why: "a lift reads a value for every field its evidence declares, so no evidence carries a field without its value"})
	}
	return out
}

// attemptClasses is the classes an activity's script starts an attempt with, and the classes its
// commands answer one with, by key.
func (l *lowering) attemptClasses(s *umpirespb.Script) (started, answered map[string]bool) {
	started, answered = map[string]bool{}, map[string]bool{}
	for _, class := range s.GetActivity().GetStarts() {
		started[l.adapter.classKey(class)] = true
	}
	for _, item := range s.GetItems() {
		if item.GetCommand().GetAttemptWithheld() != nil {
			for _, timer := range item.GetWhen() {
				answered[l.adapter.classKey(timer)] = true
			}
		}
		for _, performance := range item.GetPerforms() {
			if performance.GetCommand().GetAttemptHeartbeat() == nil {
				answered[l.adapter.classKey(performance.GetStep())] = true
			}
		}
	}
	return started, answered
}

// An on-path item emits one withholding instruction, not one per occurrence of its timer. Refuse
// a repeated selected timer rather than counting two attempt ends for that one instruction.
func (l *lowering) withholdingOccurrences() (problems []error) {
	for _, script := range l.a.r.GetScripts() {
		for _, item := range script.GetItems() {
			command := item.GetCommand()
			if command.GetAttemptWithheld() == nil {
				continue
			}
			key := l.adapter.classKey(item.GetWhen()[0])
			count := 0
			for _, taken := range l.keys {
				if taken == key {
					count++
				}
			}
			external := command.GetAttemptWithheld().GetExternalSettlement()
			if external != "" && count != 1 {
				problems = append(problems, errorAt(command.GetPosition(), "withholding command %s requires exactly one occurrence of its external answer; got %d", command.GetId(), count))
			} else if count > 1 {
				problems = append(problems, errorAt(command.GetPosition(),
					"withholding command %s requires exactly one occurrence of its armed timer; got %d", command.GetId(), count))
			}
			if external != "" && count == 1 {
				starts, _ := l.attemptClasses(script)
				var delivered int64
				for _, taken := range l.keys {
					if starts[taken] {
						delivered++
					}
					if taken == key {
						break
					}
				}
				for _, e := range l.a.r.GetExternalSettlements() {
					if e.GetAnswer() == external && e.GetAttempt() != delivered {
						problems = append(problems, errorAt(e.GetPosition(), "external settlement %s declares attempt %d; selected publication is attempt %d", external, e.GetAttempt(), delivered))
					}
				}
			}
		}
	}
	return problems
}

// unanswered is the attempts of an activity that a path starts and gives no answer: for each activity
// script, how many more steps of the path are a delivery the script starts with than are performed by
// one of its commands. An activity entrypoint's instructions are its attempts' answers, and none waits,
// so an attempt the path leaves to run out a deadline has nothing to be lowered to.
func (l *lowering) unanswered() (out []Unsupported) {
	for _, s := range l.a.r.GetScripts() {
		if s.GetActivity() == nil {
			continue
		}
		started, answered := l.attemptClasses(s)
		starts, answers := 0, 0
		for _, key := range l.keys {
			if started[key] {
				starts++
			}
			if answered[key] {
				answers++
			}
		}
		if starts > answers {
			out = append(out, Unsupported{Construct: "attempt that gives no answer", ID: s.GetId(), Position: locate(s.GetPosition()), Owner: ownerNone,
				Why: fmt.Sprintf("the path starts %d attempts of the activity and answers %d: an activity entrypoint's instructions are answers, and none waits out a deadline",
					starts, answers)})
		}
	}
	return out
}

// When a Run records each piece of a path's evidence is said here and nowhere else, and it is read
// from declarations: nothing is inferred from what kind of event evidence is. Admission sees to it
// that what a worker reports of an activation is always declared the record of an attempt.
//
//   - Evidence that is the record of an attempt (a Run Event source that names the attempt, and the
//     activity script it is an attempt of) reaches a Run once that attempt is answered: with the step
//     of the path that is the script's answer of that number, and before the evidence of that step.
//   - Every other evidence is recorded by an instruction of the Case's controller, and reaches a Run
//     as the controller runs the instruction: the command a Run Event source names, the poll that
//     reads a kind, the read that lifts it. ordered checks, on the Case, that the controller runs those
//     instructions in the order the path records their evidence.
//
// Both read the path as the order its steps are taken in. That a Run takes them in that order is what
// the Case's own commands bring about, and no reading of evidence can establish it.
//
// Two orders are the runtime's and no declaration fixes them: that the completion of the call that
// carries an attempt is recorded before the attempt's record, and that an attempt's record is recorded
// before the status a later instruction reads after the attempt's answer. A Run that breaks either
// carries evidence its Contract refuses, and is incomplete: it is given no Verdict it has not earned.

// published is one kind of evidence on a path, as the producer confirms the path by it: the place of
// the last step it confirms, and, where it is the record of an attempt, the activity script and the
// attempt it is of.
type published struct {
	kind   string
	last   int
	script string
	number int64
}

// clash is how the record of an attempt stands against the kind beside it on the path.
type clash int

const (
	// recordedLate is a record a Run records after the other kind's evidence, though it confirms
	// earlier steps.
	recordedLate clash = iota + 1
	// recordedEarly is a record a Run records before the other kind's evidence, though it confirms
	// later steps.
	recordedEarly
	// recordedTwice is a record of the attempt the other kind is the record of: the Run records the
	// attempt as one Run Event, which is evidence of one kind.
	recordedTwice
)

// misplaced is the record of an attempt that a Run would not record in the path's order, the kind
// beside it on the path that it clashes with, and how.
type misplaced struct {
	kind, beside string
	how          clash
}

// outOfOrder is the first record of an attempt that a Run would not record in the path's order, or
// nil. The kinds are in path order, and answers is, for each activity script, the places on the path
// of the steps that answer its attempts, in order. Evidence the controller records for a step is
// recorded after that step and before the next; the record of an attempt is recorded with the
// attempt's answer, before that step's own evidence. Two kinds recorded at one moment are two records
// of one attempt, which the Run records once. A record of an attempt the path does not answer is
// recorded at no step, and is in no order with the rest.
func outOfOrder(kinds []published, answers map[string][]int) *misplaced {
	// Each step is two moments: what is recorded with it, and what is recorded after it.
	type moment struct {
		of   published
		when int
	}
	var moments []moment
	for _, k := range kinds {
		switch {
		case k.script == "":
			moments = append(moments, moment{k, 2*k.last + 1})
		case k.number >= 1 && k.number <= int64(len(answers[k.script])):
			moments = append(moments, moment{k, 2 * answers[k.script][k.number-1]})
		default:
		}
	}
	for i := 1; i < len(moments); i++ {
		earlier, later := moments[i-1], moments[i]
		switch {
		case earlier.when < later.when:
		case earlier.when == later.when:
			return &misplaced{kind: later.of.kind, beside: earlier.of.kind, how: recordedTwice}
		case earlier.of.script != "":
			return &misplaced{kind: earlier.of.kind, beside: later.of.kind, how: recordedLate}
		default:
			return &misplaced{kind: later.of.kind, beside: earlier.of.kind, how: recordedEarly}
		}
	}
	return nil
}

// attemptOf is the attempt a kind of evidence is declared the record of, or nil.
func (l *lowering) attemptOf(kind string) *umpirespb.AttemptOf {
	return l.adapter.evidence[kind].GetRunEvent().GetAttempt()
}

// unstarted is, for each record of an attempt that confirms a step of the path, the error of a path
// that starts fewer attempts of the record's script than the record's number: such a record is of
// nothing a Run of the path could record.
func (l *lowering) unstarted() (problems []error) {
	for _, confirmed := range l.confirmations {
		of := l.attemptOf(confirmed.Source.KindID)
		if of == nil {
			continue
		}
		for _, s := range l.a.r.GetScripts() {
			if s.GetId() != of.GetScript() {
				continue
			}
			started, _ := l.attemptClasses(s)
			starts := int64(0)
			for _, key := range l.keys {
				if started[key] {
					starts++
				}
			}
			if of.GetNumber() > starts {
				problems = append(problems, errorAt(l.adapter.evidence[confirmed.Source.KindID].GetPosition(),
					"evidence %s is the record of attempt %d of script %s, and the path of query %s starts %d", confirmed.Source.KindID, of.GetNumber(),
					s.GetId(), l.a.q.GetName(), starts))
			}
		}
	}
	return problems
}

// late is the record of an attempt that a Run of this Query's path would not record in the path's
// order, if there is one: the Contract reads evidence in the order a Run records it, and would meet
// the record and the kind beside it the wrong way round, or one event where the path has two pieces
// of evidence.
func (l *lowering) late() []Unsupported {
	var kinds []published
	for _, confirmed := range l.confirmations {
		kind := published{kind: confirmed.Source.KindID, last: confirmed.Steps[len(confirmed.Steps)-1]}
		if of := l.attemptOf(kind.kind); of != nil {
			kind.script, kind.number = of.GetScript(), of.GetNumber()
		}
		kinds = append(kinds, kind)
	}
	answers := map[string][]int{}
	for _, s := range l.a.r.GetScripts() {
		if s.GetActivity() == nil {
			continue
		}
		_, answered := l.attemptClasses(s)
		for at, key := range l.keys {
			if answered[key] {
				publishedAt := at
				for _, item := range s.GetItems() {
					withheld := item.GetCommand().GetAttemptWithheld()
					pending := withheld != nil && withheld.GetMode() == umpirespb.WITHHOLDING_MODE_SDK_PENDING && l.adapter.classKey(item.GetWhen()[0]) == key
					finish := slices.ContainsFunc(item.GetPerforms(), func(p *umpirespb.Performance) bool {
						return p.GetCommand().GetFinish() != nil && l.adapter.classKey(p.GetStep()) == key
					})
					if !pending && !finish {
						continue
					}
					startAt := at
					for before := at - 1; before >= 0; before-- {
						if slices.ContainsFunc(s.GetActivity().GetStarts(), func(start *umpirespb.ActionClass) bool { return l.adapter.classKey(start) == l.keys[before] }) {
							startAt = before
							break
						}
					}
					if pending {
						publishedAt = startAt
					}
					// The local group settles before the SDK's later accepted answer or server timer.
					for _, prefix := range s.GetItems() {
						for _, performance := range prefix.GetPerforms() {
							if performance.GetCommand().GetAttemptHeartbeat() == nil {
								continue
							}
							prefixKey := l.adapter.classKey(performance.GetStep())
							for after := startAt + 1; after < at; after++ {
								if l.keys[after] == prefixKey {
									publishedAt = after
									break
								}
							}
						}
					}
				}
				answers[s.GetId()] = append(answers[s.GetId()], publishedAt)
			}
		}
	}
	record := outOfOrder(kinds, answers)
	if record == nil {
		return nil
	}
	gap := Unsupported{Construct: "attempt record that precedes earlier evidence", ID: record.kind, Position: locate(l.adapter.evidence[record.kind].GetPosition()),
		Owner: ownerNone, Why: fmt.Sprintf("a Run records an attempt once it is answered, which on this path is before the Run records %s, "+
			"though the record confirms later steps", record.beside)}
	switch record.how {
	case recordedLate:
		gap.Construct = "attempt record that follows later evidence"
		gap.Why = fmt.Sprintf("a Run records an attempt once it is answered, which on this path is after the Run records %s, "+
			"though the record confirms earlier steps", record.beside)
	case recordedTwice:
		of := l.attemptOf(record.kind)
		gap.Construct = "attempt recorded as two kinds of evidence"
		gap.Why = fmt.Sprintf("a Run records attempt %d of script %s as one Run Event, which is evidence of one kind, and the path confirms steps by %s as well",
			of.GetNumber(), of.GetScript(), record.beside)
	default:
	}
	return []Unsupported{gap}
}

// instructionOrder says which instructions of one entrypoint run after which.
type instructionOrder struct {
	// before is, for each instruction, the instructions it runs directly after.
	before map[string][]string
}

// runOrder reads the order of an entrypoint's instructions: one runs after the one written before
// it, or, where it names the instructions it runs after, after those and no other.
func runOrder(ids []string, after map[string][]string) instructionOrder {
	order := instructionOrder{before: map[string][]string{}}
	for i, id := range ids {
		switch named, names := after[id]; {
		case names:
			order.before[id] = named
		case i > 0:
			order.before[id] = []string{ids[i-1]}
		default:
		}
	}
	return order
}

// after is whether one instruction runs after another, directly or through the ones between.
func (o instructionOrder) after(later, earlier string) bool {
	seen := map[string]bool{}
	var reaches func(id string) bool
	reaches = func(id string) bool {
		if seen[id] {
			return false
		}
		seen[id] = true
		for _, before := range o.before[id] {
			if before == earlier || reaches(before) {
				return true
			}
		}
		return false
	}
	return reaches(later)
}

// recorders is, for each kind of evidence a Case declares, by the Case's name for it, the instruction
// of the Case that records it: the instruction a Run Event declaration names, the poll that reads the
// kind, or the read that lifts it.
func recorders(c *testpilotspb.Case) map[string]*testpilotspb.InstructionReference {
	out := map[string]*testpilotspb.InstructionReference{}
	for _, d := range c.GetProgram().GetEvidence() {
		if at := d.GetRunEvent().GetInstruction(); at != nil {
			out[d.GetEvidenceId()] = at
		}
	}
	for _, entrypoint := range c.GetProgram().GetEntrypoints() {
		for _, node := range entrypoint.GetInstructions() {
			at := &testpilotspb.InstructionReference{EntrypointId: entrypoint.GetEntrypointId(), InstructionId: node.GetInstructionId()}
			if poll := node.GetInstruction().GetReadEvidence(); poll != nil {
				out[poll.GetEvidenceId()] = at
			}
			for _, read := range node.GetInstruction().GetInvokeRpc().GetResponseReads() {
				for _, target := range read.GetTargets() {
					for _, rule := range target.GetCorrelatedEvidence().GetRules() {
						out[rule.GetEvidenceId()] = at
					}
				}
			}
		}
	}
	return out
}

// ordered checks, on the Case, that its instructions record the path's evidence in the path's order:
// every kind that confirms a step of the path and is no record of an attempt has an instruction that
// records it, and the instruction of each such kind is the one that records the kind before it or
// runs after it. Kinds one instruction records reach the Run in the order of what it reads.
func (l *lowering) ordered(c *testpilotspb.Case) error {
	local := map[string]string{}
	for _, n := range c.GetProvenance().GetLocalNames() {
		local[n.GetDefinitionId()] = n.GetLocalName()
	}
	at, query := l.a.scenario.GetPosition(), l.a.q.GetName()
	orders := map[string]instructionOrder{}
	for _, entrypoint := range c.GetProgram().GetEntrypoints() {
		var ids []string
		after := map[string][]string{}
		for _, node := range entrypoint.GetInstructions() {
			ids = append(ids, node.GetInstructionId())
			if node.GetAfter() != nil {
				after[node.GetInstructionId()] = []string{}
				for _, before := range node.GetAfter().GetInstructions() {
					after[node.GetInstructionId()] = append(after[node.GetInstructionId()], before.GetInstructionId())
				}
			}
		}
		orders[entrypoint.GetEntrypointId()] = runOrder(ids, after)
	}
	recorded := recorders(c)
	var earlier *testpilotspb.InstructionReference
	var earlierKind string
	for _, confirmed := range l.confirmations {
		kind := confirmed.Source.KindID
		if l.attemptOf(kind) != nil {
			continue
		}
		name := kind
		if renamed, ok := local[kind]; ok {
			name = renamed
		}
		by := recorded[name]
		if by == nil {
			return errorAt(at, "query %s: no instruction of the Case records evidence %s, which confirms a step of the path", query, kind)
		}
		spelled := func(ref *testpilotspb.InstructionReference) string {
			return ref.GetEntrypointId() + "/" + ref.GetInstructionId()
		}
		if earlier != nil && spelled(earlier) != spelled(by) && (earlier.GetEntrypointId() != by.GetEntrypointId() ||
			!orders[by.GetEntrypointId()].after(by.GetInstructionId(), earlier.GetInstructionId())) {
			return errorAt(at, "query %s: evidence %s is recorded by %s, which the Case does not run after %s, and the path records %s first", query, kind,
				spelled(by), spelled(earlier), earlierKind)
		}
		earlier, earlierKind = by, kind
	}
	return nil
}

// takes is whether a Case of this Query's path would carry a command: one every Case carries, one
// carried for a class the path takes, or the performance of a class the path takes.
func (l *lowering) takes(item *umpirespb.Item, performance *umpirespb.Performance) bool {
	if performance != nil {
		return slices.Contains(l.keys, l.adapter.classKey(performance.GetStep()))
	}
	return len(item.GetWhen()) == 0 || slices.ContainsFunc(item.GetWhen(), func(class *umpirespb.ActionClass) bool {
		return slices.Contains(l.keys, l.adapter.classKey(class))
	})
}

// commandsOf is every command a script declares, in declaration order.
func commandsOf(s *umpirespb.Script) []*umpirespb.Command {
	var out []*umpirespb.Command
	for _, item := range s.GetItems() {
		if item.GetCommand() != nil {
			out = append(out, item.GetCommand())
		}
		for _, p := range item.GetPerforms() {
			out = append(out, p.GetCommand())
		}
	}
	return out
}

// lowering is one Query checked and ready to be produced: the realization translated for the Case's
// identity, and the Query as the checker answers it.
type lowering struct {
	a           *asked
	mm          *interp.Machine
	adapter     *adapter
	realization *cp.Realization
	query       *check.Query
	keys        []string
	// confirmations is which kind of evidence confirms each step of the path, as the producer decides
	// it: set once the producer has read the path whole and refused nothing.
	confirmations []cp.Confirmation
	// uses is, for each hint and server step the Case's waits read, by "behavior:<id>" or
	// "server_steps:<class>", the instructions that read it.
	uses map[string][]string
}

func (p *Producer) source(q *umpirespb.Query) cp.Source {
	return cp.Source{Path: q.GetPosition().GetFile(), Provenance: "scala-model"}
}

// check reads a realization and a Query's path whole and emits nothing: what the realization writes
// against its descriptors, that a command performs every step an actor takes, that the search found a
// witness, that the Property lowers to clauses, and, once the realization itself is sound, everything
// the producer decides before it writes a Case. It reports every problem it finds, and the lowering
// is ready to be produced when it finds none.
func (p *Producer) check(a *asked, identity Identity) (*lowering, []error) {
	at, name := a.q.GetPosition(), a.q.GetName()
	mm := p.realizer.Machine(a.scenario.GetMachine())
	if a.scenario.GetFree() || len(a.scenario.GetKeys()) > 0 || mm == nil {
		return nil, []error{errorAt(at, "query %s runs %s, which pins no schedule of a machine's classes; only a pinned schedule is realized",
			name, a.scenario.GetName())}
	}
	if a.property.GetMachine() != a.scenario.GetMachine() || a.q.GetThrough() {
		return nil, []error{errorAt(at, "query %s reads a Property of %s on %s; a Case is lowered from a Property of the machine it runs", name,
			a.property.GetMachine(), a.scenario.GetMachine())}
	}
	query, err := p.realizer.Find(a.key)
	if err != nil {
		return nil, []error{fmt.Errorf("%s: query %s: %w", locate(at), name, err)}
	}
	table, err := query.Scenario.Machine.Table()
	if err != nil {
		return nil, []error{fmt.Errorf("%s: query %s: %w", locate(at), name, err)}
	}
	l := &lowering{a: a, mm: mm, query: query, keys: query.Scenario.Actions, adapter: newAdapter(a.r, table, p.realizer.ClassKey, identity.Fixture)}
	var problems []error
	l.realization, problems = l.adapter.realization()
	l.realization.Target = cp.Fingerprinted{Table: table, Fingerprint: p.realizer.TargetFingerprint(table)}
	sound := len(problems) == 0
	if err := l.performed(); err != nil {
		problems = append(problems, err)
	}
	problems = append(problems, l.withholdingOccurrences()...)
	problems = append(problems, l.resetOccurrences()...)
	receipt, ok := p.found[name]
	witnessed := ok && receipt.Kind == check.Found && receipt.Witness != nil
	if !witnessed {
		problems = append(problems, errorAt(at, "query %s has no witness to realize: its check is %s: %s", name, receipt.Kind, receipt.Explanation))
	}
	if _, err := query.Property.Lower(); err != nil {
		// The error keeps its cause: a predicate that reached a hole is told apart from one that is malformed.
		problems = append(problems, fmt.Errorf("%s: %w", locate(a.property.GetPosition()), err))
	} else if sound && witnessed {
		// A realization that crosses a descriptor is not all there for the producer to read, and a Query
		// with no witness or no clauses has nothing for it to decide; either is already reported.
		if err := cp.Preflight(query, identity, l.realization); err != nil {
			problems = append(problems, fmt.Errorf("%s: query %s: %w", locate(at), name, err))
		} else if l.confirmations, err = cp.Confirmations(query, identity, l.realization); err != nil {
			problems = append(problems, fmt.Errorf("%s: query %s: %w", locate(at), name, err))
		} else {
			problems = append(problems, l.unstarted()...)
			var refused []error
			l.adapter.waits, l.uses, refused = l.waits()
			problems = append(problems, refused...)
		}
	}
	return l, problems
}

// performed rejects a path with a step an actor takes that nothing performs: a Case that does not
// drive it would wait for something nothing does. A command performs a step, and so does the
// activation of an activity script that starts with the step's class. A step of the system needs
// neither.
func (l *lowering) performed() error {
	bound := map[string]bool{}
	for _, s := range l.a.r.GetScripts() {
		for _, class := range s.GetActivity().GetStarts() {
			bound[l.adapter.classKey(class)] = true
		}
		for _, item := range s.GetItems() {
			for _, performance := range item.GetPerforms() {
				bound[l.adapter.classKey(performance.GetStep())] = true
			}
		}
	}
	actor := map[string]string{}
	for _, class := range l.mm.Classes {
		actor[class.Key] = class.Action.GetActor()
	}
	var unperformed []error
	for _, key := range l.keys {
		if !bound[key] && actor[key] != "system" {
			unperformed = append(unperformed, errorAt(l.a.scenario.GetPosition(), "scenario %s takes %s, a step of %s, and no script of realization %s performs it",
				l.a.scenario.GetName(), key, actor[key], l.a.r.GetName()))
		}
	}
	return errors.Join(unperformed...)
}

// accounting is the inventory of one Case as it is taken: the entries so far, and the parts of the
// Case a declaration accounts for.
type accounting struct {
	l       *lowering
	c       *testpilotspb.Case
	entries []Entry
	owned   map[string]bool
	local   map[string]string
}

// realizationFields says how each field of a Realization is accounted for in a Case, and
// correlationFields each field of its Correlation. The inventory walks the two messages' descriptors,
// so a field the IR gains is an error of every lowering until it is accounted for here.
var realizationFields = map[protoreflect.Name]func(*accounting) error{
	"id":       func(a *accounting) error { return a.names("id") },
	"name":     func(a *accounting) error { return a.names("name") },
	"position": func(*accounting) error { return nil },
	"machine":  (*accounting).machine,
	"producer": func(a *accounting) error {
		return a.field("realization", "producer", a.l.a.r.GetProducer(), a.c.GetProvenance().GetProducerId(), "provenance.producer_id")
	},
	"producer_version": func(a *accounting) error {
		return a.field("realization", "producer_version", a.l.a.r.GetProducerVersion(), a.c.GetProvenance().GetProducerVersion(), "provenance.producer_version")
	},
	"roles":        (*accounting).roles,
	"learned":      (*accounting).learned,
	"observations": (*accounting).observations,
	"evidence":     (*accounting).evidence,
	"correlation":  (*accounting).correlation,
	"controls":     (*accounting).controls,
	"scripts":      (*accounting).scripts,
	"cleanup": func(a *accounting) error {
		return a.field("realization", "cleanup", a.l.a.r.GetCleanup(), a.c.GetProgram().GetCleanup().GetEntrypointId(), "program.cleanup")
	},
	"required_settings":    (*accounting).requiredSettings,
	"behavior":             (*accounting).behavior,
	"server_steps":         (*accounting).serverSteps,
	"rejection_codes":      (*accounting).rejectionCodes,
	"external_settlements": (*accounting).externalSettlements,
	"reset_settlements":    (*accounting).resetSettlements,
}

var correlationFields = map[protoreflect.Name]func(a *accounting, c *umpirespb.Correlation, contract *testpilotspb.CorrelatedContract) error{
	"position": func(*accounting, *umpirespb.Correlation, *testpilotspb.CorrelatedContract) error { return nil },
	"projection": func(a *accounting, c *umpirespb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		for _, rule := range a.c.GetProvenance().GetCorrelatedRules() {
			if rule.GetProjectionId() != c.GetProjection() {
				return a.differs("correlation", "projection", c.GetProjection(), rule.GetProjectionId(), "provenance.correlated_rules")
			}
		}
		return a.field("correlation", "projection", a.named(c.GetProjection()), contract.GetProjectionId(), "contract.correlated.projection_id")
	},
	"run": func(a *accounting, c *umpirespb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		for _, e := range a.c.GetProgram().GetEvidence() {
			if len(e.GetScope()) != 1 || e.GetScope()[0].GetFieldId() != a.named(c.GetRun()) {
				return a.differs("correlation", "run", a.named(c.GetRun()), fmt.Sprint(e.GetScope()), "program.evidence["+e.GetEvidenceId()+"]")
			}
		}
		return a.field("correlation", "run", a.named(c.GetRun()), strings.Join(contract.GetScopeFields(), ","), "contract.correlated.scope_fields")
	},
	"operation": func(a *accounting, c *umpirespb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		return a.field("correlation", "operation", a.named(c.GetOperation()), contract.GetOperationField(), "contract.correlated.operation_field")
	},
	"observation": func(a *accounting, c *umpirespb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		return a.field("correlation", "observation", c.GetObservation(), contract.GetEvidenceObservationId(), "contract.correlated.evidence_observation_id")
	},
	"events":     (*accounting).window,
	"buffered":   (*accounting).window,
	"keys":       (*accounting).window,
	"support":    (*accounting).window,
	"work":       (*accounting).window,
	"event_size": (*accounting).window,
}

// derived is the parts of a Case no declaration of the realization accounts for, and what they are
// derived from instead.
var derived = map[string]string{
	"case_id":                                  "the Case's identity",
	"version":                                  "the Case format",
	"program.program_id":                       "the Case's identity",
	"contract.contract_id":                     "the Case's identity",
	"provenance.sources":                       "where the Query was written",
	"provenance.known_gaps":                    "the steps of the witness that record nothing",
	"provenance.correlated_rules":              "the Property's clauses",
	"provenance.local_names":                   "the Case's own names",
	"provenance.abstraction_claims":            "the examples of the machine's actions",
	"contract.correlated.initial_state":        "the Scenario's start",
	"contract.correlated.initial_state_fields": "the Scenario's start",
	"contract.correlated.transitions":          "the machine's rows",
	"contract.correlated.projection_rules":     "the witness and the machine's evidence",
	"contract.correlated.rules":                "the Property's clauses",
}

// inventory accounts for every declaration of the realization against the Case, and for every part of
// the Case against the declarations: a declaration the Case does not carry as declared, and a part of
// the Case that neither a declaration nor the Query accounts for, are errors of this package.
func (l *lowering) inventory(c *testpilotspb.Case) ([]Entry, error) {
	a := &accounting{l: l, c: c, owned: map[string]bool{}, local: map[string]string{}}
	for _, n := range c.GetProvenance().GetLocalNames() {
		a.local[n.GetDefinitionId()] = n.GetLocalName()
	}
	r := l.a.r.ProtoReflect()
	fields := r.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		account, ok := realizationFields[f.Name()]
		if !ok {
			return nil, errorAt(l.a.r.GetPosition(), "a realization's %s has no place in the inventory of a Case", f.Name())
		}
		if !r.Has(f) && f.Name() != "cleanup" {
			continue
		}
		if err := account(a); err != nil {
			return nil, err
		}
	}
	if err := a.unaccounted(); err != nil {
		return nil, err
	}
	return a.entries, nil
}

// named is the Case-local name of a Definition ID.
func (a *accounting) named(id string) string {
	if name, ok := a.local[id]; ok {
		return name
	}
	return id
}

func (a *accounting) differs(kind, id, declared, carried, part string) error {
	return errorAt(a.l.a.r.GetPosition(), "%s %s of realization %s is %q, and the Case's %s carries %q", kind, id,
		a.l.a.r.GetName(), declared, part, carried)
}

// field records a declaration one part of the Case carries as the value it declares. A declaration
// left empty has no entry, and the part it would fill is then empty too.
func (a *accounting) field(kind, id, declared, carried, part string) error {
	if declared != carried {
		return a.differs(kind, id, declared, carried, part)
	}
	if declared == "" {
		return nil
	}
	a.own(kind, id, a.l.a.r.GetPosition(), part)
	return nil
}

func (a *accounting) own(kind, id string, at *umpirespb.Position, parts ...string) {
	for _, part := range parts {
		a.owned[part] = true
	}
	a.entries = append(a.entries, Entry{Kind: kind, ID: id, Position: locate(at), Disposition: InCase, As: parts})
}

// names records what identifies the realization in the IR, which no part of a Case repeats.
func (a *accounting) names(field string) error {
	a.entries = append(a.entries, Entry{Kind: "realization", ID: field, Position: locate(a.l.a.r.GetPosition()), Disposition: Names})
	return nil
}

// machine records the machine a realization runs as the target the Case's provenance binds.
func (a *accounting) machine() error {
	target := a.l.query.Scenario.Machine
	table, err := target.Table()
	if err != nil {
		return err
	}
	for _, d := range a.c.GetProvenance().GetDefinitions() {
		if d.GetKind() == testpilotspb.DEFINITION_KIND_TARGET {
			return a.field("realization", "machine", table.IDs().Target, d.GetDefinitionId(), "provenance.definitions")
		}
	}
	return a.differs("realization", "machine", table.IDs().Target, "", "provenance.definitions")
}

// carried records a declaration every Case carries under its own id.
func (a *accounting) carried(kind, id string, at *umpirespb.Position, part string, has bool) error {
	if !has {
		return errorAt(at, "%s %s of realization %s is in no part of the Case", kind, id, a.l.a.r.GetName())
	}
	a.own(kind, id, at, part)
	return nil
}

func (a *accounting) roles() error {
	for _, role := range a.l.a.r.GetRoles() {
		has := slices.ContainsFunc(a.c.GetProgram().GetRoles(), func(x *testpilotspb.Role) bool { return x.GetRoleId() == role.GetId() })
		if err := a.carried("roles", role.GetId(), role.GetPosition(), "program.roles["+role.GetId()+"]", has); err != nil {
			return err
		}
	}
	return nil
}

// requiredSettings records each setting the realization requires as the Program's setting of its
// key, which carries it in the declared order and with the declared value.
func (a *accounting) requiredSettings() error {
	carried := a.c.GetProgram().GetRequiredSettings()
	for i, s := range a.l.a.r.GetRequiredSettings() {
		part := "program.required_settings[" + s.GetKey() + "]"
		declared := s.GetKey() + "=" + s.GetValue()
		if i >= len(carried) {
			return a.differs("required_settings", s.GetKey(), declared, "", part)
		}
		if got := carried[i].GetKey() + "=" + carried[i].GetValue(); got != declared {
			return a.differs("required_settings", s.GetKey(), declared, got, part)
		}
		a.own("required_settings", s.GetKey(), a.l.a.r.GetPosition(), part)
	}
	return nil
}

// behavior records each hint of the API behavior by its id, and serverSteps each server step by its
// class: in the instructions whose waits read it, or unread. The rest of the behavior is carried as
// declared: how attempts are numbered by every activity entrypoint, the limits of an instruction
// that writes none and whether the run's record order is causal by the Program.
func (a *accounting) behavior() error {
	b := a.l.a.r.GetBehavior()
	for _, v := range b.GetVisibility() {
		a.read("behavior", v.GetId(), v.GetPosition())
	}
	for _, c := range b.GetCauses() {
		a.read("behavior", c.GetId(), c.GetPosition())
	}
	program := a.c.GetProgram()
	if n := b.GetAttemptNumbering(); n != nil {
		declared := fmt.Sprintf("from %d, one run %t", n.GetFirst(), n.GetOneRun())
		var carriers []string
		for _, e := range program.GetEntrypoints() {
			if e.GetActivity() == nil {
				continue
			}
			part := "program.entrypoints[" + e.GetEntrypointId() + "]"
			carried := e.GetActivity().GetAttemptNumbering()
			if got := fmt.Sprintf("from %d, one run %t", carried.GetFirst(), carried.GetOneRun()); carried == nil || got != declared {
				return a.differs("behavior", "attemptNumbering", declared, got, part+".activity.attempt_numbering")
			}
			carriers = append(carriers, part)
		}
		if len(carriers) > 0 {
			a.own("behavior", "attemptNumbering", n.GetPosition(), carriers...)
		} else {
			a.entries = append(a.entries, Entry{Kind: "behavior", ID: "attemptNumbering", Position: locate(n.GetPosition()), Disposition: Unread})
		}
	}
	if d := b.GetInstructionDefaults(); d != nil {
		carried := program.GetInstructionDefaults()
		declared, got := fmt.Sprintf("%d ms, %d attempts", d.GetTimeoutMs(), d.GetAttempts()),
			fmt.Sprintf("%d ms, %d attempts", carried.GetTimeoutMilliseconds(), carried.GetMaxAttempts())
		if carried.GetTimeout() == nil || carried.GetAttempts() == nil || got != declared {
			return a.differs("behavior", "instructionDefaults", declared, got, "program.instruction_defaults")
		}
		a.own("behavior", "instructionDefaults", d.GetPosition(), "program.instruction_defaults.timeout_milliseconds", "program.instruction_defaults.max_attempts")
	}
	if b.GetRunOrderIsCausal() {
		if !program.GetRunOrderIsCausal() {
			return a.differs("behavior", "runOrderIsCausal", "true", "false", "program.run_order_is_causal")
		}
		a.own("behavior", "runOrderIsCausal", a.l.a.r.GetPosition(), "program.run_order_is_causal")
	}
	return nil
}

func (a *accounting) serverSteps() error {
	for _, s := range a.l.a.r.GetServerSteps() {
		a.read("server_steps", a.l.adapter.classKey(s.GetStep()), s.GetPosition())
	}
	return nil
}

func (a *accounting) rejectionCodes() error {
	for _, code := range a.l.a.r.GetRejectionCodes() {
		a.read("rejection_codes", code.GetRejection().String(), a.l.a.r.GetPosition())
	}
	return nil
}

func (a *accounting) read(kind, id string, at *umpirespb.Position) {
	if parts := a.l.uses[kind+":"+id]; len(parts) > 0 {
		a.own(kind, id, at, parts...)
		return
	}
	a.entries = append(a.entries, Entry{Kind: kind, ID: id, Position: locate(at), Disposition: Unread})
}

func (a *accounting) learned() error {
	for _, learned := range a.l.a.r.GetLearned() {
		has := slices.ContainsFunc(a.c.GetProgram().GetSlots(), func(x *testpilotspb.Slot) bool { return x.GetSlotId() == learned.GetId() })
		if err := a.carried("learned", learned.GetId(), learned.GetPosition(), "program.slots["+learned.GetId()+"]", has); err != nil {
			return err
		}
	}
	return nil
}

func (a *accounting) observations() error {
	for _, o := range a.l.a.r.GetObservations() {
		has := slices.ContainsFunc(a.c.GetProgram().GetObservations(), func(x *testpilotspb.Observation) bool { return x.GetObservationId() == o.GetId() })
		if err := a.carried("observations", o.GetId(), o.GetPosition(), "program.observations["+o.GetId()+"]", has); err != nil {
			return err
		}
	}
	return nil
}

// evidenceFields says how each field of a kind of evidence the Case carries is accounted for. A field
// with no entry of its own is part of what the Case's evidence declaration states. The inventory walks
// the message's descriptor, so a field the IR gains is an error of every lowering until it is
// accounted for here.
var evidenceFields = map[protoreflect.Name]func(a *accounting, e *umpirespb.Evidence, d *testpilotspb.EvidenceDeclaration) ([]string, error){
	"id":         nil,
	"position":   nil,
	"records":    nil,
	"source":     nil,
	"operation":  nil,
	"commitment": nil,
	"history":    (*accounting).recorded,
	"read":       (*accounting).recorded,
	"single":     (*accounting).recorded,
	"run_event":  (*accounting).recorded,
	"fields":     (*accounting).kept,
	"exhaustive": (*accounting).closing,
	"confirms":   (*accounting).confirmed,
}

// recordedIn spells where a realization says a kind of evidence is recorded, and declaredIn where a
// Case's declaration says it is, in the same words, so that the two are compared as one text.
func recordedIn(e *umpirespb.Evidence) string {
	switch from := e.GetFrom().(type) {
	case *umpirespb.Evidence_History:
		return "history event " + from.History
	case *umpirespb.Evidence_Read:
		return "read " + from.Read.GetMethod() + " " + from.Read.GetPath()
	case *umpirespb.Evidence_Single:
		return "single read " + from.Single.GetMethod() + " " + from.Single.GetPath()
	case *umpirespb.Evidence_RunEvent:
		keyed := "nothing"
		switch key := from.RunEvent.GetKey(); {
		case key.GetRun() != nil:
			keyed = "the run"
		case key.GetPath() != nil:
			keyed = key.GetPath().GetPath()
		default:
		}
		return fmt.Sprintf("run event %s of %s/%s keyed by %s", umpirespb.RunEventSource_Kind_name[int32(from.RunEvent.GetKind())],
			from.RunEvent.GetScript(), from.RunEvent.GetCommand(), keyed)
	default:
		return "nowhere"
	}
}

func declaredIn(d *testpilotspb.EvidenceDeclaration) string {
	switch from := d.GetSource().(type) {
	case *testpilotspb.EvidenceDeclaration_HistoryEvent:
		return "history event " + from.HistoryEvent.GetAttributesField()
	case *testpilotspb.EvidenceDeclaration_Read:
		if from.Read.GetSingle() {
			return "single read " + from.Read.GetMethod() + " " + from.Read.GetPath()
		}
		return "read " + from.Read.GetMethod() + " " + from.Read.GetPath()
	case *testpilotspb.EvidenceDeclaration_RunEvent:
		kind := umpirespb.RunEventSource_KIND_UNSPECIFIED
		for declared, lowered := range runEventKinds {
			if lowered == from.RunEvent.GetKind() {
				kind = declared
			}
		}
		keyed := "the run"
		if !from.RunEvent.GetRunKeyed() {
			keyed = d.GetOperation()
		}
		return fmt.Sprintf("run event %s of %s/%s keyed by %s", umpirespb.RunEventSource_Kind_name[int32(kind)],
			from.RunEvent.GetInstruction().GetEntrypointId(), from.RunEvent.GetInstruction().GetInstructionId(), keyed)
	default:
		return "nowhere"
	}
}

// recorded checks that the Case declares a kind of evidence recorded where the realization says it
// is. The Run's own record is an instruction's, which the Case must carry: the instruction is part of
// what carries the kind.
func (a *accounting) recorded(e *umpirespb.Evidence, d *testpilotspb.EvidenceDeclaration) ([]string, error) {
	part := "program.evidence[" + d.GetEvidenceId() + "]"
	if declared, carried := recordedIn(e), declaredIn(d); declared != carried {
		return nil, a.differs("evidence", e.GetId(), declared, carried, part)
	}
	source := e.GetRunEvent()
	if source == nil {
		return nil, nil
	}
	// The guard is compared as the Case states it: lowered again from the declaration.
	payload, err := messageNamed(e.GetPosition(), instructionOutcomeMessage)
	if err != nil {
		return nil, err
	}
	guard, err := a.l.adapter.guardOf(e, source, payload)
	if err != nil {
		return nil, err
	}
	if !proto.Equal(guard, d.GetRunEvent().GetGuard()) {
		return nil, errorAt(e.GetPosition(), "evidence %s is the Run's record under a guard, and the Case's %s declares it under another", e.GetId(), part)
	}
	for _, entrypoint := range a.c.GetProgram().GetEntrypoints() {
		if entrypoint.GetEntrypointId() == source.GetScript() && slices.ContainsFunc(entrypoint.GetInstructions(), func(n *testpilotspb.InstructionNode) bool {
			return n.GetInstructionId() == source.GetCommand()
		}) {
			return []string{"program.entrypoints[" + source.GetScript() + "].instructions[" + source.GetCommand() + "]"}, nil
		}
	}
	return nil, errorAt(e.GetPosition(), "evidence %s is the Run's record of %s/%s, and the Case carries no such instruction", e.GetId(),
		source.GetScript(), source.GetCommand())
}

// kept checks that the Case keeps the fields a kind of evidence declares, each at its path in the
// Program's declaration and retained by the Contract's rule for the kind, and no other. A redacted
// field has no part of a Case.
func (a *accounting) kept(e *umpirespb.Evidence, d *testpilotspb.EvidenceDeclaration) ([]string, error) {
	part := "program.evidence[" + d.GetEvidenceId() + "]"
	var rule *testpilotspb.CorrelatedProjectionRule
	for _, r := range a.c.GetContract().GetCorrelated().GetProjectionRules() {
		if r.GetKind() == d.GetEvidenceId() {
			rule = r
		}
	}
	for i, f := range e.GetFields() {
		at := f.GetPosition()
		if at.GetFile() == "" {
			at = e.GetPosition()
		}
		if f.GetRedacted() {
			return nil, errorAt(at, "field %s of evidence %s of realization %s is carried without its value, which is in no part of the Case", f.GetId(),
				e.GetId(), a.l.a.r.GetName())
		}
		name := a.named(f.GetId())
		if i >= len(d.GetFields()) || i >= len(rule.GetFields()) || d.GetFields()[i].GetFieldId() != name || d.GetFields()[i].GetPath() != f.GetPath() ||
			rule.GetFields()[i].GetFieldId() != name || rule.GetFields()[i].GetDisposition() != testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN {
			return nil, errorAt(at, "evidence %s keeps field %s at %s, and the Case's %s does not", e.GetId(), f.GetId(), f.GetPath(), part)
		}
	}
	if kept := max(len(d.GetFields()), len(rule.GetFields())); kept > len(e.GetFields()) {
		return nil, errorAt(e.GetPosition(), "evidence %s keeps %d fields, and the Case's %s keeps %d", e.GetId(), len(e.GetFields()), part, kept)
	}
	return nil, nil
}

// confirmed checks that the Case's rule for a kind that names the steps it confirms confirms as many:
// the kind is carried for all of them or for none.
func (a *accounting) confirmed(e *umpirespb.Evidence, d *testpilotspb.EvidenceDeclaration) ([]string, error) {
	for _, rule := range a.c.GetContract().GetCorrelated().GetProjectionRules() {
		if rule.GetKind() != d.GetEvidenceId() {
			continue
		}
		// A rule confirms the steps its kind names and the steps between them that record nothing.
		named := 0
		for _, output := range rule.GetOutputs() {
			if slices.ContainsFunc(e.GetConfirms(), func(taking *umpirespb.Taking) bool {
				return a.l.adapter.classKey(taking.GetStep()) == output.GetAction().GetValue()
			}) {
				named++
			}
		}
		if named != len(e.GetConfirms()) {
			return nil, errorAt(e.GetPosition(), "evidence %s confirms %d steps, and the Case's rule for it confirms %d", e.GetId(), len(e.GetConfirms()), named)
		}
		return nil, nil
	}
	return nil, errorAt(e.GetPosition(), "evidence %s confirms %d steps, and the Case has no rule for it", e.GetId(), len(e.GetConfirms()))
}

// closing is the instruction whose read closes an exhaustive kind of evidence the Case carries: the
// command the realization names, which every Case carries.
func (a *accounting) closing(e *umpirespb.Evidence, _ *testpilotspb.EvidenceDeclaration) ([]string, error) {
	for _, s := range a.l.a.r.GetScripts() {
		for _, c := range commandsOf(s) {
			if !slices.Contains(c.GetCloses(), e.GetId()) {
				continue
			}
			for _, entrypoint := range a.c.GetProgram().GetEntrypoints() {
				if entrypoint.GetEntrypointId() == s.GetId() && slices.ContainsFunc(entrypoint.GetInstructions(), func(n *testpilotspb.InstructionNode) bool {
					return n.GetInstructionId() == c.GetId()
				}) {
					return []string{"program.entrypoints[" + s.GetId() + "].instructions[" + c.GetId() + "]"}, nil
				}
			}
			return nil, errorAt(e.GetPosition(), "evidence %s is exhaustive, and the Case carries no %s/%s to close it", e.GetId(), s.GetId(), c.GetId())
		}
	}
	return nil, errorAt(e.GetPosition(), "evidence %s is exhaustive, and no command of realization %s closes it", e.GetId(), a.l.a.r.GetName())
}

// evidence records each kind of evidence under the Case-local name the Case declares it by, with the
// source the Contract counts it in, the instruction whose record it is where it is the Run's own, and,
// for an exhaustive kind, the instruction that closes it; or as off the path where the Case does not
// carry it: no step of the path is confirmed by it, and it is not exhaustive.
func (a *accounting) evidence() error {
	sources := a.c.GetContract().GetCorrelated().GetSources()
	for _, e := range a.l.a.r.GetEvidence() {
		name := a.named(e.GetId())
		at := slices.IndexFunc(a.c.GetProgram().GetEvidence(), func(x *testpilotspb.EvidenceDeclaration) bool { return x.GetEvidenceId() == name })
		if at < 0 {
			a.entries = append(a.entries, Entry{Kind: "evidence", ID: e.GetId(), Position: locate(e.GetPosition()), Disposition: OffPath})
			continue
		}
		declaration := a.c.GetProgram().GetEvidence()[at]
		if !slices.Contains(sources, a.named(e.GetSource())) {
			return a.differs("evidence", e.GetId(), a.named(e.GetSource()), strings.Join(sources, ","), "contract.correlated.sources")
		}
		parts := []string{"program.evidence[" + name + "]", "contract.correlated.sources"}
		message := e.ProtoReflect()
		fields := message.Descriptor().Fields()
		for i := range fields.Len() {
			f := fields.Get(i)
			account, known := evidenceFields[f.Name()]
			if !known {
				return errorAt(e.GetPosition(), "a kind of evidence's %s has no place in the inventory of a Case", f.Name())
			}
			if account == nil || !message.Has(f) {
				continue
			}
			carried, err := account(a, e, declaration)
			if err != nil {
				return err
			}
			parts = append(parts, carried...)
		}
		a.own("evidence", e.GetId(), e.GetPosition(), parts...)
	}
	return nil
}

// correlation records each field of the correlation, walking its descriptor as inventory walks the
// realization's.
func (a *accounting) correlation() error {
	c, contract := a.l.a.r.GetCorrelation(), a.c.GetContract().GetCorrelated()
	message := c.ProtoReflect()
	fields := message.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		account, ok := correlationFields[f.Name()]
		if !ok {
			return errorAt(c.GetPosition(), "a correlation's %s has no place in the inventory of a Case", f.Name())
		}
		if !message.Has(f) {
			continue
		}
		before := len(a.entries)
		if err := account(a, c, contract); err != nil {
			return err
		}
		for i := before; i < len(a.entries); i++ {
			a.entries[i].ID, a.entries[i].Position = string(f.Name()), locate(c.GetPosition())
		}
	}
	return nil
}

// window records one bound of the window a check keeps. The Case carries the window in the
// fingerprint of its projection alone.
func (a *accounting) window(*umpirespb.Correlation, *testpilotspb.CorrelatedContract) error {
	if a.c.GetContract().GetCorrelated().GetProjectionFingerprint() == "" {
		return a.differs("correlation", "window", "a window", "", "contract.correlated.projection_fingerprint")
	}
	a.own("correlation", "", a.l.a.r.GetCorrelation().GetPosition(), "contract.correlated.projection_fingerprint")
	return nil
}

// controls records a control a Driver realizes as the instructions of the Case that hold and release
// what it holds, the delivery controls of its task-queue role; one no instruction of the Case uses is
// off the path. It refuses any other control in a Case: Testpilot has no part that carries one, and a
// realization that declares one is not lowered.
func (a *accounting) controls() error {
	for _, c := range a.l.a.r.GetControls() {
		if !heldByDriver(c) {
			return errorAt(c.GetPosition(), "controls %s of realization %s is in no part of the Case", c.GetId(), a.l.a.r.GetName())
		}
		entry := Entry{Kind: "controls", ID: c.GetId(), Position: locate(c.GetPosition()), Disposition: OffPath}
		for _, e := range a.c.GetProgram().GetEntrypoints() {
			for _, n := range e.GetInstructions() {
				fault := n.GetInstruction().GetInjectFault()
				if fault.GetRoleId() != c.GetRole() ||
					fault.GetKind() != testpilotspb.FAULT_KIND_DELIVERY_HOLD && fault.GetKind() != testpilotspb.FAULT_KIND_DELIVERY_RELEASE && fault.GetKind() != testpilotspb.FAULT_KIND_ADMISSION_RESPONSE_LOSS {
					continue
				}
				entry.Disposition = InCase
				entry.As = append(entry.As, "program.entrypoints["+e.GetEntrypointId()+"].instructions["+n.GetInstructionId()+"]")
			}
		}
		a.entries = append(a.entries, entry)
	}
	return nil
}

func (a *accounting) scripts() error {
	for _, s := range a.l.a.r.GetScripts() {
		if err := a.script(s); err != nil {
			return err
		}
	}
	return nil
}

// script records a script and each of its commands: a plain command under its id where the Case
// carries it, and a performance once per step of the path that takes its class.
func (a *accounting) script(s *umpirespb.Script) error {
	var entrypoint *testpilotspb.Entrypoint
	for _, e := range a.c.GetProgram().GetEntrypoints() {
		if e.GetEntrypointId() == s.GetId() {
			entrypoint = e
		}
	}
	if err := a.carried("scripts", s.GetId(), s.GetPosition(), "program.entrypoints["+s.GetId()+"]", entrypoint != nil); err != nil {
		return err
	}
	// A delivery the script starts with is performed by the script's activation, which the entrypoint is.
	for _, class := range s.GetActivity().GetStarts() {
		key := a.l.adapter.classKey(class)
		entry := Entry{Kind: "activation", ID: fmt.Sprintf("%s [%s]", s.GetId(), key), Position: locate(s.GetPosition()), Disposition: OffPath}
		if slices.Contains(a.l.keys, key) {
			entry.Disposition, entry.As = InCase, []string{"program.entrypoints[" + s.GetId() + "]"}
		}
		a.entries = append(a.entries, entry)
	}
	carries := func(id string) bool {
		return slices.ContainsFunc(entrypoint.GetInstructions(), func(n *testpilotspb.InstructionNode) bool { return n.GetInstructionId() == id })
	}
	part := func(id string) string { return "program.entrypoints[" + s.GetId() + "].instructions[" + id + "]" }
	for _, item := range s.GetItems() {
		if cmd := item.GetCommand(); cmd != nil {
			entry := Entry{Kind: "command", ID: s.GetId() + "/" + cmd.GetId(), Position: locate(cmd.GetPosition()), Disposition: OffPath}
			if carries(cmd.GetId()) {
				entry.Disposition, entry.As = InCase, []string{part(cmd.GetId())}
				a.owned[part(cmd.GetId())] = true
			}
			a.entries = append(a.entries, entry)
		}
		for _, performance := range item.GetPerforms() {
			if err := a.performance(s, performance, carries, part); err != nil {
				return err
			}
		}
	}
	return nil
}

// performance records the command of one class: once per step of the path that takes the class, a
// class taken again under its ordinal after the command's id.
func (a *accounting) performance(s *umpirespb.Script, performance *umpirespb.Performance, carries func(id string) bool,
	part func(id string) string) error {
	cmd := performance.GetCommand()
	key := a.l.adapter.classKey(performance.GetStep())
	entry := Entry{Kind: "command", ID: fmt.Sprintf("%s/%s [%s]", s.GetId(), cmd.GetId(), key), Position: locate(cmd.GetPosition()),
		Disposition: OffPath}
	for range slices.DeleteFunc(slices.Clone(a.l.keys), func(taken string) bool { return taken != key }) {
		id := cmd.GetId()
		if len(entry.As) > 0 {
			id = fmt.Sprintf("%s-%d", id, len(entry.As)+1)
		}
		if !carries(id) {
			return errorAt(cmd.GetPosition(), "the path performs %s, and the Case carries no %s/%s for it", key, s.GetId(), id)
		}
		entry.Disposition, entry.As = InCase, append(entry.As, part(id))
		a.owned[part(id)] = true
	}
	a.entries = append(a.entries, entry)
	return nil
}

// parts lists the parts of a Case: every populated field of the Case, its provenance, its Program and
// its Contract, a Program's repeated declarations each by its id, and an entrypoint's instructions
// each by its id. It walks the messages' descriptors, so a part the protocol gains is listed.
func parts(c *testpilotspb.Case) []string {
	var out []string
	var walk func(prefix string, m protoreflect.Message, depth int)
	walk = func(prefix string, m protoreflect.Message, depth int) {
		fields := m.Descriptor().Fields()
		for i := range fields.Len() {
			f := fields.Get(i)
			if !m.Has(f) {
				continue
			}
			path := prefix + string(f.Name())
			switch {
			case f.IsList() && f.Kind() == protoreflect.MessageKind && depth > 0 && strings.HasPrefix(path, "program."):
				list := m.Get(f).List()
				for j := range list.Len() {
					element := list.Get(j).Message()
					id := element.Get(element.Descriptor().Fields().ByNumber(1)).String()
					if binding, ok := element.Interface().(*testpilotspb.ActivityExternalSettlement); ok {
						id = binding.GetAnswer().GetInstructionId()
					}
					if binding, ok := element.Interface().(*testpilotspb.ActivityResetSettlement); ok {
						id = binding.GetResetRequest().GetInstructionId()
					}
					each := path + "[" + id + "]"
					out = append(out, each)
					if instructions := element.Descriptor().Fields().ByName("instructions"); instructions != nil {
						nodes := element.Get(instructions).List()
						for k := range nodes.Len() {
							node := nodes.Get(k).Message()
							out = append(out, each+".instructions["+node.Get(node.Descriptor().Fields().ByNumber(1)).String()+"]")
						}
					}
				}
			case !f.IsList() && f.Kind() == protoreflect.MessageKind && depth < 2 && path != "version" && path != "program.cleanup":
				walk(path+".", m.Get(f).Message(), depth+1)
			default:
				out = append(out, path)
			}
		}
	}
	walk("", c.ProtoReflect(), 0)
	return out
}

// unaccounted is the first part of the Case that neither a declaration of the realization nor the
// Query accounts for.
func (a *accounting) unaccounted() error {
	for _, part := range parts(a.c) {
		if _, ok := derived[part]; !ok && !a.owned[part] {
			return errorAt(a.l.a.r.GetPosition(), "the Case of query %s carries %s, which no declaration of realization %s accounts for",
				a.l.a.q.GetName(), part, a.l.a.r.GetName())
		}
	}
	return nil
}

// Identity names one Case and its executable and verification artifacts.
type Identity = cp.Identity

func IdentityFor(root, set, query string) Identity { return cp.IdentityFor(root, set, query) }
