package simulation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

func TestInspectionNetworkRawStreamBindsCanonicalIdentity(t *testing.T) {
	raw, err := os.ReadFile("../wasmhost/tests/testdata/cluster_network/a.gmnetev1")
	require.NoError(t, err)
	network := &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 8192}}
	limits := wasi.ClusterNetworkLimits{Network: network.Limits, Instance: *network, TranscriptBytes: 1 << 20}
	canonical, err := wasi.DecodeClusterNetworkEvidence(raw, limits)
	require.NoError(t, err)
	handle := cluster.NodeHandle{Node: "a", Incarnation: 1}
	execution := &cluster.WASMExecutionEvidence{Incarnations: []cluster.WASMIncarnationEvidence{{Handle: handle, NetworkSHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(canonical.Document))}}, Inspection: &cluster.WASMInspectionEvidence{Complete: true, Incarnations: []cluster.WASMInspectionIncarnation{{Handle: handle, Streams: cluster.WASMInspectionStreams{NetworkEncoding: "gomad.cluster-network-binary/v1", Network: raw}}}}}
	p := &Provider{Broker: wasi.BrokerConfig{Network: network}, Boots: map[cluster.BootID]wasi.Request{"boot": {Config: wasi.Config{NetworkV2: network, Limits: wasi.Limits{TranscriptBytes: 1 << 20}}}}}
	spec := cluster.Spec{Nodes: []cluster.NodeSpec{{ID: "a", Boot: "boot"}}}
	require.NoError(t, validateInspectionNetworks(execution, p, spec))
	execution.Incarnations[0].NetworkSHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	require.ErrorContains(t, validateInspectionNetworks(execution, p, spec), "canonical network identity")
	execution.Inspection.Incarnations[0].Streams.Network = append(append([]byte{}, raw...), 0)
	require.ErrorContains(t, validateInspectionNetworks(execution, p, spec), "raw network stream")
	execution.Inspection.Incarnations[0].Streams.Network = raw
	execution.Incarnations[0].NetworkSHA256 = fmt.Sprintf("sha256:%x", sha256.Sum256(canonical.Document))
	execution.Inspection.Incarnations[0].TerminalOrder = 1
	commitment, err := cluster.WASMInspectionStateCommitment("network", raw)
	require.NoError(t, err)
	execution.Inspection.Boundaries = []cluster.WASMInspectionBoundary{{Order: 2, Cuts: []cluster.WASMInspectionCut{{Handle: handle, StateEncoding: cluster.WASMInspectionStateEncoding, NetworkState: append([]byte{}, commitment[:]...)}}}}
	require.NoError(t, validateInspectionNetworks(execution, p, spec))
	execution.Inspection.Boundaries[0].Cuts[0].NetworkState[0] ^= 1
	require.ErrorContains(t, validateInspectionNetworks(execution, p, spec), "terminal network state commitment")
}
