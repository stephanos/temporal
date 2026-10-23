package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

const networkAdapterCorpusDirectory = "../../../rust/wasmhost/tests/testdata/network_adapter_cases"

type networkAdapterInput struct {
	Action    string `json:"action"`
	Data      []byte `json:"data,omitempty"`
	Nanos     uint64 `json:"nanos,omitempty"`
	Deadline  uint64 `json:"deadline,omitempty"`
	Link      string `json:"link,omitempty"`
	Limit     uint64 `json:"limit,omitempty"`
	Count     int    `json:"count,omitempty"`
	Symmetric bool   `json:"symmetric,omitempty"`
	Inactive  bool   `json:"inactive,omitempty"`
}

type networkAdapterStep struct {
	Input    networkAdapterInput `json:"input"`
	Expected json.RawMessage     `json:"expected"`
}

type networkAdapterCorpus struct {
	Schema         string               `json:"schema"`
	Implementation string               `json:"implementation_sha256"`
	Name           string               `json:"name"`
	Network        *NetworkModelV2      `json:"network"`
	Epoch          uint64               `json:"epoch"`
	CreationError  string               `json:"creation_error"`
	Initial        json.RawMessage      `json:"initial,omitempty"`
	Steps          []networkAdapterStep `json:"steps"`
}

type networkAdapterBuilder struct {
	t      *testing.T
	e      *Environment
	corpus networkAdapterCorpus
}

func newNetworkAdapterCorpus(t *testing.T, name string, network *NetworkModelV2, epoch uint64) *networkAdapterBuilder {
	t.Helper()
	b := &networkAdapterBuilder{t: t, corpus: networkAdapterCorpus{Schema: "gomad.network-adapter-corpus/v1", Implementation: ImplementationSHA256(), Name: name, Network: network, Epoch: epoch}}
	config := testConfig()
	config.Profile, config.Clock = CooperativeProfile, ClockPolicy{EpochNanos: epoch}
	config.NetworkV2, config.Limits.TranscriptBytes = network, 8<<20
	var err error
	b.e, err = NewEnvironment(config)
	if err != nil {
		b.corpus.CreationError = err.Error()
		return b
	}
	b.e.runtime = &runtimeSession{configured: true}
	b.corpus.Initial = b.observe(nil, nil)
	return b
}

func (b *networkAdapterBuilder) observe(data []byte, err error) json.RawMessage {
	b.t.Helper()
	message := ""
	if err != nil {
		message = err.Error()
	}
	value := map[string]any{"error": message, "data": data, "now": b.e.now, "identity": b.e.Identity()}
	if boundary, ok := err.(*BoundaryError); ok {
		value["boundary_kind"], value["boundary_dimension"] = boundary.Kind, boundary.Dimension
	}
	if n := b.e.networkV2; n != nil {
		record, err := n.model.Record()
		if err != nil {
			b.t.Fatal(err)
		}
		value["record"], value["world"], value["requests"], value["next_deadline"] = record, n.world.Snapshot(), n.requests, n.model.NextDeadline()
		evidence, evidenceErr := b.e.NetworkV2Evidence()
		if !bytes.Equal(b.e.NetworkV2Transcript(), evidence) {
			b.t.Fatal("network transcript differs from the evidence document")
		}
		value["evidence"] = evidence
		value["evidence_error"] = ""
		if evidenceErr != nil {
			value["evidence_error"] = evidenceErr.Error()
			if boundary, ok := evidenceErr.(*BoundaryError); ok {
				value["evidence_boundary_kind"], value["evidence_boundary_dimension"] = boundary.Kind, boundary.Dimension
			}
		}
	}
	encoded, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		b.t.Fatal(marshalErr)
	}
	return encoded
}

func (b *networkAdapterBuilder) step(input networkAdapterInput) []byte {
	b.t.Helper()
	var data []byte
	var err error
	switch input.Action {
	case "serve":
		b.e.runtime.configured = !input.Inactive
		if input.Nanos != 0 {
			b.e.now = input.Nanos
		}
		encoded, marshalErr := json.Marshal(bytesOutput{Data: input.Data})
		if marshalErr != nil {
			b.t.Fatal(marshalErr)
		}
		output, callErr := b.e.runtimeNetworkStream(encoded)
		data, err = output.Data, callErr
	case "advance":
		err = b.e.networkV2Advance(input.Nanos)
		if err == nil {
			b.e.now = input.Nanos
		}
	case "link":
		n := b.e.networkV2
		switch input.Link {
		case "partition":
			err = n.model.Partition(n.client.Node, n.server.Node, input.Symmetric)
		case "heal":
			err = n.model.Heal(n.client.Node, n.server.Node, input.Symmetric)
		case "delay":
			err = n.model.Delay(n.client.Node, n.server.Node, input.Nanos, input.Symmetric)
		default:
			b.t.Fatalf("unknown adapter link action %q", input.Link)
		}
	case "idle":
		output, callErr := b.e.networkV2Idle(make([]byte, 16), input.Deadline)
		data, err = output.Data, callErr
	case "runtime_idle":
		frame := make([]byte, 16)
		binary.BigEndian.PutUint32(frame, 1)
		binary.BigEndian.PutUint64(frame[8:], input.Deadline)
		encoded, marshalErr := json.Marshal(bytesOutput{Data: frame})
		if marshalErr != nil {
			b.t.Fatal(marshalErr)
		}
		output, callErr := b.e.runtimeIdle(encoded)
		data, err = output.Data, callErr
	case "poll_clock":
		encoded := json.RawMessage(fmt.Sprintf(`{"subscriptions":[{"userdata":1,"type":0,"clock_id":1,"timeout":%d,"precision":0,"flags":1}]}`, input.Nanos))
		reply, callErr := b.e.Handle(Call{Type: "call", ID: 1, Op: "poll_oneoff", Input: encoded})
		err = callErr
		data = reply.Output
		if reply.Errno != 0 {
			err = fmt.Errorf("errno %d", reply.Errno)
		}
	case "evidence_limit":
		b.e.config.Limits.TranscriptBytes = input.Limit
	case "world_ready":
		for range input.Count {
			var id world.RequestID
			id, err = b.e.networkV2.world.Register(world.Request{Kind: "network", Resource: world.ResourceID{Adapter: "wasm", Kind: "stream", Key: "readiness"}})
			if err != nil {
				break
			}
			_, err = b.e.networkV2.world.Ready(world.Readiness{RequestID: id, At: world.InitialTime + 1, Kind: "network"})
			if err != nil {
				break
			}
		}
		if err == nil {
			q, quiesceErr := b.e.networkV2.world.Quiesce()
			err = quiesceErr
			if err == nil && len(q.Deliveries) != input.Count {
				b.t.Fatalf("readiness deliveries=%d want=%d", len(q.Deliveries), input.Count)
			}
		}
	default:
		b.t.Fatalf("unknown network adapter action %q", input.Action)
	}
	b.corpus.Steps = append(b.corpus.Steps, networkAdapterStep{Input: input, Expected: b.observe(data, err)})
	return data
}

func (b *networkAdapterBuilder) serve(data []byte) []byte {
	b.t.Helper()
	return b.step(networkAdapterInput{Action: "serve", Data: data})
}

func (b *networkAdapterBuilder) pair() (uint64, uint64, uint32) {
	b.t.Helper()
	l := b.serve(streamFrame(1, 1, 0, 0, 0, 0, nil))
	x := b.serve(streamFrame(2, 2, 0, 0, binary.BigEndian.Uint32(l[48:]), 0, nil))
	y := b.serve(streamFrame(3, 3, binary.BigEndian.Uint64(l[16:]), 0, 0, 0, nil))
	return binary.BigEndian.Uint64(x[16:]), binary.BigEndian.Uint64(y[16:]), binary.BigEndian.Uint32(l[48:])
}

func networkAdapterCorpora(t *testing.T) []networkAdapterCorpus {
	var corpora []networkAdapterCorpus
	add := func(name string, network *NetworkModelV2, epoch uint64, run func(*networkAdapterBuilder)) {
		b := newNetworkAdapterCorpus(t, name, network, epoch)
		if b.e != nil {
			run(b)
		}
		corpora = append(corpora, b.corpus)
	}
	for _, limit := range []struct {
		name         string
		bytes, stall uint64
	}{
		{"minimum", 1, 1}, {"maximum", 1 << 30, math.MaxInt64 - 1}, {"zero-bytes", 0, 1}, {"oversize-bytes", 1<<30 + 1, 1},
		{"zero-stall", 1, 0}, {"threshold-overflow", 1, math.MaxInt64}, {"signed-overflow", 1, math.MaxInt64 + 1}, {"unsigned-overflow", math.MaxUint64, math.MaxUint64},
	} {
		config := streamConfig()
		config.Limits.ConnectionBytes, config.Limits.StallNanos = limit.bytes, limit.stall
		add("explicit-limits-"+limit.name, config, 0, func(*networkAdapterBuilder) {})
	}
	for _, epoch := range []uint64{math.MaxInt64, math.MaxInt64 + 1, math.MaxUint64} {
		add(fmt.Sprintf("epoch-overflow-%d", epoch), streamConfig(), epoch, func(*networkAdapterBuilder) {})
	}
	old := streamConfig()
	old.Profile = "gomad.simulation-network-stream/v2"
	add("old-profile-refused", old, 0, func(*networkAdapterBuilder) {})
	add("profile-bound-to-identity", nil, 0, func(*networkAdapterBuilder) {})
	for _, mutation := range []string{"short-frame", "reserved", "opcode", "stale-cancel", "query", "inactive"} {
		add("wire-"+mutation, streamConfig(), 0, func(b *networkAdapterBuilder) {
			frame := streamFrame(1, 1, 0, 0, 0, 0, nil)
			switch mutation {
			case "short-frame":
				frame = make([]byte, 63)
			case "reserved":
				binary.BigEndian.PutUint32(frame[40:], 1)
			case "opcode":
				binary.BigEndian.PutUint32(frame[4:], 99)
			case "stale-cancel":
				frame = streamFrame(13, 1, 0, 0, 0, 0, nil)
			case "query":
				frame = streamFrame(0, 0, 0, 0, 0, 0, nil)
			}
			b.step(networkAdapterInput{Action: "serve", Data: frame, Inactive: mutation == "inactive"})
		})
	}
	add("world-readiness-independent-of-payload", func() *NetworkModelV2 {
		c := streamConfig()
		c.Limits.Bytes, c.Limits.ConnectionBytes, c.Limits.Transitions = 1, 1, 128
		return c
	}(), 0, func(b *networkAdapterBuilder) { b.step(networkAdapterInput{Action: "world_ready", Count: 129}) })
	add("stream-delay-idle-cancel-and-payload-free-evidence", streamConfig(), 0, func(b *networkAdapterBuilder) {
		x, y, _ := b.pair()
		b.step(networkAdapterInput{Action: "link", Link: "delay", Nanos: 100})
		payload := []byte("actual modeled bytes")
		b.serve(streamFrame(5, 4, x, 0, 0, uint32(len(payload)), payload))
		b.serve(streamFrame(4, 5, y, 0, 0, 64, nil))
		idle := b.step(networkAdapterInput{Action: "runtime_idle", Deadline: 10000})
		if !bytes.Equal(idle, []byte{0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 101}) || b.e.now != 101 {
			t.Fatalf("runtime idle callback version/deadline differs: data=%x now=%d", idle, b.e.now)
		}
		poll := streamFrame(12, 5, 0, 0, 0, 64, nil)
		binary.BigEndian.PutUint32(poll[36:], 0)
		got := b.serve(poll)
		if len(got) < 64+len(payload) || !bytes.Equal(got[64:64+len(payload)], payload) {
			t.Fatal("adapter delayed payload was not delivered")
		}
		b.serve(streamFrame(4, 6, y, 0, 0, 16, nil))
		b.serve(streamFrame(13, 6, 0, 0, 0, 0, nil))
	})
	add("poll-read-rejects-short-payload", streamConfig(), 0, func(b *networkAdapterBuilder) {
		x, y, _ := b.pair()
		b.serve(streamFrame(4, 4, y, 0, 0, 16, nil))
		b.serve(streamFrame(5, 5, x, 0, 0, 4, []byte("data")))
		b.serve(streamFrame(12, 4, 0, 0, 0, 0, nil))
	})
	add("final-evidence-typed-capacity", streamConfig(), 0, func(b *networkAdapterBuilder) { b.step(networkAdapterInput{Action: "evidence_limit", Limit: 1}) })
	add("connection-deadline-retains-caller-wire-status", streamConfig(), 0, func(b *networkAdapterBuilder) {
		x, _, _ := b.pair()
		b.serve(streamFrame(11, 4, x, 5, 0, 0, nil))
		b.step(networkAdapterInput{Action: "serve", Nanos: 5, Data: streamFrame(5, 5, x, 0, 0, 1, []byte("x"))})
	})
	reset := streamConfig()
	reset.Faults = []NetworkFaultV2{{Name: "rpc-reset", Action: "reset", AtNanos: 10}}
	add("named-reset-preserves-listener-and-fault-identity", reset, 0, func(b *networkAdapterBuilder) {
		x, _, port := b.pair()
		b.step(networkAdapterInput{Action: "serve", Nanos: 10, Data: streamFrame(5, 4, x, 0, 0, 1, []byte{1})})
		b.serve(streamFrame(2, 5, 0, 0, port, 0, nil))
	})
	for _, deadline := range []uint64{0, 5} {
		config := streamConfig()
		config.Limits.StallNanos = 10
		add(fmt.Sprintf("pending-dial-timeout-%d", deadline), config, 0, func(b *networkAdapterBuilder) {
			l := b.serve(streamFrame(1, 1, 0, 0, 0, 0, nil))
			port := binary.BigEndian.Uint32(l[48:])
			b.step(networkAdapterInput{Action: "link", Link: "partition", Symmetric: true})
			b.serve(streamFrame(2, 2, 0, deadline, port, 0, nil))
			b.step(networkAdapterInput{Action: "serve", Nanos: map[bool]uint64{true: 5, false: 12}[deadline != 0], Data: streamFrame(12, 2, 0, 0, 0, 0, nil)})
			b.step(networkAdapterInput{Action: "link", Link: "heal", Symmetric: true})
			b.serve(streamFrame(2, 3, 0, 0, port, 0, nil))
		})
	}
	for _, scheduled := range []bool{false, true} {
		config := streamConfig()
		config.Limits.StallNanos = 10
		if scheduled {
			config.Faults = []NetworkFaultV2{{Name: "heal", Action: "heal", AtNanos: 100}}
		}
		add(fmt.Sprintf("clock-poll-preserves-dial-deadline-heal-%t", scheduled), config, 0, func(b *networkAdapterBuilder) {
			l := b.serve(streamFrame(1, 1, 0, 0, 0, 0, nil))
			b.step(networkAdapterInput{Action: "link", Link: "partition", Symmetric: true})
			b.serve(streamFrame(2, 2, 0, 5, binary.BigEndian.Uint32(l[48:]), 0, nil))
			b.step(networkAdapterInput{Action: "poll_clock", Nanos: 100})
			b.serve(streamFrame(12, 2, 0, 0, 0, 0, nil))
		})
	}
	for _, heal := range []uint64{0, 100, 111, 112} {
		config := streamConfig()
		config.Limits.StallNanos = 10
		config.Faults = []NetworkFaultV2{{Name: "partition", Action: "partition", AtNanos: 100}}
		if heal != 0 {
			config.Faults = append(config.Faults, NetworkFaultV2{Name: "heal", Action: "heal", AtNanos: heal})
		}
		add(fmt.Sprintf("scheduled-partition-heal-%d", heal), config, 0, func(b *networkAdapterBuilder) {
			x, y, _ := b.pair()
			b.step(networkAdapterInput{Action: "link", Link: "delay", Nanos: 200})
			b.serve(streamFrame(5, 4, x, 0, 0, 1, []byte("x")))
			if heal == 0 {
				for _, now := range []uint64{100, 110, 111} {
					b.step(networkAdapterInput{Action: "advance", Nanos: now})
				}
				return
			}
			b.step(networkAdapterInput{Action: "advance", Nanos: heal})
			b.serve(streamFrame(4, 5, y, 0, 0, 1, nil))
			if heal == 112 {
				return
			}
			b.step(networkAdapterInput{Action: "advance", Nanos: heal + 199})
			frame := streamFrame(12, 5, 0, 0, 0, 1, nil)
			binary.BigEndian.PutUint32(frame[36:], 0)
			b.serve(frame)
			b.step(networkAdapterInput{Action: "advance", Nanos: heal + 200})
			got := b.serve(frame)
			if len(got) != 65 || got[64] != 'x' || binary.BigEndian.Uint32(got[44:]) != 0 {
				t.Fatalf("scheduled adapter heal lost payload: %x", got)
			}
		})
	}
	config := streamConfig()
	config.Limits.StallNanos = 10
	add("stall-timeout-read-and-write-wire-classification", config, 0, func(b *networkAdapterBuilder) {
		x, y, _ := b.pair()
		b.step(networkAdapterInput{Action: "link", Link: "partition", Symmetric: true})
		b.serve(streamFrame(5, 4, x, 0, 0, 1, []byte("x")))
		b.step(networkAdapterInput{Action: "serve", Nanos: 12, Data: streamFrame(4, 5, y, 0, 0, 1, nil)})
		b.serve(streamFrame(5, 6, x, 0, 0, 1, []byte("x")))
	})
	networkAdapterClockCorpora(add)
	return corpora
}

func networkAdapterClockCorpora(add func(string, *NetworkModelV2, uint64, func(*networkAdapterBuilder))) {
	for _, epoch := range []uint64{0, uint64(world.InitialTime), uint64(world.InitialTime) + 100, math.MaxInt64 - 100} {
		ceiling := min(uint64(math.MaxInt64-int64(world.InitialTime)), math.MaxInt64-epoch)
		for _, operation := range []string{"held-write", "pending-dial", "partition-queued"} {
			config := streamConfig()
			config.Limits.StallNanos = ceiling - 1
			add(fmt.Sprintf("clock-ceiling-atomic-%d-%s", epoch, operation), config, epoch, func(b *networkAdapterBuilder) {
				l := b.serve(streamFrame(1, 1, 0, 0, 0, 0, nil))
				port := binary.BigEndian.Uint32(l[48:])
				var x uint64
				if operation != "pending-dial" {
					c := b.serve(streamFrame(2, 2, 0, 0, port, 0, nil))
					x = binary.BigEndian.Uint64(c[16:])
				}
				if operation == "partition-queued" {
					b.step(networkAdapterInput{Action: "link", Link: "delay", Nanos: 10})
					b.serve(streamFrame(5, 3, x, 0, 0, 4, []byte("held")))
					b.step(networkAdapterInput{Action: "link", Link: "partition"})
					return
				}
				b.step(networkAdapterInput{Action: "link", Link: "partition"})
				frame := streamFrame(5, 4, x, 0, 0, 4, []byte("held"))
				if operation == "pending-dial" {
					frame = streamFrame(2, 4, 0, 0, port, 0, nil)
				}
				b.serve(frame)
			})
		}
	}
	for _, epoch := range []uint64{0, uint64(world.InitialTime) - 100, uint64(world.InitialTime) + 100, math.MaxInt64 - 100} {
		ceiling := min(uint64(math.MaxInt64-int64(world.InitialTime)), math.MaxInt64-epoch)
		for _, below := range []uint64{0, 1} {
			for _, dial := range []bool{false, true} {
				config := streamConfig()
				config.Limits.StallNanos = ceiling - below - 2
				add(fmt.Sprintf("clock-ceiling-expiry-%d-below-%d-dial-%t", epoch, below, dial), config, epoch, func(b *networkAdapterBuilder) {
					l := b.serve(streamFrame(1, 1, 0, 0, 0, 0, nil))
					port := binary.BigEndian.Uint32(l[48:])
					var x uint64
					if !dial {
						c := b.serve(streamFrame(2, 2, 0, 0, port, 0, nil))
						x = binary.BigEndian.Uint64(c[16:])
					}
					b.step(networkAdapterInput{Action: "link", Link: "partition"})
					frame := streamFrame(5, 4, x, 0, 0, 4, []byte("held"))
					if dial {
						frame = streamFrame(2, 4, 0, 0, port, 0, nil)
					}
					b.serve(frame)
					b.step(networkAdapterInput{Action: "idle"})
					if dial {
						b.serve(streamFrame(12, 4, 0, 0, 0, 0, nil))
					}
				})
			}
		}
	}
}

func TestNetworkAdapterCorpusIsCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_NETWORK_ADAPTER_CORPUS_REGENERATE") == "1"
	if regenerate {
		if err := os.MkdirAll(networkAdapterCorpusDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]bool{}
	for _, corpus := range networkAdapterCorpora(t) {
		name := corpus.Name + ".json"
		if want[name] {
			t.Fatal("duplicate adapter corpus", name)
		}
		want[name] = true
		data, err := json.Marshal(corpus)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(networkAdapterCorpusDirectory, name)
		if regenerate {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("stale network adapter corpus %s", name)
		}
	}
	entries, err := os.ReadDir(networkAdapterCorpusDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && !want[entry.Name()] {
			t.Error("unexpected adapter corpus", entry.Name())
		}
	}
}

func TestNetworkAdapterRuntimeIdleCallbackVersionAndDeadline(t *testing.T) {
	for _, corpus := range networkAdapterCorpora(t) {
		if corpus.Name == "stream-delay-idle-cancel-and-payload-free-evidence" {
			for _, step := range corpus.Steps {
				if step.Input.Action == "idle" || step.Input.Action == "runtime_idle" {
					if step.Input.Action != "runtime_idle" || step.Input.Deadline != 10000 {
						t.Fatalf("original runtime idle callback input differs: %+v", step.Input)
					}
					return
				}
			}
			t.Fatal("original runtime idle callback step missing")
		}
	}
	t.Fatal("original stream-delay corpus missing")
}

func TestNetworkAdapterCorpus(t *testing.T) {
	pin, err := os.ReadFile("../../../rust/wasmhost/src/network/PINNED_GO_NETWORK_SHA256")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pin, []byte(netmodel.ImplementationSHA256()+"\n")) {
		t.Fatal("Rust network owner pin differs from the Go owner")
	}
	files, err := filepath.Glob(filepath.Join(networkAdapterCorpusDirectory, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("network adapter corpus selection: %v files=%d", err, len(files))
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var corpus networkAdapterCorpus
			if err := json.Unmarshal(data, &corpus); err != nil {
				t.Fatal(err)
			}
			if corpus.Schema != "gomad.network-adapter-corpus/v1" || corpus.Implementation != ImplementationSHA256() {
				t.Fatal("network adapter corpus identity")
			}
			b := newNetworkAdapterCorpus(t, corpus.Name, corpus.Network, corpus.Epoch)
			if b.corpus.CreationError != corpus.CreationError {
				t.Fatalf("creation got %q want %q", b.corpus.CreationError, corpus.CreationError)
			}
			if b.e == nil {
				if len(corpus.Initial) != 0 || len(corpus.Steps) != 0 {
					t.Fatal("rejected creation has observations")
				}
				return
			}
			compareNetworkCorpusJSON(t, b.corpus.Initial, corpus.Initial)
			for _, step := range corpus.Steps {
				b.step(step.Input)
				compareNetworkCorpusJSON(t, b.corpus.Steps[len(b.corpus.Steps)-1].Expected, step.Expected)
			}
		})
	}
}
