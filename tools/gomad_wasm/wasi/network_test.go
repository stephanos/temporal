package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func networkEnvironment(t *testing.T, model *NetworkModel) *Environment {
	t.Helper()
	config := testConfig()
	config.Profile = CooperativeProfile
	config.Network = model
	e, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	e.runtime = &runtimeSession{configured: true}
	return e
}

func networkCall(t *testing.T, e *Environment, op, local, remote uint32, count uint64) (uint32, uint64) {
	t.Helper()
	data := make([]byte, 32)
	binary.BigEndian.PutUint32(data, 1)
	binary.BigEndian.PutUint32(data[4:], op)
	binary.BigEndian.PutUint32(data[8:], local)
	binary.BigEndian.PutUint32(data[12:], remote)
	binary.BigEndian.PutUint64(data[16:], count)
	input, _ := json.Marshal(bytesOutput{Data: data})
	output, err := e.runtimeNet(input)
	if err != nil {
		t.Fatal(err)
	}
	return binary.BigEndian.Uint32(output.Data[4:]), binary.BigEndian.Uint64(output.Data[24:])
}

func TestNetworkModelFaultSchedule(t *testing.T) {
	model := &NetworkModel{Seed: 3, MaxDelayNanos: 1000, Faults: []NetworkFault{{AtNanos: 10, Action: "reset", Port: 80}, {AtNanos: 20, Action: "partition"}, {AtNanos: 30, Action: "heal"}}}
	e := networkEnvironment(t, model)
	if action, _ := networkCall(t, e, networkQuery, 0, 0, 0); action != networkRefuse {
		t.Fatal("query did not report an enabled model")
	}
	if action, _ := networkCall(t, e, networkConnect, 5, 80, 0); action != networkDeliver {
		t.Fatal("connect refused before faults")
	}
	if action, _ := networkCall(t, e, networkWrite, 80, 5, 4); action != networkDeliver {
		t.Fatal("reply refused before faults")
	}
	e.now = 10
	if action, delay := networkCall(t, e, networkWrite, 5, 80, 4); action != networkRefuse || delay != 0 {
		t.Fatal("reset did not fail the open connection")
	}
	if action, _ := networkCall(t, e, networkConnect, 6, 80, 0); action != networkDeliver {
		t.Fatal("reset affected a later connection")
	}
	e.now = 25
	if action, _ := networkCall(t, e, networkConnect, 7, 80, 0); action != networkRefuse {
		t.Fatal("partition admitted a connect")
	}
	if action, _ := networkCall(t, e, networkWrite, 6, 80, 1); action != networkRefuse {
		t.Fatal("partition admitted a write")
	}
	e.now = 30
	if action, _ := networkCall(t, e, networkConnect, 8, 80, 0); action != networkDeliver {
		t.Fatal("heal did not restore connects")
	}
	first := e.NetworkTranscript()
	if bytes.Count(first, []byte("\n")) != 7 || !bytes.Contains(first, []byte(`"action":"reset"`)) || !bytes.Contains(first, []byte(`"action":"refuse"`)) {
		t.Fatalf("transcript:\n%s", first)
	}
	// The same model and operations reproduce identical decisions and delays.
	again := networkEnvironment(t, model)
	networkCall(t, again, networkQuery, 0, 0, 0)
	for _, step := range []struct {
		now               uint64
		op, local, remote uint32
		count             uint64
	}{{0, networkConnect, 5, 80, 0}, {0, networkWrite, 80, 5, 4}, {10, networkWrite, 5, 80, 4}, {10, networkConnect, 6, 80, 0}, {25, networkConnect, 7, 80, 0}, {25, networkWrite, 6, 80, 1}, {30, networkConnect, 8, 80, 0}} {
		again.now = step.now
		networkCall(t, again, step.op, step.local, step.remote, step.count)
	}
	if !bytes.Equal(first, again.NetworkTranscript()) {
		t.Fatal("identical operations produced different network decisions")
	}
}

func TestNetworkModelValidation(t *testing.T) {
	for name, model := range map[string]*NetworkModel{
		"action": {Faults: []NetworkFault{{Action: "drop"}}},
		"order":  {Faults: []NetworkFault{{AtNanos: 2, Action: "reset"}, {AtNanos: 1, Action: "heal"}}},
		"delay":  {MaxDelayNanos: 1 << 63},
	} {
		t.Run(name, func(t *testing.T) {
			config := testConfig()
			config.Profile = CooperativeProfile
			config.Network = model
			if _, err := NewEnvironment(config); err == nil {
				t.Fatal("accepted invalid network model")
			}
		})
	}
	config := testConfig()
	config.Network = &NetworkModel{}
	if _, err := NewEnvironment(config); err == nil {
		t.Fatal("stock profile accepted a network model")
	}
	e := networkEnvironment(t, nil)
	if action, _ := networkCall(t, e, networkQuery, 0, 0, 0); action != networkDeliver {
		t.Fatal("query reported a model that is absent")
	}
}
