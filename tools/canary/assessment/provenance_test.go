package assessment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// The raw coordinates a sample provenance is made from; none may appear in its bytes.
var plantedCoordinates = []string{
	"canary-frontend.example.internal:7233", "canary-namespace-7c1", "canary-queue-2b9",
	"canary-handler-queue-5e3", "canary-endpoint-8d4",
}

// sampleProvenance is the provenance of an invocation's first iteration under the committed
// policy, with the planted coordinates' digests, whose cleanup released the lease.
func sampleProvenance(t *testing.T) *Provenance {
	t.Helper()
	canary := committed(t)
	profile, err := LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	return &Provenance{
		Version:           ProvenanceFormatVersion,
		Receipt:           recordedrun.Digest([]byte("an fn-26 receipt")),
		EvaluationProfile: profile.Identity,
		AuthorityClass:    canary.AuthorityClass,
		Workflow:          ProvenanceWorkflow{Ref: canary.Repository + "/" + canary.WorkflowPath + "@" + canary.TrustedRef, RunID: "1234567"},
		Coordinates:       policy.DigestsOf(plantedCoordinates[0], plantedCoordinates[1], plantedCoordinates[2], plantedCoordinates[3], plantedCoordinates[4]),
		Lease:             ProvenanceLease{WorkflowIDDigest: policy.Digest(canary.Lease.WorkflowID), Fence: "01a0ce8e-ba30-7919-ab7e-589b62e92ced"},
		Invocation:        ProvenanceInvocation{ID: "1234567-1", Iteration: 1, RunID: "testpilot.run.2d185b3f-c34e-4572-8ea4-4a606aa3e27a"},
		Limits:            canary.Limits,
		Cleanup:           ProvenanceCleanup{Iteration: "CLEANUP_STATUS_SUCCEEDED", Invocation: CleanupReleased},
		Isolation:         Isolation,
		Fenced:            []string{"testpilot.run.2d185b3f-c34e-4572-8ea4-4a606aa3e27a", "testpilot.run.7e0c1a52-93d4-4f7b-a1d2-0b6c1f8e4a90"},
	}
}

// The goldens pin the provenance's bytes and identity. UMPIRE_PROVENANCE_GOLDENS=write rewrites
// the bytes; the identities are pinned here, so a changed rendering is reviewed twice.
func TestProvenanceGoldens(t *testing.T) {
	uncertain := sampleProvenance(t)
	uncertain.Invocation.Iteration = 2
	uncertain.Invocation.RunID = uncertain.Fenced[1]
	uncertain.Cleanup = ProvenanceCleanup{Iteration: "CLEANUP_STATUS_FAILED", Invocation: CleanupUncertain}
	for name, test := range map[string]struct {
		provenance *Provenance
		identity   string
	}{
		"released":  {sampleProvenance(t), "15235f3e8ad67c3b4cfde3df5dff716c62746f7e7dd7fd0d2ec576719551e94b"},
		"uncertain": {uncertain, "47ca0d07e501f04da9d6460655db52d1d43910f2cff007b9a69b5ac7e69e91c6"},
	} {
		t.Run(name, func(t *testing.T) {
			rendered, err := RenderProvenance(test.provenance)
			require.NoError(t, err)
			path := filepath.Join("testdata", "provenance", name+".json")
			if os.Getenv("UMPIRE_PROVENANCE_GOLDENS") == "write" {
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, rendered, 0o644))
			}
			golden, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(golden), string(rendered))
			require.Equal(t, test.identity, ProvenanceIdentity(golden))
			require.True(t, strings.HasSuffix(string(golden), "}\n") && strings.Count(string(golden), "\n") == 1, "one compact document, one trailing newline")
			require.Contains(t, string(golden), `"releaseEligibility":false`)

			decoded, err := DecodeProvenance(golden)
			require.NoError(t, err)
			require.Equal(t, test.provenance, decoded)
			for _, raw := range plantedCoordinates {
				require.NotContains(t, string(golden), raw, "a provenance carries digests, never a raw coordinate")
			}
			require.NotContains(t, string(golden), "canary-frontend.example.internal")
		})
	}
}

// Every version, identity, status, relation and key mutation of a valid provenance is refused, by
// the renderer where it can be expressed and by the decoder in any case.
func TestProvenanceRefusesEveryMutation(t *testing.T) {
	for name, edit := range map[string]func(*Provenance){
		"another version":                       func(p *Provenance) { p.Version = 2 },
		"a receipt that is no identity":         func(p *Provenance) { p.Receipt = "receipt" },
		"a Profile that is no identity":         func(p *Provenance) { p.EvaluationProfile = strings.TrimPrefix(p.EvaluationProfile, "sha256:") },
		"another authority class":               func(p *Provenance) { p.AuthorityClass = "operator-laptop" },
		"a workflow ref with no ref":            func(p *Provenance) { p.Workflow.Ref = "temporalio/temporal/.github/workflows/x.yml" },
		"a workflow run ID of zero":             func(p *Provenance) { p.Workflow.RunID = "0" },
		"a padded workflow run ID":              func(p *Provenance) { p.Workflow.RunID = "01234567" },
		"a raw coordinate":                      func(p *Provenance) { p.Coordinates.Namespace = plantedCoordinates[1] },
		"an unconfigured coordinate":            func(p *Provenance) { p.Coordinates.GRPC = policy.Unconfigured },
		"a raw lease ID":                        func(p *Provenance) { p.Lease.WorkflowIDDigest = "umpire-canary-lease" },
		"no fence":                              func(p *Provenance) { p.Lease.Fence = "" },
		"no invocation":                         func(p *Provenance) { p.Invocation.ID = "" },
		"an invocation of another workflow run": func(p *Provenance) { p.Invocation.ID = "7654321-1" },
		"an invocation with no attempt":         func(p *Provenance) { p.Invocation.ID = "1234567" },
		"an invocation of attempt zero":         func(p *Provenance) { p.Invocation.ID = "1234567-0" },
		"a fence that is no run ID":             func(p *Provenance) { p.Lease.Fence = "umpire-canary-lease" },
		"a fenced ID that is no Testpilot Run": func(p *Provenance) {
			p.Fenced[1] = "customer-workflow"
		},
		"a workflow ref outside the workflows directory": func(p *Provenance) {
			p.Workflow.Ref = "temporalio/temporal/scripts/canary.yml@refs/heads/main"
		},
		"iteration zero":              func(p *Provenance) { p.Invocation.Iteration = 0 },
		"an iteration past the limit": func(p *Provenance) { p.Invocation.Iteration = p.Limits.Iterations + 1 },
		"a Run the lease did not fence": func(p *Provenance) {
			p.Invocation.RunID = "testpilot.run.9b8f3c2e-5d1a-4e6f-8a7b-0c9d2e1f3a4b"
		},
		"a fenced ID with no UUID":            func(p *Provenance) { p.Fenced[1] = "testpilot.run.canary-namespace-7c1" },
		"a fenced ID with an upper-case UUID": func(p *Provenance) { p.Fenced[1] = strings.ToUpper(p.Fenced[1]) },
		"a non-canonical fence":               func(p *Provenance) { p.Lease.Fence = "urn:uuid:" + p.Lease.Fence },
		"an undashed fence":                   func(p *Provenance) { p.Lease.Fence = strings.ReplaceAll(p.Lease.Fence, "-", "") },
		"a repeated fenced ID":                func(p *Provenance) { p.Fenced[1] = p.Fenced[0] },
		"more fenced IDs than iterations": func(p *Provenance) {
			p.Fenced = append(p.Fenced, "testpilot.run.third")
		},
		"no fenced IDs":        func(p *Provenance) { p.Fenced = nil },
		"a non-positive limit": func(p *Provenance) { p.Limits.ProgressBytes = 0 },
		"iterations past the policy ceiling": func(p *Provenance) {
			p.Limits.Iterations = 17
		},
		"a lease timeout no longer than the invocation and reserve": func(p *Provenance) {
			p.Limits.LeaseRunTimeoutSeconds = p.Limits.InvocationSeconds + p.Limits.CleanupReserveSeconds
		},
		"an iteration naming another fenced Run": func(p *Provenance) { p.Invocation.Iteration = 2 },
		"the fenced Runs out of order":           func(p *Provenance) { p.Fenced[0], p.Fenced[1] = p.Fenced[1], p.Fenced[0] },
		"an iteration past the fenced Runs": func(p *Provenance) {
			p.Invocation.Iteration = 2
			p.Invocation.RunID = p.Fenced[1]
			p.Fenced = p.Fenced[:1]
		},
		"a workflow ref on a tag": func(p *Provenance) {
			p.Workflow.Ref = strings.Replace(p.Workflow.Ref, "refs/heads/main", "refs/tags/v1", 1)
		},
		"an unspecified iteration cleanup": func(p *Provenance) { p.Cleanup.Iteration = "CLEANUP_STATUS_UNSPECIFIED" },
		"an unknown iteration cleanup":     func(p *Provenance) { p.Cleanup.Iteration = "cleaned" },
		"an unknown invocation cleanup":    func(p *Provenance) { p.Cleanup.Invocation = "released-probably" },
		"another isolation statement":      func(p *Provenance) { p.Isolation = "isolated" },
	} {
		t.Run(name, func(t *testing.T) {
			valid, err := RenderProvenance(sampleProvenance(t))
			require.NoError(t, err)
			provenance := sampleProvenance(t)
			edit(provenance)
			_, err = RenderProvenance(provenance)
			require.Error(t, err, "the renderer refuses it")

			// The same mutation written by hand, past the renderer, is refused by the decoder.
			written, err := encodeProvenance(provenance)
			require.NoError(t, err)
			require.NotEqual(t, valid, written)
			_, err = DecodeProvenance(written)
			require.Error(t, err, "the decoder refuses it")
		})
	}
}

// The decoder reads only the canonical rendering: no other key, spelling, spacing, document or
// release eligibility.
func TestDecodeProvenanceReadsOnlyTheCanonicalRendering(t *testing.T) {
	rendered, err := RenderProvenance(sampleProvenance(t))
	require.NoError(t, err)
	text := string(rendered)
	for name, encoded := range map[string]string{
		"an unknown key":              strings.Replace(text, `{"version":1,`, `{"version":1,"extra":1,`, 1),
		"a repeated key":              strings.Replace(text, `{"version":1,`, `{"version":1,"version":1,`, 1),
		"a case-folded key":           strings.Replace(text, `"receipt":`, `"Receipt":`, 1),
		"another key order":           strings.Replace(strings.Replace(text, `"version":1,`, ``, 1), `"isolation":`, `"version":1,"isolation":`, 1),
		"indented":                    strings.ReplaceAll(text, `,"`, `, "`),
		"no trailing newline":         strings.TrimSuffix(text, "\n"),
		"a trailing document":         text + text,
		"release eligible":            strings.Replace(text, `"releaseEligibility":false`, `"releaseEligibility":true`, 1),
		"release eligibility as text": strings.Replace(text, `"releaseEligibility":false`, `"releaseEligibility":"false"`, 1),
		"no release eligibility":      strings.Replace(text, `,"releaseEligibility":false`, ``, 1),
		"null fenced IDs":             text[:strings.Index(text, `"fenced":[`)] + `"fenced":null` + text[strings.Index(text, `],"releaseEligibility"`)+1:],
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, text, encoded)
			_, err := DecodeProvenance([]byte(encoded))
			require.Error(t, err)
		})
	}
}

// A provenance exactly at the cap renders and decodes; one byte over is refused by both.
func TestProvenanceCapAtNAndNPlusOne(t *testing.T) {
	sized := func(t *testing.T, size int) *Provenance {
		t.Helper()
		provenance := sampleProvenance(t)
		base, err := encodeProvenance(provenance)
		require.NoError(t, err)
		// The workflow file's name is the one field with no bound of its own.
		provenance.Workflow.Ref = strings.Replace(provenance.Workflow.Ref, ".yml@", strings.Repeat("x", size-len(base))+".yml@", 1)
		return provenance
	}
	atCap := sized(t, MaxProvenanceBytes)
	rendered, err := RenderProvenance(atCap)
	require.NoError(t, err)
	require.Len(t, rendered, MaxProvenanceBytes)
	_, err = DecodeProvenance(rendered)
	require.NoError(t, err)

	over := sized(t, MaxProvenanceBytes+1)
	_, err = RenderProvenance(over)
	require.ErrorContains(t, err, "over the")
	written, err := encodeProvenance(over)
	require.NoError(t, err)
	require.Len(t, written, MaxProvenanceBytes+1)
	_, err = DecodeProvenance(written)
	require.ErrorContains(t, err, "over the")
}
