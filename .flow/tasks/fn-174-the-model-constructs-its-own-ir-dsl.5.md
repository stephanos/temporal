---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.5 Compositions and syncs construct their messages

## Description
**Recut 2026-10-18 (one-language series):** fn-174 R5's oracle is now Scala checker answers, fn-203 Cases and fn-206/207 backends; the IR stays in-process (fn-201).

Depends on .4. `Composition`, members, syncs and sync moves build their messages; the composition `records` capture follows .1's decision. The lifter's `Compositions.scala` goes.
## Acceptance
- Every composition and sync comes from constructed values; `export/Compositions.scala` is deleted (R3).
- Names equal today's (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: