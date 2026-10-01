package goir

// Admission lists catalogs, classes, row keys and composed keys too, to check their identities: each
// is counted against the ceilings before it is listed, as Build's work is. Tight ceilings over small
// Models stand for large ones, so nothing here allocates much whether or not the bound holds.

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/proto"
)

func upTo(high int64) *modelirspb.TypeRef {
	return &modelirspb.TypeRef{Ref: &modelirspb.TypeRef_IntRange{IntRange: &modelirspb.IntRange{High: high}}}
}

// Two inputs of 0..59 each: every catalog is within a ceiling of 100, and their 3600 assignments are not.
func TestAProductIsCountedBeforeItIsListed(t *testing.T) {
	in := NewInterpreter(lifted(t, "presence"))
	in.ceilings = Ceilings{Members: 100, Evaluations: 1 << 20}
	_, err := in.product([]*modelirspb.Field{{Name: "x", Type: upTo(59)}, {Name: "y", Type: upTo(59)}})
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Resource: "members", Ceiling: 100, Needed: 3600}, *limit)
}

// presence with keep and forget given an input of 0..59: 123 classes, counted before any is listed by
// whoever lists them, admission included.
func TestClassesAreCountedTogetherBeforeTheyAreListed(t *testing.T) {
	m := proto.Clone(lifted(t, "presence")).(*modelirspb.Model)
	actions := map[string]*modelirspb.Action{}
	for _, a := range m.GetActions() {
		if a.GetName() == "keep" || a.GetName() == "forget" {
			a.Inputs = []*modelirspb.Param{{Name: "n", Type: upTo(59)}}
		}
		actions[a.GetId()] = a
	}
	in := NewInterpreter(m)
	in.ceilings = Ceilings{Members: 90, Evaluations: 1 << 20}
	_, err := in.classes(m.GetMachines()[0], actions)
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Resource: "classes", Ceiling: 90, Needed: 123}, *limit)
}

// The row-aliasing generic machine has 2 states and 2 classes: under 3 evaluations its row keys are
// not listed at admission, and Build refuses the machine with the count.
func TestAdmissionLeavesRowsPastTheCeilingToBuild(t *testing.T) {
	m := identityModel("S", "O", "", identityAction{name: "b", inputs: []string{"C"}}, identityAction{name: "c"})
	tight := Ceilings{Members: 1 << 16, Evaluations: 3}
	v := newValidator(m)
	v.in.ceilings = tight
	v.identities(m)
	require.NoError(t, errors.Join(v.errs...))
	_, err := BuildWithin(m, tight)
	var limit *LimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, LimitError{Machine: "m", Resource: "evaluations", Ceiling: 3, Needed: 4}, *limit)
}

// A composition whose Scenario reads its keys: with b_c and c each given an input of 0..9, its members
// have 10 classes each and its sync 10 × 10, 120 keys in all. Under a ceiling of 50 admission refuses
// it explicitly with that count, rather than listing them or admitting the Scenario's keys unchecked.
func TestComposedKeysAreCountedBeforeTheyAreListed(t *testing.T) {
	m := compositionModel("a", "d", "both")
	for _, a := range m.GetActions() {
		a.Inputs = []*modelirspb.Param{{Name: "n", Type: upTo(9)}}
	}
	for _, f := range m.GetFunctions() {
		f.Params = append(f.Params, &modelirspb.Param{Name: "n", Type: upTo(9)})
	}
	m.Scenarios = []*modelirspb.Scenario{{Machine: "p", Name: "each", Position: at(50), Keys: []string{"both-0-0"}}}
	v := newValidator(m)
	v.in.ceilings = Ceilings{Members: 50, Evaluations: 1 << 20}
	v.schedules(m, v.composedClasses(m))
	var limit *LimitError
	require.ErrorAs(t, errors.Join(v.errs...), &limit)
	require.Equal(t, LimitError{Machine: "p", Resource: "classes", Ceiling: 50, Needed: 120}, *limit)
}
