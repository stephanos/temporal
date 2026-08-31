---
satisfies: [R3, R4]
---
# fn-21-nexus-duplicate-observation-control.4 Capture fault realization as closed causal evidence

## Description
Project the requested fault, completed real receipt, injected marker, separate counts, and shared correlation into fn-19/fn-18 operational artifacts for R3/R4. Prove operational success only when all existing phase/source/cleanup obligations close.

**Size:** M
**Files:** `tools/umpire/runtime/evidence.go`, `tools/umpire/internal/artifactv2/evidence.go`, `tools/umpire/internal/runtimeengine/evidence.go`, `tools/umpire/internal/runtimeengine/engine.go`, `tools/umpire/temporal/nexus/participant.go`, `tools/umpire/temporal/nexus/evidence.go`, `tools/umpire/temporal/nexus/output.go`, `tools/umpire/temporal/nexus/evidence_test.go`, `tools/umpire/temporal/nexus/integration_test.go`, `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/**`
**Touches:** [tools/umpire/runtime/evidence.go, tools/umpire/internal/artifactv2/evidence.go, tools/umpire/internal/runtimeengine/evidence.go, tools/umpire/internal/runtimeengine/engine.go, tools/umpire/temporal/nexus/participant.go, tools/umpire/temporal/nexus/evidence.go, tools/umpire/temporal/nexus/output.go, tools/umpire/temporal/nexus/evidence_test.go, tools/umpire/temporal/nexus/integration_test.go, tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/**]

### Approach
- Reuse fn-19's four-source accumulator and fn-18 Run/RawEvidence constructors; extend the existing participant fact producer and accumulator with only the domain-neutral synthetic-contribution count and marker fields required by Task `.7`'s checked profile.
- Bind the one control attempt to the planned occurrence, exact non-null fault identity, accepted status, and one receipt fact. Project mechanical callback count one, synthetic-contribution count one, and the marker as unique ordered transport facts sharing the real run/operation/cancellation correlation.
- Retain one normal history chain with one requested and one completed cancellation event, explicit phase/source closure, and zero open handles; never label the injected observation as a second SDK/server callback or history request.
- Charge every extra fact/field before append under existing limits and emit the mutation table's operational `incomplete`/status-2 Known Gap and gap on N+1.
- Build and fn-18-admit the four-member output in memory; independently mutate callback count, contribution count, marker, fault, receipt, correlation, causality, order, disposition, closure, and capacity and pin every case to the exact table row before one bounded live control.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.7.md:13-31` — four-source evidence/output pattern
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.3.md:13-30` — accumulator, limits, and independent oracle
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:83-84` — exact phase/control attempt records
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md:136-180` — Run/RawEvidence/set invariants
- `.flow/specs/fn-21-nexus-duplicate-observation-control.md` — closed mutation/status table

### Acceptance
- [ ] The admitted faulted run has one fault-bound accepted control receipt and four closed sources with exact causal/reference closure.
- [ ] Evidence truthfully records one real callback and one separate labeled synthetic contribution while retaining the normal requested/completed lifecycle.
- [ ] Full realization/closure/cleanup is operationally succeeded; every mutation produces the exact invariant/failed/incomplete outcome in the parent table.
- [ ] Unsafe payload/header/error/authority material cannot enter retained fields.
- [ ] The normal evidence fixture remains byte-identical and cross-run facts cannot bind.
## Acceptance
- [ ] R3 operational status remains independent from the induced semantic failure.
- [ ] R4 exact fault-realization evidence is bounded, causal, closed, and admitted.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
