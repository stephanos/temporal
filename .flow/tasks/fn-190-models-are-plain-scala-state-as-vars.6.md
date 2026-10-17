---
satisfies: [R5]
---
# fn-190-models-are-plain-scala-state-as-vars.6 Facets and phases as vars; remove Phased and the contextual-phase machinery

Touches: model/framework/dsl/Facet.scala, FacetBindings.scala, FacetSource.scala, Machine.scala (Phased), Syntax.scala (when/enters), model/framework/reify/plugin (facet payloads, contextual sites), model/framework/dsl/Facets.test.scala

## Description
**Phase 1 (DSL only), owner phasing 2026-10-17.** Change model/framework/dsl, realize and samples only; the gate is the native checks (dsl/realize/samples compile, their native suites pass, block-form lint). The plugin, lifter, export and IR are untouched: facet vars, `when(phase)`/`enters(phase)` on vars and the rewritten Facets fixture natively; removing `Phased` and the facet payloads from the plugin moves to fn-190.9.

A facet trait declares `var phase: Phase` and any shared state abstractly; the Machine defines them; `when(phase)` and `enters(phase)` read and write the var. Remove `Phased`, the projection and writer, `$facet-context-v1` placeholders, substitution, FacetOutcome and the three facet payloads from the plugin. Rewrite the Facets fixture; a facet var the Machine does not define is Scala's abstract-member error. This closes the facet boundary item of fn-189.

**Size:** L

### Verification
- The Facets fixture lifts to the same bindings as before except positions and the phase access shape.
- Focused dsl and reify suites pass.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
