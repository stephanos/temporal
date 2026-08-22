package execution

import (
	"testing"

	"github.com/stretchr/testify/require"
	protocolexecution "go.temporal.io/server/tests/umpire3/protocol/execution"
)

func TestClassifyOutcomeKeepsLifecycleSeparateFromPropertyClaim(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		claim    protocolexecution.ClaimKind
		terminal *protocolexecution.TerminalEvidence
		kind     protocolexecution.OutcomeKind
	}{
		{name: "success", claim: protocolexecution.ClaimConforming, terminal: terminal("succeeded", protocolexecution.TerminalDispositionSuccess), kind: protocolexecution.OutcomeRecovered},
		{name: "untagged terminal", claim: protocolexecution.ClaimConforming, terminal: terminal("terminated", protocolexecution.TerminalDispositionUntagged), kind: protocolexecution.OutcomeRecovered},
		{name: "allowed failure", claim: protocolexecution.ClaimConforming, terminal: terminal("failed", protocolexecution.TerminalDispositionFailure), kind: protocolexecution.OutcomeDegraded},
		{name: "property violation wins", claim: protocolexecution.ClaimViolating, terminal: terminal("failed", protocolexecution.TerminalDispositionFailure), kind: protocolexecution.OutcomeFlagged},
		{name: "no terminal", claim: protocolexecution.ClaimConforming, kind: protocolexecution.OutcomeUnreached},
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

	_, err := ClassifyOutcome(protocolexecution.ClaimConforming, &protocolexecution.TerminalEvidence{State: "failed"})
	require.ErrorContains(t, err, "disposition")
}

func terminal(state string, disposition protocolexecution.TerminalDisposition) *protocolexecution.TerminalEvidence {
	return &protocolexecution.TerminalEvidence{
		State: state, Disposition: disposition, Reference: "history/terminal", EntityIdentity: "workflow/run",
	}
}
