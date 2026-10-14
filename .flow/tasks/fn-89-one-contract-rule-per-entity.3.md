---
satisfies: [R3, R4]
---
# fn-89-one-contract-rule-per-entity.3 Evaluator per Rule instance and the differential Verdict-identity test

## Description
Evaluate one Run-local state per Rule instance (R3) and prove Verdict and admission identity against the expansion with a differential test (R4). This is the spec's early proof point: if identity fails, the binding-once design is re-evaluated before the Producer fold builds on it.

**Size:** M
**Files:** `common/testing/testpilot/internal/verification/evaluator.go`, `common/testing/testpilot/internal/verification/instances_test.go` (new differential test), `common/testing/testpilot/internal/verification/evaluator_test.go` (only if helpers move), `common/testing/testpilot/internal/ir/evaluate.go` (runtime charge of an instance value reference, which `ir.Evaluate` charges nothing today at `:74`)
**Touches:** [common/testing/testpilot/internal/verification/evaluator.go, common/testing/testpilot/internal/verification/instances_test.go, common/testing/testpilot/internal/verification/evaluator_test.go, common/testing/testpilot/internal/ir/evaluate.go]

## Approach
- `newEvaluator` (`evaluator.go:86-96`) allocates one `ruleState` and one `RuleVerdict` per Rule instance, in Rule then instance declaration order, seeding `RuleId` with the instance's rule ID. Every place keyed off `m.source.RuleId` or `len(p.rules)` moves to the instance: transition traces (`:279,:309`), the `Violation.RuleID` that stops the Executor (`:446-454`), the all-satisfied count in `verdict()` (`:470-476`), and the correlated offset `Rules[len(e.prepared.rules)+i]` (`:545`).
- The resolver closure (`:280-293`) resolves the new reference kind to the evaluated instance's assigned value, and runtime work charges it the assigned value's proto size as `Evaluate` charges an inlined literal, so later rules' remaining per-event budget matches the expansion's; deadline counting (`deadlineReached` `:320`) stays per `ruleState`, so each instance's `rule_events` counter is its own (EVD-21).
- Differential test: build instanced Contracts and, independently of the preparation code, their expansion (each instance a plain Rule with values inlined as literals; capture and transition IDs may keep their names, the Verdict names neither). Prepare both, evaluate online (`Observe`) and offline (`Evaluate`), and compare Verdicts with `proto.Equal` plus a readable first-differing-rule-ID message. Follow the live-vs-offline pattern `nexusEvaluateLiveAndOffline` (`nexus_correlation_test.go:169`) and the Run builders `event`/`observed` (`evaluator_test.go:16,182`).
- Case matrix (spec R4): satisfied; violated through a safety-shaped Rule whose reject transition compares against the instance value (the pair capture shape has no violated state); inconclusive through the pair capture shape under a crossed completion; incomplete; deadline-expired; an event carrying no instance's value and one matching no instance; a plain Rule beside an instanced one; a one-instance Rule; a correlated rule after the instances; a ceiling where both admissions reject on the same ceiling. Mutation check: swap two instances' values and require the comparison to fail.

## Investigation targets
**Required**:
- `common/testing/testpilot/internal/verification/evaluator.go:34-120,266-330,440-560`
- `common/testing/testpilot/internal/verification/nexus_correlation_test.go:149-200`
- `common/testing/testpilot/internal/verification/evaluator_test.go:16-200,399-460`

**Optional**:
- `.flow/memory/bug/runtime-errors/freeze-contract-transitions-when-2026-09-05.md` — transitions freeze on incomplete execution

### Carried from fn-89.2 (2026-09-27)
- R3 gap: the whole-Contract size check (`ir.CheckSurface`) is not charged per instance, so an instanced Contract whose instance values are large and read many times can pass while its expansion fails. Charge it per instance like the other ceilings, and make the differential admission test cover a Contract that the expansion rejects on surface size.
- Review notes: the admission struct's recording state could be a small recorder owned by `bindRule`; `literalLike` should check `ok` before using `expression`.

## Acceptance
- [ ] one `RuleVerdict` per instance in declaration order; Executor stop, traces, satisfied count and correlated offset count instances
- [ ] online and offline answers are identical for instanced Contracts
- [ ] the differential test covers the full R4 matrix, reports the first differing rule ID, and fails under the value-swap mutation
- [ ] existing evaluator and correlation tests pass unchanged
- [ ] `go test -count=1 -tags test_dep ./common/testing/testpilot/...` passes; `make lint-code-fast` clean on changed packages

## Done summary
The Evaluator now keeps one Run-local state and one `RuleVerdict` per Rule instance, in Rule then instance declaration order (R3). A plain Rule counts as its own single instance. Transition traces, the Executor-stop `Violation`, the all-satisfied count and the correlated offset all count instances. The resolver reads each instance's values. `ir.Evaluate` now charges an instance-value read at the inlined literal's size, so remaining per-event budgets match the expansion's.

Carried R3 gap closed: `Prepare` also runs `ir.CheckExpandedSurface` over the Contract's expansion. That function walks each instance as its inlined plain Rule without building the whole expansion. An instanced Contract now rejects on surface size with the same diagnostic as its expansion. The fn-89.2 review notes are done: the admission ledger is a small recorder owned by `bindRule`, and `literalLike` checks `ok` first.

Tests:
- `TestRuleInstancesEvaluateAsTheirExpansion` (`instances_test.go`) covers the R4 matrix: satisfied; violated (safety reject on the instance value); inconclusive (pair capture shape with a crossed completion); incomplete; `rule_events` deadline expiry; an event carrying no instance's value and one matching no instance; a plain Rule beside instanced ones; a one-instance Rule; a correlated rule after the instances. Verdicts must be byte-identical online and offline. Failures name the Run and the first differing rule ID. Swapping two instances' values fails every case.
- `TestRuleInstancesRejectOnTheExpansionsCeiling` covers the states ceiling and surface size.
- The runtime charge is pinned in `TestInstanceValuesBindAsTheLiteralEachInstanceInlines`.

Every new test was confirmed red without its fix. No protocol or catalog change, so no pinned Runs were re-recorded.

Follow-ups: (1) `execution.Prepare` still checks the whole Case surface as authored, not as expanded. (2) The instanced Contract's own surface check can reject where the expansion would pass, if large declared values are never read (reviewer P3, pre-existing). (3) Reviewer FYIs: `inlineInstanceValues` duplicates the test's `inline`; `nexusWorld` and `nexusRun` lack `t.Helper()`.

Base note: commit a94bfa142c (fn-88, another worker) landed between base 9daf0bd and this task's commit. The review was scoped with `--base a94bfa142c`.

stage: impl-review - ran (claude, first-pass SHIP)
## Evidence
- Commits: 2178a1c70b8ff4cbac69f7aca2c363919525073e
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/..., make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD
- PRs: