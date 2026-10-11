---
satisfies: [R1, R2, R6]
---
# fn-140-one-sentence-witness-queries-with.2 Lift witnesses through existing declarations and enforce source refusals

Touches: [model/framework/Claims.scala, model/irgen/Claims.scala, model/irgen/Capabilities.scala, model/irgen/Compositions.scala, model/irgen/Order.scala, model/irgen/Structure.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/witness/**, model/irgen/testdata/lifts/expected/**, tools/umpire/ir/validate_keys.go, tools/umpire/ir/*selector*.go, tools/umpire/ir/admission_test.go, tools/umpire/check/claims.go, tools/umpire/check/*composition*test.go, tools/umpire/check/transition_test.go, tools/umpire/lint/holes.go, tools/umpire/lint/holes_test.go, tools/umpire/export/composed.go, tools/umpire/export/slice.go, tools/umpire/export/quint.go, tools/umpire/export/*test.go]

## Description
**Size:** M
**Files:** `model/framework/Claims.scala`, `model/irgen/{Claims,Syntax,Capabilities,Compositions,Order,Structure}.scala`, witness fixtures and fixture harness.
**Touches:** [model/framework/Claims.scala, model/irgen/Claims.scala, model/irgen/Syntax.scala, model/irgen/Capabilities.scala, model/irgen/Compositions.scala, model/irgen/Order.scala, model/irgen/Structure.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/witness/**, model/irgen/testdata/lifts/expected/**]

Implement the Scala/source half of R1/R2/R6 following .1. The former task's Go admission, checking, lint, export and exact-selector end-to-end proof are retained without relaxation in the new deferred consumer task; this task does not claim them. Checked-in IR/Cases stay untouched.

### Approach
- Fold a witness into the existing Property/Scenario/Query accumulators behind one cohesive helper; preserve source val capture, explicit list names, declaration ownership and modifier ordering, including exploration and live metadata. Reuse capability expansion and normal discovery.
- Resolve exact machine/derived classes and composed classes using the existing canonical composed-key function. The composition fact selector is `.records(_.member -> MemberMachine, fact)`; validate the exact source field-to-machine binding, not merely unique runtime machine membership or fact suffix. Include two same-typed members and two compositions sharing their state type.
- Select a collision-free tagged exact-composed-class encoding in the existing Property.when_action string only after inspecting admitted legacy names and independently deriving the composed class catalog. Use the same exact encoding for a witness and its ordinary core twin. No new IR field/kind, synthetic Action, generic ClassKey change or all-input widening. Preserve legacy machine selector bytes. Persist the exact encoding contract and Scala proof fixtures for the deferred Go task; full consumer proof remains there.
- Refuse repeated exact tail classes, including full input/member/sync identities; missing/empty path or records, unnamed values, too-short steps/actions limits, wrong owner/member/fact, and generated-name collisions. Collision diagnostics retain both authored positions. Seed a selector/input/member substitution and show it changes/fails the exact expected fixture.
- Read capabilities' generated triples when refusing identical default start/path/fact despite differing ends, bounds or expectations. Distinct inputs and member identities are not duplicates.
- Recognize witnesses as Query declarations in source order/placement; preserve invariant triple placement. Compare a witness with its hand-written core twin using a precise source-position map, not broad field stripping.

### Investigation targets
**Required:**
- `model/framework/Claims.scala` and `model/framework/Syntax.scala` — integrated witness types and sugar.
- `model/irgen/Claims.scala:91-178` and `:333-444` — registration, identity, bounds and Query modifiers.
- `model/irgen/Compositions.scala:375` — canonical composed schedules and keys.
- `model/irgen/Capabilities.scala:677-779` — generated triples/duplicate detection.
- `model/irgen/test/Fixtures.test.scala:76-109` — fixture/refusal harness.
- `proto/internal/temporal/server/api/umpire/v1/claim.proto:22-27`, `tools/umpire/ir/validate_keys.go:63-103` and `tools/umpire/check/claims.go:690-709` — read-only schema/admission contract; no Go edits or commands.
**Optional:** `model/irgen/Order.scala`, `model/irgen/Structure.scala`.

### Quick commands
```sh
mise exec -- scala-cli test model/irgen --test-only '*Fixtures*' -- --tests '*witness*'
make lint-model-syntax
```

Run focused exact-selector Scala fixtures in addition to the witness filter. Serialize heavy commands. Review, full gates, Go, production regeneration and live runs remain owner-deferred.
## Acceptance
- [ ] Machine/derived/composition witnesses, explicit names, modifiers and discovery lift to the existing core triple with a precise source-position-mapped twin comparison (R1/R2).
- [ ] Located source refusals cover path/records/naming/collisions/bounds and repeated complete tail identities, with both positions where required (R1/R2).
- [ ] Exact member/fact binding and capability duplicate tests distinguish same fact names on distinct members and different input classes, including seeded wrong-selector detection (R1/R6).
- [ ] Tagged exact-composed encoding is collision-free against admitted legacy names, canonical in core twins, and accompanied by the contract/fixture handoff to the deferred Go consumer task; no Go or full consumer equivalence completion is claimed.
- [ ] Quick commands and focused Scala exact-selector fixtures pass; no schema or managed artifact changes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
