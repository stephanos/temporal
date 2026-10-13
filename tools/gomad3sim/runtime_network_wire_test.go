package gomad3sim

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeNetworkConfigExplicitLimits(t *testing.T) {
	for _, test := range []struct {
		bytes, stall uint64
		valid        bool
	}{
		{1, 1, true},
		{1 << 30, math.MaxInt64 - 1, true},
		{0, 1, false},
		{(1 << 30) + 1, 1, false},
		{1, 0, false},
		{1, math.MaxInt64, false},
		{math.MaxUint64, math.MaxUint64, false},
	} {
		t.Run(fmt.Sprintf("%d/%d", test.bytes, test.stall), func(t *testing.T) {
			spec := validSpec()
			spec.Limits.NetworkConnectionBytes = test.bytes
			spec.Limits.NetworkStallNanos = test.stall
			encoded, err := encodeRuntimeNetworkConfig(spec)
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, "GOMADNC3", string(encoded[:8]))
			} else {
				require.Error(t, err, "bytes=%d stall=%d", test.bytes, test.stall)
			}
		})
	}
}

func TestRuntimeNetworkWireRecordRoundTrip(t *testing.T) {
	record := NetworkRecord{
		Transitions: []NetworkTransition{{
			Ordinal: 0, Kind: NetworkPartition,
			Source:      NetworkEndpoint{Node: "client", Address: "10.0.0.2"},
			Destination: NetworkEndpoint{Node: "server", Address: "10.0.0.1"},
			Count:       2, Outcome: NetworkOutcomeOK,
		}},
		Snapshot: NetworkSnapshot{
			Nodes: []NetworkNodeSnapshot{
				{Node: "client", Address: "10.0.0.2", LastIncarnation: 1, NextListenerPort: 20000, NextClientPort: 40000},
				{Node: "server", Address: "10.0.0.1", LastIncarnation: 1, NextListenerPort: 20001, NextClientPort: 40001},
			},
			Links: []NetworkLinkSnapshot{
				{From: "client", To: "server", DelayNanos: 7},
				{From: "server", To: "client", DelayNanos: 7},
			},
			Listeners: []NetworkListenerSnapshot{{
				Endpoint: NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233},
				Closed:   true,
			}},
			Connections: []NetworkConnectionSnapshot{{
				Identity:    1,
				Client:      NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40000},
				Server:      NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233},
				Closed:      true,
				ClientBytes: 2, ServerBytes: 3,
			}},
			NextConnection:   1,
			TransitionSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Identity:         "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
	}
	encoder := &runtimeNetworkWireEncoder{data: []byte(runtimeNetworkFinishMagic)}
	encodeRuntimeNetworkRecord(encoder, record)
	encoder.boolean(false)
	require.NoError(t, encoder.err)
	decoded, runtimeErr, err := decodeRuntimeNetworkFinish(encoder.data)
	require.NoError(t, err)
	require.Nil(t, runtimeErr)
	require.Equal(t, record, decoded)
}

func TestRuntimeNetworkWireRejectsMalformedInput(t *testing.T) {
	spec := validSpec()
	encoded, err := encodeRuntimeNetworkConfig(spec)
	require.NoError(t, err)
	require.Equal(t, runtimeNetworkConfigMagic, string(encoded[:len(runtimeNetworkConfigMagic)]))
	second, err := encodeRuntimeNetworkConfig(spec)
	require.NoError(t, err)
	require.Equal(t, encoded, second)

	finish := &runtimeNetworkWireEncoder{data: []byte(runtimeNetworkFinishMagic)}
	encodeRuntimeNetworkRecord(finish, emptyNetworkRecord())
	finish.boolean(false)
	require.NoError(t, finish.err)
	for offset := range len(finish.data) {
		_, _, err := decodeRuntimeNetworkFinish(finish.data[:offset])
		require.Error(t, err, "offset %d", offset)
	}
	trailing := append(append([]byte(nil), finish.data...), 0)
	_, _, err = decodeRuntimeNetworkFinish(trailing)
	require.Error(t, err)
	invalidBoolean := append([]byte(nil), finish.data...)
	invalidBoolean[len(invalidBoolean)-8] = 2
	_, _, err = decodeRuntimeNetworkFinish(invalidBoolean)
	require.Error(t, err)
}
