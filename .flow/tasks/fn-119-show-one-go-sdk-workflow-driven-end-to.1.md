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
Schedule an activity from the Driver's workflow, await it and complete the workflow with its result (R2).

What changed
- `worker.CommandTypes()` now lists COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK beside the Nexus schedule, so `DeriveProfile` authorizes it for a Case that carries it.
- Preparation (`internal/execution/typed.go`): Driver-reach rows for ScheduleActivityTaskCommandAttributes, ActivityType, TaskQueue, Payloads, RetryPolicy (header, request_eager_execution, use_workflow_build_id, priority, task_queue.kind/normal_name unrealized). `bindWorkflowCommand` dispatches per type; the activity schedule needs a valid activity type and a task-queue role, and its timeouts and retry intervals are checked against the Profile ceiling. A command type the Profile does not admit is rejected naming the type (`command type COMMAND_TYPE_... the Profile does not admit`); a Profile-admitted type no Driver realizes rejects at its attributes field. `bindAwait` accepts a Nexus or activity schedule (`startsAwaitable`); `startsNexusOperation` keeps its carrier-route meaning.
- Driver: `scheduleActivity` issues `workflow.ExecuteActivity` on the queue the named task-queue role binds (same namespace as the worker), with raw payloads, carried timeouts (schedule-to-close defaulting to the instruction timeout), retry policy and no eager execution; `Await` reads activity and Nexus futures alike; `Finish` completes with the awaited payload.
- Proto: doc-only update of AwaitInstruction and WorkflowCommand comments; generated comment applied to instruction.pb.go (descriptor unchanged, catalog does not move).

Decisions
- The task queue is required and names a task-queue role, because the SDK always emits one and the resource must come from a binding, as the Nexus endpoint does.
- The schedule instruction succeeds once the command is issued (the SDK has no started future for activities); how the activity ended is the Await's outcome.
- `make protoc` cannot run in this sandbox (checked-in goimports is a macOS binary) and its failed run briefly removed the CHASM gen dirs; they were restored from git immediately. The comment-only pb.go change was produced by diffing plain protoc output of HEAD vs edited proto.

Tests: Testpilot suite (`go test -tags test_dep -p 2 ./common/testing/testpilot/...`) exit 0; lint-code-fast's only findings are in another agent's untracked tools/umpire/internal/golden files. New tests: activity admit/reject matrix, command type named, Profile derivation, SDK test-environment run (retry then success, carried input/queue/timeouts) and non-retryable failure recorded on the Await.

Review: Fable (claude-fable-5-1, high) round 1 SHIP. P3 notes: the workflowSourceKey on the activity context has no reader until task .2 adds the ExecuteActivity outbound interceptor (kept for .2); timeout-copy duplication with scheduleNexus noted, not changed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 31295b0399
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -count=1 -json -tags test_dep -p 2 ./common/testing/testpilot/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 2; all 4 findings in another agent's untracked tools/umpire/internal/golden files, none in this task's paths)
- PRs: