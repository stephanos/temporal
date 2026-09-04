---
satisfies: [R3, R5]
---
# fn-64-umpire-case-runtime.3 Implement deterministic Contract evaluation

## Description
Extend task 12's generic verification module with the per-Run Evaluator (R3), independent of Temporal and
property-specific operators. Keep live monitoring and offline evaluation on one prepared transition
implementation and the internal execution MonitorFactory contract.

**Size:** M
**Files:** `tools/umpire/verification/**`, verification package documentation and focused tests
**Touches:** [tools/umpire/verification/**]

## Approach
- Consume task 12's admitted finite ordered transition machines, typed predicates, event-kind indexes, terminal
  states, supporting-event references, typed single-assignment scalar captures, and explicit
  safety/liveness horizons. Predicates read pre-transition captures; assignments and state changes
  commit atomically. Captures retain the source event and remain local to their rule and Run.
- Implement the internal execution MonitorFactory contract so every Run receives fresh state; it is
  the only production Monitor factory. Task 13 binds it through `PrepareCase`; this task proves
  factory construction using task 11's immutable Program Observation/bounds view.
- Before transitions on every recorded event, expire pending liveness when its elapsed coordinate
  is greater than or equal to the Run-relative deadline. Only earlier witnesses qualify; never arm
  an invisible timer or use target timestamps. Completed closure before a pending deadline is
  inconclusive. Once execution/evaluation is incomplete, time alone cannot establish a new absence
  violation; previously proved violations remain.
- Preserve the useful live/offline distinction from legacy Run Evaluation while deleting all
  scenario/property-specific semantics at cutover.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/portableevaluation/property.go:14-247` — property-specific behavior to eliminate
- `tools/umpire/portableevaluation/evaluator.go:16-35` — current evaluation entrypoint
- `tools/umpire/runevaluation/run_evaluation.go:71` — legacy offline evaluation boundary
- `tools/umpire/runevaluation/result.go` — existing verdict/result representation
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md` — Contract IR and approved monitor-machine semantics

**Optional** (reference as needed):
- `tools/umpire/runevaluation/README.md` — live/offline documentation to replace, not preserve as API

## Key context
Transition order is semantic. Indexing may reduce candidates by event kind but must preserve the
same first matching transition, bad prefix, and closure result in both modes.

## Acceptance
- [ ] Safety tests prove exact first-violation prefix and synchronous Stop. Liveness tests cover
  witnesses before/at/after deadline, including a 5s deadline, 6s witness and 7s closure; expiry is
  checked before transitions on observation, timeout and closure events.
- [ ] Completed early closure is inconclusive with completed disposition; incomplete observation
  at/after the deadline cannot manufacture an absence violation. Live/offline results agree.
- [ ] Capture tests correlate a prior scheduled event ID with matching and mismatched later IDs,
  retain source-event references, isolate rules/Runs, and reject unguarded missing reads, repeated
  writes, wrong types, and count/byte/work overflow at the appropriate phase.
- [ ] Live and offline evaluation yield byte-equivalent transition traces, supporting-event
  references, and Verdicts for completed, stopped, and incomplete Runs, including time horizons.
- [ ] Contracts cannot access Slots, opaque capabilities, raw payloads, or undeclared fields.
- [ ] Malformed states/transitions/predicates/captures/horizons and static work overflow fail
  preparation; Monitor error/timeout yields incomplete/inconclusive unless violation is already proved.
- [ ] Event-kind indexing stays within declared per-event work bounds without changing ordered
  semantics.
- [ ] No public API permits replacing the prepared Case's Contract Monitor.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/verification/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
