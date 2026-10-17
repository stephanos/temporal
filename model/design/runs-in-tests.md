# Runs in tests: examples and Cases leave the Model

Status: design, decided with the owner 2026-10-17, then revised the same day into specs fn-194 (parametric Models), fn-195 (ModelSuite) and fn-196 (the Scala checker as the oracle). Where this note and the specs differ, the specs rule: the `Runs` object, `.live` and `.through` became the `examples`/`explorations`/`conformance`/`unit` sections of a `ModelSuite` with `test("..."):`; the Model is a parametric class; the checker is Scala, not Go. No source has been changed. Paths are relative to
the repository root (`/Users/stephan/Workspace/skunkworks/umpire/temporal`). Line numbers come from
the tree this was written against, `37e268cf28` plus uncommitted edits to `Kettle.scala`.

## Summary

A Model's test file serves two purposes: it tests the Model, by example, by sampling or by property,
and it declares the conformance tests that run against the real server. Cases are generated only
from the test file. The Model keeps what it promises (vars, transitions, effects, invariants,
postconditions, progress, monitors, capabilities) and loses every question asked of it: `queries`,
`Query`, `find`, `verify`, `limits`, `.live`, `.explore`, and the capability expectation bindings
that made generated Queries live.

The shared kernel is the *pinned run*: a start, a schedule of classes with inputs, and expectations
along the way. The native checker replays it as an example of the Model; the lowering performs the
same path against the server as a Case. The first judgment is the second's self-test: a run that
does not hold on the Model is a wrong Case before any server is involved. Everything else in a test
file is one-sided. Hand-built states, refuted negative controls and sampled runs are model-only;
the realization (commands, evidence, waits, roles, cleanup) is server-only and stays where it is.

This follows fn-190 R14/R15, which retired `limits` and said `queries` keeps only witnesses, pinned
paths and realization checks, one step further: `queries` leaves the Model entirely.

## Decisions (owner, 2026-10-17)

| # | Question | Decision |
|---|----------|----------|
| 1 | Capability expectation bindings also generate live Cases from the Model. | They move into `Runs`. The Model declares the capability only; a Case has one source. |
| 2 | Which start does a run take when init is a set (`oneOf`)? | `run(from = state):` names one. Without `from`, a single-start init is the start and a set is refused. |
| 3 | `Exploration` carries `variations`, `runs`, `edits`, `dropPrefix`. | `oneOf` at a step is the only variation; `runs`, `edits`, `dropPrefix` and alternative `priority` are retired. A fixture that needs them is a finding. |
| 4 | Lifting a `Runs` object from a file that also holds a munit suite may be awkward. | Try one file. The fallback is `Bank.runs.scala` beside `Bank.test.scala`: two files, one declaration. |
| 5 | Each line of an `expect:` block is a bare Boolean; claim sections require `check(...)`. | Bare Booleans in `expect:` only, for now. Claim sections are revisited after living with it. |
| 6 | `find(steps = 4)` puts a number back in authored code after R14. | `steps` is the horizon, part of the question ("within four sends"); the search budget stays the gate's. |
| 7 | Who binds a live run to a realization? | A machine with one realization binds every live run to it. `.through(realization)` is required with several, refused for another machine's. |

## Shape

Two kinds of declaration, in an object the exporter lifts and the suite executes:

- `run:` is an **example**: every step and input fixed, replayed by the checker, performed by a Case.
  `oneOf(a, b)` at a step makes a finite family of fully pinned paths, each replayed and each
  lowered. Deterministic, so a Case identity is stable.
- `find:` is a **query**: some path does this, or none does. `any` at an input or an unstated step
  is a search; the answer depends on the budget and the path found is a byproduct. A `find` never
  takes `.live`. The checker prints a found witness as a `run:` block, which an author pastes in:
  the only way a search result becomes a Case is deliberately, as an example the author owns.

The rule is one line: `oneOf` enumerates, `any` searches, and only `find` may use `any`.

Sampled or property-based runs are the third kind and stay in the munit suite as plain tests over
`NativeCheck`. They have no fixed path and nothing to lift.

### Bank

```scala
// Bank.scala: no queries section. Addr and Denom stay: they are the state's domains.
object Model extends Machine[Outcome, Nothing]:
  var balances = Balances(of = Map(Addr.king -> Map(Denom.banana -> supply, Denom.burger -> supply)))
  object transitions extends Transitions: ...
  object effects: ...
  object invariants:
    val totalSupplyInv = check(totalSupply(balances))
  object postconditions:
    val failedSendChangesNothing =
      if outcome != Outcome.accepted then check(balances == before.balances)
```

```scala
// Bank.test.scala
package framework.dsl

import samples.Bank
import samples.Bank.*

// Pinned runs: plain values the exporter lifts and the suite executes. Each is an example of the
// Model from its initial state; one with `.live` is also a conformance Case.
object BankRuns extends Runs(Model):
  // the king has bananas and sends them: Quint's sendTest
  val sendTest = run:
    client.send(Addr.king, Addr.donkeykong, Denom.banana, 1)
    expect:
      balances(Addr.king, Denom.banana) == 1
      balances(Addr.donkeykong, Denom.banana) == 1
      balances(Addr.king, Denom.burger) == supply
    // donkeykong has no burgers, so this send fails and changes nothing
    client.send(Addr.donkeykong, Addr.peaches, Denom.burger, 1)
    expect:
      outcome == Outcome.invalidCoinsOrInsufficientFunds
      balances == before.balances

  // a self-send is accepted and pays nothing
  val selfSend = run:
    client.send(Addr.king, Addr.king, Denom.banana, 1)
    expect:
      outcome == Outcome.accepted
      balances == before.balances

  // a negative amount debits the receiver: Quint's robustness case
  val negativeSend = run:
    client.send(Addr.king, Addr.mario, Denom.banana, -1)
    expect:
      balances(Addr.mario, Denom.banana) == -1

  // peaches gets a banana: also a Case, expected satisfied against the server
  val peachesGetsBananas = run:
    client.send(Addr.king, Addr.peaches, Denom.banana, 1)
    expect:
      balances(Addr.peaches, Denom.banana) == 1
  .live(satisfied)

  // some send of one banana from the king leaves both holding one
  val kingCanPayPeaches = find:
    client.send(Addr.king, Addr.peaches, Denom.banana, amount = any)
    expect:
      balances(Addr.king, Denom.banana) == 1
      balances(Addr.peaches, Denom.banana) == 1

  // within four sends of anything, an overflow is reachable
  val overflowReachable = find(steps = 4):
    expect:
      outcome == Outcome.overflow

class BankTest extends munit.FunSuite:
  // every pinned run holds on the Model, live or not; a stale Case fails here, not on a server
  BankRuns.all.foreach: pinned =>
    test(s"run ${pinned.name} holds on the Model"):
      assertHolds(Model, pinned)

  // model-only: a hand-built state the server could never be put in
  test("the supply invariant refuses an inflated state"):
    val inflated = Model.init.copy(balances = held(Model.init).set(Addr.peaches, Denom.banana, 1))
    assertEquals(SampleClaims.holds(Model.invariants.totalSupplyInv, inflated), false)

  // model-only: the negative control tests the checker, not the Model
  test("a leaky bank is refuted on the supply invariant"):
    assertRefuted(BankControls.Leaky, invariant(check(totalSupply(balances))))

  // model-only: sampled runs, with no fixed path to lower
  test("sampled sends never break the supply"):
    forAll(sends(steps = 6)): path =>
      assert(NativeCheck.replay(Model, path).forall(s => totalSupply(held(s))))
```

### Kettle

Kettle's init is forty-one starts, so a run names one (decision 2):

```scala
object KettleRuns extends Runs(Model):
  val boilsFromRoomTemperature = run(from = room):
    user.pressButton
    heater.heat * 80
    expect:
      temperature == boilingTemperature
      heatingOn
    heater.failover
    expect:
      beeping
      !heatingOn
```

`heater.heat * 80` is a repeated step; whether it is spelled this way or as `repeat(80)(heater.heat)`
is a prototype item.

### A Temporal protocol

The former `queries` block of the mini Activity Protocol (`model/framework/SEMANTICS.md:583-592`),
as runs. The machine has one realization, `MiniActivityRun`, so `.through` is not written
(decision 7):

```scala
object ActivityRuns extends Runs(ActivityProtocol):
  val protocolCompleted = run:
    client.start()
    activityWorker.poll
    activityWorker.respondCompleted
    expect:
      records(Fact.statusCompleted)
  .live(satisfied)

  val retryCompleted = run:
    client.start(maxAttempts := MaxAttempts.two)
    activityWorker.poll
    activityWorker.respondFailed(Failure.retryable)
    expect:
      attempts == 1
    timers.backoff
    activityWorker.poll
    activityWorker.respondCompleted
    expect:
      records(Fact.statusCompleted)
      attempts == 2
  .live(inconclusive(Reason.explanationsDisagree,
    because = "explaining executions can disagree on the attempt count"))

  // the Nexus deadline exploration (dispatch-original System.scala:428-436) as a family: one
  // Case per alternative, each fully pinned
  val nexusDeadlines = run:
    oneOf(
      caller.schedule(startToClose := expires),
      caller.schedule(scheduleToClose := expires))
    handler.poll
    expect:
      records(Fact.timedOut)
  .live(satisfied)
```

A capability expectation (decision 1) is a run too: the Model declares the capability, the run
states what the server is expected to do with it and carries the `.live` expectation.

### Scoping and words

- Inside `expect:`, bare fields read the state after the step just taken; `outcome`, `before` and
  `records` refer to that step, as in a postcondition. The lifter already provides these.
- Each statement of an `expect:` block is one assertion (decision 5). The macro lifts the expression
  tree, as `check` does today, so the native failure can print each subexpression's value
  (power-assert style) and the lowering has the same tree for the Contract. This departs from
  `after(...):` blocks, where a bare Boolean is refused.
- `.live(expectation)` and `.through(realization)` hang off the block with a leading dot at the
  enclosing indentation. If that reads badly in practice, `run.live(satisfied):` as the opener is
  the alternative.
- `expect` was retired as an author word in fn-190.4 (`model/framework/dsl/Syntax.scala:283`
  still mentions it as "retired `expect`"). Reusing it for the block is deliberate: it now means
  one thing, the expectation after a step of a run, and only inside `Runs`.
- A `Runs` object's `all` is the one list the suite iterates and the lowering reads.

## Consequences

**DSL (this phase).** `Runs(machine)`, `run`, `run(from = ...)`, `find`, `find(steps = ...)`,
`expect:`, `oneOf` at a step, `any` at an input, `.live`, `.through`. Retired from the Model:
`object queries`, `Query`, `find`, `verify`, `witness`, `.records`, `.ends`, `.limits`, `.explore`,
`.live`, `scenario`, and the capability expectation binding's live switch. `SampleClaims.judged`
and `holds` take a claim member directly instead of `verify(member)`.

**Lint.** A run's `expect:` bodies are confined to the liftable subset (`lint/LiftableSubset`,
`lint/EligibleWitness` are the starting points). A `.live` run whose steps include a class no
realization performs or waits for is refused. `any` outside `find` is refused. A `find` with
`.live` is refused. A run without `from` on a machine whose init is a set is refused.

**Exporter and plugin (next phase).** The exporter lifts objects extending `Runs` from test files
and nothing else in them; munit on the classpath is accepted. If this proves awkward, the fallback
is `X.runs.scala` beside `X.test.scala` (decision 4). The IR `Query` message becomes a run: a start,
a schedule with `oneOf` positions, expectation blocks at positions, the live expectation and the
realization name. `Exploration` and its fields are removed from the IR.

**Go (next phase).** The lowering (`tools/umpire/lower`) performs a run's path and emits a Contract
with assertions at every `expect:` position, not only the last step. `lower.Realizable` and
`conformance.realizationOf` resolve the realization by decision 7. The checker gains `find` with a
horizon and prints a found witness as a `run:` block. Case identity stays the hash of canonical
bytes; the Case name is `<Model>.<run>`, so renaming a run is a Case change, and the pinned canary
Case (`tools/canary/casebinding`) is re-pinned once.

## Dropped, and why

- `Exploration.runs`, `edits`, `dropPrefix` and `Alternative.priority`: no shipped Model uses them
  outside the equivalence fixture copy of the Nexus System; `oneOf` enumerates the alternatives
  exhaustively, which is what a model check wants (decision 3).
- `verify` as a Query: fn-190 R15 verifies every claim from every start by default.
- Hand-written `find`s that only show an action fires: R15's vacuity reporting.
- `witness(...).records(...).ends(...)`: a `run:` with one `expect:` after its last step.

## Open items for the next phase

- Lifting from a file that also holds a munit suite (decision 4's fallback).
- Contract assertions at intermediate positions in the lowering.
- Whether claim sections also accept bare Booleans (decision 5, deferred).
- The spelling of a repeated step and of `run.live(...)` versus a trailing `.live`.
- The power-assert failure message for `expect:` lines, in the native checker and in the Go
  report against a recorded Run.
- When the Temporal feature Models return to the tree (currently zero `.scala` under
  `model/temporal`), their `queries` sections are re-homed as `Runs` objects in test files, and
  `Realizes(machine, runs = Seq(...))` loses its `runs` argument.

## Non-goals

- The realization: commands, evidence, waits, roles, cleanup. `.through` names one; nothing else
  changes.
- The model-only test vocabulary (`held`, `bank`, `SampleClaims`, native predicate truth tables).
- Whether the Go conformance harness runs the IR interpreter instead of its hand-written model
  (MILESTONES open item).
