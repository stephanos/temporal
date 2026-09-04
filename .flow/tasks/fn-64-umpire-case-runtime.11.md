---
satisfies: [R1, R2, R4, R6, R7, R9]
---
# fn-64-umpire-case-runtime.11 Admit Programs and define bounded execution contracts

## Description
Compile the Program and static Host policy into immutable execution data (R1, R2, R4, R7), and
define the private driver/Monitor contracts consumed by evaluation and scheduling. This task owns
the missing bounded activation-reservation schema discovered during preparation re-anchoring.

**Size:** M
**Files:** `tools/umpire/internal/execution/{prepare,contracts,program}.go` and tests; Program proto,
generated Go/Lean Program types and focused schema tests
**Touches:** [tools/umpire/internal/execution/**, proto/internal/temporal/server/api/umpire/v1/program.proto,
api/umpire/v1/program*.go, model/Umpire/Case/Program.lean, model/Umpire/CaseTests.lean,
model/Temporal/API/Types.lean, tools/umpire/cmd/umpire-gen-lean-api/**]

### Approach
- Consume task 2's compiled catalog/type/path/expression interface. Validate Case envelope/version,
  identifiers, proto surface, Program limits, roles, controller/worker contexts and entrypoint-local
  DAGs without importing the root facade, verification implementation, or Temporal adapters.
- Accept a root-independent immutable policy value: catalog identity, allowed endpoint roles/methods,
  SDK capabilities and resource ceilings. The root adapter in task 13 owns public Profile conversion.
- Compile Slot single-writer/dataflow, typed outcome guards, assignment/projection paths, cleanup
  indexes and activation bindings. Prove required inputs available or explicitly guarded; opaque
  capability Slots support readiness/consumption only. Freeze source graphs and policy snapshots.
- Add `ActivationReservation(entrypoint_id, count)` and the repeated `activation_reservations` field
  on InstructionNode. Ordinary controller nodes reserve declared workflow or Nexus-handler entrypoints before
  their effect; cleanup/worker nodes cannot reserve. Counts are positive, targets are unique per
  node and binding contexts match. Activity reservations reject in v1. Each controller entrypoint
  activates once per Run. Admission bounds worker reservations by the maximum sum of per-node
  reservation count times attempts, where each node respects its local attempt cap and total
  attempts respect ProgramLimits.max_attempts. Add the controller activation count and check the
  global activation limit using overflow-safe arithmetic. Host-assigned reservation IDs are Run-local and include
  the originating controller instruction/attempt and reservation ordinal.
- Define narrow private Driver/session/effect-handle and MonitorFactory/Monitor contracts, including
  bounded wait/cancel/drain, opaque capability transfer, post-close diagnostics and factory creation
  before Run creation. Reservation handles explicitly reserve/identify/consume/cancel and reject
  closed or unreserved activations; tasks 4 and 6 implement the barrier and worker side of this contract.
  Expose a narrow immutable prepared-Program observation/bounds view from internal execution for
  Contract preparation and MonitorFactory creation, never scheduling or mutable Slot state.
  Keep implementations of scheduling and evaluation with tasks 4 and 3.
- Regenerate owned Go/Lean schema surfaces through established generators; preserve comments.

### Investigation targets
**Required**:
- `tools/umpire/testplan/validate.go:63` — staged validation and bounded unknown-field checks
- `tools/umpire/testplan/plan.go:49` — clone/validate ownership pattern
- `proto/internal/temporal/server/api/umpire/v1/program.proto:53` — activation binding and graph schema
- `proto/internal/temporal/server/api/umpire/v1/value.proto:158` — Slot/outcome/reference vocabulary
- `tools/umpire/temporal/local/attached.go:62` — identity snapshot precedent
- `.flow/memory/bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04.md` — typed nil coverage

## Acceptance
- [ ] Table-driven tests reject invalid envelope/version/ID/proto surface, DAG cycles/cross-entrypoint
  edges, role/context/capability mismatches, unauthorized methods and every structural/work bound.
- [ ] Dataflow tests cover immutable single assignment, declared outcome fields and every outcome
  status, dependency/presence guards, crossed cardinality, missing required Slots, conflicting
  projections, cleanup-local references and opaque-capability inspection rejection.
- [ ] Reservation schema round-trips in Go and Lean. Admission rejects missing/wrong target binding,
  worker or cleanup reservation, unsupported Activity target, duplicate target, nonpositive count,
  combined local/global attempt-scaled overflow and
  activation ceilings. No RPC-name or request-payload inference selects activation semantics.
- [ ] Internal prepared Program/policy is immutable, factory contracts accept contexts, all nil-capable
  interface forms reject, and effect handles permit bounded cancellation/drain and post-close
  diagnostics without Executor-owned goroutine wrappers. Internal execution has no root/Temporal/
  verification imports, checked by a dependency test.
- [ ] Tagged internal IR/execution/schema tests, generation and applicable Lean/format/lint gates pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
