---
satisfies: [R3, R13, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.35 Preserve corpus reader cleanup and publication failures

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Repair the corpus reader's two ignored production Close returns and four original corpus fixture Close returns while preserving all existing helper bodies, error precedence, resource lifetimes and corpus transactions. Task34 reviewed source progress is committed at a80ad9b9d1a4195c4aeb2fe135557f71e6e6552a. Original predecessor acceptance remains open.

**Touches:** tools/gomad3/runner/internal/corpus/corpus.go, tools/gomad3/runner/internal/corpus/corpus_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-35/**

Follow the source-bound corpus recommendation in .flow/tmp/next-lint-owner-astra-research.md (SHA256 d630038390947c4aac95c322f905e712686c3f97179631351d67c9d1f7535ebd). In readSnapshot and validateEntry, use named result/error returns with fresh names that do not change any original local statement. At each existing defer, attempt the existing Close once. On nil Close preserve exact result/error identity. Only on nonnil Close clear the result; retain a sole raw cleanup error when the operation succeeded, otherwise join primary error first and cleanup error second. This explicit corrective failure behavior replaces a previously discarded genuine cleanup failure; preserve every pre-existing successful-close result and ordinary error. No errors.Join(primary,nil), second close or earlier release. Keep inner opened-handle release before the outer snapshot descriptor release. A newly failed validation must prevent publication; retain the existing committed true,error behavior when later cleanupCases fails.

Replace only four original fixture defers with checked deferred closures at the same positions, reporting via t.Errorf and preserving every existing assertion/comment/test body. Append bounded real-file characterization before production edits for direct helper success; malformed/schema/mode/identity error precedence, exact deterministic error text and zero failed results; unchanged in-memory snapshot/publication state; and fixed-input canonical snapshot bytes/digests captured against BASE. The preservation controls must pass before and after; actual baseline configured errcheck supplies RED. Passing normal real-file tests does not prove genuine first-Close or simultaneous operation-and-Close failure. Do not introduce an injectable production seam, generic resource helper, reflection/unsafe failure hack, fake second-close experiment or weakened oracle.

Root owns all Flow, parent/MILESTONES, admission, reviews, staging and commits. Worker owns only the two admitted files and task-unique proof excluding root/review reports. Preserve all other production/tests, admission.go, lock.go, artifact/hostfs owners, original schemas/policies/canonical identities, configuration/pins and older artifacts/unrelated files. No dependency/API/runtime/schema/config/suppression/error-discard/worktree/bridge/download/push/history changes.

**Quick commands:** offline cached stock Go1.27.1 first on PATH, GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off, no GOROOT/Gomad seed variables; -count=1 -tags test_dep. Baseline and final full corpus package and focused old/new preservation controls; actual five nested-root architecture/public-signature/external-consumer boundaries; actual unfiltered pinned configured corpus lint --fix=false before/after; errortype; source/gofmt and generator-input inspection, validate if relevant. Run shared-cache/toolchain checks serially. Retain commands, actual exit/time, source/tool/config hashes and diagnostic delta with one lean handover. Do not retry unchanged rootfast419/full/native/missing patched launcher failures or claim package checks fulfill full qualification.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- Both production cleanup attempts stay at their original lifetime boundaries, one attempt each. Nil cleanup preserves exact original result/error identity and all original operations/comments. A sole genuine cleanup failure returns its raw error and zero result; simultaneous failure joins primary first and cleanup second, zero result. No publication follows failed validation, and existing committed true,error paths remain unchanged.
- Four original fixture defers check actual Close via nonfatal testing.T at the same positions; every original assertion remains intact. Append direct real-file success/error-order/zero-result/snapshot-publication controls and fixed BASE canonical bytes/digests. Run them before and after production changes. Disclose genuine first-Close and simultaneous failure execution gaps.
- Baseline and final full corpus, focused controls, five actual nested-root boundaries, errortype and source/gofmt pass on available pinned stock Go. Inspect generator inputs and validate when affected. Actual unfiltered pinned corpus analyzer reproduces original findings and resolves the six admitted sites with none introduced; retain actual residual diagnostics if any. No inferred fresh whole-Gomad count or suppression.
- Only corpus.go/corpus_test.go product sources change; all protected inputs and original acceptance remain unchanged. Independent fresh source review must permit only a source-progress checkpoint before another writer. Root commits implementation, proof and Flow/docs together.
- Original R3/R13/R18/R19, shared fn108 assessment/retention, task12/relevant predecessors/task21, matched first-baseline fixed identities, full/completion/formal/affected-consumer and qualified native darwin/arm64 gates remain required. Stock developmental checks do not fulfill them. Complete the task only after all corresponding source-owned gates pass; otherwise keep acceptance blocked and retain reviewed source progress. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.


## Done summary
Corpus cleanup source progress is independently reviewed and retained for its
separate commit. The two production releases and four original fixture releases
stay at their lifetime boundaries. Nil Close preserves original results/errors;
a genuine cleanup failure clears the result and returns a raw sole error or
primary-first joined errors. Original operations/assertions/comments reconstruct
to BASE; all 1,043 protected inputs remain unchanged. Four real-file controls
preserve helper error order, snapshot/publication state and fixed nonempty BASE
canonical bytes/digests.

Fresh independent serial checks pass corpus 24/24, focused 14/14, five actual
boundaries, errortype and formatting. Actual configured unfiltered corpus lint
falls from six findings to zero with none introduced. Stock linux/arm64 checks
are developmental. Generator inputs are unaffected; validate was not required.

The source review permits a source-progress checkpoint only. Genuine first-Close
and simultaneous operation/Close failure execution remain unproved. Original
R3/R13/R18/R19, shared fn108, task12/predecessors/task21, matched original
first-baseline identities and complete/full/completion/formal/both-native/
affected-consumer qualification remain required and open. Task35/task21 stay
blocked; parent stays open with 2/35 done. Root commits this reviewed source,
proof and Flow/docs before another writer. No acceptance or SHIP claim.

Product/document and all other scoped diff checks pass. Full staged diff-check
exits 2 only for the immutable audit-environment.log's final empty-GOFLAGS line.
The raw log, receipt and worker freeze retain their bytes; the archive warning
is disclosed rather than rewritten.

Requested writer/reviewer models were gpt-6.1-sol/high, same requested family;
actual model metadata is unavailable. The root receipt-audit bookkeeping
preflight and corrected read-only pass are disclosed in acceptance-open.md.

stage: impl-review - skipped(policy: conductor-deferred; fresh source review passed, full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

## Evidence
- Commits: Source-progress commit containing this task's corpus source/tests and frozen proof; resolve from Git history. No push.
- Tests: Worker handover.md/evidence.json, independent-source-review.md/json and root-reaudit.json under .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-35/.
- PRs: None.
- Acceptance: [.flow task35 open gates](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-35/acceptance-open.md).

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
