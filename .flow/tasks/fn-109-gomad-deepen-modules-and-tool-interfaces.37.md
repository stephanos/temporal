---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.37 Preserve qualification report cleanup and publication paths

## Description
Repair the five retained unchecked cleanup calls in qualification report storage after reviewed checkpoint 22fe28d1d6b641e9deea5bf9b60a1da773c06381. This independently admitted R18/R19 source owner advances the existing task24/task25 finding backlog; it does not replace task21 or waive original acceptance. Reuse .flow/tmp/after-corpus-independent-owner-astra.md (SHA256 c5961cfbe8b042b802e995217a68b141d1766ab0ba8e1245604ecc7727396fe5) as a bounded recommendation and confirm all primary sources.

**Size:** S
**Touches:** [tools/gomad3/qualification/qualification.go, tools/gomad3/qualification/qualification_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/**]

### Cause and preservation
WriteQualificationReport currently discards its staging-name Remove and three early Chmod/Write/Sync failure-path Close results. OpenQualificationReport discards its single deferred Close. Preserve their exact lifetimes and operation order. Close the staging descriptor exactly once at the original early/success point. On nil Close retain the original contextual primary error and its concrete identity; only join a nonnil cleanup error after the primary. Keep the existing success Close context verbatim and leave directory Sync/Close precedence unchanged.

Retain staging Remove at function return with a named error result, after all descriptor/directory operations. os.ErrNotExist is the expected consumed staging name after rename and is not an error. Return any other Remove error directly when sole, otherwise join after the existing primary; preserve the existing returned path even after publication. Never remove or retry the published destination. The reader retains its one deferred Close spanning Stat, regular-file/16 MiB validation, bounded read, one-newline trimming and canonical/semantic decode. Nil Close preserves results/errors; nonnil Close clears the report and returns sole cleanup directly or primary-first joined errors. This newly observed cleanup failure is an infrastructure failure, explicitly bounded and disclosed.

Preserve every other production statement, comment, public signature shape, schema, classification, evidence/diagnostic/replay claim, report construction/validation and transaction. diagnostics_test.go and events.go are protected. No generic filesystem layer, global hook, reflection/unsafe fault hack, shared writer, analyzer-only production helper, suppressions, rule changes, assertion weakening, native/platform/policy change or new dependency.

### Verification
Read applicable AGENTS, the complete Gomad README and MILESTONES, original fn109 spec, this task, research note, SPEC QUALIFICATION contracts and existing task24/25 primary receipts. Follow TDD, code-style and verification skills. Before production edits add bounded real-file storage/read characterization and run it against unchanged source. Freeze independent literal full encoded-report SHA256/newline, decoded report/evidence identities, 0600 file/0700 directory modes, published-name shape, no staging leftovers and repeatability/replay controls. Do not compute an expected digest with the candidate as its oracle. Retain every old test body/assertion. Characterize validation-before-filesystem effects, empty root, invalid report/schema, non-directory ancestor, missing path, directory/sparse-over-16MiB file, malformed/noncanonical content, old/future schema, invalid semantic evidence and double newline, with original messages/causes and zero failed reports/empty unpublished paths. Existing successfulEvidence fixed inputs and literal campaign paths supply the vector; random filename stays outside it.

Reproduce actual unfiltered pinned package lint before production edits; this causal RED observes unchecked cleanup, not genuine nonnil cleanup runtime execution. Rerun the same exact tools/config/tags without fix/filter/download after the minimal repair, preserving the separate diagnostic-fixture gci finding and every unexpected diagnostic. Execute focused controls, all ordinary qualification tests, qualification/workload/set/soak consumers, errortype, actual five architecture/purity/edge/public/external-consumer boundaries and gofmt. Pin Go1.27.1 first PATH with GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOFLAGS=; unset all seed activation variables. Commands/results must bind frozen sources/tools/config/env/cwd, exit/elapsed and source stability. Generator inputs exclude these files; root confirms final touches and whether validation is needed. Do not rerun unchanged whole-root/native environment failures or derive a whole-scope count by subtraction. Original full host, root-fast, formal and both-native gates remain required.

Real first-Close, simultaneous primary/cleanup, staging Remove and post-rename directory faults remain unproved unless a separately admitted safe real reproducer executes them. Ordinary files, second Close and a mock wrapper prove none of those. Add no source-text test solely for ritual coverage. Preserve proof gaps for independent source review and original acceptance.

### Ownership and completion
Root owns admission, Flow/index/Git/review/commit and document pointers. One shared-checkout source/cache writer; independent research may read only unrelated stable surfaces. No green baseline handoff. No worker lifecycle, review verdict, commit, bridge, push/history/worktree/stash/cache cleanup/download/out-of-Touches change. Return only with terminal commands/delegates, frozen task-unique handover/evidence, final source hashes and truthful gaps. Root reviews bounded source progress independently and commits it with tests/docs/Flow before another source writer. Full original R18/R19/R20, task21/relevant predecessors/shared fn108, matched first-baseline fixed identities, complete/full/formal/affected-consumer/native-default and both qualified native platform gates remain required/open without evidence.

## Acceptance
- [ ] Original operation bodies/comments and test assertions remain intact outside the five cleanup wrappers/result names; cleanup order, once-only descriptor ownership, nil-cleanup primary identity and published-path returns are independently verified.
- [ ] Added real-file storage/read controls pass on saved unchanged BASE and final source with literal fixed-identity bytes/digests, private modes, residual-name and validation/error/side-effect controls.
- [ ] Actual unfiltered pinned package lint reproduces all five errcheck findings before source edits and removes exactly those on the final source, with separate gci and any other residuals retained; ordinary package/focused/consumer, errortype, boundary and formatting checks pass on frozen inputs.
- [ ] Independent fresh source review finds no actionable introduced defect; real cleanup-fault proof gaps and unchanged full/root/formal/native failures are disclosed, not substituted with ordinary-file evidence.
- [ ] Reviewed source progress, tests, docs and owned Flow evidence are committed separately before another source implementation task.
- [ ] All original R18/R19/R20, task21/predecessor/shared-fn108, matched first-baseline identities, complete/full/formal/affected-consumer/native-default and darwin/arm64 plus linux/amd64 gates pass before completion; unavailable or red gates keep this task and parent acceptance open.

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

## Evidence
- Commits: Root creates the separate reviewed source-progress checkpoint; see Git history.
- Tests: python3 .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-37/verify.py; frozen command receipts linked above.
- PRs: None; push/publication not authorized.
