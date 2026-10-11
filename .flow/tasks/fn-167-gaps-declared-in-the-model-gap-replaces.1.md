---
satisfies: [R1, R2, R8]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.1 Typed @gap surface, annotation extraction and source admission

## Description
Implement the annotation surface and typed source admission (R1, R8), with the first TASTy proof for R2. This task stays within Scala sources.

**Size:** M
**Files:** new framework gap declarations/tests; typed convention checks and fixtures; authoring docs.
**Touches:** [model/framework/Gaps.scala, model/framework/Gaps.test.scala, model/irgen/Conventions.scala, model/irgen/test/Conventions.test.scala, model/irgen/test/Gaps.test.scala, model/irgen/testdata/gaps/**, model/README.md, model/SEMANTICS.md]

### Approach
- Extend existing reference types from Action rather than adding string identifiers or a second action/class vocabulary. Inspect every current lint kind before finalizing constructor signatures, including the optional kind.
- Establish repeated curried annotation extraction from compiled fixtures before broadening the surface. Probe object/module mirrors, annotated vals, omitted expiry, typed parameters and source positions. Sort by source position; distinguish compiler mirrors from repeated user gaps.
- Add the convention rule using the indexed typed source, preserving existing compiler annotations. Check helper and test placement even when no semantic root is lifted.
- Leave semantic owner-relative resolution to task 2. Document the staged source-only behavior, not a completed consumer switch.

### Investigation targets
Required:
- `model/framework/Action.scala:211` — existing Class and ClassRef.
- `model/irgen/Context.scala:18` — complete typed source index.
- `model/irgen/Conventions.scala:306` — convention traversal.
- `model/irgen/test/Conventions.test.scala` — compiled positive/refusal fixtures.
- `tools/umpire/lint/lint.go:28` — read-only kind inventory.

### Verification
Add proposed `framework.GapsSuite` and `umpire.irgen.GapsSuite` selectors. Run those plus existing ConventionsFixtures with `mise exec -- scala-cli test ... --server=false --test-only ...`; assert tests actually execute. Serialize compiler-heavy suites with any active owner. Capture compiler/TASTy mirror and source-order fixtures as the early proof, not just successful compilation.

## Acceptance
- [ ] Typed constructor inventory and convention admission tests execute and pass.
- [ ] Compiled TASTy fixtures prove source order, object mirror handling and supported placement; invalid helper/test/foreign-constructor controls fail.
- [ ] Docs describe only the staged source-only behavior; protected artifacts and ordinary compiler annotations are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
