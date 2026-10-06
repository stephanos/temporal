package lower_test

// A Property's origin is inert: it traces a generated Property to the capability Property it was
// expanded from, and nothing that names, answers, lowers or explores a Model reads it.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/conformance"
	"go.temporal.io/server/tools/umpire/explore"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/proto"
)

func loadModel(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", name))
	require.NoError(t, err)
	return m
}

// withOrigins is the Model with every Property traced to a capability Property of its own name,
// defined where the Property is declared, and admitted so.
func withOrigins(t *testing.T, m *umpirespb.Model) *umpirespb.Model {
	t.Helper()
	out := proto.CloneOf(m)
	require.NotEmpty(t, out.GetProperties())
	for _, p := range out.GetProperties() {
		p.Origin = &umpirespb.PropertyOrigin{Name: "temporal.capabilities.Traced." + p.GetName(), Position: proto.CloneOf(p.GetPosition())}
	}
	require.NoError(t, ir.Validate(out))
	return out
}

// The activity Model, whose capabilities generate Properties, gives every check the same receipt,
// Definition ID and Behavior Fingerprint, lowers every Query to the same Case, and binds an assessment
// to the same Model identity, with and without origins.
func TestAnOriginMovesNoAnswerFingerprintLoweringOrModelIdentity(t *testing.T) {
	plain := loadModel(t, "activity-standalone.json")
	traced := withOrigins(t, plain)

	require.Equal(t, check.Check(plain, check.DefaultScope).Receipts, check.Check(traced, check.DefaultScope).Receipts)

	plainProducer, err := lower.NewProducer(plain)
	require.NoError(t, err)
	tracedProducer, err := lower.NewProducer(traced)
	require.NoError(t, err)
	lowered := 0
	for _, q := range plain.GetQueries() {
		identity := lower.IdentityFor("temporal.case", "standaloneActivityTests", q.GetName())
		want, wantErr := plainProducer.Lower(q.GetName(), identity)
		got, gotErr := tracedProducer.Lower(q.GetName(), identity)
		if wantErr != nil {
			require.EqualError(t, gotErr, wantErr.Error(), q.GetName())
			continue
		}
		require.NoError(t, gotErr, q.GetName())
		require.Equal(t, want.Standing, got.Standing, q.GetName())
		require.Equal(t, want.Unsupported, got.Unsupported, q.GetName())
		require.Equal(t, want.OffPath, got.OffPath, q.GetName())
		require.Equal(t, want.Inventory, got.Inventory, q.GetName())
		protorequire.ProtoEqual(t, want.Case, got.Case)
		if want.Standing != lower.Lowered {
			continue
		}
		lowered++
		key := check.ClaimKey{Family: "temporal.features.activity.standalone.system", Owner: "activitySystem", Name: q.GetName()}
		wantFactory, wantErr := conformance.Prepare(plain, key, want.Case, conformance.DefaultLimits())
		gotFactory, gotErr := conformance.Prepare(traced, key, got.Case, conformance.DefaultLimits())
		if wantErr != nil {
			require.EqualError(t, gotErr, wantErr.Error(), q.GetName())
			continue
		}
		require.NoError(t, gotErr, q.GetName())
		require.Equal(t, wantFactory.Binding(), gotFactory.Binding(), q.GetName())
		require.NotEmpty(t, gotFactory.Binding().Model, q.GetName())
	}
	require.Positive(t, lowered, "some Query lowers to a Case")
}

// An exploration derives the same candidates under the same digests, Case identities and Case bytes,
// and proposes the same promotion source, with and without origins.
func TestAnOriginMovesNoExplorationIdentity(t *testing.T) {
	plain := loadModel(t, "nexus-workflow-control.json")
	bare, err := explore.New(plain, "nexusControl")
	require.NoError(t, err)
	traced, err := explore.New(withOrigins(t, plain), "nexusControl")
	require.NoError(t, err)

	require.Len(t, traced.Candidates, len(bare.Candidates))
	require.NotEmpty(t, bare.Candidates)
	for i, want := range bare.Candidates {
		got := traced.Candidates[i]
		require.Equal(t, want.Key, got.Key)
		require.Equal(t, want.Priority, got.Priority, want.Key)
		require.Equal(t, want.Digest, got.Digest, want.Key)
		require.Equal(t, want.Identity, got.Identity, want.Key)
		require.Equal(t, want.Rejection, got.Rejection, want.Key)
		require.Equal(t, want.Bytes, got.Bytes, want.Key)
		if want.Rejection != "" {
			continue
		}
		wantProposal, err := bare.Proposal(want)
		require.NoError(t, err, want.Key)
		gotProposal, err := traced.Proposal(got)
		require.NoError(t, err, want.Key)
		require.Equal(t, wantProposal, gotProposal, want.Key)
	}
}
