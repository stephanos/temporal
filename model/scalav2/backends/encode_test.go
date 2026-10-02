package backends

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/scalav2/goir"
)

// itf writes a value of an IR type as the export's module would hold it in an ITF trace.
func (x *QuintExport) itf(v goir.Value, t *modelirspb.TypeRef) any {
	switch r := t.GetRef().(type) {
	case *modelirspb.TypeRef_Bool:
		return v.Bool
	case *modelirspb.TypeRef_IntRange, *modelirspb.TypeRef_Int:
		return map[string]any{"#bigint": strconv.FormatInt(v.Int, 10)}
	case *modelirspb.TypeRef_List:
		out := []any{}
		for _, item := range v.Items {
			out = append(out, x.itf(item, r.List))
		}
		return out
	default:
		decl := x.from.types[t.GetNamed()]
		record := func(fields []*modelirspb.Field) any {
			out := map[string]any{}
			for i, f := range fields {
				out["f_"+plain(f.GetName())] = x.itf(v.Fields[i], f.GetType())
			}
			return out
		}
		if decl.GetRecord() != nil {
			return record(decl.GetRecord().GetFields())
		}
		for tag, variant := range x.tags {
			if variant.typ != t.GetNamed() || variant.enum.GetName() != v.Case {
				continue
			}
			if len(variant.enum.GetFields()) == 0 {
				return map[string]any{"tag": tag, "value": map[string]any{"#tup": []any{}}}
			}
			return map[string]any{"tag": tag, "value": record(variant.enum.GetFields())}
		}
		panic("no variant for " + v.Key())
	}
}

func set(items []any) any { return map[string]any{"#set": items} }

// classITF writes a class of machine i as the module holds it.
func (x *QuintExport) classITF(i int, c goir.Class) any {
	j := slices.IndexFunc(x.bindings[i], func(a *modelirspb.Action) bool { return a.GetId() == c.Action.GetId() })
	value := map[string]any{"#tup": []any{}}
	if len(c.Inputs) > 0 {
		value = map[string]any{}
		for n, in := range c.Inputs {
			value[fmt.Sprintf("i%d", n)] = x.itf(in, c.Action.GetInputs()[n].GetType())
		}
	}
	return map[string]any{"tag": fmt.Sprintf("K%d_%d", i, j), "value": value}
}

// dumpOfComposition is composition j's part of a dump as goir's composed table gives it.
func (s *Slice) dumpOfComposition(x *QuintExport, j int) (map[string]any, error) {
	c, err := s.bound.Composition(x.Compositions[j])
	if err != nil {
		return nil, err
	}
	view, err := composedView(c)
	if err != nil {
		return nil, err
	}
	meta, t := x.composed[j], c.Table
	stateType := named(c.Decl.GetStateType())
	state := func(key string) any {
		v, _ := c.State(key)
		return x.itf(v, stateType)
	}
	states := func(keys []string) []any {
		out := []any{}
		for _, k := range keys {
			out = append(out, state(k))
		}
		return out
	}
	// Every composed class by its key: a member's own, and each pair a sync takes.
	classes := map[string]any{}
	classesOf := func(m int) []goir.Class { return s.machines[x.Machines[meta.members[m].machine]].Classes }
	for m, mb := range meta.members {
		for _, k := range classesOf(m) {
			classes[mb.field+"_"+k.Key] = map[string]any{"tag": fmt.Sprintf("C%d_o%d", j, m), "value": x.classITF(mb.machine, k)}
		}
	}
	for n, sync := range meta.syncs {
		for _, a := range classesOf(sync.first) {
			for _, b := range classesOf(sync.second) {
				if a.Action.GetName() != sync.firstAction || b.Action.GetName() != sync.secondAction {
					continue
				}
				key := sync.name + strings.TrimPrefix(a.Key, sync.firstAction) + strings.TrimPrefix(b.Key, sync.secondAction)
				classes[key] = map[string]any{"tag": fmt.Sprintf("C%d_s%d", j, n), "value": map[string]any{
					"a": x.classITF(meta.members[sync.first].machine, a), "b": x.classITF(meta.members[sync.second].machine, b)}}
			}
		}
	}
	all, rows, claims := []any{}, []any{}, []any{}
	for _, key := range t.Actions {
		all = append(all, classes[key])
	}
	for _, key := range t.States {
		by, read := []any{}, []any{}
		for _, class := range t.Actions {
			steps, readings := []any{}, []any{}
			for n, r := range view.Rows[key][class] {
				facts := []any{}
				for _, f := range r.Facts {
					facts = append(facts, f)
				}
				steps = append(steps, map[string]any{"f_outcome": r.Outcome, "f_state": state(r.State), "f_facts": facts, "f_because": r.Because})
				rec := map[string]any{}
				for k, reading := range view.Claims[key][class][n] {
					rec[fmt.Sprintf("p%d", k)] = map[string]any{"about": reading.About, "holds": reading.Holds}
				}
				readings = append(readings, rec)
			}
			by = append(by, map[string]any{"cls": classes[class], "steps": steps})
			read = append(read, map[string]any{"cls": classes[class], "steps": readings})
		}
		rows = append(rows, map[string]any{"src": state(key), "by": set(by)})
		claims = append(claims, map[string]any{"src": state(key), "by": set(read)})
	}
	out := map[string]any{"starts": states(t.Starts), "reach": set(states(t.States)), "closed": true, "ends": set(states(t.Ends)),
		"classes": set(all), "rows": set(rows)}
	if len(view.Properties) > 0 {
		out["claims"] = set(claims)
	}
	return out, nil
}

// dumpOf is machine i's part of a dump as Go's interpretation gives it.
func (s *Slice) dumpOf(x *QuintExport, i int, mm *goir.Machine) (map[string]any, error) {
	decl, t := mm.Decl, mm.Table
	stateType := named(decl.GetStateType())
	state := func(key string) any {
		v, _ := mm.State(key)
		return x.itf(v, stateType)
	}
	states := func(keys []string) []any {
		out := []any{}
		for _, k := range keys {
			out = append(out, state(k))
		}
		return out
	}
	class := func(c goir.Class) any { return x.classITF(i, c) }
	classes := []any{}
	for _, c := range mm.Classes {
		classes = append(classes, class(c))
	}
	var ends []string
	for _, e := range t.Ends {
		if slices.Contains(t.Reachable, e) {
			ends = append(ends, e)
		}
	}
	from := map[string]map[string]goir.Transition{}
	for _, tr := range mm.Transitions {
		if from[tr.Source.Key()] == nil {
			from[tr.Source.Key()] = map[string]goir.Transition{}
		}
		from[tr.Source.Key()][tr.Class.Key] = tr
	}
	rows := []any{}
	for _, key := range t.Reachable {
		by := []any{}
		for _, c := range mm.Classes {
			steps := []any{}
			for _, st := range from[key][c.Key].Steps {
				facts := []any{}
				for _, f := range st.Fields[2].Items {
					facts = append(facts, x.itf(f, named(decl.GetFactType())))
				}
				steps = append(steps, map[string]any{"f_outcome": x.itf(st.Fields[0], named(decl.GetOutcomeType())),
					"f_state": x.itf(st.Fields[1], stateType), "f_facts": facts, "f_because": st.Fields[3].Text})
			}
			by = append(by, map[string]any{"cls": class(c), "steps": steps})
		}
		rows = append(rows, map[string]any{"src": state(key), "by": set(by)})
	}
	out := map[string]any{"starts": states(t.Starts), "reach": set(states(t.Reachable)), "closed": true, "ends": set(states(ends)),
		"classes": set(classes), "rows": set(rows)}
	if view, err := s.view(mm); err != nil {
		return nil, err
	} else if len(view.Properties) > 0 {
		claims := []any{}
		for _, key := range t.Reachable {
			by := []any{}
			for _, c := range mm.Classes {
				steps := []any{}
				for _, reads := range view.Claims[key][c.Key] {
					rec := map[string]any{}
					for k, read := range reads {
						rec[fmt.Sprintf("p%d", k)] = map[string]any{"about": read.About, "holds": read.Holds}
					}
					steps = append(steps, rec)
				}
				by = append(by, map[string]any{"cls": class(c), "steps": steps})
			}
			claims = append(claims, map[string]any{"src": state(key), "by": set(by)})
		}
		out["claims"] = set(claims)
	}
	if len(mm.Monitors) == 0 {
		return out, nil
	}
	p, err := s.product(mm)
	if err != nil {
		return nil, err
	}
	w, _, err := s.watching(mm)
	if err != nil {
		return nil, err
	}
	mu := func(keys []string) any {
		rec := map[string]any{}
		for k, key := range keys {
			rec[fmt.Sprintf("m%d", k)] = x.itf(w.states[k][key], mm.Monitors[k].GetState())
		}
		return rec
	}
	flags := func(bits []bool) any {
		rec := map[string]any{}
		for k, b := range bits {
			rec[fmt.Sprintf("m%d", k)] = b
		}
		return rec
	}
	pstate := func(ps productState) any { return map[string]any{"s": state(ps.State), "mu": mu(ps.Mu)} }
	starts, edges := []any{}, []any{}
	for _, ps := range p.Starts {
		starts = append(starts, pstate(ps))
	}
	for _, key := range keysOf(p.States) {
		by := []any{}
		for _, c := range mm.Classes {
			steps := []any{}
			for _, st := range p.Steps[key][c.Key] {
				steps = append(steps, map[string]any{"mu": mu(st.Mu), "read": flags(st.Read), "viol": flags(st.Viol)})
			}
			by = append(by, map[string]any{"cls": class(c), "steps": steps})
		}
		edges = append(edges, map[string]any{"src": pstate(p.States[key]), "by": set(by)})
	}
	out["product"] = map[string]any{"starts": starts, "closed": true, "edges": set(edges)}
	return out, nil
}
