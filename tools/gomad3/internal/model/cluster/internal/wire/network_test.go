package wire

import (
	"fmt"
	"go.temporal.io/server/tools/gomad3/internal/model/cluster/internal/history"
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
			spec := validRuntimeConfig()
			spec.Limits.NetworkConnectionBytes = test.bytes
			spec.Limits.NetworkStallNanos = test.stall
			encoded, err := EncodeNetworkConfig(spec)
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
	record := history.NetworkRecord{
		Transitions: []history.NetworkTransition{{
			Ordinal: 0, Kind: "partition",
			Source:      history.NetworkEndpoint{Node: "client", Address: "10.0.0.2"},
			Destination: history.NetworkEndpoint{Node: "server", Address: "10.0.0.1"},
			Count:       2, Outcome: "ok",
		}},
		Snapshot: history.NetworkSnapshot{
			Nodes: []history.NetworkNodeSnapshot{
				{Node: "client", Address: "10.0.0.2", LastIncarnation: 1, NextListenerPort: 20000, NextClientPort: 40000},
				{Node: "server", Address: "10.0.0.1", LastIncarnation: 1, NextListenerPort: 20001, NextClientPort: 40001},
			},
			Links: []history.NetworkLinkSnapshot{
				{From: "client", To: "server", DelayNanos: 7},
				{From: "server", To: "client", DelayNanos: 7},
			},
			Listeners: []history.NetworkListenerSnapshot{{
				Endpoint: history.NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233},
				Closed:   true,
			}},
			Connections: []history.NetworkConnectionSnapshot{{
				Identity:    1,
				Client:      history.NetworkEndpoint{Node: "client", Incarnation: 1, Address: "10.0.0.2", Port: 40000},
				Server:      history.NetworkEndpoint{Node: "server", Incarnation: 1, Address: "10.0.0.1", Port: 7233},
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
	decoded, runtimeErr, err := DecodeNetworkFinish(encoder.data)
	require.NoError(t, err)
	require.Nil(t, runtimeErr)
	require.Equal(t, record, decoded)
}

func TestRuntimeNetworkWireRejectsMalformedInput(t *testing.T) {
	spec := validRuntimeConfig()
	encoded, err := EncodeNetworkConfig(spec)
	require.NoError(t, err)
	require.Equal(t, runtimeNetworkConfigMagic, string(encoded[:len(runtimeNetworkConfigMagic)]))
	second, err := EncodeNetworkConfig(spec)
	require.NoError(t, err)
	require.Equal(t, encoded, second)

	finish := &runtimeNetworkWireEncoder{data: []byte(runtimeNetworkFinishMagic)}
	encodeRuntimeNetworkRecord(finish, history.NetworkRecord{})
	finish.boolean(false)
	require.NoError(t, finish.err)
	for offset := range len(finish.data) {
		_, _, err := DecodeNetworkFinish(finish.data[:offset])
		require.Error(t, err, "offset %d", offset)
	}
	trailing := append(append([]byte(nil), finish.data...), 0)
	_, _, err = DecodeNetworkFinish(trailing)
	require.Error(t, err)
	invalidBoolean := append([]byte(nil), finish.data...)
	invalidBoolean[len(invalidBoolean)-8] = 2
	_, _, err = DecodeNetworkFinish(invalidBoolean)
	require.Error(t, err)
}
