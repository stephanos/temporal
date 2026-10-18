---
satisfies: [R2, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.2 DSL completion: authored capture, strict expectations and native replay

## Description

Implement the production authored-body boundary that later ModelSuite registration consumes. Retain named action-record calls, fieldless whole actions, finite `oneOf` alternatives, `repeat(count)(step)` and `expect` blocks as finite declaration data associated with one exact Model instance. Capture each expectation's source file, line and original text, and adapt the captured body to the existing native replay engine. The body has the same meaning in future `examples` and `conformance` tests. This checkpoint includes no MUnit facade, section enumeration, Registry changes, search API, realization binding or Case production.

Use `.flow/tmp/fn1952-plan/expect-probe.md` as admitted mechanism evidence. Its public `quotes.reflect` candidate preserves the original caller block's source/span through `Block.copy`, then uses a bounded generated expression annotation for the validated Boolean statements. The selected probe requires neither compiler internals, a compiler dependency nor a plugin. It passes the pinned strict flags when macro main sources compile before an authored `.test.scala` caller, and when the caller uses a separately packaged macro jar. Joint compilation of macro definitions and authored callers as main sources still fails and is outside the established route. Keep that limit explicit in implementation notes and verification; do not make a joint-main success claim.

The probe's assertion helper runs immediately. Production expectations must instead remain deferred until native replay supplies their state and actual preceding step. Verify that adaptation afresh. Its experimental refusals for nested blocks, local definitions and direct control flow describe the tested candidate, not permanent additions to the parent's grammar. The full production Boolean-expression contract remains a proof obligation here. A successful warning probe does not prove native capture, owner admission, restored-state reads or a ModelSuite.

**Size:** M, bounded to one authored-body grammar, its exact-instance context and its existing-checker adapter. The ordinary-MUnit suite/lifecycle integration is the next checkpoint. If retaining deferred source positions requires another compiler facility, preserve the failed evidence and resolve that bounded capture issue here before expanding scope.

**Files:** `model/framework/dsl/RunBody.scala` and `Expect.scala` with their focused `.test.scala` controls; the minimum scoped context integration in `model/framework/dsl/Syntax.scala`, `ClaimWords.scala` and `NativeRuntime.scala`; `model/framework/checker/Native.scala` as the sole NativeRuntime implementation; `model/framework/checker/SuiteExecution.scala` and `SuiteExecution.test.scala` for translation to existing replay operations. S later extends the same adapter and therefore follows P serially. Edit existing replay internals only if a demonstrated integration defect requires it.

**Touches:** [model/framework/dsl/RunBody.scala, model/framework/dsl/RunBody.test.scala, model/framework/dsl/Expect.scala, model/framework/dsl/Expect.test.scala, model/framework/dsl/Syntax.scala, model/framework/dsl/ClaimWords.scala, model/framework/dsl/NativeRuntime.scala, model/framework/checker/Native.scala, model/framework/checker/SuiteExecution.scala, model/framework/checker/SuiteExecution.test.scala, model/framework/checker/Replay.scala, model/framework/checker/Questions.test.scala, model/framework/REFERENCE.md]

**Dependencies:** Existing fn-195.1 closure `d0d2efeab0` and the already admitted fn-194/fn-196 native checkpoints. Keep existing `.2 -> .1`. This task produces the body/request seam consumed by S; it does not require whole fn-194/fn-196 completion or their held proofs. Trust the existing prerequisite admission rather than repeating it.

## Investigation targets

- `model/framework/dsl/Syntax.scala` for resolved claim-word symbols, step context and current placement refusals.
- `model/framework/dsl/ClaimStatements.scala` for lazy assertion/context callbacks and exact native runtime forwarding.
- `model/framework/dsl/NativeRuntime.scala` for the DSL-owned execution boundary and checker-required failure.
- `model/framework/checker/Native.scala` for the sole NativeRuntime implementation and narrowly delegated suite-request execution.
- `model/framework/checker/Replay.scala` for retained alternatives, positioned operations, cuts and failure evidence.
- `model/framework/checker/Spaces.scala` for restored-state expectation evaluation and actual preceding-step context.
- `.flow/tmp/fn1952-plan/expect-probe.md` for the public Block.copy route, legal compilation boundaries and remaining production grammar/replay proofs.

## Replay input evaluation

The initial adapter demonstrated an integration defect: materializing target thunks before replay
restores a frontier state binds named action inputs against the ordinary live fields, including
externally mutated values. Preserve the authored sequence's meaning by evaluating captured action
targets at their actual replay step under the exact restored owner state. Use a minimal lazy-target
operation inside the existing replay interpreter, preserving the eager ReplayStep API and existing
class selection, branching, failure/cut handling and evidence. This is a scoped existing-engine
extension, not another search algorithm. The conductor admitted `Replay.scala` and
`Questions.test.scala` to Touches for this demonstrated need. Establish the reached existing replay
baseline before editing it and retain its behavior controls.

## Import and execution boundary

Declaration carriers, macros and finite execution requests belong to `framework.dsl`. They may reference the existing generic realization declarations within the current DSL/realize module boundary; this task needs no new realization behavior. They import no `framework.checker`, MUnit, export/lifter, gate or system-specific module. Ordinary quoted macros use `scala.quoted` public APIs and the standard annotation type only. Add no `scala3-compiler` dependency or compiler-plugin route.

The product checker translates the DSL-owned request into its existing replay operations and evaluates it. Reuse or narrowly extend `NativeRuntime` rather than adding a second engine or exposing checker implementation types through the DSL. Declaration construction and loading with DSL/realize alone must not execute model callbacks, resolve a checker, search or require a checker class. An actual execution without the checker retains the existing explicit runtime-required failure.

## Public reference coverage

The existing ReferenceSurfaceTest demonstrated that this checkpoint's new public names need reference coverage before the native boundary gate can pass. Add the minimum accurate author-facing explanation of captured native bodies, and identify signature-only declaration carriers in the existing public-surface appendix. Document `repeat`, `RunBody` and the native execution seam without claiming the future ModelSuite facade or conformance is implemented. Keep REFERENCE.md free of Scala-file references; use public forms and behavior, not implementation navigation. This bounded coverage is part of the API change, not the later comprehensive .5 rewrite. Do not weaken or bypass the public-surface gate.

## Acceptance

- [ ] The admitted body grammar captures named record action calls and fieldless whole actions in order, preserves each `oneOf` alternative, and uses the prototype-selected `repeat(count)(step)` form. Execution retains the existing sole-start default or explicit `from`; omission with multiple init states refuses at the authored body/test location. Unsupported statements and one-alternative `oneOf` refuse with useful positions. No arbitrary Scala statement lifting or compatibility arithmetic spelling is introduced.
- [ ] Unannotated `expect` blocks compile under every unchanged strict flag in the legal source-test route and against a fresh separate declaration jar. Every assertion statement covered by the parent's Boolean contract, including literals, is independently retained in source order and evaluated once when replay reaches it. A false first or middle condition prevents later condition evaluation and reports its original file/line/text plus concrete state. Boolean short circuit remains ordinary Scala behavior. Empty/non-Boolean bodies and actual statements outside the parent's admitted subset refuse with useful locations. Establish production behavior for nested blocks and Boolean conditional/match/try expressions; the probe's experimental refusals alone do not authorize dropping those forms. A nested false condition followed by true must never disappear. Any refusal needs an explicit source-backed grammar reason and a matching admitted neighbor, rather than a new restriction inferred from the scratch candidate.
- [ ] Deferred expectation evaluation reads the owning Model's restored state after the actual last step. Typed `before`, `outcome` and recorded-fact access use that exact step and actual owner-bound context, including a refused outcome with unchanged state and an action that records facts. Same-class foreign owner-bound accesses/roots and incompatible fact types remain refused. Preserve the existing `G <:< F` fact-type admission; compatible shared fact values remain legal and gain no invented instance identity. Replaying a second instance cannot change retained evidence from the first. Native zero-step expectation behavior remains explicit and supplies no invented preceding step. Existing claim/effect placement rules remain effective outside this run-body context.
- [ ] The checker adapter preserves existing replay results and evidence for success, disabled steps, thrown effects, reached holes, expectation failures and budget cuts. It retains every alternative and source position while respecting the supplied caller budget; a cut never becomes verified. Capture does not eagerly evaluate callbacks, read model vars or execute effects, and nested/exceptional context restoration remains correct. It introduces no new checker verdict or search algorithm.
- [ ] DSL/realize-only declaration compile/load and execution-without-checker controls demonstrate the import boundary. Public source-test and fresh separate-library clients cover real imports, qualification and aliases; an unrelated shadowing `expect` retains its original meaning. Unrelated discarded Booleans outside `expect` and a deprecated Boolean call inside it still fail strict compilation. Single-Boolean/single-literal bodies generate no unused-suppression warning. The unsupported joint-main route remains documented with its original failing evidence.

- [ ] Named action inputs that read the bound owner's vars evaluate once in the actual restored
  frontier state after prior steps, including each repeat/alternative. Capture evaluates no input
  callbacks. External mutations of the owner's ordinary live fields before replay cannot change
  the selected path inputs. Original eager ReplayStep callers retain their behavior, ownership,
  disabled-step, hole, exception, alternative and budget/cut evidence.

## Verification

Use the then-current framework README Native checks recipe with focused capture/context/replay suites, pinned toolchain, existing IR jar, unchanged strict flags and nonzero executed tests. Exercise normal macro-main plus legal `.test.scala` source compilation, then package the declaration library and compile/run separate public clients in a fresh workspace. The production matrix must validate deferred replay and the full admitted Boolean-expression grammar, rather than reusing the immediate assertion probe as a passing replay receipt. Include the nested false-then-true adversarial case and parent-covered Boolean control-flow expressions, with no silently discarded condition.

Record exact commands, exit statuses, test counts, source positions and the intended negative diagnostics. Preserve failed attempts and the probe's joint-main limitation. Re-run the relevant existing claim-context and native replay controls when their sources change. Corpus-wide section lint remains .3's work; this task supplies located local macro/API refusals for R7.

## Coverage and held work

This checkpoint implements R2's authored body/start behavior and local R7 refusals. S consumes the proved request/adapter seam for public registration and caller budgets; X and C retain their separate exploration and conformance obligations. Existing .3 owns final corpus lint and retirement.

Parent holds remain unchanged. Native positioned declarations earn no actual lift, Contract, emitted Case or server/recorded-Run proof; mixed parents remain OPEN and review deferred.

## Done summary
# fn-195.2 handover

Implemented finite DSL-owned authored bodies, strict deferred Boolean assertions and exact-owner typed step context through the existing lazy NativeRuntime and native Replay engine. Native replay now evaluates named inputs once under each restored frontier, retains every finite alternative, preserves decisive failures through subsequent holes and keeps the original eager ReplayStep API.

Baseline: green, canonical Quick executed 4 tests before edits; base `16c863d897f4bce7b163efe6b535e1d2ba6578f9`. Source checkpoints: `455f2ecc39c9360e6ef377f95193b70377c1fa05` and `b17ac345454b23a25b6f3efdb8bd33c86c70e0d2`.

Tier: session (jev-unavailable(no_key))

stage: impl-review - skipped(config: REVIEW_MODE=none; policy: owner phase-batch review deferral)
stage: plan-sync - skipped(config: planSync=false)

### Acceptance evidence

- Grammar, exact resolved combinator admission, inherited/qualified/aliased words, positive parameter repetition, typed refused outcome/facts, compatible shared facts and foreign-owner refusals: six `RunBodyTest` controls. Arbitrary carrier/helper statements do not enter capture.
- Deferred Boolean semantics and original nested failure positions: four `ExpectTest` controls cover literals, first/middle failures, nested blocks, conditional/match/try/catch/finally/operand forms, guards and ordinary short circuit. Late hole precedence has executed RED `hole-decisive-red.log` and GREEN `hole-focused-green.log`, with explicit source-line rerun `hole-line-source-green.log` and packaged rerun `hole-line-jar-green.log`.
- Restored inputs, per-frontier branching, starts/zero-step, long-repeat cuts, lazy/eager disabled/effect/throw/hole comparison, nested/exceptional restoration and second-instance retained evidence: ten `SuiteExecutionTest` controls. Mixed input-hole siblings execute known-first/hole-first/pass-plus-hole rows; `alternatives-hole-red.log` records the intended failure, `alternatives-hole-green.log` passes the final 97-test focused suite, including existing eager `QuestionsTest` and existing claim/context suites.
- Legal public source-test and fresh separate-library strict matrix: `clients-run-v3.log` plus `clients-v3/jar-refusals/evidence.md` record loading 11, absent-checker 4 and shadowing 3 controls in each route, plus seven intended strict refusals per route (14 total). Outside discarded Booleans, deprecated Boolean calls, empty/non-Boolean bodies, one-alternative declarations and helper bypasses retain their actual compiler failures. These DSL controls are reused, not claimed as a fresh final generation: all 69 DSL/realize main source hashes match checkpoint455 and the declaration jar is unchanged.
- Final repair generation: `repair-run-v2.log` records actual exit0 for canonical Quick4, ReferenceSurface4, fresh native package, packaged public20 and jar identity verification. `repair-v2/unchanged-declaration-sources.log` and `unchanged-declaration-jars.sha256` record the exact scope and identities.
- Public reference coverage was added only after the existing ReferenceSurface gate failed: original `final-run.log` stopped at boundary-loop (65/66 passing); its raw failure is retained. REFERENCE documents current capture/native forms and signature carriers without claiming future ModelSuite/conformance integration.
- Final committed snapshot `b17ac34545`: `final-run-v2.log` is terminal exit0 with exactly 20 stage exits, all0. The fresh native recipe passes declarations/core/plugin, native-full693, Quick4, class admission7, rule suites31, boundary-loop66, source lint178 classified, sample source/client, unchanged realization fixture comparison, native realization source/library, sample library, questions source/library/client2, voting lifetime and final diff-check. Voting lifetime runs 64 instances uninitialized plus64 initialized (128 total), one round each, zero survivors; it is native lifetime evidence, not system execution. The disposable FrameworkLoop fixture retains its explicit held cases.
- `final-artifacts.sha256` binds command scripts, logs and generated jars; `final-source.sha256` binds all changed source/test/reference/task paths. The source was clean and matched b17 after all gates. Canonical Quick green receipt: `../green-receipts/b17ac345-unittest.json`. No gate was skipped or weakened. `ac-evidence-gaps.md` maps all six ACs plus bounded reference coverage; its final-verification pending statements are superseded by this terminal observation.

### Artifact identities

- Unchanged declaration jar: `clients-v3/declarations.jar`, SHA256 `6ce808fe1dac873ef85d1269492c784b6f94fdff8abeaadb541f9d6c91c4dbee`.
- Final repaired native jar: `repair-v2/native.jar`, SHA256 `9d21138d14cf087e901cafa57ec8fe9d3ac58bf51304e23023dea49dfd336863`.
- Existing unchanged IR dependency: SHA256 `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`.

### Quality and boundaries

Original independent-axis reports are preserved in `quality-reports.md`, `correctness-review.md` and `standards-review.md`; the mixed-alternative follow-up remains separate. The two correctness hole-precedence findings were reproduced before their minimal fixes; standards reported no findings. Both narrow rechecks of `455f2ecc39..b17ac34545` report no remaining actionable findings: `correctness-final-review.md`, `standards-final-review.md`, aggregated verbatim in `quality-final-reports.md`. These read-only audits inspected executed evidence and are not formal SHIP reviews.

The public Block.copy mechanism uses no compiler dependency/plugin or authored suppression and no weakened strict flags. The original joint-main macro/caller route remains unsupported with its failing evidence at `../fn1952-plan/expect-probe/public-final-joint-main.log`; legal source-test and separate-library routes are the supported proofs. Initial state acquisition remains lazily based on live fields; mutation tests explicitly retain init0 before mutation. No instance identity is invented for transferable state or compatible facts.

No MUnit facade, ModelSuite registration, exploration/conformance/Case, actual lift/export/IR regeneration, Go/system execution, held corpus access or actual repository Make execution was added. The authorized final native recipe retains its existing native plugin admission stages and disposable FrameworkLoop Make-n fixture only. Parent remains OPEN and formal review deferred.

Failed/compiler-crash/configuration attempts remain raw logs, not green receipts. The unconfigured final fmt attempt selected the wrong formatter configuration; `format-check-final-configured.log` checks all 11 touched Scala files with `model/.scalafmt.conf` and passes without source changes.
## Evidence
- Commits: 455f2ecc39c9360e6ef377f95193b70377c1fa05, b17ac345454b23a25b6f3efdb8bd33c86c70e0d2
- Tests: timeout 600 env -u BASH_ENV mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest, timeout 600 env -u BASH_ENV mise exec -- scala-cli run --test --server=false --workspace .flow/tmp/fn1952/alternatives-green-workspace model/framework/checker --main-class org.junit.runner.JUnitCore -- framework.checker.SuiteExecutionTest framework.dsl.ExpectTest bodycontrol.RunBodyTest checkercontrol.QuestionsTest framework.dsl.VarMachinesTest framework.dsl.ClaimShortCircuitSuite framework.dsl.NativeClaimHolesSuite framework.dsl.ParametricMachinesTest, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1952/clients.sh, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1952/reverify.sh, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1952/verify.sh, timeout 600 env -u BASH_ENV mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl/RunBody.scala model/framework/dsl/RunBody.test.scala model/framework/dsl/Expect.scala model/framework/dsl/Expect.test.scala model/framework/dsl/Syntax.scala model/framework/dsl/ClaimWords.scala model/framework/dsl/NativeRuntime.scala model/framework/checker/Native.scala model/framework/checker/Replay.scala model/framework/checker/SuiteExecution.scala model/framework/checker/SuiteExecution.test.scala, sha256sum --check .flow/tmp/fn1952/final-artifacts.sha256, sha256sum --check .flow/tmp/fn1952/final-source.sha256, git diff --check
- PRs: