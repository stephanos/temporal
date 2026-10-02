# Make the standalone activity Scala Model a showcase of the Umpire DSL

## Goal & Context
<!-- scope: business -->

A Temporal feature developer reads `model/scalav2/scala/temporal/standaloneactivity` to learn how a Model is written. Today the four files (2,247 lines) carry the right behavior and the wrong presentation. A review on 2026-10-01 found six machines that exist only as copies, eight composition scenarios keyed by hand-written strings, 38 lines of identity evidence mapping, about 60 declarations that write their own name twice, and a 412-line realization made of nested IR constructors.

This spec makes the user-facing Temporal Model read as the best Scala the DSL allows. Everything general-purpose moves behind the `umpire` framework and the lifter. Everything Temporal-specific and shared between features moves into one shared Temporal kit. The Model's behavior does not change.

The reader this spec serves is the feature developer who authors and reviews Models. Framework developers pay the cost in `scala/umpire` and `lifter/Lift.scala`.

## Architecture & Data Models
<!-- scope: technical -->

Three layers, with one rule for what lives where.

| Layer | Path | Holds |
| --- | --- | --- |
| Framework | the `umpire` DSL and the lifter | General-purpose declarations, helpers and their lifting. Names nothing of Temporal. |
| Temporal kit | `temporal/realize` (new) | Roles, environment bindings, the correlation window and the script helpers that the activity and Nexus realizations both use. |
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
  dispatchqueue/      Model.scala, Properties.scala, Queries.scala
  compositions/       Model.scala, Properties.scala, Queries.scala
```

| File | Holds | Why there |
| --- | --- | --- |
| `Model.scala` | Domains, actions, step functions, machines, compositions, monitors | A machine names its monitors, so a monitor in `Properties.scala` would make the two files initialize each other in a cycle |
| `Properties.scala` | Properties and progress claims | They are what the machine promises: the specification a reviewer reads on its own |
| `Queries.scala` | Scenarios, Queries, Limits | A Scenario has no meaning except as the path of a Query, so the two stay together |

`admission/` holds the record, its monitors and the two admission designs. `dispatchqueue/` holds the queue interface, the detailed provider and the violating providers. `compositions/` holds the designs over the opaque and the detailed queue. The protocol machine's competing-deadline claims, now in `System.scala`, move to the top-level `Properties.scala` and `Queries.scala`. No file is named `Claims.scala`: "claim" already means an Abstraction Claim and a progress claim in this project.

## API Contracts
<!-- scope: technical -->

The shapes below are the author surface this spec adds. Names are the contract. A task may change a name only by recording the reason in its done summary.

**Machine derivation**, beside the existing `restrict`:

```scala
val staleAdmission = currentAdmission.rebind(attemptStart ~> admitStale)
val forgetfulQueue = matchingQueue.rebind(crash ~> forgetfulCrash)
val lossyMatchingQueue = matchingQueue
  .extend(storageLoss ~> storageLossDetail)
  .refining(dispatchQueueUnderStorageLoss)
  .assuming(storageLossAssumed)
val currentRecord = currentAdmission.unmonitored
```

`rebind` replaces the step function of an action the source binds and fails when the source does not bind it. `extend` adds bindings for actions the source does not bind and fails when it does. `unmonitored` drops monitors and the refinement, which is what `currentRecord` and `staleRecord` are today. A derived machine owns its name, taken from its `val`, and its Definition IDs, as a restricted one does.

**Composition without strings:**

```scala
val currentOverQueue = compose[OverQueue](
  _.activity -> currentRecord,
  _.queue -> dispatchQueue
)
  .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
  .ends(s => admissionEnds(s.activity))

val staleDeliveryAfterPause = c.scenario.actions(
  c.synced(dispatch), c.own(_.activity, control(Control.pause)), c.synced(attemptStart))

val failedCommitKeepsTheMessage = c.property holds (after =>
  !after.records(_.activity, AdmissionFact.admissionCommitFailed) || ...)
```

A sync states its name once, where it is declared, and a Scenario refers to it by a member action it pairs (`c.synced(dispatch)`), never by retyping the name. A composition derives from another by replacing one member (`val staleOverQueue = currentOverQueue.withMember(_.activity -> staleRecord)`), so the three `sync` lines are written once per composed state type.

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

**Named inputs and bounded counters:**

```scala
start(scheduleToStart = expires)          // inputs by name, each defaulting to the domain's first value
final case class ProtocolState(phase: Phase, attempts: UpTo[2], …) derives Finite
```

**Realization script helpers**, general ones in `umpire.realize`, Temporal ones in `temporal.realize`:

```scala
val stopWorkerUntilReleased = command(fault(taskQueue, FaultKind.workerStop))

val controller = script(
  perform(workerStop -> fault(taskQueue, FaultKind.workerStop)),
  onPath(control(Control.pause))(stopWorkerUntilReleased),
  perform(start(scheduleToStart = expires) -> startActivity(scheduleToStart = deadline)),
  onPath(control(Control.pause))(
    awaitStatus(ProtocolFact.statusPaused, ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED)),
  …
)
```

`perform`, `onPath` and `always` replace the three modes `Item` encodes in optional fields. Evidence kinds name facts by the fact value, never by a retyped string. A script, a command and every other realization declaration is a `val` that others refer to by value, and Temporal API messages, methods, fields and enum values are the typed ones `fn-117-type-the-temporal-api-in-the-models` provides.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Behavior is frozen.** For every IR file, the Go reader must derive the same tables, Definition IDs, refinement rows, fingerprints and Query answers before and after every task, and the lowering must produce the same Case bytes. The golden set of fn-115 R2 is what proves it. The checked-in IR files may differ only in source positions and in the lifter-internal names of functions that moved into objects or packages. The first task records the exact allowed-difference list and a check that a diff of the IR text stays inside it.
- **Name capture must not change a name.** A captured name equals the string the declaration wrote before. The six Queries whose names are computed (`s"${m.name}.any.terminalStays"`) keep their spelling.
- **The lifter refuses what it cannot read, at its line.** Each new construct gets a lifter fixture under `lifter/testdata` for the form it lifts and one for the misuse it refuses. Misuses to refuse are `rebind` of an unbound action, `extend` of a bound one, a composition member selector that names no field, and a named input that is no input of the action.
- **Each construct is built once, in the lifter.** `fn-113-clean-up-the-scala-model-layer-around` retires the native Scala evaluator (its R14 and R15), so a new construct is a typed declaration in `umpire` plus its lifting, with no runtime implementation. This spec depends on fn-113 and starts after it closes.
- **A machine is still found by its `val`.** The lifter resolves machines by their `val` definition. Derivation ops are lifted from the `val` that calls them. Machines built inside a `def` with parameters stay unsupported.
- **`UpTo[N]` replaces a lifter rule.** Today every `Int` field of one record shares the range of the one `given Finite[Int]`. The bound moves to the field's type. `Active` in the admission record may then become a counter, provided state keys and IDs stay equal. If they cannot, `Active` stays an enum and the done summary says why.
- **Comments.** A comment that explains a rule moves with the code it describes. A comment whose code is deleted (the second and third copy of a machine) is deleted with it. References to Lean and Stainless are already gone when this spec starts (fn-115 R25, fn-113 R19), and no task brings one back.
- **Nexus Models keep compiling and lifting.** `temporal/nexuscaller` adopts the shared kit and whatever framework defaults change under it. Its IR obeys the same frozen-behavior rule.
- **Sequencing.** fn-107 is closed when this spec starts, since fn-115 waits for it and this spec comes after fn-115, fn-113 and fn-117.
- **Gates.** Each task runs the scoped parts of the model gate, `make lint-scala` and the Go tests of the Umpire tooling. The closing task runs all three in full, `make lint-code-fast`, and the Quint and P export checks where their tools are installed.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The golden set of fn-115 R2 passes before and after every task, and a check of the IR text fails on any difference outside the recorded allowed-difference list.
- **R2:** `umpire` provides `rebind`, `extend` and `unmonitored`, the lifter lifts them, and `System.scala`'s successors declare each of the admission designs, the queue providers and the record members once. No two machine declarations in the feature Model share a `steps(...)` list.
- **R3:** A composition names members and synchronized actions by field selector, and a composition derives from another by replacing a member. The feature Model contains no `actionKeys` call, no `whenAction` string, no fact or action key written as a string literal, and one set of `sync` lines per composed state type.
- **R4:** `notAdmittedWhilePaused`, `atMostOneActive` and `terminalStays` are each written once and declared on the machine and on both composition families from that one definition.
- **R5:** `umpire` provides `accept`, `disabled`, `stay`, `in` and `implies`, the lifter lifts them, and the feature Model has no private single-step helper (`productStep`, `moves`, `pauses`, `timesOut`, `views`, `behind`) and one idiom for phase-set membership.
- **R6:** `evidence` defaults a fact to evidence of its own name. The feature Model's evidence blocks list only `statusTimedOut(_)` and `attemptCount`.
- **R7:** A machine, derived machine, composition, Property, Scenario, Query, Limits, timer, action, monitor, assumption, hole and channel takes its name from its `val`. The family is a `given`. A machine declaration states its three types once. A scenario omits its start when it is the machine's.
- **R8:** A Query reads a refined machine's Property through the refinement its machine declares. `Reads.through` is gone from the feature Model.
- **R9:** Action inputs are passed by name with defaults, and `ProtocolState.attempts` is a bounded counter type with no hand-written `given Finite[ProtocolState]`.
- **R10:** Each machine's vocabulary lives in an object of its own (`Product`, `Protocol`, `Admission`, `Queue` or the names the task settles), no step function carries a `protocol` or `admission` prefix, and no two package-level declarations of the feature Model differ only by inflection (`completed`, `completes`, `completion`).
- **R11:** The feature Model has the layout above. `System.scala` and `Claims.scala` are gone. Each of the top level, `admission/`, `dispatchqueue/` and `compositions/` has a `Model.scala`, a `Properties.scala` and a `Queries.scala`, and each kind of declaration is in the file the table names. Errors: a folder with no Property or no Query omits that file and never keeps an empty one; if making a folder a package of its own would change a Definition ID or an IR name outside the allowed-difference list, its files declare the parent package and the done summary says so.
- **R12:** `temporal/realize` holds the roles, bindings and correlation window, each written once and documented, and both the activity and the Nexus realization use it.
- **R13:** `Realization.scala` is written with `script`, `perform`, `onPath` and `always`, contains no `Item(` constructor, names each fact by its value, refers to its own scripts, commands, evidence kinds, controls and learned values by value, and needs no `Control as _` import.
- **R14:** The dead `enum Delivery` is resolved, and `worker` exports names without the `worker.worker…` stutter. The alias `val workerStop = worker.workerStop` is gone.
- **R15:** `retryCompletes` distinguishes the second attempt from a later one, or the saturation at `attemptBound` is stated on the Property as a bound of the claim.
- **R16:** The lifter has one lifting fixture and one refusal fixture for each construct R2, R3, R5, R6, R7 and R9 add, and the lifter's documentation lists the constructs under what is lifted.
- **R17:** The model gate, `make lint-scala`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task, and the feature Model is at most 1,600 lines across its files (2,247 today). No per-file size limit applies.
- **R18:** The feature Model's string literals are counted when this spec starts and when it closes. It had 514 on 2026-10-01: about 148 repeat a declaration's own name, 125 are composition member, sync and action keys, 59 are Temporal API names and paths, which fn-117 removes before this spec starts, 37 are evidence lines, and most of the rest are realization ids and references to them. After this spec a string literal is one of three things: prose a view shows (`because`, an example label), an id the IR needs as text and that no `val` name supplies, written once on its declaration, or a name a declaration states because it differs from its `val`. The target is at most 60. Errors: a literal outside those three is listed with its line and the reason it stays; the counts in this criterion come from a pattern match over the sources and the task records the exact method it used.

## Boundaries
<!-- scope: business -->

- No change to what the Model says. No new Property, Scenario, machine, fault or assumption.
- No IR schema change unless a task proves one is required, and then only by amending this spec.
- No polish of `temporal/nexuscaller` beyond R12 and what the framework changes force.
- No change to what the Go reader, the lowering or the exports compute. They change only if the lifter emits a construct they already define differently.
- No new library for the Models in this spec; they already have the typed Temporal API from `fn-117-type-the-temporal-api-in-the-models`, which this spec's realization helpers are written against. A library in `scala/umpire` or the lifter follows fn-113's R25. Name capture needs neither a library nor a macro: the lifter reads the name from the `val`'s symbol.
- Defining whether a member's monitors watch a composition is out of scope. `SEMANTICS.md` leaves it undefined and `goir` refuses a Query over such a composition, so `unmonitored` names the workaround once instead.
- The archives fn-115 creates are untouched.

## Decision Context
<!-- scope: both -->

- **Derivation ops over machine factories.** A `def admission(name, admit)` reads as well, and the lifter cannot find a machine that no `val` declares. `rebind` follows the existing `restrict`, keeps the `val` rule and lifts as a copy with one binding replaced.
- **Semantic freeze over byte freeze of the IR.** Moving a step function into an object may change the function name the lifter writes. Tables, IDs, fingerprints, answers and Case bytes are what consumers read, so those are frozen.
- **Objects per machine over prefixes.** Prefixes are how the file got `protocolAttemptStartStep` beside `startStep`. Objects let both machines say `attemptStart`.
- **A shared Temporal kit over a general one.** Role ids such as `temporal.workflow-service` are Temporal's. `umpire` stays free of them.
- **Order of work.** R1 first. R3's use of `own` and `synced` needs no framework change and comes next. R2, R5, R6 and R7 are independent framework additions. R10 and R11 move code and follow them, and R18 is measured at the closing task. R12 and R13 are last. Against the neighbouring specs: this spec depends on fn-113 and on `fn-117-type-the-temporal-api-in-the-models` and starts after both close, and `fn-114-state-every-scala-model-declaration-once` rolls this spec's constructs out to the other Models afterwards.
- **Amended 2026-10-01 for fn-113.** The owner retired Lean as a reference and made the IR the only thing the Scala layer answers to. Three rules of the first version went with that: the native framework had to agree with the lifted IR, no library was allowed, and Lean provenance comments were frozen.
- **Files by kind, folders by subject.** The first version kept `Claims.scala` and replaced `System.scala` with three flat files that each mixed machines with their claims. The owner asked for Properties and Queries in files of their own. Splitting by kind alone would leave three unrelated subjects in each file, so the system contract is split by subject first and by kind within it.

## Parked unknowns

- Whether `.results("Delivery")` becomes a typed `results[Outcome]` or `Delivery` stays as the declared result domain. It depends on whether anything the goldens record reads the action's result name.
- Whether named action inputs lift from Scala named arguments directly or need a generated `apply` per action. The lifter already substitutes out-of-order named arguments.
- Whether moving declarations into objects changes any Definition ID. R1's first run answers it.
- How a Scenario refers to a sync when one member action is paired by more than one sync. `c.synced(action)` is the proposal; the task that builds it settles the form, and R3 holds either way.
- Whether an action input keeps a name written at its declaration (`.input[Timeout]("scheduleToStart")`, eight such literals today) or takes it from a named field. R18 counts them as ids written once unless a task finds a form without the literal.
