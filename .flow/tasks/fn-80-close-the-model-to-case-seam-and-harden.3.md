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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
