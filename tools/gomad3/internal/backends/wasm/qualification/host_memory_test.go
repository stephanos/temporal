package qualification

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	simulation "go.temporal.io/server/tools/gomad3/internal/backends/wasm/cluster"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
)

func temporalHostMemoryObserver(mode string, log func(string)) (func(string), error) {
	if mode == "" {
		return nil, nil
	}
	if mode != "1" || log == nil {
		return nil, errors.New("host memory diagnostics require explicit mode 1 and bounded logger")
	}
	count := 0
	return func(stage string) {
		if count >= 128 {
			return
		}
		count++
		if count == 128 {
			log("host-memory-stage diagnostic-truncated (nonacceptance)")
			return
		}
		switch stage {
		case "record.execute.before", "record.execute.after", "record.retention.before", "record.retention.after", "artifact.publish.before", "artifact.publish.after", "replay.open.before", "replay.open.after", "replay.read.before", "replay.read.after", "replay.bound.before", "replay.bound.after", "replay.encode.before", "replay.encode.after", "replay.compare.before", "replay.compare.after", "result.validate.before", "result.validate.after", "result.encode.before", "result.encode.after", "result.publish.before", "result.publish.after", "result.publication.ready":
		default:
			log("host-memory-stage invalid-label (nonacceptance)")
			return
		}
		var heap runtime.MemStats
		runtime.ReadMemStats(&heap)
		self := hostexec.CurrentProcessResourceUsage()
		available := self != nil
		if self == nil {
			self = &hostexec.ResourceUsage{}
		}
		log(fmt.Sprintf("host-memory-stage nonacceptance index=%d name=%s self-available=%t self-user-nanos=%d self-system-nanos=%d self-peak-rss=%d heap-alloc=%d heap-sys=%d heap-released=%d total-alloc=%d num-gc=%d", count, stage, available, self.UserNanos, self.SystemNanos, self.PeakRSSBytes, heap.HeapAlloc, heap.HeapSys, heap.HeapReleased, heap.TotalAlloc, heap.NumGC))
	}, nil
}

func TestHostMemoryDiagnosticIsExplicitBoundedAndHostOnly(t *testing.T) {
	var lines []string
	log := func(line string) { lines = append(lines, line) }
	absent, err := temporalHostMemoryObserver("", log)
	if err != nil || absent != nil || len(lines) != 0 {
		t.Fatal("absent diagnostic did work")
	}
	for _, mode := range []string{"0", "true", " 1", "1 "} {
		if _, err := temporalHostMemoryObserver(mode, log); err == nil {
			t.Fatalf("implicit diagnostic mode accepted: %q", mode)
		}
	}
	observe, err := temporalHostMemoryObserver("1", log)
	if err != nil {
		t.Fatal(err)
	}
	observe("replay.read.before")
	observe("replay.read.after")
	if len(lines) != 2 {
		t.Fatal("actual diagnostic observations missing")
	}
	values := func(line string) map[string]uint64 {
		out := map[string]uint64{}
		for _, field := range strings.Fields(line) {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			if number, err := strconv.ParseUint(value, 10, 64); err == nil {
				out[key] = number
			}
		}
		return out
	}
	before, after := values(lines[0]), values(lines[1])
	if before["index"] != 1 || after["index"] != 2 || before["heap-alloc"] == 0 || before["heap-sys"] == 0 || after["total-alloc"] < before["total-alloc"] || after["num-gc"] < before["num-gc"] {
		t.Fatalf("actual heap counters unavailable or nonmonotone: %v", lines)
	}
	if hostexec.CurrentProcessResourceUsage() != nil && (before["self-peak-rss"] == 0 || after["self-user-nanos"] < before["self-user-nanos"] || after["self-system-nanos"] < before["self-system-nanos"] || after["self-peak-rss"] < before["self-peak-rss"]) {
		t.Fatal("actual self resource counters unavailable or nonmonotone")
	}
	for range 256 {
		observe("replay.read.before")
	}
	observe(strings.Repeat("hostile", 1<<10))
	if len(lines) != 128 || !strings.Contains(lines[len(lines)-1], "diagnostic-truncated") {
		t.Fatal("diagnostic log count unbounded or silently truncated")
	}
	for _, line := range lines {
		if len(line) > 1024 {
			t.Fatal("diagnostic log line unbounded")
		}
	}
	var spec Spec
	if err := strictDecode([]byte(`{"observeHostMemoryStage":"hostile"}`), &spec); err == nil || spec.observeHostMemoryStage != nil {
		t.Fatal("private observer became caller-supplied configuration")
	}
}

func TestHostMemoryDiagnosticInvalidLabelsNeverCopyInput(t *testing.T) {
	var lines []string
	observe, err := temporalHostMemoryObserver("1", func(line string) { lines = append(lines, line) })
	if err != nil {
		t.Fatal(err)
	}
	observe(strings.Repeat("hostile", 1<<10))
	if len(lines) != 1 || len(lines[0]) > 1024 || strings.Contains(lines[0], "hostile") || !strings.Contains(lines[0], "invalid-label") {
		t.Fatal("untrusted diagnostic stage copied or sampled")
	}
}

func TestHostMemoryReplayErrorRetainsStagePairWithoutDispatch(t *testing.T) {
	var stages []string
	_, _, err := replayClusterArtifactObserved(t.Context(), filepath.Join(t.TempDir(), "missing"), t.TempDir(), simulation.InstalledHelper{}, func(stage string) { stages = append(stages, stage) })
	if err == nil || strings.Join(stages, ",") != "replay.open.before,replay.open.after" {
		t.Fatalf("failed open lost observation pair or dispatched: %v %v", stages, err)
	}
}

func TestHostMemoryPublicationErrorRetainsValidationStagePair(t *testing.T) {
	var stages []string
	path, receipt, err := writeResultObserved(t.TempDir(), Result{}, func(stage string) { stages = append(stages, stage) })
	if err == nil || path != "" || receipt.OperatorAfter != nil || strings.Join(stages, ",") != "result.validate.before,result.validate.after" {
		t.Fatalf("failed validation fabricated publication or lost stage: %v %v", stages, err)
	}
}
