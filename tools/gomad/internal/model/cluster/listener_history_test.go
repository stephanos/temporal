package cluster

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/wire"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

func TestRealNetworkListenerRebindHistoryRoundTrips(t *testing.T) {
	for _, keys := range []int{1, 17} {
		for _, closed := range []bool{false, true} {
			t.Run(fmt.Sprintf("keys%d_closed%t", keys, closed), func(t *testing.T) {
				limits := netmodel.Limits{Listeners: uint64(keys * 4), Connections: 128, Bytes: 1 << 20, ConnectionBytes: 65536, StallNanos: 1000, Transitions: 8192}
				config := netmodel.Config{Nodes: []netmodel.Node{{Node: "a", Address: "127.0.0.1"}}, Limits: limits}
				owner, err := netmodel.New(1, config)
				require.NoError(t, err)
				domain := netmodel.Domain{Run: 1, Node: "a", Address: "127.0.0.1", Incarnation: 1}
				require.NoError(t, owner.Register(domain))
				var sequence uint64
				dispatch := func(request netmodel.Request) netmodel.Result {
					sequence++
					request.Domain, request.RequestID = domain, sequence
					return owner.Dispatch(request)
				}
				current := make([]netmodel.Result, keys)
				for key := range current {
					current[key] = dispatch(netmodel.Request{Op: "listen"})
					require.Empty(t, current[key].Error)
				}
				for round := 0; round < 3; round++ {
					for key, previous := range current {
						require.Empty(t, dispatch(netmodel.Request{Op: "close", Handle: previous.Handle}).Error)
						current[key] = dispatch(netmodel.Request{Op: "listen", Port: previous.Local.Port})
						require.Empty(t, current[key].Error)
						require.Equal(t, previous.Local, current[key].Local)
						require.Equal(t, netmodel.ErrClosed.Error(), dispatch(netmodel.Request{Op: "close", Handle: previous.Handle}).Error)
						require.Equal(t, netmodel.ErrClosed.Error(), dispatch(netmodel.Request{Op: "accept", Handle: previous.Handle}).Error)
					}
				}
				if closed {
					for _, listener := range current {
						require.Empty(t, dispatch(netmodel.Request{Op: "close", Handle: listener.Handle}).Error)
					}
				}
				require.Equal(t, netmodel.ErrResourceExhausted.Error(), dispatch(netmodel.Request{Op: "listen"}).Error)
				recorded, err := owner.Record()
				require.NoError(t, err)
				require.Len(t, recorded.Snapshot.Listeners, keys*4)
				encoded, err := json.Marshal(recorded)
				require.NoError(t, err)
				var network NetworkRecord
				require.NoError(t, json.Unmarshal(encoded, &network))
				for index, listener := range network.Snapshot.Listeners {
					if index > 0 {
						previous := network.Snapshot.Listeners[index-1]
						left, right := wire.NetworkEndpointKey(previous.Endpoint), wire.NetworkEndpointKey(listener.Endpoint)
						require.True(t, left < right || left == right && previous.Closed)
					}
				}
				config.Replay = &recorded
				_, err = netmodel.New(2, config)
				require.NoError(t, err)
				spec := validSpec()
				identity, err := hashSpec(spec)
				require.NoError(t, err)
				record, err := buildClusterRecord(spec, identity, Result{Outcome: OutcomeCompleted, Network: network})
				require.NoError(t, err)
				encoded, err = EncodeClusterRecord(record)
				require.NoError(t, err)
				decoded, err := DecodeClusterRecord(encoded)
				require.NoError(t, err)
				require.Equal(t, record, decoded)
				transported, err := json.Marshal(decoded.Network)
				require.NoError(t, err)
				var replayRecord netmodel.Record
				require.NoError(t, json.Unmarshal(transported, &replayRecord))
				config.Replay = &replayRecord
				_, err = netmodel.New(3, config)
				require.NoError(t, err)
			})
		}
	}
}

func TestListenerHistoryEqualityGroupsRequireEarlierEntriesClosed(t *testing.T) {
	endpoint := NetworkEndpoint{Node: "a", Incarnation: 1, Address: "127.0.0.1", Port: 7233}
	for _, test := range []struct {
		name   string
		closed []bool
		valid  bool
	}{
		{"closed closed", []bool{true, true}, true},
		{"closed open", []bool{true, false}, true},
		{"multiple closed then open", []bool{true, true, true, false}, true},
		{"open closed", []bool{false, true}, false},
		{"two open", []bool{false, false}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			network := emptyNetworkRecord()
			for _, closed := range test.closed {
				network.Snapshot.Listeners = append(network.Snapshot.Listeners, NetworkListenerSnapshot{Endpoint: endpoint, Closed: closed})
			}
			var err error
			network.Snapshot.Identity, err = networkSnapshotIdentity(network.Snapshot)
			require.NoError(t, err)
			encoded, err := json.Marshal(network)
			require.NoError(t, err)
			var record netmodel.Record
			require.NoError(t, json.Unmarshal(encoded, &record))
			_, replayErr := netmodel.New(1, netmodel.Config{Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 1 << 20, ConnectionBytes: 65536, StallNanos: 1000, Transitions: 8192}, Replay: &record})
			clusterErr := validateNetworkRecord(network, DefaultLimits())
			if test.valid {
				require.NoError(t, replayErr)
				require.NoError(t, clusterErr)
			} else {
				require.ErrorContains(t, replayErr, "listener snapshot is invalid or unordered")
				require.ErrorContains(t, clusterErr, "listener snapshot is invalid or unordered")
			}
		})
	}
}
