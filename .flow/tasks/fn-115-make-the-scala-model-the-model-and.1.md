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
Recorded the reviewed Umpire module/command map and exhaustive migration audit. The inventory covers 572 package/module rows, 810 source files, 47 Case fixtures, 18 exact source-path and 12 source-label substitutions. All 843 protected originals and the index remain unchanged. No source moves or implementation changes occurred.

Nine legacy execution fixtures have real Scala Query replacements; 22 unsupported/admission-negative fixtures retain explicit exceptions, and two historical companion Case checksums match saved Runs. The map fixes public reader/producer ownership, neutral Testpilot helpers, signature ownership and coherent copy-before-relocation sequencing. Tasks 2–6 incorporate the reviewed test-claim transfers. The synthetic job test fixture has one exact row permutation; production golden ordering stays strict.

Task Quick dependency listing, Flow validation, inventory/preservation verification and diff whitespace checks passed. No code change required model/live tests or broad build/lint here. Codex gpt-6.1-sol high independently returned SHIP on round 3 after five findings were addressed. Review used exact uncommitted artifact hashes because the installed wrapper only scopes committed diffs; same-family independent context, read-only, no cross-family claim. Receipt: .flow/tmp/fn115-1-review/receipt.json. Original handover and all review rounds remain available.

stage: implementation - ran (model: gpt-6-astra); existing agent reused after host thread limit rejected a fresh worker; re-anchored from disk.
stage: verification - ran (focused audit Quick commands and preservation checks).
stage: impl-review - ran (model: gpt-6.1-sol); SHIP round 3.
stage: plan-sync - skipped(config: planSync.enabled != true); required downstream map corrections applied directly through Flow.
stage: tracker-sync - skipped(bridge inactive).
stage: commit - skipped(user explicitly reserves commits and staging).
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang mise exec -- go list -tags test_dep -deps -test ./model/scalav2/... ./common/testing/testpilot/... ./tools/umpire/... ./tools/canary/..., /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl validate --spec fn-115 --json, CC=/usr/bin/clang mise exec -- go run -tags test_dep .flow/tmp/fn115-1/imports.go, python3 .flow/tmp/fn115-1/verify_audit.py, git diff --check
- PRs: