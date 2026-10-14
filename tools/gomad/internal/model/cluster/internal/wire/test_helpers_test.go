package wire

import "go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"

func validRuntimeConfig() history.RuntimeConfig {
	return history.RuntimeConfig{
		Nodes:  []history.RuntimeNode{{ID: "node", Address: "10.0.0.1"}},
		Limits: history.RuntimeLimits{NetworkListeners: 4096, NetworkConnections: 65536, NetworkBytes: 1 << 30, NetworkConnectionBytes: 4 << 20, NetworkStallNanos: 10000000000, NetworkTransitions: 1 << 20},
	}
}
