---
satisfies: [R1, R2, R6]
---
# fn-140-one-sentence-witness-queries-with.2 Lift witnesses through existing declarations and enforce source refusals

Touches: [model/irgen/Claims.scala, model/irgen/Capabilities.scala, model/irgen/Compositions.scala, model/irgen/Order.scala, model/irgen/Structure.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/witness/**, model/irgen/testdata/lifts/expected/**]

## Description
Lift the .1 authoring surface and implement R1/R2/R6 admission using the existing claim accumulators. Keep all witness-origin lifting and refusal fixtures in one task because they share registration and the fixture harness.

**Size:** M
**Files:** `model/irgen/{Claims,Capabilities,Compositions,Order,Structure}.scala`; `model/irgen/test/Fixtures.test.scala`; new witness fixtures and their expected outputs.

### Approach
- Extend the claim fold at `Claims.scala:333` and scenario construction at `:583` to register three ordinary declarations under the captured witness name, with no new IR kind. Keep witness folding in one cohesive helper behind the existing claim registration seam rather than growing the dispatch with duplicated construction. Preserve all modifier orderings and existing `.explore` metadata.
- Resolve machine and derived owners through the existing model references; resolve composition classes through `Compositions.scala:375`. Validate the selected member's fact against this composition, including two compositions sharing one state record.
- Preserve both source positions when registration at `Claims.scala:91` detects a generated Property/Scenario name collision. Compare complete input-bearing class identities for the repeated-tail refusal, including composed keys.
- Reuse capabilities' generated Property/Scenario/Query output at `Capabilities.scala:677-779`. Refuse identical default start, ordered path and recorded fact even when ends, Limits or live expectations differ. Never conflate different inputs or members.
- Teach source order/placement checks that a witness belongs in queries; preserve the existing invariant triple placements. Keep malformed or unsupported input a located LiftError.
- Pin a witness against a hand-written core twin using the existing fixture comparison seam. Compare every emitted declaration after an explicit source-position mapping, never generic field removal.

### Investigation targets
**Required:**
- `model/irgen/Claims.scala:91-178` - registration, identities and bounds.
- `model/irgen/Claims.scala:333-444` - Query modifiers and expectation admission.
- `model/irgen/Compositions.scala:375` - composed schedules.
- `model/irgen/Capabilities.scala:677-779` - capability-generated triples.
- `model/irgen/test/Fixtures.test.scala:76-109` - compile/lift refusal harness.
**Optional:**
- `model/irgen/Order.scala` and `model/irgen/Structure.scala` - declaration classification.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only '*Fixtures*' -- --tests '*witness*'
make lint-model-syntax
```

## Acceptance
- [ ] Machine, derived and composition witnesses, named list items, ends, limits and exploration lift to the exact existing core triple (R1/R2).
- [ ] Located refusals cover empty path, missing records, repeated exact tail class, unnamed value, both collision positions, and each too-short steps/actions override.
- [ ] Capability duplicate tests cover identical path/fact despite modifiers, distinct input classes and composition members, and existing generated capability queries.
- [ ] A source-position-mapped witness/core fixture proves declaration and metadata equivalence; no schema or Go production code changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
