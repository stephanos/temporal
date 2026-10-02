package backends

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/scalav2/goir"
)

// maxDifferences is how many differences a receipt lists.
const maxDifferences = 8

// QuintAgreement reads the dump a run of an export wrote, an ITF trace whose first state holds it,
// and compares each machine's part with Go's reading of this slice. A dump that cannot be read is an
// error; a difference is a receipt.
//
// For each exported machine it gives a transition agreement, over every reachable state and class; a
// monitor agreement where the machine names monitors, over every step of the product, with Quint's
// counterexample of each violated monitor replayed through Go; a property agreement where it declares
// Properties, over every step; and what the evaluator covered. What the export leaves out follows as
// unsupported.
func (s *Slice) QuintAgreement(x *QuintExport, itf []byte) ([]Receipt, error) {
	var trace struct {
		States []map[string]any `json:"states"`
	}
	if err := json.Unmarshal(itf, &trace); err != nil {
		return nil, fmt.Errorf("the Quint dump is no ITF trace: %w", err)
	}
	if len(trace.States) == 0 {
		return nil, errors.New("the Quint dump holds no state")
	}
	out, ok := trace.States[0]["out"].(map[string]any)
	if !ok {
		return nil, errors.New("the Quint dump's first state has no variable out")
	}
	var receipts []Receipt
	// Every counterexample of one dump is replayed through one interpretation, made afresh for it.
	var fresh *Slice
	replay := func(machine, monitor string, trace *umpire.Trace) error {
		if fresh == nil {
			var err error
			if fresh, err = Open(s.Model); err != nil {
				return err
			}
		}
		return fresh.replay(machine, monitor, trace)
	}
	for i := range x.Machines {
		compared, err := s.machineReceipts(x, i, out, replay)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, compared...)
	}
	for j := range x.Compositions {
		compared, err := s.compositionReceipts(x, j, out)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, compared...)
	}
	// What the export leaves out is part of what the comparison says: it is listed, and agrees nothing.
	receipts = append(receipts, x.Unsupported...)
	for i := range receipts {
		receipts[i].Model = s.Name
	}
	return receipts, s.confirm(receipts, true)
}

// machineReceipts compares machine i's part of a dump with Go's reading of the machine.
func (s *Slice) machineReceipts(x *QuintExport, i int, out map[string]any, replay func(machine, monitor string, trace *umpire.Trace) error) ([]Receipt, error) {
	name := x.Machines[i]
	part, ok := out[fmt.Sprintf("m%d", i)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the Quint dump has no part m%d for the machine %s", i, name)
	}
	theirs, err := x.reader(i).machine(part)
	if err != nil {
		return nil, fmt.Errorf("the Quint dump of %s: %w", name, err)
	}
	mm := s.machines[name]
	if mm == nil {
		return nil, fmt.Errorf("the export names the machine %s, which the Model does not declare", name)
	}
	ours, err := s.view(mm)
	if err != nil {
		return nil, err
	}
	receipts := []Receipt{transitions(name, ours, theirs)}
	if ours.Product != nil || theirs.Product != nil {
		receipts = append(receipts, watched(mm, ours, theirs, replay))
	}
	if len(ours.Properties) > 0 || theirs.Claims != nil {
		receipts = append(receipts, claimed(name, ours, theirs))
	}
	return append(receipts, coverage(name, theirs)), nil
}

// compositionReceipts compares composition j's part of a dump with the composed table goir's checker
// builds, and with its Properties as the checker binds them.
func (s *Slice) compositionReceipts(x *QuintExport, j int, out map[string]any) ([]Receipt, error) {
	name := x.Compositions[j]
	part, ok := out[fmt.Sprintf("c%d", j)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("the Quint dump has no part c%d for the composition %s", j, name)
	}
	theirs, err := x.composedReader(j).machine(part)
	if err != nil {
		return nil, fmt.Errorf("the Quint dump of %s: %w", name, err)
	}
	c, err := s.bound.Composition(name)
	if err != nil {
		return nil, fmt.Errorf("the export names the composition %s, which Go does not build: %w", name, err)
	}
	ours, err := composedView(c)
	if err != nil {
		return nil, err
	}
	receipts := []Receipt{transitions(name, ours, theirs)}
	if len(ours.Properties) > 0 || theirs.Claims != nil {
		receipts = append(receipts, claimed(name, ours, theirs))
	}
	return append(receipts, coverage(name, theirs)), nil
}

// differences collects what differs between two readings, the first few in full.
type differences struct {
	listed []string
	more   int
}

func (d *differences) add(format string, args ...any) {
	if len(d.listed) < maxDifferences {
		d.listed = append(d.listed, fmt.Sprintf(format, args...))
		return
	}
	d.more++
}

func (d *differences) conclude(r Receipt) Receipt {
	r.Kind, r.Differences = Agreed, d.listed
	if len(d.listed) > 0 {
		r.Kind = Disagreed
		r.Explanation = fmt.Sprintf("%s and Go differ in %d places", r.Backend, len(d.listed)+d.more)
	}
	return r
}

// transitions compares a machine's reachable transition relation: the starts in order, the reachable
// states, the ends among them and the classes as sets, and for every reachable state and class the
// results in order, an empty list being a disabled pair.
func transitions(name string, ours, theirs *machineView) Receipt {
	r := Receipt{Backend: quintBackend, Claim: TransitionAgreement, Subject: name, States: len(ours.Reach)}
	var d differences
	if !slices.Equal(ours.Starts, theirs.Starts) {
		d.add("starts: Go %v, Quint %v", ours.Starts, theirs.Starts)
	}
	if !theirs.Closed {
		d.add("closed: a step leaves the states Quint reached in as many rounds as Go's table is deep")
	}
	sets(&d, "reachable states", ours.Reach, theirs.Reach)
	sets(&d, "ends", ours.Ends, theirs.Ends)
	sets(&d, "classes", ours.Classes, theirs.Classes)
	for _, state := range ours.Reach {
		for _, class := range ours.Classes {
			r.Pairs++
			mine := ours.Rows[state][class]
			// A pair Quint's dump leaves out was not evaluated there: it is not read as disabled.
			yours, evaluated := theirs.Rows[state][class]
			if len(mine) == 0 {
				r.Disabled++
			} else {
				r.Enabled++
				r.Results += len(mine)
			}
			switch {
			case !evaluated:
				d.add("%s by %s: Quint's dump has no such pair", state, class)
			case len(mine) == 0 && len(yours) > 0:
				d.add("%s by %s: disabled in Go, and Quint gives %v", state, class, yours)
			case len(mine) > 0 && len(yours) == 0:
				d.add("%s by %s: disabled in Quint, and Go gives %v", state, class, mine)
			case !slices.EqualFunc(mine, yours, sameResult):
				d.add("%s by %s: the results differ: Go %v, Quint %v", state, class, mine, yours)
			default:
			}
		}
	}
	r.Explanation = fmt.Sprintf("every one of %d reachable states by every one of %d classes: %d enabled pairs with %d results and %d disabled pairs",
		r.States, len(ours.Classes), r.Enabled, r.Results, r.Disabled)
	return d.conclude(r)
}

func sameResult(a, b result) bool {
	return a.Outcome == b.Outcome && a.State == b.State && a.Because == b.Because && slices.Equal(a.Facts, b.Facts)
}

func sets(d *differences, what string, ours, theirs []string) {
	for _, x := range ours {
		if !slices.Contains(theirs, x) {
			d.add("%s: %s is in Go's and not in Quint's", what, x)
		}
	}
	for _, x := range theirs {
		if !slices.Contains(ours, x) {
			d.add("%s: %s is in Quint's and not in Go's", what, x)
		}
	}
}

// watched compares the product of a machine and its monitors, and replays Quint's counterexample
// of every monitor it finds violated. A counterexample that does not replay is an error and stands
// before any difference.
func watched(mm *goir.Machine, ours, theirs *machineView, replay func(machine, monitor string, trace *umpire.Trace) error) Receipt {
	name := mm.Decl.GetName()
	r := Receipt{Backend: quintBackend, Claim: MonitorAgreement, Subject: name}
	var d differences
	switch {
	case theirs.Product == nil:
		d.add("monitors: Quint's dump has no product, and %s names monitors", name)
		return d.conclude(r)
	case ours.Product == nil:
		d.add("monitors: Quint's dump has a product, and %s names no monitor", name)
		return d.conclude(r)
	default:
	}
	mine, yours := ours.Product, theirs.Product
	yours.Monitors = mine.Monitors
	r.ProductStates = len(mine.States)
	if !slices.EqualFunc(mine.Starts, yours.Starts, func(a, b productState) bool { return a.key() == b.key() }) {
		d.add("product starts: Go %v, Quint %v", mine.Starts, yours.Starts)
	}
	if !yours.Closed {
		d.add("closed: a step leaves the product states Quint reached in as many rounds as Go's product is deep")
	}
	sets(&d, "product states", keysOf(mine.States), keysOf(yours.States))
	for _, key := range keysOf(mine.States) {
		for _, class := range ours.Classes {
			r.Steps += len(mine.Steps[key][class])
			taken, evaluated := yours.Steps[key][class]
			if _, reached := yours.States[key]; reached && !evaluated {
				d.add("%s by %s: Quint's product has no such pair", key, class)
				continue
			}
			monitored(&d, mine.Monitors, key+" by "+class, mine.Steps[key][class], taken)
		}
	}
	r.Explanation = fmt.Sprintf("every one of %d steps from %d states of the product with %s", r.Steps, r.ProductStates, strings.Join(mine.Monitors, ", "))
	r = d.conclude(r)
	for k, monitor := range mine.Monitors {
		trace := counterexample(mm.Table, theirs, k)
		if trace == nil {
			continue
		}
		r.Violated = append(r.Violated, monitor)
		r.Witnesses = append(r.Witnesses, Witness{Monitor: monitor, Trace: trace})
		if err := replay(name, monitor, trace); err != nil {
			r.Kind = WitnessRejected
			r.Explanation = fmt.Sprintf("Quint's counterexample of %s did not replay through Go: %v", monitor, err)
			return r
		}
	}
	return r
}

// monitored compares what the results of one row do to the monitors.
func monitored(d *differences, monitors []string, row string, mine, yours []productStep) {
	if len(mine) != len(yours) {
		d.add("%s: Go takes %d steps and Quint %d", row, len(mine), len(yours))
		return
	}
	for n := range mine {
		a, b := mine[n], yours[n]
		for k, monitor := range monitors {
			if a.Mu[k] != b.Mu[k] || a.Read[k] != b.Read[k] || a.Viol[k] != b.Viol[k] {
				d.add("%s, result %d: the monitor %s is %s (read %t, violated %t) in Go and %s (read %t, violated %t) in Quint",
					row, n, monitor, a.Mu[k], a.Read[k], a.Viol[k], b.Mu[k], b.Read[k], b.Viol[k])
			}
		}
	}
}

// claimed compares a machine's Properties: on every step from every reachable state, whether each is
// about the step and whether it holds of it.
func claimed(name string, ours, theirs *machineView) Receipt {
	r := Receipt{Backend: quintBackend, Claim: PropertyAgreement, Subject: name, Properties: len(ours.Properties)}
	var d differences
	if theirs.Claims == nil {
		d.add("properties: Quint's dump has no claims, and %s declares %d Properties", name, len(ours.Properties))
		return d.conclude(r)
	}
	for _, state := range ours.Reach {
		for _, class := range ours.Classes {
			mine := ours.Claims[state][class]
			yours, evaluated := theirs.Claims[state][class]
			switch {
			case !evaluated:
				d.add("%s by %s: Quint's claims have no such pair", state, class)
			case len(mine) != len(yours):
				d.add("%s by %s: Go reads the Properties on %d steps and Quint on %d", state, class, len(mine), len(yours))
			default:
				reads, about := readings(&d, ours.Properties, state+" by "+class, mine, yours)
				r.Reads, r.About = r.Reads+reads, r.About+about
			}
		}
	}
	r.Explanation = fmt.Sprintf("every one of %d Properties on every step from every reachable state: %d readings, %d of them of a step the Property is about",
		r.Properties, r.Reads, r.About)
	return d.conclude(r)
}

// readings compares what the Properties say of each result of one row, and counts the readings
// compared and the ones of a step a Property is about.
func readings(d *differences, properties []string, row string, mine, yours [][]claimRead) (reads, about int) {
	for n := range mine {
		for k, property := range properties {
			reads++
			if mine[n][k].About {
				about++
			}
			if k >= len(yours[n]) || mine[n][k] != yours[n][k] {
				d.add("%s, result %d: Go reads %s as %+v and Quint does not", row, n, property, mine[n][k])
			}
		}
	}
	return reads, about
}

func keysOf[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// hop is one step of a path through a product: the product state it leaves, by its key, and the
// class and result it takes.
type hop struct {
	from   string
	class  string
	result int
}

// counterexample is a shortest path through a backend's product to a step on which the monitor at
// index k is read and violated, as a trace of the machine, or nil when it has none. The path is the
// backend's own: every state, class and result on it is read off its dump.
func counterexample(t *umpire.Table, v *machineView, k int) *umpire.Trace {
	p := v.Product
	parent := map[string]*hop{}
	var queue []string
	for _, start := range p.Starts {
		if _, seen := parent[start.key()]; !seen {
			parent[start.key()] = nil
			queue = append(queue, start.key())
		}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, class := range v.Classes {
			results := v.Rows[p.States[key].State][class]
			for n, step := range p.Steps[key][class] {
				if n >= len(results) || k >= len(step.Mu) {
					continue
				}
				if step.Read[k] && step.Viol[k] {
					return pathTo(t, v, parent, hop{key, class, n})
				}
				next := productState{State: results[n].State, Mu: step.Mu}.key()
				if _, seen := parent[next]; !seen {
					parent[next] = &hop{key, class, n}
					queue = append(queue, next)
				}
			}
		}
	}
	return nil
}

// pathTo is the trace that ends in one step of a product, by the step that first reached each state.
func pathTo(t *umpire.Table, v *machineView, parent map[string]*hop, last hop) *umpire.Trace {
	hops := []hop{last}
	for h := parent[last.from]; h != nil; h = parent[h.from] {
		hops = append(hops, *h)
	}
	slices.Reverse(hops)
	states := v.Product.States
	out := &umpire.Trace{Initial: t.StateAtom(states[hops[0].from].State)}
	for _, h := range hops {
		res := v.Rows[states[h.from].State][h.class][h.result]
		step := umpire.TraceStep{Action: t.ActionAtom(h.class), Outcome: t.OutcomeAtom(res.Outcome), State: t.StateAtom(res.State)}
		for _, f := range res.Facts {
			step.Facts = append(step.Facts, t.FactAtom(f))
		}
		out.Steps = append(out.Steps, step)
	}
	return out
}

// coverage says what Quint explored of a machine, and by what. It is no agreement, and no model
// checker's result.
func coverage(name string, v *machineView) Receipt {
	r := Receipt{Backend: quintBackend, Claim: CheckerCoverage, Subject: name, Kind: Covered, States: len(v.Reach)}
	closed := "to a fixed point"
	if !v.Closed {
		closed = "without reaching a fixed point"
	}
	r.Explanation = fmt.Sprintf("Quint's evaluator enumerated the %d states the starts reach, %s, and every class from each", len(v.Reach), closed)
	if v.Product != nil {
		r.ProductStates = len(v.Product.States)
		r.Explanation += fmt.Sprintf(", and the %d states of the product with the monitors", r.ProductStates)
	}
	r.Explanation += "; no random simulation and no `quint verify` enters this result"
	return r
}
