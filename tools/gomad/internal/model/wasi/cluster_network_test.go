package wasi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"reflect"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

const clusterNetworkFixtures = "../../../rust/wasmhost/tests/testdata/cluster_network"

func clusterNetworkLimits() ClusterNetworkLimits {
	return ClusterNetworkLimits{Network: streamConfig().Limits, Instance: *streamConfig(), TranscriptBytes: 1 << 20}
}

// goBrokerFixture replays the helper's checked-in cluster network scenario
// (wasmhost `src/cluster/network/tests.rs` `fixture_scenario`) through the Go
// broker path: the Go owner, broker addressing and slot bases, and each node's
// environment with its own network World.
func goBrokerFixture(t *testing.T) (map[string][]byte, netmodel.Record) {
	t.Helper()
	config := brokerTestConfig()
	config.Network = streamConfig()
	config.Addresses = map[string]string{"b": "10.0.7.1"}
	broker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	environments := map[string]*Environment{}
	for _, id := range []string{"a", "b"} {
		e := streamEnvironment(t, streamConfig())
		e.now = 1
		if err := broker.network.attach(broker.Session(id), e); err != nil {
			t.Fatal(err)
		}
		environments[id] = e
	}
	a, b := environments["a"], environments["b"]
	listen := streamCall(t, b, streamFrame(1, 1, 0, 0, 0, 0, nil))
	listener, port := binary.BigEndian.Uint64(listen[16:]), binary.BigEndian.Uint32(listen[48:])
	dialFrame := streamFrame(2, 1, 0, 0, port, 0, nil)
	copy(dialFrame[56:60], []byte{10, 0, 7, 1})
	client := binary.BigEndian.Uint64(streamCall(t, a, dialFrame)[16:])
	server := binary.BigEndian.Uint64(streamCall(t, b, streamFrame(3, 2, listener, 0, 0, 0, nil))[16:])
	if read := streamCall(t, b, streamFrame(4, 3, server, 0, 0, 4, make([]byte, 4))); binary.BigEndian.Uint32(read[40:]) != 1 {
		t.Fatal("Go read did not wait")
	}
	streamCall(t, a, streamFrame(5, 2, client, 0, 0, 4, []byte("ping")))
	if poll := streamCall(t, b, streamFrame(12, 3, 0, 0, 0, 4, make([]byte, 4))); !bytes.Equal(poll[64:], []byte("ping")) {
		t.Fatalf("Go poll = %x", poll)
	}
	streamCall(t, a, streamFrame(13, 9, 0, 0, 0, 0, nil))
	documents := map[string][]byte{}
	for id, e := range environments {
		documents[id], err = e.NetworkV2Evidence()
		if err != nil {
			t.Fatal(err)
		}
	}
	record, err := broker.NetworkRecord()
	if err != nil {
		t.Fatal(err)
	}
	return documents, record
}

// TestClusterNetworkRustEvidenceRebuildsTheGoBrokerDocument is the golden
// round trip: the helper's binary evidence for a cross-node exchange (pending
// read, World readiness, poll, stale cancel) decodes into documents and a
// Record byte-identical to the Go broker path's for the same calls.
func TestClusterNetworkRustEvidenceRebuildsTheGoBrokerDocument(t *testing.T) {
	documents, record := goBrokerFixture(t)
	for _, id := range []string{"a", "b"} {
		data, err := os.ReadFile(clusterNetworkFixtures + id + ".gmnetev1")
		if err != nil {
			t.Fatal(err)
		}
		evidence, err := DecodeClusterNetworkEvidence(data, clusterNetworkLimits())
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if !bytes.Equal(evidence.Document, documents[id]) {
			t.Fatalf("%s: rebuilt document differs from the Go broker path:\nrust %s\ngo   %s", id, evidence.Document, documents[id])
		}
		if !reflect.DeepEqual(evidence.Record, record) {
			t.Fatalf("%s: rebuilt Record differs from the broker NetworkRecord", id)
		}
	}
}

func TestClusterNetworkDecoderRefusesMalformedEvidence(t *testing.T) {
	data, err := os.ReadFile(clusterNetworkFixtures + "b.gmnetev1")
	if err != nil {
		t.Fatal(err)
	}
	refused := func(name string, changed []byte, limits ClusterNetworkLimits) {
		t.Helper()
		_, err := DecodeClusterNetworkEvidence(changed, limits)
		var boundary *BoundaryError
		if !errors.As(err, &boundary) || boundary.Kind != "invalid" {
			t.Errorf("%s admitted or misclassified: %v", name, err)
		}
	}
	for length := range len(data) {
		refused("truncated", data[:length], clusterNetworkLimits())
	}
	refused("trailing", append(bytes.Clone(data), 0), clusterNetworkLimits())
	small := clusterNetworkLimits()
	small.TranscriptBytes = uint64(len(data)) - 1
	refused("oversize", data, small)
	mutate := func(at int, value byte) []byte {
		changed := bytes.Clone(data)
		changed[at] = value
		return changed
	}
	refused("magic", mutate(0, 'X'), clusterNetworkLimits())
	refused("version", mutate(8, 2), clusterNetworkLimits())
	profile := 9 + 1 + len(NetworkStreamProfile)
	// The first entry tag follows the profile, the log byte count (two bytes
	// here) and the entry count.
	tag := profile + 2 + 1
	if data[tag] != 1 {
		t.Fatalf("fixture layout changed: tag byte %d", data[tag])
	}
	for _, unknown := range []byte{0, 2, 4, 255} {
		refused("unknown tag", mutate(tag, unknown), clusterNetworkLimits())
	}
	accounting := mutate(profile, data[profile]+1)
	refused("log accounting", accounting, clusterNetworkLimits())
	nonMinimal := append(append(bytes.Clone(data[:profile]), data[profile]|0x80, data[profile+1]|0x80, 0), data[profile+2:]...)
	refused("non-minimal varint", nonMinimal, clusterNetworkLimits())
	// An invalid UTF-8 byte inside the profile string, a flag byte of 2 (the
	// World presence flag precedes its wire) and a flipped World wire byte.
	refused("invalid UTF-8", mutate(9+1, 0xff), clusterNetworkLimits())
	world := bytes.Index(data, []byte("GMWSNAP1"))
	if world < 0 {
		t.Fatal("fixture carries no World wire")
	}
	worldFlag := world - 1
	for data[worldFlag] != 1 {
		worldFlag--
	}
	refused("flag", mutate(worldFlag, 2), clusterNetworkLimits())
	refused("world wire", mutate(world+20, data[world+20]^1), clusterNetworkLimits())
	// A Record whose identity no longer matches its transitions is refused by
	// the generated codec's replay admission.
	digest := bytes.LastIndex(data, []byte("sha256:"))
	refused("record identity", mutate(digest+7, data[digest+7]^1), clusterNetworkLimits())
}

func TestClusterNetworkFrameRefusesWhatTheGoBrokerRefuses(t *testing.T) {
	nodes := func() map[string]*NetworkModelV2 {
		return map[string]*NetworkModelV2{"a": streamConfig(), "b": streamConfig()}
	}
	cooperative := map[string]bool{"a": true, "b": true}
	base := func() BrokerConfig {
		config := brokerTestConfig()
		config.Network = streamConfig()
		return config
	}
	frame, err := ClusterNetworkFrame(base(), nodes(), cooperative)
	if err != nil || frame == nil || frame.Model.Profile != NetworkStreamProfile {
		t.Fatalf("valid configuration refused: %+v %v", frame, err)
	}
	for name, change := range map[string]func(*BrokerConfig, map[string]*NetworkModelV2, map[string]bool){
		"per-node fault schedule": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Network.Faults = []NetworkFaultV2{{Name: "x", Action: "partition", AtNanos: 1}}
		},
		"bad address": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Addresses = map[string]string{"a": "10.0.0.256"}
		},
		"unknown link endpoint": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Links = []BrokerLink{{From: "a", To: "z", Enabled: true}}
		},
		"self link": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Links = []BrokerLink{{From: "a", To: "a", Enabled: true}}
		},
		"duplicate link": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Links = []BrokerLink{{From: "a", To: "b"}, {From: "a", To: "b"}}
		},
		"unordered fault": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Faults = []BrokerNetworkFault{{Name: "late", Action: "partition", From: "a", To: "b", AtNanos: 9}, {Name: "early", Action: "heal", From: "a", To: "b", AtNanos: 2}}
		},
		"unknown action": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Faults = []BrokerNetworkFault{{Name: "f", Action: "reset", From: "a", To: "b", AtNanos: 1}}
		},
		"node without profile": func(_ *BrokerConfig, n map[string]*NetworkModelV2, _ map[string]bool) { delete(n, "b") },
		"node local faults": func(_ *BrokerConfig, n map[string]*NetworkModelV2, _ map[string]bool) {
			n["b"].Faults = []NetworkFaultV2{{Name: "x", Action: "heal", AtNanos: 1}}
		},
		"stock node":         func(_ *BrokerConfig, _ map[string]*NetworkModelV2, c map[string]bool) { c["a"] = false },
		"no cluster network": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) { c.Network = nil },
		"faults without network": func(c *BrokerConfig, n map[string]*NetworkModelV2, _ map[string]bool) {
			c.Network = nil
			clear(n)
			c.Faults = []BrokerNetworkFault{{Name: "f", Action: "partition", From: "a", To: "b", AtNanos: 1}}
		},
		"restarted incarnation": func(c *BrokerConfig, _ map[string]*NetworkModelV2, _ map[string]bool) {
			c.Nodes[1].Incarnation = 2
		},
	} {
		t.Run(name, func(t *testing.T) {
			config, profiles, runtime := base(), nodes(), map[string]bool{"a": true, "b": true}
			change(&config, profiles, runtime)
			frame, err := ClusterNetworkFrame(config, profiles, runtime)
			var boundary *BoundaryError
			if frame != nil || !errors.As(err, &boundary) || boundary.Kind != "invalid" {
				t.Fatalf("admitted: %+v %v", frame, err)
			}
		})
	}
}
