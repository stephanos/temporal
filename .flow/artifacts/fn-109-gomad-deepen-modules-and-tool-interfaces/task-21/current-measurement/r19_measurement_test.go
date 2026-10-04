package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"runtime/pprof"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

const r19PayloadBytes = 1 << 20

type r19Snapshot struct {
	Name        string
	Returned    uint64
	Committed   uint64
	Active      int
	HeapAlloc   uint64
	HeapObjects uint64
	TotalAlloc  uint64
	NumGC       uint32
}

type r19Executor struct {
	t                         *testing.T
	jobs                      uint64
	dir                       string
	mu                        sync.Mutex
	waiting                   chan struct{}
	active                    int
	maximumActive             int
	returned                  atomic.Uint64
	committed                 atomic.Uint64
	produced                  atomic.Uint64
	worldProduced             atomic.Uint64
	transcriptProduced        atomic.Uint64
	transcriptRecordsProduced atomic.Uint64
	snapshots                 []r19Snapshot
}

func r19Stream(seed uint64, stream byte) hostexec.Output {
	payload := make([]byte, r19PayloadBytes)
	for index := range payload {
		payload[index] = byte(seed) ^ stream ^ byte(index)
	}
	digest := sha256.Sum256(payload)
	return hostexec.Output{Bytes: payload, FullSHA256: digest, RetainedSHA256: digest, TotalBytes: uint64(len(payload)), RetainedBytes: uint64(len(payload))}
}

func r19Transcript(t *testing.T) deterministicio.Transcript {
	probe := semanticTranscript(t, completionProbe)
	first, err := deterministicio.DecodeTranscript(probe.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	operations := make([]deterministicio.Operation, r19PayloadBytes/len(probe.Bytes))
	operations[0] = first[0]
	for index := 1; index < len(operations); index++ {
		operations[index] = deterministicio.Operation{Ordinal: uint64(index), Name: "host.hostname"}
	}
	payload, err := deterministicio.EncodeTranscript(operations)
	if err != nil {
		t.Fatal(err)
	}
	return deterministicio.Transcript{Bytes: payload, SHA256: sha256.Sum256(payload), Records: uint64(len(operations)), Complete: true}
}

func (executor *r19Executor) snapshot(name string) {
	// Two complete GC cycles reconcile the memory profiler's delayed frees.
	runtime.GC()
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	executor.mu.Lock()
	observed := r19Snapshot{Name: name, Returned: executor.returned.Load(), Committed: executor.committed.Load(), Active: executor.active, HeapAlloc: stats.HeapAlloc, HeapObjects: stats.HeapObjects, TotalAlloc: stats.TotalAlloc, NumGC: stats.NumGC}
	executor.snapshots = append(executor.snapshots, observed)
	executor.mu.Unlock()
	file, err := os.Create(filepath.Join(executor.dir, name+".pprof"))
	if err != nil {
		executor.t.Fatal(err)
	}
	if err := pprof.WriteHeapProfile(file); err != nil {
		executor.t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		executor.t.Fatal(err)
	}
}

func (executor *r19Executor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	result := execution.Result{Captured: true, Termination: execution.TerminationExit, GroupGone: true}
	result.Stdout = r19Stream(request.World.Seed, 'o')
	result.Stderr = r19Stream(request.World.Seed, 'e')
	result.IOTranscript = r19Transcript(executor.t)
	result.WorldRecord = completionWorldRecord(executor.t, request.World.Seed)
	executor.produced.Add(uint64(len(result.Stdout.Bytes) + len(result.Stderr.Bytes) + len(result.IOTranscript.Bytes)))
	executor.worldProduced.Add(uint64(len(result.WorldRecord)))
	executor.transcriptProduced.Add(uint64(len(result.IOTranscript.Bytes)))
	executor.transcriptRecordsProduced.Add(result.IOTranscript.Records)
	executor.mu.Lock()
	executor.active++
	executor.maximumActive = max(executor.maximumActive, executor.active)
	partner := executor.waiting
	if partner == nil {
		partner = make(chan struct{})
		executor.waiting = partner
		executor.mu.Unlock()
		select {
		case <-partner:
		case <-ctx.Done():
			return execution.Result{}, ctx.Err()
		}
	} else {
		executor.waiting = nil
		executor.mu.Unlock()
		if request.World.Seed <= 2 {
			executor.snapshot("early-pair")
		}
		if request.World.Seed >= executor.jobs-1 {
			executor.snapshot("late-pair")
		}
		close(partner)
	}
	runtime.KeepAlive(result)
	executor.mu.Lock()
	executor.active--
	executor.mu.Unlock()
	executor.returned.Add(1)
	return result, nil
}

func TestR19BoundedSeedCampaignMeasurement(t *testing.T) {
	runtime.MemProfileRate = 1
	jobs, err := strconv.ParseUint(os.Getenv("R19_JOBS"), 10, 64)
	if err != nil || (jobs != 10 && jobs != 100) {
		t.Fatal("R19_JOBS must be 10 or 100")
	}
	mode := os.Getenv("R19_MODE")
	if mode != "discard" && mode != "novel" {
		t.Fatal("R19_MODE must be discard or novel")
	}
	dir := os.Getenv("R19_RESULT_DIR")
	if dir == "" {
		t.Fatal("R19_RESULT_DIR required")
	}
	executor := &r19Executor{t: t, jobs: jobs, dir: dir}
	config, dependencies := testConfig(t, newFakePreparer(t), executor, fmt.Sprintf("1-%d", jobs), PolicyAll, 2)
	config.Artifacts = filepath.Join(dir, "artifacts")
	config.Strategy, config.Coverage = StrategySeed, CoverageSemantic
	config.ExecutionTimeout, config.OverallTimeout = 5*time.Minute, 5*time.Minute
	config.OutputLimit, config.IOTranscriptLimit, config.WorldTransitionLimit = r19PayloadBytes, 64*r19PayloadBytes, r19PayloadBytes
	config.ProgressInterval = time.Nanosecond
	config.Progress = func(event CampaignEvent) error { executor.committed.Store(event.Attempted); return nil }
	if mode == "discard" {
		config.KeepSuccesses = KeepSuccessesNone
		config.SuccessArtifactLimit, config.SuccessBytesLimit = 0, 0
	} else {
		config.KeepSuccesses = KeepSuccessesNovel
		config.SuccessArtifactLimit, config.SuccessBytesLimit = 3, 64<<20
	}
	selection, err := ParseSeeds(config.Seeds)
	if err != nil {
		t.Fatal(err)
	}
	controller, err := newShardedSeedController(selection, CampaignShard{}, map[uint64]struct{}{}, 2, PolicyAll, 1, CampaignResult{})
	if err != nil {
		t.Fatal(err)
	}
	logical := map[string]uint64{
		"controller":                   uint64(unsafe.Sizeof(*controller)),
		"selection_header":             uint64(unsafe.Sizeof(selection)),
		"job_source":                   uint64(unsafe.Sizeof(pendingJobs{})),
		"iterator":                     uint64(unsafe.Sizeof(*selection.Iterator())),
		"one_shared_range_backing":     uint64(cap(selection.ranges)) * uint64(unsafe.Sizeof(seedRange{})),
		"two_completion_channel_slots": 2 * 2 * uint64(unsafe.Sizeof(runCompletion{})),
		"two_channel_handles":          2 * uint64(unsafe.Sizeof(make(chan runCompletion, 2))),
		"completion_order_done_handle": uint64(unsafe.Sizeof(make(chan struct{}))),
	}
	var logicalTotal uint64
	for _, value := range logical {
		logicalTotal += value
	}
	logical["total_named_logical_bytes"] = logicalTotal
	// This companion uses real controller transitions and retains no job history.
	for index := uint64(0); index < jobs/2; index++ {
		for count := 0; count < 2; count++ {
			if _, ok := controller.Next(); !ok {
				t.Fatal("controller failed to fill pair")
			}
		}
		if controller.Active() != 2 {
			t.Fatal("controller pair width")
		}
		for count := 0; count < 2; count++ {
			controller.Complete(campaign.CompletedSuccess())
		}
	}
	controller = nil
	selection = SeedSelection{}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	summary, err := exploreWith(ctx, config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	executor.snapshot("completed")
	runtimeSettings := []metrics.Sample{{Name: "/gc/gogc:percent"}, {Name: "/gc/gomemlimit:bytes"}, {Name: "/sched/gomaxprocs:threads"}}
	metrics.Read(runtimeSettings)
	if runtimeSettings[0].Value.Uint64() != 100 || runtimeSettings[1].Value.Uint64() != 1<<63-1 || runtimeSettings[2].Value.Uint64() != 2 {
		t.Fatal("bound runtime settings changed")
	}
	if execution.R19TransportCalls() != 0 {
		t.Fatal("excluded transport was called")
	}
	wantRetained := uint64(0)
	if mode == "novel" {
		wantRetained = 1
	}
	if summary.Attempted != jobs || summary.Succeeded != jobs || summary.RetainedSuccesses != wantRetained || executor.maximumActive != 2 {
		t.Fatalf("unexpected campaign: attempted=%d succeeded=%d retained=%d maximumActive=%d", summary.Attempted, summary.Succeeded, summary.RetainedSuccesses, executor.maximumActive)
	}
	if summary.SemanticCoverage == nil || len(summary.SemanticCoverage.Probes) != 1 || summary.SemanticCoverage.Probes[0] != completionProbe {
		t.Fatal("fixed one-probe vocabulary changed")
	}
	opened, err := campaign.OpenCampaign(summary.CampaignPath)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Journal == nil || opened.Journal.Records != jobs {
		t.Fatal("journal record count")
	}
	capacities := opened.Record.Artifacts
	if capacities == nil {
		t.Fatal("published derived artifact capacity missing")
	}
	journalNovelProbes := make(map[string]struct{})
	journalNovelChoices := make(map[string]struct{})
	for _, entry := range opened.Executions {
		for _, probe := range entry.NovelSemanticProbes {
			journalNovelProbes[probe] = struct{}{}
		}
		for _, feature := range entry.NovelChoiceFeatures {
			journalNovelChoices[feature] = struct{}{}
		}
	}
	files, fileBytes := uint64(0), uint64(0)
	err = filepath.WalkDir(summary.CampaignPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files++
		fileBytes += uint64(info.Size())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	observed := map[string]any{
		"observed_runtime_settings": map[string]uint64{"gogc_percent": runtimeSettings[0].Value.Uint64(), "gomemlimit_bytes": runtimeSettings[1].Value.Uint64(), "gomaxprocs_threads": runtimeSettings[2].Value.Uint64()},
		"jobs":                      jobs, "mode": mode, "platform": runtime.GOOS + "/" + runtime.GOARCH, "go_version": runtime.Version(), "parallel": 2,
		"attempted": summary.Attempted, "succeeded": summary.Succeeded, "maximum_active": executor.maximumActive,
		"excluded_transport_calls": execution.R19TransportCalls(),
		"logical_policy_bytes":     logical, "snapshots": executor.snapshots,
		"per_execution_stream_bytes":              r19PayloadBytes,
		"actual_per_execution_transcript_bytes":   executor.transcriptProduced.Load() / jobs,
		"actual_per_execution_transcript_records": executor.transcriptRecordsProduced.Load() / jobs,
		"cumulative_transcript_bytes":             executor.transcriptProduced.Load(),
		"transcript_vocabulary":                   map[string]any{"boundary.probe": 1, "host.hostname": executor.transcriptRecordsProduced.Load()/jobs - 1, "semantic_probes": summary.SemanticCoverage.Probes},
		"cumulative_stream_transcript_bytes":      executor.produced.Load(), "cumulative_world_record_bytes": executor.worldProduced.Load(),
		"journal": opened.Journal, "derived_artifact_capacity": capacities,
		"campaign_file_count": files, "campaign_file_bytes": fileBytes, "artifact_count": len(summary.SuccessArtifacts) + len(summary.Artifacts),
		"retained_success_count": summary.RetainedSuccesses, "retained_success_bytes": summary.RetainedSuccessBytes,
		"observed_semantic_probe_vocabulary_count":           len(summary.SemanticCoverage.Probes),
		"journal_unique_novel_semantic_probes":               len(journalNovelProbes),
		"journal_unique_novel_choice_features":               len(journalNovelChoices),
		"internal_novelty_map_cardinality_directly_observed": false,
		"diagnostic_artifact_input_field":                    "present in current ArtifactInput; diagnostics disabled identically",
		"elapsed_nanos":                                      elapsed.Nanoseconds(),
		"limits":                                             map[string]any{"output": config.OutputLimit, "io_transcript": config.IOTranscriptLimit, "world_transition": config.WorldTransitionLimit, "success_count": config.SuccessArtifactLimit, "success_bytes": config.SuccessBytesLimit, "execution_timeout": config.ExecutionTimeout.String(), "overall_timeout": config.OverallTimeout.String()},
	}
	encoded, err := json.MarshalIndent(observed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "measurement.json"), append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Log(string(encoded))
}

func TestR19ArtifactInputAliasesProducedPayloads(t *testing.T) {
	result := processResult(0, "stdout", "stderr")
	result.IOTranscript = semanticTranscript(t, completionProbe)
	result.ChoiceTrace.Trace.Bytes = []byte("choice constructor fixture")
	result.WorldRecord = completionWorldRecord(t, 7)
	bundle, err := assessWorld(result, 7, r19PayloadBytes)
	if err != nil {
		t.Fatal(err)
	}
	bundle.Payloads.Transitions = []byte("transition constructor fixture")
	mounts := &readonlymount.CapturedInputs{Descriptor: []byte("mount descriptor fixture"), Payloads: map[string][]byte{"io/file": []byte("mount payload fixture")}}
	input := executionArtifactInput(record.ExecutionRecord{}, newFakePreparer(t).prepared, result, mounts, bundle)
	if input.ReadOnlyMounts != mounts {
		t.Fatal("mount input lost pointer identity")
	}
	fields := [][2][]byte{{input.Stdout, result.Stdout.Bytes}, {input.Stderr, result.Stderr.Bytes}, {input.IOTranscript, result.IOTranscript.Bytes}, {input.ChoiceTrace, result.ChoiceTrace.Trace.Bytes}, {input.World.Initial, bundle.Payloads.Initial}, {input.World.Transitions, bundle.Payloads.Transitions}, {input.World.Final, bundle.Payloads.Final}, {input.ReadOnlyMounts.Descriptor, mounts.Descriptor}, {input.ReadOnlyMounts.Payloads["io/file"], mounts.Payloads["io/file"]}}
	for index, pair := range fields {
		if len(pair[0]) == 0 {
			t.Fatalf("field %d is empty", index)
		}
		if !bytes.Equal(pair[0], pair[1]) || &pair[0][0] != &pair[1][0] {
			t.Fatalf("field %d lost alias", index)
		}
	}
}
