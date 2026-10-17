---
satisfies: [R3]
---
# fn-189-the-reify-plugin-keeps-only-the-body.4 Family as a given value; remove FamilyPhase

Touches: model/framework/dsl (Family), model/framework/realize/Realize.scala, model/framework/realize/Family.scala, model/framework/realize/Conversion.scala, model/framework/reify/plugin/FamilyPhase.scala, model/framework/samples, model/framework/export realize fixtures

## Description
**DSL completion phase (owner, 2026-10-17):** scheduled before the IR/plugin phase because it changes what authors write or can reach; the gate is the native checks plus the plugin compiling the six samples.

Add `Family` to the DSL with `Family.here`, a small macro that yields the enclosing package and refuses, at compile time naming the package, when used inside framework or kit packages. A feature package declares `given Family = Family.here` once. Every realize function that forms a Definition ID takes `(using f: Family)` and reads `f.id`; remove the throwing `lazy val family`, `contextualFamily`, the reserved markers in realize Family, FamilyPhase, the familyFlow annotation and the family-related refusal texts. Update the samples and the realize fixtures to declare the given. Definition IDs must be byte-identical to before: compare the realize fixtures' retained outputs.

**Size:** M

### Verification
- Realize fixtures and the six samples produce the same Definition IDs as the retained outputs.
- A kit function called without a Family in scope fails with the ordinary missing-given error; Family.here in a framework package fails naming the package.
- Focused realize, reify and export suites pass.

## Acceptance
- [ ] FamilyPhase, familyFlow, contextualFamily and the throwing family val are gone.
- [ ] given Family = Family.here per feature package; kit functions take using Family.
- [ ] Definition IDs of the samples and realize fixtures are unchanged; the two error cases behave as specified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
