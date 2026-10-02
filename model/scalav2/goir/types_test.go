package goir

import (
	"testing"

	"github.com/stretchr/testify/require"
	core "go.temporal.io/server/model/scalav2/goir/internal/checker"
)

func TestReceiptAndQueryOutcomeSpellingsRemainCompatible(t *testing.T) {
	require.Equal(t, []Outcome{core.Found, core.NotFound, core.LimitReached, core.Unresolved},
		[]Outcome{Outcome(Found), Outcome(NotFound), Outcome(LimitReached), Outcome(Unresolved)})
}
