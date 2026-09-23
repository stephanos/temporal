package policy

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheCommittedPolicyDecodesUnconfigured(t *testing.T) {
	committed, err := Embedded()
	require.NoError(t, err)
	require.False(t, committed.Configured(), "the operator commits the coordinate digests")
	require.Equal(t, "production-canary", committed.CaseProfile)
	require.Equal(t, "production-canary", committed.EvaluationProfile)
	require.Equal(t, AuthorityProtectedWorkflow, committed.AuthorityClass)
	require.Equal(t, "refs/heads/main", committed.TrustedRef)
	require.Equal(t, 2, committed.Limits.Iterations)
	require.Greater(t, committed.Limits.LeaseRunTimeout(), committed.Limits.Invocation()+committed.Limits.CleanupReserve())
}

func TestDigestIsTheWholeValue(t *testing.T) {
	require.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", Digest(""))
	require.NotEqual(t, Digest("host:7233"), Digest("host"), "the gRPC digest is of the whole dial target")
}

// Every field is checked by name, and the bytes must be the canonical rendering.
func TestDecodeRejectsEachMalformation(t *testing.T) {
	valid := string(embedded)
	configured := strings.Replace(valid, `"grpc": "unconfigured"`, `"grpc": "`+Digest("host:7233")+`"`, 1)
	decoded, err := Decode([]byte(configured))
	require.NoError(t, err)
	require.False(t, decoded.Configured(), "one digest is not all of them")
	mutate := func(old, replacement string) string {
		require.Contains(t, valid, old)
		return strings.Replace(valid, old, replacement, 1)
	}
	for name, probe := range map[string]struct {
		encoded string
		detail  string
	}{
		"another version":             {mutate(`"version": 1`, `"version": 2`), "format version 2"},
		"a Case identity that is not": {mutate(`"caseIdentity": "2ddfcf55181d3376adf119033c77e91a8a2c86a8459258fcfbfc8385d18ddf38"`, `"caseIdentity": "zero"`), "caseIdentity"},
		"a missing Profile":           {mutate(`"caseProfile": "production-canary"`, `"caseProfile": ""`), "caseProfile is missing"},
		"an unknown authority class":  {mutate(`"authorityClass": "protected-workflow"`, `"authorityClass": "anyone"`), "authorityClass"},
		"a raw coordinate":            {mutate(`"namespace": "unconfigured"`, `"namespace": "canary-prod"`), "coordinate namespace"},
		"a repository without owner":  {mutate(`"repository": "temporalio/temporal"`, `"repository": "temporal"`), "owner/name"},
		"a tag as the trusted ref":    {mutate(`"trustedRef": "refs/heads/main"`, `"trustedRef": "refs/tags/v1"`), "branch ref"},
		"another workflow directory":  {mutate(`"workflowPath": ".github/workflows/umpire-production-canary.yml"`, `"workflowPath": "scripts/run.yml"`), "workflow file"},
		"no iterations":               {mutate(`"iterations": 2`, `"iterations": 0`), "limit iterations"},
		"a negative reserve":          {mutate(`"cleanupReserveSeconds": 120`, `"cleanupReserveSeconds": -1`), "limit cleanupReserveSeconds"},
		"a lease that expires early":  {mutate(`"leaseRunTimeoutSeconds": 86400`, `"leaseRunTimeoutSeconds": 720`), "must exceed"},
		"an overflowing invocation":   {mutate(`"invocationSeconds": 600`, `"invocationSeconds": 9223372036854775807`), "at most"},
		"too many iterations":         {mutate(`"iterations": 2`, `"iterations": 17`), "at most"},
		"an unknown key":              {mutate(`"version": 1,`, `"version": 1, "override": true,`), "unknown field"},
		"a case-folded key":           {mutate(`"trustedRef"`, `"TrustedRef"`), "canonical form"},
		"a repeated key":              {mutate(`"repository": "temporalio/temporal",`, `"repository": "evil/fork", "repository": "temporalio/temporal",`), "canonical form"},
		"other spacing":               {mutate(`"version": 1`, `"version":1`), "canonical form"},
		"not JSON":                    {"{", "decode canary policy"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Decode([]byte(probe.encoded))
			require.ErrorContains(t, err, probe.detail)
		})
	}
}

func TestAuthorityClassesCannotBeWidened(t *testing.T) {
	classes := AuthorityClasses()
	classes[0] = "anyone"
	require.Equal(t, []string{AuthorityProtectedWorkflow, AuthorityHarness}, AuthorityClasses())
}
