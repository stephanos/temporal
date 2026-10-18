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
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
