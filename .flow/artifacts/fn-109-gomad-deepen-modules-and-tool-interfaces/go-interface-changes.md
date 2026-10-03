# fn-109 Go interface changes

Record of public Go interface changes made, or deliberately deferred, by fn-109 tasks.

## fn-109.2 campaign options owner

**Exported changes: none.** The exported fields of `runner.CampaignSpec`, `CampaignPlanSpec`,
`ResumeSpec` and `CampaignShardSpec` keep their names, types and order. No consumer migration is
needed.

Unexported changes inside `tools/gomad3/runner` (not visible to consumers):

- `CampaignSpec` no longer carries the unexported `guidancePlan`, `resumePreflight`,
  `failureArtifactLimit` and `failureBytesLimit`. They are Runner state, not request intent, and
  now live on the private `campaignRun`.
- The private coordinator request changed from a flat `coordinatorConfig` to
  `coordinatorRequest{Options campaignOptions; SupervisorCommand; RunnerBuild}`, with options nested
  by group (`Target`, `Search`, `Limits`, `Observation`, `Retention`). Parent and coordinator are
  the same executable, so no version skew is possible. The request is not a recorded format.
  Besides the nesting, one value changes: options are normalized before sending, so an empty
  `Strategy` now travels as `"seed"`. The coordinator normalizes again, so an omitted strategy
  still means seed.

### Deferred regrouping wish (for the executor-injection task, R5)

`CampaignSpec` is still a flat list of 48 fields that mixes intent with wiring. The private
`campaignOptions` groups show a possible public shape: separate target intent, search settings,
resource limits, observation and retention from process wiring (`SupervisorCommand`,
`CoordinatorCommand`, `RunnerBuild`, `Progress`) and injected dependencies (`Preparer`,
`Executor`, `Replayer`). Regrouping the exported fields is a source-incompatible change for
`cmd/gomad/internal/cli` and every other `CampaignSpec` literal, so it needs the consumer inventory
that the executor-injection task owns. fn-109.2 did not do it.

## fn-109.3 atomic seed completion

**Exported changes: none.** The seed controller lives in `tools/gomad3/runner/internal/campaign`,
which only `runner` imports.

Internal change: `SeedController.FinishAttempt`, `RecordSuccess`, `RecordCancelled` and
`RecordFailure(domain, reason, distinct)` are replaced by one `Complete(Completion) bool`.
A `Completion` is built by `CompletedSuccess`, `CompletedCancelled`, `CompletedUnclassified` or
`CompletedFailure(domain, reason, distinctFailures)`; its zero value is rejected. The only caller
is the seed completion loop in `runner/runner.go`.

## fn-109.4 one CLI application construction path

**Exported changes: none.** `go doc -all ./cmd/gomad/internal/cli` is identical before and after;
the package is internal to `cmd/gomad` in any case. No Runner, toolchain or other public package
changed.

Unexported changes inside `tools/gomad3/cmd/gomad/internal/cli`:

- New private `application` (built once per invocation by `Run`) and `installation` values own
  executable lookup, installation resolution, the Runner build digest and the child-mode commands.
  Commands that need them became `application` methods.
- The `identity func(string) (string, string, string, error)` field of `qualifyDependencies`,
  `resumeDependencies` and `minimizeDependencies` became `install func(string) (installation, error)`.
  New `exploreDependencies`, `replayDependencies` and `campaignShardDependencies` follow the same shape.
- `runDoctor(arguments, stdout, stderr, executable)` became `application.runDoctor(arguments, stdout, stderr)`;
  doctor's `hashExecutable` became the shared `digestRunner`.

## fn-109.5 shared plan and explore parsing

**Exported changes: two additions in `runner`, no removals or signature changes.**

- `func ParseStrategy(value string) (Strategy, error)`: the one reading of a strategy name. An
  empty name is `StrategySeed` (as an omitted `CampaignSpec.Strategy` is); an unknown name fails
  with `unknown exploration strategy %q`. `validateConfig` and the CLI's `--strategy` both use it.
- `func ParseCoverageMode(value string) (CoverageMode, error)`: the one reading of a spelled
  coverage mode. It accepts `none`, `semantic`, `choice` and `semantic+choice` and fails with
  `unknown coverage mode %q` otherwise, including for an empty name: only an omitted
  `CampaignSpec.Coverage` means none, which `validateConfig` normalizes before parsing. The CLI's
  `--coverage` and `validateConfig` both use it.

Both are additive; no consumer migration is needed. The error texts are the ones the CLI and
`validateConfig` already produced.

Unexported changes inside `tools/gomad3/cmd/gomad/internal/cli`: `runExploreWith` no longer
branches on a hidden `--__plan` flag. A private `parseCampaignRequest(campaignOperation, ...)`
parses the shared grammar into a `campaignRequest`; `runExploreWith` calls `runner.Explore` and
`runPlanWith` calls `runner.CreateCampaignPlan`. Plan's fixed `--on-failure=all` is set on the
flag set before parsing instead of being prepended to argv.

## fn-109.6 private executor dependencies (fulfils fn-105.3 D3)

**Status: inventory written before the edits (source revision 8364bd6a0).**

### Removed exported declarations

All in `go.temporal.io/server/tools/gomad3/runner`. Each mentions `runner/internal/execution`
(`execution.Spec`, `execution.Result`), which no package outside `tools/gomad3/runner/...` can
import, so no consumer outside the Runner subtree can implement them.

| Declaration | Location (pre-change) |
| --- | --- |
| `type Executor interface { Run(context.Context, execution.Spec) (execution.Result, error) }` | `runner.go:120` |
| `type ReplayExecutor interface { Run(context.Context, execution.Spec) (execution.Result, error) }` | `replay_operation.go:28` |
| `CampaignSpec.Executor Executor` | `runner.go:178` |
| `ResumeSpec.Executor Executor` | `resume.go:30` (missing from the task's inventory; found by the consumer search) |
| `CampaignShardSpec.Executor Executor` | `campaign_shard_execution.go:30` |
| `MinimizeSpec.Executor Executor` | `minimize_operation.go:35` |
| `ReplaySpec.Executor ReplayExecutor` | `replay_operation.go:41` |

No exported declaration is added or changed. `Preparer`, `ArtifactReplayer` and the
`Preparer`/`Replayer` fields of `CampaignSpec`, `ResumeSpec`, `CampaignShardSpec` and
`MinimizeSpec` stay public and unchanged. `CampaignPlanSpec` never had an executor.

### Consumers (searched across the whole repository)

- Production code outside `runner/`: none sets or names these declarations.
  `cmd/gomad/internal/cli` (`cli.go`, `resume.go`, `campaign_shards.go`) injects whole operations
  (`runner.Explore`, `runner.CreateCampaignPlan`, `runner.Resume`, `runner.RunCampaignShard`,
  `runner.Replay`, `runner.Minimize`) as functions. `qualification/workload/workload.go:115-118`
  injects `runner.Explore` and `runner.Replay` as functions. Neither changes.
- `tools/gomad3sim`, `tools/gomad3integration` and every other module of the repository: no
  import of `tools/gomad3/runner`. (`qualification/set/prune_test.go`'s
  `qualifiedWithoutReplayExecutor` is an unrelated local name.)
- Inside `runner` (production): `campaign_options.go` (`campaignRun.Executor`), `runner.go`
  (`Explore`'s isolated rejection, `runLocal`, `validateConfig` x2, `runSeed`,
  `simulationCapabilityForJob`), `choice_exploration_campaign.go`,
  `simulation_exploration_campaign.go`, `resume.go` (`Resume`, `resumeConfiguration`'s toolchain
  identity check), `campaign_shard_execution.go` (`RunCampaignShard`'s toolchain identity check),
  `minimize_operation.go` (session executor, `replayExecutor` hand-over to replay),
  `replay_operation.go` (`Replay`'s default and `replayProcessExecutor` simulation role).
- Tests (all package `runner`, 75 sites): `runner_test.go` (`testConfig` and resume calls),
  `replay_operation_test.go`, `minimize_operation_test.go`, `portable_plan_test.go`,
  `inspect_test.go`, `watchdog_replay_test.go`, `guided_selection_test.go`, `diagnostics_test.go`,
  `coverage_replay_test.go`, `choice_exploration_divergence_test.go`,
  `choice_exploration_divergence_unix_test.go`, `completion_characterization_test.go`,
  `retention_characterization_test.go`, `campaign_options_characterization_test.go` (two table
  rows: "isolated injected executor", "injected executor without supervisor command") and
  `coordinator_transport_test.go` (the `Executor` row of `coordinatorLocalOnlyFields`).

### Replacement construction

House pattern of `toolchain.Build`/`buildWith`: a private `dependencies` value and private
`...With` entry points. The zero `dependencies` is production.

- `type targetExecutor interface { Run(context.Context, execution.Spec) (execution.Result, error) }`
  (private) replaces both `Executor` and `ReplayExecutor`, which had the same method set.
- `type dependencies struct { executor targetExecutor }` (private). A nil executor means the
  supervisor process (`processExecutor`, or `replayProcessExecutor` for replay), exactly as a nil
  public field did, so every check that keyed on injection keys on `dependencies.executor`:
  `Explore`'s "isolated Runner does not accept injected preparation or execution", resume's and
  validation's "supervisor command is required" only without an executor, the toolchain identity
  checks of resume and shard runs, `simulationCapabilityForJob`'s coordinator role for the process
  executor and replay's `replayProcessExecutor` simulation role.
- `Explore`, `Resume`, `RunCampaignShard`, `Replay` and `Minimize` keep their signatures and call
  `exploreWith`, `resumeWith`, `runCampaignShardWith`, `replayWith` and `minimizeWith` with
  `dependencies{}`. `campaignRun` embeds `dependencies`, so resume, shard and minimize delegation
  carry the executor into the campaign they build.
- Minimize used to forward its executor to replay through `ReplaySpec.Executor`. The default
  replayer becomes `artifactReplayer{dependencies}`, which calls `replayWith`, so a minimization's
  candidate replays still run through the same executor. A caller-supplied `Replayer` receives the
  same `ReplaySpec` minus the removed field. Guidance replay never forwarded an executor and keeps
  `artifactReplayer{}`.
- No package-level variable holds dependencies or a hook.

### Caller migration

- Repository production callers: none needed.
- Same-package tests pass fakes through the private entry points with unchanged fakes and
  assertions.
- External callers that set one of the removed fields could not have implemented the interface,
  but could have passed nil explicitly or forwarded a value obtained from the package: they must
  delete the field. This is a source-incompatible change, permitted by the fn-109 spec (R5).
- A new external-consumer compile fixture outside the Runner subtree constructs `CampaignSpec`,
  `ResumeSpec`, `CampaignShardSpec`, `ReplaySpec`, `MinimizeSpec`, a custom `Preparer` and an
  `ArtifactReplayer` and compiles in a separate module.
