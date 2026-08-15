package campaignstore

import (
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestResolveRetainedEvidenceAllowsSharedFailureIdentityWithinBatch(t *testing.T) {
	root := t.TempDir()
	published, err := PublishArtifact(evidence.Store{Root: filepath.Join(root, "failures")}, artifactInput(t))
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(root, published.Path)
	if err != nil {
		t.Fatal(err)
	}
	signature := published.Manifest.Outcome.FailureSignature
	run := ExecutionRecord{
		SelectionOrdinal: published.Manifest.SelectionOrdinal,
		Seed:             published.Manifest.Seed,
		Domain:           published.Manifest.Outcome.Domain,
		Reason:           published.Manifest.Outcome.Reason,
		Termination:      published.Manifest.Outcome.Termination,
		FailureSignature: &signature,
		Artifact:         &reference,
	}
	evidence, err := ResolveRetainedEvidence(root, published.Manifest.CampaignID, run)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Path != published.Path || evidence.Manifest.RecordHash != published.Manifest.RecordHash || evidence.StoredBytes != published.StoredBytes {
		t.Fatalf("retained evidence = %#v", evidence)
	}
	run.SelectionOrdinal++
	run.Seed++
	if _, err := ResolveRetainedEvidence(root, published.Manifest.CampaignID, run); err != nil {
		t.Fatalf("ResolveRetainedEvidence() rejected a shared failure: %v", err)
	}
	if _, err := ResolveRetainedEvidence(root, "different-batch", run); err == nil {
		t.Fatal("ResolveRetainedEvidence() accepted a mismatched batch")
	}
	for name, mutate := range map[string]func(*ExecutionRecord){
		"domain":      func(run *ExecutionRecord) { run.Domain = "watchdog" },
		"reason":      func(run *ExecutionRecord) { run.Reason = "signal" },
		"termination": func(run *ExecutionRecord) { run.Termination = "signal" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := run
			mutate(&changed)
			if _, err := ResolveRetainedEvidence(root, published.Manifest.CampaignID, changed); err == nil {
				t.Fatalf("ResolveRetainedEvidence() accepted changed %s", name)
			}
		})
	}
}

func TestResolveRetainedEvidenceValidatesSuccessBytes(t *testing.T) {
	root := t.TempDir()
	input := artifactInput(t)
	exitCode := evidence.Uint64String(0)
	input.Manifest.ArtifactKind = evidence.ArtifactSuccess
	input.Manifest.Outcome = evidence.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode}
	published, err := PublishArtifact(evidence.Store{Root: filepath.Join(root, "successes"), Key: evidence.StoreKeyRecord}, input)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(root, published.Path)
	if err != nil {
		t.Fatal(err)
	}
	storedBytes := evidence.Uint64String(published.StoredBytes)
	run := ExecutionRecord{
		SelectionOrdinal:     published.Manifest.SelectionOrdinal,
		Seed:                 published.Manifest.Seed,
		Domain:               "success",
		SuccessArtifact:      &reference,
		SuccessArtifactBytes: &storedBytes,
	}
	if _, err := ResolveRetainedEvidence(root, published.Manifest.CampaignID, run); err != nil {
		t.Fatal(err)
	}
	storedBytes++
	if _, err := ResolveRetainedEvidence(root, published.Manifest.CampaignID, run); err == nil {
		t.Fatal("ResolveRetainedEvidence() accepted mismatched stored bytes")
	}
}

func TestRetainedChoiceSummaryBindsDerivedTapeIdentity(t *testing.T) {
	traceSHA256 := evidence.HashBytes([]byte("choice trace"))
	tapeSHA256 := evidence.HashBytes([]byte("choice tape"))
	records := evidence.Uint64String(4)
	branching := evidence.Uint64String(2)
	decisions := evidence.Uint64String(3)
	terminal := "complete"
	run := ExecutionRecord{
		ChoiceTraceSHA256: &traceSHA256, ChoiceTraceRecords: &records, ChoiceTraceBranchingRecords: &branching,
		ChoiceTraceTerminalState: &terminal, ChoiceTapeSHA256: &tapeSHA256, ChoiceDecisions: &decisions,
	}
	manifest := evidence.ExecutionRecord{ChoiceProfile: &evidence.ChoiceProfile{Trace: evidence.ChoiceTrace{
		SHA256: traceSHA256, Records: records, BranchingRecords: branching, TerminalState: terminal,
		TapeSHA256: tapeSHA256, Decisions: decisions,
	}}}
	if !retainedChoiceMatches(run, manifest) {
		t.Fatal("matching choice tape identity was rejected")
	}
	changedTape := evidence.HashBytes([]byte("changed tape"))
	run.ChoiceTapeSHA256 = &changedTape
	if retainedChoiceMatches(run, manifest) {
		t.Fatal("changed choice tape identity was accepted")
	}
}
