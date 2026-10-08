---
satisfies: [R7]
---
# fn-140-one-sentence-witness-queries-with.5 Document witness authoring and update the layout template

Touches: [model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, model/irgen/testdata/layout/lamp/product/Product.scala, model/irgen/testdata/layout/lamp/system/System.scala, model/irgen/test/Fixtures.test.scala]

## Description
Update author documentation and the existing layout template for R7. This task runs beside .4 on disjoint files and defers managed-output publication and full integration gates to .6.

**Size:** M
**Files:** `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, the lamp template's level files, existing layout fixture assertions if needed.

### Approach
- Rewrite README's path/Property/Scenario/Query example at `:62` around one witness, then give its exact ordinary core form and explain when shared triples or query verify remain appropriate. Update live Case generation at `:1325` and naming/read-order guidance without adding new semantics.
- Document the source-only reason rule for convenience, full Run and monitor expectations; distinguish the explanatory text from existing judge reason IDs and expected_run metadata.
- Add one ordinary witness to the existing lamp template while keeping its verify/refinement example. Exercise the template through the established layout fixture; keep its package-only root in normal compilation, per the current build-errors memory.
- Describe witness ownership and no-new-IR lifting in the module map. Remove obsolete Query expect examples from author documentation.
- Do not run an independent regeneration or full gate while .4 owns its proof baseline. .6 integrates the template and runs those checks once.

### Investigation targets
**Required:**
- `model/README.md:62-84` - initial author example.
- `model/README.md:1102` and `:1325` - layout and live generation.
- `model/SEMANTICS.md:923` - generated Case expectations.
- `model/irgen/testdata/layout/lamp/product/Product.scala:37` - template claims.
- `model/irgen/testdata/layout/lamp/system/System.scala` - template refinement.
**Optional:**
- `.plans/UMPIRE_MODULES.md:28` - DSL/lifter ownership.

### Quick commands
```bash
rg -n '\.expect\(' model/README.md model/SEMANTICS.md .plans/UMPIRE_MODULES.md
git diff --check
```

## Acceptance
- [ ] Author docs explain witness versus verify/shared triple, show the core triple, and identify live as the Case-generation switch (R7).
- [ ] The template contains a witness alongside its invariant/refinement example, with a focused layout fixture assertion.
- [ ] Source-only explanation text and existing emitted judge reason IDs remain distinct in the documentation.
- [ ] Documentation/task .4 ownership stays disjoint; this task publishes no model/ir or model/cases outputs.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
