package wasi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func brokerTestConfig() BrokerConfig {
	return BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}, {ID: "b", Incarnation: 1}}, Seed: 7, Limits: world.Limits{MaxRequests: 10000, MaxEvents: 10000, MaxQueuedEvents: 1000, MaxTransitions: 40000, MaxPayloadBytes: 32 << 20, MaxStringBytes: 1024}}
}

func TestBrokerBoundedDiagnosticsAndAtomicWorldCapacity(t *testing.T) {
	config := brokerTestConfig()
	config.Diagnostics = true
	config.Limits.MaxTransitions = 10000
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request := world.Request{Kind: "diagnostic", Resource: world.ResourceID{Adapter: "broker", Kind: "call", Key: "a"}}
	for i := 0; i < 10000; i++ {
		if _, err := b.world.Register(request); err != nil {
			t.Fatal(err)
		}
	}
	before := b.DiagnosticCardinality()
	if _, err := b.world.Register(request); !errors.Is(err, world.ErrCapacity) {
		t.Fatalf("world transition capacity did not fail typed: %v", err)
	}
	if after := b.DiagnosticCardinality(); after != before {
		t.Fatal("capacity rejection or cardinality read mutated World")
	}
	data := b.Diagnostics()
	if len(data) == 0 || len(data) > 2048 || before.Requests != 10000 || before.Transitions != 10000 {
		t.Fatal("bounded infrastructure diagnostic expanded full canonical World")
	}
	if after := b.DiagnosticCardinality(); after != before {
		t.Fatal("live diagnostic changed global order or time")
	}
}

func TestBrokerPreparationCancellationAndFailureRelease(t *testing.T) {
	b, err := NewBroker(brokerTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	a, peer := b.Session("a"), b.Session("b")
	if err := peer.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := a.Prepare(ctx); err != context.Canceled {
		t.Fatalf("canceled preparation: %v", err)
	}
	if b.preparing != peer || b.active != a {
		t.Fatal("preparation changed visible continuation ordering")
	}
	peer.Prepared()
	peer.Prepared()
	if err := a.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := a.Finished(); err != nil {
		t.Fatal(err)
	}
	if b.preparing != nil {
		t.Fatal("failed helper retained compilation slot")
	}
	if err := peer.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	peer.Prepared()
	if err := peer.Activate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := a.Prepare(t.Context()); err == nil {
		t.Fatal("stale incarnation obtained preparation slot")
	}
}

func TestBrokerRevokedInFlightCompletionRetainsTypedBoundary(t *testing.T) {
	b, err := NewBroker(brokerTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	s := b.Session("a")
	s.request = 1
	if err := b.Stop("a"); err != nil {
		t.Fatal(err)
	}
	var boundary *BoundaryError
	if err := s.Delivered(Call{ID: 1, Op: "fd_write"}); !errors.As(err, &boundary) || boundary.Kind != "stop" {
		t.Fatalf("revoked in-flight reply lost typed interruption: %v", err)
	}
	for _, op := range []string{"runtime_idle", "runtime_net_stream"} {
		if err := s.Delivered(Call{ID: 1, Op: op}); !errors.As(err, &boundary) || boundary.Kind != "stop" {
			t.Fatalf("revoked %s completion lost typed interruption: %v", op, err)
		}
	}
	if err := s.beginCall(&Environment{}, Call{ID: 2, Op: "fd_write"}); !errors.As(err, &boundary) {
		t.Fatalf("revoked request lost boundary: %v", err)
	}
	if err := s.finishCall(&Environment{}, Call{ID: 1, Op: "runtime_net_stream"}); !errors.As(err, &boundary) {
		t.Fatalf("revoked readiness lost boundary: %v", err)
	}
	if _, err := s.prepareIdle(1000); !errors.As(err, &boundary) || s.state != "revoked" {
		t.Fatalf("revoked participant became idle: state=%s err=%v", s.state, err)
	}
	if err := s.Finished(); err != nil {
		t.Fatal(err)
	}
	if err := s.Delivered(Call{ID: 1, Op: "runtime_idle"}); !errors.As(err, &boundary) || boundary.Kind != "invalid" {
		t.Fatalf("finished incarnation retained live boundary: %v", err)
	}
}

func TestBrokerRevocationReleasesSurvivorProgressAfterReap(t *testing.T) {
	for _, kind := range []string{"crash", "stop"} {
		t.Run(kind, func(t *testing.T) {
			b, err := NewBroker(brokerTestConfig())
			if err != nil {
				t.Fatal(err)
			}
			old := b.Session("a")
			if kind == "crash" {
				err = b.Crash("a")
			} else {
				err = b.Stop("a")
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.Current() != 1 || b.active != old {
				t.Fatal("revocation released continuation before helper reap")
			}
			if err := old.Finished(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			now, err := b.Session("b").idle(ctx, 1000)
			if err != nil || now != 1000 {
				t.Fatalf("surviving timer blocked without restart: now=%d err=%v", now, err)
			}
		})
	}
}

func TestBrokerStaleLifecycleAndContinuationGate(t *testing.T) {
	b, err := NewBroker(brokerTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	a, old := b.Session("a"), b.Session("b")
	if err = a.Activate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = b.Crash("b"); err != nil {
		t.Fatal(err)
	}
	if err = old.Activate(t.Context()); err == nil {
		t.Fatal("stale incarnation authorized")
	}
	if err = old.Finished(); err != nil {
		t.Fatal(err)
	}
	next, err := b.Restart("b")
	if err != nil || next.Node.Incarnation != 2 {
		t.Fatalf("restart %v %v", next, err)
	}
	if err = old.Delivered(Call{ID: 1}); err == nil {
		t.Fatal("stale completion accepted")
	}
	if err = a.Finished(); err != nil {
		t.Fatal(err)
	}
	if err = next.Activate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = next.Finished(); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerCompressedWorldProfileAdmissionAndTerminalOwnership(t *testing.T) {
	config := BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, WorldProfile: world.CompressedSnapshotSchema, Limits: world.CompressedSnapshotProfile().Limits}
	oversize := config
	oversize.Limits.MaxRequests++
	if broker, err := NewBroker(oversize); !errors.Is(err, world.ErrCapacity) || broker != nil {
		t.Fatalf("oversize configured profile admitted: %v", err)
	}
	unknown := config
	unknown.WorldProfile = "unknown"
	if broker, err := NewBroker(unknown); err == nil || broker != nil {
		t.Fatal("unknown World profile admitted")
	}
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.EncodeWorldEvidence(); err == nil {
		t.Fatal("live participant World was detached")
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	before := b.DiagnosticCardinality()
	data, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeCompressedSnapshot(data, world.CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
	if b.DiagnosticCardinality() != before {
		t.Fatal("terminal ownership transfer changed diagnostic cardinality")
	}
	if _, err := b.EncodeWorldEvidence(); err == nil {
		t.Fatal("terminal World encoded twice")
	}
	if _, err := b.Restart("a"); err == nil {
		t.Fatal("terminal detached World restarted")
	}
}

func TestBrokerAbsentWorldProfileUsesBinaryOwnerCodec(t *testing.T) {
	b, err := NewBroker(BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, Limits: world.CompressedSnapshotProfile().Limits})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	data, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeCompressedSnapshot(data, world.CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerImplicitProfileAdmitsDefaultSimulationLimits(t *testing.T) {
	simulation := gomad3sim.DefaultLimits()
	limits := world.Limits{
		MaxRequests: simulation.NetworkTransitions * 2, MaxEvents: simulation.NetworkTransitions * 2,
		MaxQueuedEvents: min(netmodel.MaximumPendingOperations, simulation.NetworkTransitions*2), MaxTransitions: simulation.NetworkTransitions * 8,
		MaxPayloadBytes: simulation.ObservationBytes, MaxStringBytes: 1024,
	}
	if _, err := NewBroker(BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, Limits: limits}); err != nil {
		t.Fatalf("implicit World profile rejected default simulation limits: %v", err)
	}
}

func TestBrokerLeapPreservesEarlierDialDeadline(t *testing.T) {
	for _, scheduledHeal := range []bool{false, true} {
		t.Run(strconv.FormatBool(scheduledHeal), func(t *testing.T) {
			config := brokerTestConfig()
			config.Network = streamConfig()
			config.Network.Limits.StallNanos = 10
			if scheduledHeal {
				config.Faults = []BrokerNetworkFault{{Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: 100, Symmetric: true}}
			}
			b, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			client, _ := b.network.domains(b.Session("a"))
			_, server := b.network.domains(b.Session("b"))
			m := b.network.model
			l := m.Dispatch(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
			if l.Error != "" {
				t.Fatal(l)
			}
			if err := b.ChangeLink("partition", "a", "b", 0, true); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: l.Local.Port, Deadline: 5, RequestID: 2}); r.State != "pending" {
				t.Fatal(r)
			}
			if err := b.network.advance(100); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "poll", RequestID: 2}); r.Error != "context deadline exceeded" {
				t.Fatalf("later stall overtook earlier caller deadline: %+v", r)
			}
		})
	}
}

func TestBrokerLinkGroupRejectionIsAtomic(t *testing.T) {
	for _, scheduled := range []bool{false, true} {
		for _, action := range []string{"partition", "heal", "delay"} {
			t.Run(fmt.Sprintf("%s/scheduled=%v", action, scheduled), func(t *testing.T) {
				config := brokerTestConfig()
				config.Network = streamConfig()
				config.Network.Limits.Transitions = 1
				if scheduled {
					config.Faults = []BrokerNetworkFault{{Name: "change", Action: action, From: "a", To: "b", AtNanos: 100, DelayNanos: 10, Symmetric: true}}
				}
				b, err := NewBroker(config)
				if err != nil {
					t.Fatal(err)
				}
				before, err := b.NetworkRecord()
				if err != nil {
					t.Fatal(err)
				}
				if scheduled {
					err = b.network.advance(100)
				} else {
					err = b.ChangeLink(action, "a", "b", 10, true)
				}
				if err != netmodel.ErrResourceExhausted {
					t.Fatalf("%s scheduled=%v capacity error=%v", action, scheduled, err)
				}
				after, err := b.NetworkRecord()
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("%s scheduled=%v rejected group changed links or history", action, scheduled)
				}
			})
		}
	}
}

func TestBrokerDirectionalGroupChangesPreserveReverseLinks(t *testing.T) {
	config := brokerTestConfig()
	config.Network = streamConfig()
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"partition", "delay", "heal"} {
		if err := b.ChangeLink(action, "a", "b", 10, false); err != nil {
			t.Fatal(err)
		}
		snapshot, err := b.NetworkSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		var changed, reverse int
		for _, link := range snapshot.Links {
			if link.From[0] == 'a' && link.To[0] == 'b' {
				changed++
				if link.Enabled != (action == "heal") || action != "partition" && link.DelayNanos != 10 {
					t.Fatalf("directional %s changed wrong state: %+v", action, link)
				}
			}
			if link.From[0] == 'b' && link.To[0] == 'a' {
				reverse++
				if !link.Enabled || link.DelayNanos != 0 {
					t.Fatalf("directional %s changed reverse link: %+v", action, link)
				}
			}
		}
		if changed != 4 || reverse != 4 {
			t.Fatalf("logical link expansion changed=%d reverse=%d", changed, reverse)
		}
	}
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

func TestBrokerLinkGroupReservationRejectionAndExpiry(t *testing.T) {
	for _, transitions := range []uint64{12, 13} {
		t.Run(strconv.FormatUint(transitions, 10), func(t *testing.T) {
			config := brokerTestConfig()
			config.Network = streamConfig()
			config.Network.Limits.Transitions, config.Network.Limits.StallNanos = transitions, 10
			b, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			client, _, handle, _ := reviewBrokerPair(t, b)
			if err := b.ChangeLink("delay", "a", "b", 200, true); err != nil {
				t.Fatal(err)
			}
			m := b.network.model
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "write", Handle: handle, RequestID: 4, Bytes: []byte("held")}); r.Error != "" || r.Count != 4 {
				t.Fatal(r)
			}
			before, err := b.NetworkRecord()
			if err != nil {
				t.Fatal(err)
			}
			err = b.ChangeLink("partition", "a", "b", 0, true)
			if transitions == 12 {
				if err != netmodel.ErrResourceExhausted {
					t.Fatalf("reservation admission error=%v", err)
				}
				after, err := b.NetworkRecord()
				if err != nil || !reflect.DeepEqual(before, after) || m.NextDeadline() != 0 {
					t.Fatalf("rejection mutated links, bytes, history or expiry: %v", err)
				}
				if err := b.ChangeLink("delay", "a", "b", 200, true); err != nil {
					t.Fatalf("rejected group leaked reservation: %v", err)
				}
				return
			}
			if err != nil || m.NextDeadline() != 12 {
				t.Fatalf("group failed to reserve pair expiry: %v next=%d", err, m.NextDeadline())
			}
			if err := b.ChangeLink("delay", "a", "b", 1, true); err != netmodel.ErrResourceExhausted {
				t.Fatalf("competing action consumed terminal capacity: %v", err)
			}
			if err := b.network.advance(12); err != nil {
				t.Fatal(err)
			}
			record, err := b.NetworkRecord()
			if err != nil || len(record.Transitions) != 13 || !record.Snapshot.Connections[0].Reset {
				t.Fatalf("reserved expiry failed: %v %+v", err, record)
			}
		})
	}
}

func TestBrokerScheduledPartitionAndHealUseFaultTime(t *testing.T) {
	for _, healAt := range []uint64{0, 111, 112} {
		t.Run(strconv.FormatUint(healAt, 10), func(t *testing.T) {
			config := brokerTestConfig()
			config.Network = streamConfig()
			config.Network.Limits.StallNanos = 10
			config.Faults = []BrokerNetworkFault{{Name: "partition", Action: "partition", From: "a", To: "b", AtNanos: 100, Symmetric: true}}
			if healAt != 0 {
				config.Faults = append(config.Faults, BrokerNetworkFault{Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: healAt, Symmetric: true})
			}
			b, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			client, server, c, s := reviewBrokerPair(t, b)
			if err := b.ChangeLink("delay", "a", "b", 200, true); err != nil {
				t.Fatal(err)
			}
			m := b.network.model
			if r := m.Dispatch(netmodel.Request{Domain: client, Op: "write", Handle: c, RequestID: 4, Bytes: []byte("x")}); r.Error != "" || r.Count != 1 {
				t.Fatal(r)
			}
			if err := b.network.advance(max(110, healAt)); err != nil {
				t.Fatal(err)
			}
			r := m.Dispatch(netmodel.Request{Domain: server, Op: "read", Handle: s, RequestID: 5, Count: 1})
			if healAt == 112 {
				if r.Error != "connection timed out" {
					t.Fatalf("later heal erased earlier expiry: %+v", r)
				}
				return
			}
			if r.State != "pending" {
				t.Fatalf("partition or heal used stale time: %+v", r)
			}
			if healAt == 0 {
				if m.NextDeadline() != 111 {
					t.Fatalf("partition expiry=%d want=111", m.NextDeadline())
				}
				return
			}
			if err := b.network.advance(310); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: server, Op: "poll", RequestID: 5}); r.State != "pending" {
				t.Fatalf("heal delay released early: %+v", r)
			}
			if err := b.network.advance(311); err != nil {
				t.Fatal(err)
			}
			if r := m.Dispatch(netmodel.Request{Domain: server, Op: "poll", RequestID: 5}); string(r.Bytes) != "x" || r.Error != "" {
				t.Fatalf("equal-time heal lost byte stream: %+v", r)
			}
		})
	}
}

// Recorded Go broker orders for the in-process broker (fn-178.7). Each
// scenario runs scripted node call sequences through the Go broker (sessions
// on goroutines, as helpers drive them) and records every call's output
// buffer or error, the broker World's expanded wire and each node's network
// evidence document. The Rust broker replays the same scripts
// (`wasmhost/src/broker/tests.rs`) and must match byte for byte; its network
// evidence is checked in beside the fixture and decoded here.
const brokerOrderDirectory = "../wasmhost/tests/testdata/broker_orders"

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
