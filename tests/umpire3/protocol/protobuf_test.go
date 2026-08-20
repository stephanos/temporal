package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedProtobufInventoryIsAvailableToRuntimeTools(t *testing.T) {
	t.Parallel()

	inventory, err := DefaultProtobufInventory()
	require.NoError(t, err)
	require.Len(t, inventory.Roots, 24)
	require.Len(t, inventory.Fields, 397)
	require.Equal(t, 84, inventory.Messages)
	require.Equal(t, 21, inventory.Enums)
	require.Contains(t, inventory.Roots, "temporal.api.common.v1.Callback")
	require.Contains(t, inventory.Roots, "temporal.api.common.v1.Link")
	require.Contains(t, inventory.Roots, "temporal.api.workflowservice.v1.ResetWorkflowExecutionRequest")
	require.Contains(t, inventory.Roots, "temporal.api.workflowservice.v1.StartActivityExecutionRequest")
	require.Contains(t, inventory.FieldClasses, "presence")
	require.Contains(t, inventory.FieldClasses, "map")
	require.Contains(t, inventory.FieldClasses, "recursive")
	require.Contains(t, inventory.FieldClasses, "kind:enum")
}
