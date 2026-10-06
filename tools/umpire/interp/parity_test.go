package interp

// The Nexus caller Model, lifted from model/temporal into model/ir/nexus-caller.json and interpreted
// here. The IR carries the Model whole: nothing of the Scala code is run to get these tables.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const irPath = "../../../model/ir/nexus-caller.json"

func machines(t *testing.T) map[string]*Machine {
	t.Helper()
	built, err := Build(readIR(t, irPath))
	require.NoError(t, err)
	return built
}

func TestStuckStatesAndEvidence(t *testing.T) {
	built := machines(t)
	for _, name := range []string{"nexusProduct", "nexusSystem", "polling", "handlerWorker"} {
		require.Empty(t, stuck(built[name].Table), name)
	}
	require.Equal(t, [][2]string{{"nexusOperationScheduled", "nexusOperationScheduled"}, {"nexusOperationStarted", "nexusOperationStarted"},
		{"nexusOperationCompleted", "nexusOperationCompleted"}, {"nexusOperationFailed", "nexusOperationFailed"},
		{"nexusOperationCanceled", "nexusOperationCanceled"}, {"nexusOperationTimedOut", "nexusOperationTimedOut"},
		{"pendingAttempts", "pendingAttempts"}}, built["nexusSystem"].Table.Evidence)
}
