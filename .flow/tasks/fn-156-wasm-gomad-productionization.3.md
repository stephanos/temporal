---
satisfies: [R3]
---
# fn-156-wasm-gomad-productionization.3 File-backed SQLite faults and crash recovery

## Description
Original fn-151 Phase 6: reuse volume semantics and stable fault matching, distinguish visibility/durability, fresh guest crash/reopen, independent history/convergence oracles, replayed failure canary and negative/reference checks.

The provisional implementation batch projects unchanged authoritative native filesystem/volume sources into an importable host package; projection generation fails closed on source/transport anchor drift. WASI rights remain adapter-owned, `fd_sync` reaches actual modeled durability, file-backed SQLite reads/writes/syncs cross the guest seam, crash discards volatile handles and state, and recovery receives an identity-bound durable image. Optional storage configuration is omitted from old profile canonical bytes. Preserve native overlay bytes.

**Touches:** `tools/gomad3/deterministicio/volume/**`, `tools/gomad_wasm/wasi/namespace.go`, `tools/gomad_wasm/wasi/volume*.go`, `tools/gomad_wasm/testdata/storage*/**`

Shared environment/protocol integration is serialized with the helper owner. Final admission follows `.2`; implementation may develop provisionally under the batch-first strategy. Gates include generated projection equality and source preservation, reference volume/fault sequences, actual file-backed SQLite fresh-guest crash/recovery, negative durability canary and exact replay, then Temporal recovery with detached history oracles.
## Acceptance
Original fn-151 Phase 6 exit gate.

## Done summary
Temporal committed-workflow crash/reopen and SQLite durable success/canary workload cells pass on 9d6012b52b. SQLite supplementary selection gates require a native crash catalogue and remain native-owned (fn-128/fn-149).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9d6012b52b
- Tests: TestTemporalCommittedWorkflowFreshGuestCrashReopen, TestActualSQLiteDurableQualification success/canary, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: