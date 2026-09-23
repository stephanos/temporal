package evaluation

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// localEphemeralIdentity is the identity Temporal/Evaluation/LocalTests.lean pins; the Makefile's
// byte gate keeps the embedded file equal to Lean's rendering, and this keeps the Go reading of it
// equal to Lean's identity.
const localEphemeralIdentity = "sha256:2803afa29ed404cf0a774ead9c0672de29a6d38f27fe54fd42014f0b2f78174c"

func readTestProfile(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile("testdata/test-profiles/" + name + ".json")
	require.NoError(t, err)
	return encoded
}

func TestLoadProfileSelectsAnEmbeddedProfileByItsExactName(t *testing.T) {
	names, err := ProfileNames()
	require.NoError(t, err)
	require.Equal(t, []string{"local-ephemeral"}, names)

	profile, err := LoadProfile("local-ephemeral")
	require.NoError(t, err)
	require.Equal(t, localEphemeralIdentity, profile.Identity)
	require.Equal(t, "local-ephemeral-cluster", profile.Trust)
	require.Equal(t, []string{"capability", "interpretation"}, profile.BlockingKnownGaps)
	var table []string
	for _, reason := range profile.Reasons {
		table = append(table, reason.Condition+"="+reason.Decision)
	}
	require.Equal(t, []string{
		"verdict-violated=rejected", "disposition-stopped=rejected", "verdict-inconclusive=incomplete",
		"disposition-incomplete=incomplete", "cleanup-unclosed=incomplete", "known-gap-blocking=incomplete",
		"unsupported-rule=incomplete",
	}, table)

	for _, name := range []string{"", "local", "Local-Ephemeral", "local-ephemeral.json", "profiles/local-ephemeral", "../evaluation/profiles/local-ephemeral", "local-strict"} {
		_, err := LoadProfile(name)
		require.ErrorIs(t, err, ErrUnknownProfile, "a Profile is a name from the embedded set, never a path: %q", name)
	}
}

// The test-only Profile is a second, independent Profile: it parses, and its identity is its own.
func TestTheTestOnlyProfileIsAnotherProfile(t *testing.T) {
	strict, err := parseProfile(readTestProfile(t, "local-strict"))
	require.NoError(t, err)
	require.Equal(t, "local-strict", strict.Name)
	require.NotEqual(t, localEphemeralIdentity, strict.Identity)
}

// Every load rejection Lean's declaration check makes, plus the closed vocabularies and the
// canonical form a rendering carries.
func TestParseProfileRejectsWhatLeanWouldNotDeclare(t *testing.T) {
	embedded, err := embeddedProfiles.ReadFile("profiles/local-ephemeral.json")
	require.NoError(t, err)
	valid := string(embedded)
	_, err = parseProfile(embedded)
	require.NoError(t, err)
	mutate := func(old, replacement string) []byte {
		require.Contains(t, valid, old)
		return []byte(strings.Replace(valid, old, replacement, 1))
	}
	gapReason := `,{"name":"known-gap-blocking","condition":"known-gap-blocking","decision":"incomplete"}`
	for name, probe := range map[string]struct {
		encoded []byte
		detail  string
	}{
		"unknown condition":         {mutate(`"condition":"cleanup-unclosed"`, `"condition":"cleanup-leaked"`), `unknown condition "cleanup-leaked"`},
		"unknown decision":          {mutate(`"decision":"rejected"`, `"decision":"accepted"`), `unknown decision "accepted"`},
		"unknown Known Gap kind":    {mutate(`["capability","interpretation"]`, `["capability","vibes"]`), `unknown Known Gap kind "vibes"`},
		"an empty table":            {[]byte(`{"version":1,"name":"x","claim":"c","trust":"t","blockingKnownGaps":[],"reasons":[]}` + "\n"), "reason table is empty"},
		"an empty reason name":      {mutate(`"name":"verdict-violated"`, `"name":""`), "empty name"},
		"a duplicate reason":        {mutate(`"name":"monitor-stopped"`, `"name":"verdict-violated"`), `reason "verdict-violated" is declared twice`},
		"a repeated condition":      {mutate(`"condition":"disposition-stopped"`, `"condition":"verdict-violated"`), `condition "verdict-violated" is named by two reasons`},
		"a duplicate blocking kind": {mutate(`["capability","interpretation"]`, `["capability","capability"]`), `kind "capability" is named twice`},
		"kinds out of order":        {mutate(`["capability","interpretation"]`, `["interpretation","capability"]`), "not in the kind order"},
		"blocking without kinds":    {mutate(`["capability","interpretation"]`, `[]`), "with no blocking kind"},
		"kinds without blocking":    {mutate(gapReason, ``), "no 'known-gap-blocking' reason"},
		"another format version":    {mutate(`{"version":1,`, `{"version":2,`), "format version 2"},
		"an invalid name":           {mutate(`"name":"local-ephemeral"`, `"name":"Local"`), `Profile name "Local"`},
		"an empty claim":            {mutate(`"claim":"The Case's Contract held for one closed Run of the Case against an ephemeral local test cluster, under the recorded Driver identity."`, `"claim":""`), "no claim"},
		"an empty trust basis":      {mutate(`"trust":"local-ephemeral-cluster"`, `"trust":""`), "no trust basis"},
		"an unknown field":          {mutate(`{"version":1,`, `{"version":1,"extra":1,`), "unknown field"},
		"a case-folded key":         {mutate(`"claim":`, `"Claim":`), "canonical form"},
		"a repeated key":            {mutate(`"trust":"local-ephemeral-cluster",`, `"trust":"x","trust":"local-ephemeral-cluster",`), "canonical form"},
		"other spacing":             {mutate(`{"version":1,`, `{"version": 1,`), "canonical form"},
		"no trailing newline":       {[]byte(strings.TrimSuffix(valid, "\n")), "canonical form"},
		"a second document":         {[]byte(valid + valid), "canonical form"},
		"another field order":       {mutate(`"name":"local-ephemeral","claim":`, `"claim":`), "canonical form"},
		"null blocking kinds":       {mutate(`["capability","interpretation"]`, `null`), "canonical form"},
		"not JSON":                  {[]byte("{"), "decode Profile"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseProfile(probe.encoded)
			require.ErrorContains(t, err, probe.detail)
		})
	}
}
