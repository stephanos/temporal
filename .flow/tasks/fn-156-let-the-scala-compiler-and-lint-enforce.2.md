---
satisfies: [R2]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.2 Activate strict equality across authoring and lift builds

## Description
Complete R2 using task 1's proven evidence placement. This is the mechanical equality migration across all independent Scala build roots; it serializes the files that later convention and law work uses.

**Size:** M
**Files:** shared and fixture compiler directives, `model/.scalafix.conf`, framework/Temporal/irgen comparison types and tests.
**Touches:** [model/project.scala, model/irgen/project.scala, model/irgen/testdata/**, model/.scalafix.conf, model/framework/**, model/temporal/**, model/irgen/*.scala, model/irgen/test/**, .flow/tmp/fn156/**]

### Approach

- Turn strictEquality on for the framework, Models, irgen and positive lift fixtures. Add precise CanEqual declarations for remaining compared types that do not derive Finite; retain generic method equality constraints. Avoid declarations that admit unrelated levels.
- Enable DisableSyntax.noUniversalEquality. Re-fetch its official documentation for the pinned Scalafix version and prove how its syntactic finding interacts with compiler-checked native equality. Use existing line-scoped suppression only where required, with its reason, and keep unsuppressed negative lint examples. No file-wide/project-wide suppression or weakening of strictEquality.
- Preserve source locations while changing comparisons/imports/derivations, and avoid rewriting comparisons into a form the lifter cannot read. Reuse task 1's immutable baseline for the exact scratch output comparison.
- Test standalone framework compilation, authoring tests, irgen and all fixture projects, not only the original scratch compile's production subset. Check precise retained refusal diagnostics and negative cross-type compilation after migration.

### Investigation targets

**Required:**
- `model/.scalafix.conf:22` - noUniversalEquality and existing rule configuration.
- `Makefile:674` - JDK 25 check and rewrite callers.
- `model/framework/Domain.scala:39` - equality evidence proven in task 1.
- `model/framework/Inputs.test.scala:89` - compiler negative-test precedent.
- `model/temporal/capabilities/Closable.test.scala:26` - model compile-test precedent.
- `model/irgen/test/Fixtures.test.scala:830` - existing warning and crossed fixture causes.

### Quick commands

Run framework/authoring tests and `make lint-model` through the existing explicit-root/JDK 25 setup. Run the real irgen fixture suite, preserving output and status. Production-sized commands require the shared heavy lock and immutable comparison inputs.

## Acceptance
- [ ] Strict equality applies to every required build root, all intended same-type comparisons compile, and unrelated model-type comparisons fail.
- [ ] noUniversalEquality is enabled; unsuppressed negatives fail and precise permitted typed model comparisons pass under JDK 25 without broad rule suppression.
- [ ] Every non-Finite comparison type has sound explicit evidence; compiler/refusal suites retain their complete scope and original causes.
- [ ] Scratch regeneration remains exactly equal to task 1's baseline, including source positions and all artifact files.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
