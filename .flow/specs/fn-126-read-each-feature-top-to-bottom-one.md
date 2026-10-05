# Read each feature top to bottom: one module object per machine

## Goal

A feature developer who wants to know what a machine does, what it promises and what is asked of it should read one file from the top down, the way a Quint module reads: types, then the actions every machine binds, then one object per machine holding its vocabulary, steps, machine, Properties, capabilities, Scenarios and Queries, then the next machine's object, then what the feature declares across machines, ending with its IR files. `Realization.scala` stays beside it.

Owner decision (2026-10-05): adopt option (a) of the Quint-module study, `.plans/QUINT_MODULE_LAYOUT.md`. This is **layout only**. No construct is added to `model/umpire`, nothing is written in Quint syntax, and what the Models mean does not change. The one new tool is a lint that keeps the reading order safe, because Scala initializes an object's vals in the order they are written.

Paths are the ones fn-114.9 gives: `model/irgen`, `model/check`, `model/temporal/features/*`, `model/temporal/shared/*`.

## Why

The study measured the standalone activity after fn-112, fn-114 and fn-122:

- **Many files.** The feature has 15 files, 1,762 lines, 141 top-level declarations and 3 `DefinitionScope` pins. The Nexus caller repeats the pattern in 9 files and 2,450 lines.
- **Many hops.** Following `activityProduct` from its state type to a Query takes 4 files and about 675 lines of context. A typical Query is 5 files away from its state, and a Case 9 to 11.
- **References no import shows.** Files refer to each other through same-package top-level vals, so no `import` says which file a name comes from. `Properties.scala` uses about 42 symbols from `Model.scala`, `Queries.scala` about 67, and `Realization.scala` about 70.
- **A step far from its claim.** The step `attemptResult(completed)` is in `Model.scala:270-284` and bound at `:385`. Its Property is in `Properties.scala:12-14`, its Scenario in `Queries.scala:25-26` and its Query at `:101-102`.

Some of these splits were chosen on purpose and some only happened.

- **On purpose.** fn-112 split files by kind: monitors stay in `Model.scala`, because two files would initialize each other in a cycle, and Properties form "the specification a reviewer reads on its own". fn-112 also made the subject folders and the per-machine vocabulary objects.
- **Only happened.** `Capabilities.scala` (fn-122.8) and `IrFiles.scala` (fn-114.1) each have a declaration placed on purpose, but nobody argued for a separate file. The per-kind file names themselves were never argued either.

Inside one object, the init-cycle reason goes away, provided the declaration order is checked. "Read on its own" can be kept as a section of the object. The lifter already lifts object members (`Control.forgedCompletion`, `Functional.completion`).

After the change, following the product machine from its state to its Query means reading one section of about 120 lines, and a step and its Property sit in the same object.

The cost is one regeneration of the IR and Cases. In `activity.json`, 1,161 source positions change, and 21 Case files change. Moved function symbols also have to be recorded in the golden configuration.

## Requirements

- **R1 One feature file per folder.** Each Model folder holds one feature file named after the folder, its `Realization.scala` where it has one, and its tests. Altogether, 30 per-kind files become 8 feature files.

  | Folder | Feature file | Module objects (names settled by task 1) |
  | --- | --- | --- |
  | `features/standaloneactivity` | `StandaloneActivity.scala` | `Product`, `Protocol` |
  | `…/standaloneactivity/admission` | `Admission.scala` | `Admission` (current, stale, held), `ResponseLoss` |
  | `…/standaloneactivity/compositions` | `Compositions.scala` | `OverQueue`, `OverMatching` |
  | `features/nexuscaller` | `NexusCaller.scala` | `Product`, `Protocol`, `Control` |
  | `…/nexuscaller/closepolicy` | `ClosePolicy.scala` | one for `rejectAfterClose` and its nine derivations |
  | `features/nexusoperation` | `NexusOperation.scala` | `Operation` |
  | `shared/taskqueue` | `TaskQueue.scala` | the opaque contract (`dispatchQueue` and its storage-loss variant); the matching provider (`matchingQueue`, lossy, forgetful, volatile) |
  | `shared/worker` | `Worker.scala` | one for `polling` |

  A module object is the object of one machine together with the machines derived from it, or that share its state type and vocabulary. Where a vocabulary object already exists, it becomes the module object.

- **R2 Reading order.** A feature file has the following sections, in this order:
  1. A header comment that names the module objects in order, then the imports, the file-level pin and the family objects.
  2. **Types.** Every enum, state case class and type alias, at the top level.
  3. **Signature.** Parties, entities, inputs, actions, timers, internal steps, choices, observations, shared Limits, and the top-level `given`s.
  4. **Module objects**, in dependency order: a refined machine comes before the machine that refines it, and a source machine before the machines derived from it.
  5. **The feature section.** An object named after the feature.

  Inside a module object, the order is:
  1. vocabulary (status sets and constants);
  2. step functions;
  3. monitors, assumptions, holes and channels;
  4. the machine and the machines derived from it;
  5. Properties and progress claims;
  6. capabilities;
  7. Scenarios, Limits and Queries.

  Errors: R4 refuses a section out of order.

- **R3 Placement.** Each declaration has one place:
  - A Property, Scenario or capabilities declaration over a machine is a member of that machine's module object.
  - A Query lives with its Scenario.
  - A declaration goes in the feature section if it is over a composition, spans two module objects, or reads the realization. Examples: the composition with the worker, `protocolCapabilities` (it reads `ActivityRealization.activityStatus`) and the `irFile` roots.
  - A derived machine whose source is declared in another file, such as `activityWorker` from `shared/worker`, goes in the feature section.

  Errors: R4 refuses a misplaced declaration and names the object it belongs in.

- **R4 Declaration-order lint in the model gate.** `make umpire-check-model` (`model/check`) runs a lint over every Model. It refuses each of the following at `<file>:<line>`:
  - (a) a val that is read during initialization before it is declared in the same owner;
  - (b) an initialization cycle between the owners of one Model package tree: package objects, module objects, feature objects and realization objects;
  - (c) a module-object member that is out of R2's order;
  - (d) a declaration that breaks R3.

  Reads inside a `def`, a lambda, a by-name argument or a lazy val do not count as initialization reads. Task 1 decides whether (a) and (b) come from Scala's own checkers (`-Wsafe-init`, `-Ysafe-init-global` in `model/project.scala`, with `-Werror`) or from a pass in `model/irgen`, and records the decision. Errors: each kind has one refusal fixture. The lint passes on all Models and refuses nothing in the lifter's passing fixtures.

- **R5 Meaning and identity frozen.** The following stay exactly as they are:
  - Definition IDs and IR type names;
  - the names of machines, compositions, Properties, Scenarios, Queries, Limits and law instances (`<machine>.<law>`);
  - transition tables, refinement rows and fingerprints;
  - Query answers, lint findings and coverage;
  - Quint/P agreement;
  - every Contract and Case byte, apart from source positions.

  Only these deltas are allowed, and each is recorded:
  - source file paths and lines, in the IR and in Cases;
  - the IR `source` root strings;
  - the symbols of moved functions (`Model$package$`/`Product$` → `StandaloneActivity$package$`/`Product$Steps$`).

  They are recorded in `tools/umpire/internal/golden/config.json` (`function_name_substitutions`, `source_root_moves`, path entries) for as long as that harness exists. If fn-124.7 has retired it, they are recorded in a before/after projection of the reader's outputs, stored under `.flow/tmp/`. Errors: any other difference stops the task.

- **R6 Pins.** Every owner that holds an ID-bearing declaration (an action, monitor, assumption, hole, channel or realization) pins its former owner once, through the existing `DefinitionScope`. ID-bearing declarations are direct members of a pinned owner, never of an object nested in one: nested owners keep their own IDs, and a nested pin is refused. Types stay at the top level under the file-level pin, which keeps their `pkg.Type` IR names. No per-declaration ID string is added, and no pin string beyond today's former owners. Errors: the lifter refuses a duplicate ID, as it does today.

- **R7 Machine names without literals.** Each machine's `val` keeps its name inside its module object (`Product.activityProduct`), so derived Query names (`<machine>.<scenario>.<property>`) and law names stay the same. No `machine("…")` string is added, and the string-literal count of each Model (by fn-112 R18's method) does not rise.

- **R8 What stays.** `Realization.scala` stays a sibling file. Its only changes are imports and qualified references to declarations that moved. The subject folders `admission/`, `compositions/` and `closepolicy/` stay subpackages. `model/umpire`, `model/temporal/{capabilities,realize}` and the IR schema are untouched.

- **R9 Docs.** The following documents describe the new layout and name no removed file:
  - `model/README.md`: the layout section under "Writing a Model", with the activity as its example and R2's order; "Where things are"; the sentence on capabilities that says "its own `Capabilities.scala`"; and the line-referenced examples such as `nexuscaller/Properties.scala` line 22;
  - `model/SEMANTICS.md`;
  - `.plans/UMPIRE_MODULES.md`: the Models row, the Nexus operation row and line 321;
  - `.plans/UMPIRE4_VISION.md`;
  - `AGENTS.md`.

  Errors: R10 finds a stale path.

- **R10 Old names stay retired.** fn-114.9's layout test (`tools/umpire/model/layout_test.go`, or wherever fn-124.8 moves it) fails in two cases:
  - a file named `Model.scala`, `Properties.scala`, `Queries.scala`, `Capabilities.scala` or `IrFiles.scala` exists under `model/temporal/features` or `model/temporal/shared`;
  - a live file names such a path. This includes Go tests that assert a Scala position, such as `lower/activity_test.go:742`.

  The kit's `model/temporal/capabilities/Capabilities.scala` is not matched. Errors: a test case proves that each of the five names is detected.

- **R11 Evidence.** The done summary states, for each folder:
  - files and lines before and after (`umpire.check.metrics`);
  - for `activityProduct` and `nexusProduct`, the files and lines read from the state type to a Query;
  - the count of cross-file same-package references.

  It also lists every recorded delta from R5.

## API sketch

```scala
/* The standalone activity: Product, Protocol (refines Product), then the feature: the protocol with
 * the worker of its task queue, and its IR files. Realization.scala realizes it. */
package temporal
package features.standaloneactivity

given DefinitionScope = DefinitionScope("temporal.standaloneactivity.Model$package$") // actions, types

// -- types (top level, so IR type names stay temporal.standaloneactivity.<Type>)
enum ProductPhase derives Finite: …
final case class ProductState(phase: ProductPhase) derives Finite
final case class ProtocolState(phase: Phase, attempts: UpTo[2], …) derives Finite
final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState)

// -- signature: what every machine, the composition and the realization bind
val caller = Party()
val activity = Entity(key = "activityId")
val attemptStart = action(shared.worker.party).on(activity).schema[PollActivityTaskQueueResponse]
val control = action(caller).on(activity).input(Inputs.control)
val timeout = timer

object Product:                                        // the product machine's module
  import ActivityFamily.given
  def terminal(p: ProductPhase) = p.in(completed, failed, canceled, terminated, timedOut)
  def paused(s: ProductState) = s.phase == ProductPhase.paused
  object Steps:                                        // named after the actions they answer (OQ4)
    def attemptStart(s: ProductState) = …
    def control(s: ProductState, c: Control) = …
  val activityProduct = machine[ProductState, Outcome, ProductFact] {
    forEntity(activity); starts(ProductState(scheduled)); ends(s => terminal(s.phase))
    steps(attemptStart ~> Steps.attemptStart, control ~> Steps.control, timeout ~> Steps.timeout)
  }
  val terminalIsFinal = activityProduct.property.once(terminal).keeps(_.phase)    // what it promises
  val productCapabilities = capabilities(activityProduct, limits = three)(Closable(…), Pausable(…), Pollable(…))
  val pausedThenResumed = activityProduct.scenario.actions(control(Control.pause), control(Control.unpause))

object Protocol:                                       // refines Product, so it comes after it
  …
  val activityProtocol = machine[ProtocolState, Outcome, ProtocolFact] { refines(Product.activityProduct)(productOf); … }
  …                                                    // Properties, Scenarios (today's Paths), Queries (today's Functional)

object StandaloneActivity:                             // the feature: what spans modules or reads the realization
  val activityWorker = shared.worker.Worker.polling.restrict(workerStop, serve)
  val standaloneActivity = compose[StandaloneActivityState](_.activity -> Protocol.activityProtocol, _.worker -> activityWorker)…
  val protocolCapabilities = capabilities(Protocol.activityProtocol, limits = three)(…, Describable(status = ActivityRealization.activityStatus))
  val activityFile = irFile("activity")(standaloneActivity, Product.activityProduct, Protocol.functional, …, ActivityRealization.standalone)
```

The feature section is an object, not top-level code, for initialization order. The module objects read the top-level signature, so if top-level vals also read the module objects, the package object and the module objects would initialize each other in a cycle. That is fn-112's two-file cycle again, moved into one file.

## Boundaries / Non-goals

- No Quint syntax and no keyword aliases (`run`, `temporal`, `const`). fn-120's boundary stands. The Quint export still emits one module per IR file (`tools/umpire/export/quint.go`) and does not depend on the Scala layout.
- No semantic change: no new DSL construct, no IR schema change, no change to the reader, the lowering or the exports. The lint (R4) is the only new tool code.
- `Realization.scala` is not nested in a module object. A nested pin is refused, and nesting it unpinned would change its Definition IDs.
- The subject folders stay. Their reason, that otherwise each file holds "three unrelated subjects", is unchanged.
- No index or re-export module (option b of the study). Scala `export` creates forwarder vals that the lifter would see as a second declaration of the same name.
- No settings (fn-125), faults (fn-123) or rejecting rows here.

**fn-112 and fn-114 decisions this supersedes:**

- fn-112's "files by kind, folders by subject" becomes "files by subject, sections by kind". This replaces the layout table in fn-112's Architecture section and R11's per-subject Model/Properties/Queries files. Their reasons still hold or are no longer needed:
  - the init cycle that kept monitors in `Model.scala` is gone, because R4 checks the order inside one object;
  - "Properties read on their own" survives as a contiguous section;
  - "three unrelated subjects per file" survives as the subject folders.
- fn-114 R10's four-file rule, and the `model/README.md` text that states it, are replaced by R1 and R2.
- fn-114 R7 placed `irFile` values in each folder's `IrFiles.scala`. Only that placement is superseded, subject to OQ3; the `irFile` value form is unchanged.
- fn-122.8 gave each folder its own `Capabilities.scala`. That file becomes the capabilities section of the module object. fn-122.8's reason, that an entity's adoption stays with its Model, still holds.
- **Kept:** fn-112's objects per machine (they grow into module objects), the core/sugar file rule, the `DefinitionScope` rule and the semantic freeze; fn-114's name capture; fn-120's ban on Quint syntax.

**Where this deviates from the study's sketch, and why:**

1. **Actions stay in the signature section, not in the module objects.** Product, Protocol, the composition and the realization bind the same actions. Inside one object, a step function named after its action would shadow the action. Keeping them at the top level also keeps every action ID under the existing file-level pin.
2. **`val activityProduct` keeps its name, instead of `val machine` plus a name literal.** fn-114 removed name literals, and the current names keep derived Query names and law names stable.
3. **The existing vocabulary objects become the module objects.** No new `ActivityProduct` object is created, so the function symbols of the status sets do not move.
4. **The feature section is an object, and capabilities that read the realization go there.** The realization reads the module objects, so a module object that read the realization would form a cycle.
5. **The lint covers more than order inside an object:** it also checks cycles between owners and placement.

## Owner decisions

Decided 2026-10-05: all five recommendations below are adopted. Renames the DSL-simplification study proposes and the owner approves (e.g. `accept`, `attemptStart`, the `admission`/`compositions` names) are folded into this spec so files and names change once.

1. **Do Properties form their own section after the machine, or sit next to the step they constrain?** Recommendation: their own section. A Property reads the machine's val, so placing it next to a step means placing it above the machine, where it is `null` at initialization unless it becomes a `def` or a lazy val. A separate section also keeps fn-112's "read on its own" and keeps R4 simple. The step-to-claim distance is already one object.
2. **For a folder with one machine family (nexusoperation, worker, closepolicy), a module object or top-level declarations?** Recommendation: a module object everywhere. One rule for the lint and the README, and a second machine adds an object without moving the first. The cost is one indentation level and call sites such as `Worker.polling`.
3. **Do the `irFile` roots go last in the feature section, or stay in `IrFiles.scala`?** Recommendation: the feature section. They are the module's exports, nobody argued for their own file, and the parent's section already gathers the roots of its subject folders, as `activitySystemFile` does today.
4. **How is a step named next to its action inside a module object?** The options are a nested `object Steps` that keeps fn-112's names (`attemptStart ~> Steps.attemptStart`), or the study's suffix (`attemptStartStep`). Recommendation: `Steps`, in the three folders whose step names collide with their actions (activity, Nexus caller, Nexus operation). closepolicy, worker and taskqueue keep the names they already have (`closeStep`, `serveStep`, `enqueueView`).
5. **Land before fn-124.7 retires the golden harness?** Recommendation: yes. The moves are then recorded where the other spec migrations recorded theirs, and no one-off projection is needed (R5).

## Ordering

- **Entry:** fn-114 is closed (fn-114.9, then fn-114.8), and fn-122.6's documentation has landed, since it documents the `Capabilities.scala` files this spec replaces.
- **Not concurrently with fn-124.8**, which moves `tools/umpire/model` (where R10's test lives) and forbids other edits to `model/` and `tools/umpire/model` while it runs.
- **fn-118.4 and fn-118.5:** fn-118.5 rewrites the waits in `Realization.scala`, and this spec changes only the imports there. Prefer to land after fn-118.5; otherwise, whichever lands second rebases. Case regeneration is serialized with fn-118.
- **Before fn-125 resumes** (tasks 2 to 11 are blocked). Its settings then land as members of the module objects, like Quint `const`s: a setting that one machine reads goes at the top of that machine's module object, and one read by several machines goes in the signature. fn-125's sketch, which today names `nexuscaller/Model.scala`, is amended when fn-125 resumes.
- **fn-123** (faults) has no tasks yet. If it is planned first, it must not convert the queue providers or the response-loss machine while this spec moves them.
- **fn-120** is closed and not affected. **fn-118.5** touches only `Realization.scala`, which this spec leaves alone apart from its imports.

## Early proof point

Task 1 converts `standaloneactivity` (all three subpackages) and lands R4. It shows that the lifter lifts step functions from a nested `Steps` object and `irFile` vals from an object, and that R5 holds with only recorded deltas. If R5 needs any delta beyond positions, root strings and function symbols, stop and re-evaluate the module shape before the Nexus folders and `shared/`.

## Verification

```bash
make umpire-check-model && make lint-model && make lint-code-fast   # the gate also runs the Models' munit tests (StandaloneActivityPins, Catalog)
go test -count=1 -tags test_dep ./tools/umpire/...      # golden harness, layout test, Go tests naming Scala positions
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

Regenerate with the gate's `--update`, then `make umpire-gen-cases umpire-gen-fixtures canary-gen-case`. Before and after: the reader's tables, Definition IDs, IR type names, Query answers, lint findings and Contracts are identical. Diffs in `model/ir/**` and `model/cases/**` contain only R5's recorded deltas. Each R4 refusal fixture is refused at its line.
