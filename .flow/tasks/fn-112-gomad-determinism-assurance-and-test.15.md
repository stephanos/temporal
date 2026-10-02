# fn-112-gomad-determinism-assurance-and-test.15 Make TestWatchdogDiagnosticReplayUsesCapturedInputs reliable
## Description
`TestWatchdogDiagnosticReplayUsesCapturedInputs` (added by task 13) fails intermittently: 5 of 20 focused runs at `9216937b8` on darwin/arm64 under host load, found while gating fn-114.16. It runs in the full host gate, so it makes that gate unreliable. Belongs to R9 (end-to-end CLI tests).

**Size:** S
**Files:** the test and the code it drives under `tools/gomad3/cmd/gomad/internal/cli/` or `tools/gomad3/runner/`
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/runner/**]

### Approach
- Reproduce first: run the test focused at least 40 times, with and without CPU load, and retain the failure output. Identify whether the cause is a wall-clock assumption in the test, or a real race in watchdog classification or diagnostic replay.
- A product race is fixed in the product with a deterministic regression. A test-timing assumption is replaced by a condition the test waits for; do not just raise a timeout.
- Do not weaken what the test asserts about captured inputs.

## Acceptance
- [ ] The cause is stated with retained failing output
- [ ] The test passes 100 of 100 focused runs under CPU load on darwin/arm64
- [ ] `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` passes

## Done summary
`TestWatchdogDiagnosticReplayUsesCapturedInputs` passed 100 of 100 focused runs under 16 CPU-load processes and 100 of 100 under 32 on darwin/arm64 (8 CPUs), and the full host gate passed at `25330890b` (exit 0, 45 packages, 213 s).

Cause: a wall-clock assumption in the test. The fixture target reads `/mounted/input`, prints it, and spins, under a 1 s execution timeout. The supervisor measures that deadline from its own start and sends SIGTERM 150 ms before it, so the target had 850 ms of host time to be launched, read, and print. Under load the watchdog fired first. The retained failing output shows `WatchdogTimeout:true`, empty stdout, and `IOROMounts Requests:0`. The old test failed 1 of 40 runs under 16 load processes and 1 of 20 under 64. The two real-executor replays read the same 1 s limit from the manifest, which explains the failures at all three lines fn-114.16 reported. No product race was found.

Fix, in `tools/gomad3/runner/watchdog_replay_test.go` only: the fixture and the two real-executor replays repeat a run whose watchdog fired before the target wrote output, under the next of 1, 2, 4, 8, 16, 32 s. The manifest records the timeout the fixture used. A replay is repeated only when it returned no error, diverged on `stdout.full_sha256`, and its observed stdout is empty. No assertion changed. A temporary 150 ms first timeout (reverted before the commit) drove the fixture and both replays through the repeat path, and 8 repeats occurred and recovered across the 32-process and 64-process runs.

Limits:
- Under 64 load processes, 1 of 20 runs of the new test failed with a different signature: `execution.Run` returned the untyped error `supervisor protocol requires exactly started then final reports` because the supervisor did not finish inside the 1 s deadline. The old test makes the same call. The test does not repeat on it, because repeating on an untyped error would hide a real execution failure. Follow-up candidate: a typed error in `runner/internal/execution` for a deadline that passed before the supervisor reported.
- The 100 runs under 16 load processes needed no repeat, so that count alone does not separate the new test from the old one. The 32-process runs and the forced run do.
- `cmd/gomad/watchdog_replay_e2e_test.go` (outside this task's Touches) replays a real watchdog target under a 2 s timeout and may share the assumption. Not reproduced.
- Review left one open P3: the fixture and the replay helper each carry their own repeat loop, and the replay loop depends on `replayDivergence` reporting `stdout.full_sha256` for an empty-output watchdog run.
- Not run: linux/amd64 (no native host), root `make lint-code-fast` (cannot typecheck the nested module), `make -C tools/gomad3 validate` as a separate command.

Evidence: `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-15/` (`gates.json`, `red-failing-output.txt`, `forced-escalation.txt`, `recovered-preemptions.txt`, `residual-64-load.txt`, `test-host-packages.txt`, `impl-review-receipt.json`).

GATE_SKIPPED:unittest:green-receipt 92cb85de - baseline reused from prior post-gate pass

Impl-review: SHIP, round 1 (claude:claude-fable-5-1:high, same model family as the writer).

stage: impl-review - ran [2026-10-02T21:50Z..2026-10-02T21:54Z]
## Evidence
- Commits: 25330890ba97b128bce8a93f3ad3d9427d9351b7, 134369ef6e891ed89c2f1748997028598511e7fa, 51073681f22414185d96e5b306fee6de898f44eb
- Tests: GATE_SKIPPED:unittest:green-receipt 92cb85de - baseline reused from prior post-gate pass, runner.test -test.run ^TestWatchdogDiagnosticReplayUsesCapturedInputs$ -test.count=1 -test.v: 100 of 100 under 16 load processes and 100 of 100 under 32 load processes at 25330890b (darwin/arm64, 8 CPUs), GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host: exit 0, 45 packages ok, 213 s at 25330890b (receipt 25330890), .toolchain/bin/go vet -tags test_dep ./runner: exit 0, gofmt -l runner/watchdog_replay_test.go: no files
- PRs: