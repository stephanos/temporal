package tla

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const tlcLivenessViolation = `Error: Temporal properties were violated.

Error: The following behavior constitutes a counter-example:

State 1: <Initial predicate>
phase = "unavailable"

State 2: <RecoverOwner line 24, col 5 to line 25, col 26 of module Generated>
phase = "ready"

State 3: Stuttering
`

func TestNormalizeTLCReplaysLassoWithoutPromotingFiniteSuccess(t *testing.T) {
	t.Parallel()

	mutated := temporalView(t, "delivery-fairness-removed")
	result, err := NormalizeTLC(mutated, RawResult{
		Output: tlcLivenessViolation, ExitCode: tlcLivenessViolationExitCode, Limits: testToolLimits(),
	})
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassLassoWitness, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeExternalTool, result.TrustBadge)
	require.Equal(t, &protocol.TemporalLasso{
		States:    []string{"unavailable", "ready"},
		Actions:   []protocol.ActionKind{protocol.ActionKindRecoverOwner, ""},
		LoopStart: 1,
	}, result.Lasso)

	sound := temporalView(t, "sound")
	result, err = NormalizeTLC(sound, RawResult{
		Output: "Model checking completed. No error has been found", ExitCode: 0, Limits: testToolLimits(),
	})
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassExternalNoCounterexample, result.ResultClass)
	require.False(t, result.Exact)
	require.NotEqual(t, protocol.ResultClassTemporalProved, result.ResultClass)
}

func TestNormalizeApalacheDoesNotClaimUnboundedTemporalProof(t *testing.T) {
	t.Parallel()

	view := temporalView(t, "sound")
	result, err := NormalizeApalache(view, RawResult{
		Output: "Checker reports no error", ExitCode: 0, Limits: testToolLimits(),
	})
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassBoundedSafe, result.ResultClass)
	require.Equal(t, view.Bounds.MaxTraceLength, result.Bound)
	require.Contains(t, result.Omissions, "temporal-property-not-checked")
	require.NotEqual(t, protocol.ResultClassTemporalProved, result.ResultClass)
}

func TestTemporalBackendResultRoundTripsStrictlyAndRejectsTampering(t *testing.T) {
	t.Parallel()

	view := temporalView(t, "sound")
	result, err := NormalizeTLC(view, RawResult{
		Output: "Model checking completed. No error has been found", ExitCode: 0, Limits: testToolLimits(),
	})
	require.NoError(t, err)
	encoded, err := result.CanonicalJSON(view)
	require.NoError(t, err)
	decoded, err := DecodeResult(bytes.NewReader(encoded), protocol.DefaultDecodeLimit, view)
	require.NoError(t, err)
	require.Equal(t, result, decoded)
	require.True(t, resultDigest.MatchString(result.GeneratedArtifactDigest))

	tampered := result
	tampered.Fairness = nil
	require.Error(t, tampered.Validate(view))
	tampered = result
	tampered.GeneratedArtifactDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	require.NoError(t, tampered.seal())
	require.ErrorContains(t, tampered.Validate(view), "generated TLA+ artifact")

	withUnknownField := strings.Replace(string(encoded), `"formatVersion":`, `"unknown":true,"formatVersion":`, 1)
	_, err = DecodeResult(strings.NewReader(withUnknownField), protocol.DefaultDecodeLimit, view)
	require.ErrorContains(t, err, "unknown field")
}

func TestTemporalBackendResultRejectsUnexpectedCheckerExit(t *testing.T) {
	t.Parallel()

	view := temporalView(t, "sound")
	_, err := NormalizeTLC(view, RawResult{
		Output: "Model checking completed. No error has been found", ExitCode: 1, Limits: testToolLimits(),
	})
	require.ErrorContains(t, err, "exited with status 1")
	_, err = NormalizeApalache(view, RawResult{
		Output: "Checker reports no error", ExitCode: 1, Limits: testToolLimits(),
	})
	require.ErrorContains(t, err, "exited with status 1")
}

func testToolLimits() ToolLimits {
	return ToolLimits{
		Timeout: 30 * time.Second, MaxOutputBytes: protocol.DefaultDecodeLimit,
		CPUSeconds: 30, MemoryBytes: 2 << 30,
	}
}
