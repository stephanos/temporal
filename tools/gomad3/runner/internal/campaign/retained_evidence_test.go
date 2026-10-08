package campaign

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
)

func TestOpenCampaignKeepsSameSignatureSuccessesDistinct(t *testing.T) {
	for name, key := range map[string]artifact.StoreKey{
		"signature with record fallback": artifact.StoreKeyFailureSignature,
		"execution":                      artifact.StoreKeyExecution,
	} {
		t.Run(name, func(t *testing.T) {
			input := campaignArtifactInput(t)
			journal, err := NewCampaignJournal(t.Context(), CampaignConfig{
				Root: t.TempDir(), CampaignID: input.Manifest.CampaignID, Selection: "7-8", SelectionCount: 2,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := journal.Close(); err != nil {
					t.Error(err)
				}
			})
			if err := journal.BeginPreparation(); err != nil {
				t.Fatal(err)
			}
			preparedPath := filepath.Join(journal.PreparedPath(), "build", "target")
			if err := os.MkdirAll(filepath.Dir(preparedPath), 0o700); err != nil {
				t.Fatal(err)
			}
			targetBytes, err := os.ReadFile(input.TargetPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(preparedPath, targetBytes, 0o500); err != nil {
				t.Fatal(err)
			}
			plan := testBatchPlan(journal, input.Manifest.Target.SHA256, uint64(len(targetBytes)))
			plan.Selection, plan.SelectionCount = "7-8", 2
			plan.KeepSuccesses, plan.SuccessArtifactLimit, plan.SuccessBytesLimit = "all", 2, 1<<20
			capacity, err := DeriveArtifactCapacityPlan(plan)
			if err != nil {
				t.Fatal(err)
			}
			plan.Artifacts = &capacity
			if err := journal.RecordPlan(plan); err != nil {
				t.Fatal(err)
			}
			for _, step := range []func() error{journal.CompletePreparation, journal.StartExecutions} {
				if err := step(); err != nil {
					t.Fatal(err)
				}
			}
			zero := record.Uint64String(0)
			input.Manifest.ArtifactKind = record.ArtifactSuccess
			input.Manifest.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &zero}
			store := artifact.Store{Root: journal.SuccessesPath(), Context: t.Context(), Key: key}
			var retained []artifact.Artifact
			var storedBytes uint64
			for ordinal, seed := range []uint64{7, 8} {
				run, err := journal.BeginExecution(uint64(ordinal), seed)
				if err != nil {
					t.Fatal(err)
				}
				for _, state := range []ExecutionState{ExecutionStarting, ExecutionExited, ExecutionCaptured, ExecutionClassified} {
					if err := run.Transition(state); err != nil {
						t.Fatal(err)
					}
				}
				input.Manifest.Seed, input.Manifest.SelectionOrdinal = record.Uint64String(seed), record.Uint64String(ordinal)
				input.Manifest.Environment[1].Value = fmt.Sprint(seed)
				published, err := artifact.PublishArtifact(store, input)
				if err != nil {
					t.Fatal(err)
				}
				reference, err := filepath.Rel(journal.Path(), published.Path)
				if err != nil {
					t.Fatal(err)
				}
				reference = filepath.ToSlash(reference)
				bytes := record.Uint64String(published.StoredBytes)
				if err := journal.AppendExecution(ExecutionRecord{
					SelectionOrdinal: record.Uint64String(ordinal), Seed: record.Uint64String(seed), Domain: "success", Reason: "success", Termination: "exit",
					SuccessArtifact: &reference, SuccessArtifactBytes: &bytes,
				}); err != nil {
					t.Fatal(err)
				}
				if err := run.Complete(); err != nil {
					t.Fatal(err)
				}
				retained = append(retained, published)
				storedBytes += published.StoredBytes
			}
			if err := journal.Publish(CampaignSummary{Attempted: 2, Succeeded: 2, RetainedSuccesses: 2, RetainedSuccessBytes: storedBytes, StopReason: "seeds_exhausted"}); err != nil {
				t.Fatal(err)
			}
			opened, err := OpenCampaign(journal.Path())
			if err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(journal.SuccessesPath())
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 2 || len(opened.Executions) != 2 || opened.Record.RetainedSuccesses != 2 || uint64(opened.Record.RetainedSuccessBytes) != storedBytes {
				t.Fatalf("stored=%d, executions=%#v, campaign=%#v", len(entries), opened.Executions, opened.Record)
			}
			if retained[0].Path == retained[1].Path || retained[0].Manifest.RecordHash == retained[1].Manifest.RecordHash || retained[0].Manifest.Outcome.FailureSignature != retained[1].Manifest.Outcome.FailureSignature {
				t.Fatalf("same-signature success artifacts collapsed: %#v", retained)
			}
			if filepath.Base(retained[0].Path) != "sha256-"+strings.TrimPrefix(string(retained[0].Manifest.Outcome.FailureSignature), "sha256:")[:32] {
				t.Fatalf("first success lost its signature directory: %s", retained[0].Path)
			}
			for index, run := range opened.Executions {
				evidence, err := ResolveRetainedEvidence(journal.Path(), opened.Record.CampaignID, run)
				if err != nil {
					t.Fatal(err)
				}
				if evidence.Manifest.Seed != record.Uint64String(index+7) || evidence.Manifest.SelectionOrdinal != record.Uint64String(index) || evidence.Path != retained[index].Path {
					t.Fatalf("execution %d retained %#v", index, evidence)
				}
			}
			for name, mutate := range map[string]func(*ExecutionRecord){
				"seed":    func(run *ExecutionRecord) { run.Seed++ },
				"ordinal": func(run *ExecutionRecord) { run.SelectionOrdinal++ },
				"reference": func(run *ExecutionRecord) {
					run.SuccessArtifact = opened.Executions[1].SuccessArtifact
					run.SuccessArtifactBytes = opened.Executions[1].SuccessArtifactBytes
				},
			} {
				t.Run(name, func(t *testing.T) {
					run := opened.Executions[0]
					mutate(&run)
					if _, err := ResolveRetainedEvidence(journal.Path(), opened.Record.CampaignID, run); err == nil {
						t.Fatalf("accepted mismatched success %s", name)
					}
				})
			}
		})
	}
}

func TestResolveRetainedEvidenceAllowsSharedFailureIdentityWithinBatch(t *testing.T) {
	root := t.TempDir()
	published, err := artifact.PublishArtifact(artifact.Store{Root: filepath.Join(root, "failures")}, campaignArtifactInput(t))
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
	input := campaignArtifactInput(t)
	exitCode := record.Uint64String(0)
	input.Manifest.ArtifactKind = record.ArtifactSuccess
	input.Manifest.Outcome = record.Outcome{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode}
	published, err := artifact.PublishArtifact(artifact.Store{Root: filepath.Join(root, "successes"), Key: artifact.StoreKeyRecord}, input)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := filepath.Rel(root, published.Path)
	if err != nil {
		t.Fatal(err)
	}
	storedBytes := record.Uint64String(published.StoredBytes)
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
	traceSHA256 := record.HashBytes([]byte("choice trace"))
	tapeSHA256 := record.HashBytes([]byte("choice tape"))
	records := record.Uint64String(4)
	branching := record.Uint64String(2)
	decisions := record.Uint64String(3)
	terminal := "complete"
	run := ExecutionRecord{
		ChoiceTraceSHA256: &traceSHA256, ChoiceTraceRecords: &records, ChoiceTraceBranchingRecords: &branching,
		ChoiceTraceTerminalState: &terminal, ChoiceTapeSHA256: &tapeSHA256, ChoiceDecisions: &decisions,
	}
	manifest := record.ExecutionRecord{ChoiceProfile: &record.ChoiceProfile{Trace: record.ChoiceTrace{
		SHA256: traceSHA256, Records: records, BranchingRecords: branching, TerminalState: terminal,
		TapeSHA256: tapeSHA256, Decisions: decisions,
	}}}
	if !retainedChoiceMatches(run, manifest) {
		t.Fatal("matching choice tape identity was rejected")
	}
	changedTape := record.HashBytes([]byte("changed tape"))
	run.ChoiceTapeSHA256 = &changedTape
	if retainedChoiceMatches(run, manifest) {
		t.Fatal("changed choice tape identity was accepted")
	}
}
