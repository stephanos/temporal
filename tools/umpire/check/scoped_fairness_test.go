package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

func scopedTick(member string) *umpirespb.ComposedActionRef {
	return &umpirespb.ComposedActionRef{Composition: "two", Member: member,
		Selection: &umpirespb.ComposedActionRef_Action{Action: "generic.tick"}}
}

func scopedFairness(m *umpirespb.Model, refs ...*umpirespb.ComposedActionRef) {
	m.Assumptions = append(m.Assumptions, &umpirespb.Assumption{Id: "generic.composedRuns", Name: "composedRuns",
		Position: at(88), FairRefs: refs})
	m.Progress[1].Assumptions = []string{"generic.composedRuns"}
}

func parameterizedProgressModel() *umpirespb.Model {
	m := composedProgressModel()
	m.Actions[0].Inputs = []*umpirespb.Param{{Name: "input", Type: upTo(1)}}
	step := functionNamed(m, "generic.tickStep")
	step.Params = append(step.Params, &umpirespb.Param{Name: "input", Type: upTo(1)})
	return m
}

func overloadedProgressModel() *umpirespb.Model {
	m := parameterizedProgressModel()
	m.Machines[0].Assumes = nil
	empty := proto.Clone(m.Actions[0]).(*umpirespb.Action)
	empty.Id, empty.Inputs = "generic.emptyTick", nil
	m.Actions = append(m.Actions, empty)
	step := proto.Clone(functionNamed(m, "generic.tickStep")).(*umpirespb.Function)
	step.Name, step.Params = "generic.emptyTickStep", step.Params[:1]
	m.Functions = append(m.Functions, step)
	m.Machines[0].Steps = append(m.Machines[0].Steps, &umpirespb.StepBinding{Action: empty.Id, Function: step.Name})
	return m
}

func TestComposedFairnessRefusesMachineConsumersAndMixedForms(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*umpirespb.Model)
		at     string
		why    string
	}{
		{"machine assumes", func(m *umpirespb.Model) { m.Machines[0].Assumes = []string{"generic.composedRuns"} }, "generic:", "not on counter"},
		{"mixed forms", func(m *umpirespb.Model) { m.Assumptions[len(m.Assumptions)-1].Fair = []string{"generic.tick"} }, "generic:88", "mixes legacy and composed fairness"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := composedProgressModel()
			scopedFairness(m, scopedTick("left"))
			tc.change(m)
			err := ir.Validate(m)
			require.ErrorContains(t, err, tc.why)
			require.ErrorContains(t, err, tc.at)
		})
	}
}

func TestComposedClaimFairnessResolvesMemberAndSyncClasses(t *testing.T) {
	for _, tc := range []struct {
		name string
		sync bool
		ref  *umpirespb.ComposedActionRef
		want []string
	}{
		{"left action", false, scopedTick("left"), []string{"left_tick-0", "left_tick-1"}},
		{"right repeated member", false, scopedTick("right"), []string{"right_tick-0", "right_tick-1"}},
		{"exact member class", false, &umpirespb.ComposedActionRef{Composition: "two", Member: "right",
			Selection: &umpirespb.ComposedActionRef_ActionClass{ActionClass: &umpirespb.ActionClass{Action: "generic.tick", Inputs: []*umpirespb.Value{admIntValue(1)}}}}, []string{"right_tick-1"}},
		{"member action through sync", true, scopedTick("left"), []string{"tickBoth-0-0", "tickBoth-0-1", "tickBoth-1-0", "tickBoth-1-1"}},
		{"exact member class through sync", true, &umpirespb.ComposedActionRef{Composition: "two", Member: "left",
			Selection: &umpirespb.ComposedActionRef_ActionClass{ActionClass: &umpirespb.ActionClass{Action: "generic.tick", Inputs: []*umpirespb.Value{admIntValue(1)}}}}, []string{"tickBoth-1-0", "tickBoth-1-1"}},
		{"sync action", true, &umpirespb.ComposedActionRef{Composition: "two",
			Selection: &umpirespb.ComposedActionRef_Sync{Sync: "tickBoth"}}, []string{"tickBoth-0-0", "tickBoth-0-1", "tickBoth-1-0", "tickBoth-1-1"}},
		{"exact sync class", true, &umpirespb.ComposedActionRef{Composition: "two",
			Selection: &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "tickBoth", Inputs: []*umpirespb.Value{admIntValue(1), admIntValue(0)}}}}, []string{"tickBoth-1-0"}},
		{"separator bearing sync name", true, &umpirespb.ComposedActionRef{Composition: "two",
			Selection: &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "tickBoth-extra", Inputs: []*umpirespb.Value{admIntValue(1), admIntValue(0)}}}}, []string{"tickBoth-extra-1-0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := parameterizedProgressModel()
			m.Machines[0].Assumes = nil
			if !tc.sync {
				m.Compositions[0].Syncs = nil
			} else if sync := tc.ref.GetSyncClass(); sync != nil {
				m.Compositions[0].Syncs[0].Name = sync.GetAction()
			}
			scopedFairness(m, tc.ref)
			require.NoError(t, ir.Validate(m))
			claim, subject, err := bind(m, DefaultScope).progress(m.Progress[1])
			require.NoError(t, err)
			require.Equal(t, []Assumption{{Name: "composedRuns", Fair: tc.want}}, claim.Assumptions)
			require.Empty(t, subject.table.Assumptions)
			fresh, _, err := bind(proto.Clone(m).(*umpirespb.Model), DefaultScope).progress(m.Progress[1])
			require.NoError(t, err)
			require.Equal(t, claim.Assumptions, fresh.Assumptions)
		})
	}
}

func TestComposedClaimFairnessSharesInheritanceUnionAndEnabledness(t *testing.T) {
	for _, blocked := range []bool{true, false} {
		t.Run(map[bool]string{true: "blocked partner", false: "enabled partner"}[blocked], func(t *testing.T) {
			m := composedProgressModel()
			boundarySpin(m)
			m.Machines[0].Assumes, m.Machines[1].Assumes = nil, nil
			if blocked {
				functionNamed(m, "generic.rightTickStep").Body.GetIf().Condition = expr(boolValue(false))
			}
			scopedFairness(m, scopedTick("left"), &umpirespb.ComposedActionRef{Composition: "two", Selection: &umpirespb.ComposedActionRef_Sync{Sync: "tickBoth"}})
			require.NoError(t, ir.Validate(m))
			claim, subject, err := bind(m, DefaultScope).progress(m.Progress[1])
			require.NoError(t, err)
			require.Equal(t, []Assumption{{Name: "composedRuns", Fair: []string{"tickBoth"}}}, claim.Assumptions)
			x := receiptOf(t, Check(m, DefaultScope), "progress two bothReachTop fair-cycle")
			want := Verified
			if blocked {
				want = Counterexample
			}
			require.Equal(t, want, x.Kind, x.Explanation)
			require.Equal(t, []string{"composedRuns"}, x.Assumptions)
			if blocked {
				require.Equal(t, []string{"right_spin"}, taken(x.Witness))
				require.NoError(t, claim.Replay(subject.table, umpire.CycleKind,
					ProgressVerdict{Outcome: umpire.CounterexampleFound, Witness: x.Witness, Loop: x.Loop}))
			}
			require.Equal(t, Counterexample, receiptOf(t, Check(m, DefaultScope), "progress two bothReachTop deadline").Kind)
		})
	}
}

func TestComposedClaimFairnessUsesTheDeclaredActionCatalog(t *testing.T) {
	m := parameterizedProgressModel()
	m.Machines[0].Assumes = nil
	m.Compositions[0].Syncs = nil
	m.Actions[0].Name = "tick-extra"
	scopedFairness(m, scopedTick("right"))
	require.NoError(t, ir.Validate(m))
	claim, _, err := bind(m, DefaultScope).progress(m.Progress[1])
	require.NoError(t, err)
	require.Equal(t, []Assumption{{Name: "composedRuns", Fair: []string{"right_tick-extra-0", "right_tick-extra-1"}}}, claim.Assumptions)
}

func TestComposedClaimFairnessDoesNotBroadenAnExactZeroInputClass(t *testing.T) {
	m := overloadedProgressModel()
	m.Compositions[0].Syncs = nil
	for _, tc := range []struct {
		name string
		ref  *umpirespb.ComposedActionRef
		want []string
	}{
		{"action-wide", scopedTick("left"), []string{"left_tick-0", "left_tick-1"}},
		{"exact zero-input", &umpirespb.ComposedActionRef{Composition: "two", Member: "left",
			Selection: &umpirespb.ComposedActionRef_ActionClass{ActionClass: &umpirespb.ActionClass{Action: "generic.emptyTick"}}}, []string{"left_tick"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := proto.Clone(m).(*umpirespb.Model)
			scopedFairness(fixture, tc.ref)
			require.NoError(t, ir.Validate(fixture))
			claim, _, err := bind(fixture, DefaultScope).progress(fixture.Progress[1])
			require.NoError(t, err)
			require.Equal(t, []Assumption{{Name: "composedRuns", Fair: tc.want}}, claim.Assumptions)
		})
	}
}

func TestComposedClaimFairnessResolvesExactSyncInputAlternatives(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  *umpirespb.ComposedActionRef
		want []string
	}{
		{"zero-input class", &umpirespb.ComposedActionRef{Composition: "two",
			Selection: &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "tickBoth"}}}, []string{"tickBoth"}},
		{"parameterized class", &umpirespb.ComposedActionRef{Composition: "two",
			Selection: &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "tickBoth", Inputs: []*umpirespb.Value{admIntValue(1)}}}}, []string{"tickBoth-1"}},
		{"all sync classes", &umpirespb.ComposedActionRef{Composition: "two", Selection: &umpirespb.ComposedActionRef_Sync{Sync: "tickBoth"}},
			[]string{"tickBoth", "tickBoth-0", "tickBoth-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := overloadedProgressModel()
			right := proto.Clone(m.Machines[0]).(*umpirespb.Machine)
			right.Name, right.Steps = "rightCounter", right.Steps[len(right.Steps)-1:]
			m.Machines = append(m.Machines, right)
			m.Compositions[0].Members[1].Machine = right.Name
			scopedFairness(m, tc.ref)
			require.NoError(t, ir.Validate(m))
			claim, _, err := bind(m, DefaultScope).progress(m.Progress[1])
			require.NoError(t, err)
			require.Equal(t, []Assumption{{Name: "composedRuns", Fair: tc.want}}, claim.Assumptions)
		})
	}
}

func TestComposedClaimFairnessUnionsSameNameAfterMemberReplacement(t *testing.T) {
	m := composedProgressModel()
	boundarySpin(m)
	m.Compositions[0].Syncs = nil
	provider := m.Machines[1]
	m.Functions = append(m.Functions, &umpirespb.Function{Name: "generic.asCounter", Position: at(89),
		Params: []*umpirespb.Param{{Name: "s", Type: interp.Named("Counter")}}, Body: expr("s")})
	provider.Refines = &umpirespb.Refinement{Product: "counter", Map: "generic.asCounter"}
	m.Compositions[0].Members[1].Replaces = "counter"
	m.Assumptions = append(m.Assumptions, &umpirespb.Assumption{Id: "generic.providerRuns", Name: "tickRuns", Position: at(90), Fair: []string{"generic.spin"}})
	provider.Assumes = []string{"generic.providerRuns"}
	scopedFairness(m, scopedTick("right"))
	m.Assumptions[len(m.Assumptions)-1].Name = "tickRuns"
	require.NoError(t, ir.Validate(m))
	claim, subject, err := bind(m, DefaultScope).progress(m.Progress[1])
	require.NoError(t, err)
	require.Equal(t, []Assumption{{Name: "tickRuns", Fair: []string{"left_tick", "right_spin"}}}, subject.table.Assumptions)
	require.Equal(t, []Assumption{{Name: "tickRuns", Fair: []string{"right_tick"}}}, claim.Assumptions)
	r := Check(m, DefaultScope)
	x := receiptOf(t, r, "progress two bothReachTop fair-cycle")
	require.Equal(t, Verified, x.Kind, x.Explanation)
	require.Equal(t, []string{"tickRuns"}, x.Assumptions)
	require.Equal(t, Counterexample, receiptOf(t, r, "progress two bothReachTop deadline").Kind)
	without := proto.Clone(m).(*umpirespb.Model)
	without.Progress[1].Assumptions = nil
	require.Equal(t, Counterexample, receiptOf(t, Check(without, DefaultScope), "progress two bothReachTop fair-cycle").Kind)
}

func TestComposedFairnessRejectsWrongClaimOwnerWithAttribution(t *testing.T) {
	for _, owner := range []string{"counter", "other"} {
		t.Run(owner, func(t *testing.T) {
			m := composedProgressModel()
			scopedFairness(m, scopedTick("left"))
			claim := m.Progress[0]
			where := "generic:80"
			if owner == "other" {
				other := proto.Clone(m.Compositions[0]).(*umpirespb.Composition)
				other.Name = owner
				m.Compositions = append(m.Compositions, other)
				claim = proto.Clone(m.Progress[1]).(*umpirespb.Progress)
				claim.Machine = owner
				m.Progress = append(m.Progress, claim)
				where = "generic:83"
			}
			claim.Assumptions = m.Progress[1].Assumptions
			err := ir.Validate(m)
			require.ErrorContains(t, err, where)
			require.ErrorContains(t, err, owner)
			require.ErrorContains(t, err, "two")
		})
	}
}

func TestComposedFairnessRejectsInvalidParameterizedInputs(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  *umpirespb.ComposedActionRef
		why  string
	}{
		{"member input count", &umpirespb.ComposedActionRef{Composition: "two", Member: "left", Selection: &umpirespb.ComposedActionRef_ActionClass{ActionClass: &umpirespb.ActionClass{Action: "generic.tick"}}}, "takes 1 inputs, not 0"},
		{"member input outside domain", &umpirespb.ComposedActionRef{Composition: "two", Member: "left", Selection: &umpirespb.ComposedActionRef_ActionClass{ActionClass: &umpirespb.ActionClass{Action: "generic.tick", Inputs: []*umpirespb.Value{admIntValue(2)}}}}, "not 2"},
		{"sync input count", &umpirespb.ComposedActionRef{Composition: "two", Selection: &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "tickBoth", Inputs: []*umpirespb.Value{admIntValue(0)}}}}, "takes 2 inputs, not 1"},
		{"sync crossed input type", &umpirespb.ComposedActionRef{Composition: "two", Selection: &umpirespb.ComposedActionRef_SyncClass{SyncClass: &umpirespb.ActionClass{Action: "tickBoth", Inputs: []*umpirespb.Value{admIntValue(0), boolValue(true)}}}}, "not true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := parameterizedProgressModel()
			scopedFairness(m, tc.ref)
			err := ir.Validate(m)
			require.ErrorContains(t, err, "generic:88")
			require.ErrorContains(t, err, tc.why)
		})
	}
}
