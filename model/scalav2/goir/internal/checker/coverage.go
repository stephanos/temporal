package checker

import (
	"reflect"
	"strings"
)

// CoverageGoal is what an exploratory set sets out to reach.
type CoverageGoal string

const (
	CoverRows         CoverageGoal = "rows"
	CoverResults      CoverageGoal = "results"
	CoverClassMembers CoverageGoal = "classMembers"
)

// CoverageTarget is one thing an exploration is asked to reach, in the JSON form the Lean golden
// pins (`Umpire.Command.CoverageTarget.json`).
type CoverageTarget struct {
	Kind    string   `json:"kind"`
	Key     string   `json:"key,omitempty"`
	State   string   `json:"state,omitempty"`
	Action  string   `json:"action,omitempty"`
	Results []string `json:"results,omitempty"`
	Outcome string   `json:"outcome,omitempty"`
	Member  string   `json:"member,omitempty"`
	Field   string   `json:"field,omitempty"`
	Class   string   `json:"class,omitempty"`
	Example string   `json:"example,omitempty"`
}

// CoverageTargets enumerates an exploratory set's targets as `Umpire.Command.coverageTargets`
// does: the rows an exploration within the budget's steps can take, in table order; the outcomes
// those rows reach, in catalog order; and the claims of the classes those rows' actions make, in
// claim order; per goal in the order given; cut at the budget's search count.
func CoverageTargets(model Model, goals []CoverageGoal, budget Limits) ([]CoverageTarget, error) {
	t, err := model.Table()
	if err != nil {
		return nil, err
	}
	owner := t.owner()
	sources := within(t.Rows, t.Starts, budget.Steps-1)
	inSources := map[string]bool{}
	for _, s := range sources {
		inSources[s] = true
	}
	var rows []Row
	for _, r := range t.Rows {
		// A row with no result is a disabled pair, which no exploration takes.
		if inSources[r.Source] && len(r.Results) > 0 {
			rows = append(rows, r)
		}
	}
	var rowTargets, resultTargets, memberTargets []CoverageTarget
	reached := map[string]bool{}
	actionsTaken := map[string]bool{}
	for _, r := range rows {
		var outcomes []string
		for _, res := range r.Results {
			outcomes = append(outcomes, t.Family.ID("outcome", owner, res.Outcome))
			reached[res.Outcome] = true
		}
		actionsTaken[r.Action] = true
		rowTargets = append(rowTargets, CoverageTarget{Kind: "row", Key: r.Key,
			State: t.Family.ID("state", owner, r.Source), Action: t.Family.ID("action", owner, r.Action),
			Results: outcomes})
	}
	for _, o := range t.Outcomes {
		if reached[o] {
			resultTargets = append(resultTargets, CoverageTarget{Kind: "result",
				Outcome: t.Family.ID("outcome", owner, o)})
		}
	}
	for _, claim := range t.claims() {
		if actionsTaken[claim.classKey] {
			memberTargets = append(memberTargets, CoverageTarget{Kind: "classMember",
				Member: t.Family.ID("action", owner, claim.classKey),
				Action: string(t.Family) + ".action." + claim.decl.Name,
				Field:  claim.decl.Inputs[0], Class: claim.spelling, Example: claim.example})
		}
	}
	var targets []CoverageTarget
	for _, g := range goals {
		switch g {
		case CoverRows:
			targets = append(targets, rowTargets...)
		case CoverResults:
			targets = append(targets, resultTargets...)
		case CoverClassMembers:
			targets = append(targets, memberTargets...)
		default:
			return nil, errorf("coverage", "unknown goal %q", g)
		}
	}
	if len(targets) > budget.Search {
		targets = targets[:budget.Search]
	}
	return targets, nil
}

// within is the states reached within depth sweeps of the rows from the starts. Each sweep folds
// over the rows in table order and may take a row whose source the same sweep added, exactly as
// the Lean `within` does.
func within(rows []Row, starts []string, depth int) []string {
	seen := append([]string{}, starts...)
	in := map[string]bool{}
	for _, s := range seen {
		in[s] = true
	}
	for range max(depth, 0) {
		for _, r := range rows {
			if !in[r.Source] {
				continue
			}
			for _, res := range r.Results {
				if !in[res.State] {
					in[res.State] = true
					seen = append(seen, res.State)
				}
			}
		}
	}
	return seen
}

type claim struct {
	decl     *ActionDecl
	classKey string
	spelling string
	example  string
}

// claims lists the Abstraction Claims of the actions a machine binds, in the order the actions'
// classes first appear and then in declaration order.
func (t *Table) claims() []claim {
	var out []claim
	seen := map[*ActionDecl]bool{}
	for _, a := range t.Actions {
		c, ok := t.classes[a]
		if !ok || seen[c.Decl] {
			continue
		}
		seen[c.Decl] = true
		for _, ex := range c.Decl.Examples {
			v := reflect.ValueOf(ex.Value)
			key := c.Decl.Name + "-" + keyOf(v, c.Decl.types[0])
			out = append(out, claim{decl: c.Decl, classKey: key, spelling: classSpelling(v, c.Decl.types[0]),
				example: ex.Example})
		}
	}
	return out
}

// classSpelling spells a class the way a Lean `examples:` line does: a variant by its constructor
// and named fields, `handlerError (retryable := true)`, and anything else by its key.
func classSpelling(v reflect.Value, static reflect.Type) string {
	if static.Kind() != reflect.Interface {
		return keyOf(v, static)
	}
	name := lowerFirst(v.Type().Name())
	var fields []string
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if f.IsExported() {
			fields = append(fields, fieldName(f)+" := "+keyOf(v.Field(i), f.Type))
		}
	}
	if len(fields) == 0 {
		return name
	}
	return name + " (" + strings.Join(fields, ", ") + ")"
}
