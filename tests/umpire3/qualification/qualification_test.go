package qualification

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQualifyRejectsIncompleteInput(t *testing.T) {
	_, err := Qualify(Request{})
	require.EqualError(t, err, "release, experiment, result, and profile are required")
}

func TestQualifyRejectsIncompleteCanaryEnvelope(t *testing.T) {
	_, err := DecodeResult([]byte(`{"formatVersion":"umpire3/canary/v1","runtime":{},"complete":false}`))
	require.ErrorContains(t, err, "canary result is incomplete")
}
