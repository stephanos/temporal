---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.15 Fixed global attempt journal and reservation-time admission floors

Touches: tools/gomad/rust/gomad/src/cycle/admission/**, tools/gomad/rust/gomad/src/cycle/journal/**

## Description
fixed global two-attempt journal and reservation floors

Depends on: cycle-foundation; cycle-process for the real spawn boundary seam. Original .2-owned IDs: 24 pending (host admission16, producer-memory8), plus policy-ambiguous-spawn, policy-fixed-enrollment, policy-reservation-resource-floors.

Description: preserve host/scratch filesystem admission and strict live-memory parsing; implement one fixed, locked durable attempt journal across candidates/outputs/restarts. Bind enrollment identity without replacement/reset. Reserve only after rechecking all three serving filesystems immediately before actual dispatch. No test performs full dispatch or consumes either real allowance.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-admission`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

## Acceptance
- [ ] Host/evidence/scratch canonical containment, distinct explicit temporary children, forbidden serving filesystems and fixed probe failure behavior retain original thresholds and custody. Strict decimal/MemAvailable bounded parsing rejects overflow, missing/duplicate/non-kB/unsupported/error/below-bound observations.
- [ ] Inject a real observable bounded child then interrupt before PID publication. Unknown spawn outcomes retain the reservation/charge across restart; two charged starts/ambiguities refuse third across candidates/output dirs. Only definitive authenticated no-child evidence releases a reservation. Reap all fixture children.
- [ ] Explicit initial enrollment is the only creation path. Missing/mismatched policy/path/inode/digest/journal, symlink replacement or retained started-output/high-water state refuses before reservation/spawn and never restores allowance; concurrent lock contenders cannot create histories.
- [ ] After initial/preparation pass, lower HOST45GiB, overlay20GiB or inner12GiB by one byte before reservation: each fails before reserve/spawn without charging. Exact floors pass. Probe errors fail closed; retain original overlay filesystem/path containment policy. Floor failure cannot be deferred-perf success.
- [ ] Actually execute all24 check proofs and all policy fixtures/control cases using deterministic filesystem/process seams and temporary fixture-only journal roots. Preserve the actual global journal and zero-used/two-remaining state untouched.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
