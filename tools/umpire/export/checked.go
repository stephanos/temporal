package export

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

// The names of what confirm declares, which no front end writes.
const (
	anyStep     = "backends.anyStep"
	freeQuery   = "backends.monitors"
	replayQuery = "backends.witness"
)

// confirm holds a backend's monitor verdicts to the reader's own checker, through ordinary admission and
// evaluation: `check.Check` of the Model with, for each machine a monitor agreement is about, a
// verify of a Property that holds of every step over every path from each start, and one over the
// classes of each of the backend's counterexamples. A monitor violated where it is read is the only
// way such a Query fails.
//
// The backend and the checker must say the same of each machine: violated or not, by a monitor the
// backend also found violated. A counterexample whose classes the checker finds no violation on is
// rejected, which is an error and stands before any difference. Without everyPath only the
// counterexamples are asked, for a caller that has one monitor's and not every monitor's verdict.
func (s *Slice) confirm(receipts []Receipt, everyPath bool) error {
	m, ok := proto.Clone(s.Model).(*umpirespb.Model)
	if !ok {
		return errors.New("the Model is no Model")
	}
	// Only the Queries declared here are answered: the Model's own are checked where it is gated.
	m.Queries, m.Progress = nil, nil
	m.Functions = append(m.Functions, &umpirespb.Function{Name: anyStep,
		Params: []*umpirespb.Param{{Name: "step", Type: named(interp.StepType)}},
		Body:   &umpirespb.Expr{Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: true}}}}})
	asked := func(r Receipt) bool { return r.Claim == MonitorAgreement && r.Kind != Unsupported }
	for _, r := range receipts {
		if asked(r) {
			s.ask(m, r, everyPath)
		}
	}
	if len(m.GetQueries()) == 0 {
		return nil
	}
	answers := map[string]check.Receipt{}
	for _, a := range check.Check(m, check.DefaultScope).Receipts {
		if a.Subject == check.ModelSubject {
			return fmt.Errorf("goir does not admit the Model with the backend's Queries: %s", a.Explanation)
		}
		if a.Subject == check.QuerySubject {
			answers[a.Key.Name] = a
		}
	}
	for i, r := range receipts {
		if !asked(r) || r.Kind == WitnessRejected {
			continue
		}
		starts := 0
		if everyPath {
			starts = len(s.machines[r.Subject].Decl.GetStarts())
		}
		receipts[i] = confirmed(r, starts, answers)
	}
	return nil
}

// ask declares the Queries that confirm one machine's receipt: every path from each start, and the
// classes of each counterexample from its own start.
func (s *Slice) ask(m *umpirespb.Model, r Receipt, everyPath bool) {
	mm := s.machines[r.Subject]
	starts := mm.Decl.GetStarts()
	m.Properties = append(m.Properties, &umpirespb.Property{Machine: r.Subject, Name: anyStep, Holds: anyStep})
	verify := func(name string, sc *umpirespb.Scenario, steps int) {
		sc.Machine, sc.Name = r.Subject, name
		m.Scenarios = append(m.Scenarios, sc)
		m.Queries = append(m.Queries, &umpirespb.Query{Name: name, Form: umpirespb.Query_FORM_VERIFY,
			Property: &umpirespb.ClaimRef{Machine: r.Subject, Name: anyStep}, Scenario: &umpirespb.ClaimRef{Machine: r.Subject, Name: name},
			Limits: &umpirespb.Limits{Name: "backends", Steps: int32(steps), Actions: int32(steps), Search: 1 << 20}})
	}
	if everyPath {
		for k, start := range starts {
			verify(fmt.Sprintf("%s.%s.%d", freeQuery, r.Subject, k), &umpirespb.Scenario{Start: start, Free: true}, r.ProductStates+1)
		}
	}
	for _, w := range r.Witnesses {
		// A witness that starts nowhere, or takes a class the machine has none of, was rejected when it
		// was replayed through the table: no Query is declared over it.
		k := slices.Index(mm.Table.Starts, w.Trace.Initial.Value)
		if k < 0 {
			continue
		}
		sc := &umpirespb.Scenario{Start: starts[k]}
		for _, step := range w.Trace.Steps {
			if c := slices.IndexFunc(mm.Classes, func(c interp.Class) bool { return c.Key == step.Action.Value }); c >= 0 {
				sc.Actions = append(sc.Actions, classOf(mm.Classes[c]))
			}
		}
		verify(fmt.Sprintf("%s.%s.%s", replayQuery, r.Subject, w.Monitor), sc, len(sc.GetActions()))
	}
}

