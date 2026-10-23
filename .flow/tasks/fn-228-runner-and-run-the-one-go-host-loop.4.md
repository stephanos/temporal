---
satisfies: [R4]
---
# fn-228-runner-and-run-the-one-go-host-loop.4 After .3: Temporal Driver as act-and-observe; live activity slice through umpire explore

## Description
Reduce the retained temporal/* driver to a runner.Driver: send the proposed request over the generated API, collect Observations with poll interval, read timeout and backoff from instruction fields, capture and redaction as field paths, RPC completion and evidence well-formedness checks; surface SDK callbacks from the in-process Go worker. Run the live activity slice (start, retry then completion, cancellation, host refusal) through umpire explore with the Run as the only artifact; make umpire-check-model-slice invokes it. Read timeout without Model acceptance is a DriverFault event; malformed evidence refused before Judge.

### Required integration contract

Integrate the incremental Exchange contract and Model-owned continue_polling signal with the real Temporal Driver. Start uses bootstrap and same-invocation World admission from fn-227; callbacks are serviced while the RPC/poll is active. Prove retry/completion/cancellation and an intermediate read with intervening callback before completion. Record the real Run and replay all protocol events without a live server. Preserve guest embedding/freshness and keep poll timing/backoff as transport mechanics.

### Carried from fn-228.3 (2026-10-23)

fn-228.3 retired four live tests that built the deleted commands; see `.flow/tmp/fn-228.3/retired-coverage.md`. This task owns their live replacements: `umpire explore` run as a separate process with the Temporal Driver, which exits 0 for an Explained Run and leaves the Run as the only artifact; resource provisioning and cleanup in the Temporal Driver; a live control where an unreachable endpoint is a DriverFault and exits 2; and a real callback-bearing Run recorded and then replayed without a server.
## Acceptance
- [ ] Temporal Driver implements runner.Driver without interpreting Temporal messages
- [ ] Live slice passes through umpire explore; Run is the only artifact
- [ ] DriverFault and malformed-evidence controls

## Done summary
Implemented R4: the Temporal Driver (`common/testing/testpilot/temporal/act`) acts and observes for the Runner. It sends the proposed request bytes unchanged to the WorkflowService method taking that message, records exact response bytes, treats a server status as evidence and an unanswered call as a Fault, repeats reads per the Model's `continue_polling` with the instruction's poll interval, read timeout and backoff, and never decides completion itself. The Realization gained one instruction field, `ObserveActivity.worker_tasks` (canonical Model and behavioral goldens unchanged; one expansion snapshot gained two lines; guest rebuilt): the in-process SDK worker may take a task only after a continued read, at most that many times, with reads paused while a poll or task is outstanding. Callbacks carry the task actually polled and Answer's reply goes back through the SDK. Close refuses held polls, cancels the admitted one, closes exchanges, terminates what the Driver started, releases held tasks and stops the worker, joining all work. The Driver never fabricates worker replies; a late SDK response is a teardown fault only for tasks the Model left unanswered. Redaction field paths (validated against descriptors) reach refused calls and the Run's own world. The Driver is registered with `umpire` via `UMPIRE_TEMPORAL_*`; invalid configuration exits 3.

`TestTestpilotModelSliceThroughUmpire` runs `umpire explore` as a separate process: Complete (namespace created and deleted), Cancellation (accepted and NotFound after completion), RetryThenComplete, Fail (exit 0), HostRefusal (exit 2, activity TERMINATED by Close), UnreachableEndpoint (exit 2, DriverFault) and ReplayWithoutServer (a callback-bearing Run replays to exit 0, the refused Run to its exact refusal line). The Run files are the only artifacts. These replace the four live tests retired in fn-228.3. DriverFault and malformed-evidence controls refuse before Judge; every wait has timeout and Close controls.

Review: three fresh-context rounds (Opus, same family): NEEDS_WORK (poll-gate race reproduced 3/3, Close leaving held polls, redaction missing refused calls, missing wait tests, invented replies, broad boundary, Temporal prose under model/umpire, config) -> SHIP with N1 (flaky Close test) and N2 (late reply misclassified) -> both fixed. A 200-run race check found a fake-Driver race in fn-228.3's test (12/200), fixed in the test; production Runner unchanged; 300/300 after.

Gates: focused and -race tests, lint and vet, `make umpire-check-model-slice` rc 0, `make umpire-check-model-gaps` rc 0 on the changed guest, default and required `make model-check` rc 0. `UMPIRE_BACKENDS=require make model-time` rc 0 at c3c95eb00b (shared receipt with fn-241.1): gate 2054.281 s against the 60 s soft target and inner loop 22.546 s against 10 s, at load average 37-41; not steady state. Carried to .6: retire or map `TestTestpilotModelSlice`, optionally fold `act` into the parent package.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9ddd74f15b, 780d249590, 9da5c6f44c, ae2ae4a39a, d7a144ea4f, 510b0a2f12, 22c20a7eae, aa7c23e6f1
- Tests: UMPIRE_BACKENDS=require make model-time, make model-check, make umpire-check-model-slice, make umpire-check-model-gaps, go test -race ./common/testing/testpilot/temporal/act/...
- PRs: