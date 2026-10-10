---
satisfies: [R9]
---
# fn-123-declare-faults-as-the-environments.6 Choice-level fault performance and lowering refusals for model-only faults and unperformed choices

## Description
Choice-level fault performance (R9). A performance may name the fn-120.1 choices it performs with `choosing(…)`, and the others stay model-only for that realization. Performing a `modelOnly` fault is refused. Lowering refuses a witness that takes a model-only fault or an unperformed choice, at its step, with the reason and position, and the manifest standing names it. `LostStartAnswer`'s realization performs only `committedThenLost`.

**Size:** M
**Files:** `model/framework/realize/Scripts.scala` (`choosing` on a performance), `model/irgen/Realizations.scala` (lift the choices; refuse a model-only fault), `proto/internal/temporal/server/api/umpire/v1/script.proto` (`Performance` choices) + regenerated Go, `tools/umpire/lower/lower.go` (+ tests), `tools/umpire/lower/generated.go` (standing), `model/temporal/features/activity/standalone/system/Realization.scala`, regenerated `model/ir/**`, `model/cases/**`
**Touches:** [model/framework/realize/**, model/irgen/Realizations.scala, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/**, tools/umpire/lower/**, model/temporal/features/activity/standalone/system/Realization.scala, model/ir/**, model/cases/**]
**Order:** After task 5's focused scratch budget proof against fn-149.5's grouped baseline (production race IR publishes at fn-123.8). Write against fn-133.6's typed `perform` and fn-133.3's lower-case fault instruction; re-read `Realization.scala` first.

### Approach
- Today `performed()` (`lower.go:812-838`) already refuses any non-system step no script performs, so a model-only fault on a witness already fails with a generic message. Keep that check. Add the reason and Scala position of the `modelOnly` declaration to its message, and add the choice-level refusal: a witness step whose recorded choice the performance does not name.
- Find which witness field carries the chosen arm per step (fn-120.1 records choice names, `Construct.choice` at `expression.proto:68-75`). If the witness does not carry it, record that in the done summary and derive it from the step's result the way lint's `untaken-choice` does (`lint.go:39`).
- Standing: name the reason in the manifest standing (`lower.go:27-37`, written by `generated.go:49, :113`) without adding a new `Standing` constant unless the existing ones cannot carry it. Lowered Cases and their Known Gaps stay byte-identical to fn-149.5's final grouped baseline, with its authorized composed fn-140/fn-155 mapping.
- Realization: `perform(foundations.taskqueue.fault.ackLoss.choosing(committedThenLost) -> …)` at the lost-admission realization (`Realization.scala:1124-1146`).
- Keep realization-only metadata out of transition-table reads (memory `transition-table-reads-must-omit-2026-10-06`), and keep a lowering refusal as the deciding diagnostic (memory `replay-rejection-lost-to-2026-09-27`).

### Investigation targets
**Required** (read before coding):
- `tools/umpire/lower/lower.go:27-37, 700-720, 812-838` - standing, performance lookup, `performed()`
- `model/framework/realize/Scripts.scala:30-35` - `perform`
- `model/temporal/features/activity/standalone/system/Realization.scala:1124-1146` - the lost-admission command and realization
- `proto/internal/temporal/server/api/umpire/v1/script.proto:64` - `Performance`

**Optional** (reference as needed):
- `tools/umpire/lower/generated.go:49, 113` - manifest standing
- `model/temporal/realize/Realize.scala:61-74` - `FaultKind`, `Fault`

## Acceptance
- [ ] `choosing(…)` lifts onto the performance. The lost-admission realization performs only `committedThenLost`.
- [ ] Performing a `modelOnly` fault is refused at the performance, with a fixture.
- [ ] Lowering refuses a witness through a model-only fault and one through an unperformed choice, each at its step with the declared reason and position (tests in `tools/umpire/lower`).
- [ ] The manifest standing names the reason. Lowered Cases and their Known Gaps are byte-identical to fn-149.5's final grouped baseline.
- [ ] `make umpire-check-model`, `make umpire-check-cases` and `go test -tags test_dep ./tools/umpire/lower/...` pass. Focused scratch lowering/refusal proofs run here; link complete integrated gate evidence from fn-123.8, without production publication or early task closure.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
