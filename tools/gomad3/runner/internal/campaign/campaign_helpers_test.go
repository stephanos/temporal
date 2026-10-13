package campaign

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/logstore"
)

func testBatchPlan(journal *CampaignJournal, digest record.SHA256, size uint64) CampaignPlan {
	var journalPlan *ExecutionJournalPlan
	parallel := record.Uint64String(2)
	if journal != nil {
		value := journal.ExecutionJournalPlan()
		journalPlan = &value
		parallel = value.MaximumPartialExecutions
	}
	plan := CampaignPlan{
		Schema: CampaignPlanSchema, Strategy: "seed", Selection: "7-9", SelectionCount: 3, Parallel: parallel,
		Journal:               journalPlan,
		ExecutionTimeoutNanos: record.Uint64String(time.Second), OverallTimeoutNanos: record.Uint64String(10 * time.Second), TerminateGraceNanos: record.Uint64String(100 * time.Millisecond),
		OnFailure: "all", FailureBudget: 1, OutputBytes: 64, WorldTransitionBytes: 1024,
		RunnerBuild: string(record.HashBytes([]byte("runner"))), Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Prepared: PreparedTargetPlan{
			Path:   ".prepared/build/target",
			Target: record.Target{Kind: "go-test", Source: "./pkg", SHA256: digest, Size: record.Uint64String(size), Argv: []string{"gomad3-target"}, BuildTags: []string{"gomad_fixture"}, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, CapabilityMode: "closure", BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.test/pkg.test"}},
		},
		IOProfile:   record.IOContract{Name: "gomad3-deterministic/v1", ImplementationSHA256: record.SHA256(record.HashBytes([]byte("io"))), InventorySHA256: record.SHA256(record.HashBytes([]byte("inventory")))},
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: "gomad3-deterministic/v1"}},
		Coverage:    "semantic", RequiredSemanticProbes: []string{"stdlib.os.openfile"}, KeepSuccesses: "none",
	}
	artifacts, err := DeriveArtifactCapacityPlan(plan)
	if err != nil {
		panic(err)
	}
	plan.Artifacts = &artifacts
	return plan
}

func newPreparedLifecycleJournal(t *testing.T, id string) *CampaignJournal {
	t.Helper()
	journal, err := NewCampaignJournal(context.Background(), CampaignConfig{
		Root: t.TempDir(), CampaignID: id, Selection: "7-9", SelectionCount: 3,
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
	target := []byte("prepared target")
	if err := os.WriteFile(preparedPath, target, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := journal.RecordPlan(testBatchPlan(journal, record.HashBytes(target), uint64(len(target)))); err != nil {
		t.Fatal(err)
	}
	if err := journal.CompletePreparation(); err != nil {
		t.Fatal(err)
	}
	return journal
}

func assertLifecycle(t *testing.T, path string, state, stable lifecycleState, resumable bool) {
	t.Helper()
	status, err := InspectCampaignLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != state || status.LastStableState != stable || status.Resumable != resumable {
		t.Fatalf("lifecycle status = %#v", status)
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	return ctx
}

func successExecution(ordinal, seed uint64) ExecutionRecord {
	return ExecutionRecord{SelectionOrdinal: record.Uint64String(ordinal), Seed: record.Uint64String(seed), Domain: "success", Reason: "success", Termination: "exit"}
}

// campaignLogRecords replays a campaign's log read-only and returns its
// record kinds in order.
func campaignLogRecords(t *testing.T, path string) ([]logstore.Kind, logstore.Report) {
	t.Helper()
	var kinds []logstore.Kind
	report, err := logstore.Replay(filepath.Join(path, storeDirectory), campaignLogOptions(nil, false), func(entry logstore.Record) error {
		kinds = append(kinds, entry.Kind)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return kinds, report
}

func logFileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(filepath.Join(path, storeDirectory, logstore.LogFile))
	if err != nil {
		t.Fatal(err)
	}
	return info.Size()
}

// mutationController injects mutation faults once it is enabled.
type mutationController struct {
	enabled bool
	inject  mutationHook
}

func (controller *mutationController) observe(point mutationPoint) error {
	if !controller.enabled {
		return nil
	}
	return controller.inject(point)
}
