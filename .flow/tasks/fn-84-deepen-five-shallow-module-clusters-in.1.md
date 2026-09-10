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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
