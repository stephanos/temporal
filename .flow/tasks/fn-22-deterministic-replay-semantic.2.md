---
satisfies: [R6]
---
# fn-22-deterministic-replay-semantic.2 Prove the negative-control Case before any reduction

## Description
Author `Temporal.Feature.Nexus.Control`, imported by `Temporal.Feature.Nexus` so `umpire-case` renders it: a labeled control machine over the caller Model's entities and actions whose `handlerReply` step keeps the platform's real rows and adds one row the platform never takes (a non-retryable handler error completing the operation as `succeeded` with `nexusOperationCompleted`), one Query whose exact trace selects that row and whose Property names `succeeded`, a functional set binding the same parties as `nexusCallerTests`, and a `case` block realizing it as `nexusCallerCases.realization`, registered as `temporal.case.nexusCallerControl.<query>`. Its fixture is written by `umpire-gen-case-runtime-conformance --mode functional` to `tests/testcore/testpilot/testdata/nexusCallerControl-<query>-case.json`, checked by `umpire-check-case-runtime-conformance` and listed by `fixture_table_test.go`. Prove it live: run the fixture twice against the test cluster under one switch value; both Runs in the admissible violated form with one `ViolationKey`; the Verdict reproduced offline through `PreparedCase.Evaluate`. Define the `EvidenceCore` (the events the violated rules' supporting sequences name, by sequence) and prove it omits the Run's scaffolding instruction events, named by instruction id, while the Run and Verdict are unchanged. This is the early proof point: stop and revise if the control cannot be produced under the caller Realization without scenario-specific Go, or if its Runs come back inconclusive (an unauthorized transition or an observe failure) rather than violated.

### Approach
- The control Model imports the caller Model's declarations and declares only the machine, the Query and the set; its docstring and the COVERAGE.md entry say it is a negative control that enters no functional, canary or exploratory set of the caller Model and no regression view.
- The live test follows `tests/testpilot_nexus_caller_case_test.go` and asserts through `tools/umpire/replay`; its recorded Run is kept as the test data task .1's key tests read.

### Quick commands
`cd model && lake build && cd .. && make umpire-gen-case-runtime-conformance umpire-check-case-runtime-conformance && go test -count=1 -tags test_dep ./tests/testcore/testpilot/ && go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotNexusControl'`

**Size:** L
**Files:** `model/Temporal/Feature/Nexus/Control/Model.lean`, `model/Temporal/Feature/Nexus/Control/Tests.lean`, `model/Temporal/Feature/Nexus.lean`, `model/Temporal/Feature/Nexus/Caller/COVERAGE.md`, `tests/testcore/testpilot/testdata/nexusCallerControl-*-case.json`, `tests/testcore/testpilot/fixture_table_test.go`, `tests/testpilot_nexus_control_case_test.go`, `tools/umpire/replay/core.go`, `tools/umpire/replay/core_test.go`
**Touches:** `model/Temporal/Feature/Nexus/**`, `tests/testcore/testpilot/testdata/**`, `tests/testcore/testpilot/fixture_table_test.go`, `tests/testpilot_nexus_control_case_test.go`, `tools/umpire/replay/core*.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review round one; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] The control Case uses only the caller Realization, public Program instructions and Contract rules; no Go path is specific to it; the control Model keeps every real row authorized.
- [ ] Two live Runs are in the admissible violated form with the same key and isolated from each other; the offline replay reproduces the Verdict.
- [ ] The evidence core omits the labeled scaffolding events without rewriting events, Run, Verdict or Contract.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
