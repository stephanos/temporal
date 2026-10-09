package export

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

func equalOpenMachines(t *testing.T, old, current *Slice) {
	t.Helper()
	require.Len(t, current.machines, len(old.machines))
	for name, mm := range old.machines {
		got := current.machines[name]
		require.NotNil(t, got, name)
		require.Equal(t, mm.Table, got.Table, name)
		require.Equal(t, mm.Classes, got.Classes, name)
		require.Equal(t, mm.Transitions, got.Transitions, name)
		require.Equal(t, mm.Holes, got.Holes, name)
		require.Equal(t, mm.Monitors, got.Monitors, name)
		require.Equal(t, mm.Assumptions, got.Assumptions, name)
		require.Equal(t, mm.Work, got.Work, name)
		for _, state := range mm.Table.States {
			want, found := mm.State(state)
			value, actual := got.State(state)
			require.Equal(t, found, actual, name+" "+state)
			require.Equal(t, want, value, name+" "+state)
			require.Equal(t, mm.Table.RowsFrom(state), got.Table.RowsFrom(state), name+" "+state)
		}
		require.Equal(t, mm.Table.TargetFingerprint(), got.Table.TargetFingerprint(), name)
	}
}

func pristineOpenModel(t *testing.T, pristine *umpirespb.Model, encoded []byte, got *Slice) {
	t.Helper()
	require.True(t, proto.Equal(pristine, got.Model))
	actual, err := proto.MarshalOptions{Deterministic: true}.Marshal(got.Model)
	require.NoError(t, err)
	require.Equal(t, encoded, actual)
}

func TestOpenWithinPreservesMachineValuesAndScope(t *testing.T) {
	scopes := []struct {
		name string
		edit func(*check.Scope)
	}{
		{"default", func(*check.Scope) {}},
		{"composition-only", func(s *check.Scope) { s.Compose.States = 3 }},
		{"members-tight", func(s *check.Scope) { s.Ceilings.Members = 1 }},
		{"evaluations-tight", func(s *check.Scope) { s.Ceilings.Evaluations = 1 }},
		{"zero", func(s *check.Scope) { s.Ceilings = interp.Ceilings{} }},
		{"looser", func(s *check.Scope) { s.Ceilings.Members++; s.Ceilings.Evaluations++ }},
	}
	base := loadModel(t, "nexus-workflow")
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(base)
	require.NoError(t, err)
	for _, specimen := range scopes {
		t.Run(specimen.name, func(t *testing.T) {
			scope := check.DefaultScope
			specimen.edit(&scope)
			old, err := originalOpenWithin(proto.Clone(base).(*umpirespb.Model), scope)
			require.NoError(t, err)
			current, err := OpenWithin(proto.Clone(base).(*umpirespb.Model), scope)
			require.NoError(t, err)
			equalOpenMachines(t, old, current)
			if specimen.name == "members-tight" || specimen.name == "evaluations-tight" || specimen.name == "zero" {
				require.Nil(t, current.bound.Machine("nexusSystem"), "scoped refusal does not replace the independent default-ceiling export table")
				require.NotNil(t, current.machines["nexusSystem"])
			}
			oldExport, oldErr := old.Quint()
			currentExport, exportErr := current.Quint()
			pristineOpenModel(t, base, encoded, old)
			pristineOpenModel(t, base, encoded, current)
			if oldErr != nil {
				require.Contains(t, []string{"members-tight", "evaluations-tight", "zero"}, specimen.name)
				require.Equal(t, reflect.TypeOf(oldErr), reflect.TypeOf(exportErr))
				require.EqualError(t, exportErr, oldErr.Error())
				require.Equal(t, oldErr, exportErr)
				require.Equal(t, oldExport == nil, currentExport == nil)
				return
			}
			require.NotContains(t, []string{"members-tight", "evaluations-tight", "zero"}, specimen.name)
			require.NoError(t, exportErr)
			require.Equal(t, oldExport.Text, currentExport.Text)
			require.Equal(t, oldExport.Machines, currentExport.Machines)
			require.Equal(t, oldExport.Compositions, currentExport.Compositions)
			require.Equal(t, oldExport.Unsupported, currentExport.Unsupported)
			equalOpenMachines(t, old, current)
			if scope.Ceilings == interp.DefaultCeilings {
				owner := oldExport.Machines[0]
				held := current.bound.Machine(owner)
				mapped := current.machines[owner]
				delete(current.machines, owner)
				require.Same(t, held, current.bound.Machine(owner), "local map ownership")
				current.machines[owner] = mapped
			}
			pristineOpenModel(t, base, encoded, old)
			pristineOpenModel(t, base, encoded, current)
		})
	}
}

func TestOpenWithinPreservesFailureOrderAndLocations(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			model := loadModel(t, "nexus-workflow")
			for _, owner := range []string{"handlerWorker", "nexusProduct"} {
				id := "open-test-" + owner
				model.Holes = append(model.Holes, &umpirespb.Hole{Id: id, Name: id})
				for _, mm := range model.Machines {
					if mm.GetName() == owner {
						mm.GetEnds().GetLambda().Body = &umpirespb.Expr{Position: mm.GetEnds().GetPosition(), Kind: &umpirespb.Expr_Hole{Hole: id}}
					}
				}
			}
			if reverse {
				slices.Reverse(model.Machines)
			}
			old, oldErr := originalOpenWithin(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			current, err := OpenWithin(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			require.Nil(t, old)
			require.Nil(t, current)
			require.Error(t, oldErr)
			require.Equal(t, reflect.TypeOf(oldErr), reflect.TypeOf(err))
			require.EqualError(t, err, oldErr.Error())
			require.Equal(t, oldErr, err)
			var unsupported *UnsupportedError
			require.ErrorAs(t, err, &unsupported)
			first := "handlerWorker"
			if reverse {
				first = "nexusProduct"
			}
			require.Contains(t, unsupported.Construct, "open-test-"+first)
			require.NotEmpty(t, unsupported.Position)
		})
	}
	for _, specimen := range []struct {
		name, model string
		edit        func(*umpirespb.Model)
	}{
		{"declared step hole", "nexus-workflow", func(m *umpirespb.Model) {
			m.Holes = append(m.Holes, &umpirespb.Hole{Id: "open-row", Name: "open-row"})
			stepFunction(m, "handlerWorker").Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Hole{Hole: "open-row"}}
		}},
		{"incomplete row match", "activity-standalone-record", func(m *umpirespb.Model) {
			match := function(m, "activitySystem.rules.respondFailed").GetBody().GetMatch()
			match.Cases = match.GetCases()[:1]
		}},
		{"admission first", "nexus-workflow", func(m *umpirespb.Model) { m.Machines[0].StateType = "missing.open.state" }},
	} {
		t.Run(specimen.name, func(t *testing.T) {
			model := loadModel(t, specimen.model)
			specimen.edit(model)
			old, oldErr := originalOpenWithin(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			current, err := OpenWithin(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			if oldErr == nil {
				require.NoError(t, err)
				equalOpenMachines(t, old, current)
				_, oldErr = old.Quint()
				_, err = current.Quint()
			}
			require.Error(t, oldErr)
			require.Equal(t, reflect.TypeOf(oldErr), reflect.TypeOf(err))
			require.EqualError(t, err, oldErr.Error())
			require.Equal(t, oldErr, err)
		})
	}
}

func TestOpenWithinPreservesCompleteAgreementAndIndependentSlices(t *testing.T) {
	for _, fixture := range []struct {
		name, model string
		choices     bool
	}{
		{"monitored named choices", "activity-standalone-record", true},
		{"Nexus composition", "nexus-workflow", false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			model := loadModel(t, fixture.model)
			if fixture.choices {
				steps := admittedSteps(model)
				require.Len(t, steps, 2)
				steps[0].Choice, steps[1].Choice = "accepts", "rejects"
			}
			encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(model)
			require.NoError(t, err)
			old, err := originalOpenWithin(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			require.NoError(t, err)
			current, err := OpenWithin(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			require.NoError(t, err)
			old.Name, current.Name = fixture.name, fixture.name
			equalOpenMachines(t, old, current)
			for owner := range old.machines {
				require.NotSame(t, old.machines[owner], current.machines[owner], "separate Slice interpretations")
			}
			oldExport, currentExport := exported(t, old), exported(t, current)
			require.Equal(t, oldExport.Text, currentExport.Text)
			require.Equal(t, oldExport.Machines, currentExport.Machines)
			require.Equal(t, oldExport.Compositions, currentExport.Compositions)
			require.Equal(t, oldExport.Unsupported, currentExport.Unsupported)
			dump := encodeDump(t, old, oldExport, nil)
			expected, oldErr := old.QuintAgreement(oldExport, dump)
			actual, err := current.QuintAgreement(currentExport, dump)
			require.NoError(t, oldErr)
			require.NoError(t, err)
			require.NotEmpty(t, expected)
			require.Equal(t, expected, actual)
			equalOpenMachines(t, old, current)
			pristineOpenModel(t, model, encoded, old)
			pristineOpenModel(t, model, encoded, current)
		})
	}
}

func TestOpenWithinDoesNotBuildDefaultMachinesTwice(t *testing.T) {
	model := loadModel(t, "nexus-workflow")
	measure := func(open func(*umpirespb.Model, check.Scope) (*Slice, error)) float64 {
		return testing.AllocsPerRun(1, func() {
			opened, err := open(proto.Clone(model).(*umpirespb.Model), check.DefaultScope)
			require.NoError(t, err)
			require.Len(t, opened.machines, len(model.Machines))
			runtime.KeepAlive(opened)
		})
	}
	old := measure(originalOpenWithin)
	current := measure(OpenWithin)
	t.Logf("complete default-ceiling Open allocations: original=%g current=%g", old, current)
	require.Less(t, current, old*0.9, "reuse the already interpreted complete default-ceiling machine set")
}

// originalOpenWithin is the independent literal pre-reuse constructor.
func originalOpenWithin(m *umpirespb.Model, scope check.Scope) (*Slice, error) {
	bound, err := check.NewRealizer(m, scope)
	if err != nil {
		return nil, err
	}
	machines, err := interp.Build(m)
	var hole *interp.Hole
	if errors.As(err, &hole) {
		return nil, &UnsupportedError{Backend: "backend", Construct: "a machine left without a table by " + describeHole(m, hole.ID),
			Position: hole.Position}
	}
	if err != nil {
		return nil, err
	}
	s := &Slice{Model: m, machines: machines, in: interp.NewInterpreter(m), bound: bound, types: map[string]*umpirespb.Type{},
		actions: map[string]*umpirespb.Action{}}
	for _, t := range m.GetTypes() {
		s.types[t.GetName()] = t
	}
	for _, a := range m.GetActions() {
		s.actions[a.GetId()] = a
	}
	return s, nil
}
