---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.12 DAG and resource scheduling with executed mutation proofs

Touches: tools/gomad/rust/gomad/src/cycle/scheduler/**

## Description
DAG and resource scheduling semantics

Depends on: cycle-foundation. Original .2-owned IDs: 72, all pending.

Description: implement graph validation/topology/ranking, ready selection, multidimensional resource/isolation admission, execution state machine and critical-path report. Preserve original idle-host over-cap exception and live-memory semantics faithfully; each resource and isolation rule gets its own mutation/control mapping. This task does not create child processes or charge real attempts.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-scheduler`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

## Acceptance
- [ ] Refuse invalid classes/flags/resource values, malformed rows, unknown/duplicate nodes, absent dependencies and cycles; deterministic ordering and rank preserve the reference scheduler.
- [ ] Enforce core/memory caps, locks, scratch versus cleantree in both directions, solo/exclusive/background conditions, heavy before budgeted, budgeted/heavy isolation and reserved eight-GiB guard headroom. Sample actual available-memory seam for guarded concurrent admission and preserve original empty-host behavior with documented source binding.
- [ ] Test every original can_start branch/boolean return and select rule through catching mutations and valid controls; do not equate a refused branch with proof of every disjunct.
- [ ] Fail-fast admits no new stage after failure while draining started stages. stop_at_deadline=False continues through remaining stages; elapsed-time-only overrun is a timing observation, never an infrastructure failure. Report unstarted nodes, exits and actual critical path without fabricating completion.
- [ ] Deterministic executor fixture proves serial/parallel same argv and bounds, overrun complete stage accounting, and failure priority. Real subprocess execution is cycle-process/aggregate responsibility.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
