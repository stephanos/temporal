---
satisfies: [R2]
---
# fn-94-simplify-the-testpilot-go-runtime.4 Remove dead and test-only code in the runtime core

## Description
Lane A for the facade, `contract`, `ir`, `execution` and `verification`: delete the unreachable and test-only declarations the spec lists, re-point their tests at production paths, and record the lane H decision. Split from fn-94.5 (Drivers) because the two file sets are disjoint and can run in parallel.

**Owner decision, lane H (recommended default: keep the wrapper structs).** Record it in the receipt; only their uncalled methods go here.

**Size:** M
**Files:** `common/testing/testpilot/driver.go`, `contract.go`, `prepared_case.go`, `internal/execution/program.go`, `internal/execution/prepare.go` (the `maximumActivations` computation), `internal/execution/dataflow.go` (the `environmentBindingID` assignment), `internal/execution/contracts.go`, `internal/verification/prepare.go` (`Snapshot`/`ProgramView`), `internal/ir/path.go`, their tests, `internal/execution/README.md`
**Touches:** [common/testing/testpilot/driver.go, common/testing/testpilot/contract.go, common/testing/testpilot/prepared_case.go, common/testing/testpilot/internal/execution/program.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/internal/execution/dataflow.go, common/testing/testpilot/internal/execution/contracts.go, common/testing/testpilot/internal/execution/*_test.go, common/testing/testpilot/internal/execution/README.md, common/testing/testpilot/internal/verification/prepare.go, common/testing/testpilot/internal/verification/prepare_test.go, common/testing/testpilot/internal/ir/path.go, common/testing/testpilot/internal/ir/path_test.go, common/testing/testpilot/*_test.go]

### Approach
- Delete: `PreparedProgram.PolicyIdentity` (`program.go:112`), `InstructionPlan.Assignments`/`AssignmentPlan`/`Input` (`:254,304-311`), `ResponseReads`/`ResponseReadPlan` (`:258,313-323`), `ProgramView.MaximumActivations` and its computation (`program.go:62,76`, `prepare.go:~745-758`), `assignment.environmentBindingID` (`program.go:174`, `dataflow.go:656-673`), `PreparedContract.Snapshot`/`ProgramView` (`verification/prepare.go:104-105`), `Path.CheckFanout` (`ir/path.go:51-52`).
- Facade: delete `InstructionPlan.Guard`, `OutcomeType`, `Dependencies`, the facade `Expression`/`Evaluate` (`driver.go:19-28,95-103`) and the execution methods only they called. Then remove each of the six aliases (`contract.go:11-62`) only if no remaining facade signature names it.
- `prepared_case.go:41-46`: store the prepared contract beside the `MonitorFactory` so the anonymous `Evaluate` assertion and its impossible error branch go; fake factories stay unchanged.
- Re-point tests that exercised the removed declarations at the production path, or delete them where they tested only the removed code.
- Update `internal/execution/README.md:166-174` (`OutcomeType`). Do not edit the root README or `internal/verification/README.md` here (fn-89.6 owns them now; fn-94.17 sweeps them).
- Behavior pin: `make umpire-check-case-runtime-conformance` with no diff plus the fn-94.2 goldens unchanged.

### Investigation targets
**Required:**
- `common/testing/testpilot/driver.go` — facade wrappers (140 lines)
- `common/testing/testpilot/internal/execution/program.go:50-330`
- `common/testing/testpilot/prepared_case.go:20-60`
- `common/testing/testpilot/contract.go:1-70`
**Optional:**
- `common/testing/testpilot/internal/execution/prepare_test.go:300-320,420-430,745-815`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/...
make umpire-check-case-runtime-conformance
make lint-code-fast
```

## Acceptance
- [ ] Every core declaration lane A lists is gone or recorded as kept with its production caller.
- [ ] No removed alias leaves a `contract.*` type in a facade signature.
- [ ] The lane H decision (keep) is recorded in the receipt.
- [ ] Corpus and fn-94.2 goldens unchanged; tests and `make lint-code-fast` pass.


## Done summary
Lane H decision (D2): keep the facade wrapper structs `PreparedProgram`, `EntrypointPlan` and `InstructionPlan` (recommended default); only their uncalled methods were removed.

Removed every lane A core declaration: `PreparedProgram.PolicyIdentity`, `ProgramView.MaximumActivations` and its stored result, `InstructionPlan` `Assignments`/`ResponseReads`/`Input`/`Guard`/`Dependencies`/`OutcomeType` with `AssignmentPlan`/`ResponseReadPlan`, `assignment.environmentBindingID`, `PreparedContract.Snapshot`/`ProgramView`, `Path.CheckFanout`, and the facade `InstructionPlan.Guard`/`OutcomeType`/`Dependencies` plus `Expression`/`Expression.Evaluate`. `PreparedCase` now stores its prepared Contract beside the `MonitorFactory`, so `Evaluate` loses the anonymous-interface assertion and its impossible error branch; fake factories are unchanged. The activation ceiling check in `bindReservations` stays, because it still rejects over-ceiling Cases; only the dead assignment of its total went.

Kept, with reason: all six aliases (`ReferenceKind`, `OutcomeSnapshot`, `ReservationTopology`, `ReservationRoute`, `SlotReference`, `InjectFault`). Each is still reachable from a kept facade signature: `ValueReference.Kind` in `EvaluateInput`, `ValidateOutcome` returns `*OutcomeSnapshot`, `Reservations` returns `[]ReservationTopology`, `ReservationCarrierPlan.Routes` from `ReservationCarrier`, and the `Opcode`/`ReferenceKind` constant sets. Removing any of them would expose a `contract.*` type in the public API.

Dead-code evidence: a grep over common/, tests/, tools/, service/ and cmd/ finds zero references to every removed name (tests included). `deadcode -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/...` reports nothing in the facade, `contract`, `ir`, `execution` or `verification`; its remaining reports are all fn-94.5's lane (worker/activation/tests). None of the removed methods showed up in `deadcode`, which treats exported methods as live, so grep is the evidence for them.

Tests re-pointed at production paths: the activation-ceiling cases now admit at the ceiling and reject one below with the admission error; `Path` fan-out goes through `Path.Read` with 127/128/129 items; immutability goes through `Source()` clones; derived outcome types come from `node.outcomes` (what `ValidateOutcome` checks); dependencies come from `node.dependencies`. The `execution/README.md` `OutcomeType` text is updated.

Scope note: the `PreparedCase` struct lives in `prepare.go` (facade), not `prepared_case.go`, so that one-line field addition landed there. The verification build was run in a HEAD snapshot with only this diff applied, because fn-94.5's in-progress edits leave `temporal/worker` unbuildable in the shared checkout. Measurement: production 19289, tests 20889, tests/testpilot 2399, proto 1414 lines. This commit's delta is -81 production and +14 test Go lines.

stage: impl-review - ran [fan-out rid 892d7258b69542c9a52c6f0c13a86aa3, 3/3 draws SHIP, review base 0db947dac6]
## Evidence
- Commits: 1d94ab30ff660623d77432c6c76c447935272bc5
- Tests: baseline: green (go test -race -tags test_dep ./common/testing/testpilot/... pre-edit), go test -race -tags test_dep ./common/testing/testpilot/... (HEAD snapshot + this diff; 11 packages ok), make umpire-check-case-runtime-conformance equivalent in snapshot: both generator modes, diff -ru clean on both testdata trees, generator tests and TestCaseRuntimePublicFacadeConformance ok (Lean binaries reused from model/.lake, model unchanged), make lint-code LINT_CODE_TARGETS=<facade, execution, ir, verification> GOLANGCI_LINT_BASE_REV=735caa401c: 0 issues, deadcode -test -tags 'test_dep integration' ./common/testing/testpilot/... ./tests/... ./tools/...: no core-package reports
- PRs: