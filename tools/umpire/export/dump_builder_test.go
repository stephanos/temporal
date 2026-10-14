package export

import (
	"encoding/json"
	"fmt"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

func TestMachineDumpReusesOnlyImmutableClassEncodings(t *testing.T) {
	for _, fixture := range []struct {
		model           string
		owners          []string
		product, inputs bool
	}{
		{"activity-standalone-record", []string{"activityRecord", "trustingActivityRecord"}, true, false},
		{"activity-standalone", []string{"activityProduct"}, false, true},
	} {
		t.Run(fixture.model, func(t *testing.T) {
			s := openNamed(t, fixture.model)
			pristine := proto.Clone(s.Model)
			modelBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
			require.NoError(t, err)
			x := exported(t, s)
			claims := false
			for _, owner := range fixture.owners {
				t.Run(owner, func(t *testing.T) {
					i := slices.Index(x.Machines, owner)
					require.GreaterOrEqual(t, i, 0)
					mm := s.machines[owner]
					require.NotEmpty(t, mm.Table.Reachable)
					require.NotEmpty(t, mm.Classes)
					original, err := s.originalDumpOf(x, i, mm)
					require.NoError(t, err)
					current, err := s.dumpOf(x, i, mm)
					require.NoError(t, err)
					encode := func(value any) []byte {
						t.Helper()
						encoded, err := json.Marshal(value)
						require.NoError(t, err)
						return encoded
					}
					want := encode(original)
					require.Equal(t, want, encode(current), "complete selected-owner graph")
					require.Len(t, current["rows"].(map[string]any)["#set"].([]any), len(mm.Table.Reachable))
					if current["claims"] != nil {
						claims = true
						require.NotEmpty(t, current["claims"].(map[string]any)["#set"].([]any))
					}
					if fixture.product {
						require.NotEmpty(t, current["product"].(map[string]any)["edges"].(map[string]any)["#set"].([]any))
					}
					if fixture.inputs {
						require.True(t, slices.ContainsFunc(mm.Classes, func(c interp.Class) bool { return len(c.Inputs) > 0 }))
						require.NotNil(t, current["claims"], "exercise the substituted claim class encoding")
						measure := func(build func() (map[string]any, error)) float64 {
							return testing.AllocsPerRun(1, func() {
								graph, err := build()
								require.NoError(t, err)
								runtime.KeepAlive(graph)
							})
						}
						legacy := measure(func() (map[string]any, error) { return s.originalDumpOf(x, i, mm) })
						indexed := measure(func() (map[string]any, error) { return s.dumpOf(x, i, mm) })
						t.Logf("complete %s builder allocations: original=%g current=%g", owner, legacy, indexed)
						require.Less(t, indexed, legacy*0.9, "reuse encoded classes across every row and claim")
					}
					var changed, untouched map[string]any
					require.NoError(t, json.Unmarshal(want, &changed))
					require.NoError(t, json.Unmarshal(want, &untouched))
					changed["classes"].(map[string]any)["#set"].([]any)[0].(map[string]any)["tag"] = "mutated-copy"
					require.NotEqual(t, want, encode(changed))
					require.Equal(t, want, encode(untouched))
					require.Equal(t, want, encode(current))
					fresh, err := s.dumpOf(x, i, mm)
					require.NoError(t, err)
					require.Equal(t, want, encode(fresh))
					require.True(t, proto.Equal(pristine, s.Model))
					after, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
					require.NoError(t, err)
					require.Equal(t, modelBytes, after)
				})
			}
			require.True(t, claims, "at least one complete selected owner must exercise claims")
			require.True(t, proto.Equal(pristine, s.Model))
			after, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
			require.NoError(t, err)
			require.Equal(t, modelBytes, after)
		})
	}
}

func TestMachineDumpReusesOnlyCanonicalStateEncodings(t *testing.T) {
	for _, fixture := range []struct {
		model  string
		owners []string
	}{
		{"activity-standalone-record", []string{"activityRecord", "trustingActivityRecord"}},
		{"activity-standalone", []string{"activityProduct"}},
	} {
		t.Run(fixture.model, func(t *testing.T) {
			s := openNamed(t, fixture.model)
			x := exported(t, s)
			pristine := proto.Clone(s.Model)
			before, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
			require.NoError(t, err)
			for _, owner := range fixture.owners {
				t.Run(owner, func(t *testing.T) {
					i := slices.Index(x.Machines, owner)
					require.GreaterOrEqual(t, i, 0)
					mm := s.machines[owner]
					encode := func(graph map[string]any, err error) []byte {
						t.Helper()
						require.NoError(t, err)
						encoded, err := json.Marshal(graph)
						require.NoError(t, err)
						return encoded
					}
					original, err := s.noStateReuseDumpOf(x, i, mm)
					want := encode(original, err)
					current, err := s.dumpOf(x, i, mm)
					require.Equal(t, want, encode(current, err), "complete owner including claims/product")
					rows := current["rows"].(map[string]any)["#set"].([]any)
					require.Len(t, rows, len(mm.Table.Reachable))
					require.Len(t, current["reach"].(map[string]any)["#set"].([]any), len(mm.Table.Reachable))
					originalRows := original["rows"].(map[string]any)["#set"].([]any)
					results := 0
					for n, raw := range rows {
						by := raw.(map[string]any)["by"].(map[string]any)["#set"].([]any)
						oldBy := originalRows[n].(map[string]any)["by"].(map[string]any)["#set"].([]any)
						require.Len(t, by, len(mm.Classes))
						for c, entry := range by {
							steps := entry.(map[string]any)["steps"].([]any)
							oldSteps := oldBy[c].(map[string]any)["steps"].([]any)
							require.Len(t, steps, len(oldSteps))
							for k, result := range steps {
								require.Equal(t, oldSteps[k].(map[string]any)["f_state"], result.(map[string]any)["f_state"], "every result state")
								results++
							}
						}
					}
					require.Positive(t, results)
					view, err := s.view(mm)
					require.NoError(t, err)
					require.NotEmpty(t, view.Properties)
					claimRows := current["claims"].(map[string]any)["#set"].([]any)
					require.Len(t, claimRows, len(mm.Table.Reachable))
					for n, raw := range claimRows {
						by := raw.(map[string]any)["by"].(map[string]any)["#set"].([]any)
						require.Len(t, by, len(mm.Classes))
						for c, entry := range by {
							require.Len(t, entry.(map[string]any)["steps"].([]any), len(view.Claims[mm.Table.Reachable[n]][mm.Classes[c].Key]))
						}
					}
					if fixture.model == "activity-standalone-record" {
						p, err := s.product(mm)
						require.NoError(t, err)
						edges := current["product"].(map[string]any)["edges"].(map[string]any)["#set"].([]any)
						require.NotEmpty(t, edges)
						require.Len(t, edges, len(p.States))
						for n, raw := range edges {
							by := raw.(map[string]any)["by"].(map[string]any)["#set"].([]any)
							require.Len(t, by, len(mm.Classes))
							for c, entry := range by {
								require.Len(t, entry.(map[string]any)["steps"].([]any), len(p.Steps[keysOf(p.States)[n]][mm.Classes[c].Key]))
							}
						}
					}
					var changed, untouched map[string]any
					require.NoError(t, json.Unmarshal(want, &changed))
					require.NoError(t, json.Unmarshal(want, &untouched))
					firstSource := changed["rows"].(map[string]any)["#set"].([]any)[0].(map[string]any)["src"].(map[string]any)
					firstSource["f_phase"] = "changed-state-copy"
					require.NotEqual(t, want, encode(changed, nil))
					require.Equal(t, want, encode(untouched, nil))
					require.Equal(t, want, encode(current, nil))
					require.Equal(t, want, encode(s.dumpOf(x, i, mm)))
					require.True(t, proto.Equal(pristine, s.Model))
					after, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
					require.NoError(t, err)
					require.Equal(t, before, after)
				})
			}
		})
	}
}

func TestMachineDumpCanonicalStateCostOnCompleteSystem(t *testing.T) {
	s := openNamed(t, "activity-standalone")
	x := exported(t, s)
	i := slices.Index(x.Machines, "activitySystem")
	require.GreaterOrEqual(t, i, 0)
	mm := s.machines["activitySystem"]
	require.NotNil(t, mm)
	require.Len(t, mm.Table.States, 5616)
	require.Len(t, mm.Classes, 119)
	require.Len(t, mm.Table.Rows, 121176)
	pristine := proto.CloneOf(s.Model)
	before, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
	require.NoError(t, err)
	transitions := map[string]map[string]interp.Transition{}
	allResults, reachableResults := 0, 0
	for _, tr := range mm.Transitions {
		allResults += len(tr.Steps)
		if transitions[tr.Source.Key()] == nil {
			transitions[tr.Source.Key()] = map[string]interp.Transition{}
		}
		transitions[tr.Source.Key()][tr.Class.Key] = tr
	}
	for _, key := range mm.Table.Reachable {
		for _, class := range mm.Classes {
			reachableResults += len(transitions[key][class.Key].Steps)
		}
	}
	require.Len(t, mm.Table.Reachable, 1670)
	require.Len(t, mm.Transitions, 121176)
	require.Equal(t, 121176, allResults)
	require.Equal(t, 22812, reachableResults)
	inventory := func(graph map[string]any, verifyStates bool) map[string]int {
		out := map[string]int{"starts": len(graph["starts"].([]any)), "reach": len(graph["reach"].(map[string]any)["#set"].([]any)), "ends": len(graph["ends"].(map[string]any)["#set"].([]any)), "classes": len(graph["classes"].(map[string]any)["#set"].([]any))}
		rows := graph["rows"].(map[string]any)["#set"].([]any)
		require.Len(t, rows, len(mm.Table.Reachable))
		out["rows"] = len(rows)
		for n, raw := range rows {
			by := raw.(map[string]any)["by"].(map[string]any)["#set"].([]any)
			require.Len(t, by, len(mm.Classes))
			out["rowClassCells"] += len(by)
			for c, entry := range by {
				steps := entry.(map[string]any)["steps"].([]any)
				literal := transitions[mm.Table.Reachable[n]][mm.Classes[c].Key].Steps
				require.Len(t, steps, len(literal))
				out["results"] += len(steps)
				if verifyStates {
					for k, result := range steps {
						require.Equal(t, x.itf(literal[k].Fields[1], named(mm.Decl.GetStateType())), result.(map[string]any)["f_state"], "every complete literal result state")
					}
				}
			}
		}
		require.Equal(t, reachableResults, out["results"])
		claimRows := graph["claims"].(map[string]any)["#set"].([]any)
		require.NotEmpty(t, claimRows)
		require.Len(t, claimRows, len(mm.Table.Reachable))
		out["claimRows"] = len(claimRows)
		for _, raw := range claimRows {
			by := raw.(map[string]any)["by"].(map[string]any)["#set"].([]any)
			require.Len(t, by, len(mm.Classes))
			out["claimClassCells"] += len(by)
			for _, entry := range by {
				for _, step := range entry.(map[string]any)["steps"].([]any) {
					out["claimResults"]++
					out["claimReadings"] += len(step.(map[string]any))
				}
			}
		}
		require.Equal(t, reachableResults, out["claimResults"])
		require.Nil(t, graph["product"], "the full System owner has no monitors; monitored owners retain their separate proof")
		return out
	}
	measure := func(build func() (map[string]any, error), verifyStates bool) ([]byte, map[string]int) {
		t.Helper()
		graph, err := build()
		require.NoError(t, err)
		counts := inventory(graph, verifyStates)
		encoded, err := json.Marshal(graph)
		require.NoError(t, err)
		runtime.KeepAlive(graph)
		return encoded, counts
	}
	old, oldCounts := measure(func() (map[string]any, error) { return s.noStateReuseDumpOf(x, i, mm) }, false)
	current, currentCounts := measure(func() (map[string]any, error) { return s.dumpOf(x, i, mm) }, true)
	require.True(t, slices.Equal(old, current), "complete owner canonical JSON bytes")
	require.Equal(t, map[string]int{"starts": 1, "reach": 1670, "ends": 885, "classes": 119, "rows": 1670, "rowClassCells": 198730, "results": 22812, "claimRows": 1670, "claimClassCells": 198730, "claimResults": 22812, "claimReadings": 661548}, oldCounts)
	require.Equal(t, oldCounts, currentCounts)
	require.True(t, proto.Equal(pristine, s.Model))
	after, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
	require.NoError(t, err)
	require.Equal(t, before, after)
	fresh, freshCounts := measure(func() (map[string]any, error) { return s.dumpOf(x, i, mm) }, true)
	require.True(t, slices.Equal(old, fresh), "fresh complete builder bytes")
	require.Equal(t, oldCounts, freshCounts)
	require.True(t, proto.Equal(pristine, s.Model))
	final, err := proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
	require.NoError(t, err)
	require.Equal(t, before, final)
	allocations := func(build func() (map[string]any, error)) float64 {
		return testing.AllocsPerRun(1, func() {
			graph, err := build()
			require.NoError(t, err)
			runtime.KeepAlive(graph)
		})
	}
	originalAllocs := allocations(func() (map[string]any, error) { return s.noStateReuseDumpOf(x, i, mm) })
	currentAllocs := allocations(func() (map[string]any, error) { return s.dumpOf(x, i, mm) })
	t.Logf("complete System state allocations: original=%g current=%g", originalAllocs, currentAllocs)
	require.True(t, proto.Equal(pristine, s.Model))
	final, err = proto.MarshalOptions{Deterministic: true}.Marshal(s.Model)
	require.NoError(t, err)
	require.Equal(t, before, final)
	require.Less(t, currentAllocs, originalAllocs*0.9, "canonical state sharing reduces complete eight-field System allocations")
}

func TestMachineDumpPreservesNoncanonicalStateEncoding(t *testing.T) {
	s := openNamed(t, "activity-standalone")
	x := exported(t, s)
	i := slices.Index(x.Machines, "activityProduct")
	require.GreaterOrEqual(t, i, 0)
	base := s.machines["activityProduct"]
	clone := func(v interp.Value) interp.Value { return cloneDumpState(v) }
	for _, mode := range []string{"same-key", "outside-catalog", "invalid-enum"} {
		t.Run(mode, func(t *testing.T) {
			mm := *base
			mm.Transitions = slices.Clone(base.Transitions)
			tr := slices.IndexFunc(mm.Transitions, func(tr interp.Transition) bool { return len(tr.Steps) > 0 })
			require.GreaterOrEqual(t, tr, 0)
			mm.Transitions[tr].Steps = slices.Clone(mm.Transitions[tr].Steps)
			step := clone(mm.Transitions[tr].Steps[0])
			v := clone(step.Fields[1])
			key := v.Key()
			var mutate func(*interp.Value) bool
			mutate = func(value *interp.Value) bool {
				if mode == "invalid-enum" && value.Kind == interp.EnumValue {
					value.Case = "not-a-declared-tag"
					return true
				}
				if mode != "invalid-enum" && value.Kind == interp.EnumValue {
					originalKey := value.Key()
					value.Kind = interp.TextValue
					if mode == "same-key" {
						value.Text = originalKey
						for _, key := range base.Table.Reachable {
							other, ok := base.State(key)
							if ok && other.Fields[0].Case != value.Case {
								value.Case = other.Fields[0].Case
								return true
							}
						}
						return false
					}
					value.Text = "not-a-catalog-state-key"
					return true
				}
				for n := range value.Fields {
					if mutate(&value.Fields[n]) {
						return true
					}
				}
				return false
			}
			require.True(t, mutate(&v), "private fixture includes the targeted typed field")
			if mode == "same-key" {
				require.Equal(t, key, v.Key())
				canonical, ok := base.State(key)
				require.True(t, ok)
				require.False(t, canonical.Equal(v))
			} else {
				_, ok := base.State(v.Key())
				require.False(t, ok)
			}
			step.Fields[1] = v
			mm.Transitions[tr].Steps[0] = step
			build := func(fn func(*QuintExport, int, *interp.Machine) (map[string]any, error)) (encoded []byte, failure any) {
				defer func() { failure = recover() }()
				graph, err := fn(x, i, &mm)
				require.NoError(t, err)
				encoded, err = json.Marshal(graph)
				require.NoError(t, err)
				return
			}
			want, oldPanic := build(s.noStateReuseDumpOf)
			got, newPanic := build(s.dumpOf)
			require.Equal(t, oldPanic, newPanic, "exact native panic")
			require.Equal(t, want, got, "exact literal noncanonical state encoding")
			if mode == "invalid-enum" {
				require.NotNil(t, oldPanic)
			} else {
				require.Nil(t, oldPanic)
				require.NotEmpty(t, got)
			}
		})
	}
	for _, mode := range []string{"same-key", "changed-key"} {
		t.Run("stale-catalog/"+mode, func(t *testing.T) {
			built, err := interp.Build(proto.CloneOf(s.Model))
			require.NoError(t, err)
			mm := built["activityProduct"]
			require.NotNil(t, mm)
			key := mm.Table.Reachable[0]
			value, ok := mm.State(key)
			require.True(t, ok)
			saved := cloneDumpState(value)
			defer func() { copy(value.Fields, saved.Fields) }()
			if mode == "changed-key" {
				for _, otherKey := range mm.Table.Reachable {
					other, ok := mm.State(otherKey)
					if ok && other.Fields[0].Case != value.Fields[0].Case {
						value.Fields[0].Case = other.Fields[0].Case
						break
					}
				}
				require.NotEqual(t, key, value.Key())
			} else {
				value.Fields[0].Kind = interp.TextValue
				value.Fields[0].Text = saved.Fields[0].Key()
				require.Equal(t, key, value.Key())
				require.False(t, saved.Equal(value))
			}
			build := func(fn func(*QuintExport, int, *interp.Machine) (map[string]any, error)) (encoded []byte, failure any, returned error) {
				defer func() { failure = recover() }()
				graph, returned := fn(x, i, mm)
				if returned != nil {
					return nil, nil, returned
				}
				encoded, returned = json.Marshal(graph)
				return encoded, nil, returned
			}
			want, oldPanic, oldErr := build(s.noStateReuseDumpOf)
			got, newPanic, newErr := build(s.dumpOf)
			require.Equal(t, oldErr, newErr)
			require.Equal(t, oldPanic, newPanic)
			require.Equal(t, want, got)
			copy(value.Fields, saved.Fields)
			restored, ok := mm.State(key)
			require.True(t, ok)
			require.True(t, saved.Equal(restored))
		})
	}
}

// noStateReuseDumpOf preserves the independent class-reuse/state-unshared builder.
func (s *Slice) noStateReuseDumpOf(x *QuintExport, i int, mm *interp.Machine) (map[string]any, error) {
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
	class := func(c interp.Class) any { return x.classITF(i, c) }
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
	from := map[string]map[string]interp.Transition{}
	for _, tr := range mm.Transitions {
		if from[tr.Source.Key()] == nil {
			from[tr.Source.Key()] = map[string]interp.Transition{}
		}
		from[tr.Source.Key()][tr.Class.Key] = tr
	}
	rows := []any{}
	for _, key := range t.Reachable {
		by := []any{}
		for n, c := range mm.Classes {
			steps := []any{}
			for _, st := range from[key][c.Key].Steps {
				facts := []any{}
				for _, f := range st.Fields[2].Items {
					facts = append(facts, x.itf(f, named(decl.GetFactType())))
				}
				steps = append(steps, map[string]any{"f_outcome": x.itf(st.Fields[0], named(decl.GetOutcomeType())),
					"f_state": x.itf(st.Fields[1], stateType), "f_facts": facts, "f_because": st.Fields[3].Text, "f_choice": st.Choice})
			}
			by = append(by, map[string]any{"cls": classes[n], "steps": steps})
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
			for n, c := range mm.Classes {
				steps := []any{}
				for _, reads := range view.Claims[key][c.Key] {
					rec := map[string]any{}
					for k, read := range reads {
						rec[fmt.Sprintf("p%d", k)] = map[string]any{"about": read.About, "holds": read.Holds}
					}
					steps = append(steps, rec)
				}
				by = append(by, map[string]any{"cls": classes[n], "steps": steps})
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
		for n, c := range mm.Classes {
			steps := []any{}
			for _, st := range p.Steps[key][c.Key] {
				steps = append(steps, map[string]any{"mu": mu(st.Mu), "read": flags(st.Read), "viol": flags(st.Viol)})
			}
			by = append(by, map[string]any{"cls": classes[n], "steps": steps})
		}
		edges = append(edges, map[string]any{"src": pstate(p.States[key]), "by": set(by)})
	}
	out["product"] = map[string]any{"starts": starts, "closed": true, "edges": set(edges)}
	return out, nil
}

// originalDumpOf preserves the independent pre-reuse builder.
func (s *Slice) originalDumpOf(x *QuintExport, i int, mm *interp.Machine) (map[string]any, error) {
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
	class := func(c interp.Class) any { return x.classITF(i, c) }
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
	from := map[string]map[string]interp.Transition{}
	for _, tr := range mm.Transitions {
		if from[tr.Source.Key()] == nil {
			from[tr.Source.Key()] = map[string]interp.Transition{}
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
					"f_state": x.itf(st.Fields[1], stateType), "f_facts": facts, "f_because": st.Fields[3].Text, "f_choice": st.Choice})
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
