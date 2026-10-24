package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"testing"

	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

func readinessFrame(ids ...uint64) []byte {
	data := streamFrame(14, 0, 0, 0, 0, 0, nil)
	binary.BigEndian.PutUint32(data[36:], uint32(len(ids)))
	for _, id := range ids {
		data = binary.BigEndian.AppendUint64(data, id)
	}
	return data
}

func TestNetworkV2ReadinessProbePreservesLegacyUndefinedOpcode(t *testing.T) {
	t.Parallel()
	e := streamEnvironment(t, streamConfig())
	encoded, err := json.Marshal(bytesOutput{Data: streamFrame(14, 1, 0, 0, 0, 0, nil)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.runtimeNetworkStream(encoded); err == nil || err.Error() != "WASI invalid: network v2 opcode" {
		t.Fatalf("legacy invalid opcode result changed: %v", err)
	}
}

func TestNetworkV2ReadinessProbe(t *testing.T) {
	t.Parallel()
	e := streamEnvironment(t, streamConfig())
	client, server := reviewStreamPair(t, e)
	for i, handle := range []uint64{server, client} {
		id := uint64(i + 4)
		result := streamCall(t, e, streamFrame(4, id, handle, 0, 0, 1, nil))
		if binary.BigEndian.Uint32(result[40:]) != 1 {
			t.Fatal("read did not pend")
		}
	}
	probe := readinessFrame(5, 4)
	want := readinessFrame(0, 0)
	if got := streamCall(t, e, probe); !bytes.Equal(got, want) {
		t.Fatalf("pending probe = %x, want %x", got, want)
	}
	streamCall(t, e, streamFrame(5, 6, client, 0, 0, 1, []byte("x")))
	want = readinessFrame(0, 4)
	for range 2 {
		if got := streamCall(t, e, probe); !bytes.Equal(got, want) {
			t.Fatalf("ready probe = %x, want %x", got, want)
		}
	}
	result := streamCall(t, e, streamFrame(12, 4, 0, 0, 0, 1, nil))
	if binary.BigEndian.Uint32(result[40:]) != 0 || binary.BigEndian.Uint32(result[36:]) != 1 || result[64] != 'x' {
		t.Fatalf("probe consumed read result: %x", result)
	}
}

func TestNetworkV2ReadinessProbeAdvancesDeadline(t *testing.T) {
	t.Parallel()
	e := streamEnvironment(t, streamConfig())
	client, _ := reviewStreamPair(t, e)
	streamCall(t, e, streamFrame(4, 4, client, 10, 0, 1, nil))
	e.now = 10
	probe := readinessFrame(4)
	if got := streamCall(t, e, probe); !bytes.Equal(got, probe) || !e.networkV2.worldReady[4] {
		t.Fatalf("deadline probe did not advance model and world: %x", got)
	}
	result := streamCall(t, e, streamFrame(12, 4, 0, 0, 0, 1, nil))
	if binary.BigEndian.Uint32(result[44:]) != 73 {
		t.Fatalf("deadline result was lost: %x", result)
	}
}

func TestNetworkV2ReadinessProbeBounds(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 128} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			t.Parallel()
			e := streamEnvironment(t, streamConfig())
			listener := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
			ids := make([]uint64, count)
			for i := range ids {
				ids[i] = uint64(i + 2)
				streamCall(t, e, streamFrame(3, ids[i], binary.BigEndian.Uint64(listener[16:]), 0, 0, 0, nil))
			}
			got := streamCall(t, e, readinessFrame(ids...))
			if want := readinessFrame(make([]uint64, count)...); !bytes.Equal(got, want) {
				t.Fatalf("bounded probe = %x, want %x", got, want)
			}
		})
	}
}

func TestNetworkV2ReadinessProbeRejectsInvalidUnchanged(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*testing.T, *Environment, []byte) []byte
	}{
		{"zero", func(_ *testing.T, _ *Environment, b []byte) []byte { binary.BigEndian.PutUint64(b[64:], 0); return b }},
		{"wide", func(_ *testing.T, _ *Environment, b []byte) []byte {
			binary.BigEndian.PutUint64(b[64:], 1<<48)
			return b
		}},
		{"unknown", func(_ *testing.T, _ *Environment, b []byte) []byte { binary.BigEndian.PutUint64(b[64:], 99); return b }},
		{"duplicate", func(_ *testing.T, _ *Environment, _ []byte) []byte { return readinessFrame(2, 2) }},
		{"mixed_unknown", func(_ *testing.T, _ *Environment, _ []byte) []byte { return readinessFrame(2, 99) }},
		{"short", func(_ *testing.T, _ *Environment, b []byte) []byte { return b[:len(b)-1] }},
		{"trailing", func(_ *testing.T, _ *Environment, b []byte) []byte { return append(b, 0) }},
		{"count", func(_ *testing.T, _ *Environment, b []byte) []byte { binary.BigEndian.PutUint32(b[36:], 0); return b }},
		{"oversized", func(_ *testing.T, _ *Environment, _ []byte) []byte { return readinessFrame(make([]uint64, 129)...) }},
		{"consumed", func(t *testing.T, e *Environment, b []byte) []byte {
			streamCall(t, e, streamFrame(13, 2, 0, 0, 0, 0, nil))
			return b
		}},
		{"revoked", func(t *testing.T, e *Environment, b []byte) []byte {
			if err := e.networkV2.model.Revoke(e.networkV2.server, false); err != nil {
				t.Fatal(err)
			}
			return b
		}},
		{"foreign", func(_ *testing.T, e *Environment, b []byte) []byte {
			e.networkV2.requests[2] = netmodel.Domain{Run: 1, Node: "foreign", Incarnation: 1}
			return b
		}},
		{"old_incarnation", func(_ *testing.T, e *Environment, b []byte) []byte {
			d := e.networkV2.requests[2]
			d.Incarnation++
			e.networkV2.requests[2] = d
			return b
		}},
		{"inactive", func(_ *testing.T, e *Environment, b []byte) []byte { e.runtime.configured = false; return b }},
		{"finished", func(_ *testing.T, e *Environment, b []byte) []byte { e.runtime.finished = true; return b }},
	}
	for _, offset := range []int{16, 24, 32, 40, 44, 48, 52, 56, 60} {
		cases = append(cases, struct {
			name   string
			mutate func(*testing.T, *Environment, []byte) []byte
		}{
			name:   "header_" + strconv.Itoa(offset),
			mutate: func(_ *testing.T, _ *Environment, b []byte) []byte { b[offset] = 1; return b },
		})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := streamEnvironment(t, streamConfig())
			listener := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
			streamCall(t, e, streamFrame(3, 2, binary.BigEndian.Uint64(listener[16:]), 0, 0, 0, nil))
			frame := tc.mutate(t, e, readinessFrame(2))
			e.now = 10
			before, err := e.NetworkV2Evidence()
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(bytesOutput{Data: frame})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := e.runtimeNetworkStream(encoded); err == nil {
				t.Fatal("invalid readiness probe accepted")
			}
			after, err := e.NetworkV2Evidence()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("invalid probe changed evidence")
			}
			if _, err := e.networkV2.model.Advance(1); err != nil {
				t.Fatalf("invalid probe advanced model clock: %v", err)
			}
		})
	}
}
