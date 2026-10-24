---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.14 Strict cycle configuration and frozen source/build authentication

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/build/**, tools/gomad/rust/gomad/src/cycle/config/**, tools/gomad/rust/gomad/src/cycle/custody/**, tools/gomad/rust/gomad/src/cycle/preparation/inputs/**, tools/gomad/rust/gomad/src/campaign/prepare/inventory.rs

## Description
strict configuration and frozen source/build authentication

Depends on: cycle-foundation; cycle-process only when executing bounded build/identity commands. Original .2-owned IDs: 48 = 42 pending/six keep. Also consume four originally .9-owned dry_run checks delegated by the sibling split manifest, as one proof owner; do not drop them or duplicate implementation. The entire dry_run.py endpoint (all six functions) belongs here; .9 fresh preparation calls it.

Description: load one bounded original config byte buffer, authenticate requested SHA256 and strictly decode the same bytes; guard candidate source/config/roles through stages. Preserve exact source manifest, HEAD, tracked inventory, frozen files, symlinks and permitted active scratch semantics. Replace dry-run role build/admission/custody with real Rust implementation and real Cargo observation authentication, retaining original/current source distinction. No receipt-shaped synthetic success.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-custody`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

Additionally own these four originally .9-owned dry_run predicate proofs (52 total checks): `workload-callback:dry_run:boolean_return:32:4`, `workload-callback:dry_run:boolean_return:36:4`, `workload-callback:dry_run:boolean_return:41:8`, `workload-callback:dry_run:boolean_return:43:8`. Original .9 aggregate waits this task; no proof is credited by ownership transfer.

## Acceptance
- [ ] Refuse duplicate/nonfinite/overbound/malformed configuration, invalid schema/checkout/stages/pending endpoints/final-reader bindings, synthetic qualification claims, wrong host/helper roles/digests, invalid caps/send epoch and changed original config bytes.
- [ ] Frozen source guard authenticates nonempty absolute regular nonsymlink files and hashes, complete non-Flow tracked source set, actual HEAD, exact symlink inventory and manifest; dirty paths permitted only by currently active owned scratch registry intervals. Stable handle/containment/concurrency guards cannot be trimmed.
- [ ] Authenticate role/features/compiler/target/build identity from actual bounded Cargo records and pinned files; reject foreign, missing, duplicate, stale, unfinished, changed binary or incompletely sealed source/role inputs. Build paths are fresh and actual command errors retain nonzero classification.
- [ ] Existing six keep checks remain covered; all 42 pending plus delegated four have executed source-bound catching/control proof. Use bounded representative build observations, no full profile matrix or qualification.
- [ ] Source/frozen identity primitives needed by .3/.4/.9/.10 are usable without depending on aggregate .2. Actual fresh configuration production stays .9; accepting/guarding it is this task.

## Done summary
Implemented B01 source/configuration/build custody and immutable preparation inputs as one reviewed batch. This member owns source/config/build authentication, actual descriptor/source guards and genuine role publication/admission. Exact member references remain in the original compact batch map; counts {"check_refs": 52, "function_refs": 6, "endpoint_refs": 1}. Final source fix28c02f4fb118 integrated03832e4d5d; final full tracked Rust tree equals tested worker source. Current gates pass:67 cycle library tests, source-owner cases, genuine preparation/GNU controls, actual two-host-role overwrite/admission controls, scoped Clippy and formatting. Both review findings have direct red-to-green proofs. All1411 new recorded PIDs independently absent; original2797 were separately audited. Same-session re-review SHIP at /tmp/impl-review-receipt-67d1eda7ff47-fn-242-gomad-shrink-the-qualification.14.json. Proof .flow/artifacts/fn242-custody-inputs-review-fix-proof.json plus external compact delta/handoff. This closes the library/family boundary only; actual global CLI startup/routing remains B15. No full96/final145/global qualification attempt; operational journal binding remains absent and fn232.33 remains a receiving prerequisite. Legacy compatibility policy applies.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e79ca50771, 03832e4d5d, d53f9e5e0b
- Tests: cargo +1.99.1 --config profile.release.package.gomad.opt-level=0 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle:: -- --nocapture --test-threads=1, CUSTODY_FAMILY=source/inputs/build cycle_custody_owner: exact separate commands and numeric exits in review-fix/proof-map.json, cargo +1.99.1 --config profile.release.package.gomad.opt-level=0 clippy --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib --no-deps -- -D warnings, cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check, git diff --check
- PRs:
## Owner-authorized batch B01 (custody-inputs)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.14, fn-242-gomad-shrink-the-qualification.16. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.

## Generated Go test-main inventory integration

B01 additionally owns the exact campaign inventory file in Touches solely to expose the existing cache_output_name validator as pub(crate). Reuse its content-addressed Go cache contract for genuine go-test generated main files, authenticate actual bytes and stable reads, preserve the frozen virtual source key and exact -test argv, and reject other absolute/unclean source names. Keep all original acceptance and proof ownership. This does not implement or complete fn-232.33.