// classOf writes a class as the IR names one: its action, and its inputs as literals.
func classOf(c interp.Class) *umpirespb.ActionClass {
	class := &umpirespb.ActionClass{Action: c.Action.GetId()}
	for _, input := range c.Inputs {
		class.Inputs = append(class.Inputs, literal(input))
	}
	return class
}

// confirmed is a monitor agreement with the reader's checker's answers folded in.
func confirmed(r Receipt, starts int, answers map[string]check.Receipt) Receipt {
	for _, w := range r.Witnesses {
		a := answers[fmt.Sprintf("%s.%s.%s", replayQuery, r.Subject, w.Monitor)]
		if a.Kind != check.Counterexample || a.Monitor == "" {
			r.Kind = WitnessRejected
			r.Explanation = fmt.Sprintf("the backend's counterexample of %s did not replay through goir's checker, which answers %q over its classes: %s",
				w.Monitor, a.Kind, a.Explanation)
			return r
		}
	}
	var named []string
	for k := range starts {
		a := answers[fmt.Sprintf("%s.%s.%d", freeQuery, r.Subject, k)]
		switch a.Kind {
		case check.Verified:
		case check.Counterexample:
			named = append(named, a.Monitor)
		default:
			r.Differences = append(r.Differences, fmt.Sprintf("goir's checker answers %q of the monitors from start %d: %s", a.Kind, k, a.Explanation))
		}
	}
	for _, monitor := range named {
		if !slices.Contains(r.Violated, monitor) {
			r.Differences = append(r.Differences, fmt.Sprintf("goir's checker finds the monitor %s violated, and the backend does not", monitor))
		}
	}
	if starts > 0 && len(named) == 0 && len(r.Violated) > 0 {
		r.Differences = append(r.Differences, fmt.Sprintf("the backend finds %s violated, and goir's checker verifies the monitors on every path",
			strings.Join(r.Violated, ", ")))
	}
	if len(r.Differences) > 0 {
		if r.Kind == Agreed {
			r.Explanation = "the backend and goir's checker differ"
		}
		r.Kind = Disagreed
		return r
	}
	if starts > 0 {
		r.Explanation += "; goir's checker answers the same from every start"
	}
	if len(r.Witnesses) > 0 {
		r.Explanation += ", and finds a violation over the classes of each of the backend's counterexamples"
	}
	return r
}

// literal writes a value as the IR writes one.
func literal(v interp.Value) *umpirespb.Value {
	values := func(vs []interp.Value) []*umpirespb.Value {
		out := make([]*umpirespb.Value, len(vs))
		for i, f := range vs {
			out[i] = literal(f)
		}
		return out
	}
	switch v.Kind {
	case interp.BoolValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: v.Bool}}
	case interp.IntValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: v.Int}}
	case interp.TextValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Text{Text: v.Text}}
	case interp.EnumValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: v.Type, Case: v.Case, Fields: values(v.Fields)}}}
	case interp.RecordValue:
		return &umpirespb.Value{Kind: &umpirespb.Value_Record{Record: &umpirespb.RecordValue{Type: v.Type, Fields: values(v.Fields)}}}
	default:
		return &umpirespb.Value{Kind: &umpirespb.Value_List{List: &umpirespb.ListValue{Items: values(v.Items)}}}
	}
}
