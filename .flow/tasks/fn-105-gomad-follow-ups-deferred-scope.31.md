---
satisfies: [R26]
---
# fn-105-gomad-follow-ups-deferred-scope.31 D26: put forward clock ticks on the virtual clock and remove the D16 skip

## Description
Origin: D16 investigation (`docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md`) and the 2026-10-01 quality assessment (Q5). Under `clock_tick: forward`, `time.Now` leads the timer clock by a cumulative, unbounded offset, so every deadline derived from `time.Now` fires late by that lead.

Implement the shared clock: add the forward draws to the virtual clock so `time.Now` and timers read one clock. Start by reproducing the simulation time transport failure that moved fn-103 to a separate offset, and resolve it. If the shared clock is infeasible, stop, retain the evidence, and return the report's fallbacks for a decision.

## Acceptance
- The fn-103 simulation time transport failure is reproduced and resolved, or shown infeasible with retained evidence and the task returned for a decision.
- A standard-library fixture demonstrates the late deadline on the current runtime and passes with the correction.
- The `UpdateWhilePaused_AfterWindow_ExtendsDispatch` skip is removed and the subtest passes on seeds 1 to 24.
- Every `clock_tick: forward` workload in the generated manifest qualifies on seeds 11 and 17 with traced exact replay; `strict` workloads keep their results.
- `make -C tools/gomad3 test` passes. linux/amd64 status is recorded; missing evidence leaves the task open.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
