---
satisfies: [R10, R13]
---
# fn-123-declare-faults-as-the-environments.8 umpire-faults report, docs and closing gates

## Description
The fault report command (R10), the docs (R13) and the closing gates. `umpire-faults` lists, per IR file, each machine's fault bindings with kind, budget, rows and realization, and each undeclared action performed with a `Fault` command, such as `workerStop`. The docs and the full gate run fold into this task as the spec's finalization.

**Size:** M
**Files:** new `tools/umpire/cmd/umpire-faults/main.go` + `main_test.go`, `.plans/UMPIRE_MODULES.md` (permitted executables :61; task-queue row :37), `tools/umpire/README.md`, `model/README.md` (faults and markers :694-705; how a fault is declared and budgeted and what a crash keeps), `model/SEMANTICS.md` (Assumptions :349-356; Named choices :221), `.plans/UMPIRE4_SPEC.md:202`, `Makefile` (only if the command gets a target)
**Touches:** [tools/umpire/cmd/umpire-faults/**, tools/umpire/README.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, model/README.md, model/SEMANTICS.md, Makefile]
**Order:** After tasks 6 and 7.

### Approach
- Follow `umpire-lint` (`tools/umpire/cmd/umpire-lint/main.go`): IR files as positional arguments, `cli.WriteLine` from `tools/umpire/internal/cli/cli.go`, a usage line, and a `main_test.go`. Read the IR through the reader only, as the module map requires.
- Rows column: `derived`, `authored`, or `overrides` (authored over a derived crash). Budget column: the field or `unbudgeted`. Realization column: performing realization with FaultKind and chosen choices, `model-only` with its reason, or `unperformed`. Reuse lint's unperformed-action logic (`lint/kinds.go:128-155`) for the last case rather than a second implementation.
- An IR file with no fault bindings reports none. A malformed file is reported by the reader as today.
- Test over the checked-in IR (`model/ir/activity-standalone-record.json` and `activity-standalone-race.json`), as the spec's example shows.
- Docs: `model/README.md` explains declaring and budgeting a fault and what a crash keeps, and its marker text follows task 1. `UMPIRE4_SPEC.md:202` ("a fault is an ordinary action of the party that causes it") is revised. `UMPIRE_MODULES.md` lists `umpire-faults` and updates the task-queue row.
- Closing gates once, per MILESTONES.md's verification instructions: the full Go tooling suite with `-tags test_dep -p 2 -timeout 30m`, `make umpire-check-model`, `make umpire-check-cases`, `make lint-model`, `make lint-code-fast`.

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


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
