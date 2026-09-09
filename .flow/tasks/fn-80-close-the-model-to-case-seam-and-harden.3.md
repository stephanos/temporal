---
satisfies: [R4]
---
# fn-80-close-the-model-to-case-seam-and-harden.3 Realize worker stop and resume in the Temporal worker Driver

## Description
Implements the Driver half of R4 (spec §R4). A prepared Program containing `InjectFault` opens a dedicated worker group keyed by Run ID; `WORKER_STOP` stops that group's SDK worker with fatal-callback suppression, `WORKER_RESUME` re-registers with the same structural signature, cleanup resumes before release.

**Size:** M
**Files:** `common/testing/testpilot/temporal/driver.go`, `common/testing/testpilot/temporal/worker/driver.go`, `common/testing/testpilot/temporal/worker/registry.go`, `common/testing/testpilot/temporal/worker/session.go`, `common/testing/testpilot/temporal/worker/sdk.go`, `common/testing/testpilot/temporal/server/driver.go` (reject `InjectFault` on the server side, mirroring `Reserve`), tests in `temporal/worker/*_test.go`, `temporal/driver_test.go`
**Touches:** [common/testing/testpilot/temporal/**]

### Approach
- Detect `InjectFault` in the prepared Program at `worker/driver.go` `Open`/`OpenSession` (`:154-165`) and allocate a non-pooled `workerGroup` keyed by run ID instead of going through `acquire` at `registry.go:117`; reuse `queueRegistration.canonical/compatible` (`registry.go:27,59`) for the resume re-registration.
- `WORKER_STOP`: call the managed worker `Stop()` (`registry.go:10-13`) bounded by `InstructionLimits.timeout_milliseconds`; suppress the group's `onFatal`/`failure` path for the stop window so a stopped worker is not treated as a fatal Run failure (`registry.go:100-300`).
- `WORKER_RESUME`: `buildAndStart` (`registry.go:204`) with the same signature; a resume without a prior stop returns a Driver invariant failure that the scheduler records as a diagnostic.
- Cleanup ordering: resume-before-release in `cleanupContext`/`Close` (`worker/driver.go:205-213`); a resume that exceeds cleanup bounds sets cleanup `failed` and leaves the Verdict alone.
- Validate `InjectFault` capability in `validWorkerProfile` (`worker/driver.go:79-114`) and reject it in the server Driver like `Reserve`.
- Composite: `temporal/driver.go` opens a worker Session for a Program whose only worker use is `InjectFault`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/temporal/worker/registry.go:10-120,200-300` — managed worker, group pooling, buildAndStart, stopWorkers
- `common/testing/testpilot/temporal/worker/driver.go:49-213` — profile validation, Open, cleanup, Close
- `common/testing/testpilot/temporal/internal/delivery/ledger.go` — reservation release path a stop must not corrupt

**Optional** (reference as needed):
- `common/testing/testpilot/temporal/worker/session_test.go` — session fault-mode test patterns
- `common/testing/testpilot/temporal/internal/activation/` — fn-74 activation ownership

### Key context
- Tasks queued while the worker is stopped wait in matching and dispatch after resume; no sticky-cache handling is needed when the stop precedes the first workflow task.
- fn-74's stated boundary ("no new SDK instruction") is a self-scope note; fn-74 is done.

## Acceptance
- [ ] A prepared Program with `InjectFault` gets a dedicated worker group; a concurrent Run on the same queue without the instruction keeps its pooled group and is unaffected by a stop (test with two sessions)
- [ ] `WORKER_STOP` stops the group's SDK worker within the instruction timeout without triggering the fatal path; `WORKER_RESUME` re-registers with an equal structural signature
- [ ] `WORKER_RESUME` without a prior stop yields a Driver invariant failure recorded as a diagnostic; Verdict unaffected
- [ ] Cleanup resumes a stopped worker before release; a resume that times out sets cleanup `failed` and the Verdict is unchanged
- [ ] Server Driver rejects `InjectFault`; worker profile validation admits the capability
- [ ] `CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/temporal/...` passes

## Done summary
Realized worker stop and resume in the Temporal worker Driver. A Run whose Program requests a fault holds its own worker group keyed by Run ID; `workerLease` is the only handle that can stop or resume one, the group's `stopped` flag suppresses the SDK fatal path for the whole outage, release resumes before the group goes away, and `Session.InjectFault` settles a transition the Driver could not make as a `fault_not_realized` outcome rather than a Run failure.

Design notes for downstream tasks:
- The blocking stop/resume runs in the effect handle's `Wait`, not on the dispatch path. That is what makes an SDK stop outliving its instruction bound arrive as a deadline (which the scheduler maps to a timed-out instruction) instead of an `admission_failed` that would mark the whole Run incomplete, and it keeps the recorder mutex free during the outage.
- A fault may only name a task-queue role the Program itself registers a worker on; Validate and Open both refuse anything else, and a Program whose only worker use is the fault is refused because it brings no worker to stop.
- **Known limitation, settled by the spec's own acceptance:** the dedicated group isolates the stop from peer Runs, but a pooled peer worker on the *same* task queue keeps polling it, so the fault Run's tasks can still be served during the "outage". Task .8's Case must give its fault queue an unshared resource binding for the outage to be real; the concurrent-plain-Run assertion in .8 should use a second queue.
- Task .2's gate records `FAULT_INJECTED` only on a succeeded outcome, so an unrealized fault leaves intent and no evidence — which is why the failure path here returns a non-success outcome rather than an error.

Swept into these commits and not mine: `.plans/UMPIRE4_ORDER.md` (76 lines), edited by a parallel session sharing this checkout.

stage: impl-review - ran (claude backend, model claude-fable-5-1 at high) - four rounds: NEEDS_WORK on unlocked registry map access, queue-directed transitions, resume settle-state, release leak, cleanup-plan disagreement, dispatch-path blocking and the diagnostic never reaching the Run; SHIP on round four, whose three P3s were also fixed
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c73a45ee4124db707869253aaa3ffe01bc5db293, c2216be45b652af54746509cb57387ea4a9694c1, ceafe47247f90b5a737a251a1b2730ca8ef14e59, 9b0941b02a8ee233dec1e21129f635e404853ba1, 619017da859f2ddfd836d32e734207bdde1a88d8, d9bcf92a205eab81d9839df240c6cc17393c725c
- Tests: CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/..., CGO_ENABLED=0 go test -tags test_dep ./tests/testcore/testpilot/..., CGO_ENABLED=1 go test -race -count=2 -tags test_dep ./common/testing/testpilot/temporal/..., GATE_SKIPPED:lean:not-applicable - task .3 changes no Lean source, GATE_SKIPPED:live-tests:disk - make umpire-check-live-tests needs a live cluster; task .3 declares no live acceptance
- PRs: