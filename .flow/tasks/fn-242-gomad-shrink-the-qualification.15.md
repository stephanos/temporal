---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.15 Fixed global attempt journal and reservation-time admission floors

**Touches:** tools/gomad/rust/gomad/src/cycle/admission/**, tools/gomad/rust/gomad/src/cycle/journal/**, tools/gomad/rust/gomad/src/cycle/scheduler/admission.rs, tools/gomad/rust/gomad/src/cycle/scheduler/mod.rs

## Description
fixed global two-attempt journal and reservation floors

Depends on: cycle-foundation; cycle-process for the real spawn boundary seam. Original .2-owned IDs: 24 pending (host admission16, producer-memory8), plus policy-ambiguous-spawn, policy-fixed-enrollment, policy-reservation-resource-floors.

Description: preserve host/scratch filesystem admission and strict live-memory parsing; implement one fixed, locked durable attempt journal across candidates/outputs/restarts. Bind enrollment identity without replacement/reset. Reserve only after rechecking all three serving filesystems immediately before actual dispatch. No test performs full dispatch or consumes either real allowance.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-admission`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

## Shared parser integration

B02 also owns the two exact scheduler files listed in Touches for a narrow shared-parser refactor. Add `parse_mem_available_bytes` as the single strict unsigned-byte authority with checked KiB conversion and malformed/duplicate/unsupported-field refusal. Keep the existing floating-point GiB API as a wrapper and preserve normal scheduler behavior. Admission consumes the byte API without a second parser. The reviewed scheduler implementation is already present in this batch base; run its existing tests alongside the admission tests. This extends integration scope, preserving every original acceptance and proof owner.

## Acceptance
- [ ] Host/evidence/scratch canonical containment, distinct explicit temporary children, forbidden serving filesystems and fixed probe failure behavior retain original thresholds and custody. Strict decimal/MemAvailable bounded parsing rejects overflow, missing/duplicate/non-kB/unsupported/error/below-bound observations.
- [ ] Inject a real observable bounded child then interrupt before PID publication. Unknown spawn outcomes retain the reservation/charge across restart; two charged starts/ambiguities refuse third across candidates/output dirs. Only definitive authenticated no-child evidence releases a reservation. Reap all fixture children.
- [ ] Explicit initial enrollment is the only creation path. Missing/mismatched policy/path/inode/digest/journal, symlink replacement or retained started-output/high-water state refuses before reservation/spawn and never restores allowance; concurrent lock contenders cannot create histories.
- [ ] After initial/preparation pass, lower HOST45GiB, overlay20GiB or inner12GiB by one byte before reservation: each fails before reserve/spawn without charging. Exact floors pass. Probe errors fail closed; retain original overlay filesystem/path containment policy. Floor failure cannot be deferred-perf success.
- [ ] Actually execute all24 check proofs and all policy fixtures/control cases using deterministic filesystem/process seams and temporary fixture-only journal roots. Preserve the actual global journal and zero-used/two-remaining state untouched.

## Done summary
Implemented fixed, locked attempt enrollment and journal accounting with authenticated no-child refunds, charged ambiguous/created outcomes, strict byte admission and immediate reservation-floor rechecks. Evidence and declared overlay paths require overlayfs at preparation and reservation. All24 original check IDs and three policy guarantees have executed controls/catching proof. Eight filesystem regressions reproduce the reviewed defect before the fix and refuse after it.

Validation: 30 cycle library tests and three strict process-owner fixture groups passed; scoped Gomad Clippy, formatting and diff checks exited0. The conductor verified exact integrated source/dependency equivalence and reused those gates. Independent same-receipt re-review returned SHIP. All31 logged child PIDs and24 fixture roots independently absent.

No full qualification or operational journal mutation occurred. The real binding remains absent; fn246 owns verified initial enrollment and final qualification after fn242/fn240. CLI startup integration remains B15. Proof: .flow/artifacts/fn242-attempt-admission-proof.json and fn242-attempt-admission-review-fix-proof.json.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3d942494c8, e00f6468e4, 2722976d77, f1f6471fff
- Tests: cargo +1.99.1 --config 'profile.release.package.gomad.opt-level=0' test --locked --release -p gomad --lib --test cycle_journal_owner cycle:: -- --test-threads=1 --nocapture, cargo +1.99.1 --config 'profile.release.package.gomad.opt-level=0' clippy --locked --release -p gomad --lib --no-deps -- -D warnings, cargo +1.99.1 fmt --all -- --check, git diff --check, same-receipt independent review SHIP session 01a16fcd-9123-7ec1-a7b3-94958e714e02
- PRs:
## Owner-authorized batch B02 (attempt-admission)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.15. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.
