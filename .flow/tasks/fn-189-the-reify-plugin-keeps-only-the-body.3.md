---
satisfies: [R7]
---
# fn-189-the-reify-plugin-keeps-only-the-body.3 Sugar as inline defs; remove the plugin's expansion engine

Touches: model/framework/dsl/Syntax.scala, model/framework/reify/body/Expansion.scala, model/framework/reify/body/BodyContext.scala

## Description
**DSL completion phase (owner, 2026-10-17):** scheduled before the IR/plugin phase because it changes what authors write or can reach; the gate is the native checks plus the plugin compiling the six samples.

Make enter, stay, reject and the two `in` extensions `inline def`; `disabled` becomes `inline def disabled` (an inline val cannot hold Nil). Remove the sugar annotation and the plugin's expansion: the substitution, arity and cycle checks in Expansion, expansionAllowed and the sugar branch of sourceTree. The lifter meets the core form inside an Inlined node; take the sugar's name for diagnostics from the Inlined call so an error can still say which sugar failed, at the call site. Compile the Facets fixture first: the `in` overload with a using TypeTest must still resolve its given at the call site once inline.

**Size:** M

### Verification
- The six samples and the Facets fixture compile and lift to the same IR as before except non-root positions.
- A sugar body the lifter cannot lift reports at the call site (existing fixture).
- Focused dsl, reify and export suites pass.

## Acceptance
- [ ] The five sugar defs are inline and no @sugar annotation remains.
- [ ] Expansion's substitution, arity and cycle checks and the jar sugar lookup are gone.
- [ ] Samples and the Facets fixture lift to the same IR as before except non-root positions; diagnostics name the sugar.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
