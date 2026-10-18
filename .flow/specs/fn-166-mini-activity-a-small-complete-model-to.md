# Mini activity: the smallest complete feature, the DSL's fast loop and its example

DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

> HTML render lens: `.flow/artifacts/fn-166-mini-activity-a-small-complete-model-to/spec.html` - local-only; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

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
> user: "any change to the umpire/ framework would still need to update all of activity and nexus, right? how about at the end of the spec we move them to a separate package that is not compiled [...]"
> user: "unless, could we make the FEATURES selector explciitly exclude the other fatures from compilation, too, safely and easily? that way we wouldn't hae to more any files" / "no I thoguht only on demand from seting that Makefile env?"
> user (selected): "yes do it"
> agent measurement (pasted by the user): compile 20–40 s; focused suites 20–60 s each; steps-only equivalence 40–50 s; full gate about 5 min; complete source seal (production lift, 5.3M-row native comparison, ledgers, mutation controls, broad suites) 20–30 min; the canonical Make equivalence route is blocked by stale linked protobuf generation.

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 30% [paraphrase], 20% [inferred] -->

Every DSL change is validated today against the full Temporal Models: `activity-standalone.json` alone is 4.7 MB of IR, the complete source seal compares 5.3 million rows in 20 to 30 minutes, and the time around it, deciding what proof a change needs and building proof machinery when the existing one does not cover it, is where fn-149.6 went [user, paraphrase]. The ordinary edit loop is about two minutes of tool time; what explodes is large Models times full materialization times a proof that compares every copy against every mutant, run on every change [inferred].

The owner wants a permanent small Model that drives fast feedback, with changes moved into the real Models later, and as little special casing and instruction as possible [user]. So this spec does not introduce a workaround; it makes **the feature the unit of every model operation** and adds **the smallest complete feature**. Compile, lift, check, lint and equivalence take a feature set, default all. The mini activity is a feature like the others, always built with the full set, and the one the DSL is developed against. A real feature that is not yet migrated to a DSL change is *behind*: a per-feature gate state, not a rule in a document. A DSL change ships its migration as code where it can, so moving it into the real Models is running rules, not re-deriving intent. The mini feature is also the running example of `model/framework/SEMANTICS.md`, so the guide's code compiles and cannot drift from the DSL [inferred].

## Architecture & Data Models
<!-- scope: technical -->

**The feature is the unit.** [inferred] Every model target takes `FEATURES`, a list of feature folders under `model/temporal/features/`. Its default is every feature the build *discovers* in that tree (every top-level folder under `features/` in fn-160 R14's flat layout: kind folders for shared parts, `<kind>_<form>` folders for forms), so no Makefile, gate or test keeps a list of features and adding a feature is adding a folder [user]. `FEATURES` only narrows. Selecting a form selects its kind folder: `FEATURES=activity_mini` compiles `features/activity` (the kind's shared code) with `features/activity_mini`, since the form imports the kind; a kind folder alone lifts nothing unless it declares Queries or realizations (fn-160 R10). Compile, jar, gate (lift, Go checks, Cases, lint), the Scala test suites and `--check-equivalence` operate on the listed features plus what every feature shares (framework, foundations, actors, capabilities, `Bounds`). `IrFile.discover("temporal")` already finds only what was compiled, and fn-160 collects one IR file per feature from its package, so a per-feature run is the normal path, not a mode. `make model-mini` is `make model FEATURES=activity_mini`, an alias and nothing more.

**The mini feature.** [paraphrase] `model/temporal/features/activity_mini/`, beside `features/activity_standalone/` (fn-160 R14's flat layout), by subtraction from it: `client.start` (inputs `scheduleToStart`, `startToClose`, `maxAttempts` bounded at two), `client.cancel`, one worker (`poll`, `respondCompleted`, `respondFailed`), the two timers, dispatch over the task queue foundation. No pause, reset, heartbeat, describe or by-ID client. Three levels, `product/`, `protocol/`, `realization/`, about 100, 400 and 250 lines against today's 446, 2,809 and 1,338. It is built, lifted and checked whenever the full set is; its Cases run live like any feature's. It is never removed.

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

**Behind.** [inferred] A feature under `features/` whose sources do not compile, or whose `generated/ir.json` does not match its lift, under the current framework is behind. The gate reports it per feature by name and the state is expected while a DSL change is in flight. In the current Scala DSL batch a DSL spec closes on the mini feature's gate (`FEATURES=activity_mini`); the full-set gate passes only after the owner's sweep brings the real features forward (owner, 2026-10-11). Nothing is parked, deleted or regenerated as a special step: a behind feature's IR and Cases stay in the tree as they are until its migration regenerates them.

**Migration as code.** [inferred] A DSL change that renames or reshapes a form ships a scalafix rewrite rule in the model build (fn-156 introduced scalafix there), proven by applying it to the mini feature and lifting a byte-identical or expected IR. Moving the change into a real feature is running the rule and lifting. Where a rewrite rule cannot express the change, the task says so and gives the recipe; that is the exception.

**The example.** [inferred] `model/framework/SEMANTICS.md` uses the mini feature as its running example in place of the hypothetical `job`, citing files and lines, so every snippet in the guide is compiled by the gate.

**Generated outputs live in the feature.** [user] Each feature folder holds one managed output tree, `generated/`: `ir.json`, `ir.lint.json`, `manifest.json` and one `<query>.case.json` per lowered Case (the IR-file prefix in today's Case names is carried by the folder). The central `model/ir/` and `model/cases/` trees go away. Every tool discovers managed trees by glob over `features/**/generated/`, so a per-feature run reads and rewrites one folder and cannot touch another: the scope *is* the folder, and no selector is plumbed into any tool. `.gitattributes` marks `**/generated/** linguist-generated=true` so reviews collapse the JSON, and CODEOWNERS owns a feature's sources and outputs with one line per feature folder. fn-160's "one IR file per feature" lands in this layout; what remains of fn-160 is collection from the package, the removal of `exports`/`irFile`, one machine per file and the Protocol rename. The table pins and the equivalence baseline keep naming today's seven files as the "before".

**The loop.** [inferred] `make model FEATURES=activity_mini` (alias `make model-mini`): compile, lift `activity-mini.json`, the Go checks on it, the syntax lint, then `git diff --stat model/temporal/features/activity_mini/generated/`. Target under 90 seconds with warm caches. The reviewed IR diff is the equivalence evidence at this size, together with `--check-equivalence` over `FEATURES=activity_mini` against a mini step-table baseline captured once; the real features' equivalence runs only at the owner's sweep. Under the owner-approved fast cadence below, steps run mini-focused checks and targeted tests for changed framework/lifter files; broad suites and independent review run once at the Scala batch-end gate.

**Lossless transition (owner, 2026-10-11).** The seven historical IR payloads cannot become three feature-level IR files by relocation alone. Preserve the extra payloads, their lint acceptances and associated Cases inside their owning feature's `generated/legacy/` until fn-160 performs collection and the resulting migrations. Do not merge semantics, regenerate historical payloads, invent feature folders, or change table pins and equivalence baselines as part of this relocation. [user, "Preserve legacy payloads inside their feature trees until fn-160"]

**Existing pinned outputs.** Functional/canary pins retain their existing global publication locations and identities. Their check/update targets still honor the feature-folder scope: only selected feature inputs are generated, and partial publication preserves all unselected pins and manifest rows. This narrowly approved publication repair is separate from the feature-owned generated trees, and adds no selector flag. [user, scoped publication approval below]

**Case subtree (owner, 2026-10-11).** Task 1 establishes the initial colocated layout above. The separate task 6 then moves active Case manifests and payloads into `generated/cases/`, leaving `ir.json` and `ir.lint.json` at the generated root. Historical IR/lint payloads remain under `generated/legacy/`; their Case manifests and byte-identical payloads move to `generated/cases/legacy/`, retaining historical filenames and logical identities until fn-160. This is path-only: Case publication owns its subtree and does not rewrite IR, lint acceptances, preserved legacy artifacts or another feature's outputs. [user, "add a flow spec task to move them to generated/cases/"]

## API Contracts
<!-- scope: technical -->

- `FEATURES` on every model target: compile, jar, gate, lint, Scala test suites, `--check-equivalence`; its default is discovered from `model/temporal/features/`, and no list of features exists anywhere in the build; a listed folder that does not exist is an error naming it. [user, inferred]
- `make model-mini` is exactly `make model FEATURES=activity_mini`. [user, "make model-mini"]
- Managed output trees are `model/temporal/features/<feature>/generated/` (`ir.json`, `ir.lint.json`, `manifest.json`, `<query>.case.json`), discovered by glob; `model/ir/` and `model/cases/` no longer exist. The gate, `umpire-gen-cases` (every `--kind`), `umpire-repeat`, the lint and every reader resolve a feature's outputs through one path helper per language. The Go exception is limited to these path changes and the narrowly approved scoped publication of existing global functional/canary pins, with their regression tests (owner, 2026-10-11). [user, inferred]
- The gate's per-feature report has three states: current, behind (sources do not compile or IR does not match), missing (no IR checked in). [inferred]
- A DSL change's migration is a scalafix rule under the model build, named after the spec task, applied with the existing `make lint-model-*` fix route. [inferred]
- The mini feature's outputs are `features/activity_mini/generated/`, like every feature's. [inferred]
- Final Case paths are `features/<feature>/generated/cases/manifest.json` and `generated/cases/<query>.case.json`; transitional historical Cases use `generated/cases/legacy/`. Task 6 owns this follow-up relocation, not the in-progress task 1. [user]

## Edge Cases & Constraints
<!-- scope: technical -->

- Without colocation, a per-feature gate run would rewrite the whole central `model/cases` tree from the IR files in scope and delete every other feature's Cases (`umpire-gen-cases` has only `--update` and `--kind`); the per-feature tree is therefore a prerequisite of R1, chosen over a selector flag because the scope then needs no plumbing. Path references to migrate: `model/ir` in 3 Go files, 37 Go test files, 58 Scala files and 2 Makefile lines; `model/cases` in 2 Go files, 3 Go test files, 3 Scala files and 1 Makefile line; fn-160.1's done inventory lists the consumers. [inferred]
- Shared code (foundations, actors, capabilities, `Bounds`) is compiled in every feature set; a DSL change to it is proven on the mini feature, and a real feature it breaks stays behind until the owner's sweep runs the change's rule on it. [user, owner 2026-10-11]
- The two IR-inventory tests (`IrFilesTest`, `MachineTablePins`) compare the compiled exports with the files on disk; under a feature set they must read the compiled set, and a real feature that is behind is reported, not failed, by them. [inferred]
- While a DSL change is in flight the real features are behind and a default `make model` (every feature) is red; the loop runs with `FEATURES=activity_mini` (`make model-mini`), which compiles only the selected feature and the shared code. Nothing is parked, marked or moved: exclusion from compilation is on demand, from the Makefile variable, and leaves no state behind (owner, 2026-10-11). [user]
- The table pins and the equivalence baseline are the "before" of the real features' seal and are untouched by this spec. The batch's equivalence uses a separate mini baseline (Scope below). [inferred]
- The canonical Make route is blocked before execution by stale linked protobuf generation, which is owner-held; until it is restored, per-feature runs go through the direct Scala route and record that. [paraphrase]
- The mini feature's Cases are not pinned in the functional tests or the canary until the owner decides to. [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Every model target takes `FEATURES`, whose default is every feature discovered under `model/temporal/features/` with no list kept anywhere; with one feature listed, only it, its kind folder and the shared code are compiled, lifted, checked and tested, and the others are untouched on disk; `make model-mini` is the alias. [user, inferred] Errors: a listed folder that does not exist fails naming it; a shared-code reference into an unlisted feature fails the compile naming the reference. A gate or Case-generation run scoped to one feature that changes another feature's `generated/` tree is a failure of this criterion (Go test).
- **R2:** The gate reports each feature as current, behind or missing, and a full-set run with a behind feature fails naming it. [inferred] Errors: none beyond the report.
- **R3:** `model/temporal/features/activity_mini/` holds product, protocol and realization levels within about 800 lines, lifts to `features/activity_mini/generated/ir.json`, passes the Go checks and the lint, and every hard instance in the table is present and named in the feature file's header comment. [user, inferred] Errors: a missing instance is a review finding on the task, not a gate failure.
- **R4:** The mini feature's realization lowers to Cases and at least one Case runs live to a verdict once, recorded on the task. [inferred] Errors: a Case that cannot run names the kit gap in the task's done summary.
- **R5:** `make model-mini` runs compile, lift, Go checks, lint and the IR diff in under 90 seconds on the measured machine with warm caches, stopping at the first failure. [user, inferred] Errors: a run over 90 seconds records its stage times on the task.
- **R6:** The migration-as-code convention exists: a scalafix rule location and naming under the model build, one worked example (a rename proven on the mini feature; it is not applied to a real feature, which the owner's sweep does), and one sentence in the README and in each open DSL spec of the batch pointing at it. [inferred] Errors: none (documentation and one example).
- **R7:** `model/framework/SEMANTICS.md` uses the mini feature as its running example, citing its files, and the README's "Writing a Model" names it as the smallest complete feature. [inferred] Errors: a guide snippet that does not match the mini feature's source is a review finding.
- **R8:** The two IR-inventory tests pass under the default and under a single-feature set. [inferred] Errors: a test that reads an unlisted feature's file is fixed to read the compiled set.
- **R9:** Case manifests and payloads live separately under each feature's `generated/cases/`, with historical Cases preserved under `generated/cases/legacy/`; shared readers and publication paths use that layout without changing IR/lint paths, payload bytes, logical identities, pins or equivalence baselines. A scoped Case publication touches neither sibling features nor non-Case artifacts. [user] Errors: tests refuse missing Case references and duplicate destination names.

- **R10:** The model-side Make targets drop the `umpire-` prefix and follow the repo's verb-noun convention, with the runtime targets (`umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`, `umpire-assess`, the backends, live-test and scratch targets) keeping their names: `umpire-check-model` becomes `check-model`, `umpire-gen-model` `gen-model`, `umpire-check-lint` `lint-ir` (the Scala source lint keeps `lint-model`; the two lint different things), `umpire-gen-cases`/`umpire-check-cases` `gen-cases`/`check-cases`, `umpire-gen-fixtures`/`umpire-check-fixtures` `gen-fixtures`/`check-fixtures`, `umpire-check-equivalence`/`umpire-capture-baseline` `check-equivalence`/`capture-baseline`; `model` and `model-mini` stay the loop's entry points. No alias for an old name remains, and every live reference (Makefile, CI workflows, `AGENTS.md`, the READMEs, `.plans/`, gate messages, Go and Scala comments) uses the new names; Flow's task and handoff history is not rewritten. [user, "using `model` is much clearer"; inferred] Errors: a leftover old name is a grep failure recorded on the task; a reference to a target that no longer exists fails `make` naming it.

**Error cases (negative-cases discipline):** each criterion above states its error cases inline or records that it has none.

## Boundaries
<!-- scope: business -->

- No DSL change is made by this spec; it builds the ground the batch iterates on. [inferred]
- Real features' sources are not edited, moved or deleted. Their IR, lint acceptances and Cases are relocated losslessly into their owning feature's `generated/` tree, with extra historical payloads preserved under `generated/legacy/` until fn-160; their contents, table pins and equivalence baselines remain unchanged. [user, owner clarification 2026-10-11]
- Go changes are limited to relocating managed output trees into `features/<feature>/generated/` with path helpers and tests, plus the narrowly required subset-safe publication of existing functional/canary pins approved below. No other Go consumer, lowering, runtime or schema behavior changes are authorized. [user, owner-granted 2026-10-11]
- Restoring the canonical Make route (`make protoc`, owner-held linked-API generation) is the first thing to do and is not a task of this spec. [paraphrase]
- Pinning the mini feature's Cases in the functional tests or the canary is out of scope. [inferred]

## Decision Context
<!-- scope: both -->

- First shape (2026-10-11, superseded the same day): a `MODEL_FEATURES` switch that parked the real Models, deleted their IR and Cases, and three rules in `MILESTONES.md` with a migration note per task. Rejected by the owner as special casing once the mini Model became permanent. [user]
- Chosen shape: the feature as the unit of every model operation, the mini activity as a permanent feature, "behind" as a per-feature gate state, migration as scalafix rules, and the mini feature as the guide's example. [user, "yes do it"]
- A non-Temporal example (the guide's `job`) was weighed and rejected: the realization layer, the lowering and Testpilot are Temporal's by design, so a toy realization would prove nothing about the real one. [inferred]
- Shrinking the Models does not move the compile floor (20 to 40 seconds is the framework and the lifter); fn-141 is the lever for that. [paraphrase]
- Output location (2026-10-11): a selector flag on `umpire-gen-cases` was granted first, then replaced by colocating each feature's outputs in its own `generated/` tree, because the folder is the scope and needs no plumbing, and one CODEOWNERS line then covers a feature. `generated/` was chosen over `cases/` (names the content, not the purpose), `lowered/` (wrong for the IR) and `testpilot/` (would leave the IR homeless). [user]
- Excluding the real features from compilation (2026-10-11): a switch plus deleted artifacts, then a `git mv` to `parked/`, then a `PARKED` marker were each considered. The owner chose none of them: `FEATURES` on the Makefile already excludes unselected features from the compile set, on demand and with no persistent state, and the default build being red while real features are behind is accepted for the batch. [user]
- Placement: first in the Scala batch, so fn-159 onward iterate on the mini feature. [user]

**Scope: the mini feature only (owner, 2026-10-11).** "The scope for that section IS ONLY the activity_mini feature; other features are to be ignored and not updated when the Umpire DSL changes." Every spec of the Scala DSL batch in `MILESTONES.md` changes the framework, the lifter, the shared code and `features/activity_mini`, nothing else. [user]
- The real features (standalone activity, Nexus workflow, Nexus standalone, and their `generated/legacy/` payloads) are not edited, migrated, lifted or regenerated, and do not constrain a DSL change. A change that breaks them leaves them behind. [user]
- A task written against a real feature applies its change to the mini feature where the mini feature has the construct; the rest waits for the owner's sweep. [user, chosen option]
- Each DSL change still ships its scalafix rule, proven on the mini feature and not run on the real features until the sweep. [user, chosen option]
- A mini step-table baseline is captured once, before the batch's next DSL change; per-spec equivalence and the pre-fault seal compare `FEATURES=activity_mini` against it. The batch-1d baseline stays as the real features' "before". [user, chosen option]

**Lint work (owner, 2026-10-11):** "ignore the linting." Set lint checks, lint-debt remediation and manual acceptance bookkeeping aside for this fn-166 run. Preserve existing lint companions without changing project-wide lint policy. This overrides lint verification obligations below for the current run; report those checks as not run, never as passing. The feature-selection contract for existing lint targets remains, and compile/lift/Go correctness checks, scoped-output preservation, live execution and review are still required. [user]

**Fast iteration (owner, 2026-10-11):** "yes," approving mini-focused checks per step, targeted tests for changed framework/lifter files, and one broad validation plus independent review at the Scala batch end. This supersedes per-task broad-suite and review cadence for the current batch, not correctness requirements or owner-held boundaries. Task evidence records focused results and explicitly records broad validation/review as deferred; it never claims an unrun suite passed or invents SHIP. Preserve earlier review receipts and carry any findings, fixes and remaining verification into the batch review. An already-running broad suite may finish and contribute evidence; do not launch another solely for task closure. The batch cannot be called validated or review-complete until its final gate runs. [user]

**Fixture gate repair (owner, 2026-10-11):** "approved," in response to the recorded inherited lifter-test blockers. Task 1 may refresh only the stale `hints` and `hintsRefused` test snapshots to match the existing, unchanged activity sources and correct diagnostic source-location extraction, with focused regression tests. Retain the eight other fixtures' explicit identity/position invariants. This is not authorization to change production Model behavior, regenerate real-feature IR/Cases or lint companions, alter table pins or equivalence baselines, weaken rejection assertions, or waive the required lifter suite. [user]

**Scoped pinned publication (owner, 2026-10-11):** "yes," approving the narrow Go publication repair requested after checkpoint `17c3a4006b`. Task 1 may make the functional/canary check and update targets honor `FEATURES` through the existing feature-folder scope, without a new CLI selector or public semantic API. Generate only selected feature inputs; retain the existing pin identities and global publication locations, and preserve every unselected payload, manifest row and modification time under the existing publication lock. A selected feature with no pins is a no-op. Tests cover actual Make entrypoints, check/update, no-pin selections and preservation. This is a bounded exception to the path-only Go boundary above, not authorization to regenerate protected real artifacts, repin Cases, restore protobufs or implement deferred Go work. [user]

2026-10-18, phase placement (owner): this spec's remaining work moves into Phase 3 "Temporal Models native", between DSL completion and the IR batch: re-enable `model/temporal` against the final DSL (mini first, then standalone activity, then Nexus) and prove each Model through the native checker and ModelSuite under the budget policy. Its IR-equivalence, lint-acceptance and Case proof obligations are restated as native checks here; lifting, Cases and descriptors wait for Phase 4 (IR batch) and Phase 5 (fn-201/fn-203/fn-204).

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Early proof point

Tasks 1 and 2 together: if one feature cannot be compiled, lifted and checked alone with the inventory tests passing, or `make model-mini` cannot get under 90 seconds with the product level alone, the approach is reconsidered before the protocol and realization levels are written.

## Execution

Current source order is the committed product/protocol (2, 3), Scala realization and witness authoring (8), then rule source, mini proof and guide (9). Case paths do not affect `Realizes`, evidence, deadlines, Query expectations or source lifting. Tasks 8 and 9 therefore have no dependency on the held Case relocation.

The existing completion tasks remain held. Task 4 owns Case lowering and the recorded live proof after source task 8 and Case-layout task 6. Task 10 supplies runnable custom-rule loading and the mini worked application after source task 9. Task 5 owns the canonical timed loop after task 4, task 10 and target naming task 7. Its full lint/currentness admission also checks the committed fn-167.7 sweep closure. Keeping infrastructure task 10 separate from final-loop task 5 avoids requiring gap migration to wait for the lint closure it enables. No held authority is granted by this partition.

The CLI preserves existing dependency edges, so tasks 4 and 5 retain their valid historical producer dependencies while source tasks 8 and 9 supply the current authoring lane. Completed tasks 1, 2 and 3 and their receipts are unchanged. R3, R4 and R6 remain partial until their applicable held consumer obligations are proved. Source completion is never a live verdict or production-loader result.

Each step runs mini-focused checks and targeted changed-file tests. Broad Scala validation and independent review run once at the approved mini batch-end gate. Real-feature migration and full-set currentness remain at the owner's sweep under the current Scope. Planning review is deferred under the same owner cadence; no updated SHIP verdict is claimed.

## Quick commands

```sh
make model FEATURES=activity_mini      # the loop; alias: make model-mini
make model                             # full set, only at the owner-authorized sweep
mise exec -- scala-cli test model/irgen  # broad suite, at Scala batch end
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Every model target takes `FEATURES`, whose default is every feature discovered under `model/temporal/features/` with no list kept anywhere; with one feature listed, only it, its kind folder and the shared code are compiled, lifted, checked and tested, and the others are untouched on disk; `make model-mini` is the alias. [user, inferred] Errors: a listed folder that does not exist fails naming it; a shared-code reference into an unlisted feature fails the compile naming the reference. A gate or Case-generation run scoped to one feature that changes another feature's `generated/` tree is a failure of this criterion (Go test). | fn-166-mini-activity-a-small-complete-model-to.1 | — |
| R2 | The gate reports each feature as current, behind or missing, and a full-set run with a behind feature fails naming it. [inferred] Errors: none beyond the report. | fn-166-mini-activity-a-small-complete-model-to.1 | — |
| R3 | `model/temporal/features/activity_mini/` holds product, protocol and realization levels within about 800 lines, lifts to `features/activity_mini/generated/ir.json`, passes the Go checks and the lint, and every hard instance in the table is present and named in the feature file's header comment. [user, inferred] Errors: a missing instance is a review finding on the task, not a gate failure. | fn-166-mini-activity-a-small-complete-model-to.2, fn-166-mini-activity-a-small-complete-model-to.3, fn-166-mini-activity-a-small-complete-model-to.4, fn-166-mini-activity-a-small-complete-model-to.5, fn-166-mini-activity-a-small-complete-model-to.8 | Source and consumer proofs close separately; held gates remain open |
| R4 | The mini feature's realization lowers to Cases and at least one Case runs live to a verdict once, recorded on the task. [inferred] Errors: a Case that cannot run names the kit gap in the task's done summary. | fn-166-mini-activity-a-small-complete-model-to.4, fn-166-mini-activity-a-small-complete-model-to.8 | Source in 8; actual lowering/live proof held in 4 |
| R5 | `make model-mini` runs compile, lift, Go checks, lint and the IR diff in under 90 seconds on the measured machine with warm caches, stopping at the first failure. [user, inferred] Errors: a run over 90 seconds records its stage times on the task. | fn-166-mini-activity-a-small-complete-model-to.5 | Canonical loop proof remains held |
| R6 | The migration-as-code convention exists: a scalafix rule location and naming under the model build, one worked example (a rename proven on the mini feature; it is not applied to a real feature, which the owner's sweep does), and one sentence in the README and in each open DSL spec of the batch pointing at it. [inferred] Errors: none (documentation and one example). | fn-166-mini-activity-a-small-complete-model-to.10, fn-166-mini-activity-a-small-complete-model-to.9 | Rule source/mini proof in 9; production loader held in 10 |
| R7 | `model/framework/SEMANTICS.md` uses the mini feature as its running example, citing its files, and the README's "Writing a Model" names it as the smallest complete feature. [inferred] Errors: a guide snippet that does not match the mini feature's source is a review finding. | fn-166-mini-activity-a-small-complete-model-to.9 | — |
| R8 | The two IR-inventory tests pass under the default and under a single-feature set. [inferred] Errors: a test that reads an unlisted feature's file is fixed to read the compiled set. | fn-166-mini-activity-a-small-complete-model-to.1 | — |
| R9 | Case manifests and payloads live separately under each feature's `generated/cases/`, with historical Cases preserved under `generated/cases/legacy/`; shared readers and publication paths use that layout without changing IR/lint paths, payload bytes, logical identities, pins or equivalence baselines. A scoped Case publication touches neither sibling features nor non-Case artifacts. [user] Errors: tests refuse missing Case references and duplicate destination names. | fn-166-mini-activity-a-small-complete-model-to.6 | — |
| R10 | The model-side Make targets drop the `umpire-` prefix and follow the repo's verb-noun convention, with the runtime targets (`umpire-run`, `umpire-fuzz`, `umpire-repeat`, `umpire-replay`, `umpire-assess`, the backends, live-test and scratch targets) keeping their names: `umpire-check-model` becomes `check-model`, `umpire-gen-model` `gen-model`, `umpire-check-lint` `lint-ir` (the Scala source lint keeps `lint-model`; the two lint different things), `umpire-gen-cases`/`umpire-check-cases` `gen-cases`/`check-cases`, `umpire-gen-fixtures`/`umpire-check-fixtures` `gen-fixtures`/`check-fixtures`, `umpire-check-equivalence`/`umpire-capture-baseline` `check-equivalence`/`capture-baseline`; `model` and `model-mini` stay the loop's entry points. No alias for an old name remains, and every live reference (Makefile, CI workflows, `AGENTS.md`, the READMEs, `.plans/`, gate messages, Go and Scala comments) uses the new names; Flow's task and handoff history is not rewritten. [user, "using `model` is much clearer"; inferred] Errors: a leftover old name is a grep failure recorded on the task; a reference to a target that no longer exists fails `make` naming it. | fn-166-mini-activity-a-small-complete-model-to.7 | — |

## Decision update (owner, 2026-10-13)

R6's rule-per-change is superseded: a DSL change ships a short migration recipe (concrete steps for a real feature) instead of a proven Scalafix rule, and the owner's sweep applies the recipes by hand. Rules already written stay until the sweep applies them. Task evidence is the `make model-check-changed` record plus mini equivalence, with the previous task's record as the baseline and no separate before run.

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **Obsolete:** the mini step-table baseline, and the `umpire-check-equivalence`/`umpire-capture-baseline` renames in .7.
- **.4:** lower the mini Cases after fn-173 Phase 1's fingerprint change.
- **.5:** the loop and its evidence lose step-table equivalence.
- **Unaffected:** .6/.10/.15.
