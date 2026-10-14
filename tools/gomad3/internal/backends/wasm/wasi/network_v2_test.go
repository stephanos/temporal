package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad3/internal/model/network"
	"go.temporal.io/server/tools/gomad3/internal/model/world"
)

func TestNetworkV2RustOwnerPinIsCurrent(t *testing.T) {
	const path = "../wasmhost/src/network/PINNED_GO_NETWORK_SHA256"
	pin, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := netmodel.ImplementationSHA256() + "\n"; !bytes.Equal(pin, []byte(want)) {
		t.Fatalf("update the Rust network owner and regenerate its corpus before updating %s: got %q, want %q", path, pin, want)
	}
}

func streamConfig() *NetworkModelV2 {
	return &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 8192}}
}

func TestNetworkV2ExplicitByteAndStallLimits(t *testing.T) {
	for _, test := range []struct {
		name         string
		bytes, stall uint64
		valid        bool
	}{
		{"minimum", 1, 1, true},
		{"maximum", 1 << 30, math.MaxInt64 - 1, true},
		{"zero bytes", 0, 1, false},
		{"oversized bytes", (1 << 30) + 1, 1, false},
		{"zero stall", 1, 0, false},
		{"threshold overflow", 1, math.MaxInt64, false},
		{"signed overflow", 1, uint64(math.MaxInt64) + 1, false},
		{"unsigned overflow", math.MaxUint64, math.MaxUint64, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := streamConfig()
			config.Limits.ConnectionBytes, config.Limits.StallNanos = test.bytes, test.stall
			err := validateNetworkModelV2(config)
			if test.valid && err != nil || !test.valid && err == nil {
				t.Fatalf("bytes=%d stall=%d admitted=%v want=%v", test.bytes, test.stall, err == nil, test.valid)
			}
		})
	}
}

func TestNetworkV2WorldReadinessIndependentOfPayloadBytes(t *testing.T) {
	config := streamConfig()
	config.Limits.Bytes, config.Limits.ConnectionBytes = 1, 1
	config.Limits.Transitions = 128
	e := streamEnvironment(t, config)
	for i := 0; i < 129; i++ {
		id, err := e.networkV2.world.Register(world.Request{Kind: "network", Resource: world.ResourceID{Adapter: "wasm", Kind: "stream", Key: "readiness"}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.networkV2.world.Ready(world.Readiness{RequestID: id, At: world.InitialTime + 1, Kind: "network"}); err != nil {
			t.Fatalf("request %d readiness depended on stream admission: %v", i, err)
		}
	}
	q, err := e.networkV2.world.Quiesce()
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Deliveries) != 129 {
		t.Fatalf("World lost bounded readiness events: %d", len(q.Deliveries))
	}
}
func streamEnvironment(t *testing.T, model *NetworkModelV2) *Environment {
	t.Helper()
	config := testConfig()
	config.Profile = CooperativeProfile
	config.Clock.ReadStepNanos = 0
	config.Limits.TranscriptBytes = 8 << 20
	config.NetworkV2 = model
	e, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	e.runtime = &runtimeSession{configured: true}
	return e
}
func streamFrame(op uint32, id, handle, deadline uint64, port, count uint32, payload []byte) []byte {
	data := make([]byte, 64+int(count))
	binary.BigEndian.PutUint32(data, 2)
	binary.BigEndian.PutUint32(data[4:], op)
	binary.BigEndian.PutUint64(data[8:], id)
	binary.BigEndian.PutUint64(data[16:], handle)
	binary.BigEndian.PutUint64(data[24:], deadline)
	binary.BigEndian.PutUint32(data[32:], port)
	binary.BigEndian.PutUint32(data[36:], count)
	copy(data[64:], payload)
	return data
}
func streamCall(t *testing.T, e *Environment, data []byte) []byte {
	t.Helper()
	encoded, err := json.Marshal(bytesOutput{Data: data})
	if err != nil {
		t.Fatal(err)
	}
	r, err := e.runtimeNetworkStream(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return r.Data
}

func TestNetworkV2ClockCeilingRejectsStallAdmissionAtomically(t *testing.T) {
	for _, epoch := range []uint64{0, uint64(world.InitialTime), uint64(world.InitialTime) + 100, math.MaxInt64 - 100} {
		for _, operation := range []string{"held write", "pending dial", "partition queued bytes"} {
			t.Run(fmt.Sprintf("epoch=%d/%s", epoch, operation), func(t *testing.T) {
				ceiling := min(uint64(math.MaxInt64-int64(world.InitialTime)), math.MaxInt64-epoch)
				config := testConfig()
				config.Profile, config.Clock = CooperativeProfile, ClockPolicy{EpochNanos: epoch}
				config.NetworkV2 = streamConfig()
				config.NetworkV2.Limits.StallNanos = ceiling - 1
				e := newTestEnvironment(t, config)
				e.runtime = &runtimeSession{configured: true}
				listener := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
				port := binary.BigEndian.Uint32(listener[48:])
				var client uint64
				if operation != "pending dial" {
					connected := streamCall(t, e, streamFrame(2, 2, 0, 0, port, 0, nil))
					client = binary.BigEndian.Uint64(connected[16:])
				}
				if operation == "partition queued bytes" {
					if err := e.networkV2.model.Delay(e.networkV2.client.Node, e.networkV2.server.Node, 10, false); err != nil {
						t.Fatal(err)
					}
					streamCall(t, e, streamFrame(5, 3, client, 0, 0, 4, []byte("held")))
				} else if err := e.networkV2.model.Partition(e.networkV2.client.Node, e.networkV2.server.Node, false); err != nil {
					t.Fatal(err)
				}
				before, err := e.networkV2.model.Record()
				if err != nil {
					t.Fatal(err)
				}
				beforeWorld, beforeDeadline := e.networkV2.world.Snapshot(), e.networkV2.model.NextDeadline()
				if operation == "partition queued bytes" {
					err = e.networkV2.model.Partition(e.networkV2.client.Node, e.networkV2.server.Node, false)
				} else {
					frame := streamFrame(5, 4, client, 0, 0, 4, []byte("held"))
					if operation == "pending dial" {
						frame = streamFrame(2, 4, 0, 0, port, 0, nil)
					}
					input, marshalErr := json.Marshal(bytesOutput{Data: frame})
					if marshalErr != nil {
						t.Fatal(marshalErr)
					}
					result, callErr := e.runtimeNetworkStream(input)
					err = callErr
					if err == nil && binary.BigEndian.Uint32(result.Data[44:]) == 0 {
						t.Errorf("unrepresentable stall admitted: next=%d ceiling=%d state=%d", e.networkV2.model.NextDeadline(), ceiling, binary.BigEndian.Uint32(result.Data[40:]))
					}
				}
				after, recordErr := e.networkV2.model.Record()
				if recordErr != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeWorld, e.networkV2.world.Snapshot()) || e.networkV2.model.NextDeadline() != beforeDeadline || len(e.networkV2.requests) != 0 || len(e.networkV2.model.Ready()) != 0 {
					t.Fatalf("rejected expiry changed bytes/history/identity/pending work: error=%v recordError=%v identity=%s/%s transitions=%d/%d next=%d/%d requests=%v", err, recordErr, before.Snapshot.Identity, after.Snapshot.Identity, len(before.Transitions), len(after.Transitions), beforeDeadline, e.networkV2.model.NextDeadline(), e.networkV2.requests)
				}
				if operation == "partition queued bytes" && err == nil {
					t.Fatal("queued partition admitted unrepresentable expiry")
				}
			})
		}
	}
}

func TestNetworkV2ClockCeilingAllowsRepresentableExpiry(t *testing.T) {
	for _, epoch := range []uint64{0, uint64(world.InitialTime) - 100, uint64(world.InitialTime) + 100, math.MaxInt64 - 100} {
		for _, below := range []uint64{0, 1} {
			for _, dial := range []bool{false, true} {
				t.Run(fmt.Sprintf("epoch=%d/below=%d/dial=%v", epoch, below, dial), func(t *testing.T) {
					ceiling := min(uint64(math.MaxInt64-int64(world.InitialTime)), math.MaxInt64-epoch)
					due := ceiling - below
					config := testConfig()
					config.Profile, config.Clock = CooperativeProfile, ClockPolicy{EpochNanos: epoch}
					config.NetworkV2 = streamConfig()
					config.NetworkV2.Limits.StallNanos = due - 2
					e := newTestEnvironment(t, config)
					e.runtime = &runtimeSession{configured: true}
					listener := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
					port := binary.BigEndian.Uint32(listener[48:])
					var client uint64
					if !dial {
						connected := streamCall(t, e, streamFrame(2, 2, 0, 0, port, 0, nil))
						client = binary.BigEndian.Uint64(connected[16:])
					}
					if err := e.networkV2.model.Partition(e.networkV2.client.Node, e.networkV2.server.Node, false); err != nil {
						t.Fatal(err)
					}
					frame := streamFrame(5, 4, client, 0, 0, 4, []byte("held"))
					if dial {
						frame = streamFrame(2, 4, 0, 0, port, 0, nil)
					}
					result := streamCall(t, e, frame)
					if binary.BigEndian.Uint32(result[44:]) != 0 || e.networkV2.model.NextDeadline() != due {
						t.Fatal("representable expiry was not admitted")
					}
					if _, err := e.networkV2Idle(make([]byte, 16), 0); err != nil || e.now != due {
						t.Fatalf("representable expiry could not advance World and WASI clocks: now=%d due=%d err=%v", e.now, due, err)
					}
					if dial {
						result = streamCall(t, e, streamFrame(12, 4, 0, 0, 0, 0, nil))
						if binary.BigEndian.Uint32(result[44:]) != netmodel.StreamStallTimeout {
							t.Fatal("representable dial expiry did not retain timeout")
						}
					} else {
						snapshot, err := e.networkV2.model.Snapshot()
						if err != nil || len(snapshot.Connections) != 1 || !snapshot.Connections[0].Reset || snapshot.Connections[0].ServerBytes != 0 {
							t.Fatal("representable held write expiry did not reset and release bytes", err)
						}
					}
				})
			}
		}
	}
}

func TestNetworkV2ClockEpochOverflowRefusesBeforeActivation(t *testing.T) {
	for _, epoch := range []uint64{math.MaxInt64, math.MaxInt64 + 1, math.MaxUint64} {
		config := testConfig()
		config.Profile, config.Clock = CooperativeProfile, ClockPolicy{EpochNanos: epoch}
		config.NetworkV2 = streamConfig()
		if environment, err := NewEnvironment(config); err == nil || environment != nil {
			t.Fatalf("unrepresentable network startup clock activated: epoch=%d err=%v", epoch, err)
		}
	}
}

func TestBrokerNetworkClockCeilingRejectsPendingDialAtomically(t *testing.T) {
	config := brokerTestConfig()
	config.Network = streamConfig()
	config.Network.Limits.StallNanos = math.MaxInt64 - 1
	broker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	n := broker.network
	client, _ := n.domains(broker.Session("a"))
	_, server := n.domains(broker.Session("b"))
	listener := n.model.Dispatch(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
	if err := n.model.Partition(client.Node, server.Node, false); err != nil {
		t.Fatal(err)
	}
	before, err := n.model.Record()
	if err != nil {
		t.Fatal(err)
	}
	result := n.model.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 2})
	if result.Error != netmodel.ErrUnsupported.Error() || result.State != "completed" {
		t.Fatalf("broker admitted expiry beyond World clock: %+v", result)
	}
	after, err := n.model.Record()
	if err != nil || !reflect.DeepEqual(before, after) || n.model.NextDeadline() != 0 || len(n.model.Ready()) != 0 {
		t.Fatal("broker clock-ceiling rejection changed pending work or identity", err)
	}
}

func TestBrokerNetworkAttachPreservesGuestClockCeiling(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint("existing expiry=", pending), func(t *testing.T) {
			brokerConfig := brokerTestConfig()
			brokerConfig.Network = streamConfig()
			brokerConfig.Network.Limits.StallNanos = 100
			broker, err := NewBroker(brokerConfig)
			if err != nil {
				t.Fatal(err)
			}
			n := broker.network
			session := broker.Session("a")
			client, server := n.domains(session)
			listener := n.model.Dispatch(netmodel.Request{Domain: server, Op: "listen", RequestID: 1})
			if err := n.model.Partition(client.Node, server.Node, false); err != nil {
				t.Fatal(err)
			}
			if pending {
				result := n.model.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 2})
				if result.State != "pending" || result.Deadline != 102 {
					t.Fatal(result)
				}
			}
			config := testConfig()
			config.Profile, config.Clock = CooperativeProfile, ClockPolicy{EpochNanos: math.MaxInt64 - 100}
			config.NetworkV2 = brokerConfig.Network
			e := newTestEnvironment(t, config)
			previousOwner := e.networkV2.model
			before, err := n.model.Record()
			if err != nil {
				t.Fatal(err)
			}
			err = n.attach(session, e)
			if pending {
				after, recordErr := n.model.Record()
				if err == nil || recordErr != nil || e.networkV2.model != previousOwner || !reflect.DeepEqual(before, after) || n.model.NextDeadline() != 102 {
					t.Fatal("incompatible guest clock mutated attachment or admitted work", err, recordErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			result := n.model.Dispatch(netmodel.Request{Domain: client, Op: "dial", Host: server.Address, Port: listener.Local.Port, RequestID: 3})
			after, recordErr := n.model.Record()
			if result.Error != netmodel.ErrUnsupported.Error() || recordErr != nil || !reflect.DeepEqual(before, after) || n.model.NextDeadline() != 0 {
				t.Fatalf("attachment discarded guest clock ceiling: %+v %v", result, recordErr)
			}
		})
	}
}

func TestNetworkV2StreamBytesDelayGlobalIdleAndCancel(t *testing.T) {
	e := streamEnvironment(t, streamConfig())
	l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
	listener := binary.BigEndian.Uint64(l[16:])
	port := binary.BigEndian.Uint32(l[48:])
	c := streamCall(t, e, streamFrame(2, 2, 0, 0, port, 0, nil))
	client := binary.BigEndian.Uint64(c[16:])
	s := streamCall(t, e, streamFrame(3, 3, listener, 0, 0, 0, nil))
	server := binary.BigEndian.Uint64(s[16:])
	if listener == 0 || client == 0 || server == 0 {
		t.Fatal("missing stable handles")
	}
	if err := e.networkV2.model.Delay(e.networkV2.client.Node, e.networkV2.server.Node, 100, false); err != nil {
		t.Fatal(err)
	}
	data := []byte("actual modeled bytes")
	w := streamCall(t, e, streamFrame(5, 4, client, 0, 0, uint32(len(data)), data))
	if binary.BigEndian.Uint32(w[36:]) != uint32(len(data)) {
		t.Fatal("write not committed")
	}
	r := streamCall(t, e, streamFrame(4, 5, server, 0, 0, 64, nil))
	if binary.BigEndian.Uint32(r[40:]) != 1 {
		t.Fatal("delayed read did not pend")
	}
	idle := make([]byte, 16)
	binary.BigEndian.PutUint32(idle, 1)
	binary.BigEndian.PutUint64(idle[8:], 10000)
	input, _ := json.Marshal(bytesOutput{Data: idle})
	advanced, err := e.runtimeIdle(input)
	if err != nil {
		t.Fatal(err)
	}
	if binary.BigEndian.Uint64(advanced.Data[8:]) != 101 || e.now != 101 {
		t.Fatal("clock skipped network deadline", e.now)
	}
	clear(r[40:64])
	binary.BigEndian.PutUint32(r[4:], 12)
	r = streamCall(t, e, r)
	if !bytes.Equal(r[64:64+len(data)], data) || binary.BigEndian.Uint32(r[36:]) != uint32(len(data)) {
		t.Fatal("stream data changed")
	}
	streamCall(t, e, streamFrame(4, 6, server, 0, 0, 16, nil))
	streamCall(t, e, streamFrame(13, 6, 0, 0, 0, 0, nil))
	if len(e.networkV2.model.Ready()) != 0 {
		t.Fatal("cancel retained pending completion")
	}
	var evidence struct {
		Profile string          `json:"profile"`
		Native  netmodel.Record `json:"native"`
		World   json.RawMessage `json:"world"`
	}
	transcript, err := e.NetworkV2Evidence()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(transcript, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Profile != NetworkStreamProfile || len(evidence.World) == 0 {
		t.Fatal("missing native or World evidence")
	}
	writes := 0
	for _, transition := range evidence.Native.Transitions {
		if transition.Kind == "deliver" {
			t.Fatal("network evidence retained per-delivery history")
		}
		if transition.Kind == "write" {
			writes++
			if transition.Bytes != uint64(len(data)) {
				t.Fatalf("write evidence lost structural byte count: %+v", transition)
			}
		}
	}
	if writes != 1 {
		t.Fatalf("write evidence count=%d want=1", writes)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(transcript, &envelope); err != nil {
		t.Fatal(err)
	}
	var native map[string]json.RawMessage
	if err := json.Unmarshal(envelope["native"], &native); err != nil {
		t.Fatal(err)
	}
	var transitions []map[string]json.RawMessage
	if err := json.Unmarshal(native["transitions"], &transitions); err != nil {
		t.Fatal(err)
	}
	for _, transition := range transitions {
		for _, field := range []string{"payload_sha256", "delivery"} {
			if _, present := transition[field]; present {
				t.Fatalf("network transition retained %s", field)
			}
		}
	}
	var snapshot map[string]json.RawMessage
	if err := json.Unmarshal(native["snapshot"], &snapshot); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"deliveries", "next_delivery"} {
		if _, present := snapshot[field]; present {
			t.Fatalf("network snapshot retained %s", field)
		}
	}
}

func TestNetworkV2ProfileIdentityAndNegativeFrames(t *testing.T) {
	old := streamConfig()
	old.Profile = "gomad3.simulation-network-stream/v2"
	if _, err := newNetworkV2State(old, 0); err == nil {
		t.Fatal("old network profile activated")
	}
	e := streamEnvironment(t, streamConfig())
	for _, data := range [][]byte{make([]byte, 63), streamFrame(5, 1, 1, 0, 0, 2, []byte{1, 2}), streamFrame(99, 1, 1, 0, 0, 0, nil)} {
		if len(data) >= 64 && binary.BigEndian.Uint32(data[4:]) == 5 {
			binary.BigEndian.PutUint32(data[40:], 1)
		}
		input, _ := json.Marshal(bytesOutput{Data: data})
		if _, err := e.runtimeNetworkStream(input); err == nil {
			t.Fatal("accepted malformed frame")
		}
	}
	config := testConfig()
	config.Profile = CooperativeProfile
	config.NetworkV2 = streamConfig()
	config.Clock.ReadStepNanos = 0
	v2, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	config.NetworkV2 = nil
	plain, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	if plain.Identity() == v2.Identity() {
		t.Fatal("v2 profile not bound into identity")
	}
}

func TestNetworkV2NamedResetPreservesListener(t *testing.T) {
	e := streamEnvironment(t, streamConfig())
	e.networkV2.config.Faults = []NetworkFaultV2{{Name: "rpc-reset", Action: "reset", AtNanos: 10}}
	l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
	port := binary.BigEndian.Uint32(l[48:])
	c := streamCall(t, e, streamFrame(2, 2, 0, 0, port, 0, nil))
	client := binary.BigEndian.Uint64(c[16:])
	streamCall(t, e, streamFrame(3, 3, binary.BigEndian.Uint64(l[16:]), 0, 0, 0, nil))
	e.now = 10
	w := streamCall(t, e, streamFrame(5, 4, client, 0, 0, 1, []byte{1}))
	if binary.BigEndian.Uint32(w[44:]) != 15 {
		t.Fatal("named reset not observed")
	}
	c = streamCall(t, e, streamFrame(2, 5, 0, 0, port, 0, nil))
	if binary.BigEndian.Uint32(c[44:]) != 0 || binary.BigEndian.Uint64(c[16:]) == 0 {
		t.Fatal("reset revoked listener")
	}
	if !bytes.Contains(e.NetworkV2Transcript(), []byte("rpc-reset")) {
		t.Fatal("fault identity absent")
	}
}

func TestNetworkV2FinalEvidenceLimitIsTyped(t *testing.T) {
	e := streamEnvironment(t, streamConfig())
	e.config.Limits.TranscriptBytes = 1
	data, err := e.NetworkV2Evidence()
	boundary, ok := err.(*BoundaryError)
	if data != nil || !ok || boundary.Kind != "capacity" || boundary.Dimension != "network-v2-evidence-bytes" {
		t.Fatalf("evidence cap: data=%d error=%v", len(data), err)
	}
}

func TestNetworkV2StaleCancelUsesClosedTransportVocabulary(t *testing.T) {
	e := streamEnvironment(t, streamConfig())
	result := streamCall(t, e, streamFrame(13, 1, 0, 0, 0, 0, nil))
	if binary.BigEndian.Uint32(result[44:]) != 8 || !bytes.Contains(e.NetworkV2Transcript(), []byte(`"error":"stale file handle"`)) {
		t.Fatalf("stale cancellation changed transport classification or vocabulary: %x", result)
	}
}

func TestNetworkV2RejectsPollingReadIntoShortPayload(t *testing.T) {
	e := streamEnvironment(t, streamConfig())
	l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
	c := streamCall(t, e, streamFrame(2, 2, 0, 0, binary.BigEndian.Uint32(l[48:]), 0, nil))
	s := streamCall(t, e, streamFrame(3, 3, binary.BigEndian.Uint64(l[16:]), 0, 0, 0, nil))
	streamCall(t, e, streamFrame(4, 4, binary.BigEndian.Uint64(s[16:]), 0, 0, 16, nil))
	streamCall(t, e, streamFrame(5, 5, binary.BigEndian.Uint64(c[16:]), 0, 0, 4, []byte("data")))
	encoded, _ := json.Marshal(bytesOutput{Data: streamFrame(12, 4, 0, 0, 0, 0, nil)})
	if _, err := e.runtimeNetworkStream(encoded); err == nil {
		t.Fatal("poll silently truncated completed stream bytes")
	}
}
