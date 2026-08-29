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
- Build four-source RawEvidence through an append-only accumulator enforcing source/fact/field/payload/byte/count/order/reference/DAG limits before append and explicit N+1 gaps/Known Gaps.
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
Implemented the bounded five-phase runtime engine, request-bound four-source evidence accumulator, and artifact-owned exact-set execution extension. Independent oracle coverage now includes every phase terminal outcome, both preparation actors, every receipt status, context races, compound precedence, duplicate/missing receipts, all-source N/N+1 capacity, cleanup dominance, and exact admitted Run/RawEvidence closure.

Baseline: green for runtime, temporaltest, and Temporal.System.Execution.LocalProfileTests; inherited red for local/Nexus/CLI packages, Nexus ExecutionTests, and the root run target because those later-task surfaces were absent before this task. Verification: all task-owned/currently available gates passed; the first temporaltest verification attempt hit an incomplete downloaded module cache and passed after retry against a validated complete cache. Memory capture was attempted after NEEDS_WORK to SHIP but the repository memory store is not initialized.

stage: impl-review - ran [2026-08-29T13:05:55Z..2026-08-29T13:20:40Z]
## Evidence
- Commits: 5beeb23b9fa0c9fd8111387b1d0cf95044be387b, fe9ad3ccbd37de1e303b784decc78d45b22481ef
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/artifact/..., go test -race -count=1 ./tools/umpire/runtime/..., go vet ./tools/umpire/runtime/..., go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, git diff --check, baseline: red (go test -count=1 ./tools/umpire/temporal/local/... failed pre-edit: later-task package absent), baseline: red (go test -count=1 ./tools/umpire/temporal/nexus/... failed pre-edit: later-task package absent), baseline: red (go test -count=1 ./tools/umpire/cmd/umpire-local-run/... failed pre-edit: later-task package absent), baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests failed pre-edit: later-task module absent), baseline: red (make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=umpire.local.caller-closure.run-1 failed pre-edit: later-task target absent)
- PRs: