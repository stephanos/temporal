---
satisfies: [R2, R9, R10]
---
# fn-190-models-are-plain-scala-state-as-vars.2 State as vars: Machine[O, F], synthesised State record, init and schema from declarations, erasing bounded aliases

Touches: model/framework/dsl/Machine.scala, model/framework/dsl/Domain.scala, model/framework/dsl/FiniteSchema.scala, model/framework/dsl/MachineHeaders.scala, model/framework/reify/body, model/framework/reify/plugin/SchemaPhase.scala, model/framework/reify/plugin/ReifyPlugin.scala (EndpointsPhase)

## Description
Remove the state type parameter: `Machine[O, F]`. The plugin collects the Machine's vars in declaration order, synthesises the IR record `<machine>.State`, derives its schema from the declared types (bounded aliases from the type tree, enums, records, Option, Set, Map, List, Inbox), and lifts `init` from the initialisers. `UpTo[N]` and the other bounded forms become transparent aliases of Int. Refuse a Machine without vars and a var whose type has no schema with the exact texts in the spec. This re-cuts fn-189 R4: the schema and header identity come from declarations in the plugin, so fn-189.6 is superseded; record that in fn-189's spec and MILESTONES. Property, Scenario and Query types lose their state parameter accordingly.

**Size:** L

### Verification
- A fixture Machine with every supported var type lifts to the expected State record and schema; the UpTo bound matches the retained intRange.
- The two refusal fixtures report the exact texts.
- Focused dsl and reify suites pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
