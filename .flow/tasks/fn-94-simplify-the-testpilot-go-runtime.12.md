---
satisfies: [R5]
---
# fn-94-simplify-the-testpilot-go-runtime.12 Execution and verification internals: guarded input, drains, aliases

## Description
The rest of lane D1: one guarded-input evaluation, one completion drain, one close helper, one correlated-literal switch, `ir.BindExpression` folded, and the aliases and narrowed interfaces.

**Size:** M
**Files:** `common/testing/testpilot/internal/execution/{request,scheduler,runtime,program,contracts}.go`, `internal/ir/expression.go`, `internal/verification/{correlated_prepare,evaluator}.go`, facade `prepared_case.go`, `driver.go`, `contract.go`, `profile.go`, `internal/execution/README.md`, `tools/umpire/replay/subject_test.go` (keyed `RuleViolation` literals, only if the field rename requires it)
**Touches:** [common/testing/testpilot/internal/execution/request.go, common/testing/testpilot/internal/execution/scheduler.go, common/testing/testpilot/internal/execution/runtime.go, common/testing/testpilot/internal/execution/program.go, common/testing/testpilot/internal/execution/contracts.go, common/testing/testpilot/internal/execution/*_test.go, common/testing/testpilot/internal/execution/README.md, common/testing/testpilot/internal/ir/expression.go, common/testing/testpilot/internal/ir/expression_test.go, common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/internal/verification/evaluator.go, common/testing/testpilot/prepared_case.go, common/testing/testpilot/driver.go, common/testing/testpilot/contract.go, common/testing/testpilot/profile.go, tools/umpire/replay/subject_test.go]
**Depends on (cross-spec):** fn-89-one-contract-rule-per-entity.5 (`verification`)

### Approach
- `node.evaluateGuarded` replaces `request.go:20-43`, `scheduler.prepareInput` (`:618`) and `InstructionPlan.EvaluateInput` (`program.go:350-383`) with the same work accounting; existing work-accounting tests pin it.
- `takeCompletion` plus one non-blocking drain replace the five `select`s on `s.completions` (`scheduler.go:136,194,319,364,384`); `execute`/`executeCleanup` stay separate (EVD-14).
- One close helper for the two identical blocks in `runtime.go:38-50`; the final close at `:75-80` stays.
- `validCorrelatedLiteral`/`correlatedLiteralKind` (`correlated_prepare.go:294,324`) → one switch.
- `ir.BindExpression` (`expression.go:184-198`) → `bindConditionedExpression(nil, …)`, keeping fn-89's `instanceValueReads`.
- `resolvedRole` (`program.go:98-105`) → `contract.PreparedRole`; rename `verification.Violation.Kind` to `CorrelatedKind` and make the facade `RuleViolation` an alias (check the retired-vocabulary gate before renaming); `execution.Run` takes the one method it calls and `driverAdapter`'s `Identity`/`Validate` go (`driver.go:130-136`); forwarding functions (`contract.go:67`, `profile.go:44,50`) become `var` aliases.
- Run with `-race`; keep EVD-14/EVD-15 tests green.

### Investigation targets
**Required:**
- `common/testing/testpilot/internal/execution/scheduler.go:120-400,610-640`
- `common/testing/testpilot/internal/execution/runtime.go:20-85`
- `common/testing/testpilot/internal/verification/evaluator.go:60-70,570-580`
- `common/testing/testpilot/prepared_case.go:20-35`

### Quick commands
```sh
go test -race -count=3 -tags test_dep ./common/testing/testpilot/...
go test -tags test_dep ./tools/umpire/replay/...
make umpire-check-retired-vocabulary
make lint-code-fast
```

### Carried from fn-94.7 (2026-09-27)
- Each of `execution`, `verification` and the facade still holds one-line `var` aliases (`invalid`, `validID`, `isNil`, `missing`, `invalidAt`) to the new `ir` exports; rename the call sites to the `ir` names and delete the aliases (extend Touches to those files as needed). `ir/expression.go`'s `isNilMessage` can fold into `ir.IsNil` if it is the same check.

### Carried from fn-94.10 (2026-09-27)
- The Program ceiling literal appears three times (server Driver, worker Driver, `execution.hardLimits`); export one shared ceiling (from `ir` or `execution`, wherever both Drivers may import) and use it in all three.
- fn-94.10's review receipt prose is fn-94.11's merged review (a shared scratch file was overwritten); its SHIP verdict came from its own draws. No action beyond noting it.

### Carried from fn-94.11 (2026-09-27)
- `scheduler.acceptEffect`'s `input` parameter is never read (renamed `_`); remove it and adjust `admitDispatch` and its callers.

## Acceptance
- [ ] Each D1 consolidation leaves one definition; `execute`/`executeCleanup` stay apart.
- [ ] `RuleViolation` is an alias of `verification.Violation`; the gate passes.
- [ ] Corpus unchanged; `-race -count=3` tests and lint pass.


## Done summary
Lane D1's remaining consolidations are done, and each now has one definition. `node.evaluateGuarded` serves `activationValues.request`, `scheduler.prepareInput` and `InstructionPlan.EvaluateInput`. `takeCompletion` and `drainCompletions` replace the five completion selects, and `execute`/`executeCleanup` stay separate. One `abandon` helper closes the Session when Run setup fails. `correlatedLiteralKind` is the single correlated literal switch, and `ir.BindExpression` now calls `bindConditionedExpression(nil, …)`. `resolvedRole` is now `contract.PreparedRole`. `verification.Violation.Kind` is renamed `CorrelatedKind`, and the facade `RuleViolation` is an alias of that type. `execution.Driver` holds only `Open`, and `driverAdapter` has lost `Identity`/`Validate`. `EntrypointKindOf`, `InstructionOpcode` and `EnvironmentBindingIDs` are `var` aliases.

Carried items are also done. The `invalid`/`validID`/`isNil`/`missing`/`invalidAt` aliases are deleted from execution, verification, the facade and ir: call sites now use `ir.Invalid`, `ir.ValidID` and `ir.IsNil`, and `isNilMessage` is folded into `ir.IsNil`. `execution.ProgramCeiling()` is the one Program ceiling for Prepare, the server Driver and the worker Driver. `acceptEffect`/`admitDispatch` no longer take the unread input.

Two further changes came up along the way. `slotBridge` is renamed `handleBridge`: the retired-vocabulary gate was red at HEAD because of it. At the conductor's request, the worker uses `delivery.WorkflowBinding` directly, and the `workflowRouteIndex` alias is gone. Goldens and conformance fixtures are unchanged. The rejection-order pin `TestPrepareRejectsTheFirstOfTwoDefects` still passes.

The review base is d546dc3967, the parent of the task commit. fn-94.9 and fn-94.13 receipts and commits landed between the recorded base and this commit. The unittest gate receipt was not written, because other workers had dirty `model/` files in the tree.

stage: impl-review - ran [codex fan-out rid c0f833ffe878435896cb28ef97a450a9, 3 draws SHIP]
## Evidence
- Commits: ab5756d01668a35258bd24b8bc1da29745a977c2
- Tests: go test -race -count=3 -tags test_dep ./common/testing/testpilot/..., go test -tags test_dep ./tools/umpire/replay/..., make umpire-check-retired-vocabulary, make lint-code-fast, baseline: green (testpilot -race -count=3, replay)
- PRs: