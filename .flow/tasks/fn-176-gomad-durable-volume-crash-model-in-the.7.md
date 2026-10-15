---
satisfies: [R3, R4]
---
# fn-176-gomad-durable-volume-crash-model-in-the.7 Durable gates with the Rust volume, Go reference retention and docs

## Description
Pass the durable gates with the Rust volume, settle what Go keeps, and update the docs (R3, R4).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/integration_test.go` (gate assertions), `tools/gomad/internal/backends/wasm/cluster/provider.go` (dead `crash-after` branch), `tools/gomad/internal/backends/wasm/wasi/volume.go`, `tools/gomad/internal/archcheck/edges.go`, `tools/gomad/internal/backends/wasm/architecture_test.go`, `tools/gomad/internal/backends/wasm/README.md`, `tools/gomad/internal/backends/wasm/CLI.md`, `tools/gomad/internal/backends/wasm/QUALIFICATION.md`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`, `tools/gomad/README.md`, `.flow/artifacts/` (gate record)
**Touches:** [tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/cluster/provider.go, tools/gomad/internal/backends/wasm/wasi/volume.go, tools/gomad/internal/archcheck/edges.go, tools/gomad/internal/backends/wasm/architecture_test.go, tools/gomad/internal/backends/wasm/README.md, tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/README.md, tools/gomad/internal/backends/wasm/wasi/storage_evidence.go, tools/gomad/internal/backends/wasm/wasi/storage_evidence_test.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/identity.go, tools/gomad/internal/backends/wasm/wasi/identity_test.go, tools/gomad/internal/backends/wasm/wasmhost/src/storage_transport.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, .flow/artifacts/**]

### Approach
- Gates (run alone; host memory budget):
  - `TestTemporalCommittedWorkflowFreshGuestCrashReopen` (integration_test.go:2067, needs `GOMAD_WASM_TEMPORAL_RECOVERY_ROOT`): enumeration complete, and per-state reopen with identical stdout, storage transcript and interrupted choice bytes.
  - `make -C tools/gomad/internal/backends/wasm qualify-workloads`, SQLite durable `success` and `canary` cells (`TestActualSQLiteDurableQualification`) with the env from the Makefile.
  - The Go fs-served counter must be 0 (non-Broker) or fd-1/2-only (Broker) in both.
- Owner scope correction (MILESTONES.md, immediate delivery order): SQLite selected-state and catalogue controls execute WASM, including the projection through native_crash.go, and remain mandatory. Run all four selected states plus the selection catalogue on newly produced authenticated source results. Only actual native execution remains deferred under fn-128/fn-149.
- Retained-artifact replay: run the existing retained replay selectors (`internal/backends/wasm/cluster` retained tests and `TestProviderActualCrashDurableRestart`) on newly produced evidence.
- What Go keeps (decision):
  - The generated `tools/gomad/internal/model/volume` projection stays. It is the R2 oracle and supplies decoder types, it is drift-tested against the native overlay, and it costs nothing to maintain.
  - The Go namespace and volume execution path in `wasi` stays only as the `remoteModel` reference and for stock-profile runs.
  - No production helper-owned or Broker run executes it. Assert this with the fs-served counter in the gates.
  - This deviates from "Go keeps only a decoder", because R2 needs a Go oracle.
- Delete the dead `crash-after` branch in `simulation/provider.go:741,774` (it is unreachable because `validateStorageModel` rejects the action), with a test.
- Update `internal/archcheck/edges.go` and `internal/backends/wasm/architecture_test.go` only if imports changed.
- Docs:
  - README "Storage volumes" row and storage interrupt text (README.md:28,50,120-128)
  - protocol.md storage config, evidence streams, image frames and helper-originated `storage_crash`
  - CLI.md storage sections (324-339, 373-378)
  - QUALIFICATION.md SQLite rows (31-33, 129-151)
  - a Rust sentence in `tools/gomad/README.md:25-31`
- Record gate results under `.flow/artifacts/`.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:1635,1831,2067`
- `tools/gomad/internal/backends/wasm/Makefile:55-80` (qualify-workloads)
- `tools/gomad/internal/backends/wasm/qualification/sqlite_durable_test.go`
- `.flow/artifacts/wasm-section-acceptance-2026-10-12.md:50-80`


### Pre-freeze refused-run evidence parity (conductor coordination)

- Reproduce the cross-task audit's partial-evidence advisory before freezing: final image-capacity failure and post-run enumeration failure currently appear to discard the helper's already-collected storage transcript, snapshots and operations, whereas Go retains them. Inspect `.flow/tmp/fn176-cross-task-quality-2026-10-15.md`; protocol.md promises collection at every terminal after attachment.
- Compare actual Go-owner and helper-owner results on deterministic refused-run fixtures. If the gap is proven, preserve the same collected prefix and terminal classification in the helper/strict Go decoder. Do not document the parity gap away or weaken successful-run image/identity requirements, malformed-evidence refusals, or the independent recovery-image bound.
- The added codec, transport, engine and identity test paths in Touches are admitted only for this focused contract correction, its regressions and required source identity updates. Coordinate any further source scope before edits.
- Retain both quality-axis reports verbatim in a task artifact with the conductor dispositions. Five mechanical size notices are not delivery blockers; no broad source or fixture split is requested.

### Frozen qualification coordination

- First finish all necessary source and tools/ documentation corrections, execute focused verification and commit them on gomad. Signal source-ready to the conductor; no tools/ source or documentation writes after the freeze without a coordinated new candidate.
- The conductor owns the frozen driver, worktree, process and actual gate evidence. The prepared `lxd3oV` attempt detached before any stage and supplies no pass; its `DISPOSITION.md` is retained. The accepted replacement is `/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-live-20261015.1hg1RG`, driver SHA256 `5f75b4ff9bd227abfe547ea7587789e84a7d5a9bdc3ed59f26bfcf6dc3b181cd`, native TMPDIR `/tmp/gomad-fn176-qualification-live.Uy7SIz`. Frozen source `bfb53b2c4585f39b70e0e361c45f3c3b2af49e5e` completed exit 0 at 2026-10-15T09:02:41Z: all 84 stages exited zero and final source/compiler/helper/primary preservation passed. Exact commands and full bindings are retained in the root and `.flow/artifacts/fn176-frozen-qualification-2026-10-15.md`. Lint base remains `a1f0be7e96e2746cb9c51598766a5ae96d8552c2`.
- This includes portable/source/static/pin/preservation checks, full ordinary WASM, retained replays, terminal/host admission, runtime 800 recordings and 32 replays, artifact/operator, committed crash/reopen, network, independent clusters, ten primary workload cells and aggregate, four SQLite selected states and catalogue, plus source/compiler/helper preservation.
- Do not review or mark this task done until required current-candidate gates pass. Old source-500e4d stopped qualification is historical, not acceptance. Observation timeout does not authorize restarting a live process. Native execution remains deferred; no push, PR or CI authority is granted.
- After a green frozen run, final receipts and evidence references belong in .flow/ and MILESTONES.md (outside the driver's source manifest), not a post-freeze tools/ edit. If a genuine production correction is required after freezing, the conductor decides when to stop the exact live process and prepare a new candidate and fresh evidence root.

### Review round 1 correction and new freeze

The completed `1hg1RG` run qualifies historical source `bfb53b2c45` only. Review round 1 returned NEEDS_WORK for permitted refused histories larger than the image-derived storage-state budget. The correction uses a call/file-derived finite allowance only for recognized partial-v2 streams, retaining complete-v1 and independent image caps. Derivation, actual-owner red-to-green proof and admission controls are retained in `.flow/artifacts/fn176-task7-partial-evidence-budget-2026-10-15.md`. The same review receipt remains active with round 1 consumed; no formal re-review runs before a fresh full frozen pass.

The conductor prepared an unstarted replacement driver `/Users/stephan/Workspace/skunkworks/gomad-fn176-history-qualification-20261015.Uq7Ppu/run-qualification.sh`, SHA256 `a59a3bbd2f18f7c8e1b4b4f5a9a9c589d608fd171ae743bed6575f270451237a`, with native scratch `/tmp/gomad-fn176-history-qualification.2dBvMe`. All 84 selectors/contracts are unchanged. The conductor advances the clean frozen worktree and owns launch/process/evidence after corrected source readiness. Historical bfb53 evidence cannot qualify changed Go source/model identities.
## Acceptance
- [ ] Temporal committed-workflow crash/reopen gate passes with the Rust volume (complete enumeration, every reopen identical).
- [ ] SQLite durable `success` and `canary` cells pass via `qualify-workloads`.
- [ ] Retained replay selectors pass on newly produced storage evidence; corrupt evidence is refused (not silently passed).
- [ ] Docs updated; gate record retained with commands, digests and host.
- [ ] Refused-run storage evidence parity is demonstrated for final image-capacity and post-run enumeration failures: retain the Go-equivalent collected prefix and unchanged terminal classification, without weakening ordinary image/identity or malformed-evidence refusals.
- [ ] All four actual WASM SQLite selected states and the authenticated selection catalogue pass on new current-candidate evidence; native execution remains deferred.
## Done summary
Docs (wasm README/CLI/QUALIFICATION, protocol.md storage/split/driver, tools/gomad README); dead `crash-after` provider branch removed with test; gates assert Go fs-served only fd 1/2 writes (Temporal gate helper, SQLite durable cells). Temporal committed crash/reopen gate passes on the Rust volume (non-frozen run). Frozen-candidate requalification (incl. qualify-workloads SQLite success/canary, selected-state and catalogue controls, retained replays) pending conductor go-ahead.

## Evidence
- Commits: branch `volume-batch` (rebased on gomad 7406c9ed9e); see final report for tip
- Tests: see `.flow/artifacts/fn176-source-gates-2026-10-15.md`
- PRs:
