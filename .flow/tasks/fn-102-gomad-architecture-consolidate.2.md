---
satisfies: [R2]
---
# fn-102-gomad-architecture-consolidate.2 Extract shared completed-execution assessment inside Runner

## Description
R2. Baseline duplication: seed World/coverage handling runner.go:844 onward, choice processExplorationCompletion, simulation processSimulationExplorationCompletion. Characterize existing behavior before extraction. Use a private same-package module unless a subpackage demonstrably improves locality without type forwarding/cycles. Return detached validated data/classification; callers retain prepared.Verify(), cancellation ownership, expandability, statistics, storage and error wrapping/precedence. Pass narrow inputs rather than CampaignSpec. Reuse execution.ComposeRecording/Validate and existing coverage/projectors; do not reimplement codecs. The proof point is preserving the existing differing strategy behavior while deleting shared policy copies. Preserve existing comments. No new dependencies. Run focused commands from tools/gomad3 with GOWORK=off and -tags test_dep; use the patched toolchain for runtime-consumer tests.

**Size:** M

**Touches:** [tools/gomad3/runner/runner.go, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/simulation_exploration_campaign.go, tools/gomad3/runner/completion*.go, tools/gomad3/runner/runner_test.go]

**Files:** `tools/gomad3/runner/runner.go`; `choice_exploration_campaign.go`; `simulation_exploration_campaign.go`; new private `completion.go`/`completion_test.go`; existing `runner_test.go`.

### Quick commands

`go test -tags test_dep ./runner -run 'Test(Run|ValidateConfig|ExecutionEvidence|Completion)'`

## Acceptance
- [ ] All three strategies use shared assessment for overlapping policy.
- [ ] R2 positive/negative characterization compares fixed-identity evidence and exact HostError.Reason/precedence.
- [ ] No process/file effects, full config, strategy flag matrix, or new global hooks in assessor.
- [ ] Tests exercise the private interface directly and preserve strategy-level coverage.

## Done summary
NOT IMPLEMENTED. Moved to fn-105-gomad-follow-ups-deferred-scope.1 (D1) on 2026-09-29 as a scope cut; the task text above remains the implementation brief.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: