---
satisfies: [R6]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.26 Restore Runner semantic ownership in CLI callers

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Correct the current R6 source gap after task25 progress commit 984fa118347ebc7b39b7080dd5b9e95e941a00d4. Original task5 requires Runner-owned presence-neutral semantics but the current CLI duplicates base-seed cardinality, coverage/probe validation, trace capacity and choice-coverage dependency. Its earlier summary is historical evidence, not proof of current caller ownership. Task5 is todo with task4 dependency still open; this separate correction does not force-start either, change their original acceptance, or treat their native/predecessor requirements as done.

**Size:** M
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/cli.go, tools/gomad3/cmd/gomad/internal/cli/*_test.go, tools/gomad3/testdata/runnerconsumer/consumer.go, tools/gomad3/runner_consumer_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/runner_external/consumer.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/**]
Root separately owns inventory, parent Flow and MILESTONES metadata; worker does not edit them.

### Required result
Use the existing eight public R6 additions inventoried in go-interface-changes.md before production edits. Keep Runner production signatures/types and semantics unchanged; no new dependencies, public API, globals, test hooks, policy changes or old-overlay wholesale replacement. ParseStrategy already shares NormalizeStrategy. Replace current CLI duplicated rules with ParseSingleBaseSeed in both exploration branches, ValidateCoverage for mode/probes, ValidateChoiceTraceLimit for nonzero semantic range, ValidateChoiceCoverage at the current check point and NormalizeCoverage only for CLI absent guided default. Translate typed errors with errors.As, preserving exact CLI messages, first-error order and checked-writer routing.

CLI retains flag-presence-sensitive behavior: count/seed exclusivity, guide/corpus and explicit coverage admission, every explicit positive exploration bound, disabled/irrelevant choice-byte errors, and enabled choice trace zero rejection even though Runner zero means disabled. Direct Runner guided requests still reject absent coverage. Preserve hidden-route rejection, plan's fixed on-failure policy, argv/environment/tags, installed/private modes, canonical identities and output/write-error status. Inspect direct deterministicio uses before deleting any import.

### Verification
Read full AGENTS.md, Gomad README, MILESTONES, original fn109 spec, tasks4/5 and this task, current inventory and .flow/tmp/r6-caller-repair-scout.md. Apply TDD, writing-good-tests, code-style and verification. Freeze and run current focused CLI characterization, Runner options/canonical characterization and external consumer checks before source changes. Retain literal behaviors and statuses. Add an ownership fitness regression using the existing AST/source architecture idiom to fail on actual duplicated semantic responsibility before production editing: calls must occur in the actual validation branches, not dead references. Use behavioral regressions alongside it, not AST assertions as behavior proof. Name the production regression each test detects. Actual source review must verify thin parsing/presentation consumers, not wrappers retaining duplicated rules.

Run all ordinary CLI package tests after the correction with pinned stock Go1.27.1, -count=1 -tags test_dep, plus focused Runner normalization/typed-error/canonical tests, relevant plan/input-order/default/explicit-zero/irrelevant/writer cases, root TestPackageArchitecture and retained external-consumer compilation. Extend existing external fixtures as needed to compile all eight public seams and error methods without claiming unknown consumers are migrated. Do not weaken canonical goldens or existing tests.

Inspect generator VERSION_INPUTS/BOUNDARY_INPUTS/COMPATIBILITY_INPUTS before editing CLI and record whether validate is required; preserve generated bytes, fixed identities and comparison obligations. If actual validate requires unavailable patched/native tooling, retain its failure and open gate rather than substitute stock proof. Run actual pinned unfiltered affected-package golangci and errortype before/after with unchanged gitroot config/test_dep/fix=false; inherited package findings remain qualification failures, not a filtered green. Retain exact source-bound before/after finding deltas and remaining owners. Do not rerun unchanged broad rootfast/419 nested gate unless newly changed shared inputs require it. If host tests need toolchain access unavailable here, retain focused developmental proof and unmet full/native gates honestly.

Use lean raw logs and terminal receipts with exact command, cwd, env, tool/source SHA, start/end, exit, elapsed and before/after stability. One focused freeze per actual revision; reference prior broader evidence instead of duplicating inventories. No tools/downloads/config/pin changes.

### Ownership and completion
One checkout writer. Root is sole Git/Flow/review/commit owner under conductor-deferred override; no worker lifecycle, formal review or commit. Read-only scouting may run in parallel; no worktree, stash, bridge, push, history rewrite or edits outside Touches. No green baseline handoff. Return only when commands and delegated work are terminal with task-unique handover/evidence and precise remaining requirements; any out-of-scope defect returns SCOPE_EXCEEDED. Root runs fresh independent source review and commits verified progress before the next implementation task. Full R6/R18/R19, task5/predecessors, final task21, full green-tree formal review and the qualified Darwin native gate remain open wherever source-owned evidence is incomplete. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Runner's existing semantic helpers own the actual CLI rules at their original validation points; no duplicated cardinality/probe/capacity/choice-coverage rule remains.
- [ ] Ownership regression fails before production edits and passes after; behavioral characterization preserves first-error messages, explicit-zero/irrelevant flags, guided defaults, plan routing and writer statuses.
- [ ] All ordinary affected CLI tests, focused Runner canonical/error tests, architecture and external compilation have terminal source-bound receipts; public seams/signatures and generator/fixed-identity inputs are preserved or their required gates explicitly remain open.
- [ ] Actual unfiltered affected lint/errortype have terminal before/after receipts with exact introduced versus inherited findings; no filter, suppression, error discard, pin or comparison change manufactures qualification.
- [ ] Independent source review finds no actionable introduced defect and verified owned source progress is committed before another implementation task; original task5/predecessor, R18/R19, task21, Darwin native and formal requirements remain open if unproven. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.

## Done summary
SOURCE_PROGRESS_ONLY. Authoritative Flow status is blocked on qualification.
The fresh independent source review found no actionable introduced defect.
Actual CLI branches now consume Runner's existing semantic helpers while
preserving messages, order, presence/zero behavior and checked writers. Runner
production, public signatures and original task5/predecessor criteria are unchanged.

Current focused CLI passes 34 top-level tests; saved BASE CLI passes the same
33 behavioral tests. Focused Runner passes 10 with the existing Darwin identity
golden skipped; architecture and both outside-Runner compilation checks pass.
Ordinary vet and errortype pass. All 18 worker receipts and current 980-file
freeze are verified. Unfiltered affected CLI lint retains 54 inherited findings,
zero introduced/resolved. Complete CLI/end-to-end and expanded portable-plan
tests retain missing patched-toolchain/unsupported-host failures. Full native,
formal and original R6/R18/R19/task21 acceptance remain open.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: qualification tree red; independent source review supplies no formal SHIP)
stage: plan-sync - skipped(empty: task not done; no completed wave to project)

## Evidence
- Commits: conductor-owned source-progress checkpoint carrying this task record; no task-completion claim.
- Tests: [worker evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/evidence.json), [root verification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/source-checkpoint-verification.json).
- Review: [independent source review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/independent-source-review.md), [checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/independent-source-review-checks.json).
- PRs: none.

## Blocked
[Qualification gaps](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-26/acceptance-open.md) retain original acceptance and required native/formal gates.

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
