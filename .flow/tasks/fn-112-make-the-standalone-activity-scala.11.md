---
satisfies: [R19]
---
# fn-112-make-the-standalone-activity-scala.11 Require an author-computed static combination total on every Query

## Description
Add an explicit author-computed `.total(number)` assertion to each current Scala Query so authors and reviewers see its static combination size.

**Size:** M
**Touches:** [model/umpire/Claims.scala, model/lifter/Claims.scala, model/lifter/test/**, model/lifter/testdata/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, tools/umpire/model/**, tools/umpire/internal/**, tools/umpire/explore/**, tools/umpire/lower/**, model/gate/**, model/temporal/**, model/specimens/**, model/README.md, model/SEMANTICS.md, model/ir/**]

Implement the precise R19 formula: full finite Scenario-machine states times scheduled slots within the depth for pinned paths; states times finite action classes times depth for free paths. Include input combinations and composed classes; count before reachability/deduplication. Named-choice alternatives are results, not another class factor. Require an authored nonnegative integer, without an automatic author helper. Add presence-bearing optional int64 Query.total to IR and regenerate Scala/Go bindings and the linked API jar through the existing gate. Current Scala authors must supply it; historical IR may omit it. Extend `tools/umpire/model/schema_test.go`'s historical descriptor/wire coverage for every current field without changing captured historical bytes. Reuse overflow-safe count math and locate Go mismatch errors at the Query with computed factors. Ensure fingerprints, lowering, search behavior and exploration candidate/promotion identities exclude this assertion. `tools/umpire/explore/explore.go` clones the IR, changes `Scenario.Actions`, hashes the model and calls `NewProducer`: recompute total on the internally derived candidate before validation and hash semantic IR with total excluded, while leaving the authored source Query assertion untouched. Migrate current authored Queries, including Nexus/specimens/fixtures, as needed for the final DSL contract. Document how authors calculate it and distinguish the static surface from actual visited paths.

Shared Query defs (spec R19): `providerQueries(m)` (`System.scala:757`) declares `${m.name}.any.committedStays` once over a free Scenario and is applied to providers with eight and nine bound actions, and `overMatchingQueries(c)` to two compositions, so one literal total inside the def is wrong for at least one instance. A shared def takes its total as an explicit `Int` parameter supplied at each call site (still an authored number; no helper), and `fold` (`model/lifter/Claims.scala:145`), which binds a def's arguments but folds only string literals and declarations, learns to bind an integer-literal argument (`Decl.Number`) that `.total(n)` reads; one lifting fixture (a shared def applied twice with two totals) and one refusal (a non-literal total expression). Task 12's `queueLaws(m)` and the `*.any.*` Queries use this.
## Acceptance
- [ ] Positive typed/lifted fixtures preserve the literal total; missing and duplicate assertions fail at the author source.
- [ ] A shared Query def applied to two machines with different totals lifts each instance's own total from an integer-literal argument bound in the fold; a non-literal total expression is refused at its line.
- [ ] Go tests cover pinned/free, inputs, composition and refinement, disabled/unreachable states, depth shorter/longer than schedule, zero, negative, overflow and mismatch; errors show declared/computed numbers and factors at the Query.
- [ ] Go tests prove schedule-changing exploration candidates remain valid with recomputed internal totals while their digests, promotion IDs, search results and lowered Cases are unchanged by a source total correction. The choice-branch fixture waits for fn-120.1's later schema; the formula already excludes branch results.
- [ ] Historical schema bytes remain readable; current descriptor coverage includes the new Query.total field, and IR plus linked API jar regeneration is recorded.
- [ ] All current authored Scala Queries explicitly supply totals, each shared-def instance its own; historical IR without the optional field remains readable.
- [ ] Semantic fingerprints, tables, Query answers and Case bytes match the original baseline; only Query.total is added to those IR records.
- [ ] Focused lifter/Go tests, generation and documented arithmetic examples pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
