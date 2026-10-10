---
satisfies: [R2, R3]
---
# fn-162-queries-choose-their-assessed.2 Query assessment sets in framework, lifter, IR, lowering and Go assessment

## Description
Give Queries an explicit assessment set, and make the lowering and the Go assessment use exactly that set. Detect duplicate Property declarations. Update the framework, lifter, IR, lowering and assessment, with fixtures and tests; the activity is migrated in fn-162.3.

**Size:** L
**Files:** the Query and realization types in `model/framework`, the lifter (`model/irgen`: Queries, realizations, claims), the IR schema if a field is needed (`proto/internal/temporal/server/api/umpire/v1/*.proto`, then `make protoc`), `tools/umpire/lower`, the model-assessment code behind `umpire-run --model` / `umpire-assess --model`, `model/README.md`, `model/SEMANTICS.md`, fixtures and tests in each
**Touches:** [model/framework/*.scala, model/irgen/*.scala, model/irgen/testdata/**, proto/internal/temporal/server/api/umpire/v1/*.proto, api/umpire/v1/*, tools/umpire/**, model/README.md, model/SEMANTICS.md]

### Approach
- Follow fn-162.1's findings. The design to validate:
  - **Assessment set:** a Query carries the Properties a Case lowered from it is assessed against. The default is the Property it finds or verifies; an explicit form adds more, e.g. `query find completes in heartbeatRetried assessing (completes, heartbeatExhausts)`, or whatever reads best in the DSL. The lifter writes it into the IR.
  - **Realizations:** a realization runs Queries of the machine it realizes (the base machine, e.g. `ActivitySystem`), so a claim-container subject is no longer needed. Keep `Realizes(machine)` for behaviour-changing derivations.
  - **Lowering and assessment:** a Case records its assessment set, and the Go assessment checks exactly that set. A Case with no set is refused, never silently assessed against nothing.
  - **Duplicates:** the lifter reports two Properties of one machine with identical predicates, naming both locations, as a refusal or a lint finding (pick one and say why).
- **Capability-generated Properties** keep their identities and scope. Decide whether they join a Case's assessment set by default (today's behaviour per fn-162.1) and document it.
- **Docs:** SEMANTICS (what a Query's assessment set is, and how it relates to Case verdicts) and the README (how to write claims without subject clones).
- **Fixtures:** a Query with the default set, one with an explicit set, one naming another machine's Property (refused), a duplicate predicate (reported), and a Go test that assesses a Case against exactly its set.

### Prerequisite
- fn-162.1, and a working tree that builds the lifter and passes the Cases check (the in-progress IR `Duration` migration, plus `make protoc`).

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
go test ./tools/umpire/...
```

## Acceptance
- [ ] A Query declares its assessment set (default: the Property it finds or verifies); the IR, lowered Cases and Go assessment carry and use exactly that set; a Case without one is refused.
- [ ] Duplicate predicates on one machine are reported with both locations; a Query naming another machine's Property is refused.
- [ ] README and SEMANTICS document assessment sets; fixtures and Go tests cover each case; `make umpire-check-model`, `make umpire-check-lint` and `go test ./tools/umpire/...` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
