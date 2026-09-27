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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
