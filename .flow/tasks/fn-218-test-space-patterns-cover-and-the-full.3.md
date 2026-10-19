---
satisfies: [R3, R4]
---
# fn-218-test-space-patterns-cover-and-the-full.3 Kettle and mini activity cover tests; README grammar table

## Description
First tests: Kettle's `tests.rs` gains `cover(&space, Cases(81 * 2), path![User::PressButton, [Heater::Heat; 0..=80], (User::DepressButton | Heater::Failover), .., |k| !k.heating_on]` (the archived enabledness table becomes one `cover` test as well); the mini activity's `tests.rs` (fn-216, `model/temporal`) gains `cover(&space, Cases(2 * 4), ..)`, the eight closings (`[Worker::Poll; 0..=1]` × four endings). Expansion and coverage goldens checked in; both within the inner-loop budget. README section "Describing a test space" with the grammar table and the three confidence mechanisms (no silent vacuity, expansion golden, coverage table). After .2. **Touches:** `model/umpire/tests/fixtures/kettle/tests.rs`, `model/temporal/features/activity_mini/tests.rs`, their `expected/`, `model/README.md`.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
