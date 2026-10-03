package checker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

// A table's Behavior Fingerprint is read off what the table says each time it is asked, and never
// kept with it: a table changed in place after it was fingerprinted has the fingerprint of its
// changed rows, and so has a Query over it. A caller that holds the fingerprint writes the Query's
// canonical form the table would.
func TestAChangedTableIsFingerprintedAfresh(t *testing.T) {
	tb := doorTable("door")
	q := umpire.KeyFind("q", opensLoudlyOn(tb), umpire.KeyScenario(tb, "turnThenPush", "closed-false", "turn-right-true", "push"), two)
	const property = "sha256:property"
	before, query := tb.TargetFingerprint(), q.QueryCanonical(tb, property)
	require.Equal(t, before, tb.TargetFingerprint())
	require.Equal(t, query, q.QueryCanonicalOf(tb, property, before))

	tb.Rows[0].Results = append(tb.Rows[0].Results, resultOf("ok", "closed-false"))
	after := tb.TargetFingerprint()
	require.NotEqual(t, before, after)
	require.NotEqual(t, query, q.QueryCanonical(tb, property))
	require.Equal(t, q.QueryCanonical(tb, property), q.QueryCanonicalOf(tb, property, after))
	require.NotEqual(t, q.QueryCanonical(tb, property), q.QueryCanonicalOf(tb, property, before))
}
