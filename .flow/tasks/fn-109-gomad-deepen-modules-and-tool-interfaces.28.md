---
satisfies: [R16, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.28 Preserve campaign policies while checking cleanup errors

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Bounded R16/R18/R19 campaign source repair after documentary checkpoint d7c6695cff81a1160ee482cb82b62cf592f4f199. Original task3/predecessor and task21 qualification remain unchanged; this owner does not force-start old tasks.

**Size:** M
**Touches:** [.github/.golangci.yml, cmd/tools/lintcode/lint_policy_test.go, tools/gomad3/runner/internal/campaign/controller.go, tools/gomad3/runner/internal/campaign/open_campaign.go, tools/gomad3/runner/internal/campaign/resume_plan.go, tools/gomad3/runner/internal/campaign/campaign_journal_test.go, tools/gomad3/runner/internal/campaign/merge_capacity_test.go, tools/gomad3/runner/internal/campaign/segmented_journal_test.go, tools/gomad3/runner/internal/campaign/controller_test.go, tools/gomad3/runner/internal/campaign/controller_completion_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/**]


Owner decision (2026-10-07). The owner permits a lint exception only for the two exact existing pre-mutation panic statements in SeedController.Complete in tools/gomad3/runner/internal/campaign/controller.go, with messages "gomad3: completed an inactive campaign attempt" and "gomad3: completed a campaign attempt without a classification". Preserve conditions, messages, order, panic mechanism and zero-mutation/rejection tests. Restrict the exception to that exact file and those source statements and prove actual-tool refusal for another statement/path. This supersedes the older suppression/policy-change ban only for these sites; every other lint rule and preservation requirement remains. Root admits .github/.golangci.yml and cmd/tools/lintcode/lint_policy_test.go to this task's Touches solely for this rule and its regressions. See [the bounded decisions](../artifacts/source-unblocking-20261007/owner-decisions.md).

Root separately owns parent/task Flow, MILESTONES and review/commit metadata. One checkout writer.

Read full AGENTS.md, Gomad README, MILESTONES, original fn109 spec/task3 and .flow/tmp/campaign-error-source-mapping.md plus .flow/tmp/nested-exhaustive-source-mapping.md. Reuse grounded mappings rather than repeat discovery. Follow TDD, writing-good-tests, code-style and verification-before-completion. Before production edits retain actual failing pinned lint for all17 package findings, behavioral baseline and any meaningful new regressions. The lint analyzer is the existing source-policy regression for the mapped15 diagnostics; no manufactured runtime failure or text-only test substitutes for unavailable close-failure injection.

Replace both controller failure-policy switches with direct first/budget predicates preserving validation order, initial counters, All no-action, transaction order, first-only cancel boolean, budget drain and existing literal whole-statistics vectors. Keep both invariant panic bodies and their strong zero-mutation/rejection tests unchanged. No log.Panic, wrapper, suppression, policy waiver, panic removal, fake defaults or completion-error redesign.

Check all13 mapped Close calls. Deferred test cleanup uses inline deferred closures reporting nonnil errors at the original registration points; publishMergeShard cleanup remains inside helper lifetime rather than t.Cleanup. Torn-tail setup Write failure keeps the write error primary and reports/closes once. Three production os.Root closes use named returned errors and conditional errors.Join(primary,closeErr) only when closeErr is nonnil. Preserve original error object/text when nil, returned data, public function types, statement order, publication/config-update order, canonical bytes and failure classification. Reuse existing lifecycle/segmented patterns; no new public seam, framework, retry or rollback.

Strengthen narrowly necessary literal characterization before production edits for constructor All/resumed budget and primary errors (target identity, existing-plan refusal, changed published segment) where current coverage is indirect. Expectations come from current real behavior and hand-checked values, not code under test. Do not weaken or regenerate existing expectations. Inspect active pinned Go os.Root.Close source; Unix returns nil, so disclose that no nonnil-root-close injection is demonstrated rather than adding fake coverage.

Quick verification from tools/gomad3 uses pinned stock Go1.27.1, GOWORK=off GOTOOLCHAIN=local GOPROXY=off -count=1 -tags test_dep and cleared GOMADSEED/GOMAD3_CHILD_SEED. Baseline then final whole ordinary ./runner/internal/campaign package; focused controller/rejection/plan/publication/torn-tail tests; relevant root TestPackageArchitecture and external consumer boundary checks if dependencies/public seams require. Inspect generator VERSION_INPUTS/BOUNDARY_INPUTS/COMPATIBILITY_INPUTS before edits and retain hashes/required validation disposition. No generated/runtime/protocol/pin/config/dependency changes or tools/downloads. Actual patched/native tests stay open when unavailable; stock developmental evidence does not replace them.

Run actual pinned golangci v2.13.0 unfiltered affected package and errortype before/after with unchanged gitroot config, test_dep and fix=false. Expected bounded source delta is13 errcheck plus2 exhaustive removed,2 intentional forbidigo invariant diagnostics retained; verify exact actual source-bound delta, do not label unfiltered lint green. Broader419 is a historical frozen receipt until final broad qualification reruns; do not claim a new whole-scope count from a package-only gate or rerun unchanged full rootfast now. Retain exact command/env/cwd/start/end/elapsed/exit, tools and stable focused source hashes. One lean handover/evidence, meaningful raw gate logs and before/after findings; reference broader retained evidence, no duplicate bulk manifests.

Root alone owns Git/Flow/independent fresh review/progress commit under conductor-deferred override. Worker does not stage/commit/review/complete Flow, use worktrees/stash/bridge/push/history rewrite or mutate outside Touches. Independent read-only scouts may overlap without shared source writes. No green baseline handoff. Return only when owned commands/delegates are terminal and handover/evidence enumerate genuine source progress, actual model when evidenced and remaining qualification. Root reviews and commits verified owned progress before next writer. Original R16/R18/R19/task3/predecessors/task21, exact fixed-identity, full/formal and the Darwin native gate remain open wherever unproved. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.

## Acceptance

Owner decision (2026-10-07). The owner permits a lint exception only for the two exact existing pre-mutation panic statements in SeedController.Complete in tools/gomad3/runner/internal/campaign/controller.go, with messages "gomad3: completed an inactive campaign attempt" and "gomad3: completed a campaign attempt without a classification". Preserve conditions, messages, order, panic mechanism and zero-mutation/rejection tests. Restrict the exception to that exact file and those source statements and prove actual-tool refusal for another statement/path. This supersedes the older suppression/policy-change ban only for these sites; every other lint rule and preservation requirement remains. Root admits .github/.golangci.yml and cmd/tools/lintcode/lint_policy_test.go to this task's Touches solely for this rule and its regressions. See [the bounded decisions](../artifacts/source-unblocking-20261007/owner-decisions.md).


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Both policy actions preserve literal statistics, construction/resume admission, first-only cancellation and budget drain; invariant rejection and zero-mutation remain unchanged.
- [ ] All13 mapped cleanup returns are checked at original lifetimes; conditional production joins preserve primary error/data/transaction ordering and nil-close error identity.
- [ ] Whole ordinary campaign baseline/final and meaningful focused regressions have stable terminal receipts, existing literal byte/error expectations retain strength, and unavailable nonnil-root-close proof is disclosed.
- [ ] Actual unfiltered lint/errortype before/after establish exact15 mapped repairs and all residual diagnostics without suppression; generator/public-boundary obligations are verified or explicitly remain open.
- [ ] Fresh independent source review finds no actionable introduced defect; root commits verified progress before another writer, retaining original full/native/formal and task21 acceptance.


## Done summary
SOURCE_PROGRESS_ONLY; authoritative Flow status is blocked, not done.

The reviewed seven-file source correction checks thirteen cleanup returns at their
original lifetimes and replaces two policy switches with equivalent direct
predicates. First-only cancellation, budget drain, both pre-mutation invariant
panics, original completion vectors, public function types and transaction order
remain unchanged. Conditional production joins preserve the original error object
and data when Close returns nil.

Actual unfiltered pinned campaign lint changes from 17 to 2 diagnostics: thirteen
errcheck and two exhaustive findings resolved, no introduced findings, and two
unchanged invariant forbidigo findings retained. The intermediate QF1003 findings
were fixed without a rule change. The initial failed characterization calibrated
an existing two-cause error expectation; it is not a runtime RED claim.

The worker's whole ordinary package, focused tests, errortype, boundary and make
validate checks pass on stock Go 1.27.1 developmental linux/arm64. Fresh independent
review reran ordinary package, 51 focused tests, errortype and boundary checks,
verified all selected source/tool/log receipts and found no actionable introduced
Critical, Important or Minor issue. Root verified the frozen source and review
bindings before this source-progress commit. The active Unix Root.Close returns
nil; no real nonnil-root-close execution is demonstrated.

Original task 3/predecessors, R16/R18/R19, task 21, matched first-baseline identities,
full/formal and both native patched-runtime qualifications remain open wherever
unproved. The broader 419-finding receipt is historical; no new whole-Gomad count
is inferred from package-only progress. Implementation review and plan-sync are
deferred: no formal SHIP, task-done event or downstream synchronization is claimed.

stage: impl-review - skipped(policy: conductor-deferred; fresh independent source review passed, but full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

## Evidence
- Source progress and qualification: [handover](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/handover.md), [worker evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/evidence.json), [acceptance open](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/acceptance-open.md).
- Fresh independent review: [review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/independent-source-review.md), [checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/independent-source-review-checks.json).
- Root checkpoint verification: [checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-28/source-checkpoint-verification.json).
- Commits: f34369bf64eabf00d50ac3c720dd42bda07ccc25 (verified source progress; not a Flow completion receipt).
- PRs: none; no push.

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
