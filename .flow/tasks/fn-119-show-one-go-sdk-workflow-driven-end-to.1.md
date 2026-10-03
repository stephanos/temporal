---
satisfies: [R2]
---
# fn-119-show-one-go-sdk-workflow-driven-end-to.1 Schedule an activity from the Driver's workflow, await it and complete the workflow with its result

## Description
Workflow-side command plumbing for the generic Driver primitives, in Go only, tested with hand-built Testpilot Cases that name no example. This is the early phase the spec allows: it does not depend on the example Model, fn-118 or fn-120. Activity-attempt routing is task 2.

**Cross-spec entry gate:** none beyond fn-117 (closed); may start before fn-114/fn-118/fn-120 close. fn-119's spec-level dependencies on fn-114, fn-118 and fn-120 stay (they gate the example), so a conductor must claim this task explicitly rather than wait for the spec to turn ready. fn-118.3 also edits `common/testing/testpilot/**`; whichever lands second rebases.

**Size:** M
**Files:** `common/testing/testpilot/temporal/worker/{typed,interpreter}.go`, `temporal/profile.go`, `internal/execution/typed.go`, `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` (doc and await scope only, if needed), Driver tests.
**Touches:** [common/testing/testpilot/temporal/worker/typed.go, common/testing/testpilot/temporal/worker/interpreter.go, common/testing/testpilot/temporal/profile.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/temporal/worker/*_test.go, proto/internal/temporal/server/api/testpilot/v1/**]

### Approach
- Add `COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK` to `CommandTypes()` (`worker/typed.go:35`) with a schedule function beside `scheduleNexus` (:58); the generic `WorkflowCommand` (`instruction.proto:159`) already wraps an API `Command`, so prefer no new proto message.
- Widen `Await` (`interpreter.go:107`, futures map :58) from Nexus-only to activity futures; `Finish` completes the workflow with an awaited outcome via `InstructionOutcomeReference` (`expression.proto:105`).
- Generalize the Nexus-only checks in `internal/execution/typed.go` (`driverReach` :34, `checkReach` :72, `bindWorkflowCommand` :113-129, `nexusOperationOf` :240-256). Preparation rejection names the command type (R2 errors; the current message omits it).
- Test with a hand-built Case whose activity runs on an ordinary test worker activity, so this task does not need task 2's attempt routing.

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:55-130`
- `common/testing/testpilot/temporal/worker/typed.go`
- `common/testing/testpilot/temporal/profile.go:200-290`
- `common/testing/testpilot/internal/execution/typed.go`

### Quick commands
```bash
go test -count=1 -tags test_dep ./common/testing/testpilot/...
make lint-code-fast
```

### Execution constraints
- General primitives only: no name, type or Query of the example appears in Go (R4 will enforce it).
- Existing Nexus and standalone-activity Cases prepare and run unchanged.
## Acceptance
- [ ] A Profile can authorize schedule-activity; the dynamic workflow schedules it, awaits its result and completes the workflow with it.
- [ ] A Case with a command the Profile does not list is rejected at preparation with the command type named.
- [ ] Existing Cases unchanged; Testpilot tests and lint-code-fast pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
