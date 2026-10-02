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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
