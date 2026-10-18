## Conversation Evidence

> user (2026-10-17): "the test files for a model serve two purposes: (1) test the model (example based or sampled based or property based) but we also (2) need to find a deisgn where they are translated (or only the ones we can actually translate) into conformance tests. meaning we move that repsonsibility out of the mdoel and into the test file. and the Cases are then only created from the test spec."
> user: "Quint puts those into its tests. I think we should od the same" (on `Query`, `limit`, `king, donkeykong, peaches, mario`)
> user: "I really like the `expect:` block style"
> user: "do we seprarate runs that are a single example from runs that are more like queries that explore a state?"
> user: "are we covering and clearly separting thiese: a. testing the mmodel itself as a example based test b. testing the model itself with properties c. testing the model itself with sampling/simulation d. testing the real system with examples e. testing the real system with queries or other ways to describe the state space to explore. and are c and e the same just in different contexts?"
> user: "why wouldn't we follow the test("..."): notation?"
> user: "how about using an annotation like `@...` to clearly denotate the differnet test types that aren't unit tests? or follow the same idea as the model by using sections like `object ...:`"
> user: "> client.send(Addr.king, Addr.donkeykong, Denom.banana, 1) why is that not using Send? and named params? it's hard to read what it does"
> owner decisions (AskUserQuestion, 2026-10-17): capability expectation bindings move into the test file; a run names its start when init is a set, default otherwise; `oneOf` is the only variation, `Exploration`'s `runs`/`edits`/`dropPrefix` retire; one file, fallback a separate declarations file; bare Booleans in `expect:` only; `find(steps = n)` is the horizon, part of the question; a machine with one realization binds every conformance test to it.

> user (2026-10-18): "let's file a task for standardize primary machine declarations on class Model".
> user (2026-10-18): "we should also remove support for object Model entirely for consitency".

> user (2026-10-18): "once the DSL has settled after the last task, add antoher task after to go over each umpire frameowkr sample and simpify it (wihoutht losing behavior); each one must be the best version of itself".

## Goal & Context

<!-- Goal & Context: 60% [user], 40% [paraphrase] -->

A Model's test file does two jobs. It tests the Model, by example, by exhaustive claims and by
sampling. It also declares the conformance tests that run against the real server, and those are the
only source of Cases. Today the Model carries the questions asked of it (`object queries`, `find`,
`verify`, `witness`, `limits`, `.live`, `.explore`, capability expectation bindings), the sample tests
check predicates on hand-built states, and nothing resembles Quint's `sendTest`, an init-then-send-
then-expect script. fn-190 R14/R15 already retired `limits` and reduced `queries` to witnesses and
pinned paths; this spec takes the last step and moves every question into the test file.

The five kinds of test the owner named, and where each lives:

| Form | Of the Model | Of the system |
| --- | --- | --- |
| Authored path | a. `examples` | d. `conformance` |
| Claims over a space | b. the Model's claim sections, verified by default (fn-190 R15); `explorations` for reachability | (exhaustive is impossible against a server) |
| Generated paths over a described space | c. `explorations`, `sample` | e. `conformance`, `sample` |

c and e are one activity in two contexts, as a and d are: the same seeded paths, replayed on the
Model and performed on the server, with the Model replay as each Case's oracle.

Execution is in-process on the Scala checker (fn-196). A section's tests are munit tests: a
failing `expect:` line fails the test. Phase 2, DSL completion; the lowering changes are bounded.

## Architecture & Data Models

<!-- Architecture: 50% [user], 40% [paraphrase], 10% [inferred] -->

- **`ModelSuite(instance)`** is a munit suite over one Model instance (fn-194). It defines
  `test(name, from = <start>, expectation = <RunExpectation>, through = <realization>)` returning a
  builder whose block body is the test, shadowing munit's `test`, and three section objects the suite
  registers: `examples`, `explorations`, `conformance` and `unit`. A `unit` body is plain munit, for tests of pure helpers. A suite's top level holds only its sections and shared helper definitions; a `test` outside a section is refused, the rule the Model's lint applies to a machine object.
- **Primary machines** use `class Model`, including parameterless declarations. Each ModelSuite binds a fresh stable-named instance. Singleton primary Machine declarations are no longer supported, regardless of object name. Active native fixtures migrate with the samples. Objects remain available for namespaces, constants, actors and shared declarations. The six-sample migration is followed by fn-195.6, before the final documentation task.
- **A body in `examples` or `conformance`** is a sequence of statements: an action call by its
  record's named constructor (`client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom =
  Denom.banana, amount = 1)`), a whole action with no fields (`heater.heat`), `oneOf(step, step)` for
  a finite family, and `expect:` blocks whose every statement is one Boolean assertion on the state
  after the last step, with `outcome`, `before` and `records(fact)` in scope as in a postcondition.
- **A body in `explorations`** is a `find`, a `sample` or a `refuted` over a space. A space is a
  `val anySends = explore(steps = 6): <steps with any>` in the same section. `any` at an input or an
  unstated step is a search; `space.find(steps = n): expect: ...` stops at the first path reaching
  the target; `space.sample(count, seed)` replays that many seeded paths and judges them by the
  Model's claims; `refuted(derivedMachine, claim)` is the negative control (replaces
  `assertRefuted`).
- **`conformance`** holds the Cases. Each test is also replayed on the Model as its self-test. The
  expectation defaults to `satisfied` for an authored path and `conformant` for a `sample`; others
  are written. `through` is required only when the instance has several realizations. A capability
  expectation is a conformance test over the capability's generated path.
- **Rules.** `oneOf` enumerates, `any` searches, only `explorations` may use `any`. Case names are
  `<instance>.<slug(test name)>`, a sample's Cases `<instance>.<slug>.<seed>.<index>`.
- **The Model loses** `object queries`, `Query`, `find`, `verify`, `witness`, `scenario`, `limits`,
  `.live`, `.explore`, `.records`/`.ends`, and the capability binding's live switch.
  `Realizes(machine, runs = ...)` loses `runs`. `satisfied`, `inconclusive(reason, because)` and
  `conformant` are constructors in the realize package.
- **Lifting (later phase).** The exporter lifts the three sections of a `ModelSuite` subclass from a
  test file and nothing else in it; the `expect:` trees become Contract assertions at their
  positions. Fallback if lifting from a test file is awkward: `X.runs.scala` beside `X.test.scala`.

## API Contracts

```scala
class BankTest extends ModelSuite(Bank.Model(holder = Addr.king, supply = 2)):

  object examples:
    test("the king sends bananas, donkeykong cannot send burgers"):
      client.send(fromAddr = Addr.king, toAddr = Addr.donkeykong, denom = Denom.banana, amount = 1)
      expect:
        balances(Addr.king, Denom.banana) == 1
        balances(Addr.donkeykong, Denom.banana) == 1
      client.send(fromAddr = Addr.donkeykong, toAddr = Addr.peaches, denom = Denom.burger, amount = 1)
      expect:
        outcome == Outcome.invalidCoinsOrInsufficientFunds
        balances == before.balances

  object explorations:
    val anySends = explore(steps = 6):
      client.send
    test("an overflow is reachable within four sends"):
      anySends.find(steps = 4):
        expect:
          outcome == Outcome.overflow
    test("sampled sends hold the claims"):
      anySends.sample(200, seed = 1)
    test("a leaky bank is refuted on the supply invariant"):
      refuted(BankControls.Leaky, BankControls.Leaky.invariants.totalSupplyInv)

  object conformance:
    test("peaches gets a banana"):
      client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)
      expect:
        balances(Addr.peaches, Denom.banana) == 1
    test("sampled sends conform"):
      explorations.anySends.sample(200, seed = 1)

  object unit:
    test("Balances.set drops a zero amount"):
      assertEquals(Balances(Map.empty).set(Addr.king, Denom.banana, 0), Balances(Map.empty))
```

Kettle names its start: `test("boils from room temperature", from = room):`. A conformance test with
a non-satisfied expectation: `test("retry completes", expectation = inconclusive(Reason.explanationsDisagree, because = "..."))`.

Errors, each a compile error or a located lint refusal at the test: a `test` outside the four sections; a statement in `examples` or
`conformance` outside the liftable subset; `any` outside `explorations`; a `find` or `sample` in
`examples`; a test without `from` on an instance whose init is a set; a conformance test whose steps
include a class no realization performs or waits for; `through` naming another Model's realization,
or omitted with several; an `expect:` statement that is not Boolean; an `expect:` block with no
statements; a test name that slugs to an existing name in the same suite; `oneOf` with one
alternative; a `sample` count of zero.

## Edge Cases & Constraints

- The expectation's `because` rule stays as in fn-190: non-satisfied outcomes require text,
  satisfied ones refuse it.
- A conformance test replays on the Model first; a Model replay failure is reported as that test's
  failure and the Case is not generated.
- Repeated steps: the spelling (`heater.heat * 80` or `repeat(80)(heater.heat)`) is a prototype item
  recorded in Parked unknowns.
- The sections are objects nested in a class; `ModelSuite` touches them at construction so munit
  registers their tests. Section lookup by name is the mechanism the claim sections use.
- Bare Booleans are accepted in `expect:` only; claim sections keep `check(...)`.

## Acceptance Criteria

- **R1:** `ModelSuite(instance)` registers munit tests from its `examples`, `explorations`,
  `conformance` and `unit` sections; `unit` bodies are plain munit; the top level holds only sections
  and shared helper definitions. Errors: a `test` outside a section, and a section with a non-test
  member other than a space `val`, are refused naming the member. [user]
- **R2:** An `examples` body of named action calls, whole actions, `oneOf` families and `expect:`
  blocks is replayed on the instance by the Scala checker from `from` or the single start; each
  `expect:` statement is one assertion whose failure names the line and the state. Errors: the
  liftable-subset and `from` refusals in API Contracts. [user]
- **R3:** `explorations` supports `explore(steps)` spaces with `any`, `space.find(steps)` with an
  `expect:` target, `space.sample(count, seed)` judged by the instance's claims, and
  `refuted(machine, claim)`; `assertRefuted` and `NativeControls` are retired. Errors: `any` outside
  `explorations`, zero `sample` count, `find` with no target. [user]
- **R4:** `conformance` tests are the only declarations the Case generator reads; each is replayed
  on the instance first; the expectation defaults to `satisfied` for a path and `conformant` for a
  sample; `through` binds by the decision in Architecture. Errors: the conformance refusals in API
  Contracts. [user]
- **R5:** The Model has no `queries` section, `Query`, `find`, `verify`, `witness`, `scenario`,
  `limits`, `.live`, `.explore` or capability live switch; `Realizes` has no `runs`; `satisfied`,
  `inconclusive` and `conformant` live in the realize package. Errors: the retired names are
  unreachable by authors (fn-190.16's grep). [paraphrase]
- **R6:** The six samples' tests are rewritten as `ModelSuite`s with Quint's runs as examples
  (Bank's `sendTest`, TwoPhaseCommit's `successTest`), their reachability finds as explorations,
  their negative controls as `refuted`, and pass on the Scala checker with intended answers
  unchanged. Errors: a changed answer is a finding to fix. [paraphrase]
- **R7:** The lint applies the section rules in API Contracts, each refusal located at the test.
  Errors: as listed. [inferred]
- **R8:** SEMANTICS gains a section for the test file (the five-kind table, the sections, the rules),
  §9 Scenarios and Queries is rewritten to say the Model asks nothing of itself, and the README's
  test guidance follows. Errors: none beyond review. [inferred]

- **R9:** All six primary native sample machine declarations use `class Model`, including parameterless models; suites bind fresh stable-named instances with independent state and exact ownership. Singleton primary Machine declarations are refused regardless of object name, and active native fixtures/clients migrate without a compatibility escape hatch. Existing behavior, verdicts and realization bindings remain unchanged. Objects for namespaces, constants, actors and shared declarations remain; held Temporal/IR bodies are not migrated here. [user]

- **R10:** After the DSL, ModelSuite, class-only declarations and their docs settle, review and simplify every native framework sample into its clearest idiomatic form without changing behavior, claims, bounds, counterexamples or realization bindings. Preserve test purposes and verify before/after evidence; exact upstream comparisons and held consumers remain unchanged. [user]

## Boundaries

- Lifting the sections from test files, Contract assertions at intermediate `expect:` positions, and
  Case generation from a `sample` are the IR phase; this spec ends at declarations that replay on the
  Scala checker. [paraphrase]
- The realization (commands, evidence, waits, roles, cleanup) does not change. [paraphrase]
- No power-assert rendering beyond naming the failing line and state; a diagram is a later item.
  [inferred]
- The Temporal feature Models' `queries` sections are re-homed as `ModelSuite`s when those Models
  return to the tree, by recipe, not here. [inferred]

## Decision Context

Sections over a per-test `.live` switch or an annotation: an annotation attaches to a definition,
not a statement, so it would force the per-definition marker the owner rejected for `@lifted`;
sections are the Model's own idiom and let lint judge a container, and the Case rule becomes one
greppable line. [user] `test("..."):` over `val x = run:`: the owner asked why not munit's notation;
the only thing a `val` bought was a reference, and the realization no longer names runs; spaces stay
`val`s because other tests reference them. [user] `expect:` as an indented block of bare Booleans over
`.expect(x == y)` or infix chains: Scala has no bare `assert x == y`, and the owner chose the block.
[user] Named record constructors in action calls: the owner found the positional form unreadable, and
the named call is the DSL's existing checked form. [user] Examples and queries separate: an example
is a fixed path with a stable identity, a query's path is a byproduct of search and must not become
a Case by accident; a found witness is frozen into an example by hand. [paraphrase] `sample` in both
`explorations` and `conformance` because c and e are one activity. [paraphrase] The seven owner
decisions in Conversation Evidence stand as written. [user] A fourth section `unit` for plain tests rather than top-level `test`s: the owner asked for consistency; with `test` shadowed, location would otherwise carry the lifted-or-plain difference, and the Model's lint already refuses top-level declarations in a machine object. [user] Rejected: lifting munit bodies in general
(too large a liftable subset); a separate `Runs` object (an extra container the sections make
unnecessary); keeping `.live` (a per-test switch a reviewer can miss). [inferred]

## Parked unknowns

- The spelling of a repeated step in a body (`heater.heat * 80` or `repeat(80)(heater.heat)`);
  resolved by the first Kettle example written.
- Whether claim sections also accept bare Booleans; resolved after living with `expect:`.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `ModelSuite(instance)` registers munit tests from its `examples`, `explorations` and `conformance` sections and runs `test`s outside them as plain munit. Errors: a section with a non-test member other than a space `val` is refused naming the member. | fn-195.1, fn-195.2 | — |
| R2 | An `examples` body of named action calls, whole actions, `oneOf` families and `expect:` blocks is replayed on the instance by the Scala checker from `from` or the single start; each `expect:` statement is one assertion whose failure names the line and the state. Errors: the liftable-subset and `from` refusals in API Contracts. | fn-195.1, fn-195.2 | — |
| R3 | `explorations` supports `explore(steps)` spaces with `any`, `space.find(steps)` with an `expect:` target, `space.sample(count, seed)` judged by the instance's claims, and `refuted(machine, claim)`; `assertRefuted` and `NativeControls` are retired. Errors: `any` outside `explorations`, zero `sample` count, `find` with no target. | fn-195.2 | — |
| R4 | `conformance` tests are the only declarations the Case generator reads; each is replayed on the instance first; the expectation defaults to `satisfied` for a path and `conformant` for a sample; `through` binds by the decision in Architecture. Errors: the conformance refusals in API Contracts. | fn-195.2 | — |
| R5 | The Model has no `queries` section, `Query`, `find`, `verify`, `witness`, `scenario`, `limits`, `.live`, `.explore` or capability live switch; `Realizes` has no `runs`; `satisfied`, `inconclusive` and `conformant` live in the realize package. Errors: the retired names are unreachable by authors (fn-190.16's grep). | fn-195.3 | — |
| R6 | The six samples' tests are rewritten as `ModelSuite`s with Quint's runs as examples (Bank's `sendTest`, TwoPhaseCommit's `successTest`), their reachability finds as explorations, their negative controls as `refuted`, and pass on the Scala checker with intended answers unchanged. Errors: a changed answer is a finding to fix. | fn-195.4 | — |
| R7 | The lint applies the section rules in API Contracts, each refusal located at the test. Errors: as listed. | fn-195.3 | — |
| R8 | SEMANTICS gains a section for the test file (the five-kind table, the sections, the rules), §9 Scenarios and Queries is rewritten to say the Model asks nothing of itself, and the README's test guidance follows. Errors: none beyond review. | fn-195.5 | — |
| R9 | Primary machines use class Model with independent suite instances; singleton primary Machine support is removed and active native fixtures migrate, preserving behavior and ownership. | fn-195.6 | — |
| R10 | Final per-sample idiomatic simplification after the DSL settles, with behavior and verification evidence preserved. | fn-195.7 | — |
