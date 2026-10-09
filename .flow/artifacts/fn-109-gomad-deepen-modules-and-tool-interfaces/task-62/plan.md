# Task 62 frozen control plan

Admission BASE is `7b75ae312a6641c6b00afbb63f44ae9a9dd0c2b2`. Original preservation BASE remains `951c5516e9e7b3066e7e069adda9565cfd68844c`. The root retained the blocked receive and diagnostic SIGQUIT in `../combined-57-59/hang-diagnostic.md` and `full-ordinary-runner.stdout`. No Go execution has occurred in this workspace. The root serializes the shared Go lane and owns review, commits and lifecycle.

## Fixed expectations

| Control | Expected result |
| --- | --- |
| Actual start channel is closed | Startup succeeds without consuming completion. |
| Existing errorPreparer returns `progress-start preparation sentinel` | Real exploreWith completes before Execute; startup returns an error containing the sentinel, executor.started remains open. |
| Completion carries nil before start | Startup returns `runner completed before target execution started: <nil>`. |
| Only startup deadline is ready | Startup returns `target execution did not start before startup deadline`. |
| Helper bypasses completion and deadline | Completion/nil/timeout controls fail through their bounded outer watchdog; cleanup closes their start channels and joins the helper goroutine. |
| Actual BASE fixture with healthy preparer | Bounded outer test watchdog observes the original startup hang on linux/arm64. Timeout or diagnostic abort is incomplete coverage. |
| BASE private fixture with errorPreparer substitution | The same bounded watchdog observes the startup hang despite the real preparation sentinel. Execute never runs. |
| Final private fixture with the identical errorPreparer substitution | Ordinary nonzero test termination contains the sentinel through the startup assertion; no timeout or Execute. |
| Final healthy fixture | Ordinary nonzero test termination exposes existing unsupported-host preparation failure. This remains a red gate. |

## Source boundaries

Only the periodic-progress fixture changes in runner_test.go. Register sync.OnceFunc release cleanup before launch, create a startup timer from config.OverallTimeout + config.TerminateGrace, and wait on started/completed/deadline through a test-local helper. Keep the separate time.Second heartbeat timer, 1ms progress interval, buffered callback, two ProgressRunning/Running==1 updates, successful release position and nil completion assertion. New controls and the helper live in progress_start_test.go. No production, profile, platform, dependency or skip changes.

## Matched execution

After the root grants the lane, record a pre-edit healthy-fixture diagnostic on BASE and the private errorPreparer BASE probe. Add the fixed controls with a private/extracted original receive for the intended failing reproduction, then repair the helper and fixture. Run final controls and both healthy/sentinel fixture probes with the same pinned stock Go1.27.1 and `-tags test_dep -count=1`. Retain plain numeric exits, elapsed time, terminal state and hashes of exact source, tool binaries and raw output. Run the helper completion/deadline-bypass sensitivity on private final source, with cleanup joining all started control goroutines.

Run affected stock vet, gofmt check, unfiltered configured Runner lint, standalone errortype and check-only `make lint-code-fast ... GOLANGCI_LINT_FIX=false`. Root runs the combined full ordinary Runner once after integration and measures original-base integrated lint. Existing RED60 full lint and unsupported-host failures remain open. Native fn149/fn128 stay deferred and unverified. No formal SHIP or Done follows from this source correction.

Tier: implementer gpt-6.1-sol at high; judge unavailable(no_key), project explicit tier retained; execution telemetry unobserved.
