package goir

import (
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
)

func (v *validator) exploration(q *modelirspb.Query) {
	e := q.GetExploration()
	if e == nil {
		return
	}
	at := e.GetPosition()
	if at == nil {
		at = q.GetPosition()
	}
	v.once(at, "explorations named", e.GetName())
	var scenario *modelirspb.Scenario
	for _, s := range v.in.model.GetScenarios() {
		if s.GetMachine() == q.GetScenario().GetMachine() && s.GetName() == q.GetScenario().GetName() {
			scenario = s
		}
	}
	if q.GetForm() != modelirspb.Query_FORM_FIND || scenario == nil || scenario.GetFree() || len(scenario.GetActions()) < 2 || q.GetThrough() {
		v.report(at, "exploration %s requires a find Query with a pinned action prefix", e.GetName())
		return
	}
	if e.GetRuns() < 1 || e.GetEdits() < 0 || len(e.GetVariations()) == 0 {
		v.report(at, "exploration %s requires variations, positive runs and nonnegative edits", e.GetName())
	}
	indexes := map[int32]bool{}
	size := int64(1)
	for _, axis := range e.GetVariations() {
		if axis.GetIndex() < 0 || int(axis.GetIndex()) >= len(scenario.GetActions())-1 || indexes[axis.GetIndex()] || len(axis.GetChoices()) == 0 {
			v.report(at, "exploration %s has an empty domain or repeated/non-prefix index %d", e.GetName(), axis.GetIndex())
		}
		indexes[axis.GetIndex()] = true
		v.alternatives(e, scenario, axis, at)
		if len(axis.GetChoices()) > 4096 || size > 4096/int64(max(1, len(axis.GetChoices()))) {
			v.report(at, "exploration %s finite domain exceeds 4096 combinations", e.GetName())
			return
		}
		size *= int64(len(axis.GetChoices()))
	}
}

func (v *validator) alternatives(e *modelirspb.Exploration, scenario *modelirspb.Scenario, axis *modelirspb.Variation, at *modelirspb.Position) {
	names := map[string]bool{}
	for _, choice := range axis.GetChoices() {
		if choice.GetName() == "" || strings.Contains(choice.GetName(), "+") || names[choice.GetName()] {
			v.report(at, "exploration %s has an empty, ambiguous or duplicate alternative name", e.GetName())
		}
		names[choice.GetName()] = true
		for _, action := range choice.GetActions() {
			v.actionClass(e.GetName(), v.machines[scenario.GetMachine()], action, at)
		}
	}
}
