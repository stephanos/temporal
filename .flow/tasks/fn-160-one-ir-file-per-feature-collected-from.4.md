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
Re-scoped to `features/activity_mini` and the shared code (chore commit 8fc711f8fc). The lifter part landed with fn-160.3 in adec9a39c6 (one Structure change). Batch 1d (d1536baa09) had already added the realization folder rule, a bare `…Run` suffix check and the layer/API checks in the import lint, with no fixture. This task adds the subject rule, the kit rule, refusal fixtures and docs.

baseline: fn-160.3 receipt tree (00c7d85674); this task changes no Model source.
stage: impl-review - skipped (REVIEW_MODE=none; independent review runs once at Scala batch end)

Requirement evidence:
- R8, flat `realization/` folder and subject naming: `Structure.layoutFiles` refuses a realization outside `realization/` (fixture `protocol/BoilRun.scala:6`). It reads the machine each `Realizes(machine, ...)` / `DerivesFrom(base, machine, ...)` names and refuses a name other than `<Subject>Run`, or `<Form><Kind>Run` (after the feature file, case aside) for the realization of a level's primary machine. The refusal names both: `SpareRun is a realization of BoilDry: name it after what it runs, BoilDryRun` (`realization/SpareRun.scala:7`), and `Starter ... HeatingSlowRun` (`:7`). `UrnRun` (main) and `BoilDryRun` are accepted. Convention for several realizations of one subject: they are distinguished by their subject (`<Subject>Run`); a second realization of the same subject has no admitted name yet. The mini has none, and fn-162's `DerivesFrom` fixtures (`againRun`/`completedRun`, MiniRealizationFixtures) are outside `realization/` scope rules because they are lifter fixtures without a protocol/realization folder. The decision for the real features is left to the sweep.
- Kit: a `realization/` file that is neither a realization's nor `Kit.scala` is refused (`realization/Helpers.scala:4`). `Kit.scala` is accepted.
- R9, one-way layers and API isolation (import lint, `Conventions.permitted`): fixture refusals at their lines. Product imports protocol (`product/Product.scala:5`), protocol imports realization (`protocol/Protocol.scala:5`), and the Temporal API in protocol (`protocol/BoilWired.scala:4`). All are reported as `R4.module-import ... under the Scala module map`. The one recorded exception (a Protocol's `Describable` naming the realization's status table) is documented in README; the mini uses none.
- Mini conformance: `realization/` holds `MiniActivityRun.scala` (realizes `ActivityProtocol`, the primary of `protocol/Protocol.scala`, hence `<Form><Kind>Run` = `MiniActivityRun`) and `Kit.scala`. Only those two files under the mini, foundations and actors import `io.temporal`/`temporal.server.api` (api-users.txt). The mini product imports no protocol/realization. The selected mini lift passes every rule (task3 `final.log`, exit 0).
- Rename map: `.flow/tmp/fn160/task4/mini.map`, empty with its reason. Step tables equivalent to the mini baseline on this tree under fn-160.3's map (task3 `equivalence-final.log`, exit 0, 16 tables).
- Docs: model/README.md "The layers" (realization naming by subject, kit, nothing else in the folder), the structure-lint paragraph (R9 via the import lint and its fixture), and the feature-file paragraph (`realization/` files instead of `Realization.scala`). The gate's protobuf-names step (`ProtoLiterals`) has no "only Realization.scala" rule to replace. API isolation by folder is the import lint's, enforced as above.
- No scalafix rule: no mini source changes. The real features' realization renames are listed in the migrate README recipe (step 4).

Verification: LayoutFilesFixtures 2/2 and MiniRealizationFixtures 8/8 on the final lifter (task4 suite logs); the selected mini lift and equivalence from fn-160.3 on the same tree.

Failures retained: none new. The FacetsTest/BoundsTest inherited failures are as recorded in fn-160.3.

Open obligations / deferred: real features' realization renames (`HeartbeatThenCompletion`->`HeartbeatCompletionRun`, `RetryAfterHeartbeat`->`HeartbeatRetryRun`, `ExhaustAfterHeartbeat`->`HeartbeatExhaustionRun`, `ScheduledCompletionByID`->`ByIDCompletionRun` and the rest, which the new rule refuses as behind) and the Nexus moves go to the owner's sweep. So do Case names derived from realization names, regenerated IR/Cases/fixture trees and the Go consumers (Batch 5b).

Deferred gates: make umpire-check-model/-check-lint/-check-fixtures, Go/proto/regeneration, the full irgen Fixtures and ConventionsFixtures suites, broad validation and independent review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8fc711f8fc, adec9a39c6
- Tests: timeout 2400 mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.LayoutFilesFixtures (2/2: realization folder, subject and kit refusals; layer import and API refusals), timeout 2400 mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.MiniRealizationFixtures (8/8), grep io.temporal/temporal.server.api over activity_mini, foundations, actors: realization/ only (api-users.txt), fn-160.3 selected mini lift (exit 0) and FEATURES=activity_mini --check-equivalence (equivalent, 16 tables) on the same tree; mini.map empty
- PRs: