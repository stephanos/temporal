package delivery_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
)

func TestProductionPackageBuildsForExternalConsumer(t *testing.T) {
	ledger, err := delivery.New(delivery.Config{
		RunID:     "run",
		SessionID: "session",
		Limits:    delivery.Limits{MaxRoutes: 1, MaxHeaderBytes: 1024, MaxHandles: 1, MaxDiagnostics: 1},
	})
	require.NoError(t, err)
	require.NotNil(t, ledger)
}
