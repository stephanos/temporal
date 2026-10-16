---
satisfies: [R1, R2, R3, R4]
---
# fn-154-bound-memory-for-exhaustive-quint-json.1 Generate and consume Quint JSON incrementally

## Description
Deferred by the owner on 2026-10-09. Review the existing measured dump and reader evidence, then design incremental JSON generation and consumption at the actual Quint ITF boundary. Keep exhaustive comparison coverage, native internal tables and compact JSON compatibility tests. Assess the real RunQuint file path and compatibility before an API change; do not introduce a protobuf-only replacement or check large dumps into git. This is follow-up work, not part of fn-145, and must not be activated until scheduled.

## Acceptance
- [ ] R1: Complete native values, ordered receipts, replay witnesses and JSON admission/refusal semantics remain independently verified.
- [ ] R2: Generate and consume JSON incrementally with an explicit transient-memory bound; retain every state/action comparison.
- [ ] R3: Complete deferred concurrent p2 gate and resource evidence are recorded before reactivation; inherited Activity failures stay separately attributed.
- [ ] R4: Large dumps remain temporary and untracked; document compatibility, implementation and API tradeoffs.

## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
