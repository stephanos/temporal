# R5 Go interface migration inventory (pre-edit, 2026-10-03)

This inventory is anchored to the current post-fn-108 checkout. The public
execution interfaces cannot be implemented outside `runner` because their method
signatures name `runner/internal/execution` types. The public replacement is the
existing operation functions and usable `Preparer`/`ArtifactReplayer` seams;
process execution remains their default. Tests within `runner` inject fakes at
private operation entry points through private dependencies. No new public
construction or execution interface is required.

| Exported declaration to remove or change | Repository consumers and exact migration |
| --- | --- |
| `runner.Executor` (`runner.go`), with `Run(context.Context, execution.Spec) (execution.Result, error)` | Production `runner.go`, `campaign_options.go`, `choice_exploration_campaign.go`, `simulation_exploration_campaign.go`, `campaign_shard_execution.go`, `resume.go`, `minimize_operation.go` use a private execution interface. Runner fake tests listed below use that private interface in same-package code. No outside-Runner production or sibling-module consumer imports this name. |
| `runner.ReplayExecutor` (`replay_operation.go`), with the same inaccessible method signature | Production `replay_operation.go` and `minimize_operation.go` use the same private execution interface; replay defaults to `replayProcessExecutor`. Same-package replay/watchdog tests pass fakes via a private replay entry point. No outside-Runner consumer imports this name. |
| `CampaignSpec.Executor` (`runner.go`) | `exploreWith(ctx, spec, dependencies)` passes the private value through `campaignRequestForExplore` and `campaignRequestFromSpecWith`; `Explore` supplies the zero value, which selects process execution. Coordinator injection rejection and resume supervisor validation inspect the private request executor. Fake campaign tests pass dependencies explicitly. |
| `CampaignShardSpec.Executor` (`campaign_shard_execution.go`) | `runCampaignShardWith(ctx, spec, dependencies)` forwards the private value through `campaignRequestFromSpecWith` to local execution; `RunCampaignShard` supplies production defaults. `createCampaignPlanWith` carries the same value for plan validation in fake tests. |
| `ReplaySpec.Executor` (`replay_operation.go`) | `replayWith(ctx, spec, dependencies)` receives the private value; `Replay` supplies production defaults. Verify-only and process-versus-fake capability behavior remain unchanged. |
| `MinimizeSpec.Executor` (`minimize_operation.go`) | `minimizeWith(ctx, spec, dependencies)` passes the value to the minimization session. Candidate execution and default exact replay use the same dependency; `Minimize` supplies process defaults. A custom public `Replayer` remains usable. |
| `ResumeSpec.Executor` (`resume.go`; additional field absent from the older task inventory) | `resumeWith(ctx, spec, dependencies)` forwards the value to `exploreWith`. Resume reconstruction explicitly retains it in `campaignRequestFromSpecWith`; `Resume` supplies process defaults. |

All direct fake consumers found by the pre-edit search are same-package files:
`runner_test.go`, `replay_operation_test.go`, `portable_plan_test.go`,
`minimize_operation_test.go`, `inspect_test.go`, `diagnostics_test.go`,
`coverage_replay_test.go`, `watchdog_replay_test.go`,
`guided_selection_test.go`, `choice_exploration_divergence_test.go`,
`choice_exploration_divergence_unix_test.go`,
`completion_characterization_test.go`, `retention_characterization_test.go`,
`diagnostic_identity_test.go`, and `coordinator_transport_test.go`.
Indirect fake consumers through `testConfig`, `completionCampaign`,
`retentionCampaign`, or `minimizationSpec` also include `clock_tick_test.go`,
`guidance_identity_test.go`, `runner_mode_unix_test.go`, `retention_test.go`,
and `seed_completion_characterization_test.go`. The helpers now return a
`CampaignSpec` or `MinimizeSpec` and a separate private dependency value.
These include early errors, cancellation/watchdogs, failure policy and ordinal
behavior, plan/shards, resume, minimize, replay, and simulation-capability
selection. Their fake calls and assertions migrate to the private entry points.

Outside the Runner subtree, `cmd/gomad/internal/cli/{campaign_shards.go,cli.go,
cli_test.go,diagnostics_test.go,exploration_divergence_test.go,guidance_test.go,
qualify.go,resume.go}` and `qualification/workload/{workload.go,
workload_test.go}` construct public requests or inject complete operation
functions. They do not set executor fields and require no source change.
Searches of `tools/gomad3sim`, `tools/gomad3integration`, and
`tests/gomadfunctional` found no Runner imports. The external module compiled
by `TestRunnerRequestsCompileInExternalModule` uses the fixture at
`tools/gomad3/internal/gomadtool/conformance/testdata/runner_external/consumer.go`.
It constructs all five request types with custom public `Preparer` and
`ArtifactReplayer` implementations.

`Preparer`, `ArtifactReplayer`, their public request fields, and all exported
operation signatures stay unchanged. The removed executor fields are an
intentional Go source compatibility break for an inaccessible interface.
