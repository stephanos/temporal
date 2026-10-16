---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-164-expect-blocks-for-properties.2 Implement typed expectation blocks and short-circuit runtime semantics

## Description
**DSL-first framework execution amendment (owner override).** Work only in `model/framework`; do not read, compile, migrate or restore `model/temporal` or use `model/project.scala`, Temporal API/Model jars, Go, schemas or regeneration. Start from source checkpoint `437016b7d1` and its passing native core/check suites. The owner explicitly prioritizes native DSL work before IR/exporter work; fn-188.2's incomplete broad export gate is held and is not this task's prerequisite. Prove the native framework API/runtime and compile refusals now. Do not implement or test new capture/exporter lowering, IR equivalence or IR driver repairs here. Preserve metadata needed by the deferred .3 proof without claiming that proof passed. This task does not complete Temporal migration or whole-spec requirements. Keep `.4`/`.5` and R6 open; native public finisher retirement follows a separately scoped native caller-migration checkpoint.

**First proofs.** Prove the accepted native builder and standalone ordinary-function spellings before extending callers. Follow the existing Effects-style typed contexts, with bare fields reading the resulting state and before-state evidence only in `.across`. Preserve explicit contextual source symbols for exact parameter recovery by `.3`; do not erase names into an opaque runtime wrapper or claim unrun IR parity. Use per-evaluation private control signals for conjunctive and alternative short-circuiting, catching only owned signals so holes and unrelated exceptions propagate. Add source-aware inline validation for final/nonfinal discarded Boolean expressions and locally bound/discarded path or step construction; Unit adaptation and ambient Owner absence alone are insufficient. Exact composed selectors must validate both the authored field and selected machine, including repeated same-machine members. Owner-free ordinary predicates must retain evidence validated at attachment to their declaring composition, or prove an equally typed solution; Step alone does not establish that relationship.

**Size:** M
**Files:** expectation sugar/contexts and focused framework compile/runtime tests.
**Touches:** [model/framework/dsl/**, model/framework/check/lint/SyntaxRule*.scala]

Use one shared selector implementation if witnesses need the same exact-member correction; preserve existing witness regressions. Core storage redesign and generic lifting belong outside this native task unless an observed proof requires a minimal typed metadata seam.

Implement R1-R3's API and the runtime foundation of R4, keeping core Property declaration storage unchanged. Existing public lambda finishers remain temporarily for migration and are removed together in the final surface task.

### Approach
- Add typed expectation contexts and builder extensions in the sugar layer, not core Claims. Use the spec's exact-class `after`, distinct whole-action `afterAction`, and transition `.across` head; unqualified property and generic `Declares` capability/composition contexts work without machine-only Owner.
- Prove the selected Scala heads with a small compile fixture before implementing callers. Offer explicit contextual binding names that can later reify original core before/Step parameter names; include `s`, `(b,a)` and compiler-generated source parameters. Ordinary blocks retain implicit `before`/typed `after` and bare resulting-state fields. Keep reusable predicate bridges possible without a second Boolean implementation.
- Implement Unit requirements, conditional blocks and oneOf's explicit branch grouping with immediate failure and isolated alternatives. Test failed check before a hole-bearing later guard, unsuccessful branch followed by success, success skipping a later hole, all alternatives false and an actual hole propagating. Do not catch unrelated exceptions as assertion failure; conditional-only blocks are not runtime-empty declarations.
- Retain typed fact identity: machine foreign fact compilation refusal and exact field-to-machine composed selectors. Assert two members with the same fact name cannot satisfy each other. Reuse witness selection without suffix or Any erasure.
- Add compiler-enforced predicate-scope protection for forbidden temporal/path declarations and step-making expressions, even if bound to local vals or discarded; inherited Owner does not authorize effects here. Before in same-step blocks and assignment assertions fail compilation. Unit return typing alone is insufficient for discarded final Boolean expressions; validate them explicitly.
- Every added sugar entry has its Core form, and the syntax gate inventory recognizes it without core exemptions. Preserve immutable helper lets needed by existing predicates.

### Investigation targets
**Required:** `model/framework/dsl/Claims.scala:19-48` (core predicates); `model/framework/dsl/Syntax.scala:106-171` (View/context convention); `model/framework/dsl/Machine.scala:75` (Declares); `model/framework/dsl/Witness.test.scala` (typed composed facts); `model/framework/check/lint/SyntaxRule.scala` (sugar/core boundary); `model/framework/dsl/Effects.test.scala` (compile/runtime harness).
**Optional:** `model/framework/dsl/Monitor.scala`, `model/framework/dsl/Assume.scala` (outer constructor scope).

### Quick commands
```sh
mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only '*Expect*'
mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.SyntaxRuleSuite'
```

Run the existing `framework.dsl.WitnessTest` and `framework.dsl.EffectsTest` suites when changing their shared selector/context seams, with exact positive executed counts. Keep Scala builds serialized. Native core/check results at `437016b7d1` are the prior source baseline, not evidence that this task's new code passed. Focused relevant compile/runtime and syntax checks are this task's gate. Broad exporter/capture/IR verification, `make check-framework`, Go, regeneration and live/system checks remain deferred until the later IR phase and earn no pass credit here. Do not spend this task repairing that held gate.

## Acceptance
- [ ] Same-step/transition heads compile with typed before/after and bare field reads under machine, composition and generic Declares contexts; explicit contextual names support the preservation proof (R1/R2).
- [ ] Focused runtime tests prove conjunctive/alternative/conditional truth values and hole-aware short-circuit behavior, including conditional-only blocks (R1).
- [ ] Compile negatives cover before in same-step blocks, assignments, foreign facts, final/nonfinal discarded Boolean expressions, wrong-level temporal/path declarations and locally bound/discarded step-making expressions (R1-R4).
- [ ] Two-member composed fact tests check exact declared field/machine identity, and new sugar Core forms/syntax inventory preserve core-layer separation (R1/R7).
- [ ] Quick commands pass; no managed artifacts, Go or final public-finisher removal is claimed.


## Done summary
Implemented the native typed `expect` DSL with ordinary same-step/transition predicate functions, requirement-only compile validation, and per-evaluation owned short-circuit signals. Exact composed field/machine evidence is retained through native constructor metadata and validated at property attachment, including reordered and repeated-machine members.

Tier: session (jev-unavailable(no_key)); AGENTS explicit implementer gpt-6.1-sol medium used.
stage: impl-review - skipped(config: REVIEW_MODE=none)

### Verified native acceptance

- R1/R2: builder, standalone, exact-class/whole-action and across heads infer typed contexts; generic Declares/composition builders read resulting-state bare fields. Explicit contextual names, including an unused before binding, survive in predicate metadata. Quoted source recovery follows the compiler forwarding lambda by exact argument/parameter symbol identity to its original ValDef RHS; no source parsing or generated-name guessing.
- R1/R4: conjunction, alternatives, conditional-only and if/else truth matrices preserve evaluation order. Reached holes, unrelated errors and unrelated control signals propagate; concurrent and nested predicate evaluations remain isolated.
- R1-R4: compile refusals cover same-step before evidence even when supplied ambiently, mutation, foreign facts (including Boolean guards), discarded Boolean expressions, empty/invalid structural blocks, and bound/discarded path, temporal and step construction under ambient Owner. Nested expressions and source helpers are inspected.
- R1/R7: native composed predicates and witnesses distinguish exact field plus machine, reject computed/foreign-root selectors and wrong attachment even in unreachable branches, and support repeated machines/reordered constructor declarations. Existing never/unless wrappers validate original predicate evidence before wrapping it. Typed macro bridges stay inside Syntax.scala; the generic validator and private metadata seam do not import sugar. Core Property declaration storage remains unchanged.

### Gate evidence

Baseline: green, Expect selection 3 tests and SyntaxRuleSuite 16 tests. The first baseline command lost its output handle and was treated as inconclusive; its subsequent retained-handle rerun supplied the baseline, not an assumed pass.

Final commands exited 0 with positive executed counts:

- Task native Expect selection: 19 tests; `.flow/tmp/fn1642-final-expect-root.log`.
- SyntaxRuleSuite: 19 tests; `.flow/tmp/fn1642-final-syntax.log`.
- Existing WitnessTest: 6 tests; `.flow/tmp/fn1642-final-witness-root.log`.
- Existing EffectsTest: 8 tests; `.flow/tmp/fn1642-final-effects.log`.
- Scoped scalafmt check: 11 implementation/new-test files; `.flow/tmp/fn1642-final-format.log`. Witness.test.scala only changes one diagnostic literal and preserves its existing layout; no whole-corpus formatting claim.
- `git diff --check` passed. Gate classification returned `FULL: unmatched: MILESTONES.md` (exit 1; the range contains conductor planning commits). The task's focused native/syntax gates above are authoritative under its owner amendment; no broad gate receipt was fabricated.

Red-to-green observations include source-name loss, exact selector/foreign-root construction, outer-wrapper attachment loss, and syntax inventory/core-boundary refusals. Relevant retained reds: `fn1642-native-source-tree.log`, `fn1642-native-constructor-root-red.log`, `fn1642-native-outer-red-member-access.log`, and `fn1642-syntax-red.log`. A duplicate --test-only invocation was rejected before collecting tests and receives no pass credit.

### Deferred boundaries and next native migration

This closes native task .2 only. IR/capture/exporter equality and exact reification remain held for .3; no broad framework/exporter, Temporal, Go, schema, regeneration or whole-spec gate was run or claimed. The prior broad exporter gate remains incomplete. Restriction tests prove exact-class versus whole-action stored metadata, not action filtering by a bare Step, which carries no action identity. Public lambda finishers remain until the separately scoped native migration; no SHIP verdict is claimed.

For .10, removing holds/holdsAcross must not introduce a renamed compatibility finisher. Final author forwarding (`property expect: check(named(after))` and its across form) must retain the named predicate's exact composed attachment evidence, including hidden/unreachable wrong-owner references. Preserve the native direct-attachment/never/unless outcomes through those final public forms. That forwarding migration is not implemented by .2. BlockFormRule task .11 is separate and not included in this source commit.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8d45f9557352a06f2edc4826f646587a1e2018a8, 35a3a1e6278cb9d3e6e223e29d8dfe18d0fde717, b34918c2cf7b5df3e54b40769bd26135e2786f44, 34f0002db2d5f65a953e9110c59d80249f02a402
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only '*Expect*', mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.SyntaxRuleSuite', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only 'framework.dsl.WitnessTest', mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests --test-only 'framework.dsl.EffectsTest', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/check/lint/SyntaxRule.scala model/framework/check/lint/SyntaxRule.test.scala model/framework/dsl/Claims.scala model/framework/dsl/Compose.scala model/framework/dsl/Expect.test.scala model/framework/dsl/ExpectExternal.test.scala model/framework/dsl/ExpectationBlock.scala model/framework/dsl/MemberPath.scala model/framework/dsl/MemberSelection.scala model/framework/dsl/PredicateBindings.scala model/framework/dsl/Syntax.scala, git diff --check
- PRs: