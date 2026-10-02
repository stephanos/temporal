package golden

import modelirspb "go.temporal.io/server/api/modelir/v1"

func JobModel(name string, actions ...string) *modelirspb.Model {
	const stateType, outcomeType, factType, stepType = "JobState", "JobOutcome", "JobFact", "umpire.Step"
	at := &modelirspb.Position{File: "job.go", Line: 1}
	named := func(name string) *modelirspb.TypeRef {
		return &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_Named{Named: name}}
	}
	literal := func(value *modelirspb.Value) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: value}}
	}
	text := func(value string) *modelirspb.Expr {
		return literal(&modelirspb.Value{Kind: &modelirspb.Value_Text{Text: value}})
	}
	enumValue := func(typ, value string) *modelirspb.Value {
		return &modelirspb.Value{Kind: &modelirspb.Value_Enum{Enum: &modelirspb.EnumValue{Type: typ, Case: value}}}
	}
	enum := func(typ, value string) *modelirspb.Expr { return literal(enumValue(typ, value)) }
	variable := func(name string) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Var{Var: name}}
	}
	field := func(name string) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Field{Field: &modelirspb.FieldAccess{Base: variable("after"), Field: name}}}
	}
	list := func(values ...*modelirspb.Expr) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_List{List: &modelirspb.ListOf{Items: values}}}
	}
	binary := func(op modelirspb.Binary_Op, left, right *modelirspb.Expr) *modelirspb.Expr {
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Binary{Binary: &modelirspb.Binary{Op: op, Left: left, Right: right}}}
	}
	enumeration := func(name string, values ...string) *modelirspb.Type {
		cases := make([]*modelirspb.Case, 0, len(values))
		for _, value := range values {
			cases = append(cases, &modelirspb.Case{Name: value})
		}
		return &modelirspb.Type{Name: name, Position: at, Shape: &modelirspb.Type_Enum{Enum: &modelirspb.Enum{Cases: cases}}}
	}
	step := func(state string, facts ...string) *modelirspb.Expr {
		values := make([]*modelirspb.Expr, 0, len(facts))
		for _, fact := range facts {
			values = append(values, enum(factType, fact))
		}
		return &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Construct{Construct: &modelirspb.Construct{Type: stepType,
			Args: []*modelirspb.Expr{enum(outcomeType, "accepted"), enum(stateType, state), list(values...), text("")}}}}
	}
	facts := []string{"listed", "taken", "count", "finished", "wasDropped"}
	machine := &modelirspb.Machine{Family: "fixture.job", Name: "job", Position: at, StateType: stateType,
		OutcomeType: outcomeType, FactType: factType, Starts: []*modelirspb.Expr{enum(stateType, "idle")}, Evidence: "job.evidence",
		Ends: &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Lambda{Lambda: &modelirspb.Lambda{
			Params: []*modelirspb.Param{{Name: "state", Type: named(stateType)}}, Body: binary(modelirspb.Binary_OP_OR,
				binary(modelirspb.Binary_OP_EQ, variable("state"), enum(stateType, "done")),
				binary(modelirspb.Binary_OP_EQ, variable("state"), enum(stateType, "dropped")))}}}}
	model := &modelirspb.Model{Source: "job.go", Types: []*modelirspb.Type{
		enumeration(stateType, "idle", "queued", "running", "waiting", "done", "dropped"), enumeration(outcomeType, "accepted"), enumeration(factType, facts...)},
		Machines: []*modelirspb.Machine{machine}}
	for _, row := range []struct {
		action, state string
		results       []*modelirspb.Expr
	}{
		{"submit", "idle", []*modelirspb.Expr{step("queued", "listed")}},
		{"take", "queued", []*modelirspb.Expr{step("running", "taken", "count")}},
		{"fail", "running", []*modelirspb.Expr{step("waiting", "listed", "count")}},
		{"settle", "waiting", []*modelirspb.Expr{step("queued")}},
		{"finish", "running", []*modelirspb.Expr{step("done", "finished")}},
		{"drop", "queued", []*modelirspb.Expr{step("dropped", "wasDropped")}},
		{"check", "running", []*modelirspb.Expr{step("running")}},
		{"close", "running", []*modelirspb.Expr{step("done", "finished"), step("dropped", "wasDropped")}},
	} {
		id := "job." + row.action
		model.Actions = append(model.Actions, &modelirspb.Action{Id: id, Name: row.action, Position: at, Party: "fixture"})
		model.Functions = append(model.Functions, &modelirspb.Function{Name: id + ".step", Position: at,
			Params: []*modelirspb.Param{{Name: "state", Type: named(stateType)}}, Body: &modelirspb.Expr{Position: at,
				Kind: &modelirspb.Expr_If{If: &modelirspb.If{Condition: binary(modelirspb.Binary_OP_EQ, variable("state"), enum(stateType, row.state)), Then: list(row.results...), Else: list()}}}})
		machine.Steps = append(machine.Steps, &modelirspb.StepBinding{Action: id, Function: id + ".step", Position: at})
	}
	evidence := &modelirspb.Match{Scrutinee: variable("fact")}
	for _, fact := range facts {
		evidence.Cases = append(evidence.Cases, &modelirspb.MatchCase{Pattern: &modelirspb.Pattern{Kind: &modelirspb.Pattern_Literal{Literal: enumValue(factType, fact)}}, Body: text(fact)})
	}
	model.Functions = append(model.Functions,
		&modelirspb.Function{Name: "job.evidence", Position: at, Params: []*modelirspb.Param{{Name: "fact", Type: named(factType)}}, Body: &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Match{Match: evidence}}},
		&modelirspb.Function{Name: "job.finishes", Position: at, Params: []*modelirspb.Param{{Name: "after", Type: named(stepType)}}, Body: binary(modelirspb.Binary_OP_AND,
			binary(modelirspb.Binary_OP_EQ, field("state"), enum(stateType, "done")), binary(modelirspb.Binary_OP_CONTAINS, enum(factType, "finished"), field("facts")))})
	last := actions[len(actions)-1]
	model.Properties = []*modelirspb.Property{{Machine: "job", Name: "finishes", Position: at, Holds: "job.finishes",
		When: &modelirspb.Property_WhenClass{WhenClass: &modelirspb.ActionClass{Action: "job." + last}}}}
	scenario := &modelirspb.Scenario{Machine: "job", Name: name, Position: at, Start: enum(stateType, "idle")}
	for _, action := range actions {
		scenario.Actions = append(scenario.Actions, &modelirspb.ActionClass{Action: "job." + action})
	}
	model.Scenarios = []*modelirspb.Scenario{scenario}
	model.Queries = []*modelirspb.Query{{Name: name, Position: at, Form: modelirspb.Query_FORM_FIND,
		Property: &modelirspb.ClaimRef{Machine: "job", Name: "finishes"}, Scenario: &modelirspb.ClaimRef{Machine: "job", Name: name},
		Limits: &modelirspb.Limits{Name: "eight", Steps: 8, Actions: 8, Search: 4096}}}
	return model
}
