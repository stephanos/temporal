package simulation

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/world"
)

// This explicitly requested operator probe is not a guest execution or a
// qualification cost sample. Legacy materialization runs only after measuring
// the production compressed ownership/validation path.
func TestCompressedWorldOwnerMeasuredCardinality(t *testing.T) {
	if os.Getenv("GOMAD3_COMPRESSED_WORLD_PROBE") != "1" {
		t.Skip("explicit allocation probe")
	}
	requests := 20000
	switch os.Getenv("GOMAD3_COMPRESSED_WORLD_PROBE_REQUESTS") {
	case "", "20000":
	case "200000":
		requests = 200000
	default:
		t.Fatal("probe cardinality must be explicitly 20000 or 200000")
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	logStage := func(stage string) {
		runtime.ReadMemStats(&after)
		status, err := os.ReadFile("/proc/self/status")
		if err != nil {
			t.Fatal(err)
		}
		var rss, peak uint64
		for _, line := range strings.Split(string(status), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 3 && (fields[0] == "VmRSS:" || fields[0] == "VmHWM:") {
				n, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					t.Fatal(err)
				}
				if fields[0] == "VmRSS:" {
					rss = n * 1024
				} else {
					peak = n * 1024
				}
			}
		}
		t.Logf("stage=%s requests=%d heap_live=%d heap_allocated_delta=%d process_rss=%d process_peak_rss=%d", stage, requests, after.HeapAlloc, after.TotalAlloc-before.TotalAlloc, rss, peak)
		if peak > 2<<30 {
			t.Fatal("operator whole-process peak exceeds the 2GiB investigation budget")
		}
	}
	config := worldCodecConfig()
	if requests != 20000 {
		config.Limits.MaxRequests = uint64(requests + 1)
		config.Limits.MaxEvents = uint64(requests + 1)
		config.Limits.MaxTransitions = uint64(requests*3 + 1)
	}
	model, err := world.New(config)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < requests; i++ {
		id, err := model.Register(world.Request{Kind: "runtime_net_stream", Resource: world.ResourceID{Adapter: "wasm-broker", Kind: "incarnation", Key: "active/1"}, Payload: make([]byte, 32)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := model.Ready(world.Readiness{RequestID: id, At: world.InitialTime + world.LogicalTime(i+1), Kind: "runtime_net_stream"}); err != nil {
			t.Fatal(err)
		}
		if _, err := model.Quiesce(); err != nil {
			t.Fatal(err)
		}
	}
	logStage("model")
	cardinality := model.DiagnosticCardinality()
	profile := world.CompressedSnapshotProfile()
	detached, err := model.DetachCompressedSnapshot(profile)
	if err != nil {
		t.Fatal(err)
	}
	logStage("terminal_transfer")
	encoded, err := detached.Encode()
	if err != nil {
		t.Fatal(err)
	}
	logStage("compressed_encode")
	decoded, err := world.DecodeCompressedSnapshot(encoded, profile)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.RequestCount != cardinality.Requests || decoded.EventCount != cardinality.Events || decoded.TransitionCount != cardinality.Transitions || decoded.TranscriptDigest != cardinality.TranscriptDigest {
		t.Fatal("compressed semantic validation changed terminal cardinality")
	}
	logStage("compressed_streaming_semantic_decode")
	t.Logf("envelope_bytes=%d gzip_bytes=%d expanded_bytes=%d expanded_sha256=%s implementation_sha256=%s", len(encoded), decoded.CompressedBytes, decoded.ExpandedBytes, decoded.ExpandedSHA256, world.ImplementationSHA256())
	if decoded.ExpandedBytes > world.MaximumSnapshotJSONBytes {
		t.Logf("legacy_materialization=not_attempted expanded_bytes=%d exceeds unchanged legacy_codec_capacity=%d; all compressed semantic/canonical validation completed", decoded.ExpandedBytes, world.MaximumSnapshotJSONBytes)
		return
	}

	// Materialize only for the independent legacy-owner byte equality check.
	var envelope struct {
		Gzip []byte `json:"gzip"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(envelope.Gzip))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	legacy, err := world.DecodeSnapshot(expanded)
	if err != nil {
		t.Fatal(err)
	}
	again, err := world.EncodeSnapshot(legacy)
	if err != nil || !bytes.Equal(expanded, again) {
		t.Fatalf("expanded snapshot differs from legacy canonical owner: %v", err)
	}
	logStage("legacy_independent_byte_equivalence")
	t.Logf("legacy_bytes=%d legacy_sha256=%x", len(again), sha256.Sum256(again))
}
