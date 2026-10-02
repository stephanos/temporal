# Declare how Temporal APIs behave once, and let the generated tests use it

## Goal & Context
<!-- scope: business -->

A generated test has to know more about a Temporal API than its request and response types. Some effects are visible to a read only after a delay. Some calls are safe to repeat and some are not. Some reads block until something happens. Some errors mean "not yet" and some mean "never".

Today that knowledge is scattered through each realization as hand-written waiting. On 2026-10-01 the two realizations held three `Poll` commands with a literal interval of 250 ms, two commands with a literal timeout of 5,000 ms, and comments that explain when a read can be trusted ("a pause is read back only of an activity no worker has taken"). An author who writes the next realization has to rediscover each of these facts, and one who forgets a wait gets a test that fails now and then.

This spec makes those facts declarations. An API's behavior is stated once, beside the typed API, as metadata. The lowering reads it and gives each generated Case the waiting, retrying and bounds it needs, and no more than it needs. The Go framework that runs a Case waits by condition with a declared bound, the way `require.Eventually` does, and never by sleeping.

It serves the feature developer, who writes what a run does and no longer how long to wait, and whoever reads a failing test, who is told which declared bound was exceeded.

## Architecture & Data Models
<!-- scope: technical -->

**A hint is a fact about an API, declared once.** It is attached to the typed method or message `fn-117-type-the-temporal-api-in-the-models` provides, and lives in the shared Temporal kit that fn-112 creates. A Model does not repeat it.

**A relationship is a hint between two APIs.** The most important one is visibility: the effect of a write becomes visible to a read, at once or eventually. "A pause is visible to DescribeActivityExecution eventually" is a relationship between two methods, and it is what tells the lowering that a read after that write has to wait for a condition.

**Hints travel through the IR.** The lowering is Go and reads only the IR, so a hint that changes a generated Case has to be in the IR. This spec adds the fields for the hints it adopts. That is an IR schema change, and it is in scope.

**The lowering applies them.** Where a path reads after a write whose effect is eventually visible, the generated Case polls for the condition within the declared bound. Where the effect is visible at once, it reads once. Where a call is declared safe to repeat, a transient error is retried within its bound. The realization author writes the read and the condition; the wait follows from the declarations.

**First hints, and candidates.** The first task inventories what the existing realizations and the Testpilot Driver already assume. The table is the starting point. A hint is adopted only when a Case that exists today needs it.

| Hint | What it says | What the lowering does with it | Status |
| --- | --- | --- | --- |
| Visibility of a write to a read | at once, or eventually within a bound | read once, or poll for the condition | adopt: replaces the three hand-written polls |
| Wait bounds | how long a condition may take, per kind of wait | sets the Case's instruction limits | adopt: replaces the literal 250 and 5,000 |
| Not-yet errors | which error of a read means the effect is not visible yet | keeps polling instead of failing | adopt if the inventory finds one in use |
| Repeatable call | the call is idempotent, by which key | retries a transient failure | candidate |
| Blocking read | the call returns when something happens | waits on it once instead of polling | candidate |
| Read-only call | the call changes nothing | may be issued again or alongside others | candidate |
| Cost of a call | the call is expensive | polls it less often | candidate |

## API Contracts
<!-- scope: technical -->

A sketch of the author surface. The task that builds it settles the spelling and records it here.

```scala
// in the shared Temporal kit, once
WorkflowService.pauseActivityExecution
  .visibleTo(WorkflowService.describeActivityExecution, eventually(within = statusRead))

val statusRead = waitBound(interval = 250.millis, atMost = 10.seconds)

// in a realization: no interval, no timeout, no explicit poll
onPath(control(Control.pause))(
  expectStatus(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED))
```

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
- **Order.** This spec follows fn-117, which makes methods typed values a hint can attach to, and fn-112, which creates the shared kit and the script helpers.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A committed inventory lists every wait, poll, retry, interval and timeout in the realizations, in the lowering and in the Testpilot Temporal Driver, with the fact about the API each one rests on, and proposes the hint that would carry it. Errors: a wait whose reason nobody can state is listed as unexplained, and the owner decides whether it stays.
- **R2:** The hints marked "adopt" are declarable once on a typed API in the shared kit, lift into the IR, and are validated by the Go reader. Errors: a hint that names a method or message the descriptors do not have, a relationship between a write and a read with no bound where one is required, and a bound of zero or less are each rejected at their Scala line.
- **R3:** The lowering derives waiting from the hints. A read after an eventually visible write lowers to a bounded wait for the condition; a read after a write visible at once lowers to one read. Errors: a path that reads after a write with no declared visibility is refused with both methods named, so a missing hint is found at lowering and never as a flaky test.
- **R4:** No realization contains a literal interval or timeout, and none writes an explicit poll for a condition a hint covers. Errors: a wait no hint covers is listed with its reason and keeps its explicit form.
- **R5:** The Go framework that runs a lowered Case waits by condition within the declared bound and holds no default of its own for a wait a hint covers. Errors: a wait that runs out fails with the condition, the bound and the hint's Scala position named; a Profile that scales bounds states the factor in the Run.
- **R6:** Every Contract of an existing lowered Case is unchanged, proven by the baseline goldens with the Program differences listed. Errors: a changed Contract stops the task.
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

**Metadata on the API, not on the realization.** That a pause is visible to a describe only eventually is true for every Model that pauses and describes. Declared on the API pair it is written once. Declared per realization it is rediscovered each time, which is today's state.

**Through the IR.** Go builds every Case from the IR alone. A hint kept only in Scala would have to be turned into explicit commands by Scala helpers, which hides it from the lowering and from a failure message. The cost is an IR schema change.

**Refuse a read with no declared visibility.** The alternative is a default, either "at once", which produces flaky tests, or "eventually", which makes every test slow. Refusing makes the author state the fact once.

**Start with three hints.** A general annotation system invites hints nobody uses. The three adopted ones replace code that exists today, and the rest wait for a Case that needs them.

## Parked unknowns

- Whether a visibility relationship is declared between two methods, or between a write and the field a read returns. The inventory shows which the existing waits need.
- Whether wait bounds belong with the hint or with the Profile of the environment a Case runs in. The proposal is a declared bound that a Profile may scale.
- Whether the Testpilot IR needs a new field to carry the hint's position for the failure message, or the Case's provenance already can.
