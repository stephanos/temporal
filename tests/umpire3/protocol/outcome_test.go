package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassifyOutcomeKeepsLifecycleSeparateFromPropertyClaim(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		claim    ClaimKind
		terminal *TerminalEvidence
		kind     OutcomeKind
	}{
		{name: "success", claim: ClaimConforming, terminal: terminal("succeeded", TerminalDispositionSuccess), kind: OutcomeRecovered},
		{name: "untagged terminal", claim: ClaimConforming, terminal: terminal("terminated", TerminalDispositionUntagged), kind: OutcomeRecovered},
		{name: "allowed failure", claim: ClaimConforming, terminal: terminal("failed", TerminalDispositionFailure), kind: OutcomeDegraded},
		{name: "property violation wins", claim: ClaimViolating, terminal: terminal("failed", TerminalDispositionFailure), kind: OutcomeFlagged},
		{name: "no terminal", claim: ClaimConforming, kind: OutcomeUnreached},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			outcome, err := ClassifyOutcome(test.claim, test.terminal)
			require.NoError(t, err)
			require.Equal(t, test.kind, outcome.Kind)
		})
	}
}

func TestClassifyOutcomeRejectsIncompleteTerminalEvidence(t *testing.T) {
	t.Parallel()

	_, err := ClassifyOutcome(ClaimConforming, &TerminalEvidence{State: "failed"})
	require.ErrorContains(t, err, "disposition")
}

func terminal(state string, disposition TerminalDisposition) *TerminalEvidence {
	return &TerminalEvidence{
		State: state, Disposition: disposition, Reference: "history/terminal", EntityIdentity: "workflow/run",
	}
}
