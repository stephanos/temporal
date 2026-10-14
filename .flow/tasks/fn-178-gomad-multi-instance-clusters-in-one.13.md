---
satisfies: [R6, R7]
---
# fn-178-gomad-multi-instance-clusters-in-one.13 Go cluster network configuration and strict decoding of helper network evidence

## Description
Carry the broker network configuration in the cluster execute request, and decode each instance's binary network evidence back into the document and broker network Record the Go-broker path produces, recomputing identities with the generated Go codecs (R6, R7 Go side). The Go-broker path's evidence is unchanged and stays the byte reference. Moved here from former fn-177.5 and adapted to the cluster.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/cluster_network.go` (new: request mapping and strict evidence decoder), `tools/gomad/internal/backends/wasm/wasi/cluster_network_test.go` (new), `tools/gomad/internal/backends/wasm/wasi/identity.go`, `tools/gomad/internal/backends/wasm/README.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/cluster_network*.go, tools/gomad/internal/backends/wasm/wasi/identity.go, tools/gomad/internal/backends/wasm/README.md]

### Approach
- Request: map the `BrokerConfig` network fields (`Network` limits, `Addresses`, `Links`, `StrictTopology`, `Faults`) and each node's `NetworkV2` into the cluster frame from fn-178.1 and fn-178.6. Go stays the configuration authority: run `validateNetworkModelV2` and the `newBrokerNetwork` checks before spawning the helper, so Go and Rust refuse the same configurations.
- Decoder: strict (refuse truncation, trailing data, unknown tags), with no hand-written JSON scanner (fn-153 rule). Rebuild exactly the per-instance document `NetworkV2Evidence` produces in broker mode (`network_v2.go:370-391`: profile, operations, native Record, World snapshot) and the broker `NetworkRecord` the provider reads (`simulation/provider.go:875`).
- Recompute `transition_sha256` and the snapshot identity with the generated codec (`internal/model/network/generated.go:1183-1226`). Decode the World snapshot with fn-178.4's generated decoder.
- Keep the decoder a deep module with one entry point (bytes in, document and Record out), tested in isolation against documents recorded from the Go-broker path.
- Add new Go files to `identity.go`'s embed and source list.
- README: the model table row `NetworkV2` and the "V2 network and broker runs use the generated native host model" paragraph (`README.md:114-130`). Record that in-process clusters use the Rust owner, and that the Go owner remains for single-instance runs, the native backend and the Go-broker reference path.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/network_v2.go:300-440`
- `tools/gomad/internal/backends/wasm/wasi/broker_network.go:28-107`
- `tools/gomad/internal/backends/wasm/wasi/helper_model.go:108-229` (binary stream intake and terminal byte counts)
- `tools/gomad/internal/model/network/generated.go:1183-1226`

### Quick commands
```bash
make -C tools/gomad/internal/backends/wasm helper
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestClusterNetwork' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] The decoder refuses malformed evidence (unit tests for truncation, trailing bytes, unknown tags and oversize), and identities are recomputed by the Go codec.
- [ ] Documents and Records rebuilt from encoded structure are byte-identical to ones recorded from the Go-broker path (golden round-trip tests).
- [ ] Invalid cluster network configuration is refused in Go before the helper starts, matching the Go broker's refusal.
- [ ] README states which owner serves which run.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
