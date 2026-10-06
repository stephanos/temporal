---
satisfies: [R1]
---
# fn-132-group-the-nexus-and-activity-models-by.2 Standalone activity moves under features/activity/standalone

## Description
**Size:** M
**Touches:** [model/temporal/features/standaloneactivity/**, model/temporal/features/activity/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/README.md, .plans/UMPIRE_MODULES.md, .plans/ACTIVITY_MODEL_COMPARISON.md, MILESTONES.md]

**Required investigation:** current activity feature/product/system trees; `tools/umpire/ir/layout_test.go:454`; current split-reader projection helper; fn-128/fn-129 via flowctl. Update downstream Flow prose only through flowctl. Preserve local product/system Phase/State/Fact and actual `client`/`activity` identities; no semantic renames beyond the package/file map.

Part A for the activity. Move `features/standaloneactivity` to `features/activity/standalone`, as folders and packages, with `product/` and `system/` (`System.scala`, `Record.scala`, `WithTaskQueue.scala`, `Realization.scala`). Keep the three System realizations in `standalone/system/Realization.scala`; do not create a root or Product realization file. `StandaloneActivity.scala` becomes `Standalone.scala`; the existing Scala tests move with it. Add `features/activity/Activity.scala` as the kind's general feature file (a header comment and package declaration; task 6 fills it).

Rename the IR files to match (`activity` → `activity-standalone`, and the system-level files likewise); fix the exact names here and record them. Update every reference: Go `tools/umpire`, Makefile targets, Case trees, canary, docs, and the paths in the fn-128 and fn-129 specs and `.plans/ACTIVITY_MODEL_COMPARISON.md`.

Independent of task 1; the two may run in either order but not at the same time (both regenerate `model/ir`).

## Acceptance
- [ ] `features/activity/standalone` exists with the spec's layout; `features/standaloneactivity` does not.
- [ ] A before/after projection with the path and IR-file-name map applied is identical.
- [ ] A path/package/export-aware search finds no retired standaloneactivity Model references outside history, archived plans and closed specs; fn-128/fn-129 use new paths and current local level types. All three realizations remain in the System file, with no root/Product replacement.
- [ ] Focused move proofs pass, then the shared required full model/Go/runtime/artifact/lint batch passes once after both Part A moves. The done evidence explicitly discharges the prerequisite/first move's deferred gates, retains JSON/exit/separate wall-time logs and states any not-required backend/live-run scope honestly. No golden recapture or weakened lint is accepted.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
