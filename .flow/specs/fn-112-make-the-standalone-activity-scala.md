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
| Framework | `model/scalav2/scala/umpire`, `model/scalav2/lifter` | General-purpose declarations, helpers and their lifting. Names nothing of Temporal. |
| Temporal kit | `model/scalav2/scala/temporal/realize` (new) | Roles, environment bindings, the WorkflowService method prefix, the correlation window and the script helpers that the activity and Nexus realizations both use. |
| Feature Model | `model/scalav2/scala/temporal/standaloneactivity` | Domains, actions, step functions, machines, claims, the realization script. |

The lifter stays the only front end. Each new framework construct is a construct the lifter reads from TASTy and lowers to IR that already exists. No construct adds an IR schema field unless a task shows the existing schema cannot express it.

The file layout of the feature Model after the change:

| File | Holds |
| --- | --- |
| `Model.scala` | Vocabulary, the product machine, the protocol machine, the composition with the worker |
| `Claims.scala` | Every Property, Scenario, Query and set of the product, the protocol and the worker composition, including the competing-deadline claims now in `System.scala` |
| `Admission.scala` | The record, its monitors, the two admission designs and their claims |
| `DispatchQueue.scala` | The queue interface, the detailed provider, the violating providers and their claims |
| `Compositions.scala` | The designs over the opaque and the detailed queue and their claims |
| `Realization.scala` | The realization, written with the script helpers |

## API Contracts
<!-- scope: technical -->

The shapes below are the author surface this spec adds. Names are the contract. A task may change a name only by recording the reason in its done summary.

**Machine derivation**, beside the existing `restrict`:

```scala
val staleAdmission = currentAdmission.rebind("staleAdmission")(attemptStart ~> admitStale)
val forgetfulQueue = matchingQueue.rebind("forgetfulQueue")(crash ~> forgetfulCrash)
val lossyMatchingQueue = matchingQueue
  .extend("lossyMatchingQueue")(storageLoss ~> storageLossDetail)
  .refining(dispatchQueueUnderStorageLoss)
  .assuming(storageLossAssumed)
val currentRecord = currentAdmission.unmonitored("currentRecord")
```

`rebind` replaces the step function of an action the source binds and fails when the source does not bind it. `extend` adds bindings for actions the source does not bind and fails when it does. `unmonitored` drops monitors and the refinement, which is what `currentRecord` and `staleRecord` are today. A derived machine owns its name and Definition IDs, as a restricted one does.

**Composition without strings:**

```scala
val currentOverQueue = compose[OverQueue]("currentOverQueue")(
  _.activity -> currentRecord,
  _.queue -> dispatchQueue
)
  .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
  .ends(s => admissionEnds(s.activity))

c.scenario("staleDeliveryAfterPause").actions(
  c.synced("dispatch"), c.own(_.activity, control(Control.pause)), c.synced("admit"))

c.property("failedCommitKeepsTheMessage") holds (after =>
  !after.records(_.activity, AdmissionFact.admissionCommitFailed) || ...)
```

A composition derives from another by replacing one member (`currentOverQueue.withMember("staleOverQueue")(_.activity -> staleRecord)`), so the three `sync` lines are written once per composed state type.

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
script("controller")(
  perform(workerStop -> fault(taskQueue, FaultKind.workerStop)),
  onPath(control(Control.pause))(command("stop-worker-until-released", fault(taskQueue, FaultKind.workerStop))),
  perform(start(scheduleToStart = expires) -> startActivity(scheduleToStart = deadline)),
  onPath(control(Control.pause))(awaitStatus(ProtocolFact.statusPaused, "PAUSED")),
  …
)
```

`perform`, `onPath` and `always` replace the three modes `Item` encodes in optional fields. Evidence kinds name facts by the fact value, never by a retyped string.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Behavior is frozen.** For all four IR files, `goir` must derive the same tables, Definition IDs, refinement rows, fingerprints and Query answers before and after every task, and `goir/testpilot` must lower the same Case bytes. `goir/activity_parity_test.go` keeps comparing `ir/activity.json` with `model/go/standaloneactivity` on every row and Property. The checked-in `ir/*.json` may differ only in source positions and in the lifter-internal names of functions that moved into objects. The first task records the exact allowed-difference list and a script that proves a diff stays inside it.
- **Name capture must not change a name.** A captured name equals the string the declaration wrote before. The six Queries whose names are computed (`s"${m.name}.any.terminalStays"`) keep their spelling.
- **The lifter refuses what it cannot read, at its line.** Each new construct gets a lifter fixture under `lifter/testdata` for the form it lifts and one for the misuse it refuses. Misuses to refuse are `rebind` of an unbound action, `extend` of a bound one, a composition member selector that names no field, and a named input that is no input of the action.
- **Native Scala and lifted IR agree.** The munit tests under `scala/temporal/test` and `scala/umpire/test` run the native framework. A derived machine's native table equals the table Go derives from its lifted IR.
- **A machine is still found by its `val`.** `Lift.scala:1046` resolves machines by `ValDef`. Derivation ops are lifted from the `val` that calls them. Machines built inside a `def` with parameters stay unsupported.
- **`UpTo[N]` replaces a lifter rule.** Today every `Int` field of one record shares the range of the one `given Finite[Int]`. The bound moves to the field's type. `Active` in the admission record may then become a counter, provided state keys and IDs stay equal. If they cannot, `Active` stays an enum and the done summary says why.
- **Comments are preserved.** Existing comments move with the code they describe. A comment whose code is deleted (the second and third copy of a machine) is deleted with it. No task rewrites provenance comments.
- **Nexus Models keep compiling and lifting.** `temporal/nexuscaller` adopts the shared kit and whatever framework defaults change under it. Its IR obeys the same frozen-behavior rule.
- **Sequencing with fn-107.** Tasks 10, 11 and 22 of `fn-107-scala-umpire-prototype-for-standalone` are open and edit `Realization.scala`, `System.scala` and the lifter. This spec starts after they land, or its planner coordinates file ownership with the session working them.
- **Gates.** Each task runs `model/scalav2/run.sh`, `make lint-scala`, the scoped `go test -tags test_dep ./model/scalav2/...` and `make lint-code-fast`. The closing task also runs `model/scalav2/backends/run.sh` where its tools are installed. No task installs a Lean toolchain. The Lean-dump parity tests skip without the dumps.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A baseline script records, for the four IR files, the tables, Definition IDs, fingerprints, Query answers and lowered Case bytes, and fails on any difference outside the recorded allowed-difference list. Every later task passes it.
- **R2:** `umpire` provides `rebind`, `extend` and `unmonitored`, the lifter lifts them, and `System.scala`'s successors declare each of the admission designs, the queue providers and the record members once. No two machine declarations in the feature Model share a `steps(...)` list.
- **R3:** A composition names members and synchronized actions by field selector, and a composition derives from another by replacing a member. The feature Model contains no `actionKeys` call, no `whenAction` string, no fact or action key written as a string literal, and one set of `sync` lines per composed state type.
- **R4:** `notAdmittedWhilePaused`, `atMostOneActive` and `terminalStays` are each written once and declared on the machine and on both composition families from that one definition.
- **R5:** `umpire` provides `accept`, `disabled`, `stay`, `in` and `implies`, the lifter lifts them, and the feature Model has no private single-step helper (`productStep`, `moves`, `pauses`, `timesOut`, `views`, `behind`) and one idiom for phase-set membership.
- **R6:** `evidence` defaults a fact to evidence of its own name. The feature Model's evidence blocks list only `statusTimedOut(_)` and `attemptCount`.
- **R7:** A machine, Property, Scenario, Query, Limits, timer and action takes its name from its `val`. The family is a `given`. A machine declaration states its three types once. A scenario omits its start when it is the machine's.
- **R8:** A Query reads a refined machine's Property through the refinement its machine declares. `Reads.through` is gone from the feature Model.
- **R9:** Action inputs are passed by name with defaults, and `ProtocolState.attempts` is a bounded counter type with no hand-written `given Finite[ProtocolState]`.
- **R10:** Each machine's vocabulary lives in an object of its own (`Product`, `Protocol`, `Admission`, `Queue` or the names the task settles), no step function carries a `protocol` or `admission` prefix, and no two package-level declarations of the feature Model differ only by inflection (`completed`, `completes`, `completion`).
- **R11:** `System.scala` is replaced by `Admission.scala`, `DispatchQueue.scala` and `Compositions.scala`, and the protocol's competing-deadline claims are in `Claims.scala`.
- **R12:** `temporal/realize` holds the roles, bindings, method prefix and correlation window, each written once and documented, and both the activity and the Nexus realization use it.
- **R13:** `Realization.scala` is written with `script`, `perform`, `onPath` and `always`, contains no `Item(` constructor, names each fact by its value, and needs no `Control as _` import.
- **R14:** The dead `enum Delivery` is resolved, and `worker` exports names without the `worker.worker…` stutter. The alias `val workerStop = worker.workerStop` is gone.
- **R15:** `retryCompletes` distinguishes the second attempt from a later one, or the saturation at `attemptBound` is stated on the Property as a bound of the claim.
- **R16:** The lifter has one lifting fixture and one refusal fixture for each construct R2, R3, R5, R6, R7 and R9 add, and `README.md`'s "What is lifted" lists the constructs.
- **R17:** `model/scalav2/run.sh`, `make lint-scala`, `go test -tags test_dep ./model/scalav2/...` and `make lint-code-fast` pass at the closing task, and the feature Model is at most 1,600 lines across its six files (2,247 today).

## Boundaries
<!-- scope: business -->

- No change to what the Model says. No new Property, Scenario, machine, fault or assumption.
- No IR schema change unless a task proves one is required, and then only by amending this spec.
- No polish of `temporal/nexuscaller` beyond R12 and what the framework changes force.
- No rewrite or relocation of the Lean provenance comments.
- No change to `goir`, `goir/testpilot` or `backends` semantics. They change only if the lifter emits a construct they already define differently.
- No new third-party library. Name capture is a macro written in `umpire`.
- Making the native Scala search evaluate monitors is out of scope. `unmonitored` names the workaround once instead.
- `model/scala`, `model/go` and `model/lean` are untouched.

## Decision Context
<!-- scope: both -->

- **Derivation ops over machine factories.** A `def admission(name, admit)` reads as well, and the lifter cannot find a machine that no `val` declares. `rebind` follows the existing `restrict`, keeps the `val` rule and lifts as a copy with one binding replaced.
- **Semantic freeze over byte freeze of the IR.** Moving a step function into an object may change the function name the lifter writes. Tables, IDs, fingerprints, answers and Case bytes are what consumers read, so those are frozen.
- **Objects per machine over prefixes.** Prefixes are how the file got `protocolAttemptStartStep` beside `startStep`. Objects let both machines say `attemptStart`.
- **A shared Temporal kit over a general one.** Role ids such as `temporal.workflow-service` are Temporal's. `umpire` stays free of them.
- **Order of work.** R1 first. R3's use of `own` and `synced` needs no framework change and comes next. R2, R5, R6 and R7 are independent framework additions. R10 and R11 move code and follow them. R12 and R13 are last, after fn-107's realization work lands.

## Parked unknowns

- Whether `.results("Delivery")` becomes a typed `results[Outcome]` or `Delivery` stays as the declared result domain. It depends on what `model/go/standaloneactivity` compares for the action's result name.
- Whether named action inputs lift from Scala named arguments directly or need a generated `apply` per action. The lifter already substitutes out-of-order named arguments (`Lift.scala:1057`).
- Whether moving declarations into objects changes any Definition ID. R1's first run answers it.
