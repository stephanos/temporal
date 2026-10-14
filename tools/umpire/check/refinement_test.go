package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
)

func TestRealizerGivesTheRefinementCheckReads(t *testing.T) {
	r, err := NewRealizer(activityModel(t), DefaultScope)
	require.NoError(t, err)
	rows, err := r.Refinement("activitySystem")
	require.NoError(t, err)
	carried := map[string]string{}
	for _, row := range rows {
		if row.Product != nil {
			carried[row.Key] = *row.Product
		}
	}
	require.Equal(t, "poll", carried["scheduled-now-0-unset-unset-unset-unset-unlimited-poll"])
	_, err = r.Refinement("activityProduct")
	require.ErrorContains(t, err, "refines no machine")
}

// A refinement that does not hold is the machine's, not the Model's: its table is still built, so a
// check can find the counterexample on it.
func TestARejectedRefinementStaysWithItsMachine(t *testing.T) {
	m := lifted(t, "admission")
	machines := built(t, m)
	_, err := refinementOf(t, m, "activityRecord")
	require.NoError(t, err)
	_, err = refinementOf(t, m, "trustingActivityRecord")
	require.ErrorContains(t, err, "trustingActivityRecord refines activityProduct: the row "+
		"'paused-queued-none-poll' steps from 'paused-queued-none' to 'started-empty-one', which read as 'paused' and 'started'")
	require.NotEmpty(t, machines["trustingActivityRecord"].Table.Rows)
}

func TestVisibleProjectionOfARefinement(t *testing.T) {
	// The reachable crash hole leaves the refinement unknown, not rejected: every row refines.
	rows, err := refinementOf(t, lifted(t, "declarations"), "disk")
	var incomplete *RefinementError
	require.ErrorAs(t, err, &incomplete)
	require.Equal(t, RefinementIncomplete, incomplete.Kind)
	put := "put"
	require.Equal(t, []RefinementRow{{Key: "empty-put", Product: &put}, {Key: "staged-flush"}}, rows)

	for name, visible := range map[string]string{
		"a stutter records a fact the product sees":   "disk.visible",
		"a stutter's outcome is one the product sees": "disk.visibleOutcomes",
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "declarations")).(*umpirespb.Model)
			f := function(m, visible)
			f.Body = &umpirespb.Expr{Position: f.GetBody().GetPosition(),
				Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Bool{Bool: true}}}}
			disk := built(t, m)["disk"]
			require.NotEmpty(t, disk.ReachableHoles(), "the reachable crash hole does not erase the rejection")
			_, err := refinementOf(t, m, "disk")
			require.ErrorContains(t, err, "disk refines store: the row 'staged-flush'")
		})
	}
}

// ends, visible and visibleOutcomes each made to return 3: a located error, never a state that is no
// end or a fact the product does not see. Build reads ends; the refinement Check reads the two
// others.
func TestEndsAndVisibleMustReturnABoolean(t *testing.T) {
	three := func(at *umpirespb.Expr) *umpirespb.Expr { return admLiteral(at, admIntValue(3)) }
	for name, c := range map[string]struct {
		mutate func(m *umpirespb.Model)
		want   string
	}{
		"ends": {func(m *umpirespb.Model) {
			ends := admMachine(m, "disk").GetEnds().GetLambda()
			ends.Body = three(ends.GetBody())
		}, "disk: ends is 3 at empty, not a Boolean"},
		"visible": {func(m *umpirespb.Model) {
			f := function(m, "disk.visible")
			f.Body = three(f.GetBody())
		}, "disk: disk.visible is 3 for stored, not a Boolean"},
		"visibleOutcomes": {func(m *umpirespb.Model) {
			f := function(m, "disk.visibleOutcomes")
			f.Body = three(f.GetBody())
		}, "disk: disk.visibleOutcomes is 3 for accepted, not a Boolean"},
	} {
		t.Run(name, func(t *testing.T) {
			m := proto.Clone(lifted(t, "declarations")).(*umpirespb.Model)
			c.mutate(m)
			_, err := interp.Build(m)
			if name != "ends" {
				require.NoError(t, err)
				_, err = refinementOf(t, m, "disk")
			}
			var located *interp.Error
			require.ErrorAs(t, err, &located)
			require.NotEmpty(t, located.Position)
			require.Equal(t, c.want, located.Message)
		})
	}
}
