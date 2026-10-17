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
Phase 1 (DSL only). Facet traits declare the vars they share abstractly and the Machine defines them (the schema macro collects them in the Machine's order); phases are plain vars read in transitions and assigned in effects; Phased, Phasing, Inherited, PhasesOf, enters and the when/in phase stubs are removed natively, ends are explicit (`override def end`), and assigning a Recorded phase records nothing (explicit `record(status)`). The Facets fixture is rewritten with the same native meaning (equal steps and ends over every var combination); an undefined facet var is Scala's abstract-member error, and a facet var defined with an initializer is refused. FacetBindings stays hidden for the plugin; RecordFacet keeps record-state fixtures. Native checks: compile exit 0, 505 tests, 29 lint. Plugin/lifter/export references move with fn-190.9.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4c0275ff85, ff694dea83, e9102c5d3a, 9031033c78
- Tests: compile dsl realize samples: exit 0, test dsl realize samples: 505 passed, lint *RuleSuite: 29 passed
- PRs: