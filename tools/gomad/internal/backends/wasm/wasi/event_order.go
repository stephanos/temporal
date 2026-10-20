package wasi

import "bytes"

func (request *Request) SetEventOrderDiagnostics(enabled bool) {
	filtered := request.stressHostEnvironment[:0]
	for _, entry := range request.stressHostEnvironment {
		if entry != "GOMAD_WASM_EVENT_ORDER=1" {
			filtered = append(filtered, entry)
		}
	}
	request.stressHostEnvironment = filtered
	if enabled {
		request.stressHostEnvironment = append(request.stressHostEnvironment, "GOMAD_WASM_EVENT_ORDER=1")
	}
}

func (request Request) helperOutputLimit() uint64 {
	revision := request.diagnosticConfigurationRevision
	pinned := PinnedEngine().Configuration
	if revision == pinned+"-stress-jitter" || revision == pinned+"-determinism-fixture" {
		return (32 << 20) + (64 << 10)
	}
	return 64 << 10
}

func (result ClusterResult) HelperStderr() []byte { return bytes.Clone(result.helperStderr) }
