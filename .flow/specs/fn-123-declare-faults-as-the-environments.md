# Declare faults as the environment's actions, with budgets and durable state

> HTML render lens (local): open `.flow/artifacts/fn-123-declare-faults-as-the-environments/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

Historical context: the source paths, line ranges and "today" descriptions in this proposal retain the pre-fn-112 monolithic Model snapshot; they are historical design evidence, not current source-navigation targets. The tasks carry current `path:line` targets, verified on 2026-10-06 against the tree before the DSL batch. Current names: the matching queue is `TaskQueueSystem`, the response-loss machine `admissionResponseLoss` is `LostStartAnswer`, its step `loseAdmissionAnswer` is `effects.loseResponse`, and its choices `committed` and `commitFailed` are `committedThenLost` and `failedThenLost`. The `fault` actor is a Model object in the shared task-queue package, and machines are objects with `object rules extends Rules` sections.

## Goal & Context
<!-- scope: business -->

A Model's faults are what the environment does to a machine: a crash, a lost response, lost committed storage. Today each one is an ordinary action of a party named `fault` (`model/temporal/standaloneactivity/System.scala:391`, `:477`, `:516-517`). In the IR `fault` is only a party string, and lowering only tells `system` apart from every other party (`tools/umpire/lower/lower.go:805-813`). Nothing a tool reads says which actions are faults, how often one may happen, or what survives it.

So every Model hand-encodes those answers:

- **What a crash keeps.** `crashDetail` (`System.scala:613-622`) is written by hand: it drops the poll, sends an in-flight invocation or sync match back to history, and keeps everything durable. The forgetful and volatile providers (`:693`, `:719`) are the same function with a different mistake.
- **How often.** The response-loss machine carries a `lossAvailable: Boolean` field (`:1124-1127`) that its step functions clear and test by hand.
- **Under what assumption.** Storage loss is admitted by a named assumption (`storageLossAssumed = assume("storageLoss")`, `:476`) that each machine binding the fault must also list (`:494`, `:664`).

fn-107 asked for "a finite fault budget" and said "Crash loses ephemeral state; loss of committed storage requires a separately authored fault" (`.flow/specs/fn-107-*.md:63`, `:69`). Models meet both rules by hand. Channels show the alternative: a channel declares loss, duplication and redelivery once, and Go derives the rows (`model/umpire/Channel.scala`, `model/SEMANTICS.md:116-152`).

This spec does for faults what channels did for delivery. A fault is declared once, with its kind, its budget and whether it can be realized. A machine declares which parts of its state are durable, and Go derives the crash. It is the spec fn-120's Boundaries leave open: "A choice's name is not carried into a Case or bound to a fault in a realization here."

The reader it serves is the model author who adds a fault to an entity, and the reviewer who wants to see what a crash keeps without reading a step function.

## Architecture & Data Models
<!-- scope: technical -->

This spec builds on the author surface fn-112 leaves: the shared `temporal/shared/taskqueue` package with matching, lossy, forgetful and volatile providers, `rebind`, `extend` and `assuming`, and name capture from the `val`. Line numbers below cite today's `System.scala`. fn-112 moves that code.

**Part A. Fault declarations.** A fault is an action of the framework's `fault` party (the IR string stays `"fault"`), declared with one of three kinds:

| Kind | Rows | Admitted |
| --- | --- | --- |
| `crash` | derived from the machine's durability (Part B) | always, within its budget |
| `responseLoss` | authored step function | always, within its budget |
| `storageLoss` | authored step function | only under the assumption the binding derives, named after the fault |

The action Definition IDs of today's faults (`temporal.shared.taskqueue.fault.crash`, `.ackLoss`, `.storageLoss`) and the composition keys built from them (`queue_crash`, `queue_ackLoss`) stay exact, so the declaration attaches a kind to the existing actions and moves none of them. A fault may also say `modelOnly(because = …)` when no runtime control can realize it. The IR `Action` gains an optional `Fault` record (kind, model-only reason). A fault binding records the field that holds its budget, and a machine records its durability. These are the schema additions this spec needs. Default-empty fields leave historical IR readable.

**Part B. Durability and the derived crash.** Durability is decided per value, and there are two classes: durable and in-memory. Each in-memory value names the durable value it falls back to after a crash. A machine that binds a crash classifies every field of its state type in one of three forms:

- `durable(_.f)`: every value survives.
- `ephemeral(_.f, resetTo = v)`: every value other than `v` falls back to `v`.
- `inMemory(_.f)(a -> d, …)`: the listed values fall back to the named durable values, and the rest survive.

The queue's `custody` needs the third form. `invoked` and `reserved` are held only in memory and fall back to `history`. `nowhere`, `history` and `persisted` are durable. Go derives the crash row as one result that maps each field through its classification. The row is enabled in every state, records the outcome and facts its `crashes(…)` binding names, and carries no `because`, exactly as `crashDetail` does today. The outcome is the machine's internal outcome (`QueueOutcome.internal` for the queue, today supplied by its `Ok` given), and the lifter emits no rule function for a derived crash. `SEMANTICS.md` gains a Faults section beside Channels.

**Part C. Budgets.** A budget is a state field the author writes. `f.budgetedBy(_.b)` on a binding names a field of the machine's state type as `f`'s budget: a Boolean (`true` is available, a budget of 1) or an `Int` with catalog `0..n` (a budget of `n`). Step functions keep reading and writing it, so plain-Scala tests see it, and state keys, catalogs and `Query.total` do not change. Go checks the binding over the machine's table, taken over every state of the state type's catalog, because a Scenario may start anywhere: no row raises the field, each row of `f` lowers it by one, no other action's row lowers it, and `f` has no row from a state where it is exhausted (`false` or `0`). For a Boolean, lowering is `true` to `false` and raising is `false` to `true`. A guard or an `end` that reads the field breaks no rule. A violation is refused at the binding, naming the state and action of the first offending row. A budget of zero is a start state with the field exhausted. A fault with no budget is bounded only by the Query's step limit, as every fault is today, because a budget is a field and adding one changes tables. A composition member's budget is a field of its own state.

**Part D. Realization.** A realization performs a fault with the existing `Fault(role, FaultKind)` command (`model/temporal/features/standaloneactivity/Realization.scala:793-796`). The runtime has five kinds (`proto/internal/temporal/server/api/testpilot/v1/instruction.proto:143-159`). A fault's authored alternatives are fn-120 named choices, like any step's. A performance that names no choices performs all of them. One that names choices with `choosing(…)` leaves the others model-only for that realization. Lowering refuses a witness that takes a model-only fault or choice, at its step, with the declared reason and its Scala position. The Query's manifest standing names that reason.

**Part E. Report.** One command, `umpire-faults` beside the other Umpire commands and added to the module map's permitted executables, lists, per IR file, each machine's fault bindings with kind, budget, rows (derived, authored, or authored over a derived crash) and realization: performed by which realization, FaultKind and choices, model-only with its reason, or unperformed.

## API Contracts
<!-- scope: technical -->

A sketch, written before machines became objects with `rules` sections and before the DSL batch moved rules into `when` blocks. The first task settles spellings within fn-112's operator rules (no new symbol; `->` pairs a key with its value) against the tree the DSL batch and fn-140 leave, and records them here. Read `admissionResponseLoss` as `LostStartAnswer`, `loseAdmissionAnswer` as `effects.loseResponse`, `committed` as `committedThenLost`, and `fault(taskQueue, …)` as the instruction fn-133.3 names.

```scala
val crash = fault.crash.modelOnly(because = "no control restarts matching or history in a Run")
val ackLoss = fault.responseLoss
val storageLoss = fault.storageLoss.modelOnly(because = "no control drops committed task-queue storage")

val matchingQueue = machine[QueueDetail, QueueOutcome, QueueFact] {
  refines(dispatchQueue)(viewOf)
  …
  durability(
    inMemory(_.custody)(Custody.invoked -> Custody.history, Custody.reserved -> Custody.history),
    ephemeral(_.polled, resetTo = false),
    durable(_.delivered)
  )
  crashes(crash, QueueOutcome.internal, QueueFact.crashed)
  steps(enqueue ~> enqueueDetail, …, ackLoss ~> ackLossDetail)
}
val lossyMatchingQueue = matchingQueue.extend(storageLoss ~> storageLossDetail)
  .refining(dispatchQueueUnderStorageLoss)(viewOf)            // the storageLoss assumption is derived
val forgetfulQueue = matchingQueue.rebind(crash ~> forgetfulCrash)   // a deliberate violation

val admissionResponseLoss = machine[AdmissionResponseState, Outcome, AdmissionResponseFact] {
  ends(s => !s.lossAvailable)
  steps(dispatch ~> responseLossDispatch, ackLoss.budgetedBy(_.lossAvailable) ~> loseAdmissionAnswer)
}
```

`AdmissionResponseState` and its step functions are unchanged. The binding names the field they already read and write.

```scala
perform(ackLoss.choosing(committed) -> fault(taskQueue, FaultKind.admissionResponseLoss))
```

The report, by example. The command follows `umpire-lint`'s conventions. The IR file names below predate the current layout: the queue lifts into `model/ir/activity-standalone-record.json` and `LostStartAnswer` into `model/ir/activity-standalone-race.json`.

```text
umpire faults model/ir/taskqueue.json
  matchingQueue       crash         derived    unbudgeted  model-only: no control restarts matching …   Model.scala:41
  matchingQueue       ackLoss       authored   unbudgeted  unperformed: no realization covers it
  forgetfulQueue      crash         overrides  unbudgeted  model-only                                    Model.scala:96
  lossyMatchingQueue  storageLoss   authored   unbudgeted  model-only; assumes storageLoss
umpire faults model/ir/activity.json
  admissionResponseLoss  ackLoss   authored   budget lossAvailable  activity: ADMISSION_RESPONSE_LOSS (committed); commitFailed model-only
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **Behavior is frozen.** The baseline is the checked-in `model/ir`, `model/cases` and lift goldens (`model/irgen/testdata/lifts/expected`) of the tree at fn-123's first task, after the DSL batch and fn-140 close. The migration harness and its `owners.json` were deleted with the frozen snapshots, so each task that regenerates classifies every changed line of that diff against these recorded deltas: the new fault, durability and budget-field metadata; the removal of `crashDetail` and the crash rule function as IR functions, their rows now derived from the declaration; the `storageLossAssumed` assumption's id, which becomes the derived assumption's (its name stays `storageLoss`); the lifter fixtures `model/irgen/testdata/lifts/Declarations.scala` and `Captured.scala`, whose `crash` gains a fault declaration, and their lift goldens; source positions. Tables, rows, state keys, catalogs, Query answers, Definition IDs and Case bytes stay exact otherwise. Where a derivation cannot reproduce a row, the difference is a finding listed for the owner, never accepted silently.
- **Faulty providers stay expressible.** `forgetfulCrash` and `volatileCrash` also reset `delivered` (through `idleQueue`), but only for some custody values. No per-field or per-value classification says that. They stay authored crashes bound with `rebind` over the derived one. The IR records the binding as authored over a derived crash, and fn-120's lint reports it as `fault-overridden`, accepted with its reason in the accepted-findings file. Their counterexamples and refinement targets stay the same.
- **Every field, explicitly.** On a machine that binds a crash, a field with no classification is refused, naming the field. A classification on a machine with no crash is refused too. A product-valued field is classified as a whole or by inner path, never both.
- **Fallback values are durable.** An in-memory value that falls back to another in-memory value, a value listed twice, and a reset value outside the field's domain are refused at their line.
- **Reset is a declared value, never "the start".** A Scenario may start somewhere other than its machine's declared start (`responseLossInitial`, `System.scala:1133`), so a reset to the start would depend on the question asked.
- **One way to say each thing.** Binding a fault says it can happen to this machine. `budgetedBy` names the field that says how often, and a budget of zero is a start state with that field exhausted. A storage-loss binding derives its assumption. To ask a question without a fault, restrict the machine, as today. A hand-written assumption that duplicates a derived one is refused. Because the derived assumption is appended after authored ones, `assumes(queueOpaque, storageLossAssumed)` keeps its IR order.
- **One action, two faults.** `ackLoss` is the poll answer lost in the queue (`System.scala:607-612`) and admission's response lost in the response-loss machine (`:1145-1170`). Both are response losses, so one declaration serves. Realizability is reported per binding, because only the second has a runtime control.
- **A budget is an ordinary field.** Search identity, exports and witnesses already carry it. A crash may not change it, so on a machine with a crash it is classified `durable`. Traces and the explorer (fn-120 Part C) mark it as its fault's budget and, for a derived crash, show each field that changed and the rule that changed it.
- **Exports.** Quint writes the derived crash as an action that applies the field map, or reports `UnsupportedError` with the declaration's position. Umpire has no P exporter, so no P work is planned. A machine the Quint exporter already refuses for another construct (channels, for one) keeps that refusal.
- **Plain Scala keeps the crash.** Step functions and their munit tests execute the Scala rule book, so the `crashes(…)` binding also gives the Scala table its crash row from the same classification. The lifter emits nothing for that row, and Go derives it from the IR.
- **One derived assumption per fault.** `TaskQueueProductUnderStorageLoss` and `LossyMatchingQueue` both bind `storageLoss`. Both derive the same assumption, named after the fault, so the refinement between them still matches one assumption id.
- **No Testpilot runtime change.** No Query lowered today takes a crash or a storage loss, so no new FaultKind is needed. A server-restart control for the matching crash is the first one a Query would need.
- **Order.** fn-112, fn-114, fn-120 and fn-122 are closed, so their gates are met. fn-120.4's explorer was removed, so R7 covers trace output alone. fn-123 starts after the DSL batch (fn-133, fn-134, fn-135, fn-136, fn-137, fn-139) and fn-140 close, and is written against the names they leave. The batch forbids any regeneration before its own and may leave the tree red between tasks, so no fn-123 task can run inside it. fn-139.7 moves every queue and `LostStartAnswer` rule into `when` blocks, fn-139.5 moves `LostStartAnswer` onto the shared `Outcome`, fn-133.3 lower-cases the fault instruction and renames `loseAdmissionResponse`, and fn-133.6 types `perform`. fn-140 replaces `.expect` with `.live`, requires a reason on every non-satisfied expectation, and turns `lostStartAnswer.committed` into a witness with new Definition IDs. R12's exact Definition IDs and Case bytes are those fn-140 leaves.
- **fn-141.** fn-141 keeps the IR schema and `model/ir` byte-identical at every step and moves the lifter's marker checks (the `fault` actor test) out of `model/irgen`. fn-123 changes the schema and that test. Neither blocks the other from starting, so flowctl records no edge between them. Both are deferred, and the owner orders them when they are revived. If fn-141 lands first, fn-123 adds its fields to fn-141's exporter. If fn-123 lands first, fn-141's exporter carries them.
- **fn-131.** fn-131 (Quint IR metadata) may turn the `Action` kind into one `oneof`. Whichever lands second places the `Fault` record to match.

| Phase handoff | Completion gate | Next work |
| --- | --- | --- |
| DSL batch | fn-133, fn-134, fn-135, fn-136, fn-137, fn-139 closed after the batch regeneration | fn-123.1 may start |
| Witness Queries | fn-140 closed | fn-123.1 may start; R12's baseline is fixed |
| IR generator | fn-141 order decided by the owner | the lifter or exporter task that emits fault fields |

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `umpire` declares a fault with a kind (`crash`, `responseLoss`, `storageLoss`) and an optional `modelOnly` reason. The lifter records it on the IR `Action`. The faults of `temporal/shared/taskqueue`, of the response-loss machine and of the lifter fixture `Declarations.scala` are declared this way. Errors: an action of the `fault` party with no fault declaration, a `modelOnly` with no reason, and a second declaration of one fault are refused at their line.
- **R2:** A machine that binds a crash classifies every field as `durable`, `ephemeral(resetTo)` or `inMemory` with durable fallbacks. The lifter records the classification in the IR. Errors: an unclassified or doubly classified field, an in-memory fallback, a reset value outside the domain, and a classification on a machine with no crash are refused at their line, naming the field.
- **R3:** Go derives the crash row from the classification as `SEMANTICS.md`'s new Faults section states, and the queue's derived rows equal `crashDetail`'s. Errors: a machine that binds a crash with `crashes(…)` and no classification is an admission error at the binding.
- **R4:** `rebind` of a derived crash binds an authored step, recorded as overriding the derived crash. The forgetful and volatile providers use it, and their Queries give the same counterexamples. Errors: lint reports `fault-overridden` for each such binding, and the gate fails unless it is accepted with a reason.
- **R5:** `f.budgetedBy(_.b)` names a Boolean or `0..n` `Int` field as `f`'s budget, and the lifter records it on the binding. Go checks Part C's four budget rules over the table, and a test shows a refusal for each. Errors: a field of another type, a second budget for one binding, and a budget field not classified `durable` on a machine with a crash are refused at their line. A broken rule is refused at the binding with the offending state and action.
- **R6:** A storage-loss binding derives the assumption named after the fault, one per fault and shared by every machine that binds it, appended after the machine's authored assumptions, and `storageLossAssumed` is gone from the Models. Errors: a hand-written assumption that duplicates a derived one is refused at its line.
- **R7:** Trace output marks each budget field with its fault and, for a derived crash, shows each changed field with its classification. (The explorer this criterion also named was removed with fn-120.4.) Errors: none beyond the trace renderer's own.
- **R8:** The Quint export carries derived crashes and keeps each budget field in its state variable, and Umpire's trace and explanation output show it. The Quint agreement check (`make umpire-check-backends`, run locally) passes on every converted machine the exporter does not already refuse. Umpire has no P exporter, so the P clause this criterion carried is void. Errors: an unsupported fault construct is an `UnsupportedError` with its Scala position. Nothing is exported around it.
- **R9:** A realization may restrict a fault's performance to named choices. `LostStartAnswer`'s realization performs only `committedThenLost`. Lowering refuses a witness that takes a model-only fault or an unperformed choice, at its step, with the reason and position. The manifest standing names it. Lowered Cases and their Known Gaps are unchanged. Errors: performing a `modelOnly` fault is refused at the performance.
- **R10:** The fault report lists, per IR file, every fault binding with kind, budget, rows and realization, as in the example above, and each undeclared action performed with a `Fault` command, and has a test over the checked-in IR. Errors: an IR file with no fault bindings reports none, and a malformed file is reported by the reader as today.
- **R11:** The queue providers, the storage-loss interface and the response-loss machine are converted. The diff of `model/ir`, `model/cases` and the lift goldens against the baseline holds only the recorded deltas of Edge Cases. Errors: any other difference in tables, rows, answers, fingerprints or Cases stops the task, and a row a derivation cannot reproduce is listed for the owner in the done summary.
- **R12:** `LostStartAnswer` binds `ackLoss.budgetedBy(_.lossAvailable)`. `AdmissionResponseState`, its step functions, its `end` and `StandaloneActivityPins` are unchanged from the tree fn-140 leaves, and its tables, state keys, catalogs, Definition IDs and Case bytes stay exact. Errors: a budget rule the machine breaks is refused at the binding, and the conversion stops for the owner rather than edit a step function.
- **R13:** The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task. The model's README says how a fault is declared and budgeted and what a crash keeps. Errors: none beyond the gates.

## Early proof point

Task fn-123-declare-faults-as-the-environments.3 validates the core approach (Go derives, from a per-value durability classification, crash rows equal cell by cell to `crashDetail`'s table, including outcome, facts, the missing `because` and enabledness in every catalog state). If it fails, re-evaluate per-value classification before continuing with fn-123-declare-faults-as-the-environments.4+, and list the rows the derivation cannot reproduce for the owner.

## Boundaries
<!-- scope: business -->

- No probabilistic or timed faults, and no fault whose occurrence depends on a clock.
- No Byzantine faults, corruption or duplication by a fault. Duplication stays a channel's.
- No shared network machine and no budget shared across composition members.
- No new Testpilot runtime FaultKind. No lowered Query needs one.
- No change in production Model behavior. The recorded deltas are representation only.
- No general rely-guarantee contracts. An assumption stays a name.
- Nexus `network` transport faults (`model/temporal/features/nexuscaller/Model.scala:92`) and the worker's `workerStop` are not converted here. The report lists an action a realization performs with a `Fault` command and no fault declaration as undeclared, which covers `workerStop`.
- No durability class beyond durable and in-memory.

## Decision Context
<!-- scope: both — conditionally substructured -->

**Declarations, as channels are.** A channel states loss and redelivery once, and Go derives the rows. Faults are the same kind of knowledge about the environment. Declaring them puts the author's intent in the IR, which fn-120 argues every later tool wants: lint, the report, the exports and realization.

**Explicit classification over durable-by-default.** FizzBee makes role state durable by default and resets fields declared ephemeral (`@state(ephemeral=[…])`) to their values at the end of Init. Roles with no declaration get no crash (https://fizzbee.io/design/tutorials/fault-injection/). The forgetful and volatile providers are exactly a wrong durability assumption. A default hides that decision, and a field added later would survive crashes unnoticed. Requiring every field to be classified is a small cost on the few machines that crash.

**Per-value classes over a split field or a hand-written crash.** `custody` holds durable and in-memory values in one field. Splitting it into two fields would change the state keys and every table, which the freeze forbids. Keeping `crashDetail` hand-written leaves the problem this spec exists to solve. Per-value fallbacks state exactly what `crashDetail` does, and `ephemeral(resetTo)` is their common case. Unlike FizzBee's Init reset, the value is declared, because Umpire Scenarios start where they choose.

**Budgets are author-declared fields, per binding.** Stateright bounds crashes with a framework counter, one `max_crashes` for the whole actor system (https://docs.rs/stateright/latest/stateright/actor/struct.ActorModel.html). Umpire's step functions and their tests are plain Scala that read the state, which a hidden counter would escape, and a new state component would change Case bytes. A declared field keeps both, and Go's checks make it as reliable as a counter. Umpire's one budget today belongs to one fault of one machine, so the budget sits on the binding. A shared budget waits for a Query that needs one.

**No exclusion construct.** `restrict` already derives a machine without an action, and its name says so in every result. A Scenario-level `without` would be a second way to say it.

**Budget rules over the whole catalog.** A Scenario may start in any state, so a rule checked only from `init` could pass on a machine that a Scenario then breaks. Checking every catalog state costs one pass over a table Go already builds.

**Trimmed exports and trace.** R7 lost its explorer half and R8 its P clause on 2026-10-06, because the explorer was removed and Umpire never had a P exporter. Rebuilding either for this spec was rejected as scope the spec never needed.

**Fault arms are named choices.** `loseAdmissionAnswer` has two arms, and the in-process actuator realizes only the committed one (`System.scala:1124-1126`). Today only the Query's Property keeps a witness on that arm. Naming the arm in the performance makes that limit explicit and checkable, which is the link fn-120 left to this spec.

**Why storage loss stays authored.** fn-107 says loss of committed storage is a separately authored fault. What a loss leaves (here, the idle queue) is a design decision, not a function of durability. FizzBee lists disk failure as work in progress and leaves duplication and Byzantine faults to manual models, which matches these Boundaries.

## Parked unknowns

- Whether `ackLoss` becomes two faults (poll-answer loss and admission response loss). That would change action keys and the composition key `queue_ackLoss`, so the owner decides.
- Whether Nexus transport faults and `workerStop` become fault declarations. fn-114 owns those Models.
- Whether evidence marked `Commitment.durable` (`model/umpire/realize/Realize.scala:158-160`) must name a fact recorded by a step that changes a durable value. It would be the first cross-check between a Model's durability and a realization.
- Whether the unbudgeted crash on the matching providers should get a budget, as fn-107's "finite fault budget" asks. Adding one adds a field, changes tables and is an owner decision.

## Open Questions

Planning on 2026-10-06 ran without the owner and took the defaults below. Each one changes what a task builds, so the owner confirms or overrides it before the task starts.

- **Order against fn-141.** Default: no flowctl edge; the owner orders the two specs on revival (Edge Cases, fn-141). fn-123.1 asks before it starts if neither has started.
- **Plain-Scala crash row.** Default: the `crashes(…)` binding gives the Scala rule book the derived row too, so munit tests and Scala exploration still see a crash (fn-123.2). The alternative is an IR-only crash, which changes every Scala test that steps a crash.
- **The `fault` spelling.** Default: the Model's `fault` actor keeps its object and action IDs, and the declaration attaches a kind to each action (fn-123.1). The spec's phrase "the framework's `fault` party" is read as the IR string `"fault"`, not a framework object. fn-133.3's lower-case `fault(…)` instruction must not collide with it.
- **Budget table.** Default: the four budget rules are checked over every catalog state, and end states do not suppress rows (Part C).
- **Derived assumption id.** Default: one assumption per fault, shared by the system and the product, with the id the lifter gives a declaration named after the fault (R6). fn-123.4 checks whether that id feeds any Query Definition ID or Case fingerprint and stops for the owner if it does.
- **Closing gate for Quint.** Default: fn-123.8 runs `make umpire-check-backends` locally once and records the result. It is not a CI gate.

## Status

Deferred by the owner on 2026-10-04 as not needed for the current code deliverable (the DSL and its execution). Tasks planned on 2026-10-06 so the spec can be revived as planned. It stays deferred, and fn-123.1 starts only after the gates in Edge Cases.

## Quick commands

```bash
make umpire-check-model
make umpire-check-cases
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/...
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `umpire` declares a fault with a kind (`crash`, `responseLoss`, `storageLoss`) and an optional `modelOnly` reason. The lifter records it on the IR `Action`. The faults of `temporal/shared/taskqueue`, of the response-loss machine and of the lifter fixture `Declarations.scala` are declared this way. Errors: an action of the `fault` party with no fault declaration, a `modelOnly` with no reason, and a second declaration of one fault are refused at their line. | fn-123-declare-faults-as-the-environments.1 | — |
| R2 | A machine that binds a crash classifies every field as `durable`, `ephemeral(resetTo)` or `inMemory` with durable fallbacks. The lifter records the classification in the IR. Errors: an unclassified or doubly classified field, an in-memory fallback, a reset value outside the domain, and a classification on a machine with no crash are refused at their line, naming the field. | fn-123-declare-faults-as-the-environments.2 | — |
| R3 | Go derives the crash row from the classification as `SEMANTICS.md`'s new Faults section states, and the queue's derived rows equal `crashDetail`'s. Errors: a machine that binds a crash with `crashes(…)` and no classification is an admission error at the binding. | fn-123-declare-faults-as-the-environments.3, fn-123-declare-faults-as-the-environments.4 | — |
| R4 | `rebind` of a derived crash binds an authored step, recorded as overriding the derived crash. The forgetful and volatile providers use it, and their Queries give the same counterexamples. Errors: lint reports `fault-overridden` for each such binding, and the gate fails unless it is accepted with a reason. | fn-123-declare-faults-as-the-environments.4 | — |
| R5 | `f.budgetedBy(_.b)` names a Boolean or `0..n` `Int` field as `f`'s budget, and the lifter records it on the binding. Go checks Part C's four budget rules over the table, and a test shows a refusal for each. Errors: a field of another type, a second budget for one binding, and a budget field not classified `durable` on a machine with a crash are refused at their line. A broken rule is refused at the binding with the offending state and action. | fn-123-declare-faults-as-the-environments.5 | — |
| R6 | A storage-loss binding derives the assumption named after the fault, one per fault and shared by every machine that binds it, appended after the machine's authored assumptions, and `storageLossAssumed` is gone from the Models. Errors: a hand-written assumption that duplicates a derived one is refused at its line. | fn-123-declare-faults-as-the-environments.4 | — |
| R7 | Trace output marks each budget field with its fault and, for a derived crash, shows each changed field with its classification. (The explorer this criterion also named was removed with fn-120.4.) Errors: none beyond the trace renderer's own. | fn-123-declare-faults-as-the-environments.7 | — |
| R8 | The Quint export carries derived crashes and keeps each budget field in its state variable, and Umpire's trace and explanation output show it. The Quint agreement check (`make umpire-check-backends`, run locally) passes on every converted machine the exporter does not already refuse. Umpire has no P exporter, so the P clause this criterion carried is void. Errors: an unsupported fault construct is an `UnsupportedError` with its Scala position. Nothing is exported around it. | fn-123-declare-faults-as-the-environments.7 | — |
| R9 | A realization may restrict a fault's performance to named choices. `LostStartAnswer`'s realization performs only `committedThenLost`. Lowering refuses a witness that takes a model-only fault or an unperformed choice, at its step, with the reason and position. The manifest standing names it. Lowered Cases and their Known Gaps are unchanged. Errors: performing a `modelOnly` fault is refused at the performance. | fn-123-declare-faults-as-the-environments.6 | — |
| R10 | The fault report lists, per IR file, every fault binding with kind, budget, rows and realization, as in the example above, and each undeclared action performed with a `Fault` command, and has a test over the checked-in IR. Errors: an IR file with no fault bindings reports none, and a malformed file is reported by the reader as today. | fn-123-declare-faults-as-the-environments.8 | — |
| R11 | The queue providers, the storage-loss interface and the response-loss machine are converted. The diff of `model/ir`, `model/cases` and the lift goldens against the baseline holds only the recorded deltas of Edge Cases. Errors: any other difference in tables, rows, answers, fingerprints or Cases stops the task, and a row a derivation cannot reproduce is listed for the owner in the done summary. | fn-123-declare-faults-as-the-environments.4, fn-123-declare-faults-as-the-environments.5 | — |
| R12 | `LostStartAnswer` binds `ackLoss.budgetedBy(_.lossAvailable)`. `AdmissionResponseState`, its step functions, its `end` and `StandaloneActivityPins` are unchanged from the tree fn-140 leaves, and its tables, state keys, catalogs, Definition IDs and Case bytes stay exact. Errors: a budget rule the machine breaks is refused at the binding, and the conversion stops for the owner rather than edit a step function. | fn-123-declare-faults-as-the-environments.5 | — |
| R13 | The model gate, `make lint-model`, the Go tests of the Umpire tooling and `make lint-code-fast` pass at the closing task. The model's README says how a fault is declared and budgeted and what a crash keeps. Errors: none beyond the gates. | fn-123-declare-faults-as-the-environments.8 | — |

