package tla

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func temporalView(t *testing.T, variant string) protocol.TemporalView {
	t.Helper()
	view, found, err := protocol.DefaultTemporalView(variant)
	require.NoError(t, err)
	require.True(t, found)
	return view
}
