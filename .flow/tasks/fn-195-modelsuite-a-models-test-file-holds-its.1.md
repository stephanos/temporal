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
Bank and Kettle now have a runnable native ModelSuite prototype with examples, explorations, conformance and unit sections. The revised design note records one verdict per Architecture bullet, the executed repeat(80)(heater.heat) choice and trailing expectation readability.

Task: fn-195-modelsuite-a-models-test-file-holds-its.1
Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; owner phase-batch deferral)

Base: 9ee19cccc59f149a61929e942be639fbc9cfb591
Source: 0b6b7118af6ee668942c7e34648bb371f6b01ca6
Failing reproduction: 307f56ce455db3aa76d8b342b32d758a145721c7
Final repaired source: 641cc2b19875ef730761be5a91e0dccaf4691c43

baseline: green. The worker ran the task's exact checkercontrol.CheckerTest Quick before edits and observed exit 0 with four tests. Gate check returned no honorable receipt. No handoff or prior green was used.

Verification is in final-commands.md and its hashed logs. Final source passed all 20 allowed native verifier stages with exit 0. Counts are native 673, Quick 4, class controls 7, rule suites 31, boundary controls 66, source lint 175 suites classified and separate-library question controls 2. Prototype tests are a separate 20, comprising Bank 8, literal Quint Bank 1, Kettle 2, lifecycle 7 and options registration 2. No GATE_SKIPPED lines apply.

The wrong-expectation run exited 1 for the intended semantic failure at Bank.test.scala:98, model.balances(Addr.peaches, Denom.banana) == 99, with a concrete state showing peaches banana 1. The following Boolean is true. The early-helper fixture separately exited 1 under unchanged safe-init/Werror flags. Actual Registry inventory exited 0 after confirming that only the two direct FunSuite control suites are recognized; the three ModelSuite suites need the explicit prototype selector.

Prototype verdicts:
- Keep the four sections and builder shape. Register at first munitTests before the inherited snapshot, after eager subclass helpers initialize. Repeated/concurrent enumeration, order, receivers, transforms, filtering, setup, ignore/only, unit-only/empty/absent sections, recursion, invalid members, top-level placement and sticky failures have controls.
- Keep fresh stable Bank instances. Kettle singleton migration and class-only refusal remain fn-195.6.
- Keep named record calls, whole-action adapters, finite oneOf and each located Boolean assertion. Before/outcome context adaptation and wrappers are prototype-only. Arbitrary authored-body lifting remains open.
- Keep reusable spaces, horizon-named find, seeded finite samples and unflipped refuted. Overflow is found at depth 1; repeated seeded samples contain six-step walks; Leaky is refuted by send(king, donkeykong, banana, -1). Per-input any syntax remains public implementation work.
- Keep conformance beside examples. These are native declaration/replay experiments with string metadata only. Typed expectations, realization resolution, capability paths, Case production, Contracts and server/recorded-Run execution remain unproven.
- Keep oneOf enumeration and exploration-only any. Slug and origin admission now applies to both String and TestOptions. Emitted Case identities remain held.
- Keep questions in test files. No old public query/live/realization API is removed by this task.
- One file works for native execution. Positioned replay carriers are a future seam; exporter/plugin/IR lifting remains held.

Literal Quint sendTest retains supply 10,000, king-to-donkeykong banana 2,000, exact four upstream balance/key assertions, and donkeykong-to-peaches burger 1,000 refusal. Unchanged balances after refusal are additional Scala coverage. Small-bank examples are labeled prototype-authored. The retained TwoPC successTest comparison is recorded as later fn-195.4 work, not an executed equivalence claim; neither upstream copy changed.

Production recommendation for fn-195.2: retain DSL-owned declaration carriers and checker-side execution adaptation. The existing lazy NativeRuntime boundary is a candidate, not a proven complete ModelSuite bridge. Decide munit's ordinary compile dependency and module ownership explicitly. Prototype checker/NativeClaims imports authorize no reverse DSL import. The public implementation still owes unannotated expect blocks under strict flags and Registry inheritance support. Raw E176 diagnostics and read-only compiler-source findings remain at .flow/tmp/fn1951/prototype-fourth.log and .flow/tmp/fn195-prep/expect-warning-seam.md. The narrowly matched authored nowarn annotations are a prototype-only limitation.

The options bypass was first reproduced by two semantic RED controls, committed before the shared admit helper repair, then verified green. Independent read-only source audit/follow-up is .flow/tmp/fn1951/registration-audit.md. It issues no review verdict. The concurrency probe compares two submitted calls without forcing contention; the monitor bounds the implementation.

The first aggregate at source 0b6b7118af was stopped for that repair. Host handle exited 143; owned process groups 699623 and 700483 were confirmed absent. Its logs/builds remain under full-gates-superseded.log and final/. It is inconclusive for final source and earns no pass credit. The repaired aggregate used new repaired-final/ paths, unchanged strict flags and fresh artifacts.

Quick receipt .flow/tmp/green-receipts/641cc2b1-unittest.json was written and then honored, command SHA256 e1b6e25b6a9b4a76c33295999409cfa7e6b59872a7f23dada06a9dcf71c0a6f8. That post-write check skipped no test execution.

Only model/design/runs-in-tests.md and model/design/modelsuite-prototype/** changed before the lifecycle receipt. One read-only helper completed; the worker owned all compiler/JVM and git lanes. Parent remains OPEN, tracker inactive, plan-sync disabled and review deferred. Temporal, real Go bodies, IR/export/lifting/schema/backend/generated/regeneration and held Make routes remain untouched. No publication, Make-green, speedup, Go deletion or public DSL support claim is made.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0b6b7118af6ee668942c7e34648bb371f6b01ca6, 307f56ce455db3aa76d8b342b32d758a145721c7, 641cc2b19875ef730761be5a91e0dccaf4691c43
- Tests: timeout 600 env -u BASH_ENV mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest, env -u BASH_ENV MODELSUITE_WORKSPACE=.flow/tmp/fn1951/repaired-positive-workspace bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/design/modelsuite-prototype/run.sh, env -u BASH_ENV MODELSUITE_WORKSPACE=.flow/tmp/fn1951/repaired-positive-workspace bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/design/modelsuite-prototype/run.sh framework.prototype.WrongExpectationProof, env -u BASH_ENV MODELSUITE_WORKSPACE=.flow/tmp/fn1951/repaired-early-workspace bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/design/modelsuite-prototype/run.sh --early-refusal, env -u BASH_ENV MODELSUITE_WORKSPACE=.flow/tmp/fn1951/repaired-inventory-workspace bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/design/modelsuite-prototype/run.sh --inventory, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1951/verify-repaired.sh, git diff --check
- PRs: