---
satisfies: [R2, R4]
---
# fn-146-adopt-cel-for-runtime-predicates-and.4 Move Testpilot execution bindings to native CEL

## Description
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
