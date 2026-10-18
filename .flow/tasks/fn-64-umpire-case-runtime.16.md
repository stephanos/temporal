---
satisfies: [R2, R4, R7, R9]
---
# fn-64-umpire-case-runtime.16 Share prepared worker outcome and value semantics

## Description
Expose the prepared worker value boundary required by task6, while keeping controller and SDK validation identical.

**Size:** M
**Files:** `tools/umpire/internal/execution/{program,dataflow,projection,values}.go`, focused execution tests, `tools/umpire/host.go`, execution README, bounded IR snapshot traversal and focused regression tests
**Touches:** [tools/umpire/internal/execution/**, tools/umpire/host.go, tools/umpire/internal/ir/catalog.go, tools/umpire/internal/ir/read_test.go, tools/umpire/internal/ir/runtime_value_test.go]

### Approach
- Extract opcode-aware outcome validation, declared-field lookup and ownership copies from `stageOutcome`; controller staging delegates to the same implementation.
- Expose a small immutable prepared outcome/value interface through the existing root plan wrappers, with the prepared runtime-work ceiling. Keep compiled types and mutable controller stores private.
- Let worker adapters supply replay-local lookups to already compiled expressions. Operations must be deterministic, bounded and free of controller-store locks, SDK calls, networking or goroutine scheduling.
- Preserve deterministic work-limit error precedence in shared protobuf snapshots: bound and charge present-field/map-key ordering before traversal, preserving presence and extension semantics. Include focused IR normal and race regression gates.
- Reject VALUE declarations on StartNexusOperation; retain Await and evaluated Finish/RespondNexus result semantics. Preserve guard and failure precedence.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/execution/projection.go:58` — existing stageOutcome authority
- `tools/umpire/internal/execution/values.go:153` — prepared runtime work and value ownership
- `tools/umpire/internal/execution/program.go:113` — immutable plan accessors
- `tools/umpire/internal/execution/dataflow.go:164` — outcome admission
- `tools/umpire/host.go` — public adapter seam
- `.flow/tmp/fn64-task6-investigation.md` — SDK constraints

### Key context
A future is an SDK handle, not an IR Value. No proto/Lean generation or second evaluator is needed.
## Acceptance
- [ ] Controller and worker callers use one outcome validator; tests compare statuses, fields, snapshots, work exhaustion and error precedence through both paths.
- [ ] Tests reject Start VALUE during preparation; Await accepts exactly its declared result type, with malformed/missing/undeclared/oversized failure cases.
- [ ] Concurrent activation lookups cannot mutate prepared data or another activation; false guards and missing guarded results retain existing semantics.
- [ ] Focused tagged execution/root tests and race tests, make fmt-imports and scoped make lint-code pass; document the worker API handoff.
## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: