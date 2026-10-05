---
satisfies: [R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.24 Restore repository-relative lint exclusion matching

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Restore the intended repository-relative lint path policy after task 23's source-progress checkpoint ce80d2425cf34da103939b5aa23f90bde1c2092f. This is a separate R19 configuration owner, not permission for task 21 to change implementation or task 23 to exceed its routing Touches. Original fn-109 criteria, source baselines, format preservation and native requirements remain unchanged.

**Size:** M
**Touches:** [.github/.golangci.yml, cmd/tools/lintcode/lint_policy_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24/**]

### Evidenced cause and bounded repair

Read task-23/lint-policy-path-observation.md and the actual pinned v2.13.0 source it names. The config under .github defaults to cfg-relative paths, so root-anchored exclusions receive ../tools/... even when run from a nested module. Task 24's pre-edit parsed-YAML/Go-regexp controls pass all existing path expressions; root's byte inspection also shows one literal backslash, not two. This refutes the earlier escaping hypothesis, which remains in the historical task-23 observation. Repair the demonstrated path-base defect, not working regexes or reported source findings.

Use run.relative-path-mode: gitroot to make the existing policy repository-relative for root, Gomad and mixedbrain invocations. Inventory every existing exclusion path/path-except expression and preserve the working expressions unchanged. Retain positive/negative controls and the source-bound correction of the earlier escaping hypothesis. Do not change enabled linters, rule settings, forbid patterns, issue text patterns, tool/Go pins, comparison revisions, fix flags, module manifests or source code. No new exclusion or suppression may be added. Keep existing ^.git unchanged and explicitly record that making paths root-relative activates its preexisting .github reporting limitation. Actual leftover findings stay failures with concrete source ownership.

### Behavioral verification and delivery

Write behavioral regression tests first in the declared test file, using current root dependencies and conventions. Retain a meaningful failure against ce80's config before fixing it. Parse actual YAML and exercise path regex matching with hand-derived positive/negative cases for each repaired expression; derive expected scopes independently, not by computing expectations from the same regex. Test representative root and nested paths and near misses/non-Go files. Do not rely on config line-presence or substring checks.

Also exercise the pinned real golangci binary in isolated scratch Git/module fixtures, using actual copied configuration: prove the same intended path behavior from root and nested cwd, and prove disallowed application findings still report. Tests that need a local pinned binary may use an explicit test input with a clearly reported skip when unavailable; this dispatch must actually run that test with the verified binary, not rely on the skipped default. Avoid downloading tools, widening config, artificial lint bypasses or a generic policy framework.

Use focused root tests with -tags test_dep, run all cmd/tools/lintcode contracts, unfiltered helper golangci and errortype, existing affected Make ownership, and make -C tools/gomad3 validate. Validate actual config with the pinned binary. After freezing config/source, run the real root fast and ordinary nested Gomad/mixedbrain gates using the same base 951c5516e9e7b3066e7e069adda9565cfd68844c, existing tags and GOLANGCI_LINT_FIX=false. Root fail-fast may leave later scopes unreached; run the exact affected tagged integration batch separately if needed. Preserve raw logs, command/cwd/environment/hash/exit/time receipts and old failures unchanged. A green scratch fixture does not establish a green product gate.

### Execution ownership

Read AGENTS.md, Gomad README, MILESTONES, original fn-109 spec and this task. Apply systematic debugging, TDD, code-style and verification skills. Root remains sole Git/Flow/review/commit owner under an explicit conductor-deferred override. Single checkout writer; independently read-only scouting may run in parallel. No worktree, stash, push, history rewrite or changes outside Touches. If scope is insufficient, return exact evidence and typed SCOPE_EXCEEDED rather than editing extra source.

Formal implementation review is only dispatched on a green qualification tree. If product gates remain red, return source progress and typed gaps for fresh independent source review and a progress commit, never formal SHIP or task completion. Stock Linux aarch64 checks cannot close native darwin/arm64 or linux/amd64 gates. Keep source stable during commands, return only after commands are terminal with task-unique handover/evidence. Root commits each verified task progress before another implementation task.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Existing exclusion path/path-except expressions are inventoried with independent behavioral positive/negative cases; meaningful real-tool RED reproduces the inherited path-base defect, working regexes remain unchanged, and source-bound evidence records the refuted escaping hypothesis. No new suppression or non-path policy change is introduced.
- [ ] Actual pinned golangci with the actual copied config proves repository-relative matching from root and nested module cwd; a disallowed ordinary application finding still fails, while only preexisting intended exclusions apply.
- [ ] Enabled linters, settings, forbid/text patterns, pins, baseline comparison, fix flags, manifests and product Go source remain unchanged. Existing ^.git's reporting consequence remains explicit.
- [ ] Focused policy tests, all routing contracts, unfiltered helper lint/vet, affected Make ownership, generated validation and config validation pass on frozen inputs; real root/nested/tagged gates have terminal source-bound receipts and any remaining findings retain failing qualification.
- [ ] Fresh independent source review assesses the bounded repair; no old report, original criterion, workload expectation or native gate is rewritten or declared complete without actual proof.
- [ ] Source progress, tests, docs and Flow evidence are committed before the next implementation task; formal review, original R18/R19 and the Darwin native gate remain open whenever their source-owned required checks have not passed. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.
## Done summary
SOURCE PROGRESS ONLY: repository-relative matching repaired and independently reviewed; qualification remains red. See [acceptance and remaining owners](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24/acceptance-open.md). Root will commit this checkpoint before the next implementation task. Original R18/R19, task 21, formal review and both native gates remain open.

stage: impl-review - skipped(policy: qualification tree red; independent source-progress review is not formal SHIP)
stage: plan-sync - skipped(empty: task not done; no completed wave to project)

## Evidence
- Commits: root-owned source-progress checkpoint in Git history; no worker commit.
- Tests: [terminal worker receipts](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24/evidence.json), [independent checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24/independent-source-review-checks.json), [root verification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-24/source-checkpoint-verification.json).
- PRs: none; no push authorized.

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
