---
satisfies: [R1, R2, R6]
---
# fn-140-one-sentence-witness-queries-with.2 Lift witnesses through existing declarations and enforce source refusals

Touches: [model/framework/Claims.scala, model/irgen/Claims.scala, model/irgen/Capabilities.scala, model/irgen/Compositions.scala, model/irgen/Order.scala, model/irgen/Structure.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/witness/**, model/irgen/testdata/lifts/expected/**, tools/umpire/ir/validate_keys.go, tools/umpire/ir/*selector*.go, tools/umpire/ir/admission_test.go, tools/umpire/check/claims.go, tools/umpire/check/*composition*test.go, tools/umpire/check/transition_test.go, tools/umpire/lint/holes.go, tools/umpire/lint/holes_test.go, tools/umpire/export/composed.go, tools/umpire/export/slice.go, tools/umpire/export/quint.go, tools/umpire/export/*test.go]

## Description
Lift the .1 authoring surface and implement R1/R2/R6 admission using the existing claim accumulators. Keep all witness-origin lifting and refusal fixtures in one task because they share registration and the fixture harness.

**Size:** M
**Files:** `model/irgen/{Claims,Capabilities,Compositions,Order,Structure}.scala`; `model/irgen/test/Fixtures.test.scala`; new witness fixtures and their expected outputs; typed composed `when` in `model/framework/Claims.scala`; exact selector reading/admission at `tools/umpire/ir`, checking at `check/claims.go`, coverage/holes at `lint/holes.go`, and existing composed/slice/Quint export consumers with their focused tests.

### Approach
- Extend the claim fold at `Claims.scala:333` and scenario construction at `:583` to register three ordinary declarations under the captured witness name, with no new IR kind. Keep witness folding in one cohesive helper behind the existing claim registration seam rather than growing the dispatch with duplicated construction. Preserve all modifier orderings and existing `.explore` metadata.
- Resolve machine and derived owners through the existing model references; resolve composition classes through `Compositions.scala:375`. Validate the selected member's fact against this composition, including two compositions sharing one state record.
- First prove the exact composed-tail encoding. Extend typed Property `when` to accept a composed class; use the existing canonical `Compositions.composedKey(..., classes=true)` for both pinned Scenario and same-step Property. Encode exact composed selectors in the existing Property.when_action string using a collision-free tagged convention, without new fields/kinds, synthetic Action declarations or changing general interpreter ClassKey. The precise tag is chosen only after proving it cannot collide with admitted legacy action names; otherwise stop before migration. Generated witnesses and hand-authored ordinary core twins share that encoding. Untagged action selectors and existing machine when_class selectors retain their exact bytes and meaning.
- Use one source-aware selector-decoding contract across composition admission, checking, hole/coverage lint and composed/slice/Quint export. The tagged form is legal only on a composition Property. Admission requires exact membership in its independently derived composed class catalog, including member/sync ownership, inputs and binding identity; malformed, blank, wrong-owner, wrong-input and absent keys refuse at the Property source. Runtime compares the complete admitted class key by equality, never an action-only prefix. Export the exact input guards faithfully or produce a source-located UnsupportedError without widening/exporting around it. Preserve every legacy admission diagnostic and obligation; no equality/lint exemptions.
- Before expanding migration, lift a witness/core twin whose input-bearing tail follows a different input of the same composed action producing indistinguishable state/facts. Cover own-member and renamed-sync keys, two same-typed members, wrong-owner/input refusals and repeated exact-tail rejection. A seeded all-input selector substitution must fail. Exercise full Go admission and checking, legacy byte/semantics pins, hole/coverage accounting and faithful export/located refusal; lift-only or state/fact-only equivalence is insufficient.
- Preserve both source positions when registration at `Claims.scala:91` detects a generated Property/Scenario name collision. Compare complete input-bearing class identities for the repeated-tail refusal, including composed keys.
- Reuse capabilities' generated Property/Scenario/Query output at `Capabilities.scala:677-779`. Refuse identical default start, ordered path and recorded fact even when ends, Limits or live expectations differ. Never conflate different inputs or members.
- Teach source order/placement checks that a witness belongs in queries; preserve the existing invariant triple placements. Keep malformed or unsupported input a located LiftError.
- Pin a witness against a hand-written core twin using the existing fixture comparison seam. Compare every emitted declaration after an explicit source-position mapping, never generic field removal.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/claim.proto:22-27` and `tools/umpire/ir/validate_keys.go:63-103` - existing schema and composed selector admission, read only for schema shape.
- `tools/umpire/check/claims.go:690-709`, `tools/umpire/lint/holes.go:260-295`, `tools/umpire/export/{composed,slice,quint}.go` - all selector consumers.
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
- [ ] The early exact-composed proof covers renamed sync/member ownership, input-bearing tail after a different input with indistinguishable state/facts, wrong-owner/input refusals and seeded all-input-selection rejection across lifting, Go admission/checking, lint and export. Collision-free encoding and ordinary core equivalence are proved before migration.
- [ ] A source-position-mapped witness/core fixture proves declaration and metadata equivalence; no schema changes. Go production changes are limited to the explicitly authorized common exact-composed selector seams, with exact legacy-byte/semantics pins.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
