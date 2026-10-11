---
satisfies: [R1, R2, R6]
---
# fn-140-one-sentence-witness-queries-with.2 Lift witnesses through existing declarations and enforce source refusals

## Description
**Size:** M
**Files:** `model/framework/Claims.scala`, `model/irgen/{Claims,Syntax,Capabilities,Compositions,Order,Structure}.scala`, witness fixtures and fixture harness.
**Touches:** [model/framework/Claims.scala, model/irgen/Claims.scala, model/irgen/Lifting.scala, model/irgen/Syntax.scala, model/irgen/Capabilities.scala, model/irgen/Compositions.scala, model/irgen/Order.scala, model/irgen/Structure.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/witness/**, model/irgen/testdata/lifts/expected/**]

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
- [x] Machine/derived/composition witnesses, explicit names, modifiers and discovery lift to the existing core triple with a precise source-position-mapped twin comparison (R1/R2).
- [x] Located source refusals cover path/records/naming/collisions/bounds and repeated complete tail identities, with both positions where required (R1/R2).
- [x] Exact member/fact binding and capability duplicate tests distinguish same fact names on distinct members and different input classes, including seeded wrong-selector detection (R1/R6).
- [x] Tagged exact-composed encoding is collision-free against admitted legacy names, canonical in core twins, and accompanied by the contract/fixture handoff to the deferred Go consumer task; no Go or full consumer equivalence completion is claimed.
- [x] Quick commands and focused Scala exact-selector fixtures pass; no schema or managed artifact changes.
## Done summary
Scala authors can lift machine, derived and composed witnesses into the existing Property, Scenario and Query declarations. The lifter captures names and modifiers, checks the owning composition's exact member-machine binding, preserves authored refusals and rejects capability duplicates by default start, complete pinned path and mandatory recorded fact.

The source contract emits `@exact-composed-class:<canonical class key>` through the existing Property selector. `model/irgen/testdata/witness/SELECTOR.md` records the disjoint-legacy-language proof and the deferred Go consumer requirements. The witness and ordinary core twins compare their complete IR artifacts after enumerated declaration and argument span mappings. Machine selector bytes, ordinary Query discovery and ordinary collision diagnostics retain their existing paths.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; owner deferred reviews)

Verification and scope

- Canonical Quick. `mise exec -- scala-cli test model/irgen --test-only '*Fixtures*' -- --tests '*witness*'`. Ten witness tests pass. Final output lives in `.flow/tmp/scala-dsl/fn140-2-quick-final.log`.
- Focused exact selector. `mise exec -- scala-cli test model/irgen --test-only '*WitnessFixtures*' -- --tests '*exact-composed*'`. One test passes. Output lives in `.flow/tmp/scala-dsl/fn140-2-exact.log`.
- Syntax. `make lint-model-syntax` passes after adding the required core-spelling comments to the two new Syntax hooks. Output lives in `.flow/tmp/scala-dsl/fn140-2-syntax2.log`.
- Formatting. A scoped `scala-cli fmt --check` checks all fourteen changed/new Scala files. Output lives in `.flow/tmp/scala-dsl/fn140-2-format-check.log`. `git diff --check` passes.
- Baseline. Syntax passed before edits. The pre-edit canonical witness command is inconclusive because its initial command handle was not retained and no witness test existed then. It is not counted as green. The task does not offer a pre-edit witness-baseline receipt.
- Red-to-green controls. `fn140-2-focused8.log` exposed the composed capability duplicate missed through differing synthetic default-start constructor positions. `fn140-2-tail-red.log` exposed the missing exact class identity in a refusal. `fn140-2-placement-red.log` exposed a misplaced Witness accepted by the seeded pre-feature Query classifier. Restoring the classifier and implementing the two fixes produced the green focused suite in `fn140-2-focused13.log`.
- Coverage includes exact field-machine ownership with two same-typed members and shared-state compositions; complete own/sync input identities; seeded input/member substitutions; all three generated-name collision namespaces in both root orders; authored argument/declaration lines; empty paths/records, missing records, foreign facts, unnamed witnesses and bounds; default bounds/names; `.total`, live and exploration modifier order; source placement; and capability OR-branch, input, fact, member and actual-start-value negative controls.
- `gate classify` reports FULL for executable Scala changes. The owner's task-local Scala commands define this task's verification scope. Go changes/commands, schema changes, managed IR/Cases regeneration, full gates, reviews and live runs remain deferred. This closure makes no Go consumer or full end-to-end selector claim.

The local framework jar was rebuilt to eliminate a stale pre-witness API cache. No managed artifact changed. The task's obsolete Go Touches line was removed through `task set-spec`; the current Scala Touches includes the authorized Query-subtype discovery change in `model/irgen/Lifting.scala`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bc3653ae49300c82696ababa9c78bb0ebe23b2c7
- Tests: mise exec -- scala-cli test model/irgen --test-only '*Fixtures*' -- --tests '*witness*', mise exec -- scala-cli test model/irgen --test-only '*WitnessFixtures*' -- --tests '*exact-composed*', mise exec -- scala-cli test model/irgen --test-only '*WitnessFixtures*', make lint-model-syntax, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/Claims.scala model/irgen/Claims.scala model/irgen/Syntax.scala model/irgen/Capabilities.scala model/irgen/Compositions.scala model/irgen/Order.scala model/irgen/Lifting.scala model/irgen/test/Fixtures.test.scala model/irgen/testdata/witness, git diff --check
- PRs: