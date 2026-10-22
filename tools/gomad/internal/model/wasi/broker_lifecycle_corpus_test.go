package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

const brokerLifecycleCorpusDirectory = "../../../rust/wasmhost/tests/testdata/broker_lifecycle_cases"

type brokerLifecycleInput struct {
	Action   string `json:"action"`
	Session  string `json:"session,omitempty"`
	KeepAs   string `json:"keep_as,omitempty"`
	Canceled bool   `json:"canceled,omitempty"`
	Call     Call   `json:"call,omitempty"`
	Count    int    `json:"count,omitempty"`
	Deadline uint64 `json:"deadline,omitempty"`
}

type brokerLifecycleStep struct {
	Input    brokerLifecycleInput `json:"input"`
	Expected json.RawMessage      `json:"expected"`
}

type brokerLifecycleCorpus struct {
	Schema                string                `json:"schema"`
	Implementation        string                `json:"implementation_sha256"`
	Name                  string                `json:"name"`
	Config                BrokerConfig          `json:"config"`
	CreationError         string                `json:"creation_error"`
	CreationWorldCapacity bool                  `json:"creation_world_capacity"`
	Initial               json.RawMessage       `json:"initial,omitempty"`
	Steps                 []brokerLifecycleStep `json:"steps"`
}

type brokerLifecycleBuilder struct {
	t        *testing.T
	broker   *Broker
	sessions map[string]*BrokerSession
	corpus   brokerLifecycleCorpus
}

func newBrokerLifecycleCorpus(t *testing.T, name string, config BrokerConfig) *brokerLifecycleBuilder {
	t.Helper()
	b := &brokerLifecycleBuilder{t: t, sessions: map[string]*BrokerSession{}, corpus: brokerLifecycleCorpus{Schema: "gomad.broker-lifecycle-corpus/v1", Implementation: ImplementationSHA256(), Name: name, Config: config}}
	var err error
	b.broker, err = NewBroker(config)
	b.corpus.CreationWorldCapacity = errors.Is(err, world.ErrCapacity)
	if err != nil {
		b.corpus.CreationError = err.Error()
		return b
	}
	for _, id := range b.broker.order {
		b.sessions[id] = b.broker.Session(id)
	}
	b.corpus.Initial = b.observe(nil, nil)
	return b
}

func (b *brokerLifecycleBuilder) observe(data []byte, err error) json.RawMessage {
	b.t.Helper()
	message := ""
	if err != nil {
		message = err.Error()
	}
	sessions := map[string]any{}
	for name, s := range b.sessions {
		sessions[name] = map[string]any{"node": s.Node, "state": s.state, "slot": s.slot, "request": s.request, "arrivals": s.arrivals, "pending": s.pending}
	}
	active, preparing := "", ""
	if b.broker.active != nil {
		active = b.broker.active.Node.ID
	}
	if b.broker.preparing != nil {
		preparing = b.broker.preparing.Node.ID
	}
	value := map[string]any{"error": message, "data": data, "sessions": sessions, "active": active, "preparing": preparing, "now": b.broker.Current(), "cardinality": b.broker.DiagnosticCardinality()}
	value["world_capacity"] = errors.Is(err, world.ErrCapacity)
	value["context_canceled"] = errors.Is(err, context.Canceled)
	if boundary, ok := err.(*BoundaryError); ok {
		value["boundary_kind"], value["boundary_dimension"] = boundary.Kind, boundary.Dimension
	}
	encoded, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		b.t.Fatal(marshalErr)
	}
	return encoded
}

func (b *brokerLifecycleBuilder) step(input brokerLifecycleInput) {
	b.t.Helper()
	s := b.sessions[input.Session]
	var data []byte
	var err error
	var idleNow uint64
	switch input.Action {
	case "prepare":
		ctx, cancel := context.WithCancel(b.t.Context())
		if input.Canceled {
			cancel()
		}
		err = s.Prepare(ctx)
		cancel()
	case "prepared":
		s.Prepared()
	case "activate":
		err = s.Activate(b.t.Context())
	case "finished":
		err = s.Finished()
	case "stop":
		err = b.broker.Stop(s.Node.ID)
	case "crash":
		err = b.broker.Crash(s.Node.ID)
	case "restart":
		var next *BrokerSession
		next, err = b.broker.Restart(s.Node.ID)
		if err == nil {
			b.sessions[input.KeepAs] = next
		}
	case "delivered":
		err = s.Delivered(input.Call)
	case "begin_call":
		err = s.beginCall(&Environment{}, input.Call)
	case "finish_call":
		err = s.finishCall(&Environment{}, input.Call)
	case "prepare_idle":
		_, err = s.prepareIdle(input.Deadline)
	case "idle":
		idleNow, err = s.idle(b.t.Context(), input.Deadline)
	case "diagnostic_requests":
		for range input.Count {
			_, err = b.broker.world.Register(world.Request{Kind: "diagnostic", Resource: world.ResourceID{Adapter: "broker", Kind: "call", Key: "a"}})
			if err != nil {
				break
			}
		}
	case "diagnostics":
		data = b.broker.Diagnostics()
	case "encode_world":
		data, err = b.broker.EncodeWorldEvidence()
	default:
		b.t.Fatalf("unknown broker lifecycle action %q", input.Action)
	}
	expected := b.observe(data, err)
	if input.Action == "idle" || input.Action == "encode_world" && err == nil {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(expected, &fields); err != nil {
			b.t.Fatal(err)
		}
		if input.Action == "idle" {
			fields["idle_now"], err = json.Marshal(idleNow)
		} else {
			decoded, decodeErr := world.DecodeCompressedSnapshot(data, world.CompressedSnapshotProfile())
			if decodeErr != nil {
				b.t.Fatalf("broker corpus World evidence did not decode: %v", decodeErr)
			}
			fields["decoded_world"], err = json.Marshal(decoded)
		}
		if err != nil {
			b.t.Fatal(err)
		}
		expected, err = json.Marshal(fields)
		if err != nil {
			b.t.Fatal(err)
		}
	}
	b.corpus.Steps = append(b.corpus.Steps, brokerLifecycleStep{Input: input, Expected: expected})
}

func brokerLifecycleCorpora(t *testing.T) []brokerLifecycleCorpus {
	var corpora []brokerLifecycleCorpus
	add := func(name string, config BrokerConfig, run func(*brokerLifecycleBuilder)) {
		b := newBrokerLifecycleCorpus(t, name, config)
		if b.broker != nil {
			run(b)
		}
		corpora = append(corpora, b.corpus)
	}
	add("preparation-cancel-release-and-stale-slot", brokerTestConfig(), func(b *brokerLifecycleBuilder) {
		for _, step := range []brokerLifecycleInput{
			{Action: "prepare", Session: "b"}, {Action: "prepare", Session: "a", Canceled: true}, {Action: "prepared", Session: "b"}, {Action: "prepared", Session: "b"},
			{Action: "prepare", Session: "a"}, {Action: "finished", Session: "a"}, {Action: "prepare", Session: "b"}, {Action: "prepared", Session: "b"}, {Action: "activate", Session: "b"}, {Action: "prepare", Session: "a"},
		} {
			b.step(step)
		}
	})
	for _, kind := range []string{"crash", "stop"} {
		add("revoked-completion-and-reap-"+kind, brokerTestConfig(), func(b *brokerLifecycleBuilder) {
			b.step(brokerLifecycleInput{Action: "begin_call", Session: "a", Call: Call{ID: 1, Op: "fd_write", Input: json.RawMessage(`{"fd":1,"data_base64":"eA=="}`)}})
			b.step(brokerLifecycleInput{Action: kind, Session: "a"})
			for _, op := range []string{"fd_write", "runtime_idle", "runtime_net_stream"} {
				b.step(brokerLifecycleInput{Action: "delivered", Session: "a", Call: Call{ID: 1, Op: op}})
			}
			b.step(brokerLifecycleInput{Action: "begin_call", Session: "a", Call: Call{ID: 2, Op: "fd_write"}})
			b.step(brokerLifecycleInput{Action: "finish_call", Session: "a", Call: Call{ID: 1, Op: "runtime_net_stream"}})
			b.step(brokerLifecycleInput{Action: "prepare_idle", Session: "a", Deadline: 1000})
			b.step(brokerLifecycleInput{Action: "finished", Session: "a"})
			b.step(brokerLifecycleInput{Action: "delivered", Session: "a", Call: Call{ID: 1, Op: "runtime_idle"}})
			b.step(brokerLifecycleInput{Action: "idle", Session: "b", Deadline: 1000})
		})
	}
	add("stale-incarnation-completion-after-restart", brokerTestConfig(), func(b *brokerLifecycleBuilder) {
		for _, step := range []brokerLifecycleInput{
			{Action: "activate", Session: "a"}, {Action: "crash", Session: "b"}, {Action: "activate", Session: "b"}, {Action: "finished", Session: "b"},
			{Action: "restart", Session: "b", KeepAs: "next"}, {Action: "delivered", Session: "b", Call: Call{ID: 1}},
			{Action: "finished", Session: "a"}, {Action: "activate", Session: "next"}, {Action: "finished", Session: "next"},
		} {
			b.step(step)
		}
	})
	config := brokerTestConfig()
	config.Diagnostics = true
	config.Limits.MaxTransitions = 10000
	add("bounded-diagnostics-and-atomic-world-capacity", config, func(b *brokerLifecycleBuilder) {
		b.step(brokerLifecycleInput{Action: "diagnostic_requests", Count: 10000})
		b.step(brokerLifecycleInput{Action: "diagnostic_requests", Count: 1})
		b.step(brokerLifecycleInput{Action: "diagnostics"})
	})
	profile := BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, WorldProfile: world.CompressedSnapshotSchema, Limits: world.CompressedSnapshotProfile().Limits}
	oversize := profile
	oversize.Limits.MaxRequests++
	add("compressed-profile-oversize-refused", oversize, func(*brokerLifecycleBuilder) {})
	unknown := profile
	unknown.WorldProfile = "unknown"
	add("unknown-world-profile-refused", unknown, func(*brokerLifecycleBuilder) {})
	add("compressed-world-single-terminal-owner", profile, func(b *brokerLifecycleBuilder) {
		for _, step := range []brokerLifecycleInput{{Action: "encode_world"}, {Action: "finished", Session: "a"}, {Action: "encode_world"}, {Action: "encode_world"}, {Action: "restart", Session: "a", KeepAs: "next"}} {
			b.step(step)
		}
	})
	profile.WorldProfile = ""
	add("absent-profile-binary-owner-codec", profile, func(b *brokerLifecycleBuilder) {
		b.step(brokerLifecycleInput{Action: "finished", Session: "a"})
		b.step(brokerLifecycleInput{Action: "encode_world"})
	})
	simulation := gomadsim.DefaultLimits()
	implicit := BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, Limits: world.Limits{MaxRequests: simulation.NetworkTransitions * 2, MaxEvents: simulation.NetworkTransitions * 2, MaxQueuedEvents: min(netmodel.MaximumPendingOperations, simulation.NetworkTransitions*2), MaxTransitions: simulation.NetworkTransitions * 8, MaxPayloadBytes: simulation.ObservationBytes, MaxStringBytes: 1024}}
	add("implicit-profile-default-simulation-limits", implicit, func(*brokerLifecycleBuilder) {})
	return corpora
}

func TestBrokerLifecycleCorpusIsCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_BROKER_LIFECYCLE_CORPUS_REGENERATE") == "1"
	if regenerate {
		if err := os.MkdirAll(brokerLifecycleCorpusDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]bool{}
	for _, corpus := range brokerLifecycleCorpora(t) {
		name := corpus.Name + ".json"
		if want[name] {
			t.Fatal("duplicate broker corpus", name)
		}
		want[name] = true
		data, err := json.Marshal(corpus)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(brokerLifecycleCorpusDirectory, name)
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
			t.Errorf("stale broker lifecycle corpus %s", name)
		}
	}
	entries, err := os.ReadDir(brokerLifecycleCorpusDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && !want[entry.Name()] {
			t.Error("unexpected broker corpus", entry.Name())
		}
	}
}

func TestBrokerLifecycleCorpus(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(brokerLifecycleCorpusDirectory, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("broker lifecycle corpus selection: %v files=%d", err, len(files))
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var corpus brokerLifecycleCorpus
			if err := json.Unmarshal(data, &corpus); err != nil {
				t.Fatal(err)
			}
			if corpus.Schema != "gomad.broker-lifecycle-corpus/v1" || corpus.Implementation != ImplementationSHA256() {
				t.Fatal("broker lifecycle corpus identity")
			}
			b := newBrokerLifecycleCorpus(t, corpus.Name, corpus.Config)
			if b.corpus.CreationError != corpus.CreationError {
				t.Fatalf("creation got %q want %q", b.corpus.CreationError, corpus.CreationError)
			}
			if b.corpus.CreationWorldCapacity != corpus.CreationWorldCapacity {
				t.Fatalf("creation World capacity got %t want %t", b.corpus.CreationWorldCapacity, corpus.CreationWorldCapacity)
			}
			if b.broker == nil {
				if len(corpus.Steps) != 0 || len(corpus.Initial) != 0 {
					t.Fatal("rejected creation has observations")
				}
				return
			}
			compareNetworkCorpusJSON(t, b.corpus.Initial, corpus.Initial)
			for index, step := range corpus.Steps {
				t.Logf("step %d: %s", index, step.Input.Action)
				b.step(step.Input)
				compareNetworkCorpusJSON(t, b.corpus.Steps[len(b.corpus.Steps)-1].Expected, step.Expected)
			}
		})
	}
}
