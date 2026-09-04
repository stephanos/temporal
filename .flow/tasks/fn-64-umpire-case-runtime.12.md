---
satisfies: [R2, R3]
---
# fn-64-umpire-case-runtime.12 Compile static Contract and capture admission

## Description
Compile static Contract admission and capture analysis (R2, R3) using the shared IR and prepared
Program observation view. Runtime transition evaluation remains task 3.

**Size:** M
**Files:** `tools/umpire/verification/{prepare,captures}.go` and focused tests
**Touches:** [tools/umpire/verification/**]

### Approach
- Bind rule/state/transition IDs, kinds, terminal states, horizons and closed expression references
  against the immutable Program Observation schema and task 2's typed binder.
- Consume only task 11's immutable observation/bounds view; no scheduler or mutable Slot state is
  available to admission. Task 3's factory binds that same view to each fresh evaluator.
- Prove deterministic selection under declared first-matching transition order and reject malformed
  or unsupported predicates before I/O. Overlapping predicates retain their explicit ordered
  precedence; indexing must never reorder them or introduce unordered selection.
- Analyze capture types, reachable-path definite assignment or explicit presence guards, and
  single assignment. Predicates read pre-transition state; assignments atomically retain the
  matching Observation value and source-event support for later evaluation.
- Bound capture counts/bytes, machine states/transitions, expression depth and per-event/total work
  with overflow-safe arithmetic and immutable prepared indexes. No capture can read Slots or opaque
  capabilities. Consume explicit incomplete/closure and expiry semantics from the parent spec.
- Keep admission independent from target I/O and mutable evaluation state; task 3 supplies the
  production factory and live/offline interpreter using these prepared machines.

### Investigation targets
**Required**:
- `proto/internal/temporal/server/api/umpire/v1/contract.proto:30` — captures, transitions and limits
- `proto/internal/temporal/server/api/umpire/v1/value.proto:185` — Observation/capture expressions
- `proto/internal/temporal/server/api/umpire/v1/run.proto:12` — event kinds and failure coordinates
- `tools/umpire/evaluationcontract/contract.go:186` — existing bounded proto-surface checking pattern
- `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md:245` — approved Contract semantics
## Acceptance
- [ ] Tests accept valid safety/liveness machines and reject duplicate/missing states or transitions,
  invalid terminal/horizon targets, unsupported predicates and unknown Observations. Overlapping
  valid predicates preserve deterministic first-matching declaration order.
- [ ] Capture tests cover matching types, cross-event correlation declarations, required presence
  guards, read-before-assignment, repeated assignment on any reachable path, conflicting branches,
  cycles, forbidden Slot/capability reads and retained supporting-event requirements.
- [ ] Count/byte/depth/work bounds and arithmetic overflow reject before evaluation; source mutation
  cannot change prepared Contract indexes or Observation binding.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/ir/...
  ./tools/umpire/internal/execution/... ./tools/umpire/verification/...` and format/lint pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
