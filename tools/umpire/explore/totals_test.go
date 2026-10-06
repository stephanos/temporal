package explore

import (
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// withSourceTotals is the Model with every Query asserting its static combination count, as an author
// who declares each total writes it.
func withSourceTotals(t *testing.T, m *umpirespb.Model) *umpirespb.Model {
	t.Helper()
	declared := proto.CloneOf(m)
	for _, q := range declared.GetQueries() {
		q.Total = wrapperspb.Int64(0)
	}
	out, err := ir.WithTotals(declared)
	require.NoError(t, err)
	require.NoError(t, ir.Validate(out))
	return out
}

func queryNamed(t *testing.T, m *umpirespb.Model, name string) *umpirespb.Query {
	t.Helper()
	for _, q := range m.GetQueries() {
		if q.GetName() == name {
			return q
		}
	}
	require.FailNow(t, "no Query", name)
	return nil
}

// requireRecounted holds a candidate valid with its own Query asserting the count of its own schedule.
func requireRecounted(t *testing.T, plan *Plan, c *Candidate) int64 {
	t.Helper()
	require.Empty(t, c.Rejection, c.Key)
	require.NotEmpty(t, c.Bytes, c.Key)
	q := queryNamed(t, c.Model, plan.Query)
	require.NotNil(t, q.GetTotal(), c.Key)
	recounted, err := ir.WithTotals(c.Model)
	require.NoError(t, err, c.Key)
	n := queryNamed(t, recounted, plan.Query).GetTotal().GetValue()
	require.Equal(t, n, q.GetTotal().GetValue(), c.Key)
	require.NoError(t, ir.Validate(c.Model), c.Key)
	return n
}

// A variation of another schedule length and a prefix drop each change the source Query's count; the
// candidate asserts its own recount and lowers, while the source Query keeps the author's assertion.
func TestScheduleChangingCandidatesAssertTheirOwnRecountedTotal(t *testing.T) {
	loaded, err := ir.Load("../../../model/ir/nexus-control.json")
	require.NoError(t, err)
	m := withSourceTotals(t, loaded)
	source := proto.CloneOf(m)
	authored := queryNamed(t, m, "forgedCompletion").GetTotal().GetValue()

	plan, err := New(m, "nexusControl")
	require.NoError(t, err)
	require.Len(t, plan.Candidates, 3)
	recounted := map[int64]bool{}
	for _, c := range plan.Candidates {
		recounted[requireRecounted(t, plan, c)] = true
	}
	require.Len(t, recounted, 3, "schedules of three lengths assert three counts")
	require.NotEqual(t, authored, queryNamed(t, plan.Candidates[0].Model, plan.Query).GetTotal().GetValue(), "the longer schedule is recounted")

	reduced := plan.Candidates[0]
	for _, index := range []int{4, 2} {
		previous := requireRecounted(t, plan, reduced)
		reduced, err = plan.Reduce(reduced, index)
		require.NoError(t, err)
		require.Less(t, requireRecounted(t, plan, reduced), previous, "a dropped action is recounted")
	}
	_, err = plan.Proposal(reduced)
	require.NoError(t, err)

	require.True(t, proto.Equal(source, plan.base), "the source Query's assertion is never rewritten")
	require.True(t, proto.Equal(source, m), "the caller's Model is never rewritten")
	require.Equal(t, authored, queryNamed(t, plan.base, plan.Query).GetTotal().GetValue())
}

// A total is metadata: the same Model with and without its source totals derives the same candidates
// in the same order, under the same digests, Case identities and Case bytes, and proposes the same
// promotion source, so correcting a source total renames nothing.
func TestASourceTotalCorrectionChangesNoExplorationIdentity(t *testing.T) {
	for _, explored := range []struct{ file, name string }{
		{"../../../model/ir/nexus-control.json", "nexusControl"},
		{"../../../model/ir/nexus-caller.json", "nexusDeadlines"},
	} {
		t.Run(explored.name, func(t *testing.T) {
			loaded, err := ir.Load(explored.file)
			require.NoError(t, err)
			bare, err := New(ir.WithoutTotals(loaded), explored.name)
			require.NoError(t, err)
			totaled, err := New(withSourceTotals(t, loaded), explored.name)
			require.NoError(t, err)

			require.Len(t, totaled.Candidates, len(bare.Candidates))
			for i, want := range bare.Candidates {
				got := totaled.Candidates[i]
				require.Equal(t, want.Key, got.Key)
				require.Equal(t, want.Priority, got.Priority)
				require.Equal(t, want.Digest, got.Digest, want.Key)
				require.Equal(t, want.Identity, got.Identity, want.Key)
				require.Equal(t, want.Rejection, got.Rejection, want.Key)
				require.Equal(t, want.Bytes, got.Bytes, want.Key)
				if want.Rejection != "" {
					continue
				}
				requireRecounted(t, totaled, got)

				wantReduced, wantErr := bare.Reduce(want, 0)
				gotReduced, gotErr := totaled.Reduce(got, 0)
				if wantErr != nil {
					require.EqualError(t, gotErr, wantErr.Error(), want.Key)
					wantReduced, gotReduced = want, got
				} else {
					require.NoError(t, gotErr, want.Key)
					require.Equal(t, wantReduced.Digest, gotReduced.Digest, want.Key)
					require.Equal(t, wantReduced.Identity, gotReduced.Identity, want.Key)
					require.Equal(t, wantReduced.Bytes, gotReduced.Bytes, want.Key)
					requireRecounted(t, totaled, gotReduced)
				}
				wantProposal, err := bare.Proposal(wantReduced)
				require.NoError(t, err, want.Key)
				gotProposal, err := totaled.Proposal(gotReduced)
				require.NoError(t, err, want.Key)
				require.Equal(t, wantProposal, gotProposal, want.Key)
				recovered, err := ReadProposal([]byte(gotProposal.Source))
				require.NoError(t, err, want.Key)
				require.Equal(t, gotReduced.Bytes, recovered.Bytes, want.Key)
			}
		})
	}
}
