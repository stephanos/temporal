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
Every Query now carries an author-computed static combination total, and Go checks it. Commits 46e2e8d041 and 04bc6c4723.

**What changed**
- IR: `Query.total = 10` is a `google.protobuf.Int64Value` (presence-bearing). Go bindings are regenerated with `make protoc` and a linux goimports, because `.bin/goimports-v0.49.0` is a host binary. The ScalaPB IR jar and the linked API jar are regenerated through the gate (`gen-jars.log`). ScalaPB reads the field as `Option[Long]`.
- DSL: `infix def total(n: Long): Query` is core and lives in `model/umpire/Claims.scala`. It can be written `… limits four total 48` or `(…).total(48)`.
- Lifter:
  - Adds `Decl.Number` for Int/Long literals, so a shared def's `total: Int` parameter is bound from the literal at each call site. Int widening (`int2long`, `.toLong`) is unwrapped.
  - Refuses at the author's line: a missing total (per root, after a successful lift), a duplicate, a negative total, a non-literal total, and a non-literal Int argument to a shared def.
  - New fixture `lifts/Totals.scala`, plus 6 refusals in Rejects.scala lines 757-782.
- Go (`tools/umpire/model/totals.go`):
  - `QueryTotal`, `RequireTotals`, `WithTotals` and `WithoutTotals`. `Validate` refuses a negative, overflowing or mismatched total at the Query, naming the declared count, the computed count and its factors. An absent total is still admitted, so historical IR stays readable.
  - Counting reuses the overflow-safe `count` math through ceiling-free `boundClasses` and `composedClassCount`.
  - `umpire-gen-cases` requires a total on every Query in model/ir.
- Exploration: each candidate recounts its own total with `WithTotals`. The digest and the proposal recipe hash the IR without totals, and so does the conformance Model identity (fixed in review round 1).
- Goldens: `Query.total` is an inert field in `original.json` and in the fn-115 migration projection (`config.json`). schema_test now treats "added since the capture" as a closed list: `Query.total` and the wrappers dependency. The captured bytes are unchanged; fn-120.1 extends the same list.
- Migration:
  - All 264 production Queries (96 source sites) and all fixture Queries carry totals.
  - The shared defs `providerQueries(m, anyTotal)` and `overMatchingQueries(c, anyTotal)` take their total at each call site: 2880/2880/2880/3240 and 233280/233280/246240. Every other shared-def site agrees across instances and takes a literal.
  - The largest total is 887040.
- Docs: a new "Query totals" section in model/SEMANTICS.md and an Admission bullet. model/README.md gains "Counting a Query's total" with the arithmetic: syncCompletion 192 × 2 = 384, and a free example 192 × 23 × 3 = 13248, both verified with Go.

**Decisions**
- Used a wrapper instead of proto3 `optional`, because the pinned protoc-gen-go-helpers refuses `optional`. The ProtoJSON spelling is the same.
- The Go check for missing totals lives in `umpire-gen-cases` and is not part of `Validate`, because historical IR and Go-synthesized Queries (export/checked, golden job) have no total.
- Tests that derive models by editing schedules or limits now recount them with `WithTotals`, mainly in the shared `mutated` helper.
- The choice-branch fixture waits for fn-120.1's schema.

**Review**
- Reviewer: claude-opus-5-5 at high, via `--spec claude:claude-opus-5-5:high`. The writer and the reviewer are the same model family (Opus).
- Round 1: NEEDS_WORK. One P2, the conformance identity, and one P3, both fixed.
- Round 2: SHIP.

**Deferred P3/FYI:** a candidate whose total cannot be counted is rejected with `WithTotals`'s message rather than `NewProducer`'s. This is unreachable with the current IR.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 46e2e8d041, 04bc6c4723
- Tests: make protoc (exit 0), make model/gen/ir-scalapb.jar model/gen/api-scalapb.jar (exit 0), make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), scala-cli test model/lifter (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 248 s, incl. OriginalBaseline and MigrationGoldens), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test conformance+lower+umpire-gen-cases after review fixes (exit 0)
- PRs: