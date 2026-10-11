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
Added the typed `gap` annotation and all 21 existing lint subjects, with source admission and internal TASTy extraction. Compiled isolated fixtures preserve authored order, collapse object/module mirrors, retain repeated user annotations, admit supported declarations and reject helpers, tests, lookalike annotations, foreign factories, empty gaps and blank reasons. Documentation describes the source stage and leaves semantic resolution and shared transport to later tasks.

baseline: green — existing `umpire.irgen.ConventionsFixtures`, 10 executed tests before edits (`baseline.log`). No full-suite baseline or equivalence claim.

Focused verification: `framework.GapsSuite` 2/2 (`framework-final.log`), `umpire.irgen.GapsSuite` 2/2 (`lifter-fixed.log`), existing `umpire.irgen.ConventionsFixtures` 10/10 (`conventions-fixed.log`); all final suite exits were 0. Scoped Scala formatting and `git diff --check` passed. Logs are in `.flow/tmp/fn167/task1/`.

Red evidence: framework surface initially failed compilation because the new types did not exist (`framework-red.log`); the compiled extraction test passed before admission was implemented while the admission test failed (`early-proof.log`). Existing convention verification caught an eager lookup of the new Subject in the pre-gap packaged jar (`conventions-final.log`); making that lookup lazy restored all 10 existing tests without rebuilding the jar. Earlier missing-type/import and assertion-spelling failures remain in the logs.

R1 source errors (empty annotation subject list and blank reason) and R8 errors (helper def/value/object/parameter, test placement, lookalike gap/acceptance annotation and foreign factory) are exercised by the table in `umpire.irgen.GapsSuite`. Framework string-identity refusal controls compile-check real constructor calls. R2's initial source proof checks curried annotations, omitted expiry, typed phases/classes, source positions, four raw compiler mirror annotations versus two emitted gaps, annotated vals and genuine repeated user gaps. Owner-relative matching, duplicate finding-key refusal, expiry validation and IR serialization remain later tasks.

stage: impl-review - skipped(policy: owner-approved Scala batch-end)
Broad validation and independent review deferred to the Scala batch end by the owner's fast-validation contract. `gate classify` returned FULL (unmatched unrelated `IDEAS.md`); no docs-only skip or full-suite green receipt is claimed. No full source/lifter/Go suite, equivalence, canonical regeneration or shared schema generation ran.

Tier: session (jev-unavailable(no_key)); project implementer routing retained

Unrelated IDEAS, MILESTONES, fn168 files and externally modified Flow specs were preserved. No real-feature source or generated artifact was changed. Milestone reconciliation remains conductor-owned.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a983e042baba71259b0e0fc13848ec85b13f260a
- Tests: mise exec -- scala-cli test model/project.scala model/framework --server=false --test-only 'framework.GapsSuite', mise exec -- scala-cli test model/irgen --server=false --test-only 'umpire.irgen.GapsSuite', mise exec -- scala-cli test model/irgen --server=false --test-only 'umpire.irgen.ConventionsFixtures', mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf model/framework/Gaps.scala model/framework/Gaps.test.scala model/irgen/Conventions.scala model/irgen/test/Gaps.test.scala model/irgen/testdata/gaps, git diff --check
- PRs: