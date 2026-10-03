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
Route workflow-scheduled activity attempts to the Driver's activity interpreter, and add the withhold-answer attempt instruction (R2).

What changed
- Protocol (additive): `Instruction.activity_attempt_withholding` (`ActivityAttemptWithholding {}`, field 12) and `ACTIVITY_ATTEMPT_RESPONSE_WITHHELD = 7`; doc of `ActivityAttempt.activity_run_id` says a workflow-scheduled activity's attempts name the workflow's run. New Opcode `ActivityAttemptWithholding` (MaxOpcode moves); outcome table admits SUCCEEDED+WITHHELD.
- Preparation: an activity entrypoint a workflow schedule command reaches (activity type + task-queue role) is reserved only by the carrier that reserves that workflow; the carrier plan routes the command to the activity's first attempt (`appendActivityRoutes`). A second schedule of the same activity entrypoint rejects. Carrier policies may hold three shapes. An unauthorized instruction is now rejected naming its arm (`instruction activity_attempt_withholding the Profile does not authorize`).
- DeriveProfile: StartWorkflowExecution also carries ActivityEntrypoint, counted from the attempts of workflow-scheduled activities only; StartActivityExecution keeps the others.
- Ledger: `scheduled_activity` route kind, attempts per schedule command kept for the bundle's life, `PrepareActivity`, `ScheduledEntrypoint`, `AdmitScheduledActivity` (attempt N -> Nth reservation, replay, undeclared, workflow-run and activity-ID conflicts, stale after close); `ParentTerminal` releases the attempts after the last delivered one as not needed.
- Worker: Session prepares one header entry per routed schedule command at workflow admission and indexes itself under it; outbound `ExecuteActivity` interceptor writes it (unrouted commands pass through for ordinary workers); inbound admission picks the scheduled path by header and checks the entrypoint's type/queue before consuming; withholding blocks on ctx.Done and only a deadline counts as declared; not-needed attempts settle with the workflow's run id.

Decisions
- Spelling `ActivityAttemptWithholding` (noun form beside Failure/Cancellation); response `WITHHELD`.
- A scheduled activity's `activity_run_id` is the scheduling workflow's run id: the server names no activity run, and the scheduler's per-activity run consistency check keeps working unchanged.
- Unused attempts are released when the workflow closes (ParentTerminal), mirroring the standalone closure watch; an activity with no delivered attempt keeps its reservations (never-seen), as for standalone.
- Proto regenerated with the repo's protogen pipeline in an isolated worktree with a Linux goimports (checked-in one is macOS); catalog identity moved b16586d1 -> e04c84c8, pinned Runs and receipt goldens re-recorded via `make umpire-rerecord-pinned-runs`.
- The live test is a hand-built Case in tests/ naming no example (generic `scheduled-activity-*` names).

Tests: SDK test-environment runs (complete, retry then complete, withheld until timed out, not-needed release), WorkflowReplayer over partial then full history, ledger and preparation unit tests, Profile derivation, live in-process cluster test (3 paths pass), full Go tooling suite exit 0, lint-code-fast 0 issues, testpilot protocol check exit 0.

Review: Fable (claude-fable-5-1, high) round 1 SHIP. Deferred P3: (1) addScheduledRoutes/removal duplicate the Nexus route-index code; (2) prepareActivityDispatchesLocked lacks the MaxActivations dispatch-map bound the Nexus path has (route capacity still bounds it); (3) a StartActivityExecution request that itself carries the scheduled-activity reserved header name is not refused by the carrier, so its attempts would be misrouted and the Run fail without a naming diagnostic. FYI: a withheld attempt ends at the SDK deadline (start-to-close/schedule-to-close), not at a shorter heartbeat timeout.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5c568883e7
- Tests: go test -v -count=1 -timeout 20m -tags 'test_dep integration' ./tests -run '^TestTestpilotScheduledActivityAttempts$' (exit 0), make umpire-rerecord-pinned-runs (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -count=1 -json -tags test_dep -p 2 ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 0), make umpire-check-testpilot-protocol (exit 0)
- PRs: