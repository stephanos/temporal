---
satisfies: [R2, R4]
---
# fn-146-adopt-cel-for-runtime-predicates-and.4 Move Testpilot execution bindings to native CEL

## Description
**Note 2026-10-18 (one-language series):** Depends on fn-202 (Testpilot protos move under the framework) and then fn-203's single Case regeneration.

Move instruction inputs, execution guards and evidence guards to the admitted CEL environment for R4. Prove descriptor-backed execution against offline kernel evaluation before verification rollout. Keep capture lifetimes, descriptor reads and correlated occurrence selection as domain bindings.

**Size:** M
**Files:** public Driver contracts, Temporal activation and worker consumers, replay/test Driver adapters, execution dataflow/evidence bindings, shared CEL binding helpers and focused execution fixtures
**Touches:** [common/testing/testpilot/driver.go, common/testing/testpilot/temporal/**, common/testing/testpilot/contract/**, common/testing/testpilot/internal/testsupport/**, common/testing/testpilot/replay/**, common/testing/testpilot/*_test.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/ir/**]

### Approach
- Migrate public Driver/activation contracts using `*testpilotspb.Value` and Temporal worker payload construction (`temporal/worker/typed.go`) to the admitted standard CEL value representation. Include every live-worker constructor and recorded Run serializer of the retired container family.
- Prove one awaited payload through the real Driver boundary, outcome admission and recorded Run serialization. Restore adapter and execution compilation and run Task 3's deferred value/type tests.
- Translate instruction inputs, execution guards and evidence guards to CEL; verification sites belong to Task 5.
- Use CEL-aware reference traversal for execution admission; shared rule expansion and verification walkers belong to Task 5.
- Differentially replay the same event data through online execution and offline verification, classifying intentional semantic changes.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/ir/rule_instances.go:65-150` - expression substitution and expansion
- `common/testing/testpilot/internal/execution/dataflow.go:630-790` - scope and dataflow binding
- `common/testing/testpilot/internal/verification/prepare.go:420-500` - Contract predicate admission
- `common/testing/testpilot/internal/verification/correlated_prepare.go:49-92` - correlated predicate restrictions
- `common/testing/testpilot/internal/verification/correlated.go:313-390` - current correlated evaluation

### Key context
A missing binding becomes standard CEL behavior unless a named domain requirement keeps the old rule. Preserve exact capture ordinals, global capture-ID uniqueness and bounded instance expansion.


### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/internal/ir/... ./common/testing/testpilot/internal/execution/... ./common/testing/testpilot/internal/verification/...
```

Current-format dispatch uses CEL exclusively; no historical evaluator is retained.
## Acceptance
- [ ] Native CEL evaluates instruction inputs, execution guards and evidence guards.
- [ ] Real descriptor paths, captured values and recorded event inputs agree with offline kernel evaluation.
- [ ] Scope, cancellation and cost errors match the documented site matrix.
- [ ] Focused IR and execution suites pass.
- [ ] Public Drivers and Temporal activation/worker constructors use the new value contract; an awaited payload survives outcome admission and Run serialization.
## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
