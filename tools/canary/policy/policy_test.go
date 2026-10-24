package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"umpire.local/temporal"
)

func TestTheCommittedPolicyDecodesUnconfigured(t *testing.T) {
	committed, err := Embedded()
	require.NoError(t, err)
	require.False(t, committed.Configured(), "the operator commits the coordinate digests")
	require.Equal(t, AuthorityProtectedWorkflow, committed.AuthorityClass)
	require.Equal(t, "refs/heads/main", committed.TrustedRef)
	require.Equal(t, 2, committed.Limits.Iterations)
	require.Greater(t, committed.Limits.LeaseRunTimeout(), committed.Limits.Invocation()+committed.Limits.CleanupReserve())
}

// The committed policy pins the committed guest, so a guest rebuild fails here until the reviewed
// policy update lands with it; otherwise every production dispatch would refuse as model-mismatch.
func TestTheCommittedPolicyPinsTheCommittedGuest(t *testing.T) {
	committed, err := Embedded()
	require.NoError(t, err)
	hash := sha256.Sum256(temporal.Guest())
	require.Equal(t, hex.EncodeToString(hash[:]), committed.ModelHash,
		"update modelHash in production-canary.json to the rebuilt guest's SHA-256")
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
		"another version":              {mutate(`"version": 2`, `"version": 1`), "format version 1"},
		"a model hash that is not":     {mutate(`"modelHash": "`+decoded.ModelHash+`"`, `"modelHash": "zero"`), "modelHash"},
		"a missing lease":              {mutate(`"workflowId": "umpire-canary-lease"`, `"workflowId": ""`), "lease.workflowId is missing"},
		"an unknown authority class":   {mutate(`"authorityClass": "protected-workflow"`, `"authorityClass": "anyone"`), "authorityClass"},
		"a raw coordinate":             {mutate(`"namespace": "unconfigured"`, `"namespace": "canary-prod"`), "coordinate namespace"},
		"a repository without owner":   {mutate(`"repository": "temporalio/temporal"`, `"repository": "temporal"`), "owner/name"},
		"a tag as the trusted ref":     {mutate(`"trustedRef": "refs/heads/main"`, `"trustedRef": "refs/tags/v1"`), "branch ref"},
		"another workflow directory":   {mutate(`"workflowPath": ".github/workflows/umpire-production-canary.yml"`, `"workflowPath": "scripts/run.yml"`), "workflow file"},
		"a workflow subdirectory":      {mutate(`"workflowPath": ".github/workflows/umpire-production-canary.yml"`, `"workflowPath": ".github/workflows/sub/run.yml"`), "workflow file"},
		"a workflow file with no name": {mutate(`"workflowPath": ".github/workflows/umpire-production-canary.yml"`, `"workflowPath": ".github/workflows/.yml"`), "workflow file"},
		"no iterations":                {mutate(`"iterations": 2`, `"iterations": 0`), "limit iterations"},
		"a negative reserve":           {mutate(`"cleanupReserveSeconds": 120`, `"cleanupReserveSeconds": -1`), "limit cleanupReserveSeconds"},
		"a lease that expires early":   {mutate(`"leaseRunTimeoutSeconds": 86400`, `"leaseRunTimeoutSeconds": 720`), "must exceed"},
		"an overflowing invocation":    {mutate(`"invocationSeconds": 600`, `"invocationSeconds": 9223372036854775807`), "at most"},
		"too many iterations":          {mutate(`"iterations": 2`, `"iterations": 17`), "at most"},
		"an unknown key":               {mutate(`"version": 2,`, `"version": 2, "override": true,`), "unknown field"},
		"a retired Nexus coordinate":   {mutate(`"taskQueue": "unconfigured"`, `"taskQueue": "unconfigured", "nexusEndpoint": "unconfigured"`), "unknown field"},
		"a case-folded key":            {mutate(`"trustedRef"`, `"TrustedRef"`), "canonical form"},
		"a repeated key":               {mutate(`"repository": "temporalio/temporal",`, `"repository": "evil/fork", "repository": "temporalio/temporal",`), "canonical form"},
		"other spacing":                {mutate(`"version": 2`, `"version":2`), "canonical form"},
		"not JSON":                     {"{", "decode canary policy"},
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

func TestCoordinatesMismatchNamesTheFirstDifference(t *testing.T) {
	ours := DigestsOf("grpc:7233", "namespace", "queue")
	require.Equal(t, Digest("namespace"), ours.Namespace)
	_, differs := ours.Mismatch(ours)
	require.False(t, differs)
	for name, theirs := range map[string]Coordinates{
		"grpc":      DigestsOf("other:7233", "namespace", "queue"),
		"namespace": DigestsOf("grpc:7233", "other", "queue"),
		"taskQueue": DigestsOf("grpc:7233", "namespace", "other"),
	} {
		named, differs := ours.Mismatch(theirs)
		require.True(t, differs)
		require.Equal(t, name, named)
	}
}
