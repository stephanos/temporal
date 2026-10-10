---
satisfies: [R10, R13]
---
# fn-123-declare-faults-as-the-environments.8 umpire-faults report, docs and closing gates

## Description
The fault report command (R10), the docs (R13) and the closing gates. `umpire-faults` lists, per IR file, each machine's fault bindings with kind, budget, rows and realization, and each undeclared action performed with a `Fault` command, such as `workerStop`. The docs and the shared close fold into this task. It owns the selected fn-140/fn-149/fn-123 batch's single production regeneration, complete full gates, independent review and required live/replay run; it closes all three specs only after their unchanged acceptance obligations are evidenced.

**Size:** M
**Files:** new `tools/umpire/cmd/umpire-faults/main.go` + `main_test.go`, `.plans/UMPIRE_MODULES.md` (permitted executables :61; task-queue row :37), `tools/umpire/README.md`, `model/README.md` (faults and markers :694-705; how a fault is declared and budgeted and what a crash keeps), `model/SEMANTICS.md` (Assumptions :349-356; Named choices :221), `.plans/UMPIRE4_SPEC.md:202`, `Makefile` (only if the command gets a target)
**Touches:** [tools/umpire/cmd/umpire-faults/**, tools/umpire/README.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, model/README.md, model/SEMANTICS.md, Makefile]
**Order:** After tasks 6 and 7, with fn-140.6's independent witness seal and fn-149.4/.5's complete final source/docs/layout grouping seal committed in the integrated local branch before fault changes. Those earlier tasks/specs remain pending shared-close evidence, not completion dependencies.

### Approach
- Follow `umpire-lint` (`tools/umpire/cmd/umpire-lint/main.go`): IR files as positional arguments, `cli.WriteLine` from `tools/umpire/internal/cli/cli.go`, a usage line, and a `main_test.go`. Read the IR through the reader only, as the module map requires.
- Rows column: `derived`, `authored`, or `overrides` (authored over a derived crash). Budget column: the field or `unbudgeted`. Realization column: performing realization with FaultKind and chosen choices, `model-only` with its reason, or `unperformed`. Reuse lint's unperformed-action logic (`lint/kinds.go:128-155`) for the last case rather than a second implementation.
- An IR file with no fault bindings reports none. A malformed file is reported by the reader as today.
- Test over the checked-in IR (`model/ir/activity-standalone-record.json` and `activity-standalone-race.json`), as the spec's example shows.
- Docs: `model/README.md` explains declaring and budgeting a fault and what a crash keeps, and its marker text follows task 1. `UMPIRE4_SPEC.md:202` ("a fault is an ordinary action of the party that causes it") is revised. `UMPIRE_MODULES.md` lists `umpire-faults` and updates the task-queue row.
- Preserve both frozen meaning-preserving seals and their explicit composed identity/provenance mappings, including the actual fn-155 baseline map. Classify the final diff as the separately authorized witness, grouping and fault deltas; unexplained behavior, rows, predicates, bounds, assumptions, Query/progress receipts, expected/live-replay assessments, identities or Case-byte changes stop the close. Publish production managed IR/Cases, lift goldens and changed functional/canary pins exactly once here; historical recorded Run companions and expected assessments remain untouched.
- Own the single production `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`, then the complete model, Case/fixture, functional/canary, lint and Go checks for all three specs: `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`, `make umpire-check-cases umpire-check-fixtures canary-check-case`, `make lint-model`, `make lint-code-fast`, and the full Go tooling suite with `-tags test_dep -p 2 -timeout 30m` and JSON terminal events plus measured wall time. Include task .7's required `make umpire-check-backends` agreement check and explicit unsupported receipts. Use actual `fcntl`/`flock` on `/tmp/umpire-heavy-gates.lock`; reuse applicable passing evidence only for matching inputs, fixture/assertion populations and environment, repeating invalidated checks only.
- Run the required independent review and relevant live/replay checks once at this boundary with unchanged expectations. Link all acceptance evidence back to fn-140.6 and fn-149.5 before their tasks/specs close; neither earlier source seal is passing full-gate evidence.
- Retain the full native Model/Go/Case generator OOM/ENOSPC RED as fn-157's resource obligation, Quint resource RED as fn-154's, and inherited completion, fatal-failure and pause/resume strict failures as Batch 5 obligations. Preserve the exact original evidence/owners and assertions separately from migration regressions; none receives green credit. More than one hour cumulatively stuck across validation attempts defers that validation unless every other available work path is blocked. Record commands, elapsed attempts, evidence and revisit conditions; deferral is not a pass and never closes an unmet acceptance obligation.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-lint/main.go`, `main_test.go` - command conventions
- `.plans/UMPIRE_MODULES.md:37, 61, 315-330` - module map and Make targets
- `tools/umpire/lint/kinds.go:95-155` - performable actions and unperformed actions

**Optional** (reference as needed):
- `model/README.md:680-770` - faults, markers and the `FaultyLamp` derivation example
- `.plans/UMPIRE4_SPEC.md:195-210` - the fault sentence to revise

## Acceptance
- [ ] `umpire-faults` prints, per IR file, every fault binding with kind, budget, rows and realization, and every undeclared action performed with a `Fault` command. A test runs it over the checked-in IR.
- [ ] An IR file with no fault bindings reports none; a malformed file fails through the reader, with a test.
- [ ] `UMPIRE_MODULES.md` lists the command; `model/README.md`, `SEMANTICS.md` and `UMPIRE4_SPEC.md` describe declared faults, budgets and what a crash keeps.
- [ ] The full Go tooling suite, `make umpire-check-model`, `make umpire-check-cases`, `make lint-model` and `make lint-code-fast` pass, with commands and log paths in the done summary.
- [ ] All three specs' single production regeneration and complete Case/fixture/functional/canary, backend-agreement, independent-review and required live/replay evidence are linked to the preserved witness/grouping seals and composed mappings before fn-140.6, fn-149.5 or any Batch 2 spec closes. Inherited RED and deferred validations remain explicitly unresolved, never passing evidence.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
