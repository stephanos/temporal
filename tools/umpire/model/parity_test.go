package model

// The Nexus caller Model, lifted from model/scalav2/scala into ir/nexus-caller.json and interpreted
// here. The immutable reader goldens retain its complete tables and identities;
// the IR carries the Model whole: nothing of the Scala code is run to get these tables.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const irPath = "../../../model/ir/nexus-caller.json"

func machines(t *testing.T) map[string]*Machine {
	t.Helper()
	m, err := Load(irPath)
	require.NoError(t, err)
	built, err := Build(m)
	require.NoError(t, err)
	return built
}

func TestStuckStatesAndEvidence(t *testing.T) {
	built := machines(t)
	for _, name := range []string{"nexusProduct", "nexusProtocol", "polling", "handlerWorker"} {
		require.Empty(t, built[name].Table.Stuck, name)
	}
	require.Equal(t, [][2]string{{"nexusOperationScheduled", "nexusOperationScheduled"}, {"nexusOperationStarted", "nexusOperationStarted"},
		{"nexusOperationCompleted", "nexusOperationCompleted"}, {"nexusOperationFailed", "nexusOperationFailed"},
		{"nexusOperationCanceled", "nexusOperationCanceled"}, {"nexusOperationTimedOut", "nexusOperationTimedOut"},
		{"pendingAttempts", "pendingAttempts"}}, built["nexusProtocol"].Table.Evidence)
}
