---
satisfies: [R4]
---
# fn-124-shrink-and-simplify-the-umpire-go.4 Define verdict aggregation once and document the judge's generic rules

## Description
Implements R4: one verdict-aggregation function used by the evaluator, recorder, recordedrun Agreement and replay form; the other generic judge rules documented in common/testing/testpilot/README.md as the judge's semantics, each with a test.

Dependency note (2026-10-05): the 4 → 3 dependency was dropped. Task 3's R3 work edits `internal/execution/{scheduler,response_read,values,evidence}.go`, `temporal/*`, `internal/delivery`, `conformance` and `lower`; this task edits `internal/execution/recorder.go` (+ new `verdict.go`), `internal/verification/evaluator.go`, `recordedrun/recordedrun.go` and `replay/form.go`. The only shared file is the Testpilot README (textual merge). Task 5 now depends on 3 and 4, since its lowering validates declared outcomes with `ConcludeVerdict`.

### Approach
- `Conclude(disposition RunDisposition, rules []RuleVerdictStatus) (VerdictStatus, RunDisposition)` in `internal/execution/verdict.go`, beside the `Monitor` contract; the facade re-exports it as `testpilot.ConcludeVerdict` (recordedrun and replay import only the facade, per `.plans/UMPIRE_MODULES.md`). Semantics: any violated rule → violated and the Run stopped by its Monitor; a completed Run whose rules are all satisfied → satisfied; otherwise inconclusive, disposition unchanged.
- Call sites: `internal/verification/evaluator.go` `verdict()` (≈486-499; disposition INCOMPLETE when `e.incomplete`; correlated rules pending at close stay inconclusive); `internal/execution/recorder.go` (≈397-407) close override; `recordedrun/recordedrun.go` `Agreement` (≈276-313) keeps its refusal of pending/unspecified rules then compares against `Conclude`; `replay/form.go` `ViolatedForm` (≈29-57) drops the re-derivation and keeps the incomplete/cleanup checks and today's classes.
- Out of scope: `evaluation/assess.go` (task 6); `umpire-run/run.go` and `explore/bridge.go` only map statuses.
## Acceptance
- [ ] One aggregation function (`Conclude(disposition, rules) (verdict, disposition)` in `common/testing/testpilot/internal/execution/verdict.go`, re-exported by the facade as `testpilot.ConcludeVerdict`) is the only definition; `evaluator.verdict`, the recorder's close, `recordedrun.Agreement` and `replay.ViolatedForm` call it. A grep for hand-written `VERDICT_STATUS_SATISFIED`/`VIOLATED` assignments in those four files finds none.
- [ ] A table test drives `Conclude` over every disposition × rule-status mix (no rules, pending, unspecified, stopped without a violation), and shows the old code gave the same answers before the copies were deleted.
- [ ] `common/testing/testpilot/README.md` has a section "How a Run is judged" stating the six generic rules (verdict aggregation; silence is inconclusive; freeze after the first violation; deadline checked before transitions; disposition precedence; correlated evidence deduplicated by identity, conflicting duplicate malformed), each naming one test that fails if the rule is broken. `internal/verification/README.md` links to it instead of restating.
- [ ] Existing evaluator, recorder, recordedrun and replay tests pass unchanged, including the replay classes `malformed`/`incomplete`/`non-violated`.
- [ ] `go test -tags test_dep -p 2 ./common/testing/testpilot/... ./tools/umpire/... ./tools/canary/...`, `make lint-code-fast` and `make umpire-check-cases` pass with Case bytes unchanged.
## Done summary
Verdict aggregation has one definition, and the judge's six generic rules are written down in the Testpilot README, each pinned by a named test (R4).

What changed
- `common/testing/testpilot/internal/execution/verdict.go`: `Conclude(disposition, ruleStatuses) (VerdictStatus, RunDisposition)`.
  - Any violated rule: violated, and the Run stopped by its Monitor.
  - A completed Run whose rules are all satisfied (or that has no rules): satisfied.
  - Anything else: inconclusive, disposition unchanged.
- The facade re-exports it as `testpilot.ConcludeVerdict` (in `prepared_case.go`, following `var InstructionOpcode = execution.InstructionOpcode`). The module map lists it.
- Call sites:
  - `evaluator.verdict()` treats an incomplete evaluation as an incomplete disposition, then concludes over its rule results. The unused `Evaluator.satisfied` counter and `recordCorrelated`'s bool return are gone.
  - The recorder's close forces incomplete when the recording is incomplete, then concludes the Monitor's answer.
  - `recordedrun.Agreement` still refuses unspecified and pending, then compares against `ConcludeVerdict`.
  - `replay.ViolatedForm` drops its own any-violated loop; its result classes are unchanged.
- Tests:
  - `TestConclude`: 4 dispositions × 10 rule mixes.
  - Commit 1 ran agreement tests against the four old copies before they were replaced: `TestEvaluatorVerdictConcludes`, a recorder test, `TestAgreementIsConcludeVerdict`, `TestViolatedFormConcludesThroughConcludeVerdict`.
  - New per-rule tests: `TestSilenceIsInconclusive`, `TestEvaluationFreezesAtTheFirstViolation`, `TestCorrelatedEvidenceIsDeduplicatedByIdentity`, `TestRunDispositionPrecedence`. The deadline rule uses the existing `TestEvaluatorDeadlinesAndReplay`.
- Docs: the Testpilot README gains "How a Run is judged" (six rules, each naming its test). The verification README links to it instead of restating it, and the execution README points to it.

Decisions, and why
- The recorder concludes from the Monitor's verdict status, read as one rule status (`monitorAnswer`), not from the Verdict's rules. Test Monitors return a status with no rules.
- Two behaviour changes, both for Monitors that break their contract and that no test covered:
  - A Monitor that stops the Run but answers satisfied is recorded inconclusive, still stopped. Before, it stayed satisfied, which `Agreement` already rejected.
  - An unspecified Monitor status is recorded inconclusive.
- `Conclude` keeps the disposition of a Run stopped without a violation, so `Agreement` keeps its own "stopped only beside a violation" check. Its messages and table test are unchanged.
- Each rule's test was chosen by disabling the rule in code. Before this task, nothing failed without the freeze, without the conflicting-duplicate check, or when incompleteness stopped overriding a stop without a violation. Correlated silence was caught only by the 19-fixture `TestCorrelatedCheckedLeanFixtures`.
- Known limit of rule 5: `TestRunDispositionPrecedence` tests the recorder's close. A failed cleanup wrongly making a Run incomplete would be in `runtime.go`, which the existing `TestRunTerminalPrecedence` catches.
- Out of scope: `evaluation/assess.go` (task 6) and the status mapping in `umpire-run`/`explore`. No Model, lifter or Case changed, so the model gate was not run.

Gates (under the shared heavy-suite lock; logs in `.flow/tmp/fn124-4/` of the task worktree)
- Full Go suite: `go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/... ./tools/umpire/... ./tools/canary/...`.
  - Exit 0: 48 packages, 1534 tests, 0 failures, 18 skips.
  - 263 s wall after a 98 s lock wait.
  - Slowest tests: `lower.TestOriginalBaselineCases` 53.5 s, `model.TestMigrationProjectionPreservesSemantics` 48.2 s, `model.TestOriginalBaselineModel` 47.1 s. All three belong to the migration harness that task 7 retires.
- `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast`: 0 issues.
  - Plain `make lint-code-fast` compares against the stale local `main` and auto-fixes. It reported findings outside Umpire and edited unrelated files; those edits were reverted.
- `make umpire-check-cases`: exit 0, Case bytes unchanged.
- No failures predate this task in the gated suites.

Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP with 3 P3s, all applied in a9f941437e:
- The evaluator verdict test no longer sets fields `verdict()` does not read.
- `TestRunDispositionPrecedence` pins that an unspecified Monitor answer is inconclusive.
- README rule 1 says `Agreement` also refuses a Run stopped without a violation; `recordedrun.go`'s check reads `!violated && disposition == STOPPED` (equivalent).
Focused tests after the fixes: `go test -count=1 -tags test_dep ./common/testing/testpilot/...` pass. The fixes touch tests, docs and one equivalent boolean, so the heavy gates were not repeated.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: eda11d0b5de3706c2eb6ab70426b3e695730fe90, 2e2bc11449733b437e8701908d38c6e9cb9451f6, f892230d11c48947a4f636d47da7efab40702499, 4eeb27d343a0221325475c6876382fe615f65de7, 8d0284e4fee5fe13bd64ea4197ff525049ca5ee9, a9f941437e7bd600282fca61debbae74cc334ba1
- Tests: go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/... ./tools/umpire/... ./tools/canary/... (exit 0, 1534 tests, 0 failures; .flow/tmp/fn124-4/go-suite.json), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (exit 0, 0 issues; .flow/tmp/fn124-4/lint.log), make umpire-check-cases (exit 0, Case bytes unchanged; .flow/tmp/fn124-4/check-cases.log), mutation check: each README-named judge-rule test fails when its rule is broken (scratchpad mutate.sh), go test -count=1 -tags test_dep ./common/testing/testpilot/... after review fixes (exit 0, 20 packages ok; .flow/tmp/fn124-4/review-fixes-tests.log)
- PRs: