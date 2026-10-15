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
# Task 7 handover

Implemented and qualified the durable-volume milestone gates while retaining
the Go reference projection/oracle. Actual-owner RED/GREEN regressions closed
refused-run prefix loss with explicit partial-v2 evidence, then closed the
review-discovered history bound gap without widening complete-v1 or image caps.
Strict terminal, envelope, digest, prefix, stage and malformed-data refusal
controls remain enforced. Source correction: `458c0be2d35ecaf401d8ecd6e176f6be497c5a23`.

Current frozen acceptance is recorded in
`.flow/artifacts/fn176-frozen-qualification-2026-10-15.md`; the derivation and
focused RED/GREEN observations are in
`.flow/artifacts/fn176-task7-partial-evidence-budget-2026-10-15.md` and
`.flow/artifacts/fn176-task7-acceptance-2026-10-15.md`.
The exact 84 commands/raw logs and independently audited terminal pass are in
`/Users/stephan/Workspace/skunkworks/gomad-fn176-history-qualification-20261015.Uq7Ppu`.
All stages exited zero, final PASS 2026-10-15T10:47:40Z. Source digest
`sha256:759c29aeccd974b3044038b9b8136fe68d1b9d042c8ada8ab555289d6785035e`,
admitted helper `603091fc560fa9f8271a2a8a21053d6432d2d826d067bd8ea753394cfae027c5`,
Go model `sha256:6f94dd4169ba930d44afd121c7ba13a4e7fb1f6c0601c61df9aa7b9381ed2165`.
Five primary successes qualify; five canaries are expected unqualified.
Four actual WASM SQLite selected states/catalogue passed; actual native execution
remains deferred under fn128/fn149. Earlier bfb53 qualification is historical
only; detached/stopped attempts are not passes for the current source.

Baseline: green (Rust-volume/conformance receipts cfa09430; canonical focused Go
89.563s). Verification classified FULL; the conductor-owned fresh frozen run
executed all required gates for this source, with command/exit records rather
than fabricated focused-as-full receipts. Expanded correction Go checks passed
99.514s; final affected evidence/identity 27.422s; escaped metadata/header tests
0.004s; consumer/architecture, conformance, generated validation and final lint
passed. Tests were not repeated merely to observe their exit status.

stage: impl-review - ran (round 1 NEEDS_WORK; corrected-source round 2 SHIP)

Round 2 SHIP timestamp: 2026-10-15T10:49:54.845152Z; R1–R4 met, no blocking
findings. Nontrivial correction captured in memory entry
`bug/integration/bound-refused-storage-history-2026-10-15` (moderate overlap,
retained as a distinct refused-history issue with related-entry links).
Final `flowctl validate --all` exited zero; historical warnings remain recorded.

Receipt: `/tmp/impl-review-receipt-657da2bc4466-fn-176-gomad-durable-volume-crash-model-in-the.7.json`.
Historical review base: `2237e0276f50f3b4b4b244f32a564ec17caca110`.
Explicit reviewer: codex:gpt-6.1-sol:high, same GPT family disclosed by conductor.
Round 1 contracts finding retained/consumed; integration SHIP; correctness draw
failed without verdict and was not retried/refunded. Re-review uses the same
receipt, never a fresh fan-out. Both earlier quality-axis reports are retained
verbatim with dispositions; five size-charter notices did not warrant broad
splits. Whole-spec completion review remains conductor-owned.

All 18 external planning paths remain hash-identical and unstaged. Parent-owned
fn151/fn178 tracking commits and intended fn178 MILESTONES changes are preserved;
they are not represented as task-7 source. Tracker inactive; planSync false.

Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4ac1cd7eb9a0e000c22895cee169b23dfe48b0ea, e21d275a9a7bca9f179ba7725192a9bdde7f0513, bfb53b2c4585f39b70e0e361c45f3c3b2af49e5e, 3aeea7936d780a5181ea933f59875fb138d8fd04, a3060077478a9821ab173fab6d1744ffe48c5d39, 4d2a77bb0245c6d639690bff015f3a199536af49, b3707e102090f83bf25690f032564a6857ccca4c, 458c0be2d35ecaf401d8ecd6e176f6be497c5a23, 53fb6ca139731adc1396717d17ffeb82d88fbaf9, c5c08528e79f1822448f4cb236d5c35bcff9a6d2
- Tests: Frozen driver: /Users/stephan/Workspace/skunkworks/gomad-fn176-history-qualification-20261015.Uq7Ppu/run-qualification.sh 458c0be2d35ecaf401d8ecd6e176f6be497c5a23 (84 exact stage commands in *.command; every exit 0; final PASS 2026-10-15T10:47:40Z), make test-wasm GOMAD3_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go (frozen tools/gomad cwd, ordinary.command), go test -tags test_dep -count=1 -run TestStorageEvidencePartialMetadataScalarBounds|TestStorageEvidencePartialHeaderBudgetRecognition ./internal/backends/wasm/wasi (tools/gomad cwd; exit 0), make lint-code-fast GOLANGCI_LINT_BASE_REV=a1f0be7e96e2746cb9c51598766a5ae96d8552c2 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/golangci-lint-v2.13.0 ERRORTYPE=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/errortype, flowctl validate --all (exit 0)
- PRs: