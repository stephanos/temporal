---
satisfies: [R2]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.2 Activate strict equality across authoring and lift builds

## Description

Complete R2 using task 1's proven evidence placement. This is the mechanical equality migration across all independent Scala build roots; it serializes the files that later convention and law work uses.

**Size:** M
**Files:** shared and fixture compiler directives, `model/.scalafix.conf`, framework/Temporal/irgen/gate comparison types and tests, including `model/check/project.scala`.
**Touches:** [model/project.scala, model/irgen/project.scala, model/irgen/testdata/**, model/.scalafix.conf, model/framework/**, model/temporal/**, model/irgen/*.scala, model/irgen/test/**, model/check/**, .flow/tmp/fn156/source/**]

## Approach

- Turn strictEquality on for the framework, Models, irgen and positive lift fixtures. Add precise CanEqual declarations for remaining compared types that do not derive Finite; retain generic method equality constraints. Avoid declarations that admit unrelated levels.
- Enable DisableSyntax.noUniversalEquality. Re-fetch its official documentation for the pinned Scalafix version and prove how its syntactic finding interacts with compiler-checked native equality. Use existing line-scoped suppression only where required, with its reason, and keep unsuppressed negative lint examples. No file-wide/project-wide suppression or weakening of strictEquality.
- Migrate every shared lint consumer before requiring make lint-model to pass, specifically model/check and its tests at Makefile:742. Enable strictEquality in the gate tooling root and add sound evidence for legitimate comparisons, then use only proven exact supported suppressions for typed native equality. Keep its documented no-Werror exception and diagnostics-aware process seam unchanged. Gate positives and unrelated-type/compiler plus unsuppressed syntactic-lint negatives prove the policy independently.
- Preserve source locations while changing comparisons/imports/derivations, and avoid rewriting comparisons into a form the lifter cannot read. Reuse task 1's immutable baseline for the exact scratch output comparison.
- Test standalone framework compilation, authoring tests, irgen and all fixture projects, not only the original scratch compile's production subset. Check precise retained refusal diagnostics and negative cross-type compilation after migration.

## Investigation targets

**Required:**
- `model/.scalafix.conf:22` - noUniversalEquality and existing rule configuration.
- `Makefile:674` - JDK 25 check and rewrite callers.
- `Makefile:742`, `model/check/project.scala:7` and `model/check/Gate.scala:161` - shared gate lint root, existing flag exception and comparison site.
- `model/framework/Domain.scala:39` - equality evidence proven in task 1.
- `model/framework/Inputs.test.scala:89` - compiler negative-test precedent.
- `model/temporal/capabilities/Closable.test.scala:26` - model compile-test precedent.
- `model/irgen/test/Fixtures.test.scala:830` - existing warning and crossed fixture causes.

## Quick commands

Run framework/authoring tests and `make lint-model` through the existing explicit-root/JDK 25 setup. Run the real irgen fixture suite, preserving output and status. Production-sized commands require the shared heavy lock and immutable comparison inputs.

- [ ] Strict equality applies to every required build root including gate tooling/tests reached by shared lint; all intended same-type comparisons compile, and unrelated model-type comparisons fail.
- [ ] noUniversalEquality is enabled; unsuppressed negatives fail and precise permitted typed Model/gate comparisons pass under JDK 25 without broad rule suppression; the gate's no-Werror and diagnostic-aware behavior remain intact.
- [ ] Every non-Finite comparison type has sound explicit evidence; compiler/refusal suites retain their complete scope and original causes.
- [ ] Scratch regeneration remains exactly equal to task 1's baseline, including source positions and all artifact files.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Acceptance
- [ ] TBD
