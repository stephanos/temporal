---
satisfies: [R1, R3, R7]
---
# fn-178-gomad-multi-instance-clusters-in-one.9 Go cluster client behind WASMExecution with Go-broker reference parity

## Description
Drive clusters from Go through one helper (R1, R3, R7). A new Go cluster client implements the existing `gomad3sim.WASMExecution` lifecycle over the cluster control protocol, and decodes per-instance results and World evidence into the existing consumer types. The per-process Go broker stays selectable as the end-to-end reference.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/cluster.go` (new: cluster request, control frames, result decoding), `tools/gomad/internal/backends/wasm/wasi/identity.go` (embed list), `tools/gomad/internal/backends/wasm/cluster/provider.go` (select the in-process cluster, keep the Go broker reference), `tools/gomad/internal/backends/wasm/cluster/provider_test.go`, `tools/gomad/internal/backends/wasm/wasi/integration_test.go` (Go-broker vs in-process parity)
**Touches:** [tools/gomad/internal/backends/wasm/wasi/cluster*.go, tools/gomad/internal/backends/wasm/wasi/identity.go, tools/gomad/internal/backends/wasm/wasi/identity_test.go, tools/gomad/internal/backends/wasm/cluster/provider.go, tools/gomad/internal/backends/wasm/cluster/provider_test.go, tools/gomad/internal/backends/wasm/cluster/implementation.go, tools/gomad/internal/backends/wasm/cluster/implementation_test.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go]

Integrated batch: follow the .8 amendment for the single .8–.11 owner, provisional development, acceptance dependencies and one shared handover/review. Preserve delayed/never-started instance semantics and retain the selectable Go reference. New client code and changed implementation bytes must participate in their existing source/identity owners; any additional source surface requires root coordination.

### Approach
- Map `Provider.Open`/`Start`/`Wait`/`Stop`/`Crash`/`Link`/`Finish`/`Shutdown` (provider.go:272-900) onto the cluster session:
  - Start is the cluster execute frame, or `restart{node}` for later incarnations.
  - Wait blocks on that incarnation's streamed result frame.
  - Crash/Stop with declared boundaries become config; unbound ones become live commands.
  - Finish collects per-incarnation `wasi.Result`s and World evidence.
  - Shutdown sends stop-all, then follows the existing grace and reap timings (provider.go:608-663).
- Results decode into `wasi.Result` per incarnation (stdout/stderr, transcript and digest, choice tape, storage evidence and image identity, network evidence, termination, peak memory). The Go strict decoders from fn-175, fn-176 and fn-178.13 (network evidence) are reused. Undecodable or truncated frames are an `invalid` refusal for the run.
- World evidence: decode the Rust expanded wire with the generated decoder, wrap it in `GMWORLD2` with the existing encoder, and validate with `world.Restore` before setting `Execution.World`/`WorldEncoding`.
- `bootIdentity` (provider.go:216-240): inputs unchanged per instance. The helper SHA already binds the Rust broker, World and clock. Update `internal/backends/wasm/cluster/implementation.go` only if Go files change.
- Reference selection: a test-only switch keeps the per-process Go broker path (fn-176.6 split mode). Parity test: run the broker fixtures and a two-cluster scenario through both paths and require identical per-incarnation evidence, global event order and World evidence bytes.
- Network differential (R7, moved here from former fn-177.6 and adapted to clusters): extend the parity test with `network_streams` guests in the style of `TestBrokerIndependentGuestStreamsReplay` (integration_test.go:1243). Cover partition, heal, delay, stall expiry (a partition longer than `StallNanos`), a live `ChangeLink`, an incarnation crash with open connections, and one storage-interrupt combination, over several seeds and with replay. Require byte-identical per-instance network evidence and broker `NetworkRecord` between the Go-broker and in-process paths (follow `sameEvidence`, integration_test.go:268).

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/cluster/provider.go`
- `tools/gomad/internal/model/cluster/wasm.go:27` (`WASMExecution` interface)
- `tools/gomad/internal/backends/wasm/wasi/broker.go:150-200` (`EncodeWorldEvidence`)
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:268-281,1243-1312,3594-3830` (`sameEvidence`, broker network-stream replay and comparison, two-cluster gate helpers)

## Acceptance
- [ ] Simulation provider cluster runs use one helper process; the Go-broker reference and in-process paths produce identical per-incarnation evidence, global order and World evidence on the broker fixtures and the two-cluster scenario.
- [ ] Cluster runs make no runtime-decision or cross-instance-message callbacks to Go (only lifecycle control frames), checked by a Go-side frame counter.
- [ ] The network differential fixtures give byte-identical per-instance network evidence and broker `NetworkRecord` on both paths, across seeds and with replay.
- [ ] Retained replay (`simulation/retained*` tests, `simulation/replay.go`) passes on newly produced cluster evidence; corrupt cluster frames are refused.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
