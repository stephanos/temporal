---
satisfies: [R2, R3, R6, R7]
---
# fn-19-bounded-local-temporal-execution-and.3 Implement the bounded five-phase engine and evidence accumulator

## Description
Implement R2/R3/R6's deterministic phase machine, cleanup dominance, bounded fact construction, and admitted output values using fakes first.

**Size:** M
**Files:** `tools/umpire/runtime/engine.go`, `tools/umpire/runtime/evidence.go`, `tools/umpire/runtime/engine_test.go`, `tools/umpire/runtime/evidence_test.go`
**Touches:** [tools/umpire/runtime/engine.go, tools/umpire/runtime/evidence.go, tools/umpire/runtime/engine_test.go, tools/umpire/runtime/evidence_test.go]

### Approach
- Implement exact preparation→realization→observation→isolation→cleanup transitions, phase/global deadlines, the parent operational-precedence table including compound cases, observation after started realization, and fresh isolation/cleanup contexts.
- Track acquired resources immediately and cleanup exactly once; produce canonical phase/control/source diagnostics without hiding cleanup failure.
- Build four-source RawEvidence through an append-only accumulator enforcing source/fact/field/payload/byte/count/order/reference/DAG limits before append and explicit N+1 gaps/omissions.
- Construct Run/RawEvidence/output-set values in memory and admit them through fn-18; perform no publication.
- Implement the independent exhaustive transition-table oracle required by the early proof.

### Investigation targets
**Required** (read before coding):
- Task `.2` checked interfaces
- fn-18 Run/RawEvidence validators and exact phase/status rules
- parent spec five-phase, evidence, and early-proof contracts

### Acceptance
- [ ] Fake success/failure/timeout/cancel at every phase exactly matches the independent oracle.
- [ ] Isolation/cleanup use fresh bounded contexts and cleanup count is exactly one after every acquisition path.
- [ ] N and N+1 evidence cases, causal/reference mutations, closure races, and duplicate receipts produce exact admitted partial/failed artifacts or invariant failure.
- [ ] No invalid engine output passes fn-18 admission.

## Acceptance
- [ ] R2/R3/R6 engine and evidence core passes the mandatory oracle gate.
- [ ] Preflight remains side-effect free; post-start failures remain truthful artifacts.
- [ ] Focused Go engine/evidence tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
