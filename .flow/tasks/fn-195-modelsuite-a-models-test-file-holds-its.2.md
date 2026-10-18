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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
