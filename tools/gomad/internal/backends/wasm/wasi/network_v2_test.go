package wasi

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

func streamConfig() *NetworkModelV2 {
	return &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 8192}}
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

func TestNetworkV2StateRejectsInvalidProfile(t *testing.T) {
	old := streamConfig()
	old.Profile = "gomad.simulation-network-stream/v2"
	if _, err := newNetworkV2State(old, 0); err == nil {
		t.Fatal("old network profile activated")
	}
}

func reviewStreamPair(t *testing.T, e *Environment) (uint64, uint64) {
	t.Helper()
	l := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
	c := streamCall(t, e, streamFrame(2, 2, 0, 0, binary.BigEndian.Uint32(l[48:]), 0, nil))
	s := streamCall(t, e, streamFrame(3, 3, binary.BigEndian.Uint64(l[16:]), 0, 0, 0, nil))
	return binary.BigEndian.Uint64(c[16:]), binary.BigEndian.Uint64(s[16:])
}
