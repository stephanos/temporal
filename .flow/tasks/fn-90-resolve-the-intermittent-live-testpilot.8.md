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
The pair Case now reads its scheduled events after the workflow closes. On a Case over several instances, `controllerNodes` (model/Temporal/Case/Realization/Nexus.lean) moves the controller's `await-scheduled` read to just before the `history` read. Placed right after start-workflow, the read could end its poll holding only operation 1's scheduled event, because the worker schedules operation 2 only after operation 1 has started. Operation 2's started event then reached the monitor as an `unauthorized operation transition`. Single-instance Cases keep their node order.

Baseline: green. `go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/...` and `go vet -tags 'test_dep integration' ./tests` passed before any edit (CC=/usr/bin/clang).

Changed fixtures (via `make umpire-gen-case-runtime-conformance`):
- `tests/testcore/testpilot/testdata/nexusPairTests-bothComplete-case.json` is the only one. Its controller `await-scheduled` block moved from after `start-workflow` to after `await-close`, and no fingerprint changed. New bytes: 42462 bytes, sha256 `9f52ff1e74398d20306f7a09d9f4b7e9ad76c29986fb17ccfd6bd03efb7501f4`. The new controller order is start-workflow, await-completion-authority-1, complete-nexus-operation-1, await-completion-authority-2, complete-nexus-operation-2, await-close, await-scheduled, history. fn-88.9 (pair state counts) and fn-89.4 (pair fixture regeneration) should rebase onto this commit.
- Every single-instance caller fixture, the conformance corpus and the canary fixture are byte-identical (`make canary-check-case` green).

Tests:
- Offline: `TestNexusPairCaseReadsEveryInstanceScheduledEvent` in tests/testcore/testpilot/nexus_pair_artifact_test.go. A scripted Driver answers every read with the gap history (operation 1 scheduled and started, operation 2 absent) until the close read resolves. Against the old fixture it failed with Run INCOMPLETE and `malformed at contract: unauthorized operation transition`, the live signature `b467973f771b`. Against the new fixture it passes with both capture rules SATISFIED.
- Lean pins: two `#guard`s on `controllerNodes` in Nexus.lean, and the pair controller order in Pair/Tests.lean.
- Live: `make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=20` gave 0/20 at 860ecde37d (95% CI 0.00%-16.84%), with load 2.62 and no other go test process. The full 200-iteration loop remains fn-90.7's.
- Gates: `make umpire-check-live-tests` passed at 860ecde37d. `make lint-model` passed on its second run; the first stopped on missing .olean files mid-build, which the log records as inconclusive rather than a lint finding.

Follow-up, not built: on a multi-instance path with a retryable handler error, `pending-attempts` would still run before the moved scheduled read. No such Case exists today. The single fixed pending-attempts poll is not per instance either.

Commit range note: a164116e87..HEAD also contains 4b1d55a7a6, which is fn-91.2's commit, not this task's.

stage: impl-review - ran [claude:opus:high, SHIP first pass; FYI findings: pending-attempts ordering on future multi-instance retry paths, duplicated PollRPC in the test double]
## Evidence
- Commits: 860ecde37d3238ffbbbf9d25c9fa07f324a1452c
- Tests: go test -tags test_dep -count=1 -run 'TestNexusPair|TestLean' ./tests/testcore/testpilot/ (TestNexusPairCaseReadsEveryInstanceScheduledEvent red on the old fixture with 'unauthorized operation transition', green after), make umpire-gen-case-runtime-conformance (only nexusPairTests-bothComplete-case.json changed), make canary-check-case, lake build Temporal.Feature.Nexus.Pair.Tests Temporal.Feature.Nexus.Caller.Tests, make lint-model (first run INCONCLUSIVE: missing .olean mid-build; rerun green), go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/..., go vet -tags 'test_dep integration' ./tests ./tests/testcore/testpilot, make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=20 MODE=process: 0/20 at 860ecde37d, load 2.62, make umpire-check-live-tests (green at 860ecde37d, 25 TestTestpilot tests passed; gate receipt refused: worktree dirty from other sessions)
- PRs: