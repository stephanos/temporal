---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.25 Close retirement, public parity, identity and source-size ledgers

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Owner amendment, 2026-10-25 (size counting is informational).** The `gomad-source-size/v1` counter crate and `tools/gomad/docs/source-size.md` are deleted. Record sizes with `tokei` on the named revisions, listing the path groups (production, tests, generated) next to the numbers. Size never gates or blocks closure. Requirements below for the original counter, counter versions, blob hashes/authentication and the three-path classifier correction are retired; earlier reports stay as history.

Touches: .flow/artifacts/fn232-retirement/source-size-groups.py, .flow/artifacts/fn232-retirement/source-size-2026-10-23.md, fn-232.13 common closeout ledger, MILESTONES.md

## Description
### Current execution owner (2026-10-24, cleanup-first amendment)

After the narrow fn-232.30 checkpoint, complete fn-242's Rust driver implementation/cutover, then the complete approved fn-240 extraction scope, then execute fn-246-gomad-final-qualification-after-driver, tasks 6 and 7. Those tasks own production and independent receiving of this task's remaining evidence. This task and fn-232 remain open; moving the work supplies no acceptance credit. All original requirements, behavioral assertions, bounds, custody and genuine current-candidate identity checks below remain required. Later paragraphs describing .13-first execution or cycle production are historical ordering and are superseded by this amendment. No full qualification dispatch occurs before both cleanup specs finish.

Complete this original task only after fn-246.7 reconciles actual matching proof into its original acceptance; do not require the new spec to be closed first. Preserve historical failures and the frozen Go source/executable/helper/tool inputs until their actual public18 and R4 receiving comparisons pass. Re-anchor implementation paths to the crates that own them after fn-240.

Size M. Own the common receiving/identity/size ledger to avoid parallel edits. First integrate independently audited classifier fix1f9a2ef58fa55fd88e8ada907b04c3890f99174c; preserve original reports and append the separate three-baseline-path correction. Reuse actual source-size counter and path lists, internal/archcheck/retired_test.go:20, Makefile:37/:87 and final_public_verbs.py:171/:207. Review existing retirement/default-import protection, single Cargo workspace/lock and release exclusions, Rust Make/CI/docs routing and all18 public verbs with actual same-cycle raw proof. Consume bounded receiving receipts from runtime, retained/ringpop and R4 owners. This task can prepare source/arithmetic review before cycle completion and does not depend on .13 being done. Parent .13 retains every original acceptance criterion and actual all96/final145 verdict.
## Acceptance
- [ ] Integrate the exact-parent classifier correction with independent table match: three deleted baseline host tests move from retained Go to retired host, totaling3files/12374bytes/312code lines; original counter, candidate rows and repository totals remain unchanged. Preserve originals and append correction provenance.
- [ ] Verify retirement/import/default-set, workspace/lock, release exclusions and Make/CI/docs against current source and actual passing checks; verify all18 public verbs, raw command integrity, own-lineage comparisons and unchanged goldens. Source wiring or installation preparation alone cannot prove behavior.
- [ ] Append actual final execution identities and all receiving reconciliations, distinguishing executable/source/helper/runtime/model/config/engine/compiler identities and historical rotation causes. Incorporate exact final source-size revisions/counter/path lists, production/tests/generated/support partitions and net totals without replacing history.
- [ ] Reconcile terminal source/original807/sealed806 custody and final evidence references; update MILESTONES/task presentation without changing scope/order. Missing evidence, unexplained rotations, mismatched hashes or unexecuted claims remain incomplete. The39 reach gaps remain .14-owned and uncredited; post-cycle stress/fn233/fn221/fn242 ordering remains unchanged.

## Done summary
Blocked:
Execution transferred by the owner cleanup-first amendment to fn-246-gomad-final-qualification-after-driver, tasks 6 and 7, after fn-242 and the complete fn-240 scope. Original acceptance remains pending and unchanged in substance; no missing proof is credited. Unblock/reconcile this task from real receiving evidence in fn-246.7. This is an explicit sequencing dependency, not an external impasse.
## Integrated source correction (2026-10-24)

Grouping commit1f9a2ef58fa55fd88e8ada907b04c3890f99174c is integrated through0337bda4bbdd67a5d185807845ec6ba7100d58da. Independent audit and focused RED/GREEN confirm exactly three baseline host tests move from retained Go, totaling3files/12374bytes/312code lines; corrected table matches the independent script byte-for-byte. Candidate partitions, original raw counter and repository totals remain unchanged. Source/evidence hashes are in /home/agent/wt/evidence/fn232-size-grouping-correction/handoff.json and /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/independent-size-audit-cycle5.json. The combined mandatory gate on clean0337 exits0 (589 driver tests plus Rust/WASM checks); implementation reviewers independently verified the classification correction. Preserve original reports/patches. Final current-source measurement, actual public/receiving proof and qualified ledger append remain pending; this task stays in_progress.

## Evidence
- Commits:
- Tests:
- PRs:
