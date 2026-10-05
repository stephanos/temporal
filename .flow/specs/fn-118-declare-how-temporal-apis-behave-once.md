# Declare how Temporal APIs behave once, and let the generated tests use it

## Goal & Context
<!-- scope: business -->

A generated test has to know more about a Temporal API than its request and response types. Some effects are visible to a read only after a delay. Some calls are safe to repeat and some are not. Some reads block until something happens. Some errors mean "not yet" and some mean "never".

Today that knowledge is scattered through each realization as hand-written waiting. On 2026-10-01 the two realizations held three `Poll` commands with a literal interval of 250 ms, two commands with a literal timeout of 5,000 ms, and comments that explain when a read can be trusted ("a pause is read back only of an activity no worker has taken"). An author who writes the next realization has to rediscover each of these facts, and one who forgets a wait gets a test that fails now and then.

This spec makes those facts declarations. An API's behavior is stated once, beside the typed API, as metadata. The lowering reads it and gives each generated Case the waiting, retrying and bounds it needs, and no more than it needs. The Go framework that runs a Case waits by condition with a declared bound, the way `require.Eventually` does, and never by sleeping.

It serves the feature developer, who writes what a run does and no longer how long to wait, and whoever reads a failing test, who is told which declared bound was exceeded.

## Architecture & Data Models
<!-- scope: technical -->

**A hint is a fact about an API, declared once.** It is attached to the typed method or message `fn-117-type-the-temporal-api-in-the-models` provides, and lives in the shared Temporal kit that fn-112 creates. A Model does not repeat it. The inventory first settles the helper interface fn-112 task 9 must leave available; it does not yet change a realization or a Case.

**A relationship is a hint between two APIs.** The most important one is visibility: the effect of a write becomes visible to a read, at once or eventually. "A pause is visible to DescribeActivityExecution at once" is a relationship between two methods (the code indicates at once: both go through the same server component; the inventory confirms it). An eventual relationship is what tells the lowering that a read after that write has to wait for a condition.

**A cause is a hint about what a read waits for.** Most hand-written waits today are not visibility waits. A read waits for an asynchronous cause on the path: another party's step (a worker's answer, a workflow command, a Nexus handler's reply), a server timer such as a deadline the realization sets, a server retry, or a workflow task. The lowering finds the cause in path order across scripts, and a declared wait bound for that kind of cause (for a timer, the realization's deadline plus a declared slack) bounds the wait. An instruction timeout on a performed command takes a declared bound for that kind of command, or keeps its explicit form with a reason (R4).

**Hints travel through the IR.** The lowering is Go and reads only the IR, so a hint that changes a generated Case has to be in the IR. This spec adds the fields for the hints it adopts. That is an IR schema change, and it is in scope.

**The lowering applies them.** Where a path reads after a write whose effect is eventually visible, or after an asynchronous cause, the generated Case polls for the condition within the declared bound. Where the effect is visible at once, it reads once. Where a call is declared safe to repeat, a transient error is retried within its bound. The realization author writes the read and the condition; the wait follows from the declarations.

**First hints, and candidates.** The first task inventories what the existing realizations and the Testpilot Driver already assume. The table is the starting point. A hint is adopted only when a Case that exists today needs it.

| Hint | What it says | What the lowering does with it | Status |
| --- | --- | --- | --- |
| Visibility of a write to a read | at once, or eventually within a bound | read once, or poll for the condition | adopt: governs reads after writes; together with wait bounds it replaces the three hand-written polls |
| Wait bounds | how long a condition may take, per kind of asynchronous cause | sets the Case's instruction limits | adopt per cause kind; per kind of performed command not adopted, since the two 5,000 ms limits bound nothing (task .1) |
| Not-yet errors | which error of a read means the effect is not visible yet | keeps polling instead of failing | not adopted: no Case tolerates an error (task .1) |
| Repeatable call | the call is idempotent, by which key | retries a transient failure | candidate |
| Blocking read | the call returns when something happens | waits on it once instead of polling | candidate |
| Read-only call | the call changes nothing | may be issued again or alongside others | candidate |
| Cost of a call | the call is expensive | polls it less often | candidate |

**Settled by task .1 (2026-10-03).** The inventory, the write->read pairs, the before-numbers and the exact fields are in `.plans/API_BEHAVIOR_HINTS.md`; task .1 confirmed the early proof point. In short:

- **Visibility is declared method to method.** Every write on an existing path commits its whole effect on the execution in one transaction and every read reads that execution's mutable state, so no pair splits by field. Whether a status reads `PAUSED` depends on the state the path left the activity in (a pause of a started activity is `PAUSE_REQUESTED`, reported `RUNNING`), which the realization controls by stopping the worker, not on visibility. Pause, Unpause, Terminate, Start and an activity worker's answer are visible to DescribeActivityExecution at once; StartWorkflowExecution and a workflow task to GetWorkflowExecutionHistory at once; a Nexus handler's reply to DescribeWorkflowExecution only eventually (matching returns before history records the attempt).
- **A bound belongs with the hint; a Profile only scales it** by one factor the Run records. Bounds are per cause because the server facts differ per cause; how much slower an environment is applies to all alike.
- **The Testpilot IR gets a field for the hint's position.** `CaseProvenance.sources` cannot carry it: Testpilot reads none of the provenance while the expiry message is part of the Run, the rows have no key to an instruction, and the Producer writes them at line 1. A wait node lists the hints its bound comes from, each with its id, source line and share of the bound.
- **Writes and reads are told apart by what exists today.** A read is a `Poll`, or an RPC with a response read whose method the API binds to HTTP `GET`; a write is an RPC bound to `POST` or a performed non-RPC command, identified by its cause kind; Driver controls and awaits are neither, and a `GET` whose response the Case does not read is no read. This uses the API's own `google.api.http` binding instead of the read-only candidate.
- **Causes.** The adopted cause kinds are an activity worker's answer, a workflow task, a Nexus handler's reply, the server's delivery of a task to a worker, and a server timer. A step no command performs is declared a delivery or a timer per realization; a timer step carries the deadline the realization set, and the timer's bound is the slack after it. A read waits for the step that records the fact its evidence kind confirms, found in path order across scripts; its bound is the sum of the causes since its script last synchronized, plus the visibility bound when the performing write is only eventually visible. A closing read checks no pair.
- **Not adopted:** not-yet errors (no Case tolerates an error, so no refusal can be tested), a per-command instruction timeout (the two 5,000 ms limits are never read by the Driver), a retry cause, and the remaining candidates. The Driver's own awaits (`AwaitLearned`, `AwaitCommand`, control waits) and `await-close` keep the Profile default and their explicit form under R4's errors clause.
- **Unexplained, for the owner:** the two 5,000 ms limits; the Nexus schedule-to-close timer that the Driver derives from the Profile's default instruction timeout; `wait_new_event` on the closing history read; and the retry Case's attempt poll, which rests on the retry backoff outlasting its interval and on the default (HSM) attempt counting.

## API Contracts
<!-- scope: technical -->

The author surface task .1 settled; task .2 builds it and may adjust spelling, recording it here.

```scala
// model/temporal/realize/Behavior.scala, once, each with its server citation
WorkflowServiceGrpc.METHOD_PAUSE_ACTIVITY_EXECUTION
  .visibleTo(WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce)
CauseKind.handlerReply
  .visibleTo(WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION, Visible.eventually(WaitBound(250, 2000)))
CauseKind.activityAnswer.boundedBy(WaitBound(intervalMs = 250, atMostMs = 2000))

// a realization: the steps no command performs
ServerStep(attemptStart, CauseKind.delivery)
ServerStep(scheduleToStart, CauseKind.timer, deadlineMs = ...) // from the kit's deadline value

// a read: no interval, no timeout, no explicit poll
onPath(control(Control.pause))(awaitStatus(ProtocolFact.statusPaused, ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED))
```

**IR.** The Umpire IR's `Realization` gains `ApiBehavior behavior = 15` (repeated `Visibility {id, position, write: method | cause, read, eventually: WaitBound}` and repeated `CauseBound {id, position, kind, bound}`) and `repeated ServerStep server_steps = 16` (`{position, step, kind, deadline_ms}`), with `enum CauseKind {ACTIVITY_ANSWER, WORKFLOW_TASK, HANDLER_REPLY, DELIVERY, TIMER}` and `WaitBound {position, interval_ms, at_most_ms}`. The Testpilot IR's `ReadEvidence` gains `bool once = 6` (read once, interval 0), and `InstructionNode` gains `repeated WaitHint wait_hints = 6` (`{hint_id, SourceLocation source, at_most_milliseconds}`); a node with wait hints writes its own timeout, equal to their sum, and no Profile default applies to it. Default-empty fields leave existing IR and Case bytes unchanged.

**Built by task .2 (2026-10-04).** The IR fields are `Realization.behavior = 16` and `server_steps = 17` (fn-122.4 took 15). The framework gains only the open traits `Behavior` and `SystemStep`; the hint vocabulary and `temporalBehavior` are the kit's (`model/temporal/realize/Realize.scala`, `Behavior.scala`). Ids are derived by the lifter (`visibility.<write>.<read>`, `cause.<kind>`). The POST/GET check moves to task .4, which holds the descriptors. Details in `.plans/API_BEHAVIOR_HINTS.md`, "As built by task 2".

**Built by task .4 (2026-10-05).** A `Poll` with no interval derives its wait (`tools/umpire/lower/waits.go`); one that writes an interval keeps it, so Case bytes are unchanged until task .5 clears the kit's interval. A timer's wait names `deadline.<class>` (the server step) and `cause.timer` (the slack). Calls that read are checked only in a realization that declares a behavior. The recorded reason for an explicit poll stays task .5's. Details in `.plans/API_BEHAVIOR_HINTS.md`, "As built by task 4".

**Shared-kit seam for fn-112.9.** A read is written as a typed evidence read and a typed condition (`await(evidence, role)(assign, until)`); no call site writes an interval, a timeout or `Instruction.poll`, and until task .5 the kit passes its one interval value. The script helpers take no timeout. The deadlines realizations set are kit values. Every Temporal realization is built by one kit function, where task .2 attaches `behavior`.

**Reporting.** When a wait runs out, the failure names the condition, the declared bound and the hint it came from, with the hint's Scala position.

## Edge Cases & Constraints
<!-- scope: technical -->

- **A hint never weakens a check.** Waiting for a condition that then holds is sound. A hint must not turn a wrong result into a pass: a condition that never holds within its bound fails, and evidence that is absent stays inconclusive in the Contract as it is today. No hint lets a Case accept a stale value as final.
- **A hint is a claim about the system.** A wrong hint hides a bug or wastes time. Each adopted hint cites what it rests on in the server (the code path or documented behavior), in a comment beside its declaration.
- **Bounds are declared, never guessed in Go.** The Go framework holds no default interval or timeout of its own for a wait a hint covers. A Profile may scale bounds for a slow environment, which is where a canary differs from a local test.
- **No sleeping.** A wait is a condition with a bound. The repository's linter already forbids `time.Sleep` in tests.
- **Efficiency.** A read after a write that is visible at once is not polled. The done summary states how many polls the existing Cases issue before and after.
- **Existing Cases keep their meaning.** A Case lowered after this spec asserts the same Contract as before. Its Program may differ where a hand-written wait became a derived one, and each such difference is listed.
- **Model and reality.** A hint describes how the real system behaves between two calls. It is not part of what a machine says, and it changes no table, Property or Query answer.
- **Order.** Inventory and helper-interface settlement follow fn-117 and finish before fn-112 task 9 finalizes the shared-kit interface. Hint schema, lowering and derived waits start only after fn-112 task 10 closes its structural Case freeze and after fn-114 closes, since fn-114 freezes Case bytes through its last task. These are entry gates for the tasks when this spec is planned; its spec dependency is fn-117 alone so the inventory can run while fn-112 is open. The inventory decides the exact hint fields before any schema edit; Query.total and choice-name schema changes may already have landed and must remain compatible.
- **Schema compatibility.** Any adopted hint field regenerates the IR bindings and the linked API jar. Extend the historical descriptor/wire coverage to account for every current field while preserving historical readability; do not update a stored historical descriptor as a substitute for compatibility evidence.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A committed inventory lists every wait, poll, retry, interval and timeout in the realizations, in the lowering and in the Testpilot Temporal Driver, with the fact about the API each one rests on, and proposes the hint that would carry it. Errors: a wait whose reason nobody can state is listed as unexplained, and the owner decides whether it stays.
- **R2:** The hints marked "adopt" are declarable once on a typed API in the shared kit, lift into the IR, and are validated by the Go reader. Errors: a hint that names a method or message the descriptors do not have, a relationship between a write and a read with no bound where one is required, and a bound of zero or less are each rejected at their Scala line.
- **R3:** The lowering derives waiting from the hints. A read after an eventually visible write lowers to a bounded wait for the condition; a read after a write visible at once lowers to one read. Errors: a path that reads after a write with no declared visibility is refused with both methods named, so a missing hint is found at lowering and never as a flaky test.
- **R4:** No realization contains a literal interval or timeout, and none writes an explicit poll for a condition a hint covers. Errors: a wait no hint covers is listed with its reason and keeps its explicit form.
- **R5:** The Go framework that runs a lowered Case waits by condition within the declared bound and holds no default of its own for a wait a hint covers. Errors: a wait that runs out fails with the condition, the bound and the hint's Scala position named; a Profile that scales bounds states the factor in the Run.
- **R6:** Every Contract of an existing lowered Case is unchanged, proven by the baseline goldens (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) with the Program differences listed. Errors: a changed Contract stops the task.
- **R7:** Each adopted hint has a test that fails when the hint is removed: the affected Case is refused at lowering (R3). Each has a comment citing what it rests on in the server (no error surface beyond the test).
- **R8:** The done summary states, for the existing Cases, the number of polls and the total declared wait budget before and after, and lists the candidate hints that were not adopted with the Case that would have needed each (no error surface).

## Boundaries
<!-- scope: business -->

- No hint is adopted without an existing Case that needs it. The candidates in the table stay candidates.
- No change to what any Model says, and no change to any Contract.
- No change to how Testpilot evaluates a Contract. Hints shape a Program's waiting only.
- No tuning of server behavior and no new server API.
- Hints for APIs no Model uses are not written.
- Modeling eventual consistency inside a machine (a channel, a delayed step) is a modeling decision and is not this spec.

## Decision Context
<!-- scope: both — conditionally substructured -->

The owner raised this on 2026-10-01: some actions are eventually consistent, the tests that are generated have to account for it, and other hints of the same kind would help process actions correctly and efficiently.

**Metadata on the API, not on the realization.** How soon a pause is visible to a describe is true for every Model that pauses and describes. Declared on the API pair it is written once. Declared per realization it is rediscovered each time, which is today's state.

**Through the IR.** Go builds every Case from the IR alone. A hint kept only in Scala would have to be turned into explicit commands by Scala helpers, which hides it from the lowering and from a failure message. The cost is an IR schema change.

**Refuse a read with no declared visibility.** The alternative is a default, either "at once", which produces flaky tests, or "eventually", which makes every test slow. Refusing makes the author state the fact once.

**Start with three hints.** A general annotation system invites hints nobody uses. The three adopted ones replace code that exists today, and the rest wait for a Case that needs them.

**Plan amendment (2026-10-03, task breakdown).** The Order bullet gated the behavior phase only on fn-112 task 10, while MILESTONES places it after fn-114 ("after 4"), and fn-114 freezes Case bytes through its last task and rewrites the Nexus realization. Running both at once would break fn-114's freeze, so the Order bullet now also names fn-114's close. No requirement changed. Plan review then found that most of today's waits are for an asynchronous cause rather than visibility, that Pause->Describe is visible at once, and that the writes causing a read often live in another script with no method. Architecture now names causes as a hint kind bounded by the already-adopted wait-bound hint, the sketch shows the Pause relationship as at once, and the table rows say what each adopted hint governs. R1-R8 are unchanged; the claims they rest on are corrected. The three former Parked unknowns are scheduled work for task .1, which records each answer under Architecture or API Contracts:
- whether a visibility relationship is declared between two methods, or between a write and the field a read returns;
- whether wait bounds belong with the hint, scaled by a Profile, or with the Profile;
- whether the Testpilot IR needs a new field for the hint's position, or the Case's provenance can carry it.

## Quick commands

```bash
make umpire-check-model
go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/...
make umpire-check-live-tests
```

## Early proof point

Task fn-118-declare-how-temporal-apis-behave-once.1 validates the core approach (every existing wait and write->read pair rests on a stateable API fact that a small set of hints can carry). If it fails, re-evaluate whether the hints belong on the API or whether some waits must stay explicit in realizations before continuing with .2+.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A committed inventory lists every wait, poll, retry, interval and timeout in the realizations, in the lowering and in the Testpilot Temporal Driver, with the fact about the API each one rests on, and proposes the hint that would carry it. Errors: a wait whose reason nobody can state is listed as unexplained, and the owner decides whether it stays. | fn-118-declare-how-temporal-apis-behave-once.1 | — |
| R2 | The hints marked "adopt" are declarable once on a typed API in the shared kit, lift into the IR, and are validated by the Go reader. Errors: a hint that names a method or message the descriptors do not have, a relationship between a write and a read with no bound where one is required, and a bound of zero or less are each rejected at their Scala line. | fn-118-declare-how-temporal-apis-behave-once.2 | — |
| R3 | The lowering derives waiting from the hints. A read after an eventually visible write lowers to a bounded wait for the condition; a read after a write visible at once lowers to one read. Errors: a path that reads after a write with no declared visibility is refused with both methods named, so a missing hint is found at lowering and never as a flaky test. | fn-118-declare-how-temporal-apis-behave-once.4 | — |
| R4 | No realization contains a literal interval or timeout, and none writes an explicit poll for a condition a hint covers. Errors: a wait no hint covers is listed with its reason and keeps its explicit form. | fn-118-declare-how-temporal-apis-behave-once.5 | — |
| R5 | The Go framework that runs a lowered Case waits by condition within the declared bound and holds no default of its own for a wait a hint covers. Errors: a wait that runs out fails with the condition, the bound and the hint's Scala position named; a Profile that scales bounds states the factor in the Run. | fn-118-declare-how-temporal-apis-behave-once.3 | — |
| R6 | Every Contract of an existing lowered Case is unchanged, proven by the baseline goldens (`tools/umpire/model/testdata/migration`, `tools/umpire/lower/testdata/migration`) with the Program differences listed. Errors: a changed Contract stops the task. | fn-118-declare-how-temporal-apis-behave-once.5 | — |
| R7 | Each adopted hint has a test that fails when the hint is removed: the affected Case is refused at lowering (R3). Each has a comment citing what it rests on in the server (no error surface beyond the test). | fn-118-declare-how-temporal-apis-behave-once.2, fn-118-declare-how-temporal-apis-behave-once.4 | — |
| R8 | The done summary states, for the existing Cases, the number of polls and the total declared wait budget before and after, and lists the candidate hints that were not adopted with the Case that would have needed each (no error surface). | fn-118-declare-how-temporal-apis-behave-once.5 | — |

