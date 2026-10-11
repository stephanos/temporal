---
satisfies: [R1, R3]
---
# fn-150-bounded-liveness-across-composed.3 Expose composition progress and fairness in typed authoring

## Description
Expose composition progress and fairness in typed authoring. Advances R1, R3 of the parent spec.

**Size:** M
**Files:** `model/umpire/Assume.scala`, `model/umpire/Compose.scala`, `model/umpire/IrFile.scala`, `model/irgen/Claims.scala`, `model/irgen/Declarations.scala`, `model/irgen/Lifting.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/testdata/composedProgress/**`
**Touches:** [model/umpire/Assume.scala, model/umpire/Compose.scala, model/umpire/IrFile.scala, model/irgen/Claims.scala, model/irgen/Declarations.scala, model/irgen/Lifting.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/composedProgress/**]

### Approach
- Extend leadsTo to Composition with its composed state type, preserving the Machine form, explicit positive bound and assumption arguments.
- Reuse own/synced typed selectors for fair references and lower/export them to task 2's structured representation. Preserve action-wide versus exact class intent and reject wrong owners rather than guessing.
- Align declaration construction and lifting/export discovery for Progress; consume fn-141's exporter if present and fn-149's groups if landed. Do not make grouping a prerequisite.
- Add focused typing, lift and refusal fixtures for predicates reading two members, incompatible state types, missing/invalid bounds, repeated member machines, parameterized selectors and unknown/ambiguous syncs.

### Investigation targets
**Required:**
- `model/umpire/Assume.scala:47` - Machine leadsTo extension.
- `model/umpire/Compose.scala:88` - synced selector; own at line 93.
- `model/irgen/Claims.scala:372` - progress lifting.
- `model/irgen/Declarations.scala` - assumption lifting.
- `model/umpire/IrFile.scala:60` - runtime discovery.
- `model/irgen/test/Fixtures.test.scala` - golden and compile refusal harness.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only Fixtures
```

## Acceptance
- [ ] A typed two-member progress fixture exports owner, state predicates, bound and exact fairness scope through the existing pipeline.
- [ ] Compiler/lifter refusals cover incompatible predicates, missing/invalid bounds and malformed selectors without losing source attribution.
- [ ] Existing Machine authoring fixtures stay valid; grouped or ungrouped placement follows the currently landed fn-149 contract.

## Done summary
`c.leadsTo(name)(from, to, within, under*)` on `Composition[S]` (framework/Assume.scala) with
predicates typed to the composed state; `assume.fair(first: Composed, more*)` makes composed steps
fair through the existing `synced`/`own` selectors. Lifting (irgen/Compositions.scala
`composedFairRef`, sharing one `selected` resolution with `composedKey`; Declarations.scala
`assumptionOf`/`machineAssumption`; Claims.scala progress owner check) exports Progress on the
composition and `fairRefs` preserving composition, member field, sync vs member, action-wide vs exact
class. Refusals with source lines: fairness on another composition, on a machine's claim, assumed by a
machine, mixed with machine actions, spread over two compositions, a sync class named from the side
that cannot supply inputs, an own action that steps only with its pair, an action no sync pairs, a
bound of 0; compile refusals: a member-state predicate and a missing bound. fn-149 grouping has not
landed: claims stay ungrouped (Progress in `properties`, as the layout lint requires).

## Evidence
- Commits: f7fcb54a67
- Tests: `scala-cli test model/irgen --test-only umpire.irgen.Fixtures` (UMPIRE_LIFTER_UPDATE run: all pass but the crossed positions, fixed after; verification run: 111 PASS, 0 FAIL, `.flow/tmp/fn150/fixtures.log`); framework tests 60 PASS (`.flow/tmp/fn150/framework-tests.log`); Go TestLiftedModelsAreAdmitted/composedProgress and TestLiftedComposedProgressResolvesScopedFairnessAndCountsUnrelatedSteps PASS
- PRs:
