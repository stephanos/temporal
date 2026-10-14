---
satisfies: [R1]
---
# fn-94-simplify-the-testpilot-go-runtime.1 E1 decision and correlated field validation in admission

## Description
Records the E1 owner decision and closes the admission half of the correlated gap: every field list is validated, and output-row equality compares fields as Lean's `Result` does. The monitor half (states carried as atom plus fields) is fn-94.3, split so each stays one reviewable change in one file.

**Owner decision (recommended default: read).** Record in the receipt: Lean decodes `prior_state`+`prior_fields`, `state`+`state_fields` and `initial_state`+`initial_state_fields` as `StateValue` (`model/Testpilot/Correlated.lean:103-111,284,374`; `model/Shared/SemanticData.lean:19`), and its monitor matches transitions on that value. Removal would need a Lean `StateValue` redesign and its decode agreement proof (a Case such as `nexusCallerTests-startToCloseTimeout` has initial fields no result row reproduces), so the conductor takes "read" unless the owner said otherwise.

**Size:** S
**Files:** `common/testing/testpilot/internal/verification/correlated_prepare.go`, the package's correlated admission test file
**Touches:** [common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/internal/verification/correlated_prepare_test.go]
**Depends on (cross-spec):** fn-89-one-contract-rule-per-entity.5 (edits `verification`; its code has landed, only its receipt remains)

## Approach
- In correlated admission (`correlated_prepare.go:92,160-170`), validate every entry of `StateFields`, `PriorFields` and `InitialStateFields` with the same `validModelValue` the other model values use. Add the new checks after the existing ones for the same message, so no pinned corpus path or order moves.
- Make `sameResult` (`correlated_prepare.go:17-19`) compare `StateFields` (ordered, as Lean's list equality is), matching Lean's `Result` equality.
- Focused tests: one invalid entry per list (category and path pinned); an output row whose `state_fields` differ from every authorized transition is rejected.
- Confirm the corpus is unchanged: Lean produces consistent fields, so no `expected.json` may move. A committed fixture that fails a new check is a real Go/Lean divergence: stop and report it rather than loosening the check.

## Investigation targets
**Required:**
- `common/testing/testpilot/internal/verification/correlated_prepare.go:1-30,85-175`
- `model/Testpilot/Correlated.lean:95-115,280-290,370-376` — Lean decode and `Result`
- `model/Shared/SemanticData.lean:10-30` — `StateValue`
**Optional:**
- `model/Umpire/Case/Correlated.lean:320-345` — producer and decode agreement proof

## Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/internal/verification/...
make umpire-check-case-runtime-conformance
make lint-code-fast
```

## Acceptance
- [ ] The E1 decision (read) and its evidence are recorded in the receipt.
- [ ] Admission rejects an invalid entry in each of `state_fields`, `prior_fields` and `initial_state_fields`; tests pin category and path.
- [ ] Output-row equality compares `state_fields`; a test pins the rejection.
- [ ] `make umpire-check-case-runtime-conformance` shows no fixture or `expected.json` diff.
- [ ] `go test -race -tags test_dep ./common/testing/testpilot/...` and `make lint-code-fast` pass.


## Done summary
E1 decision: read (recommended default; no owner override). Lean decodes prior_state+prior_fields, state+state_fields and initial_state+initial_state_fields as StateValue (model/Testpilot/Correlated.lean checkedState at 103-106, result at 108-112, transitions at 284, initial at 374; model/Shared/SemanticData.lean StateValue at 19) and its monitor matches transitions on that value, so Go reads the fields; removal would need a Lean StateValue redesign and its decode agreement proof.

Correlated admission now validates every initial_state_fields, prior_fields and state_fields entry as a model value (initial fields under "invalid correlated projection binding", transition fields after the fact check under "invalid correlated transition value", so no existing rejection moves), and sameResult compares StateFields in order as Lean's Result does. Tests: TestCorrelatedPrepareRejectsInvalidStateFields (one case per list, category and path pinned) and TestCorrelatedPrepareComparesOutputStateFields (absent, reordered and different output fields rejected; equal admitted) in correlated_prepare_test.go. Corpus unchanged. Note: sameResult is also the runtime monitor's output-row match (correlated.go:455), which now compares fields too, consistent with Lean; the rest of the monitor half stays with fn-94.3. Commit range ea93a2bc8b..HEAD also contains fn-94.2's concurrent commits; only 0da39564cc is this task's.

stage: impl-review - ran codex fan-out (3 draws SHIP, rid eaed2fd0b9e54645a38f5f2837123577)
## Evidence
- Commits: 0da39564cc3a679732402643e152fa6ecaf8ac2f
- Tests: go test -race -tags test_dep ./common/testing/testpilot/... (rc=0, 11 packages ok), make umpire-check-case-runtime-conformance (rc=0, no fixture or expected.json diff), make lint-code-fast (rc=0, 0 issues), baseline: green (go test -race -tags test_dep ./common/testing/testpilot/internal/verification/... and make umpire-check-case-runtime-conformance pre-edit)
- PRs: