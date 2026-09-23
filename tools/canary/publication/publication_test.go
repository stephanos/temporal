package publication

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/publish"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

const fence = "01a0ce8e-ba30-7919-ab7e-589b62e92ced"

// item makes one iteration's receipt and provenance from the test cluster's recorded canary Run,
// its subject edited to reach the wanted decision, as the n-th iteration of an invocation that
// fenced the given Runs.
func item(t *testing.T, iteration int, runID string, fenced []string, edit func(*evaluation.Subject)) Item {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("..", "assessment", "testdata", "nexusCallerCanary-syncCompletion-run.json"))
	require.NoError(t, err)
	decoded, err := recordedrun.Decode(encoded)
	require.NoError(t, err)
	canary, err := policy.Embedded()
	require.NoError(t, err)
	decoded.Run.RunId = runID
	subject, err := assessment.Admit(canary, decoded.Driver, decoded.Run)
	require.NoError(t, err)
	if edit != nil {
		edit(subject)
	}
	profile, err := assessment.LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	decision := evaluation.Assess(subject, *profile)
	receipt, err := evaluation.Render(subject, *profile, decision)
	require.NoError(t, err)
	provenance, err := assessment.RenderProvenance(&assessment.Provenance{
		Version: assessment.ProvenanceFormatVersion, Receipt: evaluation.ReceiptIdentity(receipt),
		EvaluationProfile: profile.Identity, AuthorityClass: canary.AuthorityClass,
		Workflow:    assessment.ProvenanceWorkflow{Ref: canary.Repository + "/" + canary.WorkflowPath + "@" + canary.TrustedRef, RunID: "1234567"},
		Coordinates: policy.DigestsOf("grpc:7233", "namespace", "queue", "handler", "endpoint"),
		Lease:       assessment.ProvenanceLease{WorkflowIDDigest: policy.Digest(canary.Lease.WorkflowID), Fence: fence},
		Invocation:  assessment.ProvenanceInvocation{ID: "1234567-1", Iteration: iteration, RunID: runID},
		Limits:      canary.Limits,
		Cleanup:     assessment.ProvenanceCleanup{Iteration: "CLEANUP_STATUS_SUCCEEDED", Invocation: assessment.CleanupReleased},
		Isolation:   assessment.Isolation, Fenced: fenced,
	})
	require.NoError(t, err)
	return Item{RunID: runID, Status: decision.Outcome, Receipt: receipt, Provenance: provenance}
}

var runs = []string{"testpilot.run.2d185b3f-c34e-4572-8ea4-4a606aa3e27a", "testpilot.run.7e0c1a52-93d4-4f7b-a1d2-0b6c1f8e4a90"}

func listed(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	return names
}

// Accepted, rejected and incomplete receipts each publish unchanged under their identity, followed
// by their provenance, and are recorded in order; an unconstructible iteration publishes nothing.
func TestPublishWritesEachDecisionWithItsProvenance(t *testing.T) {
	for name, edit := range map[string]func(*evaluation.Subject){
		evaluation.DecisionAccepted:   nil,
		evaluation.DecisionRejected:   func(s *evaluation.Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED },
		evaluation.DecisionIncomplete: func(s *evaluation.Subject) { s.Cleanup = testpilotspb.CLEANUP_STATUS_FAILED },
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			first := item(t, 1, runs[0], runs[:1], edit)
			require.Equal(t, name, first.Status)
			var recorded []string
			published, err := Publish(t.Context(), root, []Item{first, {RunID: runs[1], Status: StatusUnconstructible}}, func(runID string) error {
				recorded = append(recorded, runID)
				return nil
			})
			require.NoError(t, err)
			require.Len(t, published, 1)
			require.Equal(t, []string{runs[0]}, recorded, "only a published iteration is recorded")
			require.Equal(t, publish.StatusPublished, published[0].Receipt.Status)
			require.Equal(t, publish.StatusPublished, published[0].Provenance.Status)

			receiptName := evaluation.ReceiptIdentity(first.Receipt) + ".json"
			provenanceName := assessment.ProvenanceIdentity(first.Provenance) + ProvenanceSuffix
			require.Equal(t, []string{provenanceName, receiptName}, listed(t, root))
			written, err := os.ReadFile(filepath.Join(root, receiptName))
			require.NoError(t, err)
			require.Equal(t, first.Receipt, written, "the fn-26 receipt is published unchanged")
			written, err = os.ReadFile(filepath.Join(root, provenanceName))
			require.NoError(t, err)
			require.Equal(t, first.Provenance, written)
		})
	}
}

// Publishing the same documents again finds them already published and writes nothing new.
func TestPublishAgainIsAlreadyPublished(t *testing.T) {
	root := t.TempDir()
	items := []Item{item(t, 1, runs[0], runs, nil), item(t, 2, runs[1], runs, nil)}
	_, err := Publish(t.Context(), root, items, nil)
	require.NoError(t, err)
	before := listed(t, root)
	again, err := Publish(t.Context(), root, items, nil)
	require.NoError(t, err)
	require.Len(t, again, 2)
	for _, published := range again {
		require.Equal(t, publish.StatusAlreadyPublished, published.Receipt.Status)
		require.Equal(t, publish.StatusAlreadyPublished, published.Provenance.Status)
	}
	require.Equal(t, before, listed(t, root))
}

// A name holding other bytes, a partial file, a symlink or a directory is a conflict found before
// anything is published: nothing is overwritten or added.
func TestPublishConflictsChangeNothing(t *testing.T) {
	items := []Item{item(t, 1, runs[0], runs, nil), item(t, 2, runs[1], runs, nil)}
	secondReceipt := evaluation.ReceiptIdentity(items[1].Receipt) + ".json"
	secondProvenance := assessment.ProvenanceIdentity(items[1].Provenance) + ProvenanceSuffix
	for name, plant := range map[string]func(t *testing.T, root string){
		"other bytes": func(t *testing.T, root string) {
			require.NoError(t, os.WriteFile(filepath.Join(root, secondReceipt), []byte("other\n"), 0o644))
		},
		"a partial file": func(t *testing.T, root string) {
			require.NoError(t, os.WriteFile(filepath.Join(root, secondProvenance), items[1].Provenance[:10], 0o644))
		},
		"a symlink to the same bytes": func(t *testing.T, root string) {
			target := filepath.Join(t.TempDir(), "target")
			require.NoError(t, os.WriteFile(target, items[1].Receipt, 0o644))
			require.NoError(t, os.Symlink(target, filepath.Join(root, secondReceipt)))
		},
		"a directory": func(t *testing.T, root string) {
			require.NoError(t, os.Mkdir(filepath.Join(root, secondProvenance), 0o755))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			plant(t, root)
			before := listed(t, root)
			published, err := Publish(t.Context(), root, items, nil)
			var conflict *publish.ConflictError
			require.ErrorAs(t, err, &conflict)
			require.Empty(t, published)
			require.Equal(t, before, listed(t, root), "nothing is published when any name conflicts")
		})
	}
}

// A provenance that names another receipt, Profile or Run, a receipt deciding otherwise than the
// iteration, or a document that does not decode is refused before anything is published.
func TestPublishRefusesACrossedIteration(t *testing.T) {
	good := item(t, 1, runs[0], runs, nil)
	other := item(t, 2, runs[1], runs, nil)
	rejected := item(t, 1, runs[0], runs, func(s *evaluation.Subject) { s.Verdict.Status = testpilotspb.VERDICT_STATUS_VIOLATED })
	for name, crossed := range map[string]Item{
		"another receipt's provenance":   {RunID: runs[0], Status: good.Status, Receipt: good.Receipt, Provenance: rejected.Provenance},
		"another Run's documents":        {RunID: runs[0], Status: other.Status, Receipt: other.Receipt, Provenance: other.Provenance},
		"a decision the receipt differs": {RunID: runs[0], Status: evaluation.DecisionRejected, Receipt: good.Receipt, Provenance: good.Provenance},
		"an unreadable receipt":          {RunID: runs[0], Status: good.Status, Receipt: []byte("{}\n"), Provenance: good.Provenance},
		"an unreadable provenance":       {RunID: runs[0], Status: good.Status, Receipt: good.Receipt, Provenance: []byte("{}\n")},
		"an unconstructible receipt":     {RunID: runs[0], Status: StatusUnconstructible, Receipt: good.Receipt},
		"no status":                      {RunID: runs[0]},
		"a mislabelled status":           {RunID: runs[0], Status: "acepted"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			published, err := Publish(t.Context(), root, []Item{other, crossed}, nil)
			require.Error(t, err)
			require.Empty(t, published)
			require.Empty(t, listed(t, root), "a crossed iteration stops the publication before the first document")
		})
	}
}

// A publication that succeeded but could not be recorded is the named ambiguity: its documents
// stand, and the next iteration is not published because of it.
func TestAnUnrecordedPublicationIsNamedAndNothingReruns(t *testing.T) {
	root := t.TempDir()
	items := []Item{item(t, 1, runs[0], runs, nil), item(t, 2, runs[1], runs, nil)}
	lost := errors.New("the recovery record could not be written")
	calls := 0
	published, err := Publish(t.Context(), root, items, func(string) error {
		calls++
		return lost
	})
	var unreported *UnreportedError
	require.ErrorAs(t, err, &unreported)
	require.ErrorIs(t, err, lost)
	require.Equal(t, runs[0], unreported.RunID)
	require.Contains(t, err.Error(), "publication-unreported")
	require.Len(t, published, 1)
	require.Equal(t, 1, calls)
	require.Len(t, listed(t, root), 2, "the first iteration's two documents stand; the second is not published")
}

// A cancelled context publishes nothing.
func TestAnInterruptedPublicationPublishesNothing(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Publish(ctx, root, []Item{item(t, 1, runs[0], runs, nil)}, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, listed(t, root))
}
