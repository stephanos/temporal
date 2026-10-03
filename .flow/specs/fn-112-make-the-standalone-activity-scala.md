# Make the standalone activity Scala Model a showcase of the Umpire DSL

## Goal & Context
<!-- scope: business -->

A Temporal feature developer reads `model/temporal/standaloneactivity` to learn how a Model is written. The feature is now 2,830 lines after fn-107 added the held-delivery and response-loss machines; those models are part of this migration and of the closing line count. A review on 2026-10-01 found six machines that exist only as copies, eight composition scenarios keyed by hand-written strings, 38 lines of identity evidence mapping, about 60 declarations that write their own name twice, and a 412-line realization made of nested IR constructors.

This spec makes the user-facing Temporal Model read as the best Scala the DSL allows. Everything general-purpose moves behind the `umpire` framework and the lifter. Everything Temporal-specific and shared between features moves into one shared Temporal kit. The Model's behavior does not change.

The reader this spec serves is the feature developer who authors and reviews Models. Framework developers pay the cost in `model/umpire` and `model/lifter`.

## Architecture & Data Models
<!-- scope: technical -->

Three layers, with one rule for what lives where.

| Layer | Path | Holds |
| --- | --- | --- |
| Framework | the `umpire` DSL and the lifter | General-purpose declarations, helpers and their lifting. Names nothing of Temporal. |
| Temporal kit | `temporal/realize` and `temporal/taskqueue` (new), beside `temporal/worker` | Shared realization helpers and reusable Temporal entities, including the task queue's contract, matching providers and properties. |
| Feature Model | `temporal/standaloneactivity` | Domains, actions, step functions, machines, claims, the realization script. |

Paths are given relative to the model tree. `fn-115-make-the-scala-model-the-model-and` runs before this spec and decides where that tree and the Go reader live.

The lifter stays the only front end. Each new framework construct is a construct the lifter reads from TASTy and lowers to IR that already exists. No construct adds an IR schema field unless a task shows the existing schema cannot express it.

The file layout of the feature Model after the change. Files are split by kind, and the system contract is split by subject into folders that repeat the same file names:

```text
standaloneactivity/
  Model.scala         vocabulary, the product and protocol machines, the composition with the worker
  Properties.scala    what those machines promise
  Queries.scala       the Scenarios, Queries and Limits that ask about them
  Realization.scala   the realization, written with the script helpers
  admission/          Model.scala, Properties.scala, Queries.scala
  compositions/       Model.scala, Properties.scala, Queries.scala

temporal/taskqueue/
  Model.scala         queue entity, domains, actions, opaque contract and matching providers
  Properties.scala    queue guarantees and faulty-provider controls
  Queries.scala       provider checks, crash cuts and storage-loss scenarios
```

| File | Holds | Why there |
| --- | --- | --- |
| `Model.scala` | Domains, actions, step functions, machines, compositions, monitors | A machine names its monitors, so a monitor in `Properties.scala` would make the two files initialize each other in a cycle |
| `Properties.scala` | Properties and progress claims | They are what the machine promises: the specification a reviewer reads on its own |
| `Queries.scala` | Scenarios, Queries, Limits | A Scenario has no meaning except as the path of a Query, so the two stay together |

`admission/` holds the record, its monitors and the two admission designs. The shared `temporal/taskqueue` package holds the queue interface, detailed provider, storage-loss provider and the deliberately violating forgetful/volatile providers. It owns an Entity keyed by task queue and has no dependency on standalone activity vocabulary, admission or realizations. Feature compositions connect dispatch, admission and settlement to enqueue, deliver and acknowledge. Shared provider claims and Queries belong to the queue package; cross-entity activity claims remain with the feature. The existing bounded one-message/two-delivery abstraction is retained and documented, rather than advertised as a general multi-message queue. `compositions/` holds the designs over the opaque and detailed queue. The protocol machine's competing-deadline claims move to the top-level `Properties.scala` and `Queries.scala`. No file is named `Claims.scala`.

## API Contracts
<!-- scope: technical -->

The shapes below are the author surface this spec adds. Names are the contract. A task may change a name only by recording the reason in its done summary.

**Machine derivation**, beside the existing `restrict`:

```scala
val staleAdmission = currentAdmission.rebind(attemptStart ~> admitStale)
val forgetfulQueue = matchingQueue.rebind(crash ~> forgetfulCrash)
val lossyMatchingQueue = matchingQueue
  .extend(storageLoss ~> storageLossDetail)
  .refining(dispatchQueueUnderStorageLoss)(viewOf)
  .assuming(storageLossAssumed)
val currentRecord = currentAdmission.unmonitored
```

`rebind` replaces the step function of an action the source binds and fails when the source does not bind it. `extend` adds bindings for actions the source does not bind and fails when it does. `refining(product)(map)` replaces the source refinement while preserving its visibility projections, and `assuming` appends assumptions once in declaration order. `unmonitored` drops monitors and the refinement, which is what `currentRecord` and `staleRecord` are today. A derived machine owns its name, taken from its `val`, and its Definition IDs, as a restricted one does.

**Composition without strings:**

```scala
val currentOverQueue = compose[OverQueue](
  _.activity -> currentRecord,
  _.queue -> dispatchQueue
)
  .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
  .ends(s => admissionEnds(s.activity))

val staleDeliveryAfterPause = c.scenario.actions(
  c.synced(_.activity -> dispatch),
  c.own(_.activity, control(Control.pause)),
  c.synced(_.activity -> attemptStart))

val failedCommitKeepsTheMessage = c.property holds (after =>
  !after.records(_.activity, AdmissionFact.admissionCommitFailed) || ...)
```

A sync states its name once, where it is declared, and a Scenario refers to it by a qualified member/action pair (`c.synced(_.activity -> dispatch)`), never by retyping the name. A composition derives from another by replacing one member (`val staleOverQueue = currentOverQueue.withMember(_.activity -> staleRecord)`), so the three `sync` lines are written once per composed state type. `withMember` recomputes that member's `replaces` target from the selected machine's declared refinement; it does not copy the base member's target. This preserves both the ordinary queue providers that replace `dispatchQueue` and the lossy provider that replaces `dispatchQueueUnderStorageLoss`.

**Step helpers** in `umpire`:

```scala
accept(state, facts*)            // one step with the machine's accepted outcome
accept(state, facts*).because("…")
disabled                         // no step
stay(s)                          // one step that keeps the state and records nothing
phase.in(a, b, c)                // membership in a finite set of enum cases
a implies b
```

**Declarations name themselves and default what the machine already says:**

```scala
given Family = Family("temporal.activity.standalone")

val activityProduct = machine[Product] { … }          // name from the val, one type bundle
val completion = query find completes in completed limits depth(3)
val completed = activityProtocol.scenario.actions(start(), attemptStart, attemptResult(AttemptResult.completed))
```

A scenario starts at its machine's declared start unless it says otherwise. `evidence` is optional. A fact with no evidence line is confirmed by evidence of its own name, and `evidence { case ProtocolFact.attemptCount => … }` lists only the exceptions. A Query over a machine that declares `refines(p)` reads a Property of `p` with no `Reads.through` given.

**An authored total for each Query:**

```scala
val completion = (query find completes in completed limits depth(3)).total(48)
```

`total` is the author's explicit nonnegative integer assertion of the Query's static combination count. For a pinned Scenario it is the full Scenario-machine state catalog multiplied by the number of scheduled action slots actually within the step limit (`min(steps, schedule.length)`). For a free Scenario it is the state catalog multiplied by the machine/composition's finite action-class catalog, including all input assignments, multiplied by the step limit. A named choice's branch count is not another factor: alternatives are results of an action class, not additional scheduled classes. Thus total counts state/class/depth slots before reachability, disabled-row removal, deduplication or early success; it is a review/capacity estimate, not a prediction of visited paths or runtime executions. A zero step limit has total zero. Go recomputes this count, including through refinement and compositions, and rejects missing, negative, overflowing or mismatched totals at the Query's source with the factors and computed total. The author must supply the number; there is no auto-total author helper. A presence-bearing `optional int64 total` in IR Query is the required schema addition for this spec. Absent totals remain readable only on historical IR; every current Scala Query must author one. Totals are metadata only: they do not enter machine/Query semantic fingerprints, search choices, answers, lowerings, Case bytes or exploration identities. For an internally derived exploration candidate whose schedule changes, recompute its candidate-only total before validation; this does not author or change a source Query's assertion. Compute the candidate digest and promotion identity from the semantic IR without total, so a source total correction alone cannot change them.

Declarations moved under vocabulary objects keep their original symbol-based Definition IDs through
one explicit `DefinitionScope` per former source owner, not one string per declaration. The lifter
combines that pinned owner with the captured val name. Task 1 records the exact old owner/name map;
scope pins are required only where the ordinary captured symbol would change an ID.

**Named inputs and bounded counters:**

```scala
val scheduleToStart = input[Timeout]
val start = action("start", caller).input(scheduleToStart)
start(scheduleToStart := expires)         // omitted inputs default to their domain's first value
final case class ProtocolState(phase: Phase, attempts: UpTo[2], …) derives Finite
```

Scala cannot synthesize an `apply` parameter named after each value-level Action without generated
methods or a macro. The typed input token therefore owns the captured name and `:=` supplies it by
name. The lifter reorders supplied inputs, inserts omitted first finite values and refuses duplicate
or foreign tokens at their source.

**Realization script helpers**, general ones in `umpire.realize`, Temporal ones in `temporal.realize`:

```scala
val stopWorkerUntilReleased = command(fault(taskQueue, FaultKind.workerStop))

val controller = script(
  perform(workerStop -> fault(taskQueue, FaultKind.workerStop)),
  onPath(control(Control.pause))(stopWorkerUntilReleased),
  perform(start(scheduleToStart := expires) -> startActivity(scheduleToStart = deadline)),
  onPath(control(Control.pause))(
    awaitStatus(ProtocolFact.statusPaused, ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED)),
  …
)
```

`perform`, `onPath` and `always` replace the three modes `Item` encodes in optional fields. Evidence kinds name facts by the fact value, never by a retyped string. A script, a command and every other realization declaration is a `val` that others refer to by value, and Temporal API messages, methods, fields and enum values are the typed ones `fn-117-type-the-temporal-api-in-the-models` provides.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Behavior is frozen.** The reader derives the same transition tables, refinement rows and Query answers, and lowering produces the same Case bytes. The original-baseline harness permits only source positions, recorded moved-function symbols, the authored Query total field, the exact task-queue entity declaration/attachments required by R20, and named-choice IR names introduced by fn-120 Part A for existing alternatives. Choice names are inert metadata: they cannot change branch count/order, behavior, tables, Definition IDs, fingerprints, search identity, Query answers or Cases. Existing declaration IDs remain exact through DefinitionScope. Task-queue entity metadata may change only the fingerprints that actually include that metadata; the harness must prove those changes equal the fingerprint of the original model plus precisely the approved entity delta, never waive fingerprint comparison wholesale. All other fingerprint, IR, manifest and output differences fail. The fn-115 archives remain untouched.
- **Name capture must not change a name.** A captured name equals the string the declaration wrote before. The six Queries whose names are computed (`s"${m.name}.any.terminalStays"`) keep their spelling.
- **The lifter refuses what it cannot read, at its line.** Each new construct gets a lifter fixture under `lifter/testdata` for the form it lifts and one for the misuse it refuses. Misuses to refuse are `rebind` of an unbound action, `extend` of a bound one, duplicate assumptions, an incompatible replacement refinement, a composition member selector that names no field, and a named input that is no input of the action.
- **Each construct is built once, in the lifter.** `fn-113-clean-up-the-scala-model-layer-around` retires the native Scala evaluator (its R14 and R15), so a new construct is a typed declaration in `umpire` plus its lifting, with no runtime implementation. This spec depends on fn-113 and starts after it closes.
- **A machine is still found by its `val`.** The lifter resolves machines by their `val` definition. Derivation ops are lifted from the `val` that calls them. Machines built inside a `def` with parameters stay unsupported.
- **`UpTo[N]` replaces a lifter rule.** Today every `Int` field of one record shares the range of the one `given Finite[Int]`. The bound moves to `ProtocolState.attempts`'s type. `Active` stays its current `none/one/two` enum because replacing it would change the frozen state keys.
- **Comments.** A comment that explains a rule moves with the code it describes. A comment whose code is deleted (the second and third copy of a machine) is deleted with it. References to Lean and Stainless are already gone when this spec starts (fn-115 R25, fn-113 R19), and no task brings one back.
- **Nexus Models keep compiling and lifting.** `temporal/nexuscaller` adopts the shared kit and whatever framework defaults change under it. Its IR obeys the same frozen-behavior rule.
- **Sequencing.** fn-107 is closed when this spec starts, since fn-115 waits for it and this spec comes after fn-115, fn-113 and fn-117. Tasks 3, 4 and 5 precede task 11's Query.total schema and linked API jar regeneration. Task 11 finishes before fn-120.1 adds the named-choice schema and regenerates the same bindings and jar; fn-120.1 finishes before task 6 rewrites feature branches. The conductor checks the cross-spec completion gates below because flowctl stores task dependencies only within one spec. Fn-120's full tooling and unnamed-branch refusal close later. Fn-118 inventories and settles the hint-aware shared-kit interface alongside task 9; hint-driven waiting and Program changes start only after task 10 closes the structural Case freeze.

| Phase handoff | Completion gate | Next work |
| --- | --- | --- |
| Query.total schema | fn-112.11 done after fn-112.3, .4 and .5 | fn-120.1 may add choice names to the settled schema |
| Named-choice schema | fn-120.1 done, with linked API jar and compatibility checks | fn-112.6 may rewrite feature branches using the final syntax |
| Structural Case freeze | fn-112.10 done | fn-118 may introduce derived waiting and listed Program deltas |
- **Gates.** Each task runs the scoped parts of the model gate, `make lint-model` and the Go tests of the Umpire tooling. The closing task runs all three in full, `make lint-code-fast`, and the Quint and P export checks where their tools are installed.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The golden set of fn-115 R2 passes before and after every task, and a check of the IR text fails on any difference outside the recorded allowed-difference list, including only inert names for existing fn-120 Part A choices when that mechanism is used. Choice names leave branch order/count, behavior, tables, IDs, fingerprints, Query answers and Case bytes exact.
- **R2:** `umpire` provides `rebind`, `extend`, `refining`, `assuming` and `unmonitored`, the lifter lifts them, and `System.scala`'s successors declare each of the admission designs, the queue providers and the record members once. `refining` replaces the product/map while preserving visibility projections; `assuming` appends unique assumptions in order. No two machine declarations in the feature Model share a `steps(...)` list.
- **R3:** A composition names members and synchronized actions by field selector, and a composition derives from another by replacing a member. `c.synced(_.member -> action)` identifies the participating member and refuses zero or multiple matching syncs. The feature Model contains no `actionKeys` call, no `whenAction` string, no fact or action key written as a string literal, and one set of `sync` lines per composed state type.
- **R4:** `notAdmittedWhilePaused`, `atMostOneActive` and `terminalStays` are each written once and declared on the machine and on both composition families from that one definition.
- **R5:** `umpire` provides `accept`, `disabled`, `stay`, `in` and `implies`, the lifter lifts them, and the feature Model has no private single-step helper (`productStep`, `moves`, `pauses`, `timesOut`, `views`, `behind`) and one idiom for phase-set membership.
- **R6:** `evidence` defaults a fact to evidence of its own name. The feature Model's evidence blocks list only `statusTimedOut(_)` and `attemptCount`.
- **R7:** A machine, derived machine, composition, Property, Scenario, Query, Limits, timer, action, monitor, assumption, hole and channel takes its name from its `val`. The family is a `given`. A machine declaration states its three types once. A scenario omits its start when it is the machine's. Relocated symbol-based declarations use one `DefinitionScope` per former source owner so every existing Definition ID stays exact without per-declaration ID strings.
- **R8:** A Query reads a refined machine's Property through the refinement its machine declares. `Reads.through` is gone from the feature Model.
- **R9:** Action inputs are declared as captured typed input tokens and passed by name with `:=`; omitted inputs default to their domain's first finite value. Reordered and partial calls preserve current argument order in IR, while duplicate or foreign tokens are refused. `ProtocolState.attempts` is a bounded counter type with no hand-written `given Finite[ProtocolState]`; `Active` remains the existing enum.
- **R10:** Each machine's vocabulary lives in an object of its own (`Product`, `Protocol`, `Admission`, and shared `taskqueue` vocabulary or the names the task settles), no step function carries a `protocol` or `admission` prefix, and no two package-level declarations of the feature Model differ only by inflection (`completed`, `completes`, `completion`).
- **R11:** The feature Model and shared queue have the layout above. `System.scala` and `Claims.scala` are gone; no activity-local `dispatchqueue/` remains. Each subject has Model/Properties/Queries files with declarations in their stated locations. Empty files are omitted. DefinitionScope preserves existing symbol IDs across real package moves; keeping a parent package solely to defeat the reusable queue boundary is not acceptable.
- **R12:** `temporal/realize` holds the roles, bindings and correlation window, each written once and documented, and both the activity and the Nexus realization use it.
- **R13:** `Realization.scala` is written with `script`, `perform`, `onPath` and `always`, contains no `Item(` constructor, names each fact by its value, refers to its own scripts, commands, evidence kinds, controls and learned values by value, and needs no `Control as _` import.
- **R14:** The dead `enum Delivery` is resolved, and `worker` exports names without the `worker.worker…` stutter. The alias `val workerStop = worker.workerStop` is gone.
- **R15:** `retryCompletes` distinguishes the second attempt from a later one, or the saturation at `attemptBound` is stated on the Property as a bound of the claim.
- **R16:** The lifter has one lifting fixture and one refusal fixture for each construct R2, R3, R5, R6, R7 and R9 add, including refinement replacement, assumption append and DefinitionScope pins, and the lifter's documentation lists the constructs under what is lifted.
- **R17:** The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task, and the feature Model is at most 1,600 lines across its files (2,830 before this spec, including fn-107's race models). No per-file size limit applies.
- **R18:** The feature Model's string literals are counted when this spec starts and when it closes. The historical 2026-10-01 count was 514: about 148 repeated a declaration's own name, 125 were composition member, sync and action keys, 59 were Temporal API names and paths, and 37 were evidence lines. Task 1 records the current post-fn-107/post-fn-117 syntax-aware count and classifications. After this spec a string literal is one of three things: prose a view shows (`because`, an example label), an id the IR needs as text and that no `val` name supplies, written once on its declaration, or a name a declaration states because it differs from its `val`. The target is at most 60. Errors: a literal outside those three is listed with its line and the reason it stays; the counts in this criterion come from the same recorded syntax-aware method.
- **R19:** Every current authored Scala Query supplies `.total(number)`. Go validates the exact static combination formula above and reports mismatches with factors at the Query source. Coverage includes pinned/free, inputs, compositions, refinement, disabled/unreachable states, zero, missing, negative, overflow and mismatch. Historical IR without the optional field remains readable. Totals affect admission only and leave semantic fingerprints, Query answers, Case bytes and exploration digests/identities unchanged. A schedule-changing derived candidate recomputes its internal total before validation without rewriting authored assertions; named-choice alternatives do not multiply the total. Documentation explains how to do the arithmetic and its limits.
- **R20:** `temporal/taskqueue` owns the reusable queue Entity keyed by task queue, its opaque and storage-loss contracts, matching/lossy/forgetful/volatile providers, shared properties and provider Queries. Queue machines and queue-specific actions attach to that entity where the DSL supports such identity, with no invented per-message or activity identity. The package imports no standalone-activity vocabulary. Standalone activity consumes it through explicit composition synchronization, with no copied queue transition/property code. Shared checks and an independent consumer fixture prove reuse; transition/refinement/query behavior and Case bytes remain identical under the narrow R1 entity delta.

## Requirement coverage

| Req | Task(s) |
| --- | --- |
| R1 | fn-112.1, fn-112.10 |
| R2 | fn-112.3, fn-112.6, fn-112.7 |
| R3 | fn-112.4, fn-112.7 |
| R4 | fn-112.4, fn-112.7 |
| R5 | fn-112.3, fn-112.6, fn-112.7 |
| R6 | fn-112.2, fn-112.6 |
| R7 | fn-112.2, fn-112.6, fn-112.8 |
| R8 | fn-112.2, fn-112.6 |
| R9 | fn-112.5, fn-112.6 |
| R10 | fn-112.8 |
| R11 | fn-112.8 |
| R12 | fn-112.9 |
| R13 | fn-112.9 |
| R14 | fn-112.5, fn-112.6, fn-112.8 |
| R15 | fn-112.6, fn-112.8 |
| R16 | fn-112.2, fn-112.3, fn-112.4, fn-112.5, fn-112.9, fn-112.10 |
| R17 | fn-112.10 |
| R18 | fn-112.1, fn-112.8, fn-112.10 |
| R19 | fn-112.11, fn-112.10 |
| R20 | fn-112.12, fn-112.8, fn-112.10 |

## Boundaries
<!-- scope: business -->

- No change to transition behavior. The new task-queue entity and Query total assertion are the two owner-requested metadata additions; no new behavioral Property, Scenario, machine, fault or assumption is added to production Models.
- This spec adds only Query's optional total assertion; fn-120 Part A may independently add inert choice-name metadata before this spec closes. No auto-counting author escape hatch.
- No polish of `temporal/nexuscaller` beyond R12, authored totals and what framework changes force; no forced queue composition rollout into Nexus in this spec.
- No change to what the Go reader, the lowering or the exports compute. They change only if the lifter emits a construct they already define differently.
- No new library for the Models in this spec; they already have the typed Temporal API from `fn-117-type-the-temporal-api-in-the-models`, which this spec's realization helpers are written against. A library in `model/umpire` or the lifter follows fn-113's R25. Name capture needs neither a library nor a macro: the lifter reads the name from the `val`'s symbol.
- Defining whether a member's monitors watch a composition is out of scope. `SEMANTICS.md` leaves it undefined and `tools/umpire/model` refuses a Query over such a composition, so `unmonitored` names the workaround once instead.
- The archives fn-115 creates are untouched.

## Decision Context
<!-- scope: both -->

- **Derivation ops over machine factories.** A `def admission(name, admit)` reads as well, and the lifter cannot find a machine that no `val` declares. `rebind` follows the existing `restrict`, keeps the `val` rule and lifts as a copy with one binding replaced.
- **Semantic freeze over byte freeze of the IR.** Moving a step function into an object may change the function name the lifter writes. Tables, IDs, fingerprints, answers and Case bytes are what consumers read, so those are frozen.
- **Objects per machine over prefixes.** Prefixes are how the file got `protocolAttemptStartStep` beside `startStep`. Objects let both machines say `attemptStart`.
- **A shared Temporal kit over a general one.** Role ids such as `temporal.workflow-service` are Temporal's. `umpire` stays free of them.
- **Order of work.** R1 first. Typed `own`, `synced`, `records` and member replacement require framework and lifter work; they do not exist today. R2, R5, R6, R7 and R9 add the other framework surfaces. R10 and R11 move code only after those surfaces and their identity checks exist. R12 and R13 are last, and R18 is measured at both ends. Against neighbouring specs: this spec depends on fn-113 and fn-117 and starts after both close; fn-114 rolls the constructs out to other Models, fn-118 owns behavior hints, fn-120 owns named choices and fn-119 owns the Go SDK workflow.
- **Amended 2026-10-01 for fn-113.** The owner retired Lean as a reference and made the IR the only thing the Scala layer answers to. Three rules of the first version went with that: the native framework had to agree with the lifted IR, no library was allowed, and Lean provenance comments were frozen.
- **Files by kind, folders by subject.** The first version kept `Claims.scala` and replaced `System.scala` with three flat files that each mixed machines with their claims. The owner asked for Properties and Queries in files of their own. Splitting by kind alone would leave three unrelated subjects in each file, so the system contract is split by subject first and by kind within it.
- **Owner additions, 2026-10-03.** The owner requested an explicit author-computed Query total for capacity review and a reusable queue entity under `temporal/`. Use the bounded dispatch/matching model as the shared entity, including faulty providers as reusable controls. Do not turn this into a configurable queue framework or add new behavior. Task 11 establishes totals before feature migration; task 12 extracts the queue after typed compositions and before final feature organization. Task 1 records the precise new metadata deltas so moving code cannot hide semantic drift. Report standalone source metrics and combined standalone-plus-queue metrics, preventing extraction alone from being counted as simplification.
- Maintainability (plan review): duplication - none identified; structure - keep `model/lifter/Claims.scala::fold` as the declaration dispatcher and put new captured/default/refinement and typed composition/claim logic in focused helpers rather than growing one monolithic match.

## Settled planning decisions

- Preserve the lifted result metadata text `"Delivery"` exactly. The dead Scala enum may disappear only while that IR text, its finite catalogs and all ordinary-admission goldens remain unchanged.
- Named inputs use captured typed input tokens and `:=`, not Scala named arguments or generated per-action methods. A token takes its name from its `val`; no input-name string literal remains.
- Every equivalence check compares against the original task-1 baseline, never merely the preceding task. The lifter preserves existing Definition IDs for actions, monitors, realizations, machines and their tables. The allowed IR projection additionally includes only Query total and the exact R20 task-queue entity metadata; new semantic behavior is never projected away.
- A moved declaration whose compiler owner changes uses one generic `DefinitionScope` pin for its former source owner; captured val names supply the tail. The task-1 probe records every required scope and proves actions, monitors, assumptions, channels and realizations keep exact IDs. No per-declaration legacy-ID map is allowed.
- `c.synced(_.member -> action)` is the scenario form. It refuses a selector that names no member and an action with zero or multiple matching syncs; callers disambiguate by member rather than by a string sync name. `withMember` derives replacement metadata from the new member's refinement, including the two distinct queue replacement targets.
- `ProtocolState.attempts` becomes `UpTo[2]` only if its values and order remain `0,1,2`; `Active` stays an enum. `retryCompletes` documents saturation at `attemptBound` instead of changing behavior.
- R16 accepts a compiler refusal when invalid syntax cannot produce liftable TASTy; every lifter-reachable misuse still needs a located lifter refusal fixture.
