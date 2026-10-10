package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	core "go.temporal.io/server/tools/umpire/internal/engine"
)

func TestReceiptAndQueryOutcomeSpellingsRemainCompatible(t *testing.T) {
	require.Equal(t, []Outcome{core.Found, core.NotFound, core.LimitReached, core.Unresolved},
		[]Outcome{Outcome(Found), Outcome(NotFound), Outcome(LimitReached), Outcome(Unresolved)})
}

func TestReceiptClassificationIsDerivedFromItsSubject(t *testing.T) {
	for _, tc := range []struct {
		subject Subject
		want    ClaimClassification
	}{
		{QuerySubject, Safety},
		{ProgressSubject, BoundedLiveness},
		{ModelSubject, ""},
		{MachineSubject, ""},
		{RefinementSubject, ""},
		{CompositionSubject, ""},
	} {
		t.Run(string(tc.subject), func(t *testing.T) {
			for _, kind := range []ReceiptKind{Verified, Found, Counterexample, Unsupported, Incomplete, DeclarationError, LimitReached, Unresolved} {
				r := Receipt{Subject: tc.subject, Kind: kind}
				require.Equal(t, tc.want, r.Classification(), string(kind))
			}
		})
	}
	require.Equal(t, Safety, ClaimClassification("safety"))
	require.Equal(t, BoundedLiveness, ClaimClassification("bounded-liveness"))
}
