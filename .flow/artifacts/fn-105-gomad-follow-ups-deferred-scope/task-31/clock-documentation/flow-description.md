Origin: D16 investigation (`docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md`) and the 2026-10-01 quality assessment (Q5). Under `clock_tick: forward`, `time.Now` leads the timer clock by a cumulative, unbounded offset, so every deadline derived from `time.Now` fires late by that lead.

Implement the shared clock: add the forward draws to the virtual clock so `time.Now` and timers read one clock. Start by reproducing the simulation time transport failure that moved fn-103 to a separate offset, and resolve it. If the shared clock is infeasible, stop, retain the evidence, and return the report's fallbacks for a decision.

### Integrated first-party bridge pin repair

The exact D26 time bridge digest and ordered Current directive are rebound without weakening identity predicates. Original source-pin RED, corrected GREEN, twelve rejection controls, independent source audit and conductor integrated checks are retained under .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-31/bridge-pin-repair/. This is verified progress committed under MILESTONES item 5; the original R26 native shared-clock, seeded workload/replay and full gates remain open.

### Integrated clock documentation

CLI, Tutorial and Architecture describe the integrated shared process clock rather than the superseded timestamp-only offset. Their clock paragraphs and adjacent idle-jump qualifiers retain synctest precedence, next-check timer delivery, separate seeded draws and recorded replay identity. The exact patch, source-bound hashes, independent audit and conductor checks are retained under .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-31/clock-documentation/. This documentation progress does not establish native repeatability, traced exact replay or R26 completion; all original acceptance gates remain open.
