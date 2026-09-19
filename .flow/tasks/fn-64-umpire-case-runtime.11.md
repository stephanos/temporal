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
Implemented immutable Program/static policy admission, bounded reservation schema and generated Go/Lean types, compiled guarded dataflow and private execution contracts. Scheduling, evaluation, Contract compilation and public composition remain with their assigned later tasks.

Baseline: green applicable IR/schema tests, Lean model build, formatting and scoped Go lint. Baseline lint tooling failures (parallel lock, exhausted disk, cleanup overlap) were resolved before edits; authoritative successful receipt is fn64-task11-baseline-lint-clean-retry.log/rc. No new libraries or commits; staged task-start tree includes prior task 2 work.

Validation: make proto; make umpire-gen-lean-api; tagged IR/execution/schema/API tests; final IR/execution race tests; make fmt-imports; scoped no-fix make lint-code; make umpire-build-model (346 jobs); Lean builtin lint all passed. make lint-model retains one inherited simpNF failure on unchanged ActivationBinding.controller.injEq, reproduced from task-start source in fn64-task11-lean-lint-baseline-proof.log. Initial authored Lean test compile errors were fixed by pattern matching Except results; final build passes. Future live/runtime gates remain owned by tasks 4/5/6/10.

Admission tests cover envelope/policy/limits, DAGs, reservation arithmetic, all status enums, single writers, guards, projections, cleanup, opaque capabilities, ownership boundaries and mutation/concurrent reads. IR guard+value binding shares one budget. Intrinsic outcome status is available with unrelated RPC catalogs and conflicting descriptors reject. RPC raw VALUE rejects; typed projections expose response fields. Controller Slots share Run ownership; worker values stay activation-local. EntrypointPlan/InstructionPlan expose immutable compiled inputs to the future worker adapter. Lean reservation JSON round-trip is test-local; producer codecs remain task 7.

Review round 1 returned NEEDS_WORK: incompatible Await targets and repeated linear method authorization. Fixed with target-opcode validation (including source-order-independent preparation) and charged compiled method sets with duplicate rejection. Focused red regressions reproduced both; post-fix race and scoped lint pass (review-fix2 logs). Reviewer test execution was blocked by its read-only sandbox; worker validation receipts remain authoritative.

stage: impl-review - ran (codex; NEEDS_WORK → fixes → SHIP, two verdict rounds)
stage: plan-sync - skipped(config: planSync.enabled != true)

No commits were authored by this worker. External session advanced HEAD to 156c253a57422518ba7e1266755ff5392a26698b; evidence commits record that external advancement only. Reviewed immutable staged tree c8156dbe96fff2ecad993cb84bfbb0b9a3e9a082 against task-start tree 6473aa0a2761b2ecb4d72e2110f2d5018bf8f974. Owned source matches the reviewed tree after SHIP. Memory capture records producer compatibility and repeated-lookup work bounds.
## Evidence
- Commits: 156c253a57422518ba7e1266755ff5392a26698b
- Tests: baseline: green (applicable scoped commands; authoritative lint clean-retry rc0), make proto, make umpire-gen-lean-api, CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/... ./tools/umpire/cmd/umpire-gen-lean-api/... ./api/umpire/v1/..., CC=/usr/bin/clang go test -race -count=1 -tags test_dep ./tools/umpire/internal/ir/... ./tools/umpire/internal/execution/..., make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, make umpire-build-model, mise exec -- lake --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs, make lint-model: inherited simpNF ActivationBinding.controller.injEq failure reproduced from task-start tree; fn64-task11-lean-lint-baseline-proof.log, post-review fixes: tagged IR/execution race and scoped no-fix lint green (fn64-task11-review-fix2-race.log and fn64-task11-review-fix2-lint.log)
- PRs: