---
satisfies: [R3]
---
# fn-123-declare-faults-as-the-environments.3 Go derives the crash row from durability; SEMANTICS Faults section (early proof)

## Description
The early proof. Go derives the crash row from the durability record (R3), as channel rows are derived today, and `SEMANTICS.md` gains its Faults section. The proof is a Go test over a fixture copy of the queue: the derived table equals `crashDetail`'s table cell by cell. If it does not, stop and list the rows for the owner before task 4.

**Size:** M
**Files:** `tools/umpire/interp/machine.go`, new `tools/umpire/interp/faults.go` (+ `_test.go`), `tools/umpire/ir/validate.go` (admission), `tools/umpire/internal/engine/table.go` (if rows are listed there), `model/SEMANTICS.md` (Faults section beside Channels; Admission list), a fixture IR under the interp or engine testdata
**Touches:** [tools/umpire/interp/**, tools/umpire/ir/**, tools/umpire/internal/engine/**, model/SEMANTICS.md, model/irgen/testdata/**]
**Order:** After task 2. No regeneration of `model/ir`, since no Model changes.

### Approach
- Branch in `steps` the way channel actions go to `transfer` (`interp/machine.go:518-520` → `interp/channels.go:143`): a derived-crash binding computes its one result by mapping each field through its classification, with the binding's outcome and facts and no `because`.
- Enabled in every catalog state, end states included, as `crashDetail`'s `always` guard is. Bound the extra rows before enumerating them (memory `finite-ir-admission-must-count-work-2026-09-30`).
- Admission: a machine whose crash binding is derived and has no durability record is an error at the binding (R3). A classification value outside the field's catalog is an error naming the field.
- Proof test: build a fixture IR with `TaskQueueSystem`'s state type (`Custody`, `polled`, `delivered`) classified as the spec's API Contracts shows, and a second copy whose crash is the authored `crashDetail`. Compare every row (state, outcome, facts, `because`, next state) over the whole catalog. Make it a table test so task 4 can reuse it against the real Model.
- `SEMANTICS.md`: a `## Faults` section after `## Channels` (:250) stating kinds, durability forms, the derived row and the budget rules of Part C; add the new refusals to `## Admission` (:690).

### Investigation targets
**Required** (read before coding):
- `tools/umpire/interp/machine.go:510-530` - where channel classes branch
- `tools/umpire/interp/channels.go:143` - `transfer`, the derived-row pattern
- `tools/umpire/ir/validate.go:672-760` - machine, holds and step admission
- `model/SEMANTICS.md:250-290, 690-765` - Channels and Admission

**Optional** (reference as needed):
- `tools/umpire/internal/engine/table.go:160, 369-430` - `RowsFrom`, `checkSpec`, `checkRows`
- `model/temporal/shared/taskqueue/system/System.scala:110-120, 150-153` - `crashDetail` and its rule

## Acceptance
- [ ] The proof test shows the derived crash table equal to `crashDetail`'s over every catalog state: outcome `internal`, fact `crashed`, no `because`, the same next state. Or the done summary lists each row that differs, and the task stops for the owner.
- [ ] Admission refuses a derived crash with no durability record, at the binding, with a test.
- [ ] `SEMANTICS.md` has a Faults section and the new Admission entries.
- [ ] `go test -tags test_dep ./tools/umpire/interp/... ./tools/umpire/ir/... ./tools/umpire/internal/engine/...` and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
