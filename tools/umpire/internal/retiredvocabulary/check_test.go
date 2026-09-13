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

func TestAllowedNegativeFixtureHoldsRetiredTokensOnlyUnderTheMigrationBaseline(t *testing.T) {
	t.Parallel()

	token := "Transition" + "Kernel"
	require.True(t, allowedNegativeFixture(
		"common/testing/testpilot/internal/protocolmigration/testdata/baseline/fixtures/tests/testcore/testpilot/testdata/typed-nexus-case.json",
		token,
	))
	for _, path := range []string{
		"common/testing/testpilot/internal/protocolmigration/mapping.go",
		"common/testing/testpilot/internal/protocolmigration/README.md",
		"common/testing/testpilot/internal/protocolmigration/testdata/baseline-copy/case.json",
		"tests/testcore/testpilot/testdata/typed-nexus-case.json",
	} {
		require.False(t, allowedNegativeFixture(path, token), "path %q", path)
	}
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
		"await_" + "outcome",
		"behavior" + "%",
	} {
		require.NoError(t, validateRetiredToken(token), "token %q", token)
	}
}
