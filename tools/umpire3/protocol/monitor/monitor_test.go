package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonitorCatalogRejectsProductSpecificUnknownObservation(t *testing.T) {
	catalog, err := DefaultMonitorCatalog()
	require.NoError(t, err)
	catalog.Programs[0].Expression = MonitorExpression{
		Operation:   MonitorObservation,
		Observation: "unknown-observation",
		Expected:    boolPointer(true),
	}
	require.ErrorContains(t, catalog.Validate(), "unknown observation")
}

func boolPointer(value bool) *bool {
	return &value
}
