package simulation

import (
	"bytes"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

func worldCodecConfig() world.Config {
	return world.Config{Seed: 7, Limits: world.Limits{MaxRequests: 20001, MaxEvents: 20001, MaxQueuedEvents: 1, MaxTransitions: 100000, MaxPayloadBytes: 32 << 20, MaxStringBytes: 1024}}
}

func TestBrokerWorldUsesBinaryOwnerCodec(t *testing.T) {
	b, err := wasi.NewBroker(wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}}, Seed: 7, Limits: worldCodecConfig().Limits})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	e := &execution{broker: b, limits: gomad3sim.DefaultLimits()}
	evidence, err := e.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeSnapshot(evidence.Execution.World); err != nil {
		t.Fatalf("broker evidence bypasses binary World codec: %v", err)
	}
}

func TestWorldOwnerCodecMeasuredCardinality(t *testing.T) {
	if os.Getenv("GOMAD3_WORLD_CODEC_PROBE") != "1" {
		t.Skip("explicit allocation probe")
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	model, err := world.New(worldCodecConfig())
	if err != nil {
		t.Fatal(err)
	}
	const requests = 20000
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
	}
	logStage("model")
	snapshot := model.Snapshot()
	logStage("snapshot")
	encoded, err := world.EncodeSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	logStage("encode_restore_canonical")
	decoded, err := world.DecodeSnapshot(encoded)
	if err != nil {
		t.Fatal(err)
	}
	logStage("decode_restore_canonical")
	again, err := world.EncodeSnapshot(decoded)
	if err != nil || !bytes.Equal(encoded, again) {
		t.Fatalf("owner-codec replay changed: %v", err)
	}
	logStage("reencode_restore_canonical")
	t.Logf("encoded_bytes=%d bytes_per_request=%.2f transitions=%d maximum_codec_bytes=%d", len(encoded), float64(len(encoded))/requests, len(snapshot.Transitions), world.MaximumSnapshotJSONBytes)
}
