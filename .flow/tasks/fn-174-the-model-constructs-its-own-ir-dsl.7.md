---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.7 Capability expansions and gaps construct their messages

## Description
**Recut 2026-10-18 (one-language series):** fn-174 R5's oracle is now Scala checker answers, fn-203 Cases and fn-206/207 backends; the IR stays in-process (fn-201).

Depends on .6. Capability function fields are capture points. The expansion logic in `export/Capabilities.scala` and the gap resolution in `export/Gaps.scala` are rewritten to read constructed values instead of trees and to build their messages; this is logic, not data, so it moves rather than disappears. The lifter's remaining declaration readers go.
## Acceptance
- Every capability expansion and gap comes from constructed values; `export/Capabilities.scala` and `export/Gaps.scala` read no trees (R3).
- Names equal today's (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: