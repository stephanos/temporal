---
satisfies: [R9, R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.31 Measure the multi-instance and SQLite cells World transition budgets under enforced per-owner recording before the cycle reaches them

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
### Current execution owner (2026-10-24, cleanup-first amendment)

After the narrow fn-232.30 checkpoint, complete fn-242's Rust driver implementation/cutover, then the complete approved fn-240 extraction scope, then execute fn-246-gomad-final-qualification-after-driver, tasks 3 and 7. Those tasks own production and independent receiving of this task's remaining evidence. This task and fn-232 remain open; moving the work supplies no acceptance credit. All original requirements, behavioral assertions, bounds, custody and genuine current-candidate identity checks below remain required. Later paragraphs describing .13-first execution or cycle production are historical ordering and are superseded by this amendment. No full qualification dispatch occurs before both cleanup specs finish.

Complete this original task only after fn-246.7 reconciles actual matching proof into its original acceptance; do not require the new spec to be closed first. Preserve historical failures and the frozen Go source/executable/helper/tool inputs until their actual public18 and R4 receiving comparisons pass. Re-anchor implementation paths to the crates that own them after fn-240.

Predicted blocker, same mechanism as .29/.30. Since 93be185443 every Rust World owner (broker, standalone network, storage) enforces the retained world_transition_bytes before mutation; the retired Go recorder never activated it for the broker, so these cells have never run under an enforced budget: none of workload-sqlite-cooperative-* or workload-multi-instance-cooperative-* has a receipt in any of the eight retained cycle attempts. configure.py/workloads.py give both cells 32 MiB. The onebox inspection canary measured 4,718,416 broker transition bytes for one cluster over 3 virtual minutes (idles alone 1,060,466; the testcore 100 ms metadata refresher drives them). The replication scenario (testdata/temporal_broker/replication_test.go) waits in Eventually loops of 90 s, 5 min and 10 min under a 15 min context across two Temporal clusters with replication traffic, so its broker Worlds may need well over 32 MiB. Do what .30 did: run each cell once as a separately labelled diagnostic with a higher World bound (for example 256 MiB) on current gomad, authenticate the retained World and read the per-owner transition bytes and call counts; then either confirm 32 MiB fits with stated headroom or select a measured budget with about 75 percent headroom for that cell only and update configure.py, workload_tools/workloads.py, adapters.validate_budget and QUALIFICATION.md together. Keep the generic 1 MiB capacity-refusal tests and every other workload bound. Diagnostic runs are not qualification evidence. Touches: tools/gomad/rust/driver/{configure,adapters}.py, tools/gomad/rust/workload_tools/workloads.py, tools/gomad/internal/backends/wasm/QUALIFICATION.md, .flow/artifacts/fn232-world-budget-*.
## Acceptance
- [ ] Retained per-owner transition-byte and call-count measurements for the sqlite and multi-instance success cells on current gomad, with identities and rerun commands
- [ ] Either a recorded fit under 32 MiB with headroom, or a measured budget selected and mirrored in configure.py, workloads.py, validate_budget and QUALIFICATION.md in one change
- [ ] Generic 1 MiB capacity regressions and all other workload bounds unchanged
- [ ] Both cells pass their qualify success/canary commands once standalone before the next full cycle

## Done summary
Blocked:
Execution transferred by the owner cleanup-first amendment to fn-246-gomad-final-qualification-after-driver, tasks 3 and 7, after fn-242 and the complete fn-240 scope. Original acceptance remains pending and unchanged in substance; no missing proof is credited. Unblock/reconcile this task from real receiving evidence in fn-246.7. This is an explicit sequencing dependency, not an external impasse.
## Evidence
- Commits:
- Tests:
- PRs:
