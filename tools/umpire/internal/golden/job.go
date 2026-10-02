package golden

import umpirespb "go.temporal.io/server/api/umpire/v1"

func JobModel(name string, actions ...string) *umpirespb.Model {
	const stateType, outcomeType, factType, stepType = "JobState", "JobOutcome", "JobFact", "umpire.Step"
	at := &umpirespb.Position{File: "job.go", Line: 1}
	named := func(name string) *umpirespb.TypeRef {
		return &umpirespb.TypeRef{Ref: &umpirespb.TypeRef_Named{Named: name}}
	}
	literal := func(value *umpirespb.Value) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: value}}
	}
	text := func(value string) *umpirespb.Expr {
		return literal(&umpirespb.Value{Kind: &umpirespb.Value_Text{Text: value}})
	}
	enumValue := func(typ, value string) *umpirespb.Value {
		return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: value}}}
	}
	enum := func(typ, value string) *umpirespb.Expr { return literal(enumValue(typ, value)) }
	variable := func(name string) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Var{Var: name}}
	}
	field := func(name string) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Field{Field: &umpirespb.FieldAccess{Base: variable("after"), Field: name}}}
	}
	list := func(values ...*umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{Items: values}}}
	}
	binary := func(op umpirespb.Binary_Op, left, right *umpirespb.Expr) *umpirespb.Expr {
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: op, Left: left, Right: right}}}
	}
	enumeration := func(name string, values ...string) *umpirespb.Type {
		cases := make([]*umpirespb.Case, 0, len(values))
		for _, value := range values {
			cases = append(cases, &umpirespb.Case{Name: value})
		}
		return &umpirespb.Type{Name: name, Position: at, Shape: &umpirespb.Type_Enum{Enum: &umpirespb.Enum{Cases: cases}}}
	}
	step := func(state string, facts ...string) *umpirespb.Expr {
		values := make([]*umpirespb.Expr, 0, len(facts))
		for _, fact := range facts {
			values = append(values, enum(factType, fact))
		}
		return &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Construct{Construct: &umpirespb.Construct{Type: stepType,
			Args: []*umpirespb.Expr{enum(outcomeType, "accepted"), enum(stateType, state), list(values...), text("")}}}}
	}
	facts := []string{"listed", "taken", "count", "finished", "wasDropped"}
	machine := &umpirespb.Machine{Family: "fixture.job", Name: "job", Position: at, StateType: stateType,
		OutcomeType: outcomeType, FactType: factType, Starts: []*umpirespb.Expr{enum(stateType, "idle")}, Evidence: "job.evidence",
		Ends: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Lambda{Lambda: &umpirespb.Lambda{
			Params: []*umpirespb.Param{{Name: "state", Type: named(stateType)}}, Body: binary(umpirespb.Binary_OP_OR,
				binary(umpirespb.Binary_OP_EQ, variable("state"), enum(stateType, "done")),
				binary(umpirespb.Binary_OP_EQ, variable("state"), enum(stateType, "dropped")))}}}}
	model := &umpirespb.Model{Source: "job.go", Types: []*umpirespb.Type{
		enumeration(stateType, "idle", "queued", "running", "waiting", "done", "dropped"), enumeration(outcomeType, "accepted"), enumeration(factType, facts...)},
		Machines: []*umpirespb.Machine{machine}}
	for _, row := range []struct {
		action, state string
		results       []*umpirespb.Expr
	}{
		{"submit", "idle", []*umpirespb.Expr{step("queued", "listed")}},
		{"take", "queued", []*umpirespb.Expr{step("running", "taken", "count")}},
		{"fail", "running", []*umpirespb.Expr{step("waiting", "listed", "count")}},
		{"settle", "waiting", []*umpirespb.Expr{step("queued")}},
		{"finish", "running", []*umpirespb.Expr{step("done", "finished")}},
		{"drop", "queued", []*umpirespb.Expr{step("dropped", "wasDropped")}},
		{"check", "running", []*umpirespb.Expr{step("running")}},
		{"close", "running", []*umpirespb.Expr{step("done", "finished"), step("dropped", "wasDropped")}},
	} {
		id := "job." + row.action
		model.Actions = append(model.Actions, &umpirespb.Action{Id: id, Name: row.action, Position: at, Party: "fixture"})
		model.Functions = append(model.Functions, &umpirespb.Function{Name: id + ".step", Position: at,
			Params: []*umpirespb.Param{{Name: "state", Type: named(stateType)}}, Body: &umpirespb.Expr{Position: at,
				Kind: &umpirespb.Expr_If{If: &umpirespb.If{Condition: binary(umpirespb.Binary_OP_EQ, variable("state"), enum(stateType, row.state)), Then: list(row.results...), Else: list()}}}})
		machine.Steps = append(machine.Steps, &umpirespb.StepBinding{Action: id, Function: id + ".step", Position: at})
	}
	evidence := &umpirespb.Match{Scrutinee: variable("fact")}
	for _, fact := range facts {
		evidence.Cases = append(evidence.Cases, &umpirespb.MatchCase{Pattern: &umpirespb.Pattern{Kind: &umpirespb.Pattern_Literal{Literal: enumValue(factType, fact)}}, Body: text(fact)})
	}
	model.Functions = append(model.Functions,
		&umpirespb.Function{Name: "job.evidence", Position: at, Params: []*umpirespb.Param{{Name: "fact", Type: named(factType)}}, Body: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Match{Match: evidence}}},
		&umpirespb.Function{Name: "job.finishes", Position: at, Params: []*umpirespb.Param{{Name: "after", Type: named(stepType)}}, Body: binary(umpirespb.Binary_OP_AND,
			binary(umpirespb.Binary_OP_EQ, field("state"), enum(stateType, "done")), binary(umpirespb.Binary_OP_CONTAINS, enum(factType, "finished"), field("facts")))})
	last := actions[len(actions)-1]
	model.Properties = []*umpirespb.Property{{Machine: "job", Name: "finishes", Position: at, Holds: "job.finishes",
		When: &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: "job." + last}}}}
	scenario := &umpirespb.Scenario{Machine: "job", Name: name, Position: at, Start: enum(stateType, "idle")}
	for _, action := range actions {
		scenario.Actions = append(scenario.Actions, &umpirespb.ActionClass{Action: "job." + action})
	}
	model.Scenarios = []*umpirespb.Scenario{scenario}
	model.Queries = []*umpirespb.Query{{Name: name, Position: at, Form: umpirespb.Query_FORM_FIND,
		Property: &umpirespb.ClaimRef{Machine: "job", Name: "finishes"}, Scenario: &umpirespb.ClaimRef{Machine: "job", Name: name},
		Limits: &umpirespb.Limits{Name: "eight", Steps: 8, Actions: 8, Search: 4096}}}
	return model
}
