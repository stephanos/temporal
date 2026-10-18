# The Scala checker is the oracle

> HTML render lens (local-only, ignored): open `.flow/artifacts/fn-196-the-scala-checker-is-the-oracle/spec.html` locally. Regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-17): "also; should we have the simpistic (simple as can be) model checker in Scala now? the extra step via IR and Go seems slow. wed replace the Go one entirely. this is assuming it's fairly easy to write a simple, not high-performant one."
> owner decision (AskUserQuestion, 2026-10-17): "Scala checker, third spec": the Scala checker becomes the oracle, ModelSuite runs its sections in-process, this spec captures the checker move, the Go check deletion and the lowering decision after a measurement.

## Goal & Context

<!-- Goal & Context: 50% [user], 50% [paraphrase] -->

The framework already has the simple Scala checker the owner asked for. `NativeCheck` explores a
Machine's reachable graph breadth-first from its starts, runs each step through the var runtime,
checks every invariant and postcondition on every step, monitors on every path and progress with
cycle bookkeeping and fairness, under a state budget where a cut answers unknown. The sample suites
run on it. The plan of 2026-10-13 made the Go checks over the IR the oracle, with an IR evaluator
to follow and the Scala checker to be deleted. This spec inverts that: the Scala checker is the
oracle, the Go check and the planned IR evaluator go, and the IR remains the export and lowering
format. The reason is the one the owner gave: the step through the IR and a Go interpreter is slow
and indirect for a question compiled Scala can answer directly, and one oracle is enough.

What stays in Go is what performs Cases: the Testpilot runtime, the Driver and the canary. They
never execute a Model.

## Architecture & Data Models

<!-- Architecture: 30% [user], 50% [paraphrase], 20% [inferred] -->

- **A checker module** outside `framework.dsl`, the destination fn-190.15 already moves the native
  runtime to, now a product module rather than test support. `framework.dsl` keeps declarations;
  the checker executes them.
- **Its questions.** `explore(instance, starts, budget)` builds the reachable graph; `verifyAll`
  checks every claim of every section from every start under its `in` environment (fn-190 R15);
  `find(space, horizon, target)` for reachability; `replay(instance, path)` for a pinned path with
  expectations at positions (fn-195 examples and conformance self-tests); `sample(space,
  count, seed)` for seeded random walks judged by the claims; `refuted(machine, claim)`;
  `monitor` and `progress` as today.
- **Its answers.** One `Result` shape for every question: verdict (`verified`, `refuted`, `found`,
  `notFound`, `unknown`), the budget and whether exploration completed, the counterexample or
  witness path with states and outcomes, and the failing `expect:` line with the state for a
  replay; a `Report` adds vacuity (actions never fired, `after(action)` postconditions never
  triggered) in fn-190 R15's shapes.
- **Budget** is the caller's, as fn-190 R14 decided: a default plus per-suite overrides in the gate's
  configuration, never in a Model or a test; every Result carries it.
- **The gate** runs every `ModelSuite` through munit; the Scala checker is what the suites call. The
  Go `check` run of the samples leaves the gate.
- **Compositions** are checked as Machines over the product of their members' vars (fn-193).
- **Measurement before deletion.** One design note records time and live heap for the largest
  graphs available (the six samples at their intended budgets and the parked Temporal sources in
  the equivalence testdata, lifted or replayed as the note can) and a go or no-go on deleting the
  Go check. A state is keyed by a canonical hash so the visited set holds hashes, not states, if
  the measurement needs it.
- **After go:** the Go `check` package, the Go explorer's use as a checker, the planned IR evaluator
  (fn-190.3) and the deletion task fn-190.5 are retired or re-cut; conformance replay of a recorded
  Run against the Model moves to the Scala checker. The IR keeps Quint and TLA export and the
  lowering of conformance tests to Cases; where the lowering runs (Go over the IR, or Scala
  emitting the Case proto, since the realizations are Scala) is decided in the measurement note.

## API Contracts

```scala
// what a ModelSuite calls; every answer is a Result
val graph  = Checker.explore(instance, starts = instance.initialStates, budget = Budget.default)
val claims = Checker.verifyAll(graph)                    // Vector[Result], one per claim and start environment
val reach  = Checker.find(space, horizon = 4)(target)    // found | notFound | unknown, with the witness
val run    = Checker.replay(instance, path)              // verified | refuted, with the failing line and state
val walks  = Checker.sample(space, count = 200, seed = 1)
val broken = Checker.refuted(derived, claim)
```

Errors: a budget of zero or less is refused; a path whose step is disabled in the state it is taken
from is a `refuted` Result naming the step and the state, never an exception; a `find` with no
target or a horizon under one is refused; a `sample` with a count under one is refused; a Model
whose effect throws is a `refuted` Result carrying the exception's message and the state.

## Edge Cases & Constraints

- A budget-cut exploration answers `unknown` for both verification and finding, as fn-190 R14
  requires; a counterexample or witness found before the cut stands. `notFound` means the search
  completed within the requested horizon without reaching the target, not that its budget ran
  out. This distinguishes a bounded negative answer from an incomplete search. [inferred]
- Visited states are deduplicated structurally; a Model whose state holds an unordered collection
  compares equal by value, which `Finite`'s structural equality gives.
- Memory is the risk the measurement exists for; the hash-keyed visited set is the mitigation and is
  built only if the measurement asks for it.
- Nothing in the checker reads the IR; a Model that lifts and a Model that does not check alike.

## Acceptance Criteria

- **R1:** The Scala checker lives in a module outside `framework.dsl` with the public questions in
  API Contracts and one `Result` shape carrying verdict, budget, completion and the path or failing
  line; `framework.dsl` executes nothing. Errors: as in API Contracts. [paraphrase]
- **R2:** `verifyAll` reports vacuity (actions never fired, postconditions never triggered) in
  fn-190 R15's shapes, and the budget comes from the gate's configuration with per-suite overrides.
  Errors: an override naming no suite is a configuration error. [paraphrase]
- **R3:** `replay`, `find` with a horizon, `sample` with a seed and `refuted` exist and are what
  fn-195's sections call; `sample` with one seed replays the same paths twice. Errors: as in
  API Contracts. [inferred]
- **R4:** A design note in `model/design` records time and live heap for the largest available
  graphs and a go or no-go on deleting the Go check, with the lowering location decided in the same
  note. Errors: the note is not optional; a no-go keeps the Go check and reopens this spec's
  Boundaries. [user]
- **R5:** *(2026-10-18, one-language series: closed as REPLACED by fn-204, not implemented: the design changed; Scala emits the conformance descriptor (fn-204 R1) and Go judges a recorded Run from it (fn-204 R2); the clause that existing replay, binding and table guarantees remain until replaced now lives on fn-204 R2; the Go check deletion and the sample-check retirement close by reference to fn-208 and fn-201)* On go, the Go `check` package and the IR evaluator task are retired, fn-190.5 is re-cut to
  delete the Go check rather than the Scala runtime, the gate's sample check runs the Scala checker,
  and conformance replay of a recorded Run against the Model runs in Scala. Errors: a Go test that
  only exercised the Go check is deleted with it, one that exercised lowering or export stays.
  [paraphrase]
- **R6:** The six samples' intended answers on the default budget match the inventory on the Scala
  checker. Errors: a changed answer is a finding to fix. [inferred]
- **R7:** *(2026-10-18, one-language series: the IR is no longer the export and lowering format after fn-201; it is an internal lift tree)* MILESTONES' Direction, the framework README and SEMANTICS say the Scala checker is the
  oracle and the IR is the export and lowering format. Errors: none beyond review. [inferred]

- **R8:** Discarded parametric Models and their declarations become collectible after native initialization and checking when no live Model, State, Result, action or other legitimate owner remains. Runtime snapshots and discovery metadata follow those owners' lifetimes. Errors and boundaries include exact foreign-owner refusal, reference-identity lookup, stable retained-result diagnostics, live channel behavior, static/derived/composed owners, nested and exceptional restoration, and concurrent access to one owner's runtime. No disposal API or cache clearing is required. This repairs the instance-retention finding from R4's measurement before ModelSuite adopts fresh instances.

## Boundaries

The accepted measurement decision is no-go on immediate Go check package retirement. Keep existing Go consumers until native refinement/replacement obligations, the rule-disjointness disposition, recorded-Run/backend replay contracts and Case production dependencies have evidence-backed replacements. The selected future producer emits Testpilot Cases in Scala; choosing its location does not implement it or release Phase 3. Runtime and discovery lifetime repairs remain native follow-ups before ModelSuite adoption.

- The Testpilot runtime, Driver and canary stay in Go and do not change. [paraphrase]
- Quint and TLA export stay over the IR. [paraphrase]
- The lowering of conformance tests to Cases is not moved by this spec; the measurement note decides
  where it runs. [paraphrase]
- No performance target beyond the measurement; the checker stays simple, breadth-first and
  single-threaded unless the note says otherwise. [user]
- Refinement and rule-disjointness checks that exist only over the IR are not reimplemented here;
  they are listed in the measurement note as kept, moved or dropped. [inferred]

- Instance reclamation excludes intentionally live Results, States and actions, and classloader/static metadata. The repair does not change reachable graphs, budgets, verdicts, refinement standings, or the Go-deletion decision.

## Decision Context

The owner asked for the simplest Scala checker now, on the assumption it is easy to write, and to
replace the Go one entirely. [user] It is already written: `NativeCheck` is that checker, which is
why the cost is a move and a measurement rather than an implementation. [paraphrase] This reverses
the 2026-10-13 decision that the Go checks are the oracle; that decision's reason was a single
oracle, and the single oracle is now the compiled Scala one. [paraphrase] The IR survives as the
export and lowering format because Quint, TLA and Testpilot Cases need trees, which is also why the
plugin keeps lifting claims and `expect:` predicates and stops needing effects and transitions
(a diet beyond fn-189's). [inferred] Measurement before deletion because the Go run's live heap on
the Temporal graphs was the one observed number and Scala states are heavier per state. [paraphrase]
Rejected: keeping both checkers (two oracles); building the IR evaluator first (the Scala checker
already answers the samples). [inferred]

The measurement's repeated-instance controls found 0 of 64 constructed-only Models retained, all 64 initialized Models retained by the global var-runtime map, and all 64 checked Models retained by that map plus action discovery names. The native follow-up repairs runtime ownership first and declaration indexes second, then reruns the combined retention control before oracle documentation and ModelSuite admission. Weak keys alone are insufficient when cached values point back to their keys. Classloader-cache redesign, visited-set optimization and a performance target remain outside this repair.

2026-10-18, one-language series: R5 is closed as replaced by fn-204 (owner decision). The Boundaries bullet "the Testpilot runtime, Driver and canary stay in Go and do not change" bounded fn-196's own work; fn-202 is the authorized change to the runtime's layout (protos under the framework, a driver seam for the seven Temporal-typed fields, a generic/Temporal split). The bullet "Quint and TLA export stay over the IR" is superseded: backends are Scala, over an internal lift tree (fn-201); Apalache in fn-206 and TLC in fn-207 on a direct TLA+ emission, Quint in fn-205 (deferred). The opening Boundaries paragraph ("keep existing Go consumers until …") is discharged by fn-199 (refinement, replacement, composed fairness), fn-200 (lint), fn-203 (Cases), fn-204 (recorded-Run assessment) and fn-208 (deletion); fn-196.4 closes by reference to fn-208. The series is fn-199 through fn-209.

2026-10-18, performance diagnosis (task .8): R4's measurement (Bank 83 s, Voting7 cut) is dominated by one implementation choice, not by the design: `ClaimEvaluation.evaluate` defines a fresh class per claim evaluation (new ClassLoader, defineClass, Class.forName, reflective constructor), about 744k times on Bank, so claim bodies never JIT and metaspace churns. Solving Bank and Voting5 together gives about 108 us per section evaluation versus under 3 us for the rest of a step. Task .8 fixes that first (expected 83 s to 2-4 s), then the smaller per-step costs. The measurement note's conclusions about budgets and the no-go are to be re-read after .8 lands; the fn-209 latency comparisons must use post-.8 numbers.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The Scala checker lives in a module outside `framework.dsl` with the public questions in API Contracts and one `Result` shape carrying verdict, budget, completion and the path or failing line; `framework.dsl` executes nothing. Errors: as in API Contracts. | fn-196-the-scala-checker-is-the-oracle.1, fn-196-the-scala-checker-is-the-oracle.2 | — |
| R2 | `verifyAll` reports vacuity (actions never fired, postconditions never triggered) in fn-190 R15's shapes, and the budget comes from the gate's configuration with per-suite overrides. Errors: an override naming no suite is a configuration error. | fn-196-the-scala-checker-is-the-oracle.2 | — |
| R3 | `replay`, `find` with a horizon, `sample` with a seed and `refuted` exist and are what fn-195's sections call; `sample` with one seed replays the same paths twice. Errors: as in API Contracts. | fn-196-the-scala-checker-is-the-oracle.2 | — |
| R4 | A design note in `model/design` records time and live heap for the largest available graphs and a go or no-go on deleting the Go check, with the lowering location decided in the same note. Errors: the note is not optional; a no-go keeps the Go check and reopens this spec's Boundaries. | fn-196-the-scala-checker-is-the-oracle.3 | — |
| R5 | On go, the Go `check` package and the IR evaluator task are retired, fn-190.5 is re-cut to delete the Go check rather than the Scala runtime, the gate's sample check runs the Scala checker, and conformance replay of a recorded Run against the Model runs in Scala. Errors: a Go test that only exercised the Go check is deleted with it, one that exercised lowering or export stays. | fn-196-the-scala-checker-is-the-oracle.4 | — |
| R6 | The six samples' intended answers on the default budget match the inventory on the Scala checker. Errors: a changed answer is a finding to fix. | fn-196-the-scala-checker-is-the-oracle.4 | — |
| R7 | MILESTONES' Direction, the framework README and SEMANTICS say the Scala checker is the oracle and the IR is the export and lowering format. Errors: none beyond review. | fn-196-the-scala-checker-is-the-oracle.5 | — |
| R8 | Discarded parametric Models and their declarations become collectible after native initialization and checking when no live Model, State, Result, action or other legitimate owner remains. Runtime snapshots and discovery metadata follow those owners' lifetimes. Errors and boundaries include exact foreign-owner refusal, reference-identity lookup, stable retained-result diagnostics, live channel behavior, static/derived/composed owners, nested and exceptional restoration, and concurrent access to one owner's runtime. No disposal API or cache clearing is required. This repairs the instance-retention finding from R4's measurement before ModelSuite adopts fresh instances. | fn-196-the-scala-checker-is-the-oracle.6, fn-196-the-scala-checker-is-the-oracle.7 | — |
