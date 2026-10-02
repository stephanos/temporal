package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	core "go.temporal.io/server/tools/umpire/model/internal/checker"
)

func TestReceiptAndQueryOutcomeSpellingsRemainCompatible(t *testing.T) {
	require.Equal(t, []Outcome{core.Found, core.NotFound, core.LimitReached, core.Unresolved},
		[]Outcome{Outcome(Found), Outcome(NotFound), Outcome(LimitReached), Outcome(Unresolved)})
}
