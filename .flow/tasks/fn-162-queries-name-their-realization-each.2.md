---
satisfies: [R3, R6]
---
# fn-162-queries-name-their-realization-each.2 Queries name their realization or are model-only: framework, lifter, IR, lowering, assessment

## Description
Let a Query name the realization that lowers and assesses it, or mark it model-only, and make the lowering and the Go assessment use that binding instead of "the one realization of the scenario's machine". Also detect duplicate Property declarations. This is the mechanism; the activity is migrated in fn-162.3. fn-162.1 traced today's behaviour (spec, Resolved via Codebase): a Case is already assessed against its Query's one Property plus the scenario machine's monitors, so assessment itself doesn't change.

**Size:** L
**Files:** the Query/Scenario and realization types in `model/framework`, the lifter (`model/irgen`: Queries, realizations), the IR schema (`proto/internal/temporal/server/api/umpire/v1/*.proto`, then `make protoc`), `tools/umpire/lower/lower.go` (`Realizable`), `tools/umpire/conformance/plan.go` (`realizationOf`), model lint's use of the same decision, `model/README.md`, `model/SEMANTICS.md`, fixtures and tests in each
**Touches:** [model/framework/*.scala, model/irgen/*.scala, model/irgen/testdata/**, proto/internal/temporal/server/api/umpire/v1/*.proto, api/umpire/v1/*, tools/umpire/**, model/README.md, model/SEMANTICS.md]

### Approach
- **DSL:** a Query (or its scenario) names its realization, or is marked model-only. The form is designed together with fn-140's one-sentence witness Query, by one owner (owner, 2026-10-10). Settle the form with the owner before implementing.
- **Lifter and IR:** the lifter writes the binding into the IR (a new Query field, or one on the Scenario). It refuses a Query naming a realization of another machine, naming both, and reports a find Query with neither binding nor model-only mark, as "no realization" is reported today.
- **Lowering and assessment:** `lower.Realizable` and `conformance.realizationOf` read the binding instead of searching the machine's realizations. One machine may then have several realizations, each named by its Queries. Model lint uses the same decision (`Realizable` is "the one place that is decided").
- **Assessment stays as it is:** a Case is assessed against its Query's Property plus the scenario machine's monitors. No assessment sets.
- **Duplicates:** the lifter reports two Properties of one machine with identical predicates, naming both locations, as a refusal or a lint finding (pick one and say why).
- **Docs:** SEMANTICS (how a Query's realization is chosen) and README (writing Queries and realizations without claim-container copies).
- **Fixtures:** a Query bound to a realization, a model-only Query, two realizations on one machine each bound by its own Queries, a Query naming another machine's realization (refused), and a duplicate predicate (reported). Go tests for `Realizable` and `realizationOf` with bindings.

### Batching
- **Split:** the Scala part (DSL form, lifter, duplicate check) lands in Batch 1d (see MILESTONES.md). The IR schema field, `lower`, `conformance` and model lint land in Batch 5b, with its single regeneration. **Co-design:** one owner designs this with fn-140.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
go test ./tools/umpire/...
```
## Acceptance
- [ ] A Query names its realization or is marked model-only; the IR carries the binding; `lower.Realizable`, `conformance.realizationOf` and model lint use it, so one machine may have several realizations.
- [ ] A Query naming another machine's realization is refused; an unbound find Query is reported; duplicate predicates on one machine are reported with both locations.
- [ ] Assessment is unchanged (Query Property plus scenario-machine monitors); README and SEMANTICS document the binding; fixtures and Go tests cover each case; the gate checks pass at Batch 5b's close.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
