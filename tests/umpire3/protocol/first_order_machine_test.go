package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstOrderMachineReconstructsGeneratedOracle(t *testing.T) {
	t.Parallel()

	view, found, err := DefaultFirstOrderView(TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	machine, err := NewFirstOrderMachine(view)
	require.NoError(t, err)
	initials, err := machine.InitialStates()
	require.NoError(t, err)
	require.Len(t, initials, 1)

	queue := append([]FirstOrderState(nil), initials...)
	seen := make(map[string]struct{})
	for _, state := range queue {
		key, keyErr := machine.StateKey(state)
		require.NoError(t, keyErr)
		seen[key] = struct{}{}
	}
	for index := 0; index < len(queue); index++ {
		safe, invariantErr := machine.Invariant(queue[index])
		require.NoError(t, invariantErr)
		require.True(t, safe)
		steps, successorErr := machine.Successors(queue[index])
		require.NoError(t, successorErr)
		for _, step := range steps {
			key, keyErr := machine.StateKey(step.State)
			require.NoError(t, keyErr)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			queue = append(queue, step.State)
		}
	}
	require.Len(t, queue, len(view.Oracle.States))
}
