---
satisfies: [R6]
---
# fn-22-deterministic-replay-semantic.2 Prove the negative-control Case before any reduction

## Description
Author `Temporal.Feature.Nexus.Control`: a labeled control machine over the caller Model's entities and actions whose one step contradicts the platform (a handler failure reply completes the operation as `succeeded`), one Query over it, a functional set binding the same parties as `nexusCallerTests`, and a `case` block realizing it as `nexusCallerCases.realization`, registered as `temporal.case.nexusCallerControl.<query>` with a fixture. Prove it live: run the fixture twice against the test cluster under one switch value, both Runs violated with one `ViolationKey`, the Verdict reproduced offline through `PreparedCase.Evaluate`. Define the `EvidenceCore` (the violated rules' supporting events and their causal sources, by sequence) and prove it omits the scheduled-event read, which supports the schedule clause and not the violated terminal clause, while the Run and Verdict are unchanged. This is the early proof point: if the control cannot be produced under the caller Realization without scenario-specific Go, stop and revise the Producer boundary.

### Approach
- The control Model imports the caller Model's declarations and declares only the machine, the Query and the set; its docstring and COVERAGE.md entry say it is a negative control that enters no functional, canary or exploratory set and no regression view.
- The live test follows `tests/testpilot_nexus_caller_case_test.go` and asserts through `tools/umpire/replay`.

### Quick commands
`cd model && lake build && cd .. && make umpire-check-goldens && go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotNexusControl'`

**Size:** L
**Files:** `model/Temporal/Feature/Nexus/Control/Model.lean`, `model/Temporal/Feature/Nexus/Control/Tests.lean`, `model/Temporal/Feature/Nexus/Control/Fixtures/**`, `tests/testpilot_nexus_control_case_test.go`, `tools/umpire/replay/core.go`, `tools/umpire/replay/core_test.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; see the spec's **Re-plan** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] The control Case uses only the caller Realization, public Program instructions and Contract rules; no Go path is specific to it.
- [ ] Two live Runs are violated with the same key and isolated from each other; the offline replay reproduces the Verdict.
- [ ] The evidence core omits the labeled non-responsible read without rewriting events, Run, Verdict or Contract.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
