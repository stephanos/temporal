package wasi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func brokerTestConfig() BrokerConfig {
	return BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}, {ID: "b", Incarnation: 1}}, Seed: 7, Limits: world.Limits{MaxRequests: 10000, MaxEvents: 10000, MaxQueuedEvents: 1000, MaxTransitions: 40000, MaxPayloadBytes: 32 << 20, MaxStringBytes: 1024}}
}

func reviewBrokerPair(t *testing.T, b *Broker) (netmodel.Domain, netmodel.Domain, uint64, uint64) {
	t.Helper()
	client, _ := b.network.domains(b.Session("a"))
	_, server := b.network.domains(b.Session("b"))
	m := b.network.model
	l := m.Dispatch(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
	c := m.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: l.Local.Port, RequestID: 2})
	s := m.Dispatch(netmodel.Request{Domain: server, Op: "accept", Handle: l.Handle, RequestID: 3})
	if l.Error != "" || c.Error != "" || s.Error != "" || c.Handle == 0 || s.Handle == 0 {
		t.Fatalf("broker pair: listen=%+v dial=%+v accept=%+v", l, c, s)
	}
	return client, server, c.Handle, s.Handle
}

// Recorded Go broker orders for the in-process broker (fn-178.7). Each
// scenario runs scripted node call sequences through the Go broker (sessions
// on goroutines, as helpers drive them) and records every call's output
// buffer or error, the broker World's expanded wire and each node's network
// evidence document. The Rust broker replays the same scripts
// (`wasmhost/src/cluster/handoff/broker/tests.rs`) and must match byte for byte; its network
// evidence is checked in beside the fixture and decoded here.
const brokerOrderDirectory = "../../../rust/wasmhost/tests/testdata/broker_orders"

type brokerOrderRef struct {
	Node string `json:"node"`
	Step int    `json:"step"`
}

type brokerOrderStep struct {
	Op       string          `json:"op"`
	Deadline uint64          `json:"deadline,omitempty"`
	Code     uint32          `json:"code,omitempty"`
	ID       uint64          `json:"id,omitempty"`
	Handle   *brokerOrderRef `json:"handle,omitempty"`
	Port     *brokerOrderRef `json:"port,omitempty"`
	Count    uint32          `json:"count,omitempty"`
	Payload  string          `json:"payload,omitempty"`
	Address  [4]byte         `json:"address"`
	Nanos    uint64          `json:"nanos,omitempty"`
}

type brokerOrderOutput struct {
	DataBase64 string `json:"data_base64,omitempty"`
	Error      string `json:"error,omitempty"`
}

type brokerOrderScenario struct {
	Name     string                         `json:"name"`
	Nodes    []string                       `json:"nodes"`
	Network  *NetworkModelV2                `json:"network,omitempty"`
	Faults   []BrokerNetworkFault           `json:"faults,omitempty"`
	Links    []BrokerLink                   `json:"links,omitempty"`
	Strict   bool                           `json:"strict_topology,omitempty"`
	Steps    map[string][]brokerOrderStep   `json:"steps"`
	Outputs  map[string][]brokerOrderOutput `json:"outputs"`
	World    string                         `json:"world_base64"`
	Networks map[string]string              `json:"network_documents_base64,omitempty"`
}

// TestBrokerOrdersAreCurrent records the Go broker's orders for the Rust
// broker's replay; GOMAD_BROKER_ORDERS_REGENERATE=1 rewrites them.
func TestBrokerOrdersAreCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_BROKER_ORDERS_REGENERATE") == "1"
	for _, scenario := range brokerOrderScenarios() {
		recorded := runBrokerOrder(t, scenario)
		data, err := json.MarshalIndent(recorded, "", " ")
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(brokerOrderDirectory, scenario.Name+".json")
		if regenerate {
			if err := os.MkdirAll(brokerOrderDirectory, 0o755); err != nil {
				t.Fatal(err)
			}
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
			t.Errorf("Go broker order %s changed; review, then rerun with GOMAD_BROKER_ORDERS_REGENERATE=1 and the Rust broker replay", scenario.Name)
		}
	}
}

// TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments decodes the Rust
// broker replay's checked-in network evidence into the Go documents.
func TestBrokerOrdersRustNetworkEvidenceRebuildsGoDocuments(t *testing.T) {
	for _, scenario := range brokerOrderScenarios() {
		if scenario.Network == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(brokerOrderDirectory, scenario.Name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var recorded brokerOrderScenario
		if err := json.Unmarshal(data, &recorded); err != nil {
			t.Fatal(err)
		}
		for _, id := range scenario.Nodes {
			evidence, err := os.ReadFile(filepath.Join(brokerOrderDirectory, scenario.Name+"."+id+".gmnetev1"))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeClusterNetworkEvidence(evidence, ClusterNetworkLimits{Network: scenario.Network.Limits, Instance: *scenario.Network, TranscriptBytes: 8 << 20})
			if err != nil {
				t.Fatalf("%s/%s: %v", scenario.Name, id, err)
			}
			want, err := base64.StdEncoding.DecodeString(recorded.Networks[id])
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(decoded.Document, want) {
				t.Errorf("%s/%s: Rust network evidence differs from the Go broker:\nrust %s\ngo   %s", scenario.Name, id, decoded.Document, want)
			}
		}
	}
}

func brokerOrderScenarios() []brokerOrderScenario {
	idle := func(deadline uint64) brokerOrderStep { return brokerOrderStep{Op: "idle", Deadline: deadline} }
	net := func(code uint32, id uint64) brokerOrderStep { return brokerOrderStep{Op: "net", Code: code, ID: id} }
	at := func(step brokerOrderStep, handle *brokerOrderRef) brokerOrderStep { step.Handle = handle; return step }
	ref := func(node string, step int) *brokerOrderRef { return &brokerOrderRef{node, step} }
	withPayload := func(step brokerOrderStep, payload string) brokerOrderStep {
		step.Payload, step.Count = payload, uint32(len(payload))
		return step
	}
	read := func(id uint64, handle *brokerOrderRef) brokerOrderStep {
		step := at(net(4, id), handle)
		step.Count = 4
		return step
	}
	poll := func(id uint64) brokerOrderStep { step := net(12, id); step.Count = 4; return step }
	dial := func(id uint64, port *brokerOrderRef, address [4]byte) brokerOrderStep {
		step := net(2, id)
		step.Port, step.Address = port, address
		return step
	}
	network := streamConfig()
	network.Limits.StallNanos = 1500
	// a writes "held" inside a partition from 100; its stall limit is first
	// due at 1601. The heal lands before that instant or exactly on it; b wakes
	// at the partition, then at the heal or expiry, and reads again.
	healed := func(name string, healAt uint64) brokerOrderScenario {
		return brokerOrderScenario{
			Name: name, Nodes: []string{"a", "b"}, Network: network,
			Faults: []BrokerNetworkFault{
				{Name: "cut", Action: "partition", From: "a", To: "b", AtNanos: 100, Symmetric: true},
				{Name: "mend", Action: "heal", From: "a", To: "b", AtNanos: healAt, Symmetric: true},
			},
			Steps: map[string][]brokerOrderStep{
				"a": {net(1, 1), idle(50), at(net(3, 2), ref("a", 0)), idle(150), withPayload(at(net(5, 3), ref("a", 2)), "held"), idle(0), poll(3), idle(4000)},
				"b": {dial(1, ref("a", 0), [4]byte{10, 0, 0, 2}), idle(200), read(2, ref("b", 0)), idle(0), idle(0), read(3, ref("b", 0)), poll(4), idle(5000)},
			},
		}
	}
	return []brokerOrderScenario{
		{
			Name: "finish-with-pending-peer-read", Nodes: []string{"a", "b"}, Network: network,
			Steps: map[string][]brokerOrderStep{
				"a": {net(1, 1), idle(10), at(net(3, 2), ref("a", 0)), withPayload(at(net(5, 3), ref("a", 2)), "lost")},
				"b": {dial(1, ref("a", 0), [4]byte{10, 0, 0, 2}), read(2, ref("b", 0)), idle(1000), poll(2)},
			},
		},
		healed("heal-before-stall-expiry", 1000),
		healed("heal-at-stall-expiry-instant", 1601),
		{
			Name: "idle-three-equal-time", Nodes: []string{"a", "b", "c"},
			Steps: map[string][]brokerOrderStep{
				"a": {idle(1000), idle(5000)},
				"b": {idle(3000), idle(4000), idle(4000)},
				"c": {idle(3000), idle(0)},
			},
		},
		{
			Name: "ping-pong-partition-heal", Nodes: []string{"a", "b"}, Network: network,
			Faults: []BrokerNetworkFault{
				{Name: "cut", Action: "partition", From: "a", To: "b", AtNanos: 2000, Symmetric: true},
				{Name: "mend", Action: "heal", From: "a", To: "b", AtNanos: 6000, Symmetric: true},
			},
			Steps: map[string][]brokerOrderStep{
				"a": {
					net(1, 1), idle(10), idle(20), at(net(3, 2), ref("a", 0)), read(3, ref("a", 3)),
					withPayload(at(net(5, 4), ref("a", 3)), "pong"), idle(3000),
					withPayload(at(net(5, 5), ref("a", 3)), "late"), idle(7000), idle(9000),
				},
				"b": {
					idle(5), dial(1, ref("a", 0), [4]byte{10, 0, 0, 2}), withPayload(at(net(5, 2), ref("b", 1)), "ping"),
					idle(15), read(3, ref("b", 1)), idle(2500), read(4, ref("b", 1)), idle(0), poll(4), idle(9000),
				},
			},
		},
		{
			Name: "stall-expiry-inside-partition", Nodes: []string{"a", "b"}, Network: network,
			Faults: []BrokerNetworkFault{{Name: "cut", Action: "partition", From: "a", To: "b", AtNanos: 100, Symmetric: true}},
			Steps: map[string][]brokerOrderStep{
				"a": {net(1, 1), idle(50), at(net(3, 2), ref("a", 0)), idle(150), withPayload(at(net(5, 3), ref("a", 2)), "held"), idle(0), poll(3), idle(4000)},
				"b": {dial(1, ref("a", 0), [4]byte{10, 0, 0, 2}), idle(200), read(2, ref("b", 0)), idle(0), poll(2), idle(5000)},
			},
		},
		{
			// The instance clocks diverge (a reads its clock forward), and the
			// shared owner refuses b's earlier time: "network clock moved
			// backwards", the known Go broker behaviour the Rust owner mirrors.
			Name: "clock-divergence-on-shared-owner", Nodes: []string{"a", "b"}, Network: network,
			Steps: map[string][]brokerOrderStep{
				"a": {net(1, 1), {Op: "advance_now", Nanos: 500}, net(1, 2), idle(1000)},
				"b": {net(1, 1), idle(1000)},
			},
		},
	}
}

// brokerOrderData builds a step's call data. It runs on session goroutines,
// so it reports an unresolved reference instead of failing the test itself.
func brokerOrderData(step brokerOrderStep, outputs map[string][]brokerOrderOutput) ([]byte, error) {
	if step.Op == "idle" {
		data := make([]byte, 16)
		binary.BigEndian.PutUint32(data, 1)
		binary.BigEndian.PutUint64(data[8:], step.Deadline)
		return data, nil
	}
	var unresolved error
	resolve := func(ref *brokerOrderRef, at int, width int) uint64 {
		if ref == nil {
			return 0
		}
		var data []byte
		var err error
		if ref.Step < len(outputs[ref.Node]) {
			data, err = base64.StdEncoding.DecodeString(outputs[ref.Node][ref.Step].DataBase64)
		}
		if err != nil || len(data) < 64 {
			unresolved = fmt.Errorf("unresolved reference %+v", *ref)
			return 0
		}
		if width == 8 {
			return binary.BigEndian.Uint64(data[at:])
		}
		return uint64(binary.BigEndian.Uint32(data[at:]))
	}
	payload := []byte(step.Payload)
	if step.Code == 4 || step.Code == 12 {
		payload = make([]byte, step.Count)
	}
	data := streamFrame(step.Code, step.ID, resolve(step.Handle, 16, 8), 0, uint32(resolve(step.Port, 48, 4)), step.Count, payload)
	if step.Code == 12 {
		binary.BigEndian.PutUint32(data[36:], 0)
	}
	copy(data[56:60], step.Address[:])
	return data, unresolved
}

// runBrokerOrder runs one scenario through the Go broker.
func runBrokerOrder(t *testing.T, scenario brokerOrderScenario) brokerOrderScenario {
	t.Helper()
	config := brokerTestConfig()
	config.Nodes = nil
	for _, id := range scenario.Nodes {
		config.Nodes = append(config.Nodes, BrokerNode{ID: id, Incarnation: 1})
	}
	config.Network, config.Faults, config.Links, config.StrictTopology = scenario.Network, scenario.Faults, scenario.Links, scenario.Strict
	broker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	scenario.Outputs = map[string][]brokerOrderOutput{}
	documents := map[string][]byte{}
	var group sync.WaitGroup
	for _, id := range scenario.Nodes {
		e := streamEnvironment(t, scenario.Network)
		e.now = 1
		if scenario.Network == nil {
			e.networkV2 = nil
		}
		session := broker.Session(id)
		group.Add(1)
		go func() {
			defer group.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			record := func(output brokerOrderOutput) {
				mu.Lock()
				scenario.Outputs[id] = append(scenario.Outputs[id], output)
				mu.Unlock()
			}
			if err := session.Activate(ctx); err != nil {
				record(brokerOrderOutput{Error: err.Error()})
			}
			calls := uint64(0)
			for _, step := range scenario.Steps[id] {
				if step.Op == "advance_now" {
					e.now += step.Nanos
					record(brokerOrderOutput{})
					continue
				}
				mu.Lock()
				data, err := brokerOrderData(step, scenario.Outputs)
				mu.Unlock()
				if err != nil {
					t.Error(err)
					break
				}
				op := "runtime_idle"
				if step.Op == "net" {
					op = "runtime_net_stream"
				}
				input, err := json.Marshal(bytesOutput{Data: data})
				if err != nil {
					t.Error(err)
					break
				}
				calls++
				call := Call{Type: "call", ID: calls, Op: op, Input: input}
				reply, err := session.Handle(ctx, e, call)
				if err == nil {
					err = session.Delivered(call)
				}
				if err != nil {
					record(brokerOrderOutput{Error: err.Error()})
					break
				}
				var output bytesOutput
				if err := json.Unmarshal(reply.Output, &output); err != nil {
					t.Error(err)
				}
				record(brokerOrderOutput{DataBase64: base64.StdEncoding.EncodeToString(output.Data)})
			}
			if e.networkV2 != nil {
				document, err := e.NetworkV2Evidence()
				if err != nil {
					t.Error(err)
				}
				mu.Lock()
				documents[id] = document
				mu.Unlock()
			}
			if err := session.Finished(); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	envelope, err := broker.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := world.DecodeCompressedSnapshotState(envelope, world.CompressedSnapshotProfile())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(envelope[64:]))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := io.ReadAll(reader)
	if err != nil || len(snapshot.Transitions) == 0 {
		t.Fatal("broker World evidence", err)
	}
	scenario.World = base64.StdEncoding.EncodeToString(expanded)
	if len(documents) != 0 {
		scenario.Networks = map[string]string{}
		for id, document := range documents {
			scenario.Networks[id] = base64.StdEncoding.EncodeToString(document)
		}
	}
	return scenario
}
