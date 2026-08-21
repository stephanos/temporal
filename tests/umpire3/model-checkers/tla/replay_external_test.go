//go:build umpire3_tla_experiment

package tla

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestCanonicalLeanReplayChecksTLCLasso(t *testing.T) {
	command := os.Getenv("UMPIRE3_TEMPORAL_LASSO_REPLAY")
	if command == "" {
		t.Skip("canonical temporal lasso replay executable is not configured")
	}
	view := temporalView(t, "delivery-fairness-removed")
	input := protocol.TemporalLassoReplayInput{
		FormatVersion: protocol.TemporalLassoReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  view.SemanticHash,
		Lasso: protocol.TemporalLasso{
			States:    []string{"unavailable", "ready"},
			Actions:   []protocol.ActionKind{protocol.ActionKindRecoverOwner, ""},
			LoopStart: 1,
		},
	}
	receipt, err := ReplayLasso(context.Background(), []string{command}, input)
	require.NoError(t, err)
	require.Equal(t, protocol.TrustBadgeCheckedCertificate, receipt.TrustBadge)

	backend, err := NormalizeTLC(view, RawResult{
		Output: tlcLivenessViolation, ExitCode: tlcLivenessViolationExitCode, Limits: testToolLimits(),
	})
	require.NoError(t, err)
	backend, err = AttachReplay(backend, receipt)
	require.NoError(t, err)
	require.Equal(t, protocol.TrustBadgeCheckedCertificate, backend.TrustBadge)
	require.NotNil(t, backend.Replay)
}
