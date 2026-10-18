---
satisfies: [R1, R2, R3]
---
# fn-196-the-scala-checker-is-the-oracle.2 DSL completion: replay, find with a horizon, seeded sample, refuted, vacuity and the gate budget

## Description
**DSL completion phase (owner, 2026-10-17).** Add the questions fn-195's sections call. `replay(instance, path)` takes a pinned path with expectation blocks at positions and answers `verified` or `refuted` with the failing expectation line and the state; a disabled step is a `refuted` Result naming the step and the state; an effect that throws is a `refuted` Result carrying the message. `find(space, horizon)(target)` stops at the first path reaching the target and answers `found`, `notFound` or `unknown` with the witness. `sample(space, count, seed)` replays `count` seeded random walks of the space and judges them by the instance's claims; one seed gives the same paths twice. `refuted(machine, claim)` replaces `NativeControls.assertRefuted`. `verifyAll` reports vacuity (actions never fired, `after(action)` postconditions never triggered) in fn-190 R15's shapes. The budget is a default plus per-suite overrides in the gate's configuration, never in a Model or test; an override naming no suite is a configuration error; every Result carries its budget.

**Size:** L

R1 is shared with fn-196.1: the move establishes the checker module and common Result,
and this task completes its public questions and error contracts. R1 is not complete at the
module-move checkpoint alone.

**Files:** `model/framework/checker/Checker.scala`, `Replay.scala`, `Spaces.scala`, `Sampling.scala`, `Vacuity.scala` (new), the budget configuration reader, tests beside each.

### Verification
- Checker tests: a replay failing at its second expectation names that line and state; a disabled step refutes without throwing; `find` within a horizon finds Bank's overflow and answers `notFound` one step short; `sample` with one seed is deterministic; `refuted` on the Leaky bank reproduces today's negative control; a vacuous `after(action)` is reported.
- Quick: the checker native suite.

## Acceptance
- [ ] replay, find, sample and refuted exist with the Result shapes and refusals of the spec
- [ ] Vacuity reported in R15's shapes
- [ ] Budget from the gate's configuration with per-suite overrides; configuration error for an unknown suite

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
