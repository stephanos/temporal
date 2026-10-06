# Lean, typed realizations: what the review of the activity realization found

## Goal & Context
<!-- scope: business -->

A review on 2026-10-05 of `features/standaloneactivity/Realization.scala` (341 lines) asked whether the file is clear, lean, easy to reason about, and consistent. The answer was "partly" on each count. The same patterns appear in the two Nexus realizations (`nexuscaller/Realization.scala`, 522 lines; `nexusoperation/Realization.scala`, 107 lines).

**Not lean.** Most of the length is repetition the kit could absorb:
- The six status facts are listed three times: a `statusTable`, six `awaitX` vals, and six `status(fact)` evidence entries.
- `namespace := workerNamespace` and the operation id are assigned in every RPC (6 times in the activity, 4 in the standalone Nexus operation, 5 in the Nexus caller).
- `status` and `awaitStatus` are written nearly verbatim in both the activity and the standalone Nexus realization.
- An application failure is a 14-line protobuf literal.

**Hard to reason about.**
- The controller mixes `perform` and `onPath` items, and their order is the path's call order. That order is visible only in line order.
- `onPath(deadline.scheduleToClose, …)` binds a class that no start ever sets (the file's own comment says so).
- Which start classes are covered depends on whether a class pattern such as `caller.start(scheduleToStart := expires)` is exact or partial, and the file does not say which.
- `AttemptResult.failed(retryable)` maps to `attemptFailure(nonRetryable)`, a double negative.

**Unclear.**
- Names describe things the values are not:
  - `stopWorker` and `stopWorkerUntilReleased` are the same `Fault`;
  - `loseAdmissionResponse` is an ack-loss fault named `"release-dispatch"`.
- Command identity is invisible. A command is named after its `val`, and `withFields` keeps the original's name, so `answered(statusScheduled, startActivity)` matches every start variant without the code saying so.
- Evidence takes bare positional arguments (`delivered(fact, attempts, 1, startActivity, Taking(…, 1))`) and string ids (`answeredAs("statusScheduledAgain", …)`).
- Comments float away from the declaration they explain.

**Inconsistent.**
- Instruction constructors are mixed case: `rpc`, `command` and `await` beside `Fault`, `Command`, `Hold`, `Release` and `Finish`.
- `command(Release(…), closes = …)` sits next to `Command("release-dispatch", …)`.
- Literals have two spellings each: `ProtoValue.Text` vs `ProtoValue.text`, and `Operand.Literal` vs `Operand.text`.
- Import aliases are forced by name collisions: `deadline as requestDeadline`, `worker as process`.
- `perCase("activity")` and `script("activity", …)` reuse one string in two namespaces.

**Untyped.**
- `temporalRealization(machine: Machine[?, ?, ?], …)` takes facts typed `Fact = AnyRef`.
- A realization can name another machine's fact as evidence, or bind another feature's action, and it compiles.
- A realization object (`ActivityRealization`) is a namespace for three realizations of three machines, unlike machine objects, which are one machine each with named sections.

**A second pass over the Nexus caller realization** found more of the same, and some of it larger:
- **Protobuf literals are the biggest cost.** Every field is `ProtoField.typed(Field[T, V](_.f), ProtoValue.x(…))`, often with explicit type parameters.
  - `handlerFailure`, `handlerError`, the three replies, the schedule command and `textPayload` take about 120 of the file's 522 lines.
  - `rpc` already has a terse scope (`field(_.x) := …`); protobuf literals have none.
- **A call that reads its response falls back to the core form.** The `rpc` scope cannot read the response, so `history` is written in about 30 lines of the core form `Instruction.rpc(…)(Vector(Assignment.typed(…)), Vector(ResponseRead.typed(…)))`. Its request repeats `awaitClose`'s.
- **History evidence is repeated per kind.** Five `historyKind(…)` calls differ only in the event's attributes field, the same pattern `describedStatus` removes for Describe.
- **Deadlines are bound per class, by hand, in both realizations.** One start or schedule variant is written for each class that sets a deadline. Combined classes and schedule-to-close are silently unrealized. `onPath` then lists all three schedule classes where `onPath(caller.schedule)` would do; the standalone Nexus realization already writes the action-level form.
- **Each realization repeats what its machine or kit already knows:**
  - `operation` is the machine's entity (fn-126 decision 28 infers it);
  - `roles` follow from the scripts' calls and activations;
  - `serverSteps` restate that a deadline timer fires at `deadlineMs` and the backoff at `firstRetryBackoffMs`.
- **Two realizations that differ by one step need a local factory.** `asyncNexus` and `forgedCompletion` share everything but the control's inspect, so the file needs `realization(machine, steps*)` and a controller built by concatenating vectors. Machines have `Derived` for this; realizations have nothing.
- **More inconsistencies:**
  - evidence ids are strings here (`evidenceId("scheduled")`, `sourceId("history")`) but derived from facts in the activity;
  - a request deadline is a local `Duration` proto here but the kit's `deadline` operand in the activity;
  - `Field[HistoryEvent, Long](…)` is spelled with types here and as `Field(_.x)` elsewhere;
  - a second borrowed name, `Command("start-nexus-operation", …)`.

This spec makes realizations as lean and checkable as the machines fn-126 reshaped. It serves the realization author, who writes each fact, call and binding once, and the reader, who can tell from the code what a binding matches and which classes a realization covers. fn-126's principle applies: lighten the author's cognitive load.

## Architecture & Data Models
<!-- scope: technical -->

Paths are fn-132's (`features/activity/standalone`, `features/nexus/{workflow,standalone}`); names are fn-126's. Preserve fn-126.11's level placement: the activity realization lives at `model/temporal/features/activity/standalone/system/Realization.scala` after fn-132 moves the feature (currently `features/standaloneactivity/system/Realization.scala`). Its three existing System realizations stay there when Part C replaces the wrapper objects; do not reintroduce a root or Product realization file for them. The root path in the review context above is historical.

**Part A. Kit modules (no meaning change).**
- **A described status.** One kit module covers what the activity and standalone Nexus realizations each write by hand: a status table, the read of a describe method keyed by the operation id, the evidence for each listed fact, and an await per fact. The author writes the table and the method once. Sketch:
  ```scala
  val described = describedStatus(workflowService, METHOD_DESCRIBE_ACTIVITY_EXECUTION, Field(_.getInfo), Field(_.activityId))(
    ProtocolFact.statusPaused -> ACTIVITY_EXECUTION_STATUS_PAUSED, …)
  … onPath(caller.control(Control.pause))(described.await(ProtocolFact.statusPaused))
  … evidence = described.evidence ++ Vector(…)
  ```
  The task fixes the exact form. It must lift to today's IR.
- **A request base.** A realization declares once the fields every call on a role carries (namespace, operation id from `run`). `rpc` and `await` apply them, and a call can still assign a field explicitly.
- **Literal helpers.** `applicationFailure(type, message, retryable)` builds the `Failure` proto. `jsonPayload(text)` builds the SDK's default-encoded payload. `finish(text)` builds a text result. `duration(seconds)` gives one request-deadline value for both RPC fields and protobuf literals. One spelling per literal (the lower-case helpers) across all realizations.
- **A protobuf literal scope.** `proto[T] { field(_.x) := … }` builds a protobuf literal the way `rpc` builds a request: nested messages, enums, maps and roles, with types inferred. `Proto[T](ProtoField.typed(…))` stays as the core form the lifter reads.
- **A response read in the call scope.** Inside `rpc(…) { … }`, a line such as `read(historyEvents, each) into (historyEvent, correlated)` declares a response read, so no realization needs the core `Instruction.rpc` form. A call can extend another call's request, so `history` extends `awaitClose`'s request instead of repeating it. A call also declares what it answers per outcome (`answers(Outcome.accepted -> ok, …)`), read from the Run's `InstructionOutcome.protocol_code`. Request, response and error then sit in one place, as in a `stamp` handler, and evidence confirms the outcome, not only success.
- **History evidence.** A kit module declares a workflow's history-event evidence once, one entry per fact and its attributes field, keyed by the field that names the operation. It covers the exhaustive kinds and their closing read. It is the history counterpart of the described status.

**Part B. Clarity and reasoning.**
- **One constructor convention.** An author writes lower-case forms for every instruction (`fault`, `hold`, `release`, `finish`, `command`). The upper-case case classes stay the core forms the lifter reads.
- **No borrowed names.** No command takes another command's name. Evidence that two races share names both commands, or the kit gains an explicit alias that the evidence line shows. The task decides which.
- **Command identity shown.** `withFields`'s "keeps the call's name" is stated where it matters. Either the evidence line names the variants, or the kit docs and README say that an evidence line naming a call matches all its `withFields` variants.
- **Named evidence arguments.** `delivered` and `answeredAs` take named parameters (`attempt = 1`, `after = startActivity`). Kinds that differ from their fact are named by a fact or a declared constant, not a free string.
- **Coverage reported.** The gate or lint reports:
  - a `perform` or `onPath` binding of a class that no realizable path takes (an unreachable binding);
  - a class of a performed action that no binding covers, where a Query's path takes it.

  README states whether a class pattern is exact or partial.
- **Deadlines bound once.** A realization declares which request field each `Timeout` input sets, e.g. `deadlines(scheduleToStart -> _.getScheduleToStartTimeout, startToClose -> _.getStartToCloseTimeout)`. The kit generates the binding of every class of the start or schedule action, including combined classes. Where a class cannot be realized (the server refuses a start with no close deadline), the declaration says so and the coverage report shows it.
- **Action-level `onPath`.** A list of every class of one action is written as the action, `onPath(caller.schedule)`, as the standalone Nexus realization already does.
- **Evidence ids from facts.** An evidence kind is named after its fact or a declared constant, not a string literal (`evidenceId("scheduled")`, `sourceId("history")`).
- **Fields without spelled types.** `Field[T, V](…)` is written `Field(_.x)` wherever the scope infers it.
- **Name collisions removed.** The kit's `deadline` operand and the shared worker party's name no longer collide with a feature's `deadline` and `worker` sections, so no realization imports an alias. The task picks the renames, consistent with fn-126.8.
- **Local fixes in the activity realization:**
  - `stopWorkerUntilReleased` is named for what it is (the pause path's stop);
  - the unreachable `scheduleToClose` await goes;
  - `attemptFailure` takes `retryable`;
  - `lostAdmissionResponse` gets its own section;
  - floating comments attach to their declarations;
  - the header and `status` doc are rewritten plainly;
  - the `perCase` and script names stop sharing one string.

**Part C. Typed realization objects.**
- **One object per realization, typed by its machine:** `object Standalone extends Realization(ActivityProtocol)`. Its header holds roles and the operation, and its named sections are, in a fixed order: the controller, the worker scripts, evidence, server steps and controls.
- **Facts and classes are typed.** Evidence facts, status-table keys and `perform`/`onPath` classes are checked against the machine's fact and action types, so naming another machine's fact or action does not compile.
- **The lint reads the sections.** fn-126's structure lint (R20) checks realization objects like machine objects: the section names and order, and one realization per object.
- **Realization files hold realization objects.** No wrapper object holds several.
- **Derived from what is known.** A realization object does not restate what its machine or the kit knows:
  - the operation is its machine's entity;
  - its roles are those its scripts' calls and activations name;
  - deadline and backoff timers get the kit's default server steps (`deadlineMs`, `firstRetryBackoffMs`) unless the object overrides them.
- **Derived realizations.** A realization can be declared as another one with steps added or replaced, as a `Derived` machine is: `object ForgedCompletionRealization extends Realization(ForgedCompletion) derives asyncNexus with perform(inspection.inspect -> inspectWorkflow)`. The Nexus caller's local `realization(machine, steps*)` factory and its vector-concatenating controller go.

**Part D. Carrier schemas from typed bindings.** The realization binding associates a concrete action class with the instruction that performs it. Derive the protobuf carrier from that instruction's typed RPC descriptor, command payload, or worker/handler message, and expose the mapping per realization and class. Remove action-level `.schema[T]` lists. Auxiliary reads are evidence, not action carriers; unrealized classes stay unmapped and R6 reports required coverage. Model declarations remain independent of execution details.

## Edge Cases & Constraints
<!-- scope: technical -->

- **R13 may add realized classes.** Combined deadline classes that today have no binding become realizable, and Queries whose paths take them may gain Cases. That is the one intended meaning change in Part B: each new Case is listed, and no existing Case changes.
- **Part A and Part B change no meaning, apart from R13.** A before/after projection is identical except for the recorded deltas:
  - reader tables, Query answers, verdicts, fingerprints, Case bytes and lint findings;
  - the unreachable binding Part B deletes;
  - any IR command id the alias decision changes.
- **Part C changes IDs once.** Realization and script IDs follow the new objects' fully qualified names (fn-126 decision 23). The projection holds with an ID map applied.
- **A shared race command stays one command where the Run records one.** Renaming `loseAdmissionResponse` must not split evidence the held and lost races share. If it would, the alias of Part B carries the shared name.
- **The coverage report must accept today's realizations after Part B's fixes**, or record each finding with its decision. It reports, never silently drops, a class a Query's path takes that no binding covers.
- **The kit stays Temporal's.** `describedStatus`, the request base and `applicationFailure` live in `model/temporal/realize`. The generic `model/umpire/realize` gains only what Part C's typed object needs.
- **fn-131 Part F** moves the realization messages to their own proto file, byte-identical. If both are in flight, this spec's IR proofs use whichever schema has landed.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The activity and standalone Nexus realizations declare their described statuses with one kit module, each fact written once. No `awaitStatus`/`status` helper or per-fact await val remains in a feature file. Errors: a fact listed twice in a table is refused at its line; an await of a fact the table does not list is refused at its line.
- **R2:** Every realization declares its common request fields once, and no RPC in a feature file assigns `namespace` or the operation id unless it overrides the base. Errors: an override equal to the base is a lint finding.
- **R3:** `applicationFailure` and `finish` exist, and the three realizations use them. No feature file spells `ProtoValue.Text`, `Operand.Literal` or a `Proto[Failure]` literal.
- **R4:** Feature files write only lower-case instruction forms. No command takes another command's name. Evidence that matches `withFields` variants is documented in the kit and README. Errors: a lint finding for an upper-case instruction form in a feature file; the lifter refuses two commands of one script with one name, unless the kit's explicit alias declares it.
- **R5:** `delivered` and `answeredAs` are called with named arguments, and no evidence kind is a free string.
- **R6:** The gate or lint reports unreachable bindings and uncovered classes, with one fixture each. README states the class-pattern rule. The activity realization has no unreachable binding.
- **R7:** No realization imports an alias forced by a name collision.
- **R8:** The activity realization's local fixes (Part B list) are done.
- **R9:** Each realization is its own object typed by its machine, with named sections in a fixed order. A fact or class of another machine is a compile error (one negative compile fixture each), and the structure lint checks the sections. Errors: a realization object with a section out of order, or holding two realizations, is refused at its line.
- **R10:** Line counts of the three realizations before and after are in the done summary, with the projection proof per part.
- **R11:** No feature file builds a protobuf literal with `ProtoField.typed`, or a call with `Instruction.rpc`/`Assignment.typed`/`ResponseRead.typed`. `proto[T] { … }` and the call scope's `read` lift to the same IR as the core forms they replace. `jsonPayload` and `duration` exist and are used. Errors: assigning a field twice in one literal scope is refused at its line.
- **R12:** The Nexus caller's history evidence is declared with one kit module, each fact and its attributes field written once, and the closing read is derived from it. Errors: a history entry whose attributes field has no operation key is refused at its line.
- **R13:** Deadline-setting classes are bound by one declaration per realization, which covers combined classes. A class the declaration marks unrealizable is shown by R6's report. A list of every class of one action in `onPath` is written as the action. Errors: a declaration naming an input that is not a `Timeout` of the action is refused at its line.
- **R14:** No feature file passes a string literal to `evidenceId`/`sourceId`, or spells `Field[T, V]` where the scope infers it.
- **R15:** No realization object states its operation, its roles, or the default server steps. Each is derived and equal to today's IR. Errors: an explicit value equal to the derived one is a lint finding; a script naming a role the realization cannot derive is refused at its line.
- **R16:** `forgedCompletion` is a realization derived from `asyncNexus`, and the factory and concatenated controller are gone. Errors: a derived realization that replaces a step its base does not have is refused at its line.

- **R17:** Carrier schemas are derived per realization and action class from typed perform bindings, including non-RPC protobuf carriers. The activity's four control classes have four explicit derived request mappings. No live action-level `.schema[T]` list remains; unrealized and internal classes are distinguished without guessing. Partial patterns, derived overrides, multiple carriers, and auxiliary reads have tests. Intentional Umpire metadata deltas are recorded; existing Model and Case behavior is unchanged.

## Boundaries
<!-- scope: business -->

- No change to what an existing Case does or what evidence means; R13 may only add Cases. The activity's semantic changes are fn-128's.
- No new realizations and no new Driver instructions.
- The Go tooling changes only where the coverage report, typed realization objects, or derived carrier metadata need it.
- No change to the Testpilot IR.

## Decision Context
<!-- scope: both -->

- **Kit modules over per-file helpers.** `status`/`awaitStatus` are already duplicated across two features and will be written again by every Describe-backed feature (activity workflow form, schedules). A deep module, one call that hides the read, table, awaits and evidence, removes about 60 lines here and about 25 in the standalone Nexus realization, and it rarely changes.
- **Typed objects, not a marker.** fn-126 decision 25 removed markers whose only job was telling the lint what an object is. A `Realization(machine)` base carries behaviour: it types facts and classes against the machine, which is what catches the mistakes `Fact = AnyRef` lets through.
- **After fn-132, before fn-128.** fn-132 moves these files. fn-128 adds start delay, retry policy and rejection evidence to the activity realization and should write them against the lean, typed kit.

## Ordering
- After fn-126 closes (IDs as fully qualified names, structure lint) and after fn-132 closes (paths).
- Before fn-128 starts.
- Inside the spec: Part A, then Part B, then Part C, then Part D (task 8), then close (task 7). Each regenerates `model/ir` and is proved on its own.

## Verification

```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

Regenerate with `make umpire-gen-model`, then `make umpire-gen-cases umpire-gen-fixtures canary-gen-case`. Every proof is by projection (`projtool` and a before/after IR projection), never a golden re-capture.

## Requirement coverage

| Task | Requirements |
| --- | --- |
| 1. Literal and call scopes: `proto[T] { … }`, response reads, literal helpers | R3, R11, R14 (fields) |
| 2. Kit evidence modules: described status, history evidence, request base | R1, R2, R12 |
| 3. One instruction convention, own names, named evidence arguments, ids from facts | R4, R5, R14 (ids) |
| 4. Coverage report, class-pattern rule, deadlines bound once, action-level `onPath` | R6, R13 |
| 5. Name collisions and the activity realization's local fixes | R7, R8 |
| 6. Typed realization objects, derived operation, roles and server steps, derived realizations | R9, R15, R16 |
| 8. Derive per-class carrier schemas from typed realizations | R17 |
| 7. Close (after task 8) | R10 |

## Parked unknowns
- Whether a class pattern like `caller.start(scheduleToStart := expires)` is exact or partial today. R6's task reads the lifter and records it before writing the README rule.
- Whether the shared race name needs an explicit alias or can name both commands. Decided by R4's task.
