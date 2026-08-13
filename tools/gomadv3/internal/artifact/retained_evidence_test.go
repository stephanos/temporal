package artifact

import (
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

func TestResolveRetainedEvidenceValidatesRunIdentity(t *testing.T) {
	root := t.TempDir()
	published, err := (Store{Root: filepath.Join(root, "failures")}).Publish(artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(root, published.Path)
	if err != nil {
		t.Fatal(err)
	}
	signature := published.Manifest.Outcome.FailureSignature
	run := RunRecord{
		SelectionOrdinal: published.Manifest.SelectionOrdinal,
		Seed:             published.Manifest.Seed,
		Domain:           "target",
		FailureSignature: &signature,
		Artifact:         &reference,
	}
	evidence, err := ResolveRetainedEvidence(root, published.Manifest.BatchID, run)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Path != published.Path || evidence.Manifest.RecordHash != published.Manifest.RecordHash || evidence.StoredBytes != published.StoredBytes {
		t.Fatalf("retained evidence = %#v", evidence)
	}
	run.Seed++
	if _, err := ResolveRetainedEvidence(root, published.Manifest.BatchID, run); err == nil {
		t.Fatal("ResolveRetainedEvidence() accepted a mismatched seed")
	}
}

func TestResolveRetainedEvidenceValidatesSuccessBytes(t *testing.T) {
	root := t.TempDir()
	input := artifactInput(t)
	exitCode := record.Uint64String(0)
	input.Manifest.ArtifactKind = record.ArtifactSuccess
	input.Manifest.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode}
	published, err := (Store{Root: filepath.Join(root, "successes"), Key: StoreKeyRecord}).Publish(input)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(root, published.Path)
	if err != nil {
		t.Fatal(err)
	}
	storedBytes := record.Uint64String(published.StoredBytes)
	run := RunRecord{
		SelectionOrdinal:     published.Manifest.SelectionOrdinal,
		Seed:                 published.Manifest.Seed,
		Domain:               "success",
		SuccessArtifact:      &reference,
		SuccessArtifactBytes: &storedBytes,
	}
	if _, err := ResolveRetainedEvidence(root, published.Manifest.BatchID, run); err != nil {
		t.Fatal(err)
	}
	storedBytes++
	if _, err := ResolveRetainedEvidence(root, published.Manifest.BatchID, run); err == nil {
		t.Fatal("ResolveRetainedEvidence() accepted mismatched stored bytes")
	}
}
