---
satisfies: [R3, R4, R5]
---
# fn-174-the-model-constructs-its-own-ir-dsl.7 Capability expansions and gaps construct their messages

## Description
Depends on .6. Capability function fields are capture points. The expansion logic in `export/Capabilities.scala` and the gap resolution in `export/Gaps.scala` are rewritten to read constructed values instead of trees and to build their messages; this is logic, not data, so it moves rather than disappears. The lifter's remaining declaration readers go.

## Acceptance
- Every capability expansion and gap comes from constructed values; `export/Capabilities.scala` and `export/Gaps.scala` read no trees (R3).
- Names equal today's (R4).
- Receipts, Cases and backend answers unchanged (R5).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
