---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-164-expect-blocks-for-properties.2 Implement typed expectation blocks and short-circuit runtime semantics

## Description
**Size:** M
**Files:** expectation sugar/contexts and focused framework compile/runtime tests.
**Touches:** [model/framework/**, model/check/SyntaxRule.scala, model/check/*test.scala]

Implement R1-R3's API and the runtime foundation of R4, keeping core Property declaration storage unchanged. Existing public lambda finishers remain temporarily for migration and are removed together in the final surface task.

### Approach
- Add typed expectation contexts and builder extensions in the sugar layer, not core Claims. Use the spec's exact-class `after`, distinct whole-action `afterAction`, and transition `.across` head; unqualified property and generic `Declares` capability/composition contexts work without machine-only Owner.
- Prove the selected Scala heads with a small compile fixture before implementing callers. Offer explicit contextual binding names that can later reify original core before/Step parameter names; include `s`, `(b,a)` and compiler-generated source parameters. Ordinary blocks retain implicit `before`/typed `after` and bare resulting-state fields. Keep reusable predicate bridges possible without a second Boolean implementation.
- Implement Unit requirements, conditional blocks and oneOf's explicit branch grouping with immediate failure and isolated alternatives. Test failed check before a hole-bearing later guard, unsuccessful branch followed by success, success skipping a later hole, all alternatives false and an actual hole propagating. Do not catch unrelated exceptions as assertion failure; conditional-only blocks are not runtime-empty declarations.
- Retain typed fact identity: machine foreign fact compilation refusal and exact field-to-machine composed selectors. Assert two members with the same fact name cannot satisfy each other. Reuse witness selection without suffix or Any erasure.
- Add compiler-enforced predicate-scope protection for forbidden temporal/path declarations and step-making expressions, even if bound to local vals or discarded; inherited Owner does not authorize effects here. Before in same-step blocks and assignment assertions fail compilation. Unit return typing alone is insufficient for discarded final Boolean expressions; validate them explicitly.
- Every added sugar entry has its Core form, and the syntax gate inventory recognizes it without core exemptions. Preserve immutable helper lets needed by existing predicates.

### Investigation targets
**Required:** `model/framework/Claims.scala:19-48` (core predicates); `model/framework/Syntax.scala:106-171` (View/context convention); `model/framework/Machine.scala:75` (Declares); `model/framework/Witness.test.scala` (typed composed facts); `model/check/SyntaxRule.scala` (sugar/core boundary); `model/framework/Effects.test.scala` (compile/runtime harness).
**Optional:** `model/framework/Monitor.scala`, `model/framework/Assume.scala` (outer constructor scope).

### Quick commands
```sh
mise exec -- scala-cli test model/project.scala model/framework --test-only '*Expect*'
make lint-model-syntax
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
