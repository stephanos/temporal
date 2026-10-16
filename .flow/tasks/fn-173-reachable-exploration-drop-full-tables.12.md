---
satisfies: [R2, R3, R8]
---
# fn-173-reachable-exploration-drop-full-tables.12 Native reach/open/checker authoring and Query total retirement

## Description
Finish native authoring portions currently mixed into .2/.3/.8. Backend execution, export coverage admission and reached-graph semantics stay held.

**Size:** M
**Files:** framework DSL Claims/Machine/Syntax, native Query tests/examples and local guidance.
**Touches:** [model/framework/dsl/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Implement reach/open/checkedBy/confirm under the parent's Native query configuration policy. Reuse Limits in Claims.scala:90 and the shared Declares ownership seam in Machine.scala:60, including Composition. Do not introduce a separate composition settings language or mutable global default.
- Pin frameworkReach's exact fields independently. Validate reach at native consumption, without toIR/export and without globally changing ordinary Limits zero validity. Preserve all inherited/replaced reach settings and avoid base-constructor reads of overridden vals.
- Consolidate configuration copying across Query live/explore at Claims.scala:115 and Witness requiring/ends/limits/live/explore at Claims.scala:223. Preserve specialized Witness authoring after new modifiers, exact predicates/selector/owners/names/limits/expectedRun/exploration, and source reach from the Scenario owner rather than the Property owner.
- Retire public Query.total authoring and migrate native callers/docs with no compatibility alias. Internal IR/carrier reader retirement remains .3's later obligation; do not force a schema migration now.
- Apply the explicit single-assignment/open/checker/confirmation policy; preserve exact reason text and authored confirmation order. Accept all three checker identifiers as metadata without availability/coverage checks. Prove the checkedBy-before-confirm ordering rule, not universal setter commutativity.
- Remove the four native ReliableBroadcast total clauses and obsolete count comment, and replace Witness.test.scala's public total getter assertion with independent declaration pins. External negative clients must refuse both getter and writer, including Witness. A private compile-only carrier bridge, if needed, earns no held integration credit.
- Keep configuration distinct from a checked answer. No backend invocation, export, replay, coverage admission or apparent successful result is introduced by this task.

### Investigation targets
**Required:** model/framework/dsl/Claims.scala; model/framework/dsl/Machine.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/IrFile.test.scala; model/design/reachable-exploration.md.

### Verification
Run native query/machine/syntax suites and external author compile tests with nonzero counts. No schema/IR/exporter/Go/backends or Temporal inputs.

Use the focused native gate shape `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only '<native suite>'`. Re-anchor concrete selectors after committed .11. Do not run whole-state table helpers as admission gates. External clients use the native strict warning flags, including Werror/deprecation, and no old Model/Temporal jars. Independent poison callbacks must show metadata construction, copying and inspection performs no model execution.

## Acceptance
- [ ] Native reach/open/checker/confirmation authoring retains exact typed configuration, defaults, subject and limits; malformed configuration rejects.
- [ ] Machine/composition defaults, intentional overrides, derivations/replacement and refined-Property/Scenario ownership retain reach without initialization hazards; bounds reject negatives/zero search while zero depth remains valid.
- [ ] Find/verify and ordinary Query/Witness chains preserve every unrelated field and specialized authoring in both modifier orders where permitted. Nonblank/repeated open, repeated primary, empty/duplicate/repeated/self confirmation, and primary-after-confirm cases have independent diagnostics/control tests.
- [ ] Public total authoring is inaccessible externally and native callers/docs use the final surface.
- [ ] Independent ordinary Query declaration pins and native tests pass; configuration is never reported as backend execution or a checked answer.
- [ ] Full R2/R3 engine/carrier retirement and R8 coverage/replay/backend proofs remain open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
