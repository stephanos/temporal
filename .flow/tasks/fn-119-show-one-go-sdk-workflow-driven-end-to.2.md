---
satisfies: [R2]
---
# fn-119-show-one-go-sdk-workflow-driven-end-to.2 Route workflow-scheduled activity attempts to the Driver's activity interpreter

## Description
Let the Driver answer each attempt of an activity its own workflow scheduled, as the Case's path says: complete, fail retryably then complete, or withhold its answer so the server times it out. Withholding is a new general attempt instruction (spec Architecture, amended at planning); today an attempt with no enabled instruction is refused non-retryably (`interpreter.go:263-292`, `sdk.go:230-234`), which is why the standalone `startToCloseTimeout` Case stands `unsupported` in `model/cases/manifest.json`.

**Size:** M
**Files:** `common/testing/testpilot/temporal/profile.go` (`carried` :265), `temporal/worker/{driver,sdk,interpreter}.go`, `internal/delivery/{carrier,activity}.go`, tests.
**Touches:** [common/testing/testpilot/temporal/**, common/testing/testpilot/internal/delivery/**, common/testing/testpilot/internal/execution/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**]

### Approach
- Activity reservations ride only on `StartActivityExecution` today (`temporal/profile.go:265` `carried`, `worker/driver.go:413-418`, `delivery/carrier.go:60-67`, `delivery/activity.go:15`). Add a workflow-start carrier for activity entrypoints, propagate the routing header through an `ExecuteActivity` outbound interceptor (mirror `ExecuteNexusOperation` at `sdk.go:122`), and dispatch attempts to `executeActivity` (`interpreter.go:233`), where attempt N runs the instruction at its ordinal.
- Timeout: add a withhold-answer attempt instruction (spelling settled here) to the Testpilot proto, preparation (`internal/execution`) and the Profile's instruction list; the Driver holds the attempt until its context is cancelled by the server's start-to-close timeout (block on `ctx.Done()`, no sleep). fn-118.3 also edits the Testpilot proto; whichever lands second rebases.
- Replay with `worker.WorkflowReplayer` over partial history (memory: workflow replay must complete unfinished admissions once).

### Investigation targets
**Required:**
- `common/testing/testpilot/temporal/worker/interpreter.go:200-260`
- `common/testing/testpilot/temporal/worker/driver.go:400-425`
- `common/testing/testpilot/internal/delivery/carrier.go`
**Optional:**
- `.flow/memory/bug/runtime-errors/workflow-replay-must-complete-2026-09-05.md`

### Quick commands
```bash
go test -count=1 -tags test_dep ./common/testing/testpilot/...
```

### Execution constraints
- Existing standalone-activity and Nexus deliveries unchanged.
## Acceptance
- [ ] Attempts of a workflow-scheduled activity are answered per instruction: complete, retryable failure then complete, and withheld until the server times the attempt out, against the in-process cluster.
- [ ] A Profile that does not list the withhold instruction rejects a Case using it at preparation, naming it.
- [ ] Replay over partial history is covered.
- [ ] Existing Cases unchanged; Testpilot tests and lint-code-fast pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
