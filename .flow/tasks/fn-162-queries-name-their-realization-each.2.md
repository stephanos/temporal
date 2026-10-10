---
satisfies: [R3, R8]
---
# fn-162-queries-name-their-realization-each.2 Realizations name the Queries they run: framework, lifter, IR, lowering, assessment

## Description
Let a realization name the Queries it runs (an unnamed find Query is model-only), and make the lowering and the Go assessment use that binding instead of "the one realization of the scenario's machine". Also detect duplicate Property declarations. This is the mechanism; the activity is migrated in fn-162.3. fn-162.1 traced today's behaviour (spec, Resolved via Codebase): a Case is already assessed against its Query's one Property plus the scenario machine's monitors, so assessment itself doesn't change.

**Size:** L
**Files:** the Query/Scenario and realization types in `model/framework`, the lifter (`model/irgen`: Queries, realizations), the IR schema (`proto/internal/temporal/server/api/umpire/v1/*.proto`, then `make protoc`), `tools/umpire/lower/lower.go` (`Realizable`), `tools/umpire/conformance/plan.go` (`realizationOf`), model lint's use of the same decision, `model/README.md`, `model/SEMANTICS.md`, fixtures and tests in each
**Touches:** [model/framework/*.scala, model/irgen/*.scala, model/irgen/testdata/**, proto/internal/temporal/server/api/umpire/v1/*.proto, api/umpire/v1/*, tools/umpire/**, model/README.md, model/SEMANTICS.md]

### Approach
- **DSL (owner, 2026-10-10):** the realization names its Queries, e.g. `Realizes(ActivitySystem, runs = Seq(Heartbeat.queries.heartbeatThenCompletes, …), observations = …)`. The Query syntax doesn't change, which respects the one-way layer dependency (realization → system → product). A find Query no realization names is model-only and isn't lowered (today's "no-realization" standing; lint keeps reporting it).
- **Lifter and IR:** the lifter writes each realization's Queries into the IR. It refuses a realization naming another machine's Query, and a Query named by two realizations, naming both.
- **Lowering and assessment:** `lower.Realizable` and `conformance.realizationOf` read the binding instead of searching the machine's realizations. One machine may then have several realizations, each named by its Queries. Model lint uses the same decision (`Realizable` is "the one place that is decided").
- **Assessment stays as it is:** a Case is assessed against its Query's Property plus the scenario machine's monitors. No assessment sets.
- **Duplicates:** the lifter reports two Properties of one machine with identical predicates, naming both locations, as a refusal or a lint finding (pick one and say why).
- **Docs:** SEMANTICS (how a Query's realization is chosen) and README (writing Queries and realizations without claim-container copies).
- **Fixtures:** a realization naming its Queries, an unnamed (model-only) find Query, two realizations on one machine each bound by its own Queries, a realization naming another machine's Query (refused), a Query named by two realizations (refused), and a duplicate predicate (reported). Go tests for `Realizable` and `realizationOf` with bindings.

### Batching
- **Split:** the Scala part (DSL form, lifter, duplicate check) lands in Batch 1d (see MILESTONES.md). The IR schema field, `lower`, `conformance` and model lint land in Batch 5b, with its single regeneration. **Co-design:** fn-140's Query form carries no realization information; one owner keeps them consistent.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
go test ./tools/umpire/...
```
## Acceptance
- [ ] A realization names the Queries it runs (unnamed find Queries are model-only); the IR carries the binding; `lower.Realizable`, `conformance.realizationOf` and model lint use it, so one machine may have several realizations.
- [ ] A realization naming another machine's Query, or a Query named by two realizations, is refused; an unnamed find Query is reported as model-only; duplicate predicates on one machine are reported with both locations.
- [ ] Assessment is unchanged (Query Property plus scenario-machine monitors); README and SEMANTICS document the binding; fixtures and Go tests cover each case; the gate checks pass at Batch 5b's close.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
