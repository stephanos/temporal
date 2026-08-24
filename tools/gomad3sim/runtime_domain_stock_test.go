//go:build !gomad3_toolchain

package gomad3sim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunFailsClosedWithoutPinnedRuntime(t *testing.T) {
	bootID := uniqueBootID("cluster-stock-runtime")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error { return nil }))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendInProcess,
		Fidelity: FidelitySimulationModel,
		Seed:     71,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	called := false
	result, err := Run(context.Background(), spec, func(context.Context, Cluster) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, ErrRuntimeUnavailable)
	require.Equal(t, Result{}, result)
	require.False(t, called)
}

func TestRunRejectsUnavailableProcessBackendBeforeScenario(t *testing.T) {
	bootID := uniqueBootID("cluster-process-runtime")
	require.NoError(t, RegisterBoot(bootID, func(context.Context, NodeContext) error { return nil }))
	spec := Spec{
		Schema:   SpecSchema,
		Backend:  BackendProcess,
		Fidelity: FidelityHardIsolation,
		Seed:     75,
		Limits:   DefaultLimits(),
		Nodes:    []NodeSpec{{ID: "node", Boot: bootID, Address: "10.0.0.1"}},
	}
	called := false
	result, err := Run(context.Background(), spec, func(context.Context, Cluster) error {
		called = true
		return nil
	})
	var unavailable *BackendUnavailableError
	require.ErrorAs(t, err, &unavailable)
	require.Equal(t, BackendProcess, unavailable.Backend)
	require.Equal(t, Result{}, result)
	require.False(t, called)
}
