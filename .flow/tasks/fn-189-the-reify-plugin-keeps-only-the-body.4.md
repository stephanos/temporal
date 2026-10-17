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
`15aac500d4`, merged as `b7d4b2bc13`. Authors write `given Family = Family.here` once per feature package, and realize functions that form a Definition ID take `(using f: Family)`. `Family.here` is refused in framework, realization-kit and empty packages; a missing given is the ordinary error. FamilyPhase (421 lines), `realize/Family.scala`, `lazy val family`, `contextualFamily` and the familyFlow carrier/check are deleted. Definition IDs match `export/testdata/constructed/family-original.json` byte for byte (`realize/Family.test.scala`).

Native suites: 639 in the lane, 644 on the merged tree that includes the owner's WIP. The plugin packages and the realization-mode `-Xplugin` sample compile pass, as do lint RuleSuite 31 and Boundary 3. Five ConstructedNamesSuite tests of the deleted family machinery now fail; they are re-anchored or removed in Phase 3. Open: the disabled Temporal kit's ~99 `contextualFamily` calls migrate at the sweep.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 15aac500d4, b7d4b2bc13
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, ReifyBoundarySuite (two-parameter-list fixtures), mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite
- PRs: