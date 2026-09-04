---
satisfies: [R2, R3, R5, R6]
---
# fn-63-consolidate-umpire-go-tests-into-golden.2 Consolidate portable execution and evaluation scenarios

## Description
Migrate duplicated portable execution/evaluation behavior to the shared scenario corpus (R2/R3/R5/R6) after the proof task fixes the harness contract. Preserve state-machine and fail-closed invariants as focused tests.

**Size:** M
**Files:** surviving post-`fn-61` executor tests, `tools/umpire/portableevaluation/evaluator_test.go`, `tools/umpire/portableevaluation/parity_test.go`, `tools/umpire/portableevaluation/portable_test.go`, `tools/umpire/portableevaluation/testdata/**`
**Touches:** [tools/umpire/umpire_test.go, tools/umpire/internal/execution/**/*_test.go, tools/umpire/executor/*_test.go, tools/umpire/portableevaluation/*_test.go, tools/umpire/portableevaluation/testdata/**]

### Approach
- Re-anchor current executor/evaluator matrices to `fn-61`'s surviving root facade and internal evaluator; never restore legacy HTTP or `ExecuteRequest` paths merely to preserve a test.
- Express broad success, duplicate-delivery violation, incomplete/correlation-conflict inconclusive, required-obligation, and crossed binding/checksum pre-I/O rejection paths as named scenarios using Lean-owned plan/contract/evidence/result fixtures.
- Remove duplicated request, runner, result, and parity-loading helpers once their callers use the shared harness.
- Retain compact direct tests for single-flight overlap, cancellation/deadlines, reusable versus poisoned cleanup state, typed-nil boundaries, mutation precedence, and operator branches not proven by a scenario.
- Map every removed test to a scenario or retained invariant category in the task completion summary.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/executor/portable_executor_test.go:68-249` — portable lifecycle matrix being relocated by `fn-61`
- `tools/umpire/portableevaluation/evaluator_test.go:20-1057` — broad evaluator matrices and focused operator cases
- `tools/umpire/portableevaluation/parity_test.go:482-555` — current model-derived outcome parity
- `tools/umpire/executor/executor_test.go:122-440` — stateful lifecycle cases that must remain focused
- `tools/umpire/executor/executor_test.go:443-495` — duplicated fixture request/runner setup

**Optional** (reference as needed):
- `tools/umpire/portableevaluation/testdata/**` — existing plans, contracts, evidence, and Lean results

## Acceptance
- [ ] The named portable scenarios cover success, violation, inconclusive, required-obligation, and crossed binding/checksum pre-I/O rejection through surviving supported boundaries.
- [ ] Scenario failures assert exact typed status/error categories and established precedence, not only pass/fail text.
- [ ] Single-flight, cancellation, cleanup poisoning, typed-nil, bounded operator, and mutation-precedence cases remain focused where static fixtures cannot prove them.
- [ ] Duplicated fixture/request/runner/result helpers and superseded broad table cases are removed, with each removal mapped in the task summary.
- [ ] Focused package tests and portable fixture-diff checks pass without runtime Lean invocation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
