---
satisfies: [R3, R5]
---
# fn-64-umpire-case-runtime.3 Implement deterministic Contract evaluation

## Description
Create the generic verification module and per-Run Evaluator (R3), independent of Temporal and
property-specific operators. Keep live monitoring and offline evaluation on one prepared transition
implementation and execution's MonitorFactory contract.

**Size:** M
**Files:** `tools/umpire/verification/**`, verification package documentation and focused tests
**Touches:** [tools/umpire/verification/**]

## Approach
- Prepare finite ordered transition machines, typed predicates, event-kind indexes, terminal
  states, supporting-event references, and explicit safety/liveness horizons.
- Implement the execution MonitorFactory contract so every Run receives fresh state.
- Evaluate time horizons only from recorded Executor monotonic coordinates on explicit timeout and
  Run-closure events; never arm an invisible timer or use target timestamps.
- Preserve the useful live/offline distinction from legacy Run Evaluation while deleting all
  scenario/property-specific semantics at cutover.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/portableevaluation/property.go:14-247` — property-specific behavior to eliminate
- `tools/umpire/portableevaluation/evaluator.go:16-35` — current evaluation entrypoint
- `tools/umpire/runevaluation/run_evaluation.go:71` — legacy offline evaluation boundary
- `tools/umpire/runevaluation/result.go` — existing verdict/result representation
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md:224-251` — approved monitor-machine semantics

**Optional** (reference as needed):
- `tools/umpire/runevaluation/README.md` — live/offline documentation to replace, not preserve as API

## Key context
Transition order is semantic. Indexing may reduce candidates by event kind but must preserve the
same first matching transition, bad prefix, and closure result in both modes.

## Acceptance
- [ ] Safety tests prove exact first-violation prefix and synchronous Stop; liveness tests prove
  witnesses and no failure before a recorded horizon event.
- [ ] Live and offline evaluation yield byte-equivalent transition traces, supporting-event
  references, and Verdicts for completed, stopped, and incomplete Runs, including time horizons.
- [ ] Contracts cannot access Slots, opaque capabilities, raw payloads, or undeclared fields.
- [ ] Malformed states/transitions/predicates/horizons and work overflow fail preparation; Monitor
  error/timeout yields incomplete/inconclusive unless violation is already proved.
- [ ] Event-kind indexing stays within declared per-event work bounds without changing ordered
  semantics.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/verification/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
