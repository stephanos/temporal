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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
