package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner"
)

func readInspectFixtureLog(t *testing.T, path string) []byte {
	t.Helper()
	log, err := os.ReadFile(filepath.Join(path, "store", "log"))
	if err != nil {
		t.Fatal(err)
	}
	return log
}

// A recognized store whose replay fails is diagnosed on stderr alone, as text
// or as one JSON report, with status 2; a diagnosis that cannot be written
// returns status 3. Nothing reaches stdout and the log is left as it was.
func TestRunInspectDiagnosesAnInvalidStoreOnStderrOnly(t *testing.T) {
	path := writeInspectBatchFixture(t)
	log := readInspectFixtureLog(t, path)
	envelope := int64(16 + 4 + len(`{"format":"gomad3.campaign-log/v1","campaign_id":"run-inspect-command","strategy":"seed","selection":"7","selection_count":"1"}`))
	log[envelope+20] ^= 0xff
	writeInspectLog(t, filepath.Join(path, "store"), log)

	var stdout, stderr bytes.Buffer
	if status := runInspect([]string{path}, &stdout, &stderr); status != 2 || stdout.Len() != 0 {
		t.Fatalf("text status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	text := stderr.String()
	for _, want := range []string{"gomad inspect: kind=campaign path=" + path + "\n", "log: format=gomad3.campaign-log/v1 frame-version=1 state=invalid records=1 valid-bytes=" + itoa(envelope), "invalid-store: problem=corrupt offset=" + itoa(envelope) + " records=1 reason=payload checksum mismatch before the final frame\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text diagnosis = %q, missing %q", text, want)
		}
	}
	if strings.Contains(text, "inspect "+path+":") || strings.Contains(text, "lifecycle:") || strings.Contains(text, "campaign:") {
		t.Fatalf("text diagnosis repeats the error or reports state: %q", text)
	}

	stdout.Reset()
	stderr.Reset()
	if status := runInspect([]string{"--json", path}, &stdout, &stderr); status != 2 || stdout.Len() != 0 {
		t.Fatalf("json status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if lines := strings.Count(stderr.String(), "\n"); lines != 1 {
		t.Fatalf("json diagnosis has %d lines: %q", lines, stderr.String())
	}
	var report runner.Inspection
	decoder := json.NewDecoder(&stderr)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != "gomad3.inspect/v8" || report.Kind != "campaign" || report.Log == nil || report.Log.State != "invalid" || report.Log.Invalid == nil ||
		report.Log.Invalid.Offset != uint64(envelope) || report.Log.Invalid.Records != 1 || report.Campaign != nil || report.Lifecycle != nil || report.Committed != nil {
		t.Fatalf("json diagnosis = %#v %#v", report, report.Log)
	}

	for _, arguments := range [][]string{{path}, {"--json", path}} {
		stdout.Reset()
		if status := runInspect(arguments, &stdout, failingWriter{}); status != 3 || stdout.Len() != 0 {
			t.Fatalf("%v undeliverable diagnosis status=%d stdout=%q", arguments, status, stdout.String())
		}
	}
	if after := readInspectFixtureLog(t, path); !bytes.Equal(after, log) {
		t.Fatal("inspect changed the damaged log")
	}
}

// An observed torn final record is a successful report on stdout that leaves
// the torn bytes for the next writer to repair.
func TestRunInspectReportsATornTailWithoutRepairingIt(t *testing.T) {
	path := writeInspectBatchFixture(t)
	clean := readInspectFixtureLog(t, path)
	torn := append(bytes.Clone(clean), inspectFrame(2, []byte(`{"state":"running"}`))[:11]...)
	writeInspectLog(t, filepath.Join(path, "store"), torn)
	for _, test := range []struct {
		arguments []string
		want      []string
	}{
		{arguments: []string{path}, want: []string{"state=torn-tail records=5 valid-bytes=" + itoa(int64(len(clean))) + " snapshot-bytes=" + itoa(int64(len(torn))) + " tail=short-frame tail-bytes=11 repaired-bytes=0\n", "lifecycle: state=published", "campaign: id=run-inspect-command"}},
		{arguments: []string{"--json", path}, want: []string{`"state":"torn-tail"`, `"tail":"short-frame","tail_bytes":11,"repaired_bytes":0`, `"campaign_id":"run-inspect-command"`}},
	} {
		var stdout, stderr bytes.Buffer
		if status := runInspect(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
			t.Fatalf("%v status=%d stdout=%q stderr=%q", test.arguments, status, stdout.String(), stderr.String())
		}
		for _, want := range test.want {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("%v output = %q, missing %q", test.arguments, stdout.String(), want)
			}
		}
	}
	if after := readInspectFixtureLog(t, path); !bytes.Equal(after, torn) {
		t.Fatal("inspect repaired the torn tail")
	}
}

// A guided corpus directory is inspected by the existing PATH dispatch: its
// log lineage and live state report in text and JSON, and --choices still
// requires a traced artifact.
func TestRunInspectReportsAGuidedCorpusDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corpus")
	store := filepath.Join(path, "store")
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	header := `{"format":"gomad3.guide-corpus-log/v1","identity":{"environment_sha256":"sha256:` + strings.Repeat("e", 64) + `","target_sha256":"sha256:` + strings.Repeat("a", 64) + `","toolchain":{"go_version":"go1.26.4","build_key":"key","target_goos":"linux","target_goarch":"arm64"},"boundary_version":"v1","boundary_sha256":"sha256:` + strings.Repeat("b", 64) + `","instrumentation_schema":"gomad3.semantic-features/v1","instrumentation_sha256":"sha256:` + strings.Repeat("c", 64) + `","manifest_schema_version":1,"manifest_record_contract":"gomad3.execution-record/v1"}}`
	writeInspectLog(t, store, inspectFrame(1, []byte(header)))
	for _, test := range []struct {
		arguments []string
		want      []string
	}{
		{arguments: []string{path}, want: []string{"gomad inspect: kind=corpus path=" + path + "\n", "log: format=gomad3.guide-corpus-log/v1 frame-version=1 state=valid records=1 ", "corpus: schema=gomad3.guide-corpus/v2 generation=0 entries=0 max-entries=1024 entry-stored-bytes=0 max-bytes=1073741824 target=sha256:" + strings.Repeat("a", 64) + " toolchain=key instrumentation=gomad3.semantic-features/v1"}},
		{arguments: []string{"--json", path}, want: []string{`"kind":"corpus"`, `"log":{"format":"gomad3.guide-corpus-log/v1"`, `"corpus":{"schema":"gomad3.guide-corpus/v2","generation":0,`, `"maximum_entries":1024`, `"cases":[]`}},
	} {
		var stdout, stderr bytes.Buffer
		if status := runInspect(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
			t.Fatalf("%v status=%d stdout=%q stderr=%q", test.arguments, status, stdout.String(), stderr.String())
		}
		for _, want := range test.want {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("%v output = %q, missing %q", test.arguments, stdout.String(), want)
			}
		}
	}
	var stdout, stderr bytes.Buffer
	if status := runInspect([]string{"--choices", path}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "traced artifact") {
		t.Fatalf("corpus --choices status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

// Committed outcomes, the admission frontier, exploration rounds and merged
// source provenance print one line each.
func TestPrintInspectionReportsCommittedStateRoundsAndMergedSources(t *testing.T) {
	report := runner.Inspection{
		Kind: "campaign", Path: "/batch",
		Log:       &runner.LogInspection{Format: "gomad3.campaign-log/v1", FrameVersion: 1, State: "valid", Records: 9, ValidBytes: 900, SnapshotBytes: 900, Tail: "none"},
		Lifecycle: &runner.CampaignLifecycleInspection{State: "running", Resumable: true},
		Committed: &runner.CommittedCampaignInspection{
			Outcomes: 2, OutcomeBytes: 400, Limits: runner.ExecutionJournalLimitsInspection{MaximumExecutions: 4, MaximumBytes: 4096, MaximumPartialExecutions: 2},
			Admitted:  []runner.AdmittedOrdinalInspection{{Ordinal: 2, Seed: 9, Receipt: &runner.AdmissionReceiptInspection{Domain: "target", Reason: "exit", Termination: "exit"}}, {Ordinal: 3, Seed: 10}},
			Available: 0, PendingStop: "first_failure",
		},
		Exploration: &runner.ExplorationInspection{
			Strategy: "choice-exploration", ChainSHA256: "sha256:chain", RecoveryExecutions: 1,
			Rounds:      []runner.ExplorationRoundInspection{{Index: 0, RoundBytes: 77, FirstOrdinal: 0, Executions: 2, RoundSHA256: "sha256:round", SegmentSHA256: "sha256:segment"}},
			StagedRound: &runner.ExplorationStagedRound{Index: 1, Candidates: 3},
		},
	}
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"committed: outcomes=2 bytes=400 max-executions=4 max-bytes=4096 max-partials=2 admitted=2 available=0 pending-stop=first_failure stopped=\n",
		"admitted: ordinal=2 seed=9 receipt=target/exit/exit\n", "admitted: ordinal=3 seed=10 receipt=none\n",
		"exploration-rounds: strategy=choice-exploration rounds=1 recovery-executions=1 chain=sha256:chain\n",
		"exploration-round: index=0 bytes=77 first-ordinal=0 executions=2 sha256=sha256:round segment-sha256=sha256:segment\n",
		"exploration-staged-round: index=1 candidates=3\n",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("inspection output = %q, missing %q", output.String(), want)
		}
	}

	merged := runner.Inspection{Kind: "merged-campaign", Path: "/merged", Merged: &runner.MergedCampaignInspection{
		PlanSHA256: "sha256:plan", Shards: 1, SourceCampaignIDs: []string{"campaign-a"},
		Limits:           runner.ExecutionJournalLimitsInspection{MaximumExecutions: 4, MaximumBytes: 4096, MaximumPartialExecutions: 2},
		ArtifactCapacity: runner.ArtifactCapacityInspection{TotalBytes: 8192},
		Sources:          []runner.MergedSourceInspection{{Index: 0, Count: 1, CampaignID: "campaign-a", Attempted: 4, JournalBytes: 400, Records: 9, LogSHA256: "sha256:log", Path: "/shards/a"}},
	}}
	output.Reset()
	if err := printInspection(&output, merged); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"merged-limits: max-executions=4 max-bytes=4096 max-partials=2 artifact-max-bytes=8192\n", "source-campaign: campaign-a\n", "source-log: index=0 count=1 campaign=campaign-a attempted=4 journal-bytes=400 records=9 log-sha256=sha256:log path=/shards/a\n"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("merged output = %q, missing %q", output.String(), want)
		}
	}
}

func itoa(value int64) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
