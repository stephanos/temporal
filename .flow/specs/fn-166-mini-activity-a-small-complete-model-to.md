# Mini activity: the smallest complete feature, the DSL's fast loop and its example

## Conversation Evidence

> user (2026-10-11): "our turnaround time is way too slow; since our model size is too large and validation time is too slow."
> user: "how about the put the current model aside for a moment and work with a simplified one for now."
> user: "it should still be realistic in that it touches on all these DSL aspects including product vs protocol vs realization and has some depth, but is small enough to iterate _much_ faster on"
> user: "it might be a good idea to keep the mini model around forever; and allow that to drive fast feedback loops; only later to move the changes into the "real" models."
> user: "we want to reduce the special casing and extra instructions as much as possible"
> user: "by defeault we don't want to exlciitly need to list all features"
> user: "how about we store the case outputs inside each feature? then codeowners and feature selection becomes simpler"
> user: "\"cases\" is a weird word for the output folder; maybe find a better one?"
> user (selected): "yes  amend fn-166.1 to colocate instead of select"
> user (selected): "yes do it"
> agent measurement (pasted by the user): compile 20–40 s; focused suites 20–60 s each; steps-only equivalence 40–50 s; full gate about 5 min; complete source seal (production lift, 5.3M-row native comparison, ledgers, mutation controls, broad suites) 20–30 min; the canonical Make equivalence route is blocked by stale linked protobuf generation.

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 30% [paraphrase], 20% [inferred] -->

Every DSL change is validated today against the full Temporal Models: `activity-standalone.json` alone is 4.7 MB of IR, the complete source seal compares 5.3 million rows in 20 to 30 minutes, and the time around it, deciding what proof a change needs and building proof machinery when the existing one does not cover it, is where fn-149.6 went [user, paraphrase]. The ordinary edit loop is about two minutes of tool time; what explodes is large Models times full materialization times a proof that compares every copy against every mutant, run on every change [inferred].

The owner wants a permanent small Model that drives fast feedback, with changes moved into the real Models later, and as little special casing and instruction as possible [user]. So this spec does not introduce a workaround; it makes **the feature the unit of every model operation** and adds **the smallest complete feature**. Compile, lift, check, lint and equivalence take a feature set, default all. The mini activity is a feature like the others, always built with the full set, and the one the DSL is developed against. A real feature that is not yet migrated to a DSL change is *behind*: a per-feature gate state, not a rule in a document. A DSL change ships its migration as code where it can, so moving it into the real Models is running rules, not re-deriving intent. The mini feature is also the running example of `model/SEMANTICS.md`, so the guide's code compiles and cannot drift from the DSL [inferred].

## Architecture & Data Models
<!-- scope: technical -->

**The feature is the unit.** [inferred] Every model target takes `FEATURES`, a list of feature folders under `model/temporal/features/`. Its default is every feature the build *discovers* in that tree (a folder holding a feature file, by the existing layout rule), so no Makefile, gate or test keeps a list of features and adding a feature is adding a folder [user]. `FEATURES` only narrows. Compile, jar, gate (lift, Go checks, Cases, lint), the Scala test suites and `--check-equivalence` operate on the listed features plus what every feature shares (framework, foundations, actors, capabilities, `Bounds`). `IrFile.discover("temporal")` already finds only what was compiled, and fn-160 collects one IR file per feature from its package, so a per-feature run is the normal path, not a mode. `make model-mini` is `make model FEATURES=activity/mini`, an alias and nothing more.

**The mini feature.** [paraphrase] `model/temporal/features/activity/mini/`, beside `standalone/`, by subtraction from it: `client.start` (inputs `scheduleToStart`, `startToClose`, `maxAttempts` bounded at two), `client.cancel`, one worker (`poll`, `respondCompleted`, `respondFailed`), the two timers, dispatch over the task queue foundation. No pause, reset, heartbeat, describe or by-ID client. Three levels, `product/`, `protocol/`, `realization/`, about 100, 400 and 250 lines against today's 446, 2,809 and 1,338. It is built, lifted and checked whenever the full set is; its Cases run live like any feature's. It is never removed.

**The hard instances it must hold, one of each.** [inferred] A DSL change proven on the mini feature transfers to the real ones only if the mini feature contains what the real ones contain.

| Concept | Instance |
| --- | --- |
| Refinement | protocol refines product, with one `query verify` of a product Property through it |
| Shared input | `scheduleToStart` on both `start` and the queue's `schedule` |
| Shared status fact | a protocol phase (`backingOff`) that reports the same status as another (`scheduled`) |
| Facets | at least two (`Dispatching`, `Completion`), with one rule whose guard reads the other facet's phase |
| Capabilities | `Closable`, `Pollable`, `Retryable`, `Expirable`; one `except` waiver and one `overriding` with a reason |
| Composition and channel | the protocol over the task queue with one `sync`; the worker's answers on one bounded lossy channel |
| Derivations | one `FailureModel` with a `NegativeControl` (a dispatch that trusts a stale answer) |
| Progress | one `leadsTo` under `assume.fair` |
| Monitor, hole, assumption | one of each, the hole marking a path the mini feature leaves open |
| Realization | evidence for every fact, one `.live` witness with an `inconclusive` expectation and its reason, one `deadlines` use |
| Queries | one `find`, one `verify`, one witness, with bounds in `Bounds` |

**Behind.** [inferred] A feature whose sources do not compile, or whose checked-in IR does not match its lift, under the current framework is behind. The gate reports it per feature by name and the state is expected while a DSL change is in flight; the full-set gate passing is what closes a DSL spec, as `MILESTONES.md` already requires with its batch-end sweep. Nothing is parked, deleted or regenerated as a special step: a behind feature's IR and Cases stay in the tree as they are until its migration regenerates them.

**Migration as code.** [inferred] A DSL change that renames or reshapes a form ships a scalafix rewrite rule in the model build (fn-156 introduced scalafix there), proven by applying it to the mini feature and lifting a byte-identical or expected IR. Moving the change into a real feature is running the rule and lifting. Where a rewrite rule cannot express the change, the task says so and gives the recipe; that is the exception.

**The example.** [inferred] `model/SEMANTICS.md` uses the mini feature as its running example in place of the hypothetical `job`, citing files and lines, so every snippet in the guide is compiled by the gate.

**Generated outputs live in the feature.** [user] Each feature folder holds one managed output tree, `generated/`: `ir.json`, `ir.lint.json`, `manifest.json` and one `<query>.case.json` per lowered Case (the IR-file prefix in today's Case names is carried by the folder). The central `model/ir/` and `model/cases/` trees go away. Every tool discovers managed trees by glob over `features/**/generated/`, so a per-feature run reads and rewrites one folder and cannot touch another: the scope *is* the folder, and no selector is plumbed into any tool. `.gitattributes` marks `**/generated/** linguist-generated=true` so reviews collapse the JSON, and CODEOWNERS owns a feature's sources and outputs with one line per feature folder. fn-160's "one IR file per feature" lands in this layout; what remains of fn-160 is collection from the package, the removal of `exports`/`irFile`, one machine per file and the Protocol rename. The table pins and the equivalence baseline keep naming today's seven files as the "before".

**The loop.** [inferred] `make model FEATURES=activity/mini` (alias `make model-mini`): compile, lift `activity-mini.json`, the Go checks on it, the syntax lint, then `git diff --stat model/temporal/features/activity/mini/generated/`. Target under 90 seconds with warm caches. The reviewed IR diff is the equivalence evidence at this size; the same `--check-equivalence` runs for a real feature at its size. The lifter fixture suite and the broad Scala suites run at the end of a task.

## API Contracts
<!-- scope: technical -->

- `FEATURES` on every model target: compile, jar, gate, lint, Scala test suites, `--check-equivalence`; its default is discovered from `model/temporal/features/`, and no list of features exists anywhere in the build; a listed folder that does not exist is an error naming it. [user, inferred]
- `make model-mini` is exactly `make model FEATURES=activity/mini`. [user, "make model-mini"]
- Managed output trees are `model/temporal/features/<feature>/generated/` (`ir.json`, `ir.lint.json`, `manifest.json`, `<query>.case.json`), discovered by glob; `model/ir/` and `model/cases/` no longer exist. The gate, `umpire-gen-cases` (every `--kind`), `umpire-repeat`, the lint and every reader resolve a feature's outputs through one path helper per language. This relocation and its path helpers and tests are the one Go change of this spec, an explicit exception to the batch's "only `model/` is edited" rule (owner, 2026-10-11). [user, inferred]
- The gate's per-feature report has three states: current, behind (sources do not compile or IR does not match), missing (no IR checked in). [inferred]
- A DSL change's migration is a scalafix rule under the model build, named after the spec task, applied with the existing `make lint-model-*` fix route. [inferred]
- The mini feature's outputs are `features/activity/mini/generated/`, like every feature's. [inferred]

## Edge Cases & Constraints
<!-- scope: technical -->

- Without colocation, a per-feature gate run would rewrite the whole central `model/cases` tree from the IR files in scope and delete every other feature's Cases (`umpire-gen-cases` has only `--update` and `--kind`); the per-feature tree is therefore a prerequisite of R1, chosen over a selector flag because the scope then needs no plumbing. Path references to migrate: `model/ir` in 3 Go files, 37 Go test files, 58 Scala files and 2 Makefile lines; `model/cases` in 2 Go files, 3 Go test files, 3 Scala files and 1 Makefile line; fn-160.1's done inventory lists the consumers. [inferred]
- Shared code (foundations, actors, capabilities, `Bounds`) is compiled in every feature set; a DSL change to it is proven on the mini feature and migrates the real features like any other. [inferred]
- The two IR-inventory tests (`IrFilesTest`, `MachineTablePins`) compare the compiled exports with the files on disk; under a feature set they must read the compiled set, and a real feature that is behind is reported, not failed, by them. [inferred]
- A real feature behind for a whole batch leaves the full-set build red for that period. That is the price of deferring migration; the per-feature report makes it a listed state rather than a surprise. [inferred]
- The table pins and the equivalence baseline are the "before" of the real features' seal and are untouched by this spec. [inferred]
- The canonical Make route is blocked before execution by stale linked protobuf generation, which is owner-held; until it is restored, per-feature runs go through the direct Scala route and record that. [paraphrase]
- The mini feature's Cases are not pinned in the functional tests or the canary until the owner decides to. [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Every model target takes `FEATURES`, whose default is every feature discovered under `model/temporal/features/` with no list kept anywhere; with one feature listed, only it and the shared code are compiled, lifted, checked and tested, and the others are untouched on disk; `make model-mini` is the alias. [user, inferred] Errors: a listed folder that does not exist fails naming it; a shared-code reference into an unlisted feature fails the compile naming the reference. A gate or Case-generation run scoped to one feature that changes another feature's `generated/` tree is a failure of this criterion (Go test).
- **R2:** The gate reports each feature as current, behind or missing, and a full-set run with a behind feature fails naming it. [inferred] Errors: none beyond the report.
- **R3:** `model/temporal/features/activity/mini/` holds product, protocol and realization levels within about 800 lines, lifts to `features/activity/mini/generated/ir.json`, passes the Go checks and the lint, and every hard instance in the table is present and named in the feature file's header comment. [user, inferred] Errors: a missing instance is a review finding on the task, not a gate failure.
- **R4:** The mini feature's realization lowers to Cases and at least one Case runs live to a verdict once, recorded on the task. [inferred] Errors: a Case that cannot run names the kit gap in the task's done summary.
- **R5:** `make model-mini` runs compile, lift, Go checks, lint and the IR diff in under 90 seconds on the measured machine with warm caches, stopping at the first failure. [user, inferred] Errors: a run over 90 seconds records its stage times on the task.
- **R6:** The migration-as-code convention exists: a scalafix rule location and naming under the model build, one worked example (a rename proven on the mini feature and applied to a real feature), and one sentence in the README and in each open DSL spec of the batch pointing at it. [inferred] Errors: none (documentation and one example).
- **R7:** `model/SEMANTICS.md` uses the mini feature as its running example, citing its files, and the README's "Writing a Model" names it as the smallest complete feature. [inferred] Errors: a guide snippet that does not match the mini feature's source is a review finding.
- **R8:** The two IR-inventory tests pass under the default and under a single-feature set. [inferred] Errors: a test that reads an unlisted feature's file is fixed to read the compiled set.

**Error cases (negative-cases discipline):** each criterion above states its error cases inline or records that it has none.

## Boundaries
<!-- scope: business -->

- No DSL change is made by this spec; it builds the ground the batch iterates on. [inferred]
- Real features' sources, IR and Cases are not edited, moved or deleted. [user]
- The only Go change is the relocation of managed output trees into `features/<feature>/generated/` with its path helpers and tests, granted as an exception to the batch's "only `model/` is edited" rule because R1 is impossible without it; no Go consumer changes beyond resolving paths through the helper. [inferred, owner-granted 2026-10-11]
- Restoring the canonical Make route (`make protoc`, owner-held linked-API generation) is the first thing to do and is not a task of this spec. [paraphrase]
- Pinning the mini feature's Cases in the functional tests or the canary is out of scope. [inferred]

## Decision Context
<!-- scope: both -->

- First shape (2026-10-11, superseded the same day): a `MODEL_FEATURES` switch that parked the real Models, deleted their IR and Cases, and three rules in `MILESTONES.md` with a migration note per task. Rejected by the owner as special casing once the mini Model became permanent. [user]
- Chosen shape: the feature as the unit of every model operation, the mini activity as a permanent feature, "behind" as a per-feature gate state, migration as scalafix rules, and the mini feature as the guide's example. [user, "yes do it"]
- A non-Temporal example (the guide's `job`) was weighed and rejected: the realization layer, the lowering and Testpilot are Temporal's by design, so a toy realization would prove nothing about the real one. [inferred]
- Shrinking the Models does not move the compile floor (20 to 40 seconds is the framework and the lifter); fn-141 is the lever for that. [paraphrase]
- Output location (2026-10-11): a selector flag on `umpire-gen-cases` was granted first, then replaced by colocating each feature's outputs in its own `generated/` tree, because the folder is the scope and needs no plumbing, and one CODEOWNERS line then covers a feature. `generated/` was chosen over `cases/` (names the content, not the purpose), `lowered/` (wrong for the IR) and `testpilot/` (would leave the IR homeless). [user]
- Placement: first in the Scala batch, so fn-159 onward iterate on the mini feature. [user]

## Early proof point

Tasks 1 and 2 together: if one feature cannot be compiled, lifted and checked alone with the inventory tests passing, or `make model-mini` cannot get under 90 seconds with the product level alone, the approach is reconsidered before the protocol and realization levels are written.

## Execution

Order: the feature-set parameter and the per-feature report (1), product level (2), protocol level (3), realization (4), migration-as-code and the guide (5). The loop is usable from task 2 on. Each task ends with the lifter fixture suite and the Scala suites for the mini feature and once for the full set, so the real features are known current at this spec's close.

## Quick commands

```sh
make model FEATURES=activity/mini      # the loop; alias: make model-mini
make model                             # full set, at task end
mise exec -- scala-cli test model/irgen  # at task end
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `FEATURES` on every model target; alias | 1 | |
| R2 | Per-feature gate report: current, behind, missing | 1 | |
| R3 | Mini feature with the hard instances | 2, 3, 4 | |
| R4 | Real realization, one live run | 4 | |
| R5 | Loop under 90 s | 5 | |
| R6 | Migration as code | 5 | |
| R7 | Guide and README use the mini feature | 5 | |
| R8 | Inventory tests under both sets | 1 | |
