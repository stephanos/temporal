---
satisfies: [R7]
---
# fn-140-one-sentence-witness-queries-with.5 Document witness authoring and update the layout template

Touches: [model/README.md, model/SEMANTICS.md, model/irgen/testdata/layout/lamp/product/Product.scala, model/irgen/testdata/layout/lamp/system/System.scala, model/irgen/test/Fixtures.test.scala]

## Description
Update author documentation and the existing layout template for R7. This task follows .4 in the owner's single-worker Scala lane. Only model/ authoring documents and template files are edited; module-map documentation, managed publication and full integration gates remain in deferred .6.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, the lamp template's level files, existing layout fixture assertions if needed.

### Approach
- Rewrite README's path/Property/Scenario/Query example at `:62` around one witness, then give its exact ordinary core form and explain when shared triples or query verify remain appropriate. Update live Case generation at `:1590` and naming/read-order guidance without adding new semantics.
- Document the source-only reason rule for convenience, full Run and monitor expectations; distinguish the explanatory text from existing judge reason IDs and expected_run metadata.
- Add one ordinary witness to the existing lamp template while keeping its verify/refinement example. Exercise the template through the established layout fixture; keep its package-only root in normal compilation, per the current build-errors memory.
- Describe witness ownership and no-new-IR lifting in Model author documentation. Remove obsolete Query expect examples there. The matching .plans/UMPIRE_MODULES.md update stays in deferred .6, because the current owner mode allows source edits under model/ only.
- Preserve .4's committed proof baseline; .10 includes this documentation/template in its final source seal. Run the template's focused Scala compile/lift assertions positively with a nonzero population, without a production regeneration or full gate. Deferred .6 consumes this source and closes the actual full checks once at fn-123.8.
- Preserve the user's existing uncommitted model/SEMANTICS.md draft in the main checkout. Read it for overlap and reconcile required localized documentation edits with the conductor before changing or merging that file; do not copy or commit unrelated draft changes.

### Investigation targets
**Required:**
- `model/README.md:62-84` - initial author example.
- `model/README.md:1127` and `:1590` - layout and live generation.
- `model/SEMANTICS.md:984` - generated Case expectations.
- `model/irgen/testdata/layout/lamp/product/Product.scala:37` - template claims.
- `model/irgen/testdata/layout/lamp/system/System.scala` - template refinement.
**Optional:**
- `.plans/UMPIRE_MODULES.md:28` - DSL/lifter ownership.

### Quick commands
```bash
rg -n '\.expect\(' model/README.md model/SEMANTICS.md
git diff --check
```

## Acceptance
- [ ] Author docs explain witness versus verify/shared triple, show the core triple, and identify live as the Case-generation switch (R7).
- [ ] The template contains a witness alongside its invariant/refinement example, with a focused layout fixture assertion.
- [ ] Source-only explanation text and existing emitted judge reason IDs remain distinct in the documentation.
- [ ] Documentation/task .4 ownership stays disjoint and the single-worker lane is retained; this task edits only model/ and publishes no model/ir or model/cases outputs. Module-map documentation remains required in deferred .6.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
