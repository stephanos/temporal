package tla

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestPinnedTLCChecksSoundProgressAndFindsMutation(t *testing.T) {
	javaPath := os.Getenv("UMPIRE_JAVA_TOOL")
	tlaJar := os.Getenv("UMPIRE_TLA_JAR")
	if javaPath == "" || tlaJar == "" {
		t.Skip("pinned TLC environment is not loaded")
	}
	limits := externalLimits()
	sound, err := CheckTLC(context.Background(), temporalView(t, "sound"), javaPath, tlaJar, limits)
	require.NoError(t, err, sound.Output)
	require.Contains(t, sound.Output, "Model checking completed. No error has been found")

	mutated, err := CheckTLC(context.Background(), temporalView(t, "delivery-fairness-removed"),
		javaPath, tlaJar, limits)
	require.NoError(t, err, mutated.Output)
	require.Equal(t, tlcLivenessViolationExitCode, mutated.ExitCode)
	require.Contains(t, mutated.Output, "Temporal properties were violated")
	normalized, err := NormalizeTLC(temporalView(t, "delivery-fairness-removed"), mutated)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassLassoWitness, normalized.ResultClass)
}

func TestPinnedApalacheChecksBoundedTypeInvariant(t *testing.T) {
	apalachePath := os.Getenv("UMPIRE_APALACHE_TOOL")
	if apalachePath == "" {
		t.Skip("pinned Apalache environment is not loaded")
	}
	result, err := CheckApalache(context.Background(), temporalView(t, "sound"),
		apalachePath, externalLimits())
	require.NoError(t, err, result.Output)
	require.Contains(t, result.Output, "Checker reports no error")
	normalized, err := NormalizeApalache(temporalView(t, "sound"), result)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassBoundedSafe, normalized.ResultClass)
}

func externalLimits() ToolLimits {
	return ToolLimits{
		Timeout: 30 * time.Second, MaxOutputBytes: 4 << 20,
		CPUSeconds: 30, MemoryBytes: 2 << 30,
	}
}
