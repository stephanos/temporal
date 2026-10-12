---
satisfies: [R8, R9]
---
# fn-160-one-ir-file-per-feature-collected-from.4 Realization layer: flat realization/ folder, <Subject>Run naming, dependency direction: mini feature and shared code (real features at the sweep)

## Description
**Scope (2026-10-12, Scala DSL batch rules; supersedes the text below where they conflict):** re-scoped to `features/activity_mini` and the shared code. The lifter enforces the realization layer (realizations only in `realization/`, each `<Subject>Run` or, for a level's primary machine, `<Form><Kind>Run`, refusing a mismatch naming both; the folder holds those and one `Kit.scala`), and the import lint the one-way layer dependencies and the Temporal API only in `realization/`, each with a refusal fixture. The mini already conforms (`realization/MiniActivityRun.scala`, `Kit.scala`), so no mini rename and no scalafix rule is needed; the mini step tables stay equivalent. The real features' realization renames (`HeartbeatThenCompletion` -> `HeartbeatCompletionRun` and the rest), the nexus moves, regenerated IR/Cases and fixture trees, Go consumers and the `make umpire-check-*` gates go to the owner's sweep and Batch 5b.

Make realizations a layer of their own. Each feature with realizations gets a flat `realization/` folder beside `product/` and `system/`. Each subject's realization is a file named `<Subject>Run.scala`, and the shared helpers go in one kit file. Layering is enforced by package: `product` imports neither `system` nor `realization`, `system` doesn't import `realization`, and only `realization/` may use the Temporal API.

**Size:** M–L
**Files:** `model/temporal/features/activity/standalone/system/Realization.scala` (split up), the nexus realizations (nexus standalone and nexus workflow `Realization.scala`, today at their feature roots), the layout and order lint in `model/irgen` (`Structure.scala`, `Order.scala`), the protobuf-names gate step in `model/check` (`ProtoLiterals` / `Gate.scala`), `model/README.md` (layout, the realization placement paragraph), irgen layout fixtures, regenerated `model/ir/*` and `model/cases/*`
**Touches:** [model/temporal/features/**/*.scala, model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/testdata/layout/**, model/irgen/testdata/layoutRefusals/**, model/check/*.scala, model/README.md, model/ir/*, model/cases/*, tests/testcore/testpilot/testdata/generated/*, tools/canary/casebinding/testdata/*]

### Approach
- **Layout:** `feature/realization/` is a flat folder in its own package (`…standalone.realization`), with no subfolders. Like the level folders, its files are grouped by name prefix (fn-160.3): `Kit.scala` holds the shared helpers (`startActivity`, `described`, `correlated`, `publicAttemptCount`, `heartbeatAttempts`, …), and each realization object gets its own file, `<Subject>Run.scala`.
- **Naming rule:** a realization is named after the subject it realizes plus `Run`. For example `HeartbeatThenCompletion` → `HeartbeatCompletionRun` (realizes `HeartbeatCompletion`), `RetryAfterHeartbeat` → `HeartbeatRetryRun`, `ExhaustAfterHeartbeat` → `HeartbeatExhaustionRun`, `ScheduledCompletionByID` → `ByIDCompletionRun`, `system.Standalone` → `StandaloneActivityRun` (or the name of the subject it realizes). The lifter enforces it: it knows the machine each `Realizes(...)` / `DerivesFrom(...)` names, and it refuses a mismatch with a fixture. Where one subject has several realizations, they take a suffix after `Run` (decide the convention). List every rename: realization names reach the IR and Case names.
- **Dependency direction:** enforce with the layout lint (or package visibility where Scala allows it): no import of `system` or `realization` from `product`, and no import of `realization` from `system`. Refusal fixtures for each.
- **API isolation by folder:** only files under a `realization/` folder may import `io.temporal.api` / protobuf types or name protobuf fields. Replace the current "only `Realization.scala`" rule in the gate and the README.
- **Product realizations:** only when a product has an executable realization (README rule). It sits in the same `realization/` folder.
- **Nexus:** move its realizations from the feature roots into `realization/` folders the same way.
- **Regenerate** the IR and Cases. Differences are limited to the listed realization renames (and Case names derived from them) and source paths. Update the pinned fixture trees.

### Prerequisite
- fn-160.3 (flat prefix layout). Batch 5b (see MILESTONES.md): the lint rules land with the batch's tooling, the moves after fn-160.3's, and the regeneration is the batch's single one at fn-160.2. A working tree that builds the lifter and passes the Cases check.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria. Realization and Case renames reach Go and the fixture trees in 5b.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
make umpire-check-fixtures
```
## Acceptance
- [ ] Every realization lives in a flat `realization/` folder of its feature, one object per `<Subject>Run.scala` file, with shared helpers in one kit file; the lifter refuses a realization not named after its subject.
- [ ] `product` imports neither `system` nor `realization`, `system` doesn't import `realization`, and only `realization/` uses the Temporal API; each is enforced with a refusal fixture, and the README states the rules.
- [ ] Regenerated IR and Cases differ only in the listed realization renames, derived Case names and source paths; `make umpire-check-model`, `make umpire-check-lint` and `make umpire-check-fixtures` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
