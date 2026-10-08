---
satisfies: [R9]
---
# fn-114-gomad-correct-search-path-defects-and.13 Order runtime-owned goroutines by a fixed rule and offer only user goroutines as alternatives

## Description

Current acceptance candidate is documented in [the retained R9 source handover](../artifacts/fn-114-gomad-correct-search-path-defects-and/task-13/source-acceptance-20261007/handover.md) and [source-bound evidence](../artifacts/fn-114-gomad-correct-search-path-defects-and/task-13/source-acceptance-20261007/evidence.json). Fresh review must inspect the entire retained R9 selector, runqget path, primary zero/one/two/busy fixtures and matcher, secondary finalizer-as-user assertions, canonical RUNTIME.SCHEDULING, controller identity/refusal and exact first-committed-body/compact preservation map. A compact alpha-rename diff alone supplies no R9 approval. The handover identifies the pre-existing secondary fixture header discrepancy for review and preserves the historical dirty-snapshot/native-measurement limits. Native runtime/fixture execution and parent aggregate lint remain with their mapped owners.

Current source acceptance reuses the integrated R9 implementation and the completed fn-110.2 scheduler/field-compaction preservation chain. Conductor admits task-local source-acceptance evidence only to Touches. Native cause, progress, decision-count, runtime, replay and full-host executions remain with the mapped fn-149/fn-128 owners. The worker keeps implementation and acceptance criteria intact, binds relevant current source and runs uncovered ordinary/static/generated checks. Root alone owns Flow, milestones, review, completion and Git.

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

E4 (R9): the run-queue choice offers only user goroutines; runtime-owned goroutines are ordered by a fixed rule. Last of the three runtime edits. Depends on task 12 only to keep the overlay, patch, and fixture edits serial.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch` (the `runqget` hunk, only if the call site must change), a control-probe fixture under `tools/gomad3/internal/gomadtool/conformance/testdata/`, `runtime_scheduling.go`, `tools/gomad3/SPEC.md` (`[RUNTIME.SCHEDULING]`), `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/control-probe.md`
**Touches:** [tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/**, tools/gomad3/toolchain/**, tools/gomad3/choice/**, tools/gomad3/internal/gomadtool/conformance/**, tools/gomad3/SPEC.md, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/control-probe.md, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-13/source-acceptance-20261007/**]

### Approach
- First step: confirm the inferred cause on a fixture that deliberately starts two user goroutines. Show that the extra branching decisions have runtime-owned goroutines among their alternatives. The historical D21 control source starts no goroutine explicitly; its reported peak of 2 is not a substitute for this fixture. If the cause is absent, record the narrowed E4 finding with evidence and stop.
- Classify run-queue entries with the runtime's own system-goroutine test. Use the task 5 inventory as the list of runtime-owned kinds, and state how a finalizer or cleanup goroutine running user code is classified.
- The rule: state it once, for example runtime-owned goroutines run before user goroutines in queue order. It must be a function of the queue contents only, and must not starve either class.
- A queue with at most one user goroutine records no decision and still yields a deterministic pick. A user dispatch with two or more user goroutines records one decision whose alternatives are the user goroutines only; a runtime-owned queue head runs deterministically without recording a decision.
- Replay and forced prefixes apply the same rule; a tape recorded under the old rule is rejected by controller identity, never reinterpreted.
- Collector workers are picked outside the run queue. State in the contract that the rule covers the local run queue and name what it leaves out.
- Write the rule into `[RUNTIME.SCHEDULING]` in `SPEC.md`.
- Measure the control probe's decision counts before and after and retain them.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:778-803` — run-queue choice and its buffers
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:572-592` — `runqget` hunk and the pick it performs
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:495-520` — decision recording, including the single-alternative path
- `tools/gomad3/SPEC.md:218-228` — scheduling and choice contract
- `docs/research/gomad/2026-10-01-feasibility-schedule-search.md:739` — the control probe and its 26 decisions

**Optional** (reference as needed):
- `tools/gomad3/internal/gomadtool/conformance/testdata/runqueue/` — existing run-queue fixture
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:355-362`, `:478-482` — existing uses of the system-goroutine test
- `tools/gomad3/choice/trace.go:100` — controller implementation identity

### Key context
- Task 1 re-anchor (2026-10-02): E4 is changed in its historical probe premise. `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-control.go.txt` has no `go` statement. Its seed-11 report still records 26 branching Runnable decisions and peak goroutines 2 (seed 17 records 29). `gomadChoiceRunqIndex` still includes every local run-queue identity without a system-goroutine filter, but the reported peak does not establish two deliberately started user goroutines or attribute each extra decision. Keep the explicit two-user fixture, cause check, fairness, and before/after acceptance; no runtime correction was made in task 1.
- This changes schedules for every seed. It is a new choice-controller identity, and existing dispositions must not be weakened to obtain a pass in task 14.
- The run-queue choice runs on the system stack with fixed buffers. Classification must not allocate.
- fn-110 task 2 moves the scheduler implementations into the overlay; fn-112 tasks 3 and 5 edit the same file. Check their state first and rebase onto whichever landed.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] The cause is confirmed on the control probe, or E4 is recorded as changed with the evidence
- [ ] A program with two user goroutines records only decisions whose alternatives are user goroutines
- [ ] A queue with zero or one user goroutine records no decision and the pick is equal across runs
- [ ] A fixture with a busy runtime-owned goroutine and two user goroutines shows neither class is starved
- [ ] A tape recorded before the change is rejected by identity
- [ ] `[RUNTIME.SCHEDULING]` states the rule for runtime-owned goroutines and what it leaves out
- [ ] Control-probe decision counts before and after are retained in `control-probe.md` in the spec's artifacts directory
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded
## Done summary
Gomad developers have retained source acceptance for the runtime-owned local-queue rule. The current selector dispatches runtime-owned heads without a choice and offers only user identities for user dispatches. Canonical RUNTIME.SCHEDULING, primary zero/one/two/busy fixtures, all-alternative assertions, queue advancement and old-controller refusal agree. This run changed no product source.

The first committed R9 selector at a3b9f80efab9356c0be2080779133337e2471ac0 equals the current full body after one gomadIdentity-to-gomadID substitution. Four primary fixture files remain byte-identical to that commit and the historical dirty snapshot. The historical receipt's d635e23 HEAD alone contains pre-R9 code and does not reproduce its dirty snapshot. Conductor verified 87 current preservation inputs, all 72 retained raw files with 521,317 original bytes, tool/archive hashes, 31 references and the 5,089-file current source snapshot. Both-source-set materialization, inventories and identity evidence remain reusable under those exact bindings.

Fresh checks pass for generated validation, 50 portable tests, four architecture/public-signature/purity/vet tests, scoped configured lint/errortype and formatting. Conductor reran 26 selected controller/replay/trace tests with zero skips, including old-controller refusal, and scoped configured lint. The wire package collected no matching tests in that conductor command and receives no test credit. Scoped lint uses the recorded pre-R9 new-from-rev filter; untouched inherited findings and parent whole-project lint remain open.

Fresh correctness, contracts and integration reviews each return SHIP with zero findings and met source R9 coverage. Reviewers inspected the full retained selector, queue path, fixtures, controller identity/refusal and canonical contract. They independently checked materialization and preservation; attempted Go reruns could not create build directories in the read-only sandbox. Mechanical finalization records SHIP. Reviewer and writer selectors are gpt-6.1-sol/high, the same GPT family; actual executing model metadata is unavailable.

Historical 32-seed measurements retain their original identities and totals of 2,261 to 2,080 decisions and 31 to zero runtime selections. Current native cause/progress/count/runtime/replay/full-host evidence remains deferred under fn-149/fn-128; stock linux/arm64 supplies no native pass. The secondary fixture's pre-existing run-first header remains unchanged and disclosed; canonical head-class wording and its finalizer-as-user assertions remain consistent. No push, PR, CI or toolchain download/build occurred.

stage: impl-review - SHIP (retained full R9 source context and nonempty compact source range; source-review.json)
stage: plan-sync - skipped(config: planSync.enabled=false)
Quality checks: applicable current source, preservation and scoped standards pass; parent aggregate lint remains open.
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: a3b9f80efab9356c0be2080779133337e2471ac0, a936b597b4c62fa50f11a6c16c91111cd52b1ec3
- Tests: make -C tools/gomad3 validate, go -C tools/gomad3 test -tags test_dep -count=1 -v -run '^(TestResolve.*|TestRuntimeCampaign.*|TestRequireStockCompatibilitySelectsPinnedToolchain|TestValidate.*|TestBenchmarkMedianNS|TestRepeatabilityMismatchRetainsDivergentEvidence|TestTimerCallbackAssociationRejectsUnidentifiedHandoff|TestStableHandoffsRequireOneAlternativeSetAndConsistentLeaders|TestRun(Upstream|Builder|LiveCapability|Accepts|Rejects|Reports|Interception).*|TestExecWrapper.*|TestRuntimeOwnedRejectsPreviousController|Test.*Replay.*|Test.*Trace.*|Test.*Alternative.*|Test.*Implementation.*|Test.*Prefix.*)$' ./choice/... ./internal/gomadtool/conformance, go -C tools/gomad3 test -tags test_dep -count=1 -v -run '^(TestPackageArchitecture|TestPublicPackagesDoNotExportTypeAliases|TestPureModulesHaveNoHostEffects|TestHostPackageVet)$' ., make lint-code GOLANGCI_LINT_BASE_REV=d635e23f00d926a43b942f25a9d05bd0ccb72025 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype LINT_CODE_DIR=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3 'LINT_CODE_TARGETS=./internal/gomadtool/conformance ./choice/internal/wire' ALL_TEST_TAGS=test_dep, go -C tools/gomad3 test -tags test_dep -count=1 -v -run '^(TestRuntimeOwnedRejectsPreviousController|Test.*Replay.*|Test.*Trace.*|Test.*Alternative.*|Test.*Implementation.*|Test.*Prefix.*)$' ./choice/... ./internal/gomadtool/conformance
- PRs:
## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implemented scheduler/search behavior, current-source Darwin runtime/full/core/smoke/representative exact replay, measurements, docs and review. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
