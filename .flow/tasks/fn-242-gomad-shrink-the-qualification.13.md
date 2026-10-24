---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.13 Real process custody, accounting, busy workers and cleanup

Touches: tools/gomad/rust/gomad/src/cycle/process/**

## Description
real child custody, accounting, gate and cleanup

Depends on: cycle-foundation. Original .2-owned IDs: 27, all pending. Own the real implementation of launch.py even though .10 owns the proposed endpoint routing record; .10 calls this single implementation.

Description: implement actual bounded stage/leaf subprocess creation with pre-exec registration/admission gate, environment/cwd/argv and PID identity, exact wait4 child accounting, exit/signal retention, command/process observation and fail-closed cleanup. Reuse existing low-level primitives without widening measurement internals. No long wait while holding fork_order spawn lock.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-process`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

## Acceptance
- [ ] Successful, nonzero, signaled, spawn-refused, interrupted and timed-out commands execute actual bounded processes and retain numeric exits/signals, actual parent/child IDs and monotonic spawn/reap boundaries; wait4 user/system/RSS bytes refer to exactly the launched child once. No elapsed-time CPU approximation or RUSAGE_CHILDREN subtraction.
- [ ] Gate release only follows durable registration; refused or interrupted admission cannot execute the leaf. Protect exact/private observation context and explicit environment typing; environment sanitization preserves original contract and excludes task-host GIT_*/SANDBOX_START_DIR/GOROOT for build children.
- [ ] Reap and verify entire owned process groups and sessions, including leader-exits-with-descendant, changed descendant group, malformed/unobservable procfs and interrupted cleanup. Retain original failure precedence; cleanup ambiguity cannot become success.
- [ ] Busy-boundaries/busy-grouped launch twelve real busy worker processes, retain custody and reap every worker on success/failure/interruption before return. Fixtures use bounded handshakes, not full qualification.
- [ ] Prove registry freshness, duplicate PID/registration refusal, accepted status semantics and every assigned predicate with valid controls. No attributable live child may survive tests or handoff.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
