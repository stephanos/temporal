---
satisfies: [R2]
---
# fn-189-the-reify-plugin-keeps-only-the-body.5 One lifting rule: Lifted container marker, step-list type, no per-def @lifted

Touches: model/framework/dsl/Reify.scala, model/framework/dsl/Machine.scala, model/framework/reify/plugin/ReifyPlugin.scala (DefinitionStore, StorePhase), model/framework/reify/plugin/PluginContext.scala, model/framework/samples, model/framework/export reify fixtures

## Description
Introduce the marker trait `framework.dsl.Lifted`. The store phase writes liftedIr for a def when its owner derives from Lifted, when its result type is a step list `List[Step[S, O, F]]`, or when it overrides Machine.end or Machine.evidence. Machine's own `effects` object derives from Lifted through the framework so no sample changes. Remove the public `lifted` annotation and the five role-specific carriers (sectionIr, facetIr, recordedStatusIr, declaredOutcomeIr and the namedResult marker's storage role) together with their site indexes, recursion guards and refusal replays in DefinitionStore; Recorded statuses and Ok outcomes are no longer lifted at definition but referenced by name and converted by their DSL values at construction. The `effects`/`rules` names are no longer recognised by spelling. A capture that reaches a def in another file that no rule covers is refused at the capture with `<fullName> is reached from another file: define it in an object that extends Lifted`; a full compile reports the same text. Same-file helpers lift from their tree as today; a test asserts tree-lift and stored IR agree for a def covered by both.

**Size:** L

### Verification
- Full compile of the six samples with a helper moved into a Lifted object in another file.
- Warm Bloop session (the prototype's run shape) editing only the helper body: the dependent Model recompiles and lifts the new IR; a def compiled without the plugin triggers the R2 refusal naming it.
- Focused dsl, reify and export suites pass; fixtures for sections, facets, statuses and outcomes are rewritten to the single carrier.

## Acceptance
- [ ] No public @lifted annotation exists; Lifted container, step-list type and Machine.end/evidence override are the only storage rules.
- [ ] Only liftedIr and capture remain as reify annotations in the DSL.
- [ ] Full-compile and warm-edit scenarios pass; the cross-file refusal text matches the spec in both.
- [ ] Same-file tree lift and stored IR agree (test).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
