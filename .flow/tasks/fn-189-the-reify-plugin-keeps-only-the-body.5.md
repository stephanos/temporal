---
satisfies: [R2]
---
# fn-189-the-reify-plugin-keeps-only-the-body.5 One lifting rule: Lifted container marker, step-list type, no per-def @lifted

Touches: model/framework/dsl/{Reify,Machine}.scala, model/framework/reify/plugin/{ReifyPlugin,PluginContext}.scala (DefinitionStore, StorePhase), model/framework/reify/body/{BodyArguments,BodyContext,BodyEngine,BodySyntax,Expressions}.scala, model/framework/{README,SEMANTICS,REFERENCE}.md, model/README.md, model/framework/samples, model/framework/export/{ConstructedEndpoints,ConstructedRules,ReifyBoundary}.test.scala and migrated export/testdata reify fixtures

## Description
**DSL completion phase (owner, 2026-10-17):** scheduled before the IR/plugin phase because it changes what authors write or can reach; the gate is the native checks plus the plugin compiling the six samples.

**Current milestone execution boundary:** this task closes the native-source marker/carrier checkpoint, with focused full-mode helper storage and incremental controls. Plain-Machine var/effect lowering remains fn-190.9 work. Parent R2's full Model-with-effects compile and warm-edit proof remains open under fn-189.7 after that lowering lands; realization-mode sample compilation or a pure-helper fixture is not that proof. Preserve the original verification requirements below.

Introduce the marker trait `framework.dsl.Lifted`. The store phase writes liftedIr for a def when its owner is a Lifted container or a nested container within it, when its result type is a step list `List[Step[S, O, F]]`, or when it overrides Machine.end or Machine.evidence. Machine derives from Lifted through the framework, covering plain nested `object effects:` without sample changes; lexical coverage stops at method/local scopes, and local defs are not stored independently. Remove the public `lifted` annotation and the five role-specific carriers (sectionIr, facetIr, recordedStatusIr, declaredOutcomeIr and the namedResult marker's storage role) together with their site indexes, recursion guards and refusal replays in DefinitionStore; Recorded statuses and Ok outcomes are no longer lifted at definition but referenced by name and converted by their DSL values at construction. The `effects`/`rules` names are no longer recognised by spelling. A capture that reaches a def in another file that no rule covers is refused at the capture with `<fullName> is reached from another file: define it in an object that extends Lifted`; a full compile reports the same text. Same-file helpers lift from their tree as today; a test asserts tree-lift and stored IR agree for a def covered by both.

**Size:** L

### Verification
- Full compile of the six samples with a helper moved into a Lifted object in another file.
- Warm Bloop session (the prototype's run shape) editing only the helper body: the dependent Model recompiles and lifts the new IR; a def compiled without the plugin triggers the R2 refusal naming it.
- Focused dsl, reify and export suites pass; fixtures for sections, facets, statuses and outcomes are rewritten to the single carrier.

## Acceptance
- [x] No public @lifted annotation exists; Lifted container, step-list type and Machine.end/evidence override are the only storage rules.
- [x] Only liftedIr and capture remain as reify annotations in the DSL.
- [ ] Full-compile and warm-edit scenarios pass; the cross-file refusal text matches the spec in both.
- [x] Same-file tree lift and stored IR agree (test).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
