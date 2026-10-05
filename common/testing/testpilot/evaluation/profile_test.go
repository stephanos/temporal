package evaluation

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// localEphemeralIdentity is the identity the embedded Profile was rendered under. No live target
// renders the file any more, so this pin is what keeps the Go reading of the checked-in bytes equal
// to that identity.
const localEphemeralIdentity = "sha256:05b2152d5129ccb4e8fe9fd6625b423ddfcf0264cafbcfeff9910cf2819c7d5a"

func readTestProfile(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile("testdata/test-profiles/" + name + ".json")
	require.NoError(t, err)
	return encoded
}

func TestLoadProfileSelectsAnEmbeddedProfileByItsExactName(t *testing.T) {
	profile, err := LoadProfile("local-ephemeral")
	require.NoError(t, err)
	require.Equal(t, localEphemeralIdentity, profile.Identity)
	require.Equal(t, "local-ephemeral-cluster", profile.Trust)
	require.Equal(t, []string{"capability", "interpretation"}, profile.BlockingKnownGaps)
	require.Equal(t, DecisionIncomplete, profile.UnsupportedRule)

	for _, name := range []string{"", "local", "Local-Ephemeral", "local-ephemeral.json", "profiles/local-ephemeral", "../evaluation/profiles/local-ephemeral", "local-strict"} {
		_, err := LoadProfile(name)
		require.ErrorIs(t, err, ErrUnknownProfile, "a Profile is a name from the embedded set, never a path: %q", name)
		require.True(t, strings.HasSuffix(err.Error(), "; the Profiles are local-ephemeral"), "the embedded set is the one Profile: %v", err)
	}
}

// The test-only Profile is a second, independent Profile: it parses, and its identity is its own.
func TestTheTestOnlyProfileIsAnotherProfile(t *testing.T) {
	strict, err := ParseProfile(readTestProfile(t, "local-strict"))
	require.NoError(t, err)
	require.Equal(t, "local-strict", strict.Name)
	require.NotEqual(t, localEphemeralIdentity, strict.Identity)
}

// Every load rejection: the format version, the name, the claim and trust, the closed Known Gap
// kinds in their order, the unsupported-rule decision, and the canonical form a rendering carries.
// A version 1 Profile, which carried a reason table, is another format.
func TestParseProfileRejectsAnInvalidProfile(t *testing.T) {
	embedded, err := embeddedProfiles.ReadFile("profiles/local-ephemeral.json")
	require.NoError(t, err)
	valid := string(embedded)
	_, err = ParseProfile(embedded)
	require.NoError(t, err)
	mutate := func(old, replacement string) []byte {
		require.Contains(t, valid, old)
		return []byte(strings.Replace(valid, old, replacement, 1))
	}
	for name, probe := range map[string]struct {
		encoded []byte
		detail  string
	}{
		"unknown unsupported-rule decision": {mutate(`"unsupportedRule":"incomplete"`, `"unsupportedRule":"accepted"`), `unknown decision "accepted"`},
		"no unsupported-rule decision":      {mutate(`"unsupportedRule":"incomplete"`, `"unsupportedRule":""`), `unknown decision ""`},
		"unknown Known Gap kind":            {mutate(`["capability","interpretation"]`, `["capability","vibes"]`), `unknown Known Gap kind "vibes"`},
		"a duplicate blocking kind":         {mutate(`["capability","interpretation"]`, `["capability","capability"]`), `kind "capability" is named twice`},
		"kinds out of order":                {mutate(`["capability","interpretation"]`, `["interpretation","capability"]`), "not in the kind order"},
		"a reason table":                    {mutate(`,"unsupportedRule":`, `,"reasons":[],"unsupportedRule":`), "unknown field"},
		"the format with a reason table":    {[]byte(`{"version":1,"name":"x","claim":"c","trust":"t","blockingKnownGaps":[],"reasons":[{"name":"r","condition":"verdict-violated","decision":"rejected"}]}` + "\n"), "unknown field"},
		"another format version":            {mutate(`{"version":2,`, `{"version":1,`), "format version 1"},
		"an invalid name":                   {mutate(`"name":"local-ephemeral"`, `"name":"Local"`), `Profile name "Local"`},
		"an empty claim":                    {mutate(`"claim":"The Case's Contract held for one closed Run of the Case against an ephemeral local test cluster, under the recorded Driver identity."`, `"claim":""`), "no claim"},
		"an empty trust basis":              {mutate(`"trust":"local-ephemeral-cluster"`, `"trust":""`), "no trust basis"},
		"an unknown field":                  {mutate(`{"version":2,`, `{"version":2,"extra":1,`), "unknown field"},
		"a case-folded key":                 {mutate(`"claim":`, `"Claim":`), "canonical form"},
		"a repeated key":                    {mutate(`"trust":"local-ephemeral-cluster",`, `"trust":"x","trust":"local-ephemeral-cluster",`), "canonical form"},
		"other spacing":                     {mutate(`{"version":2,`, `{"version": 2,`), "canonical form"},
		"no trailing newline":               {[]byte(strings.TrimSuffix(valid, "\n")), "canonical form"},
		"a second document":                 {[]byte(valid + valid), "canonical form"},
		"another field order":               {mutate(`"name":"local-ephemeral","claim":`, `"claim":`), "canonical form"},
		"null blocking kinds":               {mutate(`["capability","interpretation"]`, `null`), "canonical form"},
		"not JSON":                          {[]byte("{"), "decode Profile"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseProfile(probe.encoded)
			require.ErrorContains(t, err, probe.detail)
		})
	}
}

// A Profile with no blocking kind blocks on none.
func TestAProfileMayBlockOnNoKnownGapKind(t *testing.T) {
	embedded, err := embeddedProfiles.ReadFile("profiles/local-ephemeral.json")
	require.NoError(t, err)
	profile, err := ParseProfile([]byte(strings.Replace(string(embedded), `["capability","interpretation"]`, `[]`, 1)))
	require.NoError(t, err)
	require.Empty(t, profile.BlockingKnownGaps)
}
