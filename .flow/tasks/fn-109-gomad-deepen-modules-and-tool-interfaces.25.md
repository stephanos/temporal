---
satisfies: [R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.25 Preserve lifecycle fault resolution while repairing exhaustive lint

## Description
Repair the single root fast-lint exhaustive finding retained by task 24 at tools/gomad3sim/controller.go:159:3, after source-progress commits e0ff3c5a13 and ff7da7419b. This is a bounded R19 source owner, not permission for task 21 to implement or to waive the remaining 419 nested findings. Original fn-109 criteria, identities, failure precedence, native gates and workload/default requirements remain unchanged.

**Size:** S
**Touches:** [tools/gomad3sim/controller.go, tools/gomad3sim/controller_resolution_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/**]

### Cause and preservation
The outer switch restricts the nested lifecycle branch to FaultGracefulStop, FaultHarshCrash and FaultRestart. The inner switch repeats that subset of the eight-kind enum; pinned exhaustive reports five missing network kinds that cannot reach it. Preserve behavior by expressing the actual restart-versus-stop/crash distinction without a redundant enum switch. Keep all outer cases, condition precedence, target selection, TargetFrom/candidate handling, operation exclusion, allowed node states, incarnation matching, clone semantics and realization identity unchanged. Do not add a default that hides new cases, unreachable network branches, suppressions, weaker tests, blanket error discards, new public APIs or policy changes.

### Verification
Read AGENTS.md, complete Gomad README, MILESTONES.md, original fn-109 spec, this task and task24's source-bound review/remaining-owner evidence. Apply systematic debugging, TDD, code-style and verification skills. Write literal behavioral characterization first and run it on the unchanged source: all three lifecycle kinds across all declared node states and nonnil operation, exact target/incarnation and error identity, explicit node/Match.Node/incarnation rejection, candidate and prior-target resolution as applicable. Include representative outer network/unknown-kind controls so the restriction is not inferred solely from duplicated switch text. Freeze before/after realization identities or literal expected values independently, not by copying the implementation as the oracle. Preserve existing tests and checks; patched-toolchain-tag tests do not count as run on stock Go.

Retain actual pinned unfiltered package lint RED before source edit; then GREEN on unchanged config/tool/tags without fix, filters or downloads. Run focused new tests and all ordinary gomad3sim package tests with -count=1 -tags test_dep. Run unfiltered actual package golangci/errortype, root lintcode contracts, then real root lint-code-fast with base 951c5516e9e7b3066e7e069adda9565cfd68844c, GOLANGCI_LINT_FIX=false and existing tags. Broader root checks may reveal additional failures; retain terminal receipts and exact next owners without editing outside Touches. The 419 Gomad lint findings are unchanged qualification gaps; do not rerun that unchanged broad gate. Run make -C tools/gomad3 validate only if generator input inspection requires it, documenting the decision. Keep raw logs lean: one source freeze per actual frozen revision and references to it, no repeated full-repository inventories or success-lines copies for every command. Exact command, cwd, environment, tool/source hashes, exit, elapsed and stability must remain auditable.

### Ownership and completion
Root is sole Git/Flow/review/commit owner under explicit conductor-deferred override. One checkout writer, read-only independent scouting may run in parallel. No worktree, stash, bridge, push, history rewrite, external writes or edits outside Touches. No green baseline handoff. Return only when commands and delegates are terminal with task-unique handover and evidence; no formal review/lifecycle/commit by worker. If further fixes require additional source owners, return typed SCOPE_EXCEEDED with exact gate receipts. Root freshly reviews source progress and commits it before the next implementation task. Formal review requires green qualification; original R18/R19, task21 and both native gates remain open without actual proof.

## Acceptance
- [ ] Before/after literal behavior characterization covers lifecycle state/operation/target/incarnation and relevant outer controls, preserving exact successful realizations and failure precedence/identity.
- [ ] The actual pinned linter reproduces the original exhaustive defect before source edits and passes the affected package after the minimal source correction; no rule/default/suppression/baseline/manifest/public API change is used.
- [ ] New and existing ordinary gomad3sim tests, unfiltered affected lint/errortype and helper contracts pass on frozen source with test_dep; actual root fast gate has terminal receipts with any remaining failure explicitly owned.
- [ ] Independent fresh source review finds no actionable introduced defect; original evidence and criteria remain unchanged, and unavailable native or broad qualification is not marked passing.
- [ ] Verified source progress, tests, docs and owned Flow evidence are committed before another implementation task; task21, formal green-tree review, original R18/R19 and both native gates remain open wherever required checks are incomplete.

## Done summary
SOURCE PROGRESS ONLY: lifecycle fault-resolution behavior preserved while the actual exhaustive package finding is repaired. Fresh independent source review and package checks pass; root-fast advances to the unchanged 419 nested findings. See [acceptance and remaining qualification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/acceptance-open.md). Root owns the progress commit before the next implementation task. Original R18/R19, task21, formal review and native gates remain open.

stage: impl-review - skipped(policy: qualification tree red; independent source-progress review is not formal SHIP)
stage: plan-sync - skipped(empty: task not done; no completed wave to project)

## Evidence
- Commits: root-owned source-progress checkpoint in Git history; no worker commit.
- Tests: [terminal worker receipts](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/evidence.json), [independent checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/independent-source-review-checks.json), [root verification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-25/source-checkpoint-verification.json).
- PRs: none; no push authorized.
