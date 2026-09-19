---
satisfies: [R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.9 Transfer operation cancellation capability to deferred fn-79

## Description
**Scope transferred to deferred fn-79 by user decision. No cancellation implementation is authorized here. Original requirements below are archival.**

Add the minimal generic per-operation cancellation instruction and capability required by D4/D5. Testpilot authorizes cancellation of one previously started operation; the Temporal worker owns the SDK cancellation handle and submits cancellation without treating submission as semantic confirmation or forcing the final result.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/{instruction,program}.proto`, generated Testpilot Go/Lean protocol files, `model/Testpilot/Authoring.lean`, `common/testing/testpilot/internal/execution/{contracts,program,prepare,scheduler}*.go`, `common/testing/testpilot/temporal/worker/{interpreter,session,driver,runtime_fixture_test,runtime_test,sdk_test}.go`
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/instruction.proto, proto/internal/temporal/server/api/testpilot/v1/program.proto, api/testpilot/v1/**, model/Testpilot/Authoring.lean, common/testing/testpilot/internal/execution/**, common/testing/testpilot/temporal/worker/**]

### Approach
- Add one feature-neutral instruction referencing an admitted in-flight operation/effect identity; do not add a Nexus branch to generic server transport.
- Admit the reference, dependency, entrypoint context, capability claim, and work bounds statically before Driver I/O.
- Have the Temporal worker create and retain a distinct SDK cancel function with each started Nexus operation, invoke exactly that handle, and keep workflow cancellation and activation shutdown separate.
- Return only effect-submission status. Cancellation confirmation and canceled/completed resolution remain observation-owned semantic evidence.
- Make completion-versus-cancellation, duplicate cancellation, close, and cleanup races deterministic and bounded; never retry automatically.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` — current closed instruction union
- `common/testing/testpilot/internal/execution/contracts.go` — admitted opcode/capability contracts
- `common/testing/testpilot/internal/execution/scheduler.go` — effect scheduling and cancellation ownership
- `common/testing/testpilot/temporal/worker/interpreter.go` — SDK instruction execution
- `common/testing/testpilot/temporal/worker/sdk_test.go:364-389` — current async Nexus future path

### Key context
- The instruction authorizes cancellation submission only; a correlated history event is required for the model step.
- Per-operation handles must survive only for their owning Run/activation and must not become workflow or server-global state.

## Acceptance
- [ ] A closed feature-neutral instruction can cancel one admitted in-flight operation/effect by stable identity through the existing generic server capability seam.
- [ ] Prepare rejects missing, duplicate, cross-entrypoint, wrong-kind, unbounded, or unauthorized cancellation references before Driver I/O.
- [ ] The Temporal worker stores a distinct SDK cancel function for each started Nexus operation and never substitutes workflow cancellation or activation shutdown.
- [ ] Cancellation submission produces effect status only and cannot emit cancellation-confirmed or terminal semantic steps.
- [ ] Tests cover cancel-before-await, completion-before-cancel, cancellation/completion races, duplicate/late cancel, missing handle, Run close, cleanup, and concurrent operation isolation without deadlock or automatic retry.
- [ ] Existing completion and success behavior remains valid; server dependency tests stay Nexus-free.
- [ ] Protocol/Lean generation, focused admission/worker tests with `-tags test_dep`, compatibility fixtures, and scoped lints pass.


## Done summary
Administrative scope transfer only: cancellation requirements deferred to fn-79 by explicit user decision. No cancellation implementation is claimed complete. Original requirements retained in fn-79; generic syntax/qualification from task 8 retained in fn-78.10. Existing unfinished source edits preserved and worker stopped. This task closes only the scope transfer so generic fn-78 and fn-70 can proceed.
## Evidence
- Commits:
- Tests:
- PRs: