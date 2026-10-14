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
`wasi/cluster_network.go`: `ClusterNetworkFrame` runs `NewBroker` (so Go refuses exactly what `newBrokerNetwork` refuses: per-node fault schedule, bad IPv4, self/unknown/duplicate link endpoints, unordered/duplicate faults, unknown actions, faults without network) plus the per-node attach rules (stream profile present, no local faults, cooperative runtime) and the cluster's own rules (no node profile without a cluster network, incarnation 1), then maps the broker network fields into the cluster frame. `DecodeClusterNetworkEvidence` is one deep entry point (bytes in; broker-mode document, Record and network World out): strict GMNETEV1 decode (truncation, trailing bytes, unknown tags, non-minimal varints, bad flags, invalid UTF-8, oversize, log accounting mismatch all `invalid`), operations log rebuilt with Go's own types and checked against the helper's Go-JSON byte count, Record identities recomputed through the generated codec's replay admission (`netmodel.New` with `Replay`), network World replayed with the new `world.DecodeExpandedSnapshot`, document built by the same `networkV2Document` that `NetworkV2Evidence` uses (bounded as `network-v2-evidence-bytes`). Golden test: the helper's checked-in evidence for a cross-node exchange (pending read, World readiness, poll, stale cancel) rebuilds documents and a Record byte-identical to the Go broker path running the same calls. New file added to the WASI identity embed/list (pin rotated). README states which owner serves which run. Production callers arrive with the Go cluster client in fn-178.9.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b5ccd7a65f, 050839393d, ebd685a55c
- Tests: go test -tags test_dep -run 'TestClusterNetwork|TestImplementationIdentity|TestNetworkV2' ./internal/backends/wasm/wasi (exit 0), go test ./internal/model/world/... and TestWorldConformance* (exit 0, World corpus regenerated for the new source hash), make lint-code-gomad-wasm GOLANGCI_LINT_BASE_REV=418c95a30b (exit 0 after errcheck fixes), make -C tools/gomad test-wasm (exit 0 at 050839393d-era tree; rerun on final tree recorded in report), flowctl claude impl-review fn-178.13 --spec claude:claude-opus-5-5:high: SHIP (4 P3 follow-ups applied in ebd685a55c)
- PRs: