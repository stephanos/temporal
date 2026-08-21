package tla

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateTLAFromTemporalView(t *testing.T) {
	t.Parallel()

	view := temporalView(t, "sound")
	generated, err := Generate(view)
	require.NoError(t, err)
	require.Equal(t, "Umpire3FoundationDeliverySafetySound", generated.Module)
	require.Contains(t, string(generated.TLA), `RecoverOwner ==`)
	require.Contains(t, string(generated.TLA), `ProgressEntity ==`)
	require.Contains(t, string(generated.TLA),
		`ResponsiveRecoverOwner == [](phase \in {"unavailable"} => <> (phase \notin {"unavailable"}))`)
	require.Contains(t, string(generated.TLA),
		`Progress == [](phase \in {"ready", "unavailable"} => <> (phase \in {"completed"}))`)
	require.Equal(t,
		"SPECIFICATION Spec\nINVARIANT TypeOK\nPROPERTY Progress\nCHECK_DEADLOCK FALSE\n",
		string(generated.Config))
}

func TestMutationOmitsDeliveryTransitionAndFairness(t *testing.T) {
	t.Parallel()

	view := temporalView(t, "delivery-fairness-removed")
	generated, err := Generate(view)
	require.NoError(t, err)
	require.Contains(t, string(generated.TLA), "ProgressEntity ==\n    FALSE")
	require.NotContains(t, string(generated.TLA), "ResponsiveProgressEntity")
}
