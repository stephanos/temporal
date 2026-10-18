---
satisfies: [R1, R2, R6, R16, R18, R22]
---
# fn-115-make-the-scala-model-the-model-and.1 Audit live ownership and record the module and artifact migration map

## Description
Audit live ownership and record the module and artifact migration map. Implements R1, R2, R6, R16, R18, R22 using the reviewed parent contracts.

**Size:** M
**Files:** .plans/UMPIRE_MODULES.md; .plans/umpire-migration-audit.json; .plans/umpire-migration-manifest.json
**Touches:** [.plans/UMPIRE_MODULES.md, .plans/umpire-migration-*.json]

### Approach
- Inventory every package in the six legacy model/tooling trees using production and test import graphs, Make/CI callers and generated-file ownership. Include the new exploration adapter, campaign/replay protocols, umpire-ir-bridge and umpire-gen-cases; distinguish live consumers from consumers being archived.
- Capture the actual current tree and index state, including fn-107's uncommitted/untracked files, before edits. Record protected archive hashes, existing archive-document collisions, reproducible-cache exclusions, exact command baselines, and a copy-before-cleanup strategy.
- Choose final module interfaces/imports, command names, Scala project roots, future example/explorer locations and neutral helper ownership. Resolve public checker types before hiding its implementation. Record the exploration allowance and export's domain dependency rule.
- Define exact source/source-label path substitutions and all derived identity/hash effects; preserve line/column coordinates and stable identity namespaces. Inventory every legacy fixture's replacement Query or missing primitive and its policy/recorded-run consequences. Keep this artifact migration separate from semantic golden comparison.
- Produce a small reviewable map plus machine-readable evidence. Conductor obtains independent map review before restructuring; no staging, commits, worktrees or source moves in this task.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/goir/checking.go:16`
- `model/scalav2/explore/bridge.go:13`
- `tools/umpire/replay/recorded.go:21`
- `tools/canary/casebinding/casebinding.go:26`
- `Makefile:1182`
- `model/scalav2/lifter/Lift.scala:76`

### Quick commands
mise exec -- go list -tags test_dep -deps -test ./model/scalav2/... ./common/testing/testpilot/... ./tools/umpire/... ./tools/canary/...; flowctl validate --spec fn-115 --json

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.

## Acceptance
- [ ] Every audited package has importer/command/generated-output evidence and a destination; no unresolved live importer is hidden in an archive row.
- [ ] The reviewed map fixes public interfaces, allowed imports, command names, archive-copy ownership and the closed path/hash migration; later tasks can execute it without inventing architecture.
- [ ] Current-tree/archive manifests include authorized uncommitted files and protected originals; index state is unchanged.
- [ ] Fixture migration inventory names supported replacements and explicit missing primitives, with pinned canary and recorded-run implications separated from semantic drift.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: