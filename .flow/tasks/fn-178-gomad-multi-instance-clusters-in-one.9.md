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
Go cluster client behind WASMExecution (R1/R3/R7): simulation.Provider runs clusters through one helper (wasi.OpenCluster) with the per-process Go broker kept as a test-only reference (goBrokerReference); results decode into wasi.Result per incarnation with strict frame/World/network decoding; refused starts roll back (155d6eccf3); final World/network evidence validated after output capture (7833a0bfe0); cluster.go exempted as a host process transport in the architecture check (f1bf003cbe; follow-up to split cluster_process.go recorded in spec Known follow-ups).

Evidence: TestProvider*/retained/replay (cluster-provider, retained-cluster, retained-durable-restart, busy-grouped x10 under 12 busy loops) and wasi cluster parity (identical per-incarnation evidence, World bytes, zero GuestCallbacks); network differential TestBrokerIndependentGuestStreamsReplay seeds 7/19/42 partition/heal/delay, stall, live link, open-stream crash, storage crash with replay, byte-identical network evidence and NetworkRecord (report owner rust-cluster, 3 distinct traces and runnable orders per fixture); corrupt frames refused (TestClusterRejects*). Integrated batch .8–.11 (MILESTONES batch-first). Frozen qualified candidate `e75ac7df25fe59934630510dd3bc235f47557c92` (branch `gomad-fn178-batch-20261015`, joined with gomad and fn176-close 3d9c60813f). Fast frozen driver `/home/agent/wt/fn178-tmp/qual/run-qualification-fast.sh` (same 89 stages as the serial driver, asserted) PASSED: root `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual`, 3226 s, all stages EXIT 0 except 3 keyed REUSED phase-0 passes (lint-router, lint-router-vet from window 1; rust-fmt from window 2). Source digest `sha256:c80a0428…4d53`; authenticated release helper sha256 `5278fab3…0161`. Independent batch impl review (flowctl claude, claude-opus-5-5 high, base gomad): NEEDS_WORK -> SHIP; focused re-reviews of post-SHIP commits: SHIP (27c51cc28a..27822d5e86, P3s fixed in b4aad2b1c7) and NEEDS_WORK -> SHIP (b4aad2b1c7..e75ac7df25). Handover: `.flow/tmp/handovers/fn178-batch-summary.md`. Native execution remains deferred (fn-128/fn-149).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: adcafa6327, b34d77aff2, 0fc4e3dca5, 1972c44aef, 4d9f85cea0, 155d6eccf3, f1bf003cbe, 7833a0bfe0
- Tests: frozen driver /Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual PASS (cluster-provider, retained-cluster, retained-durable-restart, busy-grouped x10, cluster-functional, network-streams, ordinary), go test -tags test_dep ./cluster (exit 0), go test -run TestBrokerIndependentGuestStreamsReplay ./wasi (exit 0)
- PRs:
### Detail (re-appended after flowctl done, 2026-10-15)
- Qualified frozen candidate e75ac7df25; driver stages log `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/stages.log`, per-stage `.command`/`.log`, lanes A/B/C 1319/2291/2165 s, total 3226 s. Toolchains: stock Go 1.27.1 (sha256 1675694e…), rustc 1.94.1. Host Linux/aarch64, boot e1418388.
- Refused-start rollback red/green: `.flow/tmp/fn178-resume2/` (refused-start-red: 'refused restart kept Go state'). Final-evidence capture red: `.flow/tmp/fn178-resume2/redgreen/clusters-dev-v11.log` ('command output pipes remained open after cleanup'); green `.flow/tmp/fn178-resume2/redgreen/clusters-dev-v11b.log`, `.flow/tmp/fn178-resume2/redgreen/clusters-dev-v11c.log`. Architecture red: `.flow/tmp/fn178-resume2/redgreen/ordinary-architecture-red.log`; green `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/ordinary.log`.
- Network differential report: `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/network-streams/broker-network-streams-report.json` (owner rust-cluster, distinct traces/orders 3/3/3).
- Known follow-up (conductor-accepted): split the cluster transport into `wasi/cluster_process.go` so `cluster.go` parsing stays pure-checked (spec Known follow-ups).
