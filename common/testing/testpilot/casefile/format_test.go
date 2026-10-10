package casefile

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCELFormatBoundary(t *testing.T) {
	require.NoError(t, checkVersion(CELMajor, CELMinor, CELMajor, CELMinor))
	for _, major := range []int32{0, 1, 2, 3, 5, 99} {
		require.Error(t, checkVersion(major, 0, CELMajor, CELMinor))
	}
	require.Error(t, checkVersion(CELMajor, 1, CELMajor, CELMinor))
}

func TestFormatEnvelopePreservesProtoJSONNumbers(t *testing.T) {
	for _, version := range []string{`{"major":1e0}`, `{"major":"1e0","minor":0e0}`} {
		raw, err := JSONVersion([]byte(`{"program":{"retiredPayload":true},"version":` + version + `}`))
		require.NoError(t, err)
		require.Equal(t, version, string(raw))
	}
}
