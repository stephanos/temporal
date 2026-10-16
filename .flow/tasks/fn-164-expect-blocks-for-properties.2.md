---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-164-expect-blocks-for-properties.2 Implement typed expectation blocks and short-circuit runtime semantics

## Description
**Framework-only execution amendment (owner override).** Work only in `model/framework`; do not read, compile, migrate or restore `model/temporal` or use `model/project.scala`, Temporal API/Model jars, Go, schemas or regeneration. fn-188.2's committed framework-only loop closure is the admission prerequisite. This task proves the native framework API/runtime and compile refusals; it does not complete fn-164's Temporal migration or whole-spec requirements. Keep `.4`/`.5` and R6 open; public lambda finishers remain until the separately authorized final surface task.

**First proofs.** Prove the accepted native builder and standalone ordinary-function spellings before extending callers. Follow the existing Effects-style typed contexts, with bare fields reading the resulting state and before-state evidence only in `.across`. Preserve explicit contextual source symbols for exact parameter recovery by `.3`; do not erase names into an opaque runtime wrapper or claim unrun IR parity. Use per-evaluation private control signals for conjunctive and alternative short-circuiting, catching only owned signals so holes and unrelated exceptions propagate. Add source-aware inline validation for final/nonfinal discarded Boolean expressions and locally bound/discarded path or step construction; Unit adaptation and ambient Owner absence alone are insufficient. Exact composed selectors must validate both the authored field and selected machine, including repeated same-machine members. Owner-free ordinary predicates must retain evidence validated at attachment to their declaring composition, or prove an equally typed solution; Step alone does not establish that relationship.

**Size:** M
**Files:** expectation sugar/contexts and focused framework compile/runtime tests.
**Touches:** [model/framework/dsl/**, model/framework/check/lint/SyntaxRule*.scala]. Use one shared selector implementation if witnesses need the same exact-member correction; preserve existing witness regressions. Core storage redesign and generic lifting belong outside this native task unless an observed proof requires a minimal typed metadata seam.

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
make check-framework
```

Do not run heavyweight commands concurrently; Go, regeneration, full gates/review/live remain deferred.

## Acceptance
- [ ] Same-step/transition heads compile with typed before/after and bare field reads under machine, composition and generic Declares contexts; explicit contextual names support the preservation proof (R1/R2).
- [ ] Focused runtime tests prove conjunctive/alternative/conditional truth values and hole-aware short-circuit behavior, including conditional-only blocks (R1).
- [ ] Compile negatives cover before in same-step blocks, assignments, foreign facts, final/nonfinal discarded Boolean expressions, wrong-level temporal/path declarations and locally bound/discarded step-making expressions (R1-R4).
- [ ] Two-member composed fact tests check exact declared field/machine identity, and new sugar Core forms/syntax inventory preserve core-layer separation (R1/R7).
- [ ] Quick commands pass; no managed artifacts, Go or final public-finisher removal is claimed.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
