---
satisfies: [R8]
---
# fn-162-queries-name-their-realization-each.4 Go consumers read the realization binding (held Go/schema half of fn-162.2)

## Description
**Size:** M
**Files:** generated `api/umpire/v1/realization.pb.go` (from the committed `realization.proto` field 21, `Realization.queries`), `tools/umpire/lower/lower.go` (`Realizable`), `tools/umpire/conformance/plan.go` (`realizationOf`), umpire-lint's use of `Realizable` (`tools/umpire/cmd/umpire-lint/main.go`), their tests and fixtures.
**Touches:** [api/umpire/v1/realization.pb.go, tools/umpire/lower/**, tools/umpire/conformance/**, tools/umpire/cmd/umpire-lint/**, tools/umpire/ir/**]

Owner-held Go/schema half split from fn-162.2 (precedent fn-140.7/.8/.9 split from .2/.3/.4). fn-162.2 delivered the Scala half: `Realizes(machine, runs = Seq(...))` / `DerivesFrom(base, machine, runs = ...)`, the lifter writing `Realization.queries` (proto field 21, already in `realization.proto`), its refusals (another machine's Query; a Query two realizations name) and the duplicate-Property refusal. The Go consumers still pick "the one realization of the Scenario's machine" and cannot read the binding: the Go bindings are not regenerated (`make protoc` is owner-held). Do not admit while MILESTONES holds Go/schema/regeneration work.

### Approach
- Regenerate the Go API from the committed proto (`make protoc`); no new schema field beyond field 21.
- `lower.Realizable`: a find Query is lowered through the realization whose `queries` names it; one no realization names is `NoRealization` (model-only, lint keeps reporting it). Keep it the one place that is decided, for the manifest and for model lint.
- `conformance.realizationOf`: the realization naming the Query, not the unique realization of the machine. One machine may have several realizations.
- Assessment unchanged: a Case is assessed against its Query's Property plus the Scenario machine's monitors.
- Go tests for `Realizable` and `realizationOf` with bindings: one realization naming its Queries, an unnamed find (model-only), and two realizations on one machine each bound by its own Queries.

### Quick commands (only once the owner lifts the hold)
```sh
make protoc
go test ./tools/umpire/lower/... ./tools/umpire/conformance/... ./tools/umpire/cmd/umpire-lint/...
make umpire-check-lint
```

## Acceptance
- [ ] The Go API carries `Realization.queries`; `lower.Realizable`, `conformance.realizationOf` and model lint read the binding, so one machine may have several realizations; an unnamed find Query is model-only (`NoRealization`).
- [ ] Assessment is unchanged (Query Property plus Scenario-machine monitors); Go tests cover a named binding, an unnamed find, two realizations on one machine.
- [ ] Admitted only after the owner reopens the held Go/schema lane; the gate checks pass at that batch's close.


## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
