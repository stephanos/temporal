---
satisfies: [R1, R6, R7]
---
# fn-84-deepen-five-shallow-module-clusters-in.1 Worker outage module owned by the registry

## Description
Move the deliberate-outage decision (R1) out of the worker Driver's definition preparation, the Session's dispatch and the registry's four flip sites into one `outage` module the registry owns. Validate and Open share one `OutagePlan`; the fault tests drive the production path through the module's interface. First task because the code moves largely intact inside one package and it is the spec's early proof point.

**Size:** M
**Files:** `common/testing/testpilot/temporal/worker/outage.go` (new), `outage_test.go` (new), `driver.go`, `session.go`, `registry.go`, `fault_test.go`, `README.md`; `common/testing/testpilot/temporal/README.md`; `tools/umpire/CONTEXT.md`
**Touches:** [common/testing/testpilot/temporal/worker/**, common/testing/testpilot/temporal/README.md, tools/umpire/CONTEXT.md]

### Approach
- Read the worker README's outage paragraph first; it is the contract the module must keep true, and its sentence about a refused or incomplete transition being a failed outcome plus an invariant diagnostic is draft EVD-20 in package prose.
- Baseline before moving anything: run the fault suite and the two live worker-outage tests and record the exact diagnostics and Run Event sequences (the memory entry on behavior-neutral refactors is the reason).
- Shape `outage.go` after the in-package state-with-lock modules (`reservation.go`, `carrier.go`) and reuse `contextMutex` from `registry.go`; the delivery `Ledger` and its test file are the model for a state machine tested only through its verbs.
- `PlanOutages` absorbs the queue resolution in `driver.go` (populate step) and the registered-queue check (validation step); `OutagePlan.Requires()` replaces the `hasFault` to `dedicated` input; delete `faultQueues`/`hasFault` from the program definition. Name it `OutagePlan` everywhere: `Plan` is fn-82's word for the model-level test and an avoided word for Program.
- `Begin` owns the flip and the fatal-failure suppression window under the registry lock; `Settle` is the blocking stop or resume `faultEffect.Wait` drives; `Restore` replaces the resume-then-release ordering in `Close` and the resume in `release`; `Stopped(ctx)` is the observation the tests need and takes the registry lock the way `stoppedQueues(ctx)` does today.
- Keep the non-dedicated refusal returning the same `ErrUnsupportedOperation`; keep the server Session's `InjectFault` refusal and its comment untouched; keep the scheduler-side success-only `FAULT_INJECTED` emission as is and pin that a `Settle` error yields zero fault events.
- Keep today's post-failure state: `finishResume` marks the group stopped on error and `release` deletes a dedicated group outright, so after a failed resume `Restore` still releases the hold and the group is gone. Do not add a failure marker the registry does not have.
- Delete the test-only `stopWorker`/`resumeWorker`/`transition` facade and rewrite `fault_test.go` onto `Begin`/`Settle`/`Restore`/`Stopped`; delete `TestFaultValidationAgreesWithOpen`. The runtime tests' registry-emptiness reads are unrelated to faults and stay.
- Docs: rewrite the worker README outage paragraph in module terms and add the previously undocumented refusal of a fault queue no entrypoint registers; add one fault-routing sentence to the composite README mirroring the server README's `Reserve` sentence; add an `Outage plan` glossary entry under Testpilot execution that separates it from the Fault instruction and the Run Event and marks it Driver-internal.

### Investigation targets
**Required** (read before coding; line refs are at HEAD ebb94a44e):
- `common/testing/testpilot/temporal/worker/driver.go:137-166, 216, 284-291, 381-388` — admission half
- `common/testing/testpilot/temporal/worker/session.go:18-29, 131-264, 288-323` — carrier fields, faultEffect, InjectFault, Close ordering
- `common/testing/testpilot/temporal/worker/registry.go:71-95, 110-126, 325-360, 370-436, 456-500, 526-575` — contextMutex, group state, test facade, transitions, release, fail suppression
- `common/testing/testpilot/temporal/worker/fault_test.go` — the 16 tests to rewrite; note the 17 private-state reads and the hand-patched definition at 508-510
- `common/testing/testpilot/internal/execution/scheduler.go:783-791` — where FAULT_INJECTED is emitted (succeeded outcome only)

**Optional:**
- `common/testing/testpilot/temporal/internal/delivery/ledger.go` and `ledger_test.go` — interface-tested state machine pattern
- `common/testing/testpilot/temporal/driver.go:94, 112-117, 246-252` and `common/testing/testpilot/temporal/server/session.go:44-52` — routing and refusal to leave alone
- `common/testing/testpilot/temporal/worker/runtime_test.go:43` — registry reads that are out of scope
- `.plans/UMPIRE4_SPEC.md` EVD-20 (draft), MOD-13

### Key context
- fn-82 task .7 renames `plan.Context()` to `Kind()` and the facade's opcode names; this task starts after fn-82 closes, so use the landed names.
- New identifiers must avoid the retired-vocabulary tokens (`Capability{Contract,Provider,Connector}` among them); `CapabilityBridge` is fine.
- Preserve existing comments when moving code (global instruction).
## Acceptance
- [ ] `outage.go` exports `PlanOutages`, `OutagePlan.Requires`, `Begin`, `Settle`, `Restore`, `Stopped(ctx)`; `driver.go` calls `PlanOutages` once and both `Validate` and `Open` read the returned plan; `faultQueues`/`hasFault` are gone from the program definition
- [ ] `stopWorker`/`resumeWorker`/`transition` and `TestFaultValidationAgreesWithOpen` are deleted; no fault test reads `registry.groups`, `.stopped` or `.failure`
- [ ] new interface tests: unregistered fault queue refused at plan time with today's message; non-dedicated lease returns `ErrUnsupportedOperation`; `Begin` after `Close` began errors and flips nothing; `Begin` twice in the same direction conflicts; `Restore` after a failed resume still releases the hold and the dedicated group is gone; `fail` inside a stop window is swallowed, outside it is not; a `Settle` error yields a non-succeeded outcome and zero `FAULT_INJECTED` events
- [ ] focused suite: `go test -count=1 -tags test_dep ./common/testing/testpilot/temporal/worker/` green; `make umpire-check-live-tests` green with the worker-outage tests' Verdicts and Run Event kinds unchanged from the recorded baseline
- [ ] worker README outage paragraph, composite README fault-routing sentence and `CONTEXT.md` `Outage plan` entry updated; the documentation gate in `tools/umpire/regression/ci_workflow_test.go` passes
- [ ] `make lint-code` clean; `make umpire-check-regression` green; task summary records the baseline comparison
## Done summary
Worker outages now live in one `outage` module (`common/testing/testpilot/temporal/worker/outage.go`) that the registry owns. `PlanOutages` runs once inside definition preparation, so `Validate` and `Open` read the same `OutagePlan`. `OutagePlan.Requires` feeds the registry's dedicated input. `Outage.Begin`/`Settle`/`Restore`/`Stopped` replace the lease transition facade, the Session's resolution of queues and directions, and the resume-then-release ordering in `Session.Close`. `faultQueues`/`hasFault` are gone from the program definition. `stopWorker`/`resumeWorker`/`transition` and `TestFaultValidationAgreesWithOpen` are deleted. The fault tests (`fault_test.go`, new `outage_test.go`) drive only the module interface and never read `registry.groups`, `.stopped` or `.failure`. The worker README, the composite README and the `Outage plan` glossary entry were updated.

Tests for each AC error case:
- unregistered fault queue and workerless Program: `TestPreparedDefinitionPlansOutages`
- non-dedicated lease: `TestFaultTransitionsRequireADedicatedGroup`
- Begin after Restore began: `TestFaultBeginAfterRestoreBeganFlipsNothing`
- same-direction conflict: `TestFaultStopAndResumeKeepTheSameRegistration`
- Restore after a failed resume: `TestFaultRestoreResumesBeforeReleasing`
- fail inside and outside the stop window: `TestFaultStopSuppressesTheFatalPath`
- Settle error gives a non-succeeded outcome and no FAULT_INJECTED, run end to end through the scheduler: `TestFaultSettleErrorRecordsNoFaultEvent`

`TestFaultBeginAfterRestoreBeganFlipsNothing` and `TestFaultSettleErrorRecordsNoFaultEvent` were checked red against a deliberately broken implementation, then restored.

Baseline comparison (R6): before any edit, the two live worker-outage tests were run with a `go test -overlay` that logs the Run status, cleanup, Verdict, rule terminal states, diagnostics and the full ordered Run Event kind/source-id list. The same overlay after the change produced a byte-identical dump: Completed, cleanup Succeeded, Satisfied (worker-outage-order resumed, workflow-completed completed), no diagnostics, FaultInjected at n0 and n2. Logs are in `.flow/tmp/fn-84.1-baseline-live-outage.log` and `.flow/tmp/fn-84.1-after-live-outage.log`.

Gates:
- `make lint-code`: 161 issues, the inherited baseline, from a clean-cache run (14506 before processing), none in touched files.
- `make umpire-check-regression`: exit 0, 571 Lean jobs, 9 passing live identities. The first attempt failed because the Lean `Protobuf` dependency had never been built on this host (`unknown module prefix 'Protobuf'` in `umpire-check-testpilot-protocol`). Running `lake build Protobuf`, which writes build artifacts only, fixed it.

Decisions (autonomous):
- `PlanOutages` takes the entrypoint plans, the roles map and a registered-queue set instead of the spec sketch's instruction plans and roles slice. These are the values `prepareDefinitionResources` already holds, and `DeclaresFault` takes the same list.
- Moving fault-role resolution after registration assembly changes which error a Program with two independent defects returns first. Both errors are rejections at Validate/Open, and no test or fixture tells them apart.
- `Begin` refuses with `ErrClosed` once `Restore` has begun (a `restoring` flag under the registry lock). This makes the spec's "Begin after Close began flips nothing" hold mid-Close and closes a narrow race where a concurrent stop could land after Restore chose which groups to resume.
- `InjectFault` still calls `OutagePlan.resolve` before taking the Session lock, so the order of its dispatch-rejection errors and the queue named in the diagnostic detail stay as they were.
- `workerLease.release` stays for pooled holds, which the runtime tests use.

Follow-up (recorded as `CONSIDER(umpire)` in `session.go`): a Settle failure records no Driver invariant diagnostic, only the failed outcome, although the README says both refused and incomplete transitions do. This was already true before the change; a probe Run confirmed it.

stage: impl-review - ran [2026-09-12T19:2x..2026-09-12T19:29:41Z] claude backend, SHIP first round (2 P3 polish findings: README rewrap applied; helper renamed; resolve-twice kept deliberately for error precedence)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b2c0686e7708e02f4f5582f306f876793ee2858e, d0863f26f00cf740397b83c002cb934682d37914
- Tests: go test -count=1 -race -tags test_dep ./common/testing/testpilot/temporal/..., go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotWorkerOutage' (with a -overlay dump of Verdicts and Run Events; diff against pre-edit baseline: identical), make lint-code GOLANGCI_LINT_FIX=false (after go clean -cache): 161 issues, 14506 before processing, none in touched files, make umpire-check-regression: exit 0, 571 Lean jobs, 9 passing live TestTestpilot identities
- PRs: