---
satisfies: [R2, R9, R10]
---
# fn-190-models-are-plain-scala-state-as-vars.2 State as vars: Machine[O, F], synthesised State record, init and schema from declarations, erasing bounded aliases

Touches: model/framework/dsl/Machine.scala, model/framework/dsl/Domain.scala, model/framework/dsl/FiniteSchema.scala, model/framework/dsl/MachineHeaders.scala, model/framework/reify/body, model/framework/reify/plugin/SchemaPhase.scala, model/framework/reify/plugin/ReifyPlugin.scala (EndpointsPhase)

## Description
**Phase 1 (DSL only), owner phasing 2026-10-17.** Change model/framework/dsl, realize and samples only; the gate is the native checks (dsl/realize/samples compile, their native suites pass, block-form lint). The plugin, lifter, export and IR are untouched: `Machine[O, F]` with declared vars; Property/Scenario/Query types lose the state parameter; until fn-190.3/.5 the native runtime reads and writes a Machine's vars as a snapshot (a record of their values) so effects, guards and claims execute natively and the sample suites keep their assertions. The plugin parts (State record synthesis, schema and init from declarations, plugin refusals) move to fn-190.9.

Remove the state type parameter: `Machine[O, F]`. The plugin collects the Machine's vars in declaration order, synthesises the IR record `<machine>.State`, derives its schema from the declared types (bounded aliases from the type tree, enums, records, Option, Set, Map, List, Inbox), and lifts `init` from the initialisers. `UpTo[N]` and the other bounded forms become transparent aliases of Int. Refuse a Machine without vars and a var whose type has no schema with the exact texts in the spec. This re-cuts fn-189 R4: the schema and header identity come from declarations in the plugin, so fn-189.6 is superseded; record that in fn-189's spec and MILESTONES. Property, Scenario and Query types lose their state parameter accordingly.

**Size:** L

### Verification
- A fixture Machine with every supported var type lifts to the expected State record and schema; the UpTo bound matches the retained intRange.
- The two refusal fixtures report the exact texts.
- Focused dsl and reify suites pass.
## Acceptance
- [ ] TBD

## Done summary
Phase 1 (DSL only). `Machine[O, F]` declares state as vars with a macro-derived schema (names, declared types, bounds, Inbox message types, comparisons), a typed `State` record and the native snapshot runtime (restore before each step/claim, reset to init after; accessors found by name); `before` is the typed State record; Property and Scenario lose the state parameter. Refusals with the spec's texts, tested with compileErrors: a Machine without vars, a var with no schema, `before` where no step started. All six samples and NativeInputs migrated, every native test keeping its cases (states built from init by named updates). Bridge: the DSL's own ~177 fixtures stay on `RecordMachine[S, O, F]` (shared effects and case-class assertions) until fn-190.4/.7; var forms of where/~>/leadsTo/once/keeps exist as extensions. Native checks: compile exit 0, 481 tests, 29 lint tests (block-form 128 files, 0 findings). Deferred: `UpTo` as an erasing alias (breaks finite catalogs until enumeration is removed: fn-190.5/.9); outcome/fact as abstract type members (engine reparameterisation: fn-190.4/.7); choose and phases on var machines, private vars (fn-190.4/.6).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 05fdc2e, 2253c76, 2444303, c30fab3, e08f355, e0f49b1, 6ec0366, b429e46, 589c444, 3981e0f, 226bc15, eaec94c, a491f19
- Tests: compile dsl realize samples: exit 0, test dsl realize samples: 481 passed, lint *RuleSuite: 29 passed
- PRs: