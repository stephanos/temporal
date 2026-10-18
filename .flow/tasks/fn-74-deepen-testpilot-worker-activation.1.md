---
satisfies: [R1, R3, R4, R5]
---
# fn-74-deepen-testpilot-worker-activation.1 Encapsulate pure activation state and work accounting

## Description
Build the private activation owner and its isolated tests (R1, R3–R5). Interpreter migration follows in task 2.

**Size:** M
**Files:** new `common/testing/testpilot/temporal/internal/activation/activation.go` and `activation_test.go`.
**Touches:** [common/testing/testpilot/temporal/internal/activation/**]

### Approach
- Construct from public `testpilot.EntrypointPlan`; reject the comparable zero plan before getters and reject controller context. Cache the instruction plans once and derive the existing production work ceiling.
- Move reference resolution, guard/input evaluation, cumulative charging and atomic owned outcome storage behind construct/evaluate/admit. Delegate semantics to existing public plan methods; do not import generic Testpilot internal packages.
- Track each instruction's evaluated/skipped/admitted/failed state. Reject repeated evaluation and admission without an enabled evaluation; do not enforce DAG traversal or introduce scheduling.
- Preserve returned success/error work charges and existing copy units. Exact remaining allowance can succeed; subsequent operations at zero reject. Private tests may inject a tight allowance without exposing an override API.
- Test guard/presence behavior, invalid indices/contexts, mutation isolation, rejected partial fields, canceled contexts, and independent sequential/concurrent activations sharing a prepared plan.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:304` — current composition and charges.
- `common/testing/testpilot/temporal/worker/values.go:9` — resolver and snapshot ownership.
- `common/testing/testpilot/driver.go:221` — public evaluation/validation authority.
- `common/testing/testpilot/internal/execution/program.go:233` — getter preconditions and guard-first semantics (read-only reference).
- `common/testing/testpilot/internal/execution/values.go:156` — ceiling and work-unit semantics (read-only reference).
**Optional:**
- `common/testing/testpilot/runtime_fixture_test.go` — admitted plan fixture pattern.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot/temporal/internal/activation`
`go test -race -tags test_dep ./common/testing/testpilot/temporal/internal/activation`

## Acceptance
- [ ] Private construct/evaluate/admit interface satisfies R1 without widening public plan APIs or importing Testpilot internals.
- [ ] Interface-level tests cover R3's enabled/skipped/error states and repeated evaluation/admission rejection without resets.
- [ ] R4 accounting tests pin exact-bound success, one-less failure, consumed error-path charges, zero allowance and per-activation isolation against existing work units.
- [ ] R5 tests retain immutable input/outcome snapshots and atomic failure behavior across malformed, nil, oversized and canceled inputs plus concurrent independent states.
- [ ] Focused unit/race commands pass; preserve existing comments and all unrelated changes.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: