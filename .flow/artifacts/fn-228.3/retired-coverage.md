# fn-228.3 retired coverage and deferrals

R3 deletes umpire-run, umpire-assess, umpire-fuzz, umpire-replay and umpire-repeat. Four live
tests (`make umpire-check-live-tests`, `./tests -run '^TestTestpilot'`) built those commands and
were retired in e98ec44df6. They cannot pass without the deleted commands, so each one's coverage
is listed here with the task that replaces it.

## Retired live tests

| Retired test (file) | Behaviour it covered | Replaced by |
| --- | --- | --- |
| `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint` (tests/testpilot_umpire_run_test.go) | Black-box mode: a separate process given only a fixture and addresses ran a Nexus `asyncCompletion` Case against the test cluster; exit 0, "run Completed", "verdict Satisfied"; `--create` provisioned and deleted its namespace and Nexus endpoint; `--record` capture for umpire-repeat. | .4: the live activity slice through `umpire explore` as a separate process with the Temporal Driver; exit 0 for an Explained Run, the Run as the only artifact. Resource provisioning and cleanup belong to the Temporal Driver in .4. Nexus Cases have no Model or Driver yet (spec Boundaries: no Nexus Drivers); .6 records that gap in its coverage map. |
| `TestTestpilotUmpireRunRejectsAnUnreachableEndpoint` (same file) | An unreachable endpoint exits 3 with no verdict: infrastructure kept separate from an inconclusive Verdict. | .4: an unreachable endpoint is a DriverFault event and `umpire explore` exits 2 (Refused). This is pinned generically by runner tests (`TestDriverFaultsRefuseTheRun/start`, `TestWaitsEndInDriverFaults/driver_unavailable`, `TestUmpireCommandExitCodes`); .4 adds the live control. |
| `TestTestpilotNexusControlReplaysThroughTheCommand` (tests/testpilot_nexus_control_case_test.go) | End to end through umpire-run and umpire-replay: the Nexus negative control recorded a violated Run (exit 1); replay admitted it, replayed it offline, reproduced it on two fresh Runs, minimized it through the IR bridge and wrote a proposal. | Replay and reduce are runner modes. Offline replay, tamper rejection and reduction are pinned by fake-Driver tests in .3 (`TestReplay*`, `TestReduceReportsTheShortestReproductionFound`). .4 records a real callback-bearing Run and replays it without a server (spec "Run and replay"). The Nexus control and IR-bridge minimization have no Runner equivalent; .6 maps them (the IR bridge/explore/replay packages retire there). `TestTestpilotNexusControlForgedCompletionIsViolated` stays in the same file. |
| `TestTestpilotAssessRecordedRuns` (tests/testpilot_assess_test.go) | Offline assessment: umpire-assess judged a recorded Run under the `local-ephemeral` Evaluation Profile, published one receipt idempotently ("already-published" on the second run), left the Run unchanged, and rejected the pinned negative control (`verdict-violated`, `known-gap-blocking`). | Profiles and receipts are removed by spec (no profiles, receipts or provenance). Offline judgment is `umpire replay` (exit 0/1/2). Idempotent, never-overwrite publication is pinned by `TestConcurrentPublicationPublishesOnce`, `TestPublicationNeverOverwrites` and `TestRepeatRecordsIndependentRunsOfTheSameSeeds` (.3). The negative-control rejection maps to .6's legacy-failure and coverage mapping. |

`buildUmpireCommand` moved to tests/testpilot_exploration_test.go for its remaining user (umpire-ir-bridge).

## Removed Make wiring

- `umpire-rerecord-pinned-runs` and `UMPIRE_PINNED_RUNS`: the only pinned Run's probe was
  `TestAssessWithAModelReproducesTheLiveAssessment` in the deleted umpire-assess. The record
  `common/testing/testpilot/replay/testdata/nexus-workflow-control-forgedCompletion-run.json` and the
  `UMPIRE_CONTROL_RECORD` hook in `TestTestpilotNexusControlForgedCompletionIsViolated` remain; they
  have no consumer now. .6 deletes them with the replay package.
- `UMPIRE_REPEAT_RUN_DIR` (tests/testpilot_signature_test.go, testpilot_generated_test.go) was
  umpire-repeat's capture hook. It is inert now; .6 removes it.

## Deferrals for .6 (please file)

1. **Canary reduced to lease and policy.** tools/canary still runs a Case through
   casebinding/assessment/provenance/publication. It can produce Runs only through the Temporal
   Driver (.4), and .6 owns the tools/canary deletions with evaluation and casefile. In .6 keep only
   `controller` lease/fencing and `policy`, and have the canary publish Runs with `runner.Publish`.
2. **Retire `common/testing/testpilot/publish`.** The runner has its own never-overwrite `Publish`
   because the model module cannot import the server module. Importers that remain:
   common/testing/testpilot/evaluation/receipt_test.go, common/testing/testpilot/replay/proposal.go,
   tools/canary/cmd/umpire-canary/run.go, tools/canary/controller/controller.go,
   tools/canary/publication/publication.go, tools/canary/publication/publication_test.go,
   tools/umpire/internal/cli/proposal.go, tools/umpire/internal/cli/proposal_test.go.
3. **Stale prose in packages .6 deletes** that still names the deleted commands:
   common/testing/testpilot/temporal/binding/binding.go:1, common/testing/testpilot/temporal/README.md:93,102,
   common/testing/testpilot/recordedrun/signature.go:14,72, common/testing/testpilot/recordedrun/recordedrun.go:38,
   common/testing/testpilot/replay/bridge.go:164.

## Pre-existing live failures (not caused by fn-228.3)

`make umpire-check-live-tests` after the fixes: rc 2 (live-tests.log, 1200 s), with four failure
identities. The same four tests fail with the same lines at the base 0c7822bc43, in a clean worktree
`.worktrees/fn2283-baseline`, since removed (baseline-four.log, rc 1). They are pre-existing:

| Test | Failure line (base and after) |
| --- | --- |
| `TestTestpilotExplorationDiscoversUnpinnedExecution` | `open .../model/ir/nexus-workflow.json: no such file or directory` (testpilot_exploration_test.go:55) |
| `TestTestpilotGeneratedCases` | `lstat .../model/temporal/features: no such file or directory` (testpilot_generated_test.go:145) |
| `TestTestpilotNexusControlForgedCompletionIsViolated` | `malformed at contract: invalid or duplicate correlated transition` (testpilot_live_case_test.go:87) |
| `TestTestpilotNexusPairCase` | run disposition expected 1, actual 3; `observe_failed ... optional.none() dereference` (testpilot_signature_test.go:74) |

The first two read the retired Scala-era IR/feature tree. They are .6's to retire or map, along
with the seven legacy failures.
