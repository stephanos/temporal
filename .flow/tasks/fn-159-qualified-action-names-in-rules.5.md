---
satisfies: [R11]
---
# fn-159-qualified-action-names-in-rules.5 Move every Limits bound into Bounds

## Description
Move every `Limits` bound into the shared `Bounds` object, so each bound is declared once and can be used from any folder. Today `Bounds` holds only `three` and `four`. The other bounds sit in their feature files: the activity's `five`/`six`/`eight`, nexus workflow's `two`/`control`, the close policy's `four`/`five`/`twelve`, and the task queue's `seven`/`twelve`.

**Size:** M
**Files:** `model/temporal/Bounds.scala`, `model/temporal/features/activity/standalone/Standalone.scala`, `model/temporal/features/nexus/workflow/Workflow.scala`, `model/temporal/features/nexus/workflow/system/ClosePolicy.scala`, `model/temporal/foundations/taskqueue/TaskQueue.scala`, every Query that names a moved bound, the README's Bounds paragraph, regenerated `model/ir/*.json` and Cases if bound names change
**Touches:** [model/temporal/Bounds.scala, model/temporal/features/**/*.scala, model/temporal/foundations/**/*.scala, model/README.md, model/ir/*.json, model/cases/*.json]

### Approach
- **Clashing names have to be resolved first.** A Query's receipt records its bound *by name* (the IR carries `"limits": {"name": "three", ...}`), and three names currently mean different budgets in different folders:
  - `four`: `Bounds` (4, 4, 32768) vs close policy (4, 4, 1<<20)
  - `five`: activity (5, 5, 65536) vs close policy (5, 5, 1<<20)
  - `twelve`: close policy (12, 12, 1<<22) vs task queue (12, 12, 262144)
  
  One duplicate by value: the activity's `eight` and nexus workflow's `control` are both (8, 8, 262144). Merge them under one name.
- **Naming:** keep a name where it's unique. For each clash, keep the name for the most-used budget and give the other a name that says how it differs. For example, the close policy's searches are wider, so `fourWide`, `fiveWide`, `twelveWide`. Confirm the scheme with the owner before regenerating, since receipts and IR change with the names.
- **Move:** declare every bound in `object Bounds`, with a one-line comment for any bound whose name doesn't say its budget. Remove the per-folder declarations and import from `Bounds` at the use sites (`import Bounds.{five, six}`). Rewrite the comment at the top of `Bounds.scala`, which currently says bounds used by a single folder stay local, and the matching README text.
- **Regenerate:** `make umpire-gen-model`. The diff should only rename the bounds that were renamed (the receipts' `limits.name`); budgets and totals don't change. If `control` is merged into `eight` (or the other way), its Queries' receipts change name too.
- Leave `capabilities.bound(...)` overrides and Query bodies as they are, apart from the renamed identifiers.

### Prerequisite
- The working tree must build the lifter and pass the Cases check (the in-progress IR `Duration` migration, plus `make protoc`), because this task regenerates the IR.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```

## Acceptance
- [ ] Every `Limits` value in the temporal Models is declared in `Bounds`; no feature or foundation file declares one.
- [ ] No two bounds share a name with different budgets, and no two names share one budget; the naming scheme was confirmed with the owner.
- [ ] The `Bounds.scala` header and the README describe the single location.
- [ ] Regenerated IR and Cases differ only in renamed bound names; `make umpire-check-model` and `make umpire-check-lint` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
