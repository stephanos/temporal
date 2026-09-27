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

## Acceptance
- [ ] Each D1 consolidation leaves one definition; `execute`/`executeCleanup` stay apart.
- [ ] `RuleViolation` is an alias of `verification.Violation`; the gate passes.
- [ ] Corpus unchanged; `-race -count=3` tests and lint pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
