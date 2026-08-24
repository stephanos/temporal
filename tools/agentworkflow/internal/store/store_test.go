package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStorePublishesAndValidatesImmutableEvidence(t *testing.T) {
	root, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run, err := root.Create("run-one", []byte(`{"task":"test"}`), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := run.WriteCheckpoint([]byte(`{"phase":"plan"}`), "running", "plan", "", time.Unix(2, 0)); err != nil {
		t.Fatal(err)
	}
	recorder, err := run.StartAttempt("plan", 1<<20, 10, time.Unix(3, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Emit([]byte(`{"kind":"started"}`)); err != nil {
		t.Fatal(err)
	}
	if err := recorder.SetSession("session-one"); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Finish("completed", "session-one", []byte(`{"ok":true}`), nil, time.Unix(4, 0)); err != nil {
		t.Fatal(err)
	}
	if err := run.PublishResult([]byte(`{"outcome":"succeeded"}`), "succeeded", "complete", time.Unix(5, 0)); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	inspection, err := root.Inspect("run-one", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Manifest.State != "terminal" || inspection.Recoverable || len(inspection.Attempts) != 1 {
		t.Fatalf("inspection = %#v", inspection)
	}
	attempt := inspection.Attempts[0]
	if attempt.Session != "session-one" || attempt.Status != "completed" || attempt.EventCount != 1 {
		t.Fatalf("attempt = %#v", attempt)
	}
}

func TestStoreDetectsArtifactCorruption(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := run.PublishResult([]byte(`{"ok":true}`), "succeeded", "complete", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root.Root(), "run", "result.json"), []byte(`{"ok":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Inspect("run", 1<<20); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Inspect() error = %v, want corruption", err)
	}
}

func TestStoreRejectsConcurrentRunAcquisition(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := root.Acquire("run"); !errors.Is(err, ErrLocked) {
		t.Fatalf("Acquire() error = %v, want lock", err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := root.Acquire("run")
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRecoversDeadOwnerLockAndRetainsIt(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	host, _ := os.Hostname()
	dead := lockRecord{Schema: "agentworkflow.lock/v1", PID: 1 << 30, Host: host, Token: "dead", Acquired: time.Now().String()}
	if err := createExclusiveJSON(filepath.Join(root.Root(), "run", "running.lock"), dead); err != nil {
		t.Fatal(err)
	}
	recovered, err := root.Acquire("run")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root.Root(), "run", "stale-lock-dead.json")); err != nil {
		t.Fatal(err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRecorderStopsBeforeExceedingEventBudgetAndCanPublishFailure(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := run.StartAttempt("stage", 32, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Emit([]byte(`{"x":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Emit([]byte(`{"x":2}`)); !errors.Is(err, ErrCapacity) {
		t.Fatalf("Emit() error = %v, want capacity", err)
	}
	if err := recorder.Finish("failed", "", nil, ErrCapacity, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	inspection, err := root.Inspect("run", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Attempts[0].EventCount != 1 || inspection.Attempts[0].Status != "failed" {
		t.Fatalf("attempt = %#v", inspection.Attempts[0])
	}
}

func TestRunReadersAndMetadataAccessorsVerifyArtifacts(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if run.ID() != "run" || run.Directory() != filepath.Join(root.Root(), "run") || run.Manifest().RunID != "run" {
		t.Fatalf("run metadata = %q %q %#v", run.ID(), run.Directory(), run.Manifest())
	}
	if request, err := run.ReadRequest(1 << 20); err != nil || string(request) != `{"task":1}` {
		t.Fatalf("request = %q, %v", request, err)
	}
	checkpoint := []byte(`{"phase":"test"}`)
	if err := run.WriteCheckpoint(checkpoint, "running", "test", "", time.Now()); err != nil {
		t.Fatal(err)
	}
	if stored, err := run.ReadCheckpoint(1 << 20); err != nil || string(stored) != string(checkpoint) {
		t.Fatalf("checkpoint = %q, %v", stored, err)
	}
	result := []byte(`{"outcome":"failed"}`)
	if err := run.PublishResult(result, "failed", "test", time.Now()); err != nil {
		t.Fatal(err)
	}
	if stored, err := run.ReadResult(1 << 20); err != nil || string(stored) != string(result) {
		t.Fatalf("result = %q, %v", stored, err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRecorderOversizedFinalOutputPublishesFailedAttemptAndReturnsCapacity(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := run.StartAttempt("stage", 16, 10, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Finish("completed", "session", []byte(`{"value":"far too large"}`), nil, time.Now()); !errors.Is(err, ErrCapacity) {
		t.Fatalf("Finish() error = %v, want capacity", err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	inspection, err := root.Inspect("run", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Attempts[0].Status != "failed" || inspection.Attempts[0].OutputPath != "" {
		t.Fatalf("attempt = %#v", inspection.Attempts[0])
	}
}

func TestStoreSupportsConcurrentIndependentRuns(t *testing.T) {
	root, _ := Open(t.TempDir())
	const count = 16
	var group sync.WaitGroup
	errorsByRun := make([]error, count)
	for index := 0; index < count; index++ {
		index := index
		group.Add(1)
		go func() {
			defer group.Done()
			request, _ := json.Marshal(map[string]int{"run": index})
			run, err := root.Create("run-"+string(rune('a'+index)), request, time.Now())
			if err == nil {
				err = run.Close()
			}
			errorsByRun[index] = err
		}()
	}
	group.Wait()
	for _, err := range errorsByRun {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestStoreStrictlyRejectsUnknownManifestFields(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root.Root(), "run", "run.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] = ','
	data = append(data, []byte(`"unknown":true}`)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Inspect("run", 1<<20); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Inspect() error = %v, want corruption", err)
	}
}

func TestStoreInspectsLegacyV1StageRecordsReadOnly(t *testing.T) {
	root, _ := Open(t.TempDir())
	runDirectory := filepath.Join(root.Root(), "legacy-run")
	stageDirectory := filepath.Join(runDirectory, "stages", "plan")
	if err := os.MkdirAll(stageDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	events := []byte("{\"type\":\"item.completed\"}\n")
	stderr := []byte("diagnostic")
	if err := os.WriteFile(filepath.Join(stageDirectory, "events.jsonl"), events, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDirectory, "stderr.log"), stderr, 0o600); err != nil {
		t.Fatal(err)
	}
	record := legacyStageResult{
		Schema: "agentworkflow.stage-result/v1", RunID: "legacy-run", Stage: "plan", Status: "completed",
		ThreadID: "thread", FinalOutput: "plan", EventCount: 1,
		StdoutSHA256: digest("agentworkflow.stdout/v1", events), StderrSHA256: digest("agentworkflow.stderr/v1", stderr),
		StdoutBytes: uint64(len(events)), StderrBytes: uint64(len(stderr)),
		RunDirectory: runDirectory, StageDirectory: stageDirectory,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDirectory, "stage.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	inspection, err := root.Inspect("legacy-run", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Manifest.Schema != "agentworkflow.run/v1" || inspection.Manifest.State != "legacy" || len(inspection.Attempts) != 1 {
		t.Fatalf("inspection = %#v", inspection)
	}
	if inspection.Attempts[0].Schema != "agentworkflow.stage-result/v1" || inspection.Attempts[0].Stage != "plan" || inspection.Attempts[0].Status != "completed" {
		t.Fatalf("legacy attempt = %#v", inspection.Attempts[0])
	}
	if _, err := os.Stat(filepath.Join(runDirectory, "run.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy inspection mutated the run: %v", err)
	}
}

func TestStoreRejectsCorruptLegacyV1Evidence(t *testing.T) {
	root, _ := Open(t.TempDir())
	runDirectory := filepath.Join(root.Root(), "legacy-run")
	stageDirectory := filepath.Join(runDirectory, "stages", "plan")
	if err := os.MkdirAll(stageDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	record := legacyStageResult{
		Schema: "agentworkflow.stage-result/v1", RunID: "legacy-run", Stage: "plan", Status: "completed",
		StdoutSHA256: "sha256:wrong", StderrSHA256: digest("agentworkflow.stderr/v1", nil),
		RunDirectory: runDirectory, StageDirectory: stageDirectory,
	}
	encoded, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(stageDirectory, "stage.json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDirectory, "events.jsonl"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stageDirectory, "stderr.log"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Inspect("legacy-run", 1<<20); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Inspect() error = %v, want corruption", err)
	}
}

func TestStoreRecoversRunningAttemptWithPublishedOutput(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := run.StartAttempt("implement", 1<<20, 10, time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{
		`{"kind":"invocation-started"}`,
		`{"kind":"session-identified","session":"session-one"}`,
		`{"kind":"invocation-completed"}`,
	} {
		if err := recorder.Emit([]byte(event)); err != nil {
			t.Fatal(err)
		}
	}
	if err := recorder.SetSession("session-one"); err != nil {
		t.Fatal(err)
	}
	if err := recorder.events.Close(); err != nil {
		t.Fatal(err)
	}
	output := []byte(`{"summary":"implemented"}`)
	if err := atomicWrite(filepath.Join(recorder.directory, "output.json"), output); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := root.Acquire("run")
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.RecoverAttempts(1<<20, 10, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	attempt, stored, found, err := recovered.ReadCompletedAttempt("implement", 1<<20)
	if err != nil || !found || string(stored) != string(output) || attempt.Status != "completed" || attempt.Session != "session-one" {
		t.Fatalf("completed attempt = %#v, %q, %t, %v", attempt, stored, found, err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
	inspection, err := root.Inspect("run", 1<<20)
	if err != nil || inspection.Recoverable || inspection.Attempts[0].Status != "completed" {
		t.Fatalf("inspection = %#v, %v", inspection, err)
	}
}

func TestStoreFinalizesRunningAttemptWithoutOutputAsInterrupted(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := run.StartAttempt("implement", 1<<20, 10, time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Emit([]byte(`{"kind":"progress"}`)); err != nil {
		t.Fatal(err)
	}
	if err := recorder.events.Close(); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	recovered, err := root.Acquire("run")
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.RecoverAttempts(1<<20, 10, time.Unix(3, 0)); err != nil {
		t.Fatal(err)
	}
	if _, _, found, err := recovered.ReadCompletedAttempt("implement", 1<<20); err != nil || found {
		t.Fatalf("completed attempt found=%t error=%v", found, err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
	inspection, err := root.Inspect("run", 1<<20)
	if err != nil || inspection.Recoverable || inspection.Attempts[0].Status != "interrupted" {
		t.Fatalf("inspection = %#v, %v", inspection, err)
	}
}

func TestStoreReportsRunningAttemptsAsRecoverable(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.StartAttempt("implement", 1<<20, 10, time.Now()); err != nil {
		t.Fatal(err)
	}
	inspection, err := root.Inspect("run", 1<<20)
	if err != nil || !inspection.Recoverable || inspection.Attempts[0].Status != "running" {
		t.Fatalf("inspection=%#v error=%v", inspection, err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRejectsCorruptRunningEventPrefixDuringRecovery(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	recorder, err := run.StartAttempt("implement", 1<<20, 10, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.events.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(recorder.directory, "events.jsonl"), []byte("{not-json}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
	recovered, err := root.Acquire("run")
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.RecoverAttempts(1<<20, 10, time.Now()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("RecoverAttempts() error=%v", err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreValidatesAttemptLifecycleAndReaderBounds(t *testing.T) {
	root, _ := Open(t.TempDir())
	run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.ReadResult(1 << 20); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing result error=%v", err)
	}
	if _, err := run.StartAttempt("bad/stage", 1<<20, 10, time.Now()); err == nil {
		t.Fatal("invalid stage was accepted")
	}
	if _, err := run.StartAttempt("stage", 0, 10, time.Now()); err == nil {
		t.Fatal("invalid attempt bound was accepted")
	}
	recorder, err := run.StartAttempt("stage", 1<<20, 10, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Emit([]byte("not-json")); err == nil {
		t.Fatal("invalid event was accepted")
	}
	if err := recorder.SetSession(""); err == nil {
		t.Fatal("empty session was accepted")
	}
	if err := recorder.SetSession("one"); err != nil {
		t.Fatal(err)
	}
	if err := recorder.SetSession("two"); err == nil {
		t.Fatal("changed session was accepted")
	}
	if err := recorder.Finish("unknown", "one", nil, nil, time.Now()); err == nil {
		t.Fatal("unknown terminal status was accepted")
	}
	if err := recorder.Finish("interrupted", "one", nil, context.Canceled, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := recorder.Emit([]byte(`{}`)); err == nil {
		t.Fatal("closed recorder accepted an event")
	}
	if err := run.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRejectsTamperedRequestAndCheckpointEvidence(t *testing.T) {
	for _, artifact := range []string{"request.json", "checkpoint"} {
		t.Run(artifact, func(t *testing.T) {
			root, _ := Open(t.TempDir())
			run, err := root.Create("run", []byte(`{"task":1}`), time.Now())
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root.Root(), "run", "request.json")
			if artifact == "checkpoint" {
				if err := run.WriteCheckpoint([]byte(`{"phase":"test"}`), "running", "test", "", time.Now()); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(root.Root(), "run", filepath.FromSlash(run.Manifest().CheckpointPath))
			}
			if err := run.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(`{"tampered":true}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := root.Inspect("run", 1<<20); !errors.Is(err, ErrCorrupt) {
				t.Fatalf("Inspect() error=%v", err)
			}
		})
	}
}
