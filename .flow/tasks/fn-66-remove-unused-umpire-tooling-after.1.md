---
satisfies: [R1, R2, R3, R4]
---
# fn-66-remove-unused-umpire-tooling-after.1 Inventory retained consumers and remove unused Umpire tooling

## Description
After fn-64 and fn-62 complete, inventory and remove residual unused tooling across
`tools/umpire`; use the post-cutover tree, not today's legacy package list, as authority.

**Size:** M
**Files:** `tools/umpire/**` and directly owned orphaned inputs/outputs or references selected by inventory
**Touches:** [tools/umpire/**, Makefile, .github/workflows/umpire.yml, tests/umpire*, model/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, .plans/UMPIRE4_*]

### Approach
- Confirm both prerequisite specs are complete and read fn-64.8/.10 migration evidence and fn-62's final public authoring surface.
- Account for every remaining package and command in an ownership ledger at `tools/umpire/CLEANUP_INVENTORY.md`. Name the concrete runtime, Producer, authoring, generator, regression, CLI, or retained downstream consumer for each retained item. Record removal reasons and references for obsolete items.
- Search repository consumers, scripts, build/generation targets, workflow commands, fixtures, docs, and retained Flow specs. Include older-generation and general artifact support; absent Go imports alone do not justify removing a command or public contract.
- Delete proven unused packages/commands and exclusively owned helpers, fixtures, and tests. Extend fn-64's removed Test/Fuzz accounting with preserved/replaced/intentionally-retired decisions and reasons. Resolve ambiguous ownership before deleting the affected item.
- Remove direct obsolete references and regenerate managed output through its owner when necessary. Preserve fn-5 generic promotion, Case Runtime and ordinary authoring, active fn-65 consumers, and concrete retained downstream contracts.
- Verify focused retained consumer suites, then the complete post-cutover model/runtime gates. Compare inherited failures and exact retained artifacts. Preserve comments on retained/moved code; do not add compatibility wrappers, broad refactors, or weaker test selectors.

### Investigation targets
- `tools/umpire/CONTEXT.md` and retained package READMEs: current vocabulary and public ownership.
- `tools/umpire/cmd/**`: command entrypoints and generation consumers.
- `Makefile` and `.github/workflows/umpire.yml`: post-cutover build and regression selectors.
- fn-64.8/.10, fn-62, fn-65, and retained downstream Flow specs: deletion evidence and required contracts.

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/...
make umpire-build-model
make umpire-check-regression
make lint-model
GOLANGCI_LINT_FIX=false make lint-code
```
Inspect the post-cutover command definitions first. Use focused package tests during deletion;
run complete gates once after the final change, including the inherited live integration selector.
## Acceptance
- [ ] Fn-64 and fn-62 are complete before execution; all remaining tooling packages and commands have a concrete consumer or evidence-backed removal decision.
- [ ] Proven unused code and its exclusively owned tests/fixtures and direct references are removed; every newly deleted Test/Fuzz is accounted for.
- [ ] No ambiguous ownership or dangling reference remains; retained authoring, Case Runtime, generators, fn-5 promotion, and explicit downstream consumers preserve behavior and canonical artifacts.
- [ ] Managed outputs and active documentation match the retained surface; existing comments remain intact on retained/moved code.
- [ ] Focused tagged tests and complete post-cutover generation/model/runtime/lint gates pass or show only verified inherited failures, without narrowing selectors to hide regressions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
