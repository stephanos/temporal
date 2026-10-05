---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.37 Preserve qualification report cleanup and publication paths

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Repair the five retained unchecked cleanup calls in qualification report storage after reviewed checkpoint 22fe28d1d6b641e9deea5bf9b60a1da773c06381. This independently admitted R18/R19 source owner advances the existing task24/task25 finding backlog; it does not replace task21 or waive original acceptance. Reuse .flow/tmp/after-corpus-independent-owner-astra.md (SHA256 c5961cfbe8b042b802e995217a68b141d1766ab0ba8e1245604ecc7727396fe5) as a bounded recommendation and confirm all primary sources.

**Size:** S
**Touches:** [tools/gomad3/qualification/qualification.go, tools/gomad3/qualification/qualification_test.go, tools/gomad3/qualification/diagnostics_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/**]

### Import-only source admission (2026-10-05)

Revive this task for its remaining qualification-package gci finding from integrated checkpoint 9728756b2e4c81c7f5b322862ca4acd523b85a30. The earlier cleanup source at reviewed checkpoint 771c70c54efb2076b3b09cb18793e38a1902c184 remains unchanged. This amendment permits only import ordering and inter-group whitespace in diagnostics_test.go: retain the same imports and aliases, every comment, and every byte after its import block. Keep qualification.go, qualification_test.go, events.go, tool/config/module/pin files and all other source protected during this correction. This explicit exception supersedes the diagnostic-file protection and instruction to preserve the residual gci only for this correction; historical cleanup receipts retain their original meaning. No new owner or dependency is needed: task21 already depends on this task.

Reproduce the single finding with the actual pinned unfiltered package lint before editing, then run the same command without fix/filter/suppression to verify zero findings. Run all ordinary qualification tests, TestPackageArchitecture, errortype and gofmt on frozen source; preserve diagnostic-off digests and storage controls. Import formatting needs no new source-text test and is outside generator input lists. Per MILESTONES' small-fix rerun rule, historical consumer/boundary results remain historical; this correction's fresh package and architecture checks do not fulfill unproved original full/affected gates. Retain separate admission, commands/results and byte-preservation evidence under task-37/import-order-2026-10-05; do not rewrite the earlier evidence or verifier, which binds the previously protected fixture. Root owns admission/review/Flow/Git, admits one fresh worker and commits independently reviewed source progress before another writer. Original six acceptance bullets remain in force except the explicitly superseded import protection/residual-gci instruction; full/native/formal, original preservation and matched-baseline requirements remain open without proof. Linux stays with fn-128.

### Current import-only source progress (2026-10-05)

The admitted diagnostic import layout is corrected. Same pinned unfiltered package lint reproduced one gci before the edit and reports zero findings in both worker and fresh root runs. Current-source qualification passes 23 top-level tests (66 with subtests), architecture passes one test, and errortype/gofmt/diff checks pass. Independent fresh source review found zero actionable introduced defects. Root verified exact diagnostic body/import identities, protected source/tool/config bindings and immutable earlier artifacts. The [current progress and raw command evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/import-order-2026-10-05/progress.md) is distinct from historical cleanup proof and formal SHIP. Original full/native/formal/dependency/matched-baseline/affected-consumer acceptance remains open; Linux stays with fn-128.

### Original cleanup cause and preservation
WriteQualificationReport currently discards its staging-name Remove and three early Chmod/Write/Sync failure-path Close results. OpenQualificationReport discards its single deferred Close. Preserve their exact lifetimes and operation order. Close the staging descriptor exactly once at the original early/success point. On nil Close retain the original contextual primary error and its concrete identity; only join a nonnil cleanup error after the primary. Keep the existing success Close context verbatim and leave directory Sync/Close precedence unchanged.

Retain staging Remove at function return with a named error result, after all descriptor/directory operations. os.ErrNotExist is the expected consumed staging name after rename and is not an error. Return any other Remove error directly when sole, otherwise join after the existing primary; preserve the existing returned path even after publication. Never remove or retry the published destination. The reader retains its one deferred Close spanning Stat, regular-file/16 MiB validation, bounded read, one-newline trimming and canonical/semantic decode. Nil Close preserves results/errors; nonnil Close clears the report and returns sole cleanup directly or primary-first joined errors. This newly observed cleanup failure is an infrastructure failure, explicitly bounded and disclosed.

Preserve every other production statement, comment, public signature shape, schema, classification, evidence/diagnostic/replay claim, report construction/validation and transaction. diagnostics_test.go and events.go are protected. No generic filesystem layer, global hook, reflection/unsafe fault hack, shared writer, analyzer-only production helper, suppressions, rule changes, assertion weakening, native/platform/policy change or new dependency.

### Verification
Read applicable AGENTS, the complete Gomad README and MILESTONES, original fn109 spec, this task, research note, SPEC QUALIFICATION contracts and existing task24/25 primary receipts. Follow TDD, code-style and verification skills. Before production edits add bounded real-file storage/read characterization and run it against unchanged source. Freeze independent literal full encoded-report SHA256/newline, decoded report/evidence identities, 0600 file/0700 directory modes, published-name shape, no staging leftovers and repeatability/replay controls. Do not compute an expected digest with the candidate as its oracle. Retain every old test body/assertion. Characterize validation-before-filesystem effects, empty root, invalid report/schema, non-directory ancestor, missing path, directory/sparse-over-16MiB file, malformed/noncanonical content, old/future schema, invalid semantic evidence and double newline, with original messages/causes and zero failed reports/empty unpublished paths. Existing successfulEvidence fixed inputs and literal campaign paths supply the vector; random filename stays outside it.

Reproduce actual unfiltered pinned package lint before production edits; this causal RED observes unchecked cleanup, not genuine nonnil cleanup runtime execution. Rerun the same exact tools/config/tags without fix/filter/download after the minimal repair, preserving the separate diagnostic-fixture gci finding and every unexpected diagnostic. Execute focused controls, all ordinary qualification tests, qualification/workload/set/soak consumers, errortype, actual five architecture/purity/edge/public/external-consumer boundaries and gofmt. Pin Go1.27.1 first PATH with GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS=; unset all seed activation variables. Commands/results must bind frozen sources/tools/config/env/cwd, exit/elapsed and source stability. Generator inputs exclude these files; root confirms final touches and whether validation is needed. Do not rerun unchanged whole-root/native environment failures or derive a whole-scope count by subtraction. Original full host, root-fast, formal and Darwin native gates remain required. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.

Real first-Close, simultaneous primary/cleanup, staging Remove and post-rename directory faults remain unproved unless a separately admitted safe real reproducer executes them. Ordinary files, second Close and a mock wrapper prove none of those. Add no source-text test solely for ritual coverage. Preserve proof gaps for independent source review and original acceptance.

### Ownership and completion
Root owns admission, Flow/index/Git/review/commit and document pointers. One shared-checkout source/cache writer; independent research may read only unrelated stable surfaces. No green baseline handoff. No worker lifecycle, review verdict, commit, bridge, push/history/worktree/stash/cache cleanup/download/out-of-Touches change. Return only with terminal commands/delegates, frozen task-unique handover/evidence, final source hashes and truthful gaps. Root reviews bounded source progress independently and commits it with tests/docs/Flow before another source writer. Full original R18/R19/R20, task21/relevant predecessors/shared fn108, matched first-baseline fixed identities, complete/full/formal/affected-consumer/native-default and qualified Darwin native platform gates remain required/open without evidence. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

The 2026-10-05 import-only admission explicitly supersedes diagnostics_test.go import protection and residual-gci preservation only for this new correction. The earlier cleanup criterion and receipts retain their historical meaning; every original completion gate remains required.

- [ ] Original operation bodies/comments and test assertions remain intact outside the five cleanup wrappers/result names; cleanup order, once-only descriptor ownership, nil-cleanup primary identity and published-path returns are independently verified.
- [ ] Added real-file storage/read controls pass on saved unchanged BASE and final source with literal fixed-identity bytes/digests, private modes, residual-name and validation/error/side-effect controls.
- [ ] Actual unfiltered pinned package lint reproduces all five errcheck findings before source edits and removes exactly those on the final source, with separate gci and any other residuals retained; ordinary package/focused/consumer, errortype, boundary and formatting checks pass on frozen inputs.
- [ ] Independent fresh source review finds no actionable introduced defect; real cleanup-fault proof gaps and unchanged full/root/formal/native failures are disclosed, not substituted with ordinary-file evidence.
- [ ] Reviewed source progress, tests, docs and owned Flow evidence are committed separately before another source implementation task.
- [ ] All original R18/R19/R20, task21/predecessor/shared-fn108, matched first-baseline identities, complete/full/formal/affected-consumer/native-default and darwin/arm64 gates pass before completion; unavailable or red source-owned gates keep this task and parent acceptance open. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.
- [ ] The separately admitted import-only correction preserves all import identities/comments and the complete diagnostic fixture body byte-for-byte, reproduces the single gci finding before editing and removes it without new findings under the same pinned unfiltered lint, passes current-source package/architecture/errortype/gofmt checks and independent source review, and commits progress with truthful original acceptance gaps.
## Done summary
SOURCE_PROGRESS_ONLY; original acceptance BLOCKED.

The five cleanup calls are checked without changing original production/test
bodies, cleanup timing, nil-cleanup primary identity or published-path returns.
Frozen [handover](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/handover.md),
[evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/evidence.json) and
[proof freeze](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/freeze.json)
retain the exact commands, source/tool bindings, raw residuals and unproved faults.
Root freshly ran the read-only proof and confirmed saved BASE/final controls 8/8,
ordinary package 23/23, consumers 60/60, five boundaries and two purity/edge
controls. Errortype and formatting pass; actual package lint stays red on the
unchanged diagnostics import-order finding, with five errcheck resolved and none
introduced. Full-host/root-fast/formal/native qualification remains open.

[Fresh bounded source review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/independent-source-review.md)
and [root final scope gate](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/root-verification.json)
are separate from formal SHIP and supply no task completion claim. Requested
reviewer and writer are gpt-6.1-sol/high, same family, in fresh contexts; executed
model identity is not exposed. Root commits this task's reviewed progress before
another source writer. Task21's historical blocked reason is unchanged; parent
remains open with completion review unknown, 2/37 accepted.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: actual lint and original qualification are not green; formal review deferred)
stage: plan-sync - skipped(config: planSync.enabled=false; no task completed)

[Outstanding acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/acceptance-open.md)
retains original R18/R19/R20, task21/predecessor/shared-fn108, matched
first-baseline identities and complete/full/completion/formal/affected-consumer/
native-default/both-native requirements. Genuine first-Close, simultaneous
primary/cleanup, Remove and post-Rename directory fault execution is unproved.
Worker handover lifecycle text is its immutable prior in-progress snapshot, not
a claim about the later root-owned blocked state.

Blocked:
ORIGINAL_QUALIFICATION_OPEN: qualification cleanup and the separately admitted diagnostic import-order correction have reviewed source progress; current unfiltered package lint is clean and scoped package/architecture/errortype/gofmt checks pass. Original full/root-fast/native Darwin/formal/predecessor/shared-fn108/matched-first-baseline/affected-consumer/native-default acceptance is still unproved. Genuine cleanup-fault execution remains unproved. Do not substitute focused developmental linux/arm64 evidence or historical receipts for those gates. Linux execution belongs to fn-128 and does not block this task. See task-37/import-order-2026-10-05/progress.md. Commit verified progress while retaining blocked status; revive only for an admitted source correction or a changed original-gate prerequisite.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
