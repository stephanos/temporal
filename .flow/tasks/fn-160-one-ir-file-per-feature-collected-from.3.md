---
satisfies: [R7]
---
# fn-160-one-ir-file-per-feature-collected-from.3 One machine per file; flat level folders grouped by name prefix

## Description
Require each Model file to declare at most one machine, composition or derivation (`Machine`, `Composition` or `Derived` object), in a file named after it. Level folders stay flat, with no nested packages. Files are grouped by a shared name prefix: the facet name from the feature's vocabulary (fn-161), or the design family's name. That way the sorted file list shows every machine and how they group. Today 53 machine objects sit in 24 files, and only 12 of those files hold exactly one. The largest, nexus workflow's close policy, has 9 designs in about 1,200 lines.

**Size:** L
**Files:** `model/irgen/Structure.scala` / `Order.scala` (the layout and order checks), `model/irgen/testdata/layout*` (new refusal fixture), `model/README.md` (layout rules, feature-file reading order), every multi-machine file under `model/temporal` (listed below) and the feature-file headers that list files, regenerated `model/ir/*`
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/testdata/layout/**, model/irgen/testdata/layoutRefusals/**, model/README.md, model/temporal/features/**/*.scala, model/temporal/foundations/**/*.scala, model/ir/*]

### Approach
- **The rule:** a file holds at most one top-level `Machine`, `Composition` or `Derived` object, and the file is named after it (`ResetSettlement.scala`). Types, signature declarations and realizations follow the existing layout rules. Decide with the owner whether realization objects fall under the rule too.
- **Flat folders, prefix grouping (owner's decision):** no subfolders or nested packages inside `product/`, `system/` or `realization/`. A family's files share a prefix and sort together, e.g. `system/Heartbeat.scala` (the facet), `HeartbeatCompletion.scala`, `HeartbeatRetry.scala`, `HeartbeatExhaustion.scala`; `ResetSettlement.scala`, `ResetKeepingPause.scala`, `ResetDeferred.scala`; `ClosePolicyRejectAfterClose.scala` and so on. Where an object's name doesn't start with its family prefix, rename the object so it does (e.g. `DeferredReset` → `ResetDeferred`; the close-policy designs → `ClosePolicy<Design>`, or another prefix the owner confirms). This renames IR ids, so list every rename in the done summary.
- **Prefix rule in the lint:** every file in a level folder is the level file, a shared kit file, or starts with a facet name from the feature's vocabulary or a declared design-family prefix. Refuse others, with a fixture.
- **Files to split** (as surveyed on 2026-10-10):
  - nexus workflow `system/ClosePolicy.scala` (9)
  - activity `system/DispatchWithTaskQueue.scala` (6)
  - task queue `system/System.scala` (4)
  - activity `system/Heartbeat.scala`, `system/Reset.scala`, `system/ResponseByID.scala` and nexus workflow `system/System.scala` (3 each)
  - activity `system/DispatchWithWorker.scala`, `system/Timeouts.scala`, `system/Dispatch.scala`, `system/DispatchRaces.scala` and task queue `product/Product.scala` (2 each)
  
  Re-survey before starting; some names may have changed.
- **Shared helpers:** where machines in one file share private helpers, values or comments, move them to the family's folder file, or to the level file. Keep comments with the machine they describe.
- **Enforce in the lifter's layout check** with a refusal naming the file and its second machine, plus a `layoutRefusals` fixture. Update the README layout rules and the reading order.
- **Drop the hand-maintained file lists** at the top of each feature file (e.g. the standalone activity's list of ~17 files and their contents) once the tree shows them. Keep a pointer to the template layout.
- **Regenerate:** object renames for prefixes change IR ids; and the IR records source file paths (`ResponseByID.scala` appears 52 times in `activity-standalone.json`), so moving machines changes the IR. Check that the diff contains only the listed renames, source paths and positions.

### Prerequisite
- Batch 7 (see MILESTONES.md): the lint rules land with Batch 7's tooling; the file moves come after fn-162.3 removes the claim-container subjects (so they aren't split only to be deleted), and before fn-160.2's collection and the batch's single regeneration. No separate regeneration here. A working tree that builds the lifter (the in-progress IR `Duration` migration, plus `make protoc`).

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```
## Acceptance
- [ ] No Model file declares more than one `Machine`, `Composition` or `Derived` object, and each such file is named after its object; level folders are flat, and families share a name prefix (facet or design family) enforced by the lint.
- [ ] The lifter refuses a second machine in a file, naming the file and the machine, with a refusal fixture; the README layout rules state the rule.
- [ ] Feature files no longer carry hand-maintained file lists.
- [ ] Regenerated IR differs only in the listed prefix renames, source paths and positions; `make umpire-check-model` and `make umpire-check-lint` pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
