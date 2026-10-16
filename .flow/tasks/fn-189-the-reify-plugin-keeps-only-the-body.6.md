---
satisfies: [R4]
---
# fn-189-the-reify-plugin-keeps-only-the-body.6 Machine shape by declaration: init/end/evidence as members, schemas and header identity derived at runtime

Touches: model/framework/dsl/Machine.scala, model/framework/dsl/MachineEndpoints.scala, model/framework/dsl/MachineHeaders.scala, model/framework/dsl/FiniteSchema.scala, model/framework/dsl/Domain.scala, model/framework/dsl/Action.scala, model/framework/reify/plugin/ReifyPlugin.scala (EndpointsPhase), model/framework/reify/plugin/SchemaPhase.scala

## Description
Machine declares `def init: S`, `def end(state: S): Boolean = true` and `def evidence: PartialFunction[F, String] = PartialFunction.empty`; Phased overrides end with the projection-based default the plugin synthesises today. The plugin recognises end and evidence as overrides of those symbols and lifts them through the R2 rule; it never lifts init, which the DSL encodes from its value through the state's Finite schema at construction. Move schema and header production out of the compiler: the S/O/F schemas with nominal identity, the Machine header identity, the Actor's name and the enum-case fact registrations are derived at runtime from the Finite instances and scala.deriving.Mirror; input and last-input schemas likewise. Delete EndpointsPhase and SchemaPhase's schema, header, actor, endpoint and evidence injection (retainHeaderIdentity, retainHeaderSchema, retainActorName, retainEndpoints, retainEvidence, declaredFact, retainInputSchema, retainLastInputSchema) and the retention twins. Leave the name-related retain* rewrites (NamesPhase) untouched. Inbox state-field ownership moves with the schema derivation.

**Size:** L

### Verification
- The six samples' headers, schemas, endpoints and evidence match the retained old-reader output except positions (existing comparison harness).
- A Machine without init fails with Scala's abstract-member error; a state type without a Finite instance fails with the ordinary missing-given error at the Machine.
- Focused dsl, realize, reify and export suites pass.

## Acceptance
- [ ] init, end and evidence are Machine members; the plugin identifies them by override; init is never lifted.
- [ ] No retain* schema/header/actor/endpoint/evidence or declaredFact injection remains in the plugin; the DSL derives the same messages at runtime.
- [ ] Samples match retained output except positions; the two error cases behave as specified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
