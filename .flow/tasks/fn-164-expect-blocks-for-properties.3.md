---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-164-expect-blocks-for-properties.3 Lift framework expectation predicates with exact core IR and ordinary-function bridges

## Description
**Note 2026-10-18 (one-language series):** Paths `framework/export/*` become `framework/lift` (fn-201); closes with fn-190.9 as recorded.

**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Runs after fn-190 phase 1 (fn-190.7) against the final DSL, batched with the other IR/plugin work; one full framework gate at the end. The source export and plugin capture of expectation blocks are done (ExpectFixtures, ExpectCaptureSuite); the bridges and exact composed facts are parked on branch `work/fn1643-bridges`. fn-190.4 retires the public `expect` surface for `check` blocks under `invariant:`/`transition:`/`after(action):`, so the remaining lowering is re-targeted there in fn-190.9 and this task closes with it.

Source inspection informs this task; no implementation, compilation, tests or acceptance evidence is claimed by this plan.

**Dependency:** `fn-164-expect-blocks-for-properties.2`, with its verified closure committed in `umpire` history. Consume `.2`'s actual native API, typed metadata, validator and contextual-name receipts rather than the earlier proposal's unproved carrier spelling. fn-188.2's committed isolated runner is already a prerequisite of `.2`. Existing fn-140 witness and fn-149 grouping source receipts are admission evidence to preserve, not whole-spec dependencies or permission to inspect excluded sources.

**Framework-only execution amendment:** Work only in `model/framework`. Do not read, compile, migrate or restore `model/temporal`, its disabled sources or generated payloads. Do not use `model/project.scala`, Temporal Model/API jars, Go, schemas, backend work or regeneration. Generic temporal logic remains in scope: ordinary framework claim patterns, Queries, progress forms and monitors retain their semantics. That mathematical vocabulary does not authorize Temporal system integration. Completing this task contributes framework proofs to R1-R5; it does not complete the full spec or held Temporal R6/R8 obligations.

**Size:** M, with an early API/capture proof that must succeed before widening the fixture set.

**Files:** shared expectation-body recognition and interpretation, source-aware core Property registration, ordinary-function capture metadata, focused paired fixtures and their generic-suite registration.

**Primary touches:**

```text
model/framework/reify/body/BodyContext.scala
model/framework/reify/body/BodySyntax.scala
model/framework/reify/body/Expressions.scala
model/framework/reify/body/BodyEngine.scala
model/framework/export/Claims.scala
model/framework/dsl/Syntax.scala
model/framework/export/Expect.test.scala                    (new ExpectFixtures)
model/framework/export/ExpectCapture.test.scala             (new ExpectCaptureSuite)
model/framework/export/testdata/expect/**                   (new isolated generic fixtures)
model/framework/check/FrameworkLoop.scala                   (generic suite registration)
model/framework/check/FrameworkLoop.test.scala              (selection regression)
```

**Conditional touches, only when a concrete proof reaches them:** `reify/plugin/{ReifyPlugin,PluginContext}.scala`, `export/{Context,Declarations,Capabilities,Compositions,Syntax,Constructed}.scala`, `dsl`'s `.2` predicate/selector metadata seam, `export/ModelBuild.test.scala`, `export/InProcess.test.scala`, and `check/lint/{SyntaxRule,SyntaxRule.test}.scala`. Extend an existing shared seam when needed; no blanket rewrite or new parallel capture/export pipeline.

No production caller migration or public finisher removal belongs here. `.7` migrates framework callers, `.8` removes both public lambda finishers and finishes guidance, and `.9` completes framework colon enforcement. Those framework tasks proceed independently of held original `.4`/`.5`/`.6`/`.1`.

### Approach

### 1. Consume and prove the native/capture contract first

Read `.2`'s committed implementation, tests and receipts. Pin the accepted builder heads (`after`, distinct `afterAction`, `.across expect`, unrestricted `property expect`), standalone same-step/transition constructor spellings and exact contextual binding forms. The reusable results must have the ordinary function types `Step[S,O,F] => Boolean` and `(S,Step[S,O,F]) => Boolean`, not a second predicate AST or a second handwritten Boolean implementation.

Compile and execute a minimal same-step and transition fixture through the actual reify plugin, then export the same complete declaration through the source path. Prove that compiler expansion preserves the original expectation body and contextual source symbols long enough for the shared interpreter to recover them. Explicit names `s`, `(b,a)`, ordinary `before`/`after`, compiler-generated placeholder names and named helper forwarding all need exact core parameter recovery. A runtime carrier is not itself `S` or `Step`; map its source identity to the emitted parameter without inventing uniform names or requiring new author vocabulary. If `.2` left insufficient metadata, repair that smallest shared seam before adding migration callers or comparison exemptions.

Extend the current block recognition deliberately. `BodyContext.blockCall/blockVal` and `Expressions.blockFunction` currently treat `is`/`effect` as section values and invent `s`; adding `expect` to that set alone is incorrect. Expectation arguments also occur inline, in named helpers, in generic capability constructors and under compositions. Add function-valued block recognition at the shared body/capture boundary without weakening the existing placement rules of unrelated forms.

### 2. One source-aware interpretation, ordinary Boolean core IR

Use the shared `reify/body` engine for compiler capture and source export. Lower the validated requirements to existing Boolean/conditional/Let/Call expressions, with no IR schema change and no runtime control-signal or exception node. Keep source-aware claim registration, declaration naming and restrictions on the current core path.

Preserve exact expression structure, not just truth values:

- Conjunctive requirement order and Boolean association remain exact. Do not insert identity literals (`true &&`, `false ||`), flatten association or remove wrappers in the comparator.
- Conditional requirements without an else retain the old implication core `!c || r`. Conditionals with an else retain the actual core conditional shape where the old twin has one. A small `check(if ... then ... else ...)` can preserve an existing core conditional; do not change the accepted vocabulary.
- `oneOf` lowers explicit grouped `branch` requirements to the corresponding OR tree, preserving conjunction grouping within each branch and written short-circuit order.
- Pure immutable Boolean helper lets remain eager, once-only and in their original lexical placement. Do not sink a preexisting `restarted` let into a selected branch, delay it past a failing requirement, duplicate it, or discard an unused-but-observable read.
- A pure single-check forwarding body retains the original referenced function and its binding identity instead of introducing a wrapper. Preserve named-def references, call grouping, generic type substitution, parameter order/names/types and the ordinary function metadata retained by capture. A comparator that dereferences only the body hides extra functions and is insufficient.
- Bare getters read `after.state` in both predicate forms. Only the transition form has before-state evidence. Preserve exact-class `after` and whole-action `afterAction` restriction metadata, including String and Composed action cases.

The full function/type/declaration inventory is part of the oracle. Preserve `Captured.root`, its reachable `model.functions`/`model.types`, ordinary callable identity, transitive callees, parameter identities, declaration/function references, source positions and any origin/call-group metadata retained by the existing capture seam. Do not add annotations whose only purpose is to exempt expectation functions from existing capture validation.

### 3. Establish the strict early paired fixtures

Create generic synthetic fixtures under `export/testdata/expect`, without reading the held system examples:

- Heartbeat-shaped same-step fact AND implication trees, including restricted and unrestricted heads.
- Reset-shaped transition predicates with the eager `restarted` helper let, six alternatives on one path and three on the other, preserving the old conditional/branch structure.
- Explicit `s`, `(b,a)` and compiler-generated source parameter names; inline and named predicate forwarding; nested conditionals and branch conjunctions.

Each expectation fixture has an independently authored ordinary-core twin. `.3` may consume existing core lambda finishers temporarily, but prefer direct core construction for the independent twin where practical so `.8` can remove the authoring methods without losing the oracle. Do not derive the core twin by calling the expectation implementation.

Compare complete native-captured and source-exported Models against the core twins: declaration/function/type inventories, names, references, parameters, expression trees, metadata and explicitly mapped positions. When paired fixture roots necessarily have different namespaces, use a finite, explicit bijective declaration-root map; do not normalize parameters/locals or helper identities. Check the positions against an authored source-span map first, then apply only that permitted mapping for structural comparison. Whole-tree position erasure without a source-position oracle, a structural fingerprint, an operator/fact substring check or a runtime truth table is not this proof. Add negative controls that change association, a parameter, a let position, an extra wrapper function, selected fact/member or a source span and prove the comparator rejects each change.

Stop at a failed native/capture/paired proof and repair the API or shared interpretation. Do not migrate callers or weaken equality to make the fixtures pass.

### 4. Ordinary-function bridges and level protection

Construct, capture and lift inline and named same-step expectations used by `never`, `stays(...).unless`, `sticky` and `Monitor.readAfter`, and transition expectations used by `stickyAcross`. Retain ordinary state predicates for `once`, `stays`, `from` and `leadsTo`; expectation support does not turn a step predicate into a state predicate. Include reusable helpers over generic `Declares[S]`, capability constructors and composed states, rather than requiring machine-only `Owner` evidence. Query verify/find and progress/monitor declarations remain outside the predicate; no new temporal operator is introduced.

Keep `.2`'s compile protection effective through lifting. Cover forbidden path/temporal declarations and step construction directly, locally bound, discarded, nested in guards/checks/helper RHSs and hidden through reached source helpers, including a machine with inherited givens. Include before-state access in a same-step block, assignment-shaped assertions, final/nonfinal discarded Booleans, unsupported statements, empty block/branch, helpers-only blocks, insufficient alternatives and misplaced branches. Compile refusals must remain compile refusals where required by the spec; an exporter-only failure is additional coverage, not a replacement. Assert physical authored positions and no written output on refusal. Preserve conditional-only nonempty blocks and the complete short-circuit/hole/exception behavior already proved natively in `.2`.

### 5. Exact composed facts and current witness preservation

Use `.2`'s exact authored field plus selected-machine evidence, with the selected machine fixing the fact type. Prove the native and lifted Boolean fact query and Unit requirement agree. Cover same-named facts, members with equal state types, a wrong field/machine pair, repeated instances of the same machine in distinct fields, foreign fact compile refusal and opaque selector refusal where the current capture contract requires a direct field. No suffix matching, all-member widening, `Any` fact erasure or unique-machine heuristic may establish identity.

An owner-free ordinary Step function cannot derive composition ownership from Step alone. Preserve its retained selector field/machine evidence and prove validation on attachment to the declaring composition, including monitor attachment where relevant. Do not credit source-only owner validation as native runtime evidence. If `.2` cannot prove this contract, resolve it before completing the bridges.

Consume the actual current witness receipts and generic witness fixtures. Preserve present/absent primary polarity, every ordered required/forbidden fact guard, selected member, condition helper identity and generated ordinary Property/Scenario/Query names and metadata. Do not recreate retired private triples. Compare complete witness declarations/functions and source mappings against their predecessor, not merely the operators or names they contain.

### 6. Generic expansion and isolated suite integration

Preserve current definition expansion for marked helper/pattern/monitor sugar. Expectation control execution does not justify an independent permanent exporter name-dispatch registry. Keep source/capture interpretation in the shared block seam, document any remaining construct-level handoff explicitly for fn-141, and record the accepted API and exact preservation fixtures that it must consume. Every added public sugar definition remains in the framework sugar file with `Core form:` documentation; core files must not import sugar.

Use the established `ModelBuild.frameworkLibrary`/`reifyPlugin`/`classpathEntries` and `InProcess.frameworkLift` seams for new generic fixtures. Source copies and directive rewrites must preserve authored offsets. Build only against the current isolated runner's fresh `framework.jar`, `reify.jar`, classpath and the existing generic IR jar. Do not fall back to stale `model-scala.jar`, `api-scalapb.jar`, system fixture roots, `ModelBuild.mini` or the held generic `Fixtures` suite that contains system-backed data.

Register the exact new suite names `framework.export.ExpectFixtures` and `framework.export.ExpectCaptureSuite` in `FrameworkSuites.genericExport` and add a selection regression. In the touched `FrameworkLoopSuite`, also assert the current repository's `Boundary.findings(root, Seq("temporal", "io.temporal.api"))` is empty, so the focused task command checks the real boundary as well as synthetic classifier/poisoning controls. The runner intentionally rejects unclassified exporter suites; an ignored or held new suite cannot earn acceptance credit. No broad exception or classifier default is allowed.

### Investigation targets

**Required:**

- `.2`'s committed `dsl/Syntax.scala`, expectation validator/metadata and native `Expect` suite; `dsl/{Claims,Machine,Monitor,Assume}.scala` for the ordinary core signatures and generic declarer types.
- `reify/body/BodyContext.scala` (`BlockCall`, `blockVal`, parameter/function identities), `BodySyntax.scala` (existing statement interpretation), `Expressions.scala` (`blockFunction`, `functionOf`, `giving`/`wrongLevel`) and `BodyEngine.scala` (`stepFunctionOrigin`).
- `export/Claims.scala` (core registration and generic callback binding), `DefinitionExpansion.test.scala`, `ReifyBoundary.test.scala`, `ConstructedRules.test.scala`, `SourcePositions.test.scala`, and the current generic witness cases in `Fixtures.test.scala`.
- `check/FrameworkLoop.scala`/`.test.scala`, `export/ModelBuild.test.scala`, `export/InProcess.test.scala` and `export/FrameworkFixtureSettings.test.scala` for fresh generic artifacts, classification and positioned fixtures.

**Conditional:** `reify/plugin` and the generic `export` context/composition/capability seams reached by the proof. No system-tree investigation targets or legacy `model/irgen`/`model/check` paths apply.

### Verification commands

Follow `MILESTONES.md`'s owner cadence: focused touched suites during iteration and at this task's closure; the full framework gate and independent review run once at source-batch end. The current runner has no focused-suite flag. Do not run its complete suite set merely to refresh compiler artifacts.

For each changed source checkpoint needing capture/export verification, build fresh core/plugin artifacts into an explicit new task-local directory using the same inputs and options as `FrameworkLoop.check`. The actual `ModelSources.frameworkCore` is exactly `model/framework/dsl model/framework/realize`. This command block is a documented direct route, not a new permanent runner. It serializes the two builds, retains their logs and validates the returned classpath before any exporter fixture can run. Scala CLI's printed compiler errors fail even if its process exits zero, matching `Tools.Ran.orFail`.

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
set -e
unset FEATURES BASE INTERPRETER_PARITY SAMPLE_REPORT_INPUT UMPIRE_LIFTER_UPDATE UMPIRE_FRAMEWORK_BUILD
FN164_FRAMEWORK_BUILD="$(mktemp -d "$PWD/.flow/tmp/checks/fn164.3-focused.XXXXXX")"
test -s model/build/ir-scalapb.jar
mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning --print-class-path --compilation-output "$FN164_FRAMEWORK_BUILD/framework.jar" --scalac-option -sourceroot --scalac-option "$PWD/model" model/framework/dsl model/framework/realize > "$FN164_FRAMEWORK_BUILD/core.raw-classpath" 2> "$FN164_FRAMEWORK_BUILD/core.stderr"
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/core.raw-classpath" "$FN164_FRAMEWORK_BUILD/core.stderr"; then exit 1; fi
test -f "$FN164_FRAMEWORK_BUILD/framework.jar"
test -s "$FN164_FRAMEWORK_BUILD/framework.jar"
IFS=: read -r -a FN164_FRAMEWORK_CP_ENTRIES < "$FN164_FRAMEWORK_BUILD/core.raw-classpath"
test "${#FN164_FRAMEWORK_CP_ENTRIES[@]}" -gt 0
for FN164_FRAMEWORK_ENTRY in "${FN164_FRAMEWORK_CP_ENTRIES[@]}"; do
  test -e "$FN164_FRAMEWORK_ENTRY"
  case "$FN164_FRAMEWORK_ENTRY" in
    */api-scalapb.jar|*/model-scala.jar|*/model/temporal/*) exit 1 ;;
  esac
done
FN164_FRAMEWORK_CP="$FN164_FRAMEWORK_BUILD/framework.jar"
for ((FN164_CP_INDEX=1; FN164_CP_INDEX<${#FN164_FRAMEWORK_CP_ENTRIES[@]}; FN164_CP_INDEX++)); do
  FN164_FRAMEWORK_CP+=":${FN164_FRAMEWORK_CP_ENTRIES[FN164_CP_INDEX]}"
done
printf '%s\n' "$FN164_FRAMEWORK_CP" > "$FN164_FRAMEWORK_BUILD/framework.classpath"
mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --assembly --preamble=false --main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 model/framework/reify/plugin -f -o "$FN164_FRAMEWORK_BUILD/reify.jar" > "$FN164_FRAMEWORK_BUILD/plugin.log" 2>&1
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/plugin.log"; then exit 1; fi
test -s "$FN164_FRAMEWORK_BUILD/reify.jar"
test -s "$FN164_FRAMEWORK_BUILD/framework.classpath"
```

Preserve the directory path in the task's evidence. It is the fresh core/plugin build for the following focused checks. After changing `dsl`, `realize`, shared body or plugin inputs, rerun that narrow build block into another new directory; a full gate is not a prerequisite. If only test/exporter code changes and the frozen core/plugin inputs are checksum-identical, this exact directory may be reused. Never select an old directory by a glob or omit `UMPIRE_FRAMEWORK_BUILD`: the exporter harness otherwise defaults to system artifacts. This matches `FrameworkLoop`: validate the existing raw entries and regular fresh jar, then write `(freshCoreJar +: rawEntries.drop(1))`. The pinned compiler's first raw entry is a classes directory, so retain it only in the raw receipt and replace it with the fresh jar in the fixture classpath.

Run the touched suites with that explicit fresh environment. Redirect each JUnit command to its task-local log so its compiler diagnostics and positive executed counts can be checked, even on a zero process exit:

```sh
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli run --test --server=false model/framework/export --main-class org.junit.runner.JUnitCore -- framework.export.ExpectFixtures framework.export.ExpectCaptureSuite > "$FN164_FRAMEWORK_BUILD/expect-tests.log" 2>&1
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli run --test --server=false model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.lint.SyntaxRuleSuite framework.check.FrameworkLoopSuite > "$FN164_FRAMEWORK_BUILD/check-tests.log" 2>&1
for FN164_TEST_LOG in "$FN164_FRAMEWORK_BUILD/expect-tests.log" "$FN164_FRAMEWORK_BUILD/check-tests.log"; do
  if rg -q '\[error\]' "$FN164_TEST_LOG"; then exit 1; fi
  rg '^OK \(([1-9][0-9]*) tests?\)' "$FN164_TEST_LOG"
done
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only '*Expect*' > "$FN164_FRAMEWORK_BUILD/native-expect.log" 2>&1
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/native-expect.log"; then exit 1; fi
```

For each JUnit command, reject any `[error]` diagnostics and require `^OK \(([1-9][0-9]*) tests?\)` in its retained output, matching the framework runner's nonempty-test contract. The native `test --require-tests` command must report a positive executed count for its expectation suite too; a selected-but-ignored suite earns no credit. Add `framework.export.DefinitionExpansionTest`, `ReifyBoundarySuite`, `WitnessFixtures` and/or `SourcePositionsFixtures` to a focused exporter command when their shared implementation surfaces were touched, and run that affected set at task closure. Do not eagerly run unrelated suites after an unchanged focused pass.

The task receipt records the fresh artifact directory, input revision/diff or checksums, exact commands, exits, positive counts, strict paired-comparison evidence and deferred batch gate. `FrameworkLoopSuite` supplies real boundary and classification/poisoning checks; `ExpectFixtures`/`ExpectCaptureSuite` prove their fixture classpaths contain only current generic artifacts. When fixtures run their own compile/JVM commands, inspect their recorded arguments for the same isolation. Scratch comparison output is permitted; managed Model IR/Cases remain unchanged. No root equivalence/step-table target or held system-backed `Fixtures` suite is required.

At source-batch completion, run the existing canonical gate once, serially:

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
make check-framework
```

That final gate rebuilds artifacts, checks the boundary and all discovered suite classifications, and runs nonempty generic core/check/export suites. Its printed record directory, counts and explicit held integration counts belong to the batch receipt. Until then, this task's receipt explicitly says the full gate and independent review are deferred; it claims only the focused checks actually run.
### Approach

### 1. Consume and prove the native/capture contract first

Read `.2`'s committed implementation, tests and receipts. Pin the accepted builder heads (`after`, distinct `afterAction`, `.across expect`, unrestricted `property expect`), standalone same-step/transition constructor spellings and exact contextual binding forms. The reusable results must have the ordinary function types `Step[S,O,F] => Boolean` and `(S,Step[S,O,F]) => Boolean`, not a second predicate AST or a second handwritten Boolean implementation.

Compile and execute a minimal same-step and transition fixture through the actual reify plugin, then export the same complete declaration through the source path. Prove that compiler expansion preserves the original expectation body and contextual source symbols long enough for the shared interpreter to recover them. Explicit names `s`, `(b,a)`, ordinary `before`/`after`, compiler-generated placeholder names and named helper forwarding all need exact core parameter recovery. A runtime carrier is not itself `S` or `Step`; map its source identity to the emitted parameter without inventing uniform names or requiring new author vocabulary. If `.2` left insufficient metadata, repair that smallest shared seam before adding migration callers or comparison exemptions.

Extend the current block recognition deliberately. `BodyContext.blockCall/blockVal` and `Expressions.blockFunction` currently treat `is`/`effect` as section values and invent `s`; adding `expect` to that set alone is incorrect. Expectation arguments also occur inline, in named helpers, in generic capability constructors and under compositions. Add function-valued block recognition at the shared body/capture boundary without weakening the existing placement rules of unrelated forms.

### 2. One source-aware interpretation, ordinary Boolean core IR

Use the shared `reify/body` engine for compiler capture and source export. Lower the validated requirements to existing Boolean/conditional/Let/Call expressions, with no IR schema change and no runtime control-signal or exception node. Keep source-aware claim registration, declaration naming and restrictions on the current core path.

Preserve exact expression structure, not just truth values:

- Conjunctive requirement order and Boolean association remain exact. Do not insert identity literals (`true &&`, `false ||`), flatten association or remove wrappers in the comparator.
- Conditional requirements without an else retain the old implication core `!c || r`. Conditionals with an else retain the actual core conditional shape where the old twin has one. A small `check(if ... then ... else ...)` can preserve an existing core conditional; do not change the accepted vocabulary.
- `oneOf` lowers explicit grouped `branch` requirements to the corresponding OR tree, preserving conjunction grouping within each branch and written short-circuit order.
- Pure immutable Boolean helper lets remain eager, once-only and in their original lexical placement. Do not sink a preexisting `restarted` let into a selected branch, delay it past a failing requirement, duplicate it, or discard an unused-but-observable read.
- A pure single-check forwarding body retains the original referenced function and its binding identity instead of introducing a wrapper. Preserve named-def references, call grouping, generic type substitution, parameter order/names/types and the ordinary function metadata retained by capture. A comparator that dereferences only the body hides extra functions and is insufficient.
- Bare getters read `after.state` in both predicate forms. Only the transition form has before-state evidence. Preserve exact-class `after` and whole-action `afterAction` restriction metadata, including String and Composed action cases.

The full function/type/declaration inventory is part of the oracle. Preserve `Captured.root`, its reachable `model.functions`/`model.types`, ordinary callable identity, transitive callees, parameter identities, declaration/function references, source positions and any origin/call-group metadata retained by the existing capture seam. Do not add annotations whose only purpose is to exempt expectation functions from existing capture validation.

### 3. Establish the strict early paired fixtures

Create generic synthetic fixtures under `export/testdata/expect`, without reading the held system examples:

- Heartbeat-shaped same-step fact AND implication trees, including restricted and unrestricted heads.
- Reset-shaped transition predicates with the eager `restarted` helper let, six alternatives on one path and three on the other, preserving the old conditional/branch structure.
- Explicit `s`, `(b,a)` and compiler-generated source parameter names; inline and named predicate forwarding; nested conditionals and branch conjunctions.

Each expectation fixture has an independently authored ordinary-core twin. `.3` may consume existing core lambda finishers temporarily, but prefer direct core construction for the independent twin where practical so `.8` can remove the authoring methods without losing the oracle. Do not derive the core twin by calling the expectation implementation.

Compare complete native-captured and source-exported Models against the core twins: declaration/function/type inventories, names, references, parameters, expression trees, metadata and explicitly mapped positions. When paired fixture roots necessarily have different namespaces, use a finite, explicit bijective declaration-root map; do not normalize parameters/locals or helper identities. Check the positions against an authored source-span map first, then apply only that permitted mapping for structural comparison. Whole-tree position erasure without a source-position oracle, a structural fingerprint, an operator/fact substring check or a runtime truth table is not this proof. Add negative controls that change association, a parameter, a let position, an extra wrapper function, selected fact/member or a source span and prove the comparator rejects each change.

Stop at a failed native/capture/paired proof and repair the API or shared interpretation. Do not migrate callers or weaken equality to make the fixtures pass.

### 4. Ordinary-function bridges and level protection

Construct, capture and lift inline and named same-step expectations used by `never`, `stays(...).unless`, `sticky` and `Monitor.readAfter`, and transition expectations used by `stickyAcross`. Retain ordinary state predicates for `once`, `stays`, `from` and `leadsTo`; expectation support does not turn a step predicate into a state predicate. Include reusable helpers over generic `Declares[S]`, capability constructors and composed states, rather than requiring machine-only `Owner` evidence. Query verify/find and progress/monitor declarations remain outside the predicate; no new temporal operator is introduced.

Keep `.2`'s compile protection effective through lifting. Cover forbidden path/temporal declarations and step construction directly, locally bound, discarded, nested in guards/checks/helper RHSs and hidden through reached source helpers, including a machine with inherited givens. Include before-state access in a same-step block, assignment-shaped assertions, final/nonfinal discarded Booleans, unsupported statements, empty block/branch, helpers-only blocks, insufficient alternatives and misplaced branches. Compile refusals must remain compile refusals where required by the spec; an exporter-only failure is additional coverage, not a replacement. Assert physical authored positions and no written output on refusal. Preserve conditional-only nonempty blocks and the complete short-circuit/hole/exception behavior already proved natively in `.2`.

### 5. Exact composed facts and current witness preservation

Use `.2`'s exact authored field plus selected-machine evidence, with the selected machine fixing the fact type. Prove the native and lifted Boolean fact query and Unit requirement agree. Cover same-named facts, members with equal state types, a wrong field/machine pair, repeated instances of the same machine in distinct fields, foreign fact compile refusal and opaque selector refusal where the current capture contract requires a direct field. No suffix matching, all-member widening, `Any` fact erasure or unique-machine heuristic may establish identity.

An owner-free ordinary Step function cannot derive composition ownership from Step alone. Preserve its retained selector field/machine evidence and prove validation on attachment to the declaring composition, including monitor attachment where relevant. Do not credit source-only owner validation as native runtime evidence. If `.2` cannot prove this contract, resolve it before completing the bridges.

Consume the actual current witness receipts and generic witness fixtures. Preserve present/absent primary polarity, every ordered required/forbidden fact guard, selected member, condition helper identity and generated ordinary Property/Scenario/Query names and metadata. Do not recreate retired private triples. Compare complete witness declarations/functions and source mappings against their predecessor, not merely the operators or names they contain.

### 6. Generic expansion and isolated suite integration

Preserve current definition expansion for marked helper/pattern/monitor sugar. Expectation control execution does not justify an independent permanent exporter name-dispatch registry. Keep source/capture interpretation in the shared block seam, document any remaining construct-level handoff explicitly for fn-141, and record the accepted API and exact preservation fixtures that it must consume. Every added public sugar definition remains in the framework sugar file with `Core form:` documentation; core files must not import sugar.

Use the established `ModelBuild.frameworkLibrary`/`reifyPlugin`/`classpathEntries` and `InProcess.frameworkLift` seams for new generic fixtures. Source copies and directive rewrites must preserve authored offsets. Build only against the current isolated runner's fresh `framework.jar`, `reify.jar`, classpath and the existing generic IR jar. Do not fall back to stale `model-scala.jar`, `api-scalapb.jar`, system fixture roots, `ModelBuild.mini` or the held generic `Fixtures` suite that contains system-backed data.

Register the exact new suite names `framework.export.ExpectFixtures` and `framework.export.ExpectCaptureSuite` in `FrameworkSuites.genericExport` and add a selection regression. In the touched `FrameworkLoopSuite`, also assert the current repository's `Boundary.findings(root, Seq("temporal", "io.temporal.api"))` is empty, so the focused task command checks the real boundary as well as synthetic classifier/poisoning controls. The runner intentionally rejects unclassified exporter suites; an ignored or held new suite cannot earn acceptance credit. No broad exception or classifier default is allowed.

### Investigation targets

**Required:**

- `.2`'s committed `dsl/Syntax.scala`, expectation validator/metadata and native `Expect` suite; `dsl/{Claims,Machine,Monitor,Assume}.scala` for the ordinary core signatures and generic declarer types.
- `reify/body/BodyContext.scala` (`BlockCall`, `blockVal`, parameter/function identities), `BodySyntax.scala` (existing statement interpretation), `Expressions.scala` (`blockFunction`, `functionOf`, `giving`/`wrongLevel`) and `BodyEngine.scala` (`stepFunctionOrigin`).
- `export/Claims.scala` (core registration and generic callback binding), `DefinitionExpansion.test.scala`, `ReifyBoundary.test.scala`, `ConstructedRules.test.scala`, `SourcePositions.test.scala`, and the current generic witness cases in `Fixtures.test.scala`.
- `check/FrameworkLoop.scala`/`.test.scala`, `export/ModelBuild.test.scala`, `export/InProcess.test.scala` and `export/FrameworkFixtureSettings.test.scala` for fresh generic artifacts, classification and positioned fixtures.

**Conditional:** `reify/plugin` and the generic `export` context/composition/capability seams reached by the proof. No system-tree investigation targets or legacy `model/irgen`/`model/check` paths apply.

### Verification commands

Follow `MILESTONES.md`'s owner cadence: focused touched suites during iteration and at this task's closure; the full framework gate and independent review run once at source-batch end. The current runner has no focused-suite flag. Do not run its complete suite set merely to refresh compiler artifacts.

For each changed source checkpoint needing capture/export verification, build fresh core/plugin artifacts into an explicit new task-local directory using the same inputs and options as `FrameworkLoop.check`. The actual `ModelSources.frameworkCore` is exactly `model/framework/dsl model/framework/realize`. This command block is a documented direct route, not a new permanent runner. It serializes the two builds, retains their logs and validates the returned classpath before any exporter fixture can run. Scala CLI's printed compiler errors fail even if its process exits zero, matching `Tools.Ran.orFail`.

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
set -e
unset FEATURES BASE INTERPRETER_PARITY SAMPLE_REPORT_INPUT UMPIRE_LIFTER_UPDATE UMPIRE_FRAMEWORK_BUILD
FN164_FRAMEWORK_BUILD="$(mktemp -d "$PWD/.flow/tmp/checks/fn164.3-focused.XXXXXX")"
test -s model/build/ir-scalapb.jar
mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning --print-class-path --compilation-output "$FN164_FRAMEWORK_BUILD/framework.jar" --scalac-option -sourceroot --scalac-option "$PWD/model" model/framework/dsl model/framework/realize > "$FN164_FRAMEWORK_BUILD/core.raw-classpath" 2> "$FN164_FRAMEWORK_BUILD/core.stderr"
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/core.raw-classpath" "$FN164_FRAMEWORK_BUILD/core.stderr"; then exit 1; fi
test -f "$FN164_FRAMEWORK_BUILD/framework.jar"
test -s "$FN164_FRAMEWORK_BUILD/framework.jar"
IFS=: read -r -a FN164_FRAMEWORK_CP_ENTRIES < "$FN164_FRAMEWORK_BUILD/core.raw-classpath"
test "${#FN164_FRAMEWORK_CP_ENTRIES[@]}" -gt 0
for FN164_FRAMEWORK_ENTRY in "${FN164_FRAMEWORK_CP_ENTRIES[@]}"; do
  test -e "$FN164_FRAMEWORK_ENTRY"
  case "$FN164_FRAMEWORK_ENTRY" in
    */api-scalapb.jar|*/model-scala.jar|*/model/temporal/*) exit 1 ;;
  esac
done
FN164_FRAMEWORK_CP="$FN164_FRAMEWORK_BUILD/framework.jar"
for ((FN164_CP_INDEX=1; FN164_CP_INDEX<${#FN164_FRAMEWORK_CP_ENTRIES[@]}; FN164_CP_INDEX++)); do
  FN164_FRAMEWORK_CP+=":${FN164_FRAMEWORK_CP_ENTRIES[FN164_CP_INDEX]}"
done
printf '%s\n' "$FN164_FRAMEWORK_CP" > "$FN164_FRAMEWORK_BUILD/framework.classpath"
mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --assembly --preamble=false --main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 model/framework/reify/plugin -f -o "$FN164_FRAMEWORK_BUILD/reify.jar" > "$FN164_FRAMEWORK_BUILD/plugin.log" 2>&1
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/plugin.log"; then exit 1; fi
test -s "$FN164_FRAMEWORK_BUILD/reify.jar"
test -s "$FN164_FRAMEWORK_BUILD/framework.classpath"
```

Preserve the directory path in the task's evidence. It is the fresh core/plugin build for the following focused checks. After changing `dsl`, `realize`, shared body or plugin inputs, rerun that narrow build block into another new directory; a full gate is not a prerequisite. If only test/exporter code changes and the frozen core/plugin inputs are checksum-identical, this exact directory may be reused. Never select an old directory by a glob or omit `UMPIRE_FRAMEWORK_BUILD`: the exporter harness otherwise defaults to system artifacts. This matches `FrameworkLoop`: validate the existing raw entries and regular fresh jar, then write `(freshCoreJar +: rawEntries.drop(1))`. The pinned compiler's first raw entry is a classes directory, so retain it only in the raw receipt and replace it with the fresh jar in the fixture classpath.

Run the touched suites with that explicit fresh environment. Redirect each JUnit command to its task-local log so its compiler diagnostics and positive executed counts can be checked, even on a zero process exit:

```sh
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli run --test --server=false model/framework/export --main-class org.junit.runner.JUnitCore -- framework.export.ExpectFixtures framework.export.ExpectCaptureSuite > "$FN164_FRAMEWORK_BUILD/expect-tests.log" 2>&1
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli run --test --server=false model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.lint.SyntaxRuleSuite framework.check.FrameworkLoopSuite > "$FN164_FRAMEWORK_BUILD/check-tests.log" 2>&1
for FN164_TEST_LOG in "$FN164_FRAMEWORK_BUILD/expect-tests.log" "$FN164_FRAMEWORK_BUILD/check-tests.log"; do
  if rg -q '\[error\]' "$FN164_TEST_LOG"; then exit 1; fi
  rg '^OK \(([1-9][0-9]*) tests?\)' "$FN164_TEST_LOG"
done
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only '*Expect*' > "$FN164_FRAMEWORK_BUILD/native-expect.log" 2>&1
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/native-expect.log"; then exit 1; fi
```

For each JUnit command, reject any `[error]` diagnostics and require `^OK \(([1-9][0-9]*) tests?\)` in its retained output, matching the framework runner's nonempty-test contract. The native `test --require-tests` command must report a positive executed count for its expectation suite too; a selected-but-ignored suite earns no credit. Add `framework.export.DefinitionExpansionTest`, `ReifyBoundarySuite`, `WitnessFixtures` and/or `SourcePositionsFixtures` to a focused exporter command when their shared implementation surfaces were touched, and run that affected set at task closure. Do not eagerly run unrelated suites after an unchanged focused pass.

The task receipt records the fresh artifact directory, input revision/diff or checksums, exact commands, exits, positive counts, strict paired-comparison evidence and deferred batch gate. `FrameworkLoopSuite` supplies real boundary and classification/poisoning checks; `ExpectFixtures`/`ExpectCaptureSuite` prove their fixture classpaths contain only current generic artifacts. When fixtures run their own compile/JVM commands, inspect their recorded arguments for the same isolation. Scratch comparison output is permitted; managed Model IR/Cases remain unchanged. No root equivalence/step-table target or held system-backed `Fixtures` suite is required.

At source-batch completion, run the existing canonical gate once, serially:

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
make check-framework
```

That final gate rebuilds artifacts, checks the boundary and all discovered suite classifications, and runs nonempty generic core/check/export suites. Its printed record directory, counts and explicit held integration counts belong to the batch receipt. Until then, this task's receipt explicitly says the full gate and independent review are deferred; it claims only the focused checks actually run.
## Approach

### 1. Consume and prove the native/capture contract first

Read `.2`'s committed implementation, tests and receipts. Pin the accepted builder heads (`after`, distinct `afterAction`, `.across expect`, unrestricted `property expect`), standalone same-step/transition constructor spellings and exact contextual binding forms. The reusable results must have the ordinary function types `Step[S,O,F] => Boolean` and `(S,Step[S,O,F]) => Boolean`, not a second predicate AST or a second handwritten Boolean implementation.

Compile and execute a minimal same-step and transition fixture through the actual reify plugin, then export the same complete declaration through the source path. Prove that compiler expansion preserves the original expectation body and contextual source symbols long enough for the shared interpreter to recover them. Explicit names `s`, `(b,a)`, ordinary `before`/`after`, compiler-generated placeholder names and named helper forwarding all need exact core parameter recovery. A runtime carrier is not itself `S` or `Step`; map its source identity to the emitted parameter without inventing uniform names or requiring new author vocabulary. If `.2` left insufficient metadata, repair that smallest shared seam before adding migration callers or comparison exemptions.

Extend the current block recognition deliberately. `BodyContext.blockCall/blockVal` and `Expressions.blockFunction` currently treat `is`/`effect` as section values and invent `s`; adding `expect` to that set alone is incorrect. Expectation arguments also occur inline, in named helpers, in generic capability constructors and under compositions. Add function-valued block recognition at the shared body/capture boundary without weakening the existing placement rules of unrelated forms.

### 2. One source-aware interpretation, ordinary Boolean core IR

Use the shared `reify/body` engine for compiler capture and source export. Lower the validated requirements to existing Boolean/conditional/Let/Call expressions, with no IR schema change and no runtime control-signal or exception node. Keep source-aware claim registration, declaration naming and restrictions on the current core path.

Preserve exact expression structure, not just truth values:

- Conjunctive requirement order and Boolean association remain exact. Do not insert identity literals (`true &&`, `false ||`), flatten association or remove wrappers in the comparator.
- Conditional requirements without an else retain the old implication core `!c || r`. Conditionals with an else retain the actual core conditional shape where the old twin has one. A small `check(if ... then ... else ...)` can preserve an existing core conditional; do not change the accepted vocabulary.
- `oneOf` lowers explicit grouped `branch` requirements to the corresponding OR tree, preserving conjunction grouping within each branch and written short-circuit order.
- Pure immutable Boolean helper lets remain eager, once-only and in their original lexical placement. Do not sink a preexisting `restarted` let into a selected branch, delay it past a failing requirement, duplicate it, or discard an unused-but-observable read.
- A pure single-check forwarding body retains the original referenced function and its binding identity instead of introducing a wrapper. Preserve named-def references, call grouping, generic type substitution, parameter order/names/types and the ordinary function metadata retained by capture. A comparator that dereferences only the body hides extra functions and is insufficient.
- Bare getters read `after.state` in both predicate forms. Only the transition form has before-state evidence. Preserve exact-class `after` and whole-action `afterAction` restriction metadata, including String and Composed action cases.

The full function/type/declaration inventory is part of the oracle. Preserve `Captured.root`, its reachable `model.functions`/`model.types`, ordinary callable identity, transitive callees, parameter identities, declaration/function references, source positions and any origin/call-group metadata retained by the existing capture seam. Do not add annotations whose only purpose is to exempt expectation functions from existing capture validation.

### 3. Establish the strict early paired fixtures

Create generic synthetic fixtures under `export/testdata/expect`, without reading the held system examples:

- Heartbeat-shaped same-step fact AND implication trees, including restricted and unrestricted heads.
- Reset-shaped transition predicates with the eager `restarted` helper let, six alternatives on one path and three on the other, preserving the old conditional/branch structure.
- Explicit `s`, `(b,a)` and compiler-generated source parameter names; inline and named predicate forwarding; nested conditionals and branch conjunctions.

Each expectation fixture has an independently authored ordinary-core twin. `.3` may consume existing core lambda finishers temporarily, but prefer direct core construction for the independent twin where practical so `.8` can remove the authoring methods without losing the oracle. Do not derive the core twin by calling the expectation implementation.

Compare complete native-captured and source-exported Models against the core twins: declaration/function/type inventories, names, references, parameters, expression trees, metadata and explicitly mapped positions. When paired fixture roots necessarily have different namespaces, use a finite, explicit bijective declaration-root map; do not normalize parameters/locals or helper identities. Check the positions against an authored source-span map first, then apply only that permitted mapping for structural comparison. Whole-tree position erasure without a source-position oracle, a structural fingerprint, an operator/fact substring check or a runtime truth table is not this proof. Add negative controls that change association, a parameter, a let position, an extra wrapper function, selected fact/member or a source span and prove the comparator rejects each change.

Stop at a failed native/capture/paired proof and repair the API or shared interpretation. Do not migrate callers or weaken equality to make the fixtures pass.

### 4. Ordinary-function bridges and level protection

Construct, capture and lift inline and named same-step expectations used by `never`, `stays(...).unless`, `sticky` and `Monitor.readAfter`, and transition expectations used by `stickyAcross`. Retain ordinary state predicates for `once`, `stays`, `from` and `leadsTo`; expectation support does not turn a step predicate into a state predicate. Include reusable helpers over generic `Declares[S]`, capability constructors and composed states, rather than requiring machine-only `Owner` evidence. Query verify/find and progress/monitor declarations remain outside the predicate; no new temporal operator is introduced.

Keep `.2`'s compile protection effective through lifting. Cover forbidden path/temporal declarations and step construction directly, locally bound, discarded, nested in guards/checks/helper RHSs and hidden through reached source helpers, including a machine with inherited givens. Include before-state access in a same-step block, assignment-shaped assertions, final/nonfinal discarded Booleans, unsupported statements, empty block/branch, helpers-only blocks, insufficient alternatives and misplaced branches. Compile refusals must remain compile refusals where required by the spec; an exporter-only failure is additional coverage, not a replacement. Assert physical authored positions and no written output on refusal. Preserve conditional-only nonempty blocks and the complete short-circuit/hole/exception behavior already proved natively in `.2`.

### 5. Exact composed facts and current witness preservation

Use `.2`'s exact authored field plus selected-machine evidence, with the selected machine fixing the fact type. Prove the native and lifted Boolean fact query and Unit requirement agree. Cover same-named facts, members with equal state types, a wrong field/machine pair, repeated instances of the same machine in distinct fields, foreign fact compile refusal and opaque selector refusal where the current capture contract requires a direct field. No suffix matching, all-member widening, `Any` fact erasure or unique-machine heuristic may establish identity.

An owner-free ordinary Step function cannot derive composition ownership from Step alone. Preserve its retained selector field/machine evidence and prove validation on attachment to the declaring composition, including monitor attachment where relevant. Do not credit source-only owner validation as native runtime evidence. If `.2` cannot prove this contract, resolve it before completing the bridges.

Consume the actual current witness receipts and generic witness fixtures. Preserve present/absent primary polarity, every ordered required/forbidden fact guard, selected member, condition helper identity and generated ordinary Property/Scenario/Query names and metadata. Do not recreate retired private triples. Compare complete witness declarations/functions and source mappings against their predecessor, not merely the operators or names they contain.

### 6. Generic expansion and isolated suite integration

Preserve current definition expansion for marked helper/pattern/monitor sugar. Expectation control execution does not justify an independent permanent exporter name-dispatch registry. Keep source/capture interpretation in the shared block seam, document any remaining construct-level handoff explicitly for fn-141, and record the accepted API and exact preservation fixtures that it must consume. Every added public sugar definition remains in the framework sugar file with `Core form:` documentation; core files must not import sugar.

Use the established `ModelBuild.frameworkLibrary`/`reifyPlugin`/`classpathEntries` and `InProcess.frameworkLift` seams for new generic fixtures. Source copies and directive rewrites must preserve authored offsets. Build only against the current isolated runner's fresh `framework.jar`, `reify.jar`, classpath and the existing generic IR jar. Do not fall back to stale `model-scala.jar`, `api-scalapb.jar`, system fixture roots, `ModelBuild.mini` or the held generic `Fixtures` suite that contains system-backed data.

Register the exact new suite names `framework.export.ExpectFixtures` and `framework.export.ExpectCaptureSuite` in `FrameworkSuites.genericExport` and add a selection regression. In the touched `FrameworkLoopSuite`, also assert the current repository's `Boundary.findings(root, Seq("temporal", "io.temporal.api"))` is empty, so the focused task command checks the real boundary as well as synthetic classifier/poisoning controls. The runner intentionally rejects unclassified exporter suites; an ignored or held new suite cannot earn acceptance credit. No broad exception or classifier default is allowed.

## Investigation targets

**Required:**

- `.2`'s committed `dsl/Syntax.scala`, expectation validator/metadata and native `Expect` suite; `dsl/{Claims,Machine,Monitor,Assume}.scala` for the ordinary core signatures and generic declarer types.
- `reify/body/BodyContext.scala` (`BlockCall`, `blockVal`, parameter/function identities), `BodySyntax.scala` (existing statement interpretation), `Expressions.scala` (`blockFunction`, `functionOf`, `giving`/`wrongLevel`) and `BodyEngine.scala` (`stepFunctionOrigin`).
- `export/Claims.scala` (core registration and generic callback binding), `DefinitionExpansion.test.scala`, `ReifyBoundary.test.scala`, `ConstructedRules.test.scala`, `SourcePositions.test.scala`, and the current generic witness cases in `Fixtures.test.scala`.
- `check/FrameworkLoop.scala`/`.test.scala`, `export/ModelBuild.test.scala`, `export/InProcess.test.scala` and `export/FrameworkFixtureSettings.test.scala` for fresh generic artifacts, classification and positioned fixtures.

**Conditional:** `reify/plugin` and the generic `export` context/composition/capability seams reached by the proof. No system-tree investigation targets or legacy `model/irgen`/`model/check` paths apply.

## Verification commands

Follow `MILESTONES.md`'s owner cadence: focused touched suites during iteration and at this task's closure; the full framework gate and independent review run once at source-batch end. The current runner has no focused-suite flag. Do not run its complete suite set merely to refresh compiler artifacts.

For each changed source checkpoint needing capture/export verification, build fresh core/plugin artifacts into an explicit new task-local directory using the same inputs and options as `FrameworkLoop.check`. The actual `ModelSources.frameworkCore` is exactly `model/framework/dsl model/framework/realize`. This command block is a documented direct route, not a new permanent runner. It serializes the two builds, retains their logs and validates the returned classpath before any exporter fixture can run. Scala CLI's printed compiler errors fail even if its process exits zero, matching `Tools.Ran.orFail`.

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
set -e
unset FEATURES BASE INTERPRETER_PARITY SAMPLE_REPORT_INPUT UMPIRE_LIFTER_UPDATE UMPIRE_FRAMEWORK_BUILD
FN164_FRAMEWORK_BUILD="$(mktemp -d "$PWD/.flow/tmp/checks/fn164.3-focused.XXXXXX")"
test -s model/build/ir-scalapb.jar
mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning --print-class-path --compilation-output "$FN164_FRAMEWORK_BUILD/framework.jar" --scalac-option -sourceroot --scalac-option "$PWD/model" model/framework/dsl model/framework/realize > "$FN164_FRAMEWORK_BUILD/core.raw-classpath" 2> "$FN164_FRAMEWORK_BUILD/core.stderr"
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/core.raw-classpath" "$FN164_FRAMEWORK_BUILD/core.stderr"; then exit 1; fi
test -f "$FN164_FRAMEWORK_BUILD/framework.jar"
test -s "$FN164_FRAMEWORK_BUILD/framework.jar"
IFS=: read -r -a FN164_FRAMEWORK_CP_ENTRIES < "$FN164_FRAMEWORK_BUILD/core.raw-classpath"
test "${#FN164_FRAMEWORK_CP_ENTRIES[@]}" -gt 0
for FN164_FRAMEWORK_ENTRY in "${FN164_FRAMEWORK_CP_ENTRIES[@]}"; do
  test -e "$FN164_FRAMEWORK_ENTRY"
  case "$FN164_FRAMEWORK_ENTRY" in
    */api-scalapb.jar|*/model-scala.jar|*/model/temporal/*) exit 1 ;;
  esac
done
FN164_FRAMEWORK_CP="$FN164_FRAMEWORK_BUILD/framework.jar"
for ((FN164_CP_INDEX=1; FN164_CP_INDEX<${#FN164_FRAMEWORK_CP_ENTRIES[@]}; FN164_CP_INDEX++)); do
  FN164_FRAMEWORK_CP+=":${FN164_FRAMEWORK_CP_ENTRIES[FN164_CP_INDEX]}"
done
printf '%s\n' "$FN164_FRAMEWORK_CP" > "$FN164_FRAMEWORK_BUILD/framework.classpath"
mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --assembly --preamble=false --main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 model/framework/reify/plugin -f -o "$FN164_FRAMEWORK_BUILD/reify.jar" > "$FN164_FRAMEWORK_BUILD/plugin.log" 2>&1
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/plugin.log"; then exit 1; fi
test -s "$FN164_FRAMEWORK_BUILD/reify.jar"
test -s "$FN164_FRAMEWORK_BUILD/framework.classpath"
```

Preserve the directory path in the task's evidence. It is the fresh core/plugin build for the following focused checks. After changing `dsl`, `realize`, shared body or plugin inputs, rerun that narrow build block into another new directory; a full gate is not a prerequisite. If only test/exporter code changes and the frozen core/plugin inputs are checksum-identical, this exact directory may be reused. Never select an old directory by a glob or omit `UMPIRE_FRAMEWORK_BUILD`: the exporter harness otherwise defaults to system artifacts. This matches `FrameworkLoop`: validate the existing raw entries and regular fresh jar, then write `(freshCoreJar +: rawEntries.drop(1))`. The pinned compiler's first raw entry is a classes directory, so retain it only in the raw receipt and replace it with the fresh jar in the fixture classpath.

Run the touched suites with that explicit fresh environment. Redirect each JUnit command to its task-local log so its compiler diagnostics and positive executed counts can be checked, even on a zero process exit:

```sh
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli run --test --server=false model/framework/export --main-class org.junit.runner.JUnitCore -- framework.export.ExpectFixtures framework.export.ExpectCaptureSuite > "$FN164_FRAMEWORK_BUILD/expect-tests.log" 2>&1
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli run --test --server=false model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.lint.SyntaxRuleSuite framework.check.FrameworkLoopSuite > "$FN164_FRAMEWORK_BUILD/check-tests.log" 2>&1
for FN164_TEST_LOG in "$FN164_FRAMEWORK_BUILD/expect-tests.log" "$FN164_FRAMEWORK_BUILD/check-tests.log"; do
  if rg -q '\[error\]' "$FN164_TEST_LOG"; then exit 1; fi
  rg '^OK \(([1-9][0-9]*) tests?\)' "$FN164_TEST_LOG"
done
UMPIRE_FRAMEWORK_BUILD="$FN164_FRAMEWORK_BUILD" mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only '*Expect*' > "$FN164_FRAMEWORK_BUILD/native-expect.log" 2>&1
if rg -q '\[error\]' "$FN164_FRAMEWORK_BUILD/native-expect.log"; then exit 1; fi
```

For each JUnit command, reject any `[error]` diagnostics and require `^OK \(([1-9][0-9]*) tests?\)` in its retained output, matching the framework runner's nonempty-test contract. The native `test --require-tests` command must report a positive executed count for its expectation suite too; a selected-but-ignored suite earns no credit. Add `framework.export.DefinitionExpansionTest`, `ReifyBoundarySuite`, `WitnessFixtures` and/or `SourcePositionsFixtures` to a focused exporter command when their shared implementation surfaces were touched, and run that affected set at task closure. Do not eagerly run unrelated suites after an unchanged focused pass.

The task receipt records the fresh artifact directory, input revision/diff or checksums, exact commands, exits, positive counts, strict paired-comparison evidence and deferred batch gate. `FrameworkLoopSuite` supplies real boundary and classification/poisoning checks; `ExpectFixtures`/`ExpectCaptureSuite` prove their fixture classpaths contain only current generic artifacts. When fixtures run their own compile/JVM commands, inspect their recorded arguments for the same isolation. Scratch comparison output is permitted; managed Model IR/Cases remain unchanged. No root equivalence/step-table target or held system-backed `Fixtures` suite is required.

At source-batch completion, run the existing canonical gate once, serially:

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
make check-framework
```

That final gate rebuilds artifacts, checks the boundary and all discovered suite classifications, and runs nonempty generic core/check/export suites. Its printed record directory, counts and explicit held integration counts belong to the batch receipt. Until then, this task's receipt explicitly says the full gate and independent review are deferred; it claims only the focused checks actually run.

## Acceptance

- [ ] `.2`'s committed native spellings/context metadata are consumed; inline, named, generic Declares/capability and composed expectation functions retain ordinary core function types, native callable behavior and captured root/transitive metadata. Explicit and compiler-generated parameters keep their exact identities (R1-R5).
- [ ] Complete heartbeat/reset-shaped paired Models match independent ordinary-core twins except finite declared root/source mappings, including exact association, conditional shapes, eager let order/placement, function inventories, named forwarding, parameter/types and all retained metadata. Comparator negative controls reject substantive and source-span corruption (R5).
- [ ] All listed generic temporal-logic/pattern/monitor bridges construct, capture and lift without changing their outer semantics or accepting path/step creation inside predicates. Source-located compile/lift negatives and no-output refusals remain covered; no new author vocabulary is introduced (R1-R4).
- [ ] Exact composed field/fact/machine binding is independently proved natively and through capture/export, including attachment of owner-free predicates and repeated same-machine members. Current positive/absent/conditional witnesses retain full ordinary declarations/functions/metadata and guard order (R1,R2,R4,R5).
- [ ] Source and capture use the shared interpretation; no permanent parallel lowering registry, alpha-normalization, Boolean reassociation, wrapper exemption, source-span erasure or structural-hash substitution replaces exact proof. The accepted generic-expansion contract and fn-141 handoff are recorded (R5).
- [ ] Both new exporter suites are explicitly classified generic and execute nonzero focused tests against fresh isolated core/plugin artifacts; touched native, syntax, capture, definition-expansion, witness and position regressions pass with no compiler-error diagnostics. The committed task receipt cites actual commands/counts/artifact provenance and strict comparison evidence; full `make check-framework` and independent review remain recorded source-batch-end obligations.
- [ ] Framework `.7` may begin from this committed closure. `.8` finisher removal and `.9` colon cleanup remain required after `.7`; held system migration/seals, full-spec R6/R8, regeneration and other excluded work are not claimed complete.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: