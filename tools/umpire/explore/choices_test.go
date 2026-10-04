package explore

// Named choices are inert (model/SEMANTICS.md, Named choices): an exploration of a Model whose step
// records name their alternatives has the candidates, digests, Case identities and Case bytes of the
// Model without the names.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// namedSteps is a copy of a Model that names each step record it constructs, by a name of its own.
func namedSteps(t *testing.T, m *umpirespb.Model) *umpirespb.Model {
	t.Helper()
	out := proto.CloneOf(m)
	n := 0
	var visit func(protoreflect.Message)
	visit = func(msg protoreflect.Message) {
		if c, ok := msg.Interface().(*umpirespb.Construct); ok && c.GetType() == umpiremodel.StepType {
			n++
			c.Choice = fmt.Sprintf("alternative-%d", n)
		}
		msg.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			switch {
			case f.Message() == nil:
			case f.IsList():
				for i := range v.List().Len() {
					visit(v.List().Get(i).Message())
				}
			default:
				visit(v.Message())
			}
			return true
		})
	}
	visit(out.ProtoReflect())
	require.Positive(t, n)
	require.NoError(t, umpiremodel.Validate(out))
	_, err := umpiremodel.Build(out)
	require.NoError(t, err)
	return out
}

// TestNamedChoicesExploreTheSameCandidates explores nexus-caller's declared deadline variations with
// every step record named. The candidates, their digests, Case identities and Case bytes, and the
// proposal re-answered from the named Model are the unnamed Model's, though the named Model's own
// bytes differ: its candidates are digested without the names.
func TestNamedChoicesExploreTheSameCandidates(t *testing.T) {
	plain, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	named := namedSteps(t, plain)
	require.False(t, proto.Equal(plain, named))
	require.True(t, proto.Equal(plain, umpiremodel.WithoutChoiceNames(named)))

	was, err := New(plain, "nexusDeadlines")
	require.NoError(t, err)
	is, err := New(named, "nexusDeadlines")
	require.NoError(t, err)
	require.NotEmpty(t, was.Candidates)
	require.Len(t, is.Candidates, len(was.Candidates))
	lowered := 0
	for i, c := range was.Candidates {
		d := is.Candidates[i]
		require.Equal(t, []any{c.Key, c.Priority, c.Digest, c.Identity, c.Rejection, string(c.Bytes)},
			[]any{d.Key, d.Priority, d.Digest, d.Identity, d.Rejection, string(d.Bytes)}, c.Key)
		if c.Rejection == "" {
			lowered++
		}
	}
	require.Positive(t, lowered)

	first := was.Candidates[0]
	require.Empty(t, first.Rejection)
	proposal, err := is.Proposal(is.Candidates[0])
	require.NoError(t, err)
	require.Equal(t, first.Digest, proposal.Digest)
	retained, err := ReadProposal([]byte(proposal.Source))
	require.NoError(t, err)
	require.Equal(t, []any{first.Digest, first.Identity, string(first.Bytes)}, []any{retained.Digest, retained.Identity, string(retained.Bytes)})
}
