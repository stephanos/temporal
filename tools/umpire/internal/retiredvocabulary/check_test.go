package retiredvocabulary

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Token literals are split so this file, which the scan reads like any other,
// does not trip the rules it exercises.
func TestRetiredRulesAreConfigured(t *testing.T) {
	t.Parallel()

	require.NoError(t, retiredRulesError)
	require.NotEmpty(t, retiredRules)
}

func TestValidateRetiredTokenRejectsBareWords(t *testing.T) {
	t.Parallel()

	for _, token := range []string{"Target", "Behavior", "Case", "Step", "bou" + "nds", "T"} {
		require.ErrorContains(t, validateRetiredToken(token), "bare word", "token %q", token)
	}
	require.ErrorContains(t, validateRetiredToken(""), "must not be empty")

	for _, token := range []string{
		"Transition" + "Kernel",
		"semantic" + "Identity",
		"umpire-gen-regression-" + "projections",
		"Umpire." + "Refinement",
		"umpire-experiment/" + "v1",
		"await_outcome",
		"behavior%",
	} {
		require.NoError(t, validateRetiredToken(token), "token %q", token)
	}
}
