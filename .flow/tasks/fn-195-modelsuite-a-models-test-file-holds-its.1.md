---
satisfies: [R1, R2]
---
# fn-195-modelsuite-a-models-test-file-holds-its.1 DSL completion: prototype a ModelSuite on Bank and record the verdict

## Description
**DSL completion phase (owner, 2026-10-17).** Cross-spec prerequisites: fn-194.2 (the parametric Bank instance) and fn-196.2 (`replay`, `find`, `sample`, `refuted`). Prototype the suite on Bank before changing the DSL: `class BankTest extends ModelSuite(Bank.Model(...))` with `examples`, `explorations`, `conformance` and `unit` section objects the suite touches at construction so munit registers their tests; a `test(name, from, expectation, through)` builder shadowing munit's `test`, whose block body is the test; an `expect:` macro lifting each bare Boolean statement to one assertion that fails naming the line and the state; named record-constructor action calls, whole actions, and `oneOf(step, step)` families; `explore(steps)` spaces with `any`. Decide the spelling of a repeated step (`heater.heat * 80` or `repeat(80)(heater.heat)`) on a Kettle example, and whether the trailing-parameter form `test("...", expectation = ...)` reads well. Record the verdict per Architecture bullet by revising `model/design/runs-in-tests.md`, and keep the prototype beside a README and run script.

**Size:** M

**Files:** `model/design/runs-in-tests.md`, a prototype under `model/design/modelsuite-prototype/` (ModelSuite sketch, Bank and Kettle suites, README, run.sh).

### Verification
- The prototype suites pass with Quint's `sendTest` as an example, an overflow `find`, a seeded `sample` and the Leaky `refuted`; a deliberately wrong `expect:` line fails naming its line and state.
- The note states a verdict per bullet, the repeated-step spelling, and resolves the Parked unknowns it can.

## Acceptance
- [ ] Verdict note revised in `model/design` with one verdict per Architecture bullet
- [ ] Prototype kept with README and run script; section registration and the `expect:` failure message demonstrated
- [ ] Repeated-step spelling decided

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
