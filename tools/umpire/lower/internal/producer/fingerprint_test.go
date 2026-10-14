package producer_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
)

// fingerprints is what a Case says of its table's Behavior Fingerprint: the fingerprint it binds the
// target to, and the Query and projection fingerprints, which are of canonical forms that name it.
func fingerprints(c *testpilotspb.Case) (target, query, projection string) {
	for _, d := range c.GetProvenance().GetDefinitions() {
		switch d.GetKind() {
		case testpilotspb.DEFINITION_KIND_TARGET:
			target = d.GetBehaviorFingerprint()
		case testpilotspb.DEFINITION_KIND_QUERY:
			query = d.GetBehaviorFingerprint()
		default:
		}
	}
	return target, query, c.GetContract().GetCorrelated().GetProjectionFingerprint()
}

// A caller that holds the fingerprint of the Query's table hands it to the production, which writes
// the Case it writes when it computes the fingerprint itself. A fingerprint handed for another table,
// such as a changed copy of this one, is not read. The one handed for this table is what the Case
// names in each place it names one.
func TestAHeldFingerprintIsReadOnlyForItsTable(t *testing.T) {
	q := jobQuery(t, "once", jobOnce...)
	table, err := q.Scenario.Machine.Table()
	require.NoError(t, err)
	identity, source := jobIdentity("once"), cp.Source{Path: "job.go", Provenance: "test"}
	produce := func(target cp.Fingerprinted) *testpilotspb.Case {
		r := jobRealization(t, jobSources())
		r.Target = target
		c, err := cp.Produce(q, identity, r, source)
		require.NoError(t, err)
		return c
	}
	computed := produce(cp.Fingerprinted{})
	target, query, projection := fingerprints(computed)
	require.Equal(t, table.TargetFingerprint(), target)

	protorequire.ProtoEqual(t, computed, produce(cp.Fingerprinted{Table: table, Fingerprint: table.TargetFingerprint()}))

	changed := *table
	changed.Facts = append(slices.Clone(table.Facts), "unbound")
	require.NotEqual(t, target, changed.TargetFingerprint())
	protorequire.ProtoEqual(t, computed, produce(cp.Fingerprinted{Table: &changed, Fingerprint: changed.TargetFingerprint()}))

	const held = "sha256:held"
	heldTarget, heldQuery, heldProjection := fingerprints(produce(cp.Fingerprinted{Table: table, Fingerprint: held}))
	require.Equal(t, held, heldTarget)
	require.NotEqual(t, query, heldQuery)
	require.NotEqual(t, projection, heldProjection)
}
