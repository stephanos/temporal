---
satisfies: [R3, R4]
---
# fn-141-shrink-the-ir-generator-one-description.3 Generic definition expansion for function-body sugar

## Description
Part A, first half: the lifter expands a function-level sugar from its Scala definition, with its arguments bound, then lifts core constructs and types. Remove the hand-written expression lowerings for enter, stay, reject, disabled, membership/role in, implies and records, covering all their current function-body overloads. Declaration-level patterns, witness builders and block/rule forms remain explicitly open in .4 and the constructed-value export tasks; they are not permanent exceptions to R3/R4. Current paths and mini-only execution follow MILESTONES.md and the activated parent spec.

**Size:** L
**Touches:** [model/framework/dsl/**, model/framework/export/**, model/framework/inspect/**, model/framework/lint/**, model/framework/check/**, model/temporal/check/**, model/temporal/migrate/README.md, model/README.md, model/SEMANTICS.md]

### Approach
- Re-read export/{Syntax,Expressions,Claims,Context,Lift}.scala, inspect/Index.scala, dsl/Syntax.scala and the existing Sugar.scala fixture. Index the framework and fixture kit definitions as well as Model TASTy while excluding framework definitions from author roots and source lint. Reuse one shared index and existing parameter/type/function bindings.
- Select a generic sugar-definition marker (Syntax.scala membership or a dedicated annotation) from the established syntax contract and record the choice. The expansion and reducers must name no particular sugar. A new fixture-kit sugar expands without changing exporter code or registering its name.
- Preserve the existing IR, including node positions: generated core nodes use the author call's position, while explicit argument expressions retain the positions the existing lowering gives. Reuse the same fixture/golden assertions instead of recapturing a bespoke baseline. Split sugar tests from Fixtures.test.scala beside expansion code.
- Implement generic reductions only as needed: known constructor field access, supplied/default arguments, type/value/function binding, repeated arguments and core collection/boolean/type-test operations. Preserve implication's short circuit and holes, ordered facts, method arity and finite role membership. Do not replace named sugar cases with a renamed whitelist or another hand-written per-sugar dispatcher.
- Composition records must honor the selected member's exact composed fact key. Its current native suffix-only body ignores the selector while its IR uses the member key; reconcile through a core definition/capture mechanism and prove the native answers match the existing IR contract, including same-named facts in different members. If recording selector data needs a capture point, the parent architecture already allows inline/macros there only; keep sugar plain and report the design. No macro may lift a function body, and no arbitrary compiler-name/reflection heuristic is an accepted selector substitute.
- Refuse unliftable sugar definitions with their name and definition position, and direct/indirect recursion with a named cycle. Keep the rejection at the author-facing call where appropriate. Add meaningful focused red/green tests for extension/context/default/type/vararg binding, kit sugar, cycles, errors and source positions; reuse existing controls for forms they already cover.
- Preserve live mini/shared/sample sources and runtime answers, and fixture core oracles including the reserved Bindings mechanism. Real-feature sources/generated outputs remain untouched. A concise migration recipe is required only for an author-visible change; no new Scalafix rule or bespoke proof/capture/hash manifest.
- Baseline: fn-141.2's green .flow/tmp/checks/20261013-031649/record.md (597 tests), with the recorded PropertyGroupsNative classpath race separate. Iterate affected suites, then run make model-check-changed BASE=<task-base> once at end, with a bound matching its documented 8–13 minute range and mini step-table equivalence. A genuinely failed run licenses diagnosis and a warranted rerun; an observation timeout never licenses restarting a live command. Strict generated-IR/full gates, broad lint, independent review, Go and regeneration remain held until the batch boundary.

## Acceptance
- R3/R4: every current function-body overload of the seven named sugar families expands generically from its definition. The expression pipeline contains no named lowering arm, intrinsic masquerading as a sugar-specific fallback, or name-registration requirement for a new kit sugar. Remaining declaration work is recorded for .4/.10-.12 and the final audit.
- Sugar/core fixtures retain the same IR and source positions under their existing comparison contract; a new fixture-kit sugar lifts without an exporter change. Runtime controls include short circuit, ordered facts, role membership and exact composition member identity.
- Unliftable sugar and direct/indirect sugar cycles are refused with the required names and positions. Surviving refusal/golden contracts and mini/shared semantics are preserved.
- The task's model-check-changed record passes with mini step-table equivalence. Before/after source counts, marker/position/binding design, focused results, any migration recipe and deferred gates are in the receipt. No generated payload or real-feature source changes.

## Done summary
Kit authors can add function-body sugar through a plain `@sugar` definition without registering its name in the exporter. Every current function-body overload of enter, stay, reject, disabled, membership/role in, implies and step records now uses indexed definition expansion. Composition records answers the selected member's exact canonical fact key in Scala and IR.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; activated batch policy)

### Design and compatibility

- The dedicated `framework.dsl.sugar` annotation marks defs and the disabled val. A single Index includes framework, kit and Model definitions. Index.trees retains every supplied tree for explicit module lint. Its authorTrees projection and Context.defs use source provenance for the DSL/realize core; author admission receives that same source predicate. Framework-package fixture sources remain eligible for explicit lint. Root discovery and authored refusal contracts remain intact.
- Expansion binds type, value, function, contextual, default and repeated arguments. Generated nodes use the author call position. Argument bindings retain their original placement. Original repeated-parameter types are checked before de-aliasing, preserving the splatted membership refusal.
- Observable by-value arguments use ordered core Let bindings when a body drops, delays, duplicates or reorders their reads. Hole reads, source calls, partial matches and effectful defaults are detected generically. A single unconditional read in caller order can substitute directly, preserving existing IR shapes. By-name arguments remain lazy. Staged expression reads precede raw bindings and constructor-field reduction cannot re-read a staged receiver. Temporary names avoid names already in scope.
- Caller arguments retain their caller expansion stack. Direct and indirect definition cycles name the cycle; a legal nested invocation of the same marker succeeds. An unliftable body names its sugar and definition position.
- Composition records keeps `after.records(_.member, fact)` and placeholder inference. Its thin inline boundary structurally captures the typed path through MemberPath.capture. The capture carries the selected member and fact term so the exporter can validate composition membership and machine fact typing. The plain marked recorded definition builds the key with core valueKey and tests facts membership. There is no records-name lowering or predicate intrinsic.
- Native capture rejects opaque selector parameters and local/object selector vals. The lifter preserves the existing direct-selector admission and compile-then-lift refusal layer, including nested-field diagnostics. Kit definitions can compose a direct selector in their body. Forwarding reusable recorded selector data remains an explicit fn-141.5 obligation. README, SEMANTICS and the migration recipe describe that boundary and exact-member correction.
- SugarFixtures owns the moved existing sugar/core assertions unchanged. FixtureDeclarations shares the original comparator. Coverage registers SugarNativeTest, DefinitionExpansionTest and SugarFixtures separately. Existing expected IR and refusal ledgers were not edited.
- Generic functionArgument binding expands default-getter bodies with supplied types/values before applying their returned structural callback. Named/default callback tests pin the exact callee, argument, arity and equality body. Explicit opaque producers remain refused. Order's existing eligible-witness helper expansion uses authored definitions, preserving its primitive framework-builder recognition after the full index became available.
- Isolated MiniLayout probes use a normally packaged framework-only library through ModelBuild.frameworkJarArgument. Its source-root settings match the packaged Models, keeping framework positions stable while adding no production Model roots. The fixture-only machine assertions are unchanged. A speculative general-scaffold admission change was reverted.

### Source counts

Counts are physical Scala lines at task base and this checkpoint, rather than a claim that the whole infrastructure already shrank.

| File | Before | After |
| --- | ---: | ---: |
| export/Syntax.scala | 1529 | 1382 |
| export/Expressions.scala | 960 | 1024 |
| export/Context.scala | 438 | 455 |
| export/Expansion.scala | 0 | 256 |
| dsl/Syntax.scala | 1011 | 1018 |
| dsl/MemberPath.scala | 0 | 25 |
| inspect/Index.scala | 91 | 103 |
| lint/Admission.scala | 110 | 110 |
| lint/Order.scala | 1221 | 1221 |

Aggregate production counts exclude test files, testdata and project.scala. The new generic expansion infrastructure raises this intermediate total by 200 lines; the seven expression lowerings are removed and the remaining declaration reductions are still open.

| Production module | Before | After |
| --- | ---: | ---: |
| export | 12238 | 12426 |
| lint | 2995 | 2995 |
| inspect | 95 | 107 |
| Total | 15328 | 15528 |

### Verification

- Baseline green via the activated fn-141.2 handoff, `.flow/tmp/checks/20261013-031649/record.md`, 597 tests. The previously recorded PropertyGroupsNative classpath race had already passed its unchanged scoped rerun.
- Meaningful red controls covered marked kit expansion, equal-state/different-member native records, canonical enum facts with fields, opaque selector native/lift parity, and unused/conditional/repeated by-value Hole arguments. The strict red run collected 13 tests and failed those three controls before staging.
- The terminal affected run passed 102 tests with exit 0. Command: `mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.DefinitionExpansionTest framework.export.SugarFixtures framework.export.Fixtures`. Log: `.flow/tmp/fn-141.3-fixtures.log`. It includes the frozen IR/position contracts and complete authored refusal ledger.
- The terminal native run passed 4 tests with exit 0. Command: `mise exec -- scala-cli run --server=false --test model/framework/dsl model/framework/realize --main-class org.junit.runner.JUnitCore -- framework.dsl.SugarNativeTest`. Log: `.flow/tmp/fn-141.3-native.log`. It covers exact members, canonical facts, ordered duplicate facts, implication laziness, strict argument order and opaque selector refusals.
- An intermediate overlapping exporter rebuild caused a genuine RefusalContract ClassNotFound failure. Subsequent same-project runs were serialized and passed; that failure was not waived. New native-test compile warnings were corrected by asserting the returned exception and initializing the selector field before test registration.
- First scoped gate genuinely failed selected syntax lint after 2.3 seconds because the shortened exporter sugar doc omitted the required Core form spelling. Record: `.flow/tmp/checks/20261013-050246/record.md`. Commit 32f0162864 restored the accurate declaration-level Step spelling without changing a gate.
- The warranted scoped rerun exited 2 after 598.3 seconds. Record: `.flow/tmp/checks/20261013-050351/record.md`; log `.flow/tmp/fn-141.3-scoped-rerun.log`. Selected syntax lint, check scalafix, framework boundary, all 3 samples, feature compile/lift, mini baseline step tables and 143 Model tests passed. Lifter suites collected 304 tests with 3 failures. MiniLayoutFixtures omitted framework definitions, ConventionsFixtures lost intentional framework-package trees, and EligibleWitnessFixtures expanded newly indexed primitive framework builders. Check suites were not reached. These failures were corrected in the bounded continuation and verified by the final scoped pass below.
- A later producer regression confirmed a dropped eager Hole in `ignoreFn(makeFn())` (16 tests, 1 intended failure). Function arguments now require a structural lambda or named def; opaque producers are refused generically at their argument. Observable classification exempts structural lambda creation rather than every function-typed term. The focused producer rerun passed all 16 DefinitionExpansion tests with exit 0; logs `.flow/tmp/fn-141.3-producer-{red,green}.log`.
- The first strong callback control exposed a call to selected$default$2 in place of the returned predicate. Generic default-function binding corrected that callee without relaxing the assertion. The continuation's integration run passed 37 tests across DefinitionExpansionTest, MiniLayoutFixtures, ConventionsFixtures, EligibleWitnessFixtures and AdmissionSuite. Log: `.flow/tmp/fn-141.3-integration.log`.
- The final affected/refusal run passed 105 tests with exit 0 using the same DefinitionExpansionTest, SugarFixtures and Fixtures command above. Log: `.flow/tmp/fn-141.3-final-affected.log`. It retains all existing IR, source-position and authored refusal assertions after the continuation fixes.
- Final gate `make model-check-changed BASE=fc3c86f592` passed with exit 0 in 617.4 seconds. Record: `.flow/tmp/checks/20261013-054046/record.md`; log `.flow/tmp/fn-141.3-scoped-final.log`. All 98 selected suites passed, with 143 Model tests, 307 lifter tests and 154 check tests (604 total), all 3 framework samples and mini baseline step-table equivalence. Seventeen real-feature suites were explicitly skipped by scope. Fresh generated mini IR drift remains held under the activated policy; its step-table equivalence passed.

The first timebox ended at a coherent partial checkpoint. One bounded continuation corrected the integration failures, added strict producer/default callback controls, and obtained the fresh terminal scoped pass. No command remains running. REVIEW_MODE=none is an explicit batch-policy skip; this task claims no independent review verdict.

Declaration-level witnesses/rules/blocks remain open for .4 and constructed export .10-.12. Bindings remains reserved for the .3/.4 core oracle, with final removal tracked by .10/.14. No real-feature source, generated payload, schema, Go, tools or Makefile change was made. Strict generated-IR/full gates, broad lint, Go, regeneration and independent review remain held at the batch boundary. planSync=false.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 024c9ddf5eb61d37b15771c93e7da95606c85e2b, 32f01628647e926b1ec6919072727f84513c2b32, 7037a7b8a51a26cb58fecf23a14c75a51491fcad, 601eede9d9fb2c39a82b8992cb57581fa800b5e7
- Tests: mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.DefinitionExpansionTest framework.export.SugarFixtures framework.export.Fixtures, mise exec -- scala-cli run --server=false --test model/framework/dsl model/framework/realize --main-class org.junit.runner.JUnitCore -- framework.dsl.SugarNativeTest, mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.DefinitionExpansionTest framework.export.MiniLayoutFixtures framework.lint.ConventionsFixtures framework.lint.EligibleWitnessFixtures framework.lint.AdmissionSuite, make model-check-changed BASE=fc3c86f592
- PRs: