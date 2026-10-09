---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.57 Check test cleanup results without changing resource lifetimes

## Description
Own exactly16 retained unchecked test cleanup results in12 existing files. New assertions may report genuine cleanup errors; healthy cleanup and all original primary assertions, process statuses, receiver/argument evaluation, defer registration/lifetime/LIFO and callback results remain unchanged. Inline deferred closures keep Close delayed; never write defer require.NoError(t, file.Close()) or replace defers with t.Cleanup.

**Touches:** [tools/gomad3/choice/diagnostic_test.go, tools/gomad3/internal/hostfs/open_test.go, tools/gomad3/runner/choice_exploration_divergence_unix_test.go, tools/gomad3/runner/diagnostics_test.go, tools/gomad3/runner/inspect_test.go, tools/gomad3/runner/retention_characterization_test.go, tools/gomad3/runner/runner_test.go, tools/gomad3/runner/internal/execution/descriptor_dup_linux_test.go, tools/gomad3/runner/internal/execution/process_test.go, tools/gomad3/runner/internal/execution/process_unix_test.go, tools/gomad3/runner/internal/minimizer/workspace_unix_test.go, tools/gomad3/runner/cleanup_test.go]

### Exact correction boundary

The inventory is choice diagnostic session Close1, hostfs root Close1, killed choice-divergence helper stdin Close1, diagnostics artifact Close1, inspect journal Close2, retention replayer artifact Close1, runner tests Close3, Linux descriptor syscall.Close2, flooded process stdout/stderr heads Close2, Unix LaunchResources close1 and killed minimizer lock-holder stdin Close1. Ordinary file/session/root/journal and syscall cleanup require nil. Only the killed helper's parent-owned stdin calls may also accept errors.Is(os.ErrClosed), because Cmd.Wait closes those existing parent descriptors. Never accept EBADF for Linux duplication cleanup. Keep separate deferred registrations and reverse cleanup order, especially stderr before stdout and minimizer process cleanup before stdin cleanup.

Retention replay callback uses its existing replayer.t error reporting without changing stored result/error. matchingReplayer.Replay has no testing.T; use a named error result and inline defer at the current registration point. Preserve detached ReplayResult/call count and current primary error when Close succeeds. Return genuine sole cleanup error directly or primary-first errors.Join only on genuine combined failure, without closing before Snapshot. mutatingExecutor.Run's Write-failure branch closes once before returning; preserve raw Write error when Close succeeds and join Write-first only when Close fails. Its existing successful explicit Close remains unchanged.

Keep child fixture output writes, watchdog readiness, intentional spin loops, exhaustive switches and all other findings outside this correction. Controls include existing healthy close/idempotence/descriptor lifetime/process kill-wait coverage, the guided-admission and matchingReplayer consumers, and additive prepared-target mutation controls where ordinary source execution reaches them. Linux descriptor tests are Linux source coverage; supported Darwin execution stays with its native owner. Existing full replay fixtures may require the unavailable patched toolchain; retain exact failures and their ownership rather than weakening fixtures. Standard successful mutation does not execute simultaneous Write/Close failure. Do not steal descriptors or add races to manufacture errors.


### Execution and retained acceptance

Root admits this exact correction to unblock fn-112.10's required integrated source lint. Task21 consumes its evidence. Depend only on Done task52; progress-only correction owners are verification consumers rather than prerequisite-completion cycles. Root owns selection, integration, review, lifecycle and target commits. A fresh worker owns implementation/tests/evidence in an isolated worktree after planning records are committed. No shared checkout writer or concurrent Go/build/lint/generator gate is admitted. Root schedules the shared gate lane; keep each checked candidate frozen. Handovers under the task-unique artifact directory are lifecycle output, excluded from product Touches.

Read AGENTS.md, tools/gomad3/README.md, MILESTONES.md, the parent/task and retained task55 reconciled proof/review plus task56's reviewed handover when available. Re-anchor at the committed worktree base; keep actual original-base RED and exact source hashes as defect evidence. Use apply_patch and established libraries, pinned stock Go1.27.1, all Go tests -tags test_dep -count=1, the established cache/file-proxy environment and private temporary directories. Run focused baseline/final controls, ordinary affected packages, affected vet/standalone errortype, relevant architecture/public/purity/private-injection checks, both supported static source sets, fresh check-only validation, format, actual unfiltered affected configured lint, actual FIX=false make lint-code-fast against the worktree admission base, and original-base make --trace lint-code-gomad3 against951c5516e9e7b3066e7e069adda9565cfd68844c. Reuse valid exact-source receipts. Retain raw source/tool/command/exit/elapsed bindings and exact removed/introduced/residual findings; fast-filtered green does not replace red unfiltered lint or integrated errortype.

Parallel scope independence grants no acceptance waiver. Root independently verifies and reviews the integrated target and runs focused integrated controls before completion. A red required source gate licenses only a reviewed source-progress commit, keeps formal acceptance in_progress, and supplies no formal SHIP or full-spec acceptance. Preserve original first-baseline/preservation/R18/R19 requirements and explicit native fn149/fn128 deferrals. No native-host spoofing, new fault hook/seam, framework/library, policy/pin/API change, weakened assertion, unrelated comment edit, PR, push or CI authority. Disclose every unreachable genuine cleanup-failure branch precisely.

## Acceptance
- [ ] Exactly16 cleanup findings are removed with zero introduced findings. Each call retains original lifetime/order/evaluation and healthy primary outcomes; only admitted genuine cleanup errors change reporting.
- [ ] Baseline/final ordinary reachable controls retain all original assertions and meaningful lifetime/process/descriptor/result observations. Genuine cleanup and native execution gaps have source proof and precise ownership.
- [ ] Frozen focused/ordinary/static/validation/format/vet/errortype/actual configured-fast-original lint receipts bind the actual candidate. Required red source gates remain open.
- [ ] Root verifies the integrated target, obtains fresh independent review and commits this task separately; Done requires all still-owned source acceptance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
