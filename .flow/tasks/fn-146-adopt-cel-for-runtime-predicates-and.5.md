---
satisfies: [R4, R6]
---
# fn-146-adopt-cel-for-runtime-predicates-and.5 Move Contract and correlated verification to CEL

## Description
Move ordinary and correlated Contract predicates, rule instances, capture binding and offline verification to the same CEL environment after the execution proof.

**Size:** M
**Files:** `common/testing/testpilot/internal/verification/**`, `common/testing/testpilot/internal/ir/rule_instances.go`, shared expression canonicalization and reference discovery
**Touches:** [common/testing/testpilot/internal/verification/**, common/testing/testpilot/internal/ir/rule_instances.go, common/testing/testpilot/internal/ir/*expression*]

### Approach
- Reuse the environment, adapter and site matrix from Tasks 2 through 4.
- Move predicate and correlated-fact evaluation, instance substitution, reference discovery and capture scope checks together.
- Differentially verify identical events with exact capture ordinals, existential multi-fact matching and instance ceilings.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/verification/prepare.go:420-500` - Contract predicates
- `common/testing/testpilot/internal/verification/correlated_prepare.go:49-92` - restricted conditions
- `common/testing/testpilot/internal/verification/correlated.go:313-390` - correlated values and evaluation
- `common/testing/testpilot/internal/ir/rule_instances.go:65-150` - substitution and expansion

### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/internal/ir/... ./common/testing/testpilot/internal/verification/...
```

## Acceptance
- [ ] R4 and R6 use CEL at every successor-format verification site and walker.
- [ ] Exact capture namespaces, ordinals, occurrence lifetime and expansion ceilings remain pinned.
- [ ] Online and offline verdict behavior agrees on the same recorded inputs.
- [ ] Focused verification and instance tests pass.

## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
