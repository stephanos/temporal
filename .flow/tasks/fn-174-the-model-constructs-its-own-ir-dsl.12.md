---
satisfies: [R3, R4, R6, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.12 Retain and construct native composed action references in isolation

## Description
Retain exact native composition selections and construct isolated ComposedActionRef contributions (R3/R4/R6/R9). This source checkpoint supplies the reference capability shared by complete Assumption conversion and future composition conversion. It does not emit a complete Composition or activate production export.

**Size:** M
**Files:** Compose.scala, Assume.scala, MemberPath.scala, co-located composed-reference converter/tests, plugin signature attachment phase, compiled-client fixtures and README. StepTables.scala may adapt only to a retained Syncs.Statement shape without changing exploration semantics.
**Touches:** [model/framework/dsl/Compose.scala, model/framework/dsl/Assume.scala, model/framework/dsl/MemberPath.scala, model/framework/dsl/ComposedReference.scala, model/framework/dsl/ComposedReference.test.scala, model/framework/dsl/StepTables.scala, model/framework/reify/plugin/SchemaPhase.scala, model/framework/reify/plugin/ReifyPlugin.scala, model/framework/export/ConstructedReferences.test.scala, model/framework/export/testdata/constructedReferences/**, model/README.md]

### Approach
- Prove the existing ComposedProgress fixture first. Its five fairRefs cover own/synced, bare action/explicit class and two member fields containing the same Tanks machine. Read its frozen old output independently; do not regenerate that output from new conversion.
- Retain compiler-authored direct member selectors, own/synced operation, actual Action/Class references, bare-versus-explicit selection and sync endpoint fields. A zero-input bare action and an explicitly selected empty class remain different. Reuse MemberPath's direct-field policy; do not guess paths by executing callbacks on fabricated state or infer ownership by object equality.
- Extend the existing post-pickler signature attachment seam, preserving saved TASTy consumed by the current reader. Preserve ordinary Scala evaluation, public author types, helper forwarding and existing native selector behavior. New attachment must not invoke a supplied selector or evaluate its action/class expression twice.
- Retain an effective ordered field-to-member map through Composition.Shape.Members, derived Of/With and withMember, replacing the captured field and inheriting base syncs. Current Shape.With.members appends the replacement; never zip that list with mirror labels or find a member by machine identity. Reference lookup needs the effective captured map, without expanding this task into full composition conversion. Read native Machine.table bound declarations only. Do not execute guards/effects, enumerate states/catalogs, run StepTables exploration or require complete machine/composition IR. Leave full replacement/refinement/type admission with original .5.
- Construct exact ir.ComposedActionRef variants. Bare synced selects sync; explicit synced selects syncClass; bare own selects member plus canonical action ID; explicit own selects member plus actionClass. Reuse retained action IDs/schema keys and existing Construction.actionClass input encoding/admission. Bare input-taking actions denote every class and bypass class encoding.
- Validate actual selected-member action ownership, missing/ambiguous sync endpoint matches, own selecting a paired action, unique sync names and one-sided sync-class compatibility. The opposite endpoint must have no input when one side names a specific class. Repeated machine instances in distinct fields remain distinct coordinates. Preserve order and canonical identities through aliases/helpers/derived forms.
- Make Assumption.fair chains lossless, retaining both ordered ordinary and composed selections across chained calls, including duplicates/empty segments. This task supplies no complete Assumption message. Keep mixed/multiple-owner selections available for the subsequent converter's diagnostics; do not silently erase either category.
- Keep production reader active and unchanged. Add separate-JVM deterministic compiled/native checks and contextual missing-metadata/invalid-selector failures. Helper forms newly supported by metadata get qualified native tests, not historical parity claims.

### Investigation targets
**Required:**
- model/framework/dsl/Compose.scala:68 - native shape, selectors, Syncs.Statement and erased Composed.selected.
- model/framework/dsl/MemberPath.scala:6 - existing direct-field metadata policy.
- model/framework/dsl/Machine.scala:493 - native bound action declarations, including derived restrictions.
- model/framework/export/Compositions.scala:476 - exact reference variants and selection validation.
- model/framework/dsl/Signature.scala:72 - retained action schemas/IDs; realize/Conversion.scala:219 for shared class encoding.
- model/framework/export/testdata/lifts/ComposedProgress.scala:44 - repeated-member independent oracle; expected/composedProgress.json and ComposedProgressRejects.scala.
- model/framework/export/ConstructedSignature.test.scala:14 - isolated framework/plugin/client packaging and two-JVM comparisons.

### Verification and boundaries
Run the new native/compiled reference suites, affected DSL/realization, ConstructedSignature, ConstructedNames and reify checks; include affected StepTables checks if its consumer changes. Freeze source before final affected checks and configured scoped formatting. Keep all six samples unchanged read-only regressions. No model/temporal, tools/umpire, Go, root Model/Make runs, schemas/generated outputs, registry/module-map/migration edits. Root owns Flow/MILESTONES and completion. Independent review and broad validation remain at the owner-authorized source-batch end.

### Planning receipt
SHORT delta research used current native composition/reference contracts, full dependency graph and active memory. This capability precedes complete residual signature conversion without adding a .3 dependency on full .5, which would cycle through .4. Original full .2/.3 gates and fn-169.2 remain unchanged. No new library or reference registry is added. Historical memory paths supply lessons, not current-defect findings.

stage: plan-review - skipped(policy: owner defers independent review to source-batch end)
stage: html-render - skipped(policy: owner focus remains framework code and bounded tracking)
memory: bm25 (jev-unavailable(no_key)); targeted memory scout completed.

## Acceptance
- Exact compiler-retained member paths, selection operations, Action/Class identity and sync endpoint metadata survive aliases, helper jars and required derived composition shapes. Bare zero-input actions remain distinct from explicit empty classes. Author typing and ordinary evaluation remain unchanged; callback/action evaluation-count controls pass.
- Isolated native ComposedActionRef conversion matches the five frozen old ComposedProgress fairRefs in order, stripping only positions. All four variants and repeated same-machine member fields are covered. Two separate JVM projections are identical.
- Missing metadata, invalid/nested/opaque selectors, missing member/action ownership, foreign same-spelling actions, missing/ambiguous sync matches, duplicate sync names, own selecting a paired action and wrong-sided sync classes refuse contextually. Bare input-taking actions remain all-class references; selected class values use existing schema encoding and catalog admission.
- Chained Assumption.fair calls retain every ordered ordinary/composed selection, including duplicates and empty segments, so mixed chains in either order and multiple composition owners remain diagnosable. No complete Assumption or Composition message is claimed.
- No conversion runs guards/effects, explores transition/state tables, enumerates catalogs to discover ownership, or alters production export. Frozen affected tests and configured scoped formatting pass; README states the dormant contribution and deferred consumer-owner/refinement checks. Full .2/.3/R5, reader retirement and held integration remain open.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
