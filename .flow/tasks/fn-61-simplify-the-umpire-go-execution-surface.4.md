---
satisfies: [R3, R5, R6]
---
# fn-61-simplify-the-umpire-go-execution-surface.4 Unify runtime contracts and the execution state machine

## Description
Consolidate the domain-neutral runtime vocabulary and `internal/runtimeengine` state machine into the private execution module established by Task `.3` (R3). Reduce constructor and wrapper layers while preserving the exhaustive lifecycle oracle.

**Size:** M
**Files:** `tools/umpire/runtime/**`, `tools/umpire/internal/runtimeengine/**`, `tools/umpire/internal/execution/**`, `tools/umpire/executor/**`, `tools/umpire/runevaluation/**`
**Touches:** [tools/umpire/runtime/**, tools/umpire/internal/runtimeengine/**, tools/umpire/internal/execution/**, tools/umpire/executor/**, tools/umpire/runevaluation/**]

### Approach
- Move the runtime request, authority, program/command, participant/environment, output, phase, receipt, resource, and Evidence contracts beside the state machine that owns them.
- Replace the 16-argument authority constructor with cohesive internal construction owned by the sole Temporal profile; do not expose a generic builder or options object.
- Fold one-to-one engine/output/evidence forwarding layers into the internal module and delete duplicate aliases while retaining separately testable pure admission and state-transition functions.
- Migrate Run Evaluation's operational-status helper, Evidence constants, and tests to the retained Artifact/offline boundary or the private execution module without exposing that module through the public checker API.
- Port the exhaustive phase/outcome oracle, capacity boundaries, cancellation isolation, cleanup precedence, and typed-nil cases before deleting old packages.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runtime/runtime.go:22-182` — phases, limits, authority, and wide constructor
- `tools/umpire/runtime/participant.go:14-153,620-636` — program and execution interfaces
- `tools/umpire/runtime/request.go:12-120` — checked request admission
- `tools/umpire/internal/runtimeengine/engine.go:152-330` — lifecycle state machine
- `tools/umpire/internal/runtimeengine/engine_test.go:63-182,283-505` — independent oracle and boundary matrix
- `tools/umpire/runevaluation/result.go:1-140` — offline result construction that consumes runtimeengine status

**Optional** (reference as needed):
- `tools/umpire/runtime/request_test.go` — checked-value and collection boundaries

### Key context
The state-machine complexity is real distributed-systems correctness and should remain explicit and directly testable. The simplification target is package/export/constructor duplication, not removal of cleanup or causal-Evidence invariants.

### Acceptance
- [ ] Runtime and runtimeengine no longer exist as separate public/shallow packages; one internal execution module owns their contracts and state machine.
- [ ] The wide generic authority constructor and pass-through output/engine wrappers are removed in favor of closed internal construction.
- [ ] The public offline Run Evaluation package compiles without public runtime/runtimeengine imports and retains its Artifact inputs, status mapping, CLI behavior, and tests.
- [ ] Exhaustive phase/outcome, N/N+1 Evidence capacity, cancellation/deadline, cleanup dominance, single-flight, source closure, and typed-nil tests retain exact results.
- [ ] The internal module remains testable without a Temporal cluster.
- [ ] Production Go lines do not increase across the migrated runner/runtime/runtimeengine stack.

## Acceptance
- [ ] Runtime vocabulary and state transitions have one private owner.
- [ ] Exhaustive pure tests preserve all lifecycle semantics while shallow layers disappear.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
