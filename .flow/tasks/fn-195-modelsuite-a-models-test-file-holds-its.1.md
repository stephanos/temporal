---
satisfies: [R1, R2]
---
# fn-195-modelsuite-a-models-test-file-holds-its.1 DSL completion: prototype a ModelSuite on Bank and record the verdict

## Description
**DSL completion phase (owner, 2026-10-17).** Cross-spec prerequisites: fn-194.2 (the parametric Bank instance) and fn-196.2 (`replay`, `find`, `sample`, `refuted`). Prototype the suite on Bank before changing the DSL: `class BankTest extends ModelSuite(Bank.Model(...))` with `examples`, `explorations`, `conformance` and `unit` section objects the suite touches at construction so munit registers their tests; a `test(name, from, expectation, through)` builder shadowing munit's `test`, whose block body is the test; an `expect:` macro lifting each bare Boolean statement to one assertion that fails naming the line and the state; named record-constructor action calls, whole actions, and `oneOf(step, step)` families; `explore(steps)` spaces with `any`. Decide the spelling of a repeated step (`heater.heat * 80` or `repeat(80)(heater.heat)`) on a Kettle example, and whether the trailing-parameter form `test("...", expectation = ...)` reads well. Record the verdict per Architecture bullet by revising `model/design/runs-in-tests.md`, and keep the prototype beside a README and run script.

**Size:** M

**Touches:** `model/design/runs-in-tests.md`, `model/design/modelsuite-prototype/**`

**Files:** `model/design/runs-in-tests.md`, a prototype under `model/design/modelsuite-prototype/` (ModelSuite sketch, Bank and Kettle suites, README, run.sh).

### Verification
- The prototype suites pass with Quint's `sendTest` as an example, an overflow `find`, a seeded `sample` and the Leaky `refuted`; a deliberately wrong `expect:` line fails naming its line and state.
- The note states a verdict per bullet, the repeated-step spelling, and resolves the Parked unknowns it can.


### Native admission and execution boundary

The parent admission record names integrated fn-194.1-.5 and fn-196.1-.3/.6/.7/.5 source
and closure commits. This prototype changes no public DSL/checker implementation. It may retain
prototype-only declarations and adapters under its own directory; label them honestly and replay
through the shipped native checker. Conformance here proves only native declaration/replay
behavior, not Case production, Contract lifting or server execution.

Read `.flow/tmp/fn195-prep/seams.md`, `registration-lifecycle.md`, and
`sample-obligations.md` before choosing seams. MUnit section registration must be demonstrated
after concrete subclass initialization and before enumeration; constructor touches are not
assumed safe. Prototype the actual four-section shape, initialized shared helpers, exactly-once
registration and deterministic registration order. The spec's R1 coverage row now agrees with the
owner's unit section/top-level-test refusal. Do not introduce a DSL-to-checker reverse import in
the shipped library to make a prototype compile.

Decide prototype spelling and routine trade-offs under the standing autonomous policy. Distinguish
such recommendations from direct owner quotations. Do not ask the owner to repeat decisions
already recorded. Keep a verdict per architecture bullet and explicit unresolved held proofs.

Own pre-edit baseline before prototype/test/helper-writer edits; no inherited green credit.
Run the existing checker Quick below, then prototype positives and the intended wrong-expectation
failure with exact line/state evidence. Cross-check the literal Quint example against the retained
local upstream copy without modifying comparison files. No new upstream-authored examples.

Fresh gate classification/check/receipt and appropriate affected native/source/library gates remain
required. If FULL, reuse `.flow/tmp/fn1965/verify.sh` with fresh task paths and strict flags, not
a docs skip. Serialize compiler/JVM and git lanes. Root alone edits MILESTONES; scoped commits,
review none under phase-batch deferral, tracker inactive, parent OPEN. Exclude all disabled Temporal
sources, broad GuideSourcesTest, real-Go SamplesSuite bodies, IR/export/lifter/schema/backend/
generated/regeneration and held Make routes before execution. No publication, Go deletion,
Make-green or review verdict.

### Quick command

```sh
timeout 600 env -u BASH_ENV mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest
```
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
