---
satisfies: [R26]
---
# fn-105-gomad-follow-ups-deferred-scope.31 D26: put forward clock ticks on the virtual clock and remove the D16 skip

## Description
Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Shared-clock correction, strict preservation, conformance, removed-skip seeds 1-24 and forward seeds 11/17 traced replay on Darwin. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Origin: D16 investigation (`docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md`) and the 2026-10-01 quality assessment (Q5). Under `clock_tick: forward`, `time.Now` leads the timer clock by a cumulative, unbounded offset, so every deadline derived from `time.Now` fires late by that lead.

Implement the shared clock: add the forward draws to the virtual clock so `time.Now` and timers read one clock. Start by reproducing the simulation time transport failure that moved fn-103 to a separate offset, and resolve it. If the shared clock is infeasible, stop, retain the evidence, and return the report's fallbacks for a decision.

### Integrated first-party bridge pin repair

The exact D26 time bridge digest and ordered Current directive are rebound without weakening identity predicates. Original source-pin RED, corrected GREEN, twelve rejection controls, independent source audit and conductor integrated checks are retained under .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-31/bridge-pin-repair/. This is verified progress committed under MILESTONES item 5; the original R26 native shared-clock, seeded workload/replay and full gates remain open.

### Integrated clock documentation

CLI, Tutorial and Architecture describe the integrated shared process clock rather than the superseded timestamp-only offset. Their clock paragraphs and adjacent idle-jump qualifiers retain synctest precedence, next-check timer delivery, separate seeded draws and recorded replay identity. The exact patch, source-bound hashes, independent audit and conductor checks are retained under .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-31/clock-documentation/. This documentation progress does not establish native repeatability, traced exact replay or R26 completion; all original acceptance gates remain open.
## Acceptance
Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- The fn-103 simulation time transport failure is reproduced and resolved, or shown infeasible with retained evidence and the task returned for a decision.
- A standard-library fixture demonstrates the late deadline on the current runtime and passes with the correction.
- The `UpdateWhilePaused_AfterWindow_ExtendsDispatch` skip is removed and the subtest passes on darwin/arm64 on seeds 1 to 24.
- Every `clock_tick: forward` workload in the generated manifest qualifies on darwin/arm64 on seeds 11 and 17 with traced exact replay; `strict` workloads keep their results.
- `make -C tools/gomad3 test` passes on darwin/arm64. Static coverage of both supported source sets remains required. Native linux/amd64 test, seeded workload and replay evidence belong to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md); missing transferred Linux evidence does not block this task.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
