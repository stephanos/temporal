# Independent qualification storage source review

Verdict: SOURCE_PROGRESS_COMMIT_ONLY. No actionable introduced defect was found. This permits the separate task 37 progress commit under MILESTONES; it is neither formal SHIP nor task/spec completion.

Reviewed branch `gomad`, BASE/HEAD `22fe28d1d6b641e9deea5bf9b60a1da773c06381`, with the integrated candidate uncommitted. Requested reviewer and writer: `gpt-6.1-sol at high`, same requested family, fresh review context. Executed model identity is not exposed; `actual_model=null`. Tier: session (jev-unavailable(no_key)). The configured formal backend `claude:claude-fable-5-1:high` remains unchanged and uninvoked.

## Strengths

The five cleanup sites stay at their original boundaries. At qualification.go:248, deferred Remove addresses only the staging name and preserves the returned publication path. Its ErrNotExist exception requires that publication supplied a path. Chmod/Write/Sync failure paths at :260–274 close once; nil cleanup retains the contextual primary directly, while nonnil cleanup joins primary first. Existing successful Close context and directory Sync-before-Close precedence remain intact. Reader cleanup at :298 spans all Stat, regular/16 MiB, bounded read, one-newline and canonical/semantic decode work, and clears the report only on nonnil Close.

The tests at qualification_test.go:232 add real-file preservation controls with independent saved BASE literal bytes, modes, names, validation order, messages/causes, decoded evidence and replay/repeatability fields. All original test assertions/comments remain intact. The saved historical fixture Go/platform values remain data. Source reconstruction and 1,045 protected input comparisons support the deliberately narrow scope.

Actual callers preserve failure propagation: workload.publishWorkload returns path and error, CLI qualify.go:172 maps storage errors to runner_failure/status 3, and set.retainedQualification and soak.openBatchReport reject reader errors before consuming the report. These are source-inspection conclusions, not fault executions.

## Findings

Critical: none. Important: none introduced within the admitted source-progress scope. Minor: none.

## Fresh verification

The exact argv, cwd, environment, tools/config/source hashes, UTC times, results, output hashes and source stability are in [the review receipt](independent-source-review.json). All actual Go commands used existing stock Go1.27.1 first on PATH, GOENV/GOWORK off, GOTOOLCHAIN local, GOPROXY off, empty GOFLAGS and all three seed variables unset. The host is developmental linux/arm64.

| Check | Result | Seconds |
| --- | --- | --- |
| source-verification | exit 0 | 0.716 |
| saved-base-controls | 8/8 passed; exit 0 | 1.061 |
| verified-controls | 8/8 passed; exit 0 | 0.366 |
| verified-package | 23/23 passed; exit 0 | 0.377 |
| verified-consumers | 60/60 passed; exit 0 | 0.808 |
| verified-boundaries | 5/5 passed; exit 0 | 9.356 |
| verified-purity-edges | 2/2 passed; exit 0 | 29.698 |
| verified-errortype | exit 0 | 0.655 |
| verified-gofmt | exit 0 | 0.010 |
| final-lint-corrected | exit 1; unchanged protected gci | 0.394 |

The fresh read-only worker verifier matches retained output SHA-256 `4c8cae8c5719a3134a0a6cb63cf6602db4b234037df35a5bccfb882ebafad0d3`. Final source hashes are `063486bb1bdd13c0f1c92b454b6f10befaa0e13e7b4dff7fc990710dfa09550c` and `c4d26e3b02322961d5e1a85f0995b0b5a9cd34d695b2e6e55825700b3e5d7e19`. The 59-file worker freeze is unchanged.

Raw baseline lint reproduces exactly five original errcheck findings plus protected diagnostics_test.go:11:1 gci. Fresh final unfiltered pinned lint retains exactly that gci and no introduced findings. Both lint stages are RED overall. The intermediate QF1001 and wrong new-test malformed-JSON literal are retained as authoring/intermediate results; neither supplies cleanup-fault RED. No current whole-scope count is inferred from the historical task25 419-finding receipt.

An independent read-only comparison verified all 31,879 BASE tracked nonowned paths, including file modes, symlinks and Gitlink. The two root prose additions reconstruct exactly; parent metadata changes only its timestamp, task21 adds only task37's dependency plus timestamp, its historical blocked reason is unchanged, and task37's original prefix/metadata remain preserved. Task37/task21 are blocked; parent remains open with completion review unknown and 2/37 accepted. Existing root record hashes match its intended 74 paths. The reviewed record intentionally remains PENDING.

The Makefile generation prerequisites and actual protocol schema/template/identity input lists exclude these two changed files; those inputs stay pinned. Generation is not required for this bounded edit. Scoped product/root-document diff checks are clean.

## Required limits and handoff

- Real first-Close errors, early primary plus Close errors, reader primary plus Close errors, sole/multiple staging Remove errors, and post-Rename directory open/Sync/Close errors are UNEXECUTED. Source inspection and ordinary files are not runtime fault proof.
- Actual unfiltered qualification package lint is RED on protected diagnostics_test.go:11:1 gci. Original root-fast/full host and formal review have not passed.
- All original R18/R19/R20, predecessors/task21/shared fn108, availability/default/format, matched first-baseline fixed identities, complete/full/completion/formal/affected-consumer/native-default, integration/smoke/core/runtime/process/overlay and both darwin/arm64 plus linux/amd64 obligations remain REQUIRED/OPEN.
- Developmental linux/arm64 stock Go results establish no supported-platform patched runtime confidence or replay qualification.
- Root must bind these reports and verdict, run the unmodified root-scope-gate.py --staged for the exact 74 paths, commit separately and verify after commit. That gate is NOT RUN by this reviewer.

Root must bind both review files and the verdict before running the unmodified final `root-scope-gate.py --staged`. That gate has NOT passed as part of this review. Root retains commit/index/Flow ownership and must verify the separate progress commit afterward.

Only these two review reports were written. No product, frozen proof, root record, config, index, HEAD, branch/history, worktree or backend mutation; no network/download/cache cleanup or delegates. All command handles are terminal.

Report-path correction: the first patch resolved relative to workspace cwd and created only two report copies outside the repository. Both misplaced files were removed with apply_patch, and these reports were written at the admitted absolute paths; no product/proof/config/Git/Flow record was touched.
