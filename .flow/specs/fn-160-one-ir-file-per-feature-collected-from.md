# One IR file per feature, collected from the feature's package

> HTML render lens (local): open `.flow/artifacts/fn-160-one-ir-file-per-feature-collected-from/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Framework-only execution boundary (owner override)

Existing source layout checkpoints stay complete. Remaining feature collection/identity and Temporal Protocol/layout migration proofs are held; none gates native framework DSL. Do not reopen integrated source renames from stale task metadata or touch disabled Temporal sources. Re-anchor any demonstrated generic authoring gap separately before the later IR phase.

Native DSL changes ship a concise migration recipe in framework-local guidance, not a task-specific Scalafix implementation or copied-Temporal proof. This owner iteration policy supersedes older rule/sweep mandates below. Temporal migration, production rule loading and full integration proofs remain held.

## Conversation Evidence

> user (turn 15): "explain the idea behind irFile - wht are there more than one per feature?"
> user (turn 16): "it seems overly complicated; why not a single file per feature? where you only specify the root of the graph and everything else is collected along with it?"
> user (turn 17): "yes, make separate spec"
> user (turn 17): "should we require a file to at most contain only one Machine? that would force a separate file for each and help scan them quickly as they are all visible from the file tree"
> user (turn 18): "yes add to fn-160."
> user (turn 33): "yes do A"
> user (turn 33): "but let's not have nested packages like pausing/ and instead a flat list of files grouped by their prefix"
> user (turn 41): "yes both"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 50% [paraphrase] -->

Every feature currently declares its checked-in IR files by hand. An `exports` object lists one `irFile(name)(roots*)` per file, and each root (machine, composition, `queries` section, `capabilities` section, realization) is named explicitly. The standalone activity splits into three files with about fifty roots between them, and nexus workflow into three more. The user finds this overly complicated. The lists are an inventory that has to be kept in step with the code, and a `queries` section nobody lists silently drops out of the IR.

The goal: one IR file per feature, collected automatically, with nothing to list.

## Architecture & Data Models
<!-- scope: technical -->

References in the model graph point downward: a Query to its machine, a realization to its machine, a machine to the machine it refines. No single object reaches everything a feature declares, so a single root object would miss the Queries, capability claims and realizations. [paraphrase]

The feature's **package** is the root instead. The IR generator, which already reads every Model's compiled TASTy in one run, collects every machine object, composition, `queries` section, `capabilities` section, progress claim and realization declared in the feature's package and its sub-packages (`product/`, `system/`). It then adds everything those reach, as today, e.g. a shared foundation's machines. One feature yields one IR file. [paraphrase]

What a feature folder is follows the existing layout rules: a folder with a feature file, such as the standalone activity, nexus standalone or nexus workflow. [inferred]

## Edge Cases & Constraints
<!-- scope: technical -->

- A declaration reached from two features is lifted into each feature's file, as a root of two files is today. [inferred]
- Today's split separates design alternatives and adversarial models from the main model: the history-record and admission designs, the held race, the forged control, and the close and reset designs. Merged in, they stay distinguishable by their machines and by the expectations their Queries already declare. [paraphrase]
- Generated Case file names start with their IR file's name, and two other trees pin Cases byte for byte (the functional tests' generated fixtures and the canary's case bindings). Merging files renames Cases in all three. [paraphrase]
- Go tests and the canary preflight harness name today's files (`activity-standalone-race`, `activity-standalone-record`, …) directly. [paraphrase]
- Each IR file has its own accepted-lint-findings file; merged files need merged acceptances. [paraphrase]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Before any migration, every consumer of the IR's file boundaries is identified: Go tools, the gate, Quint export slices, Case naming and fixture trees, the canary, and any checker that is given one whole IR file and relies on it holding only some Queries. Each gets a stated migration or a stated reason to keep a split. [inferred] Errors: if a consumer needs a split that a single file can't serve, the spec records it and the user decides before migration starts.
- **R2:** Each feature folder produces exactly one IR file, holding every machine, composition, Query section, capability section, progress claim and realization declared in its package and sub-packages, plus what they reach. [paraphrase] Errors: a feature package that declares nothing liftable produces no file and no error; a declaration the generator cannot lift is refused as today, naming it.
  - *Amended 2026-10-11 (owner, via fn-166):* the one IR file lives in the feature folder as `generated/ir.json`, beside `ir.lint.json`, `manifest.json` and the feature's `<query>.case.json` files; the central `model/ir/` and `model/cases/` trees are removed by fn-166.1, which also migrates every consumer in R1's inventory to a path helper. fn-160.2 collects the file from the package into that location.
- **R3:** No Model lists IR roots: `irFile` and the `exports` objects are removed, and the gate's construct-every-root test constructs what each feature package holds. [paraphrase] Errors: no error surface beyond R2.
- **R4:** A Query section, capability section or realization can no longer be left out of the IR by omission. [paraphrase] Errors: if a declaration must stay out of `model/ir` on purpose, an explicit opt-out marks it, and the generator reports each opted-out declaration.
- **R5:** The merged IR keeps every machine, Query, Case and claim the split files had, with the same behaviour. The only changes are file boundaries, file names and the names derived from them (Case files, lint acceptances, export slices). [inferred] Errors: any other difference in the regenerated IR or Cases is a defect to explain before it is accepted.
- **R6:** Every consumer from R1 is migrated (Go tests, canary preflight, fixture trees, export README tables, README), and the full gate passes. [inferred] Errors: no error surface beyond R5.
- **R7:** Each Model file declares at most one machine, composition or derivation, in a file named after it, in flat level folders where a family's files share a name prefix (no nested packages), so the sorted file list shows every machine and its group. [user] ("let's not have nested packages like pausing/ and instead a flat list of files grouped by their prefix"; "should we require a file to at most contain only one Machine? that would force a separate file for each and help scan them quickly as they are all visible from the file tree"; "yes add to fn-160.") Errors: a file with a second machine is refused by the lifter, naming the file and the machine; the regenerated IR differs only in source paths and positions.
- **R8:** Realizations form a layer of their own: a flat `realization/` folder beside `product/` and `system/`, each realization in a file named after its subject plus `Run`, with shared helpers in one kit file. [user] ("yes do A") Errors: a realization not named after the subject it realizes is refused by the lifter, naming both.
- **R9:** Layer dependencies point one way, realization → system → product: product imports neither system nor realization, system doesn't import realization, and only the realization folder may use the Temporal API. [paraphrase] Errors: an import against the direction, or an API use outside `realization/`, fails the lint at its line.
- **R10:** A foundation or actor package (e.g. the task queue, the worker), or a kind folder (`features/activity`, `features/nexus`), produces its own IR file when it declares Queries or realizations. A feature still lifts the foundation machines it reaches, but never another package's Queries, so nothing is lifted twice. [user] ("yes both", choosing option (a) for foundations) Errors: a Query section reachable only from another package's file is reported by the lifter, naming its package.
- **R11:** Exploration checks only the Query it was asked about, and an exploration candidate's id no longer hashes the whole IR file, so merging a feature's files doesn't multiply candidate work or churn candidate ids. [paraphrase] Errors: no error surface beyond today's exploration refusals.

- **R12:** The level between Product and Realization is named **Protocol**, not System: level folders `system/` become `protocol/`, the level's machines and files are renamed (`ActivitySystem` → `ActivityProtocol`, `system/System.scala` → `protocol/Protocol.scala`, and likewise for Nexus and the task queue), and the README, SEMANTICS, structure lint and level vocabulary say Product / Protocol / Realization, so the three levels sort alphabetically in their conceptual order. [user] ("yes, add task to rename System to Protocol") Errors: no error surface; machine-scoped ids change and are listed in the batch rename map.

- **R13:** A kind folder holds its kind file (`Nexus.scala`, `Activity.scala`: shared types and signature) and its forms (`standalone/`, `workflow/`); anything two forms share lives under `shared/`. The Nexus Product, one machine both forms refine (`NexusProduct`), moves from `features/nexus/product/` to `features/nexus/shared/product/`. A form's root feature file is named `<Form><Kind>.scala` (`StandaloneActivity.scala`, `StandaloneNexus.scala`, `WorkflowNexus.scala`). The structure lint enforces both. [user] Errors: no error surface; package and file ids change and are listed in the batch rename map.
  - *Superseded in part on 2026-10-11 (owner): forms are no longer nested under the kind folder; see R14. The file naming (`<Form><Kind>.scala`) stands.*
- **R14:** `features/` is flat. A kind folder, `features/activity/` or `features/nexus/`, holds only what the kind's forms share: the kind file (`Activity.scala`, `Nexus.scala`) and the shared machines that R13 put under `shared/` (the Nexus Product). Each form is a top-level folder named `<kind>_<form>`: `features/activity_standalone/`, `features/activity_mini/`, `features/nexus_workflow/`, `features/nexus_standalone/`, with its levels (`product/`, `protocol/`, `realization/`) and its `generated/` tree directly inside. Packages follow folders (`temporal.features.activity_standalone`). A form depends on its kind folder and on nothing else under `features/`; a feature selection that names a form includes its kind folder. The structure lint refuses a form folder nested under a kind folder, a kind folder that holds a level folder, and an import from one form into another. [user] ("let's flatten the structure; features/activity and nexus shouldn't be nested; instead use features/activity and features/activity_standalone etc. this is simpler. for nexus it might be features/nexus (for shared) and then nexus_workflow/ and nexus_standalone/ as well") Errors: package and file ids change and are listed in the batch rename map; a nested form folder or a level folder in a kind folder is refused by the lint naming the folder.

- **R15:** Case identities derive from the feature package, not from a file name or an authoring language: the `scala.` set in generated Case ids (`temporal.case.scala.<model file>.<query>`) becomes the feature's package spine, as the definition ids already carry it (`temporal.features.<feature>.case.<query>`, with the level named per R12); `scala.explore.` on exploration candidates becomes `explore.`; the `umpire-scala.` prefix of activity type names and the `scala-model` provenance string drop `scala` likewise. The `set` is computed once, in the lowering, from the IR's package root. Every pinned consumer copy (functional fixtures, the canary binding and its Driver Profile, unit tests asserting a literal id) is regenerated or rewritten in the same change, and the equivalence baseline compares the before under a mapped identity so the rename reads as a rename, not a semantic change. [user, "drop `scala.` from paths"; inferred] Errors: a Case id that still carries `scala.` after lowering fails the gate naming the Case; a Driver Profile that names a changed activity type or task queue is rewritten deliberately on the task and recorded, never widened silently.

## Boundaries
<!-- scope: business -->

- Changing what the models say (machines, Queries, claims, realizations) is out of scope; only how IR files are assembled and named, and how Model files are laid out, changes. [inferred]
- Changing the IR schema is out of scope. [inferred]

## Decision Context
<!-- scope: both — conditionally substructured -->

The user asked for a single file per feature where only the root of the graph is specified. [user] A single root object was rejected because model references point downward and no object reaches a feature's Queries, claims and realizations. The feature's package serves as that root instead. [paraphrase] Keeping explicit `irFile` lists but defaulting the main file to its folder's sections was considered as a middle ground. It was not chosen: it keeps the multi-file split and part of the inventory. [paraphrase]

- **Level names (owner, 2026-10-10):** Product and Realization stay; System becomes Protocol: the server's protocol between its parts (history, matching, timers, durability) that refines the Product. Rejected: Guarantee/Mechanism (owner keeps Product), Production (reads as the prod environment), Provider (task-queue providers exist). [user]

- **Kind layout (owner, 2026-10-10):** a Product shared by two forms goes under the kind's `shared/`, not beside the forms; one Product per form (duplicating NexusProduct) was rejected, since both forms refine the same promises (fn-126 decision 16). Form feature files are `<Form><Kind>.scala`. `nexus/workflow/Workflow.scala` becomes `WorkflowNexus.scala` (owner, 2026-10-10). [user]

- **Run names (conductor, 2026-10-11, under the owner's recommendation mandate):** a form's main realization is `<Form><Kind>Run` (`StandaloneActivityRun`, `StandaloneNexusRun`, `WorkflowNexusRun`), matching the form's file, because both Nexus forms' subject is `NexusProtocol` and `<Subject>Run` would collide; any further realization is `<Subject>Run`.

## Parked unknowns

- Whether any consumer relies on today's file boundaries in a way one file per feature can't serve. The exploration and race checkers are the suspects, from the race file's comment "so the history record's Queries are the ones its checkers were given". R1's investigation resolves this. [inferred]

## Resolved via Codebase

Read-only audit for fn-160.1 (R1). Paths are relative to the repository root. "Inferred" marks what was read but not run. The working tree's fn-147 change breaks the IR generator build, so nothing was regenerated.

### Today's IR files (model side)

| File | Declared at | Roots | Why separate (its comment) |
| --- | --- | --- | --- |
| `activity-standalone` | `model/temporal/features/activity/standalone/Standalone.scala:149-188` | 38: product, capabilities, 17 `queries` sections, `StandaloneActivity` and 15 realizations | The main Model |
| `activity-standalone-record` | same file `:191-206` | 11 `queries` sections, 4 of them `foundations.taskqueue.system.*`; compositions `RecordOverForgetful` and `RecordOverVolatile`, which no Query runs over | "Its history record, the admission designs, and the shared task queue's providers it composes" |
| `activity-standalone-race` | same file `:210-215` | `HeldDispatch.queries`, `HeldDelivery`, `LostStartAnswer.queries`, `LostAdmissionResponse` | "a Model of its own, so the history record's Queries are the ones its checkers were given" (see the answer below) |
| `nexus-standalone` | `features/nexus/standalone/Standalone.scala:46-51` | `NexusSystem`, its capabilities and queries, the realization `Standalone` | Already one file per feature |
| `nexus-workflow` | `features/nexus/workflow/Workflow.scala:117-127` | 9, including `temporal.actors.worker.Polling` (outside the package; also reached through `HandlerWorker extends Derived(Polling.restrict…)`, `system/System.scala:482`) | The main Model |
| `nexus-workflow-control` | same file `:130-131` | `TrustingCaller.queries`, `ForgedControl` | "The forged completion a caller must refuse" |
| `nexus-workflow-close` | same file `:134-155` | 9 design `queries` sections and 10 progress claims from `properties` sections | "The close and reset designs" |

Coverage of collecting by package: every machine object, `queries` section, realization and progress claim in the three form packages is rooted or reached today. The check compared each top-level object with the IR's machines and compositions, each `object queries`, and each realization in `Realization.scala`. `ActivityProduct.queries` (`product/Product.scala:300-301`) holds only `capabilities.bound(three)`, which is already applied. The capability sections of the record designs (`Dispatch.scala:268,334`, `DispatchWithTaskQueue.scala:120-272`) are reached through their `capabilities.bound(five)`. Package collection therefore adds nothing from the form packages (inferred until a lift confirms it). It loses the four `foundations.taskqueue` `queries` sections (`foundations/taskqueue/system/System.scala:184,267,284,292`): they are roots, not reached, and they sit outside the activity package. See the decision list.

Merge safety: Query names must be unique per file (`tools/umpire/ir/validate.go:147-148`). Across each feature's files the only shared names are the three `activityProduct.*` capability Queries, which are the same declaration (same position, same limits) and collapse to one copy. No machine, composition, property, scenario, progress or realization name collides with a different declaration. Each machine keeps at most one realization: activity 15 + 2, nexus `nexusSystem` plus `trustingCaller`. `lower.Realizable` (`tools/umpire/lower/lower.go:154-174`) and `conformance.realizationOf` (`conformance/plan.go:160-177`) need exactly that.

Collection rule: the lifter's `Index.defs` (`model/irgen/Context.scala:17-30`) already holds every `ValDef`/`DefDef` symbol of every inspected TASTy file in one run. `liftRoot` (`model/irgen/Lifting.scala:24-60`) already tells machine, composition, realization, `capabilities` and `queries` objects apart by symbol. Enumerating a package means filtering `defs` by `fullName` prefix. A "feature" is already defined by the structure lint (`model/irgen/Structure.scala:10-44,58-71`): a package under `features`, `foundations` or `actors`, with kinds (`features/nexus`, `features/activity`) holding forms `workflow/` and `standalone/`. A kind's shared Product (`features.nexus.product.NexusProduct`) sits outside both form packages and is reached through each form's refinement, so it is lifted into both files, as today.

### Consumers of file boundaries or names

| Consumer | Keys on | Under one file per feature | Migration |
| --- | --- | --- | --- |
| Lifter `--ir` | `Lifting.irFiles` (`model/irgen/Lifting.scala:86-121`); `Lift.scala:73-86` (`Target.IrFiles`, optional file names); `source = "model: <roots>"` (`Lift.scala:124`); waivers by file (`:141,253-266`) | Files come from feature packages, not `irFile` vals | Replace `irFiles` with package enumeration. `source` lists the collected roots. Waivers stay keyed by file name |
| Structure and order lints | `object exports` required in a root feature file (`Structure.scala:554-567`); `Kind.File` and the `exports` ordering (`Order.scala:320,340,529-553`) | `exports` is gone (R3) | Drop the requirement and the ordering kind; refuse a stray `irFile` |
| irgen fixtures | `model/irgen/testdata/**` (`irFile` in `layout`, `layoutRefusals`, `lifts`, `irFileRefusals`, `crossed`, `derivedTwins`, …); `test/Fixtures.test.scala:1097-1147,3084-3097,3257-3275` | The fixture contract changes | Rewrite them as package-collected fixtures; `irFileRefusals` becomes opt-out (R4) refusals |
| Gate | `Gate.scala:504-523` (one lift, names no file); `settle` (`:559-589`) holds `model/ir` to what the lift produced; `acceptWaivers` (`:591-620`) | Nothing in the gate names a file. On update, `settle` refuses orphans, so the old files must be deleted, not just overwritten | Delete the 4 retired `.json` and `.lint.json` files in the same change. Update the orphan message "declare it with irFile" (`:566`). In `Gate.test.scala:573-581,736-787`, update the 7 stand-in names |
| `IrFiles` test | `model/temporal/IrFiles.test.scala:12-16,41-52`: `declaring` = the 3 `exports`; `IrFile.declared` equals `model/ir`; `construct()` covers every lifted machine | No `exports` or `IrFile` (R3) | The JVM can't list a package's objects. Read each `model/ir/<f>.json` `source` (FQNs) and load each root reflectively (`Class.forName(n + "$")`), then reuse `IrFile.construct`'s walk, moved out of `IrFile` (inferred approach). `framework/Rules.test.scala:464` uses `IrFile(...).construct()` directly |
| Case naming and manifest | `lower/generated.go:103-109`: name `<file>-<query>-case.json`, Case identity `temporal.case.scala.<file>.<query>`; manifest `model` field | Race Cases become `activity-standalone-heldDispatch.staleDelivery` and `activity-standalone-lostStartAnswer.committed`; control becomes `nexus-workflow-forgedCompletion`. With a `<kind>-<form>` file name, every main-file Case keeps its name and identity (inferred) | `make umpire-gen-model`; R5 holds every other Case byte-identical |
| Functional fixture tree | `cmd/umpire-gen-cases/main.go:30-44` pins `{nexus-workflow-control.json, forgedCompletion}`; `tests/testcore/testpilot/testdata/generated/`; `generated-case-names.txt` | The control pin renames | Change the pin to `nexus-workflow.json`, regenerate (`umpire-gen-fixtures`), update the names list |
| Canary tree | Pins only `{nexus-workflow.json, syncCompletion}` (`main.go:43`); `tools/canary/casebinding/casebinding.go:28` embeds `nexus-workflow-syncCompletion-case.json` | Unchanged if the file stays `nexus-workflow` | None beyond `canary-gen-case --check` |
| Pinned recorded Run | `Makefile:579-580`, `common/testing/testpilot/replay/testdata/nexus-workflow-control-forgedCompletion-run.json`. `FindGeneratedCase` (`lower/generated.go:256-290`) matches by `CaseFingerprint`, a hash of the whole Case including its id (`common/testing/testpilot/assessment.go:192-201`) | The control Case's id changes, so the recorded Run no longer matches | Re-record with `make umpire-rerecord-pinned-runs` (live) and rename the file. `recordedrun/testdata/companion-migration.json` holds the old names as history only (inferred) |
| `umpire-run --model`, `umpire-assess --model` | `umpire-run/run.go:170-185`, `umpire-assess/run.go:197-206`: Case found by fingerprint, then `ir.Load(<entry.Model>)` and `conformance.Prepare(model, entry.Query)` | The model is bigger and the Query is looked up by name (unique) | None |
| `umpire-ir-bridge`, `umpire-fuzz --set`, `umpire-replay --set` | `--set` is a model-declared `explore(...)` name, not a file (`umpire-fuzz/run.go:368`, `umpire-replay/run.go:120`). The bridge loads every file (`cmd/umpire-ir-bridge/main.go:36-49`) and picks the one whose Query declares it, failing on "ambiguous exploration" (`explore/bridge.go:330-346`) | The selection is unaffected (2 explorations, `nexusControl` and `nexusDeadlines`, both in nexus-workflow afterwards). But a candidate's digest hashes the whole file (`explore/explore.go:95-115`) and becomes its Case id (`:127`). Each candidate also runs `NewProducer`, which `check.Check`s every Query of the file (`lower/lower.go:104-116`) | Candidate ids change once, and from then on change whenever anything in the feature changes. Per-candidate cost goes from 1 Query (control) to about 180 plus 10 progress claims (inferred; not measured). Recommended: check only the asked Query in `Producer`, or explore a model pruned to the Query's reach. No checked-in fixture pins a candidate digest (grep `scala.explore`) |
| Lint acceptances | `<file>.lint.json` beside each file. `umpire-lint` fails on an orphan (`cmd/umpire-lint/main.go:85-100,168`), on a duplicate (`lint/accept.go:87`) and on a stale acceptance | Merging duplicates 314 activity acceptance subjects (none have conflicting reasons). Kinds judged over the file's Queries can flip and go stale: `unread-refinement`, `untaken-choice`, `unrealized-find`, `unfired-verify` (`lint/kinds.go:56,192,305,619`) | Merge the files with duplicates removed, rerun `umpire-lint`, drop what is now stale, delete the old `.lint.json` files |
| Quint export | `tools/umpire/export/quint_test.go:26` slices `activity-standalone`, `-record`, `nexus-workflow`, `-close`; per-slice compositions `:1015-1029`; README tables `export/README.md:36-45,146` | Slices become activity-standalone and nexus-workflow, and now also hold the race and control machines and nexus-standalone. Nothing shows why race, control and nexus-standalone were never exported, and whether export accepts them was not verified | Retarget `irFiles` and `exportedOf`, rerun `make` (the backend target, `Makefile:651-660`), rewrite the README tables from the run. A slice that fails to open is an R5 defect to explain |
| Go tests pinning a whole file's receipts | `check/activity_system_test.go:128-213` (exact receipt map of `-record`); `check/replay_test.go:24-44` (Found list per file, 38 total); `check/nexus_close_test.go:264,817`; `check/nexus_close_baseline_test.go:301` (count = Queries + 3×progress); `export/quint_test.go:1015-1029` | Merged files add receipts from other designs | Filter each assertion to its machines and compositions (by `ClaimKey.Owner` or family) instead of the whole file |
| Go tests naming a retired file (path or Case name) | `tools/umpire/{check/activity_parity_test,composed_test, conformance/race_test, explore/bridge_test,totals_test, export/{dump_builder,open,raw,verify}_test, ir/{diagnostics,helpers,realization}_test, lint/{coverage,kinds}_test + testdata/coverage.golden, lower/{choices,fixture,generated,lower,origin_identity,race,waits}_test, realization/carriers_test, cmd/umpire-{assess,run}/run_test, cmd/umpire-gen-cases/main_test}.go`; `tools/canary/preflight/harness_test.go:67`; `tests/testcore/testpilot/{model_fixture_test.go:110, switch_test.go:128}`; `tests/testpilot_nexus_control_case_test.go:34,115` | The paths and names move | Mechanical rename. `lower/inventory_test.go:116,347` reads `GetRealizations()[0]` of nexus-workflow, which becomes 2 realizations sorted by id: still `asyncNexus` first (inferred) |
| Docs | `model/README.md:217-222,379,657-767,1366-1367`; the `irFile` header comment in `model/framework/IrFile.scala`; the taskqueue header (`foundations/taskqueue/TaskQueue.scala:29-30`); `TrustingCaller.scala:5`; `export/README.md` | They describe `irFile`/`exports` | Rewrite them with the migration |

### Does any checker rely on a file holding only some Queries?

No production checker relies on it semantically. Every tool looks a Query up by name within a file: `check.Realizer.Declared` (`check/claims.go:323-333`), `Producer.found` (`lower/lower.go:110-113`), `generateCase` (`lower/generated.go:103-113`). Those names are validated unique per file, and a realization is chosen by the Query's own Scenario machine (`lower/lower.go:154-174`). The race comment is historical. When the race split off (commit `336436b9b3`), the held realization was declared on the record's own machine (`machine = currentAdmission` → `heldAdmission`). In the record's file, `Realizable` would have lowered every record find Query through it, and `TestActivitySystemResults` pinned that file's whole receipt map. Today `HeldDispatch` is a machine of its own with its own realization (`system/DispatchRaces.scala:31`, `system/Realization.scala:1015`), so that reason is gone. What does depend on today's files:

- Go tests that pin a whole file's receipts (listed above). They need filtering, not a split.
- Exploration: candidate identity and cost scale with the whole file (`explore/explore.go:95-127`, `lower/lower.go:104-116`). Not a correctness dependency.
- File-scoped lint kinds, whose accepted findings must be recomputed.

None of these needs a split.

### Needs owner decision

1. **Foundations and actors.** `foundations.taskqueue`'s four `queries` sections (task queue, forgetful, volatile and lossy providers: 25 Queries) are rooted today only by `activity-standalone-record`, from outside the activity package. Its header says "The queue declares no IR file of its own: the features that compose it export it" (`foundations/taskqueue/TaskQueue.scala:29-30`). Package collection drops them unless one of these is chosen:
   - (a) A foundation (and an actor with something to lift) is a feature folder with its own IR file, `taskqueue`. Its Queries move there and leave the activity file.
   - (b) A feature also collects the `queries` sections of foundations whose machines it reaches. This repeats them in every composing feature.

   `actors/worker` (`Polling`, no Queries) produces either a `worker` file under (a) or nothing new under (b), since it is already reached. This also decides whether kind folders (`features/nexus`, `features/activity`) produce a file. Today they hold `NexusProduct` and no machine respectively, and the inferred default is no.

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope. Layout, naming and the Protocol rename apply to the mini feature and the shared code. Moving and renaming the real features' files, including task .7's real forms, waits for the sweep.

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **.2/.6/.7/.8:** `STEPS_ONLY` checks become a fresh-lift IR diff.
- **.8:** its canary and fixture regeneration should share fn-173 Phase 1's.
