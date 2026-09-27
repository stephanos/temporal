---
satisfies: [R5]
---
# fn-90-resolve-the-intermittent-live-testpilot.8 Await every instance's scheduled event in the Nexus realization

## Description
Fix the pair Case's scheduled-evidence race fn-90.7 reproduced (1/200 on `TestTestpilotNexusPairCase` at be465e5966, signature `b467973f771b`; Run record `/private/tmp/fn-90-after/loop2.runs/20260927T054210-184/`). The Nexus realization's controller emits one fixed `await-scheduled` step for the whole Case; it returns as soon as any scheduled event exists and emits evidence only for the scheduled events it saw. The workflow schedules operation 2 only after operation 1 starts, so a read in that gap never emits operation 2's scheduled evidence, and its started event is then rejected by the monitor as `unauthorized operation transition` (Run INCOMPLETE, Verdict INCONCLUSIVE).

**Size:** M
**Files:** `model/Temporal/Case/Realization/Nexus.lean` (the await step and its use), regenerated Case fixtures (the pair fixture changes; single-instance caller fixtures stay byte-identical), `tests/testpilot_*` only if a test pins the step list
**Touches:** [model/Temporal/Case/Realization/Nexus.lean, model/Temporal/Feature/Nexus/**/Fixtures/**, tests/testdata/**, tests/testpilot_nexus_pair_test.go]

### Approach
- Make the await wait per instance (one await per operation instance, keyed by its scheduled identity) or until every instance's scheduled event is present, whichever keeps single-instance Cases byte-identical and keeps the Case the only authority on timeouts.
- Regenerate fixtures through the existing generator targets; list every changed fixture and why.
- Add a deterministic unit or conformance test that reproduces the gap offline (history with op1 scheduled+started before op2 scheduled, controller read at the gap) and fails on the old step.
- Coordinate: fn-88 pins Nexus pair state counts (its .9) and fn-89 regenerates the pair fixture (its .4); record the new fixture bytes in the receipt so both rebase onto them. Land on the current toolchain before fn-88.12 moves it, or on 4.32.0 if .12 has landed.
- Short live confirmation: `make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=20`; the full 200-iteration loop is fn-90.7's rerun.

### Key context
- The server-written started events on the completion-before-start path carried local time labelled UTC (`2026-09-26T22:44:46Z` beside `05:44:46Z` events); out of scope here, recorded as a follow-up in fn-90.7.

## Acceptance
- [ ] The pair Case emits scheduled evidence for every instance regardless of when the controller reads, proved by an offline test that fails on the old step.
- [ ] Single-instance fixtures are byte-identical; every changed fixture is listed with its reason.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
