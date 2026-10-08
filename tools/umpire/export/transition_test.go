package export

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

func transitionSlice(t *testing.T, selector string) *Slice {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "declarations.json"))
	require.NoError(t, err)
	disk := m.GetMachines()[slices.IndexFunc(m.GetMachines(), func(mm *umpirespb.Machine) bool { return mm.GetName() == "disk" })]
	disk.Refines, disk.Monitors = nil, nil
	m.Machines, m.Compositions, m.Progress = []*umpirespb.Machine{disk}, nil, nil
	p := m.GetProperties()[slices.IndexFunc(m.GetProperties(), func(p *umpirespb.Property) bool { return p.GetName() == "durableStays" })]
	p.Name = "flushSettles"
	if selector == "class" {
		p.When = &umpirespb.Property_WhenClass{WhenClass: &umpirespb.ActionClass{Action: "fixture.declarations.flush"}}
	} else {
		action := "flush"
		if selector == "disabled" {
			action = "crash"
		}
		p.When = &umpirespb.Property_WhenAction{WhenAction: action}
	}
	m.Properties = []*umpirespb.Property{p}
	sc := m.GetScenarios()[slices.IndexFunc(m.GetScenarios(), func(sc *umpirespb.Scenario) bool {
		return sc.GetMachine() == "disk" && sc.GetFree()
	})]
	m.Scenarios = []*umpirespb.Scenario{sc}
	q := m.GetQueries()[slices.IndexFunc(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetName() == "durableStays" })]
	q.Property.Name = p.GetName()
	m.Queries = []*umpirespb.Query{q}
	for _, f := range m.GetFunctions() {
		switch f.GetName() {
		case p.GetHolds():
			requires := proto.Clone(f.GetBody().GetBinary().GetLeft()).(*umpirespb.Expr)
			requires.GetBinary().Op = umpirespb.Binary_OP_EQ
			requires.GetBinary().GetRight().GetLiteral().GetEnum().Case = "staged"
			f.Requires = requires
			f.Body = proto.Clone(f.GetBody().GetBinary().GetRight()).(*umpirespb.Expr)
		case "fixture.declarations.Declarations$package$.crashStep":
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(), Kind: &umpirespb.Expr_List{List: &umpirespb.ListOf{}}}
		}
	}
	m, err = ir.WithTotals(m)
	require.NoError(t, err)
	s := openSlice(t, m)
	s.Name = "selected transition " + selector
	return s
}

func TestTransitionSelectorsAgreeWithTheReader(t *testing.T) {
	for _, selector := range []string{"class", "action", "disabled"} {
		t.Run(selector, func(t *testing.T) {
			s := transitionSlice(t, selector)
			report := check.Check(s.Model, check.DefaultScope)
			require.Len(t, report.Receipts, 1)
			r := report.Receipts[0]
			require.Equal(t, check.Verified, r.Kind, r.Explanation)
			require.Equal(t, selector != "disabled", r.Exercised)
			bound, err := s.bound.Bound(r.Key)
			require.NoError(t, err)
			view, err := s.view(s.machines["disk"])
			require.NoError(t, err)
			require.Len(t, view.Reach, 3)
			selected, skipped := 0, 0
			for state, by := range view.Claims {
				for action, steps := range by {
					about := selector != "disabled" && action == "flush"
					require.Equal(t, about, bound.Property.About(action))
					for i, reads := range steps {
						require.Equal(t, claimRead{About: about, Holds: true}, reads[0])
						if about {
							require.Equal(t, "staged", state)
							row := bound.Table.RowsFrom(state)[0]
							held, err := bound.Property.Holds(state, row.Results[i])
							require.NoError(t, err)
							require.True(t, held)
							selected++
						} else {
							skipped++
						}
					}
				}
			}
			if selector == "disabled" {
				require.Zero(t, selected)
				require.Equal(t, 2, skipped)
			} else {
				require.Equal(t, 1, selected)
				require.Equal(t, 1, skipped)
			}
			x := exported(t, s)
			receipts, err := s.QuintAgreement(x, encodeDump(t, s, x, nil))
			require.NoError(t, err)
			a := only(t, receipts, PropertyAgreement, "disk")
			require.Equal(t, Agreed, a.Kind, "%v", a.Differences)
			require.Equal(t, selected, a.About)
		})
	}
}

func TestQuintReadsSelectedTransitionsBeforeState(t *testing.T) {
	needs(t, QuintTool)
	for _, selector := range []string{"class", "action", "disabled"} {
		t.Run(selector, func(t *testing.T) {
			s := transitionSlice(t, selector)
			x := exported(t, s)
			receipts, err := s.QuintAgreement(x, quintDump(t, x))
			require.NoError(t, err)
			r := only(t, receipts, PropertyAgreement, "disk")
			require.Equal(t, Agreed, r.Kind, "%v", r.Differences)
			want := 1
			if selector == "disabled" {
				want = 0
			}
			require.Equal(t, want, r.About)
			report(t, r)
		})
	}
}
