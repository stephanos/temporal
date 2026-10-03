# fn-109 Go interface changes

The pre-edit inventory and implementation record below describe their own
source snapshots. Their compile and test evidence does not qualify the
combined integration candidate on either native platform.

## R5 Go interface migration inventory (pre-edit, 2026-10-03)

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
## Integrated candidate record

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

**Status: implemented.** The inventory below was committed before any edit (9b5ae6b39, on
8364bd6a0) and is updated here to match the result. `go doc -all ./runner` before and after differ
in exactly the seven removed lines/blocks listed below
(`task-6/godoc-runner.diff`); every other exported declaration, including the `Explore`,
`Resume`, `RunCampaignShard`, `Replay` and `Minimize` signatures with their result names, is
unchanged. No other public package changed.

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
- External-consumer compile fixture: `tools/gomad3/testdata/runnerconsumer/consumer.go`, built by
  `TestRunnerExternalConsumerCompiles` (`tools/gomad3/runner_consumer_test.go`, part of
  `test-host`) as a separate module (`example.com/runnerconsumer`) that replaces
  `tools/gomad3` with the working tree. It constructs `CampaignSpec`, `CampaignPlanSpec`,
  `CampaignShardSpec`, `ResumeSpec`, `ReplaySpec` and `MinimizeSpec`, implements a custom
  `Preparer` and `ArtifactReplayer`, and calls all six operations. Checked by hand: the same
  module fails to build with "use of internal package ... runner/internal/execution not allowed"
  when it implements an executor, and with "unknown field Executor" when it sets the removed field.

### Result notes

- Test migration (package `runner` only): two test-side carriers, `injectedCampaign`
  (`CampaignSpec` plus `dependencies`, returned by `testConfig` and the campaign helpers built on
  it) and `injectedMinimization` (`MinimizeSpec` plus `dependencies`, returned by
  `minimizationSpec`), make every former injection site a compile error until it calls the
  private entry point (`exploreWith(ctx, config.CampaignSpec, config.dependencies)`,
  `minimizeWith(...)`). Request literals that set `Executor:` became
  `opWith(ctx, Spec{...}, dependencies{executor: X})`. Fakes and assertions are unchanged.
  `injectedCampaign.run()` replaces `newCampaignRun(config)` in tests that call `validateConfig` or
  `manifestForRun` directly, so they still see the substituted executor.
- `coordinator_transport_test.go`: `Executor` left `coordinatorLocalOnlyFields` (it is no longer a
  `CampaignSpec` field) and `executor` joined `campaignRunPrivateFields`.
  `campaign_options_characterization_test.go` gained a `dependencies` column for its two
  injected-executor rows; the pinned JSON table is byte-identical.
- Pre-existing quirk, not changed: a substituted replay with neither a supervisor command nor a
  bootstrap command panics in `replayBootstrapCommand` (index 0 of an empty slice). It was
  reachable before only through the public `ReplaySpec.Executor`. It is now reachable only from
  same-package tests. The characterization pins the supported shape (an explicit bootstrap command).

## R15 installation description (task 10, 2026-10-03)

No exported declaration is removed or changed. `target.Spec.ToolchainRoot`,
`target.ReadToolchainIdentity`, `target.ReadModuleCache`, `target.DownloadModule` and
`toolchain.ResolveInstallation`/`Installation` keep their signatures and results.

- Added public package `toolchain/installation`: `Layout` (`At`, unvalidated locations the
  builder publishes), `Build` (locations inside one build) and `Description` (`Describe`,
  validated launcher, build key and pinned build). It imports only the standard library.
- Private/internal only: `target/internal/build.PrepareCache` now takes the cache path
  supplied by the description instead of `(toolchainRoot, buildKey)`; target's private
  `readToolchainIdentityWith` became `readPinnedToolchainWith`, returning the identity
  together with the description; the `preparedTargetCacheRoot` test hook takes the
  description.
- Architecture edge: `target` and `deterministicio` may import `toolchain/version` and
  `toolchain/installation` from the toolchain owner, nothing else (`ownerMayImport`).

## R17 capability ownership and source inventory (task 11, 2026-10-03)

One exported declaration is removed: `target.DigestAdapterSourceInventory(root string)
(string, error)`. Its only consumers were `deterministicio/adapter_copy.go`, which wrapped it
for every adapter, and `deterministicio/grpc_adapter_test.go`. Both now use the private
`internal/sourceinventory.Digest`, which keeps the same algorithm, limits (5000 files,
512 MiB) and error text. Searches of `tools/gomad3sim`, `tools/gomad3integration`,
`tests/gomadfunctional` and the runner external-module fixture found no other use.
`target.AdapterCapacityError` and `deterministicio.AdapterCapacityError` are unchanged, and
each consumer maps `sourceinventory.CapacityError` to its own type.

`ReviewCapabilities`, `ReviewCapabilityClosure`, `CapabilityReview`, `CapabilityFinding`,
`UnsupportedCapabilityError`, `VerifyCompatibility` and the `Finding*` constants keep their
signatures and values. The five closure finding kinds are now defined from
`target/internal/capabilitypolicy` constants with the same strings. Everything else is private:
the collection, evaluation and linked-projection functions in `target`, and the new internal
packages `target/internal/capabilitypolicy` and `internal/sourceinventory`.

## R13 Artifact reference and owned handle (task 12, pre-edit inventory 2026-10-03)

Anchored to `48c95c0c97` (post-fn-108 R6/R7). Today one `artifact.Artifact` value is both
the published reference and the live handle: it holds an unexported `root *os.Root` and an
exported, mutable `Manifest` whose slices and pointers alias the handle's state.

| Declaration (package `artifact`) | Change |
| --- | --- |
| `type Artifact struct { Path; Manifest; StoredBytes; TargetSharing; root *os.Root }` | Loses `root`. It becomes only a detached reference: exported fields unchanged in name, type, order and JSON shape, so `runner.ReplayResult.Artifact` and `runner.MinimizeResult.Artifact` keep their encoding. |
| `func OpenArtifact(path string) (Artifact, error)` | Returns `(*Opened, error)`. New `type Opened` (owned handle) keeps path, manifest, stored bytes and the pinned root private. |
| `func (*Artifact) Close() error` | Moves to `func (*Opened) Close() error` (idempotent, nil-safe). |
| `func (Artifact) Detached() Artifact` | Replaced by `func (*Opened) Snapshot() Artifact`, which deep-copies the manifest. |
| none | Added accessors `(*Opened) Path() string`, `Manifest() record.ExecutionRecord` (deep copy), `StoredBytes() uint64`. |
| `func OpenPayload(Artifact, string, uint64) (*os.File, error)`, `ReadPayload(Artifact, string, uint64) ([]byte, error)`, `CopyPayload(Artifact, string, string, os.FileMode) error` | Become methods of `*Opened` with the same remaining parameters, validation and error texts. |
| `func TargetSharingOf(Artifact) (TargetSharing, error)` | Becomes `func (*Opened) TargetSharing() (TargetSharing, error)`. |

`PublishArtifact` and `Store.PublishArtifact` keep their signatures and return a detached
`Artifact` as today. Publication (staging, no-replace rename, manifest last, validated reuse)
is unchanged.

Consumers and migration (production): `runner/replay_operation.go` (`replayWith`, `preflight`,
`choiceCapabilityForArtifact`, `simulationCapabilityForArtifact`,
`verifyReplayCapabilityManifest`, `readWorldPayloads` take `*artifact.Opened`; `Detached()`
becomes `Snapshot()`), `runner/minimize_operation.go` (session handle, accepted/retained
reopen, `readRetainedMinimizationPayload`), `runner/inspect.go` (open, `TargetSharing()`,
`projectChoices`), `runner/resume.go` and `runner/internal/campaign/retained_evidence.go`
(`StoredBytes()`/`Manifest()` accessors), `runner/internal/corpus/corpus.go` (case
validation), `qualification/set/execution.go` (`validateArtifactIdentity`,
`projectArtifactChoice`). `runner/runner.go`, the exploration campaigns,
`runner/internal/corpus/admission.go` and `corpus.go` `merge`/`discard`/`entryFor` only hold
published references and need no change. Tests: `artifact/*_test.go` and the runner tests
that open artifacts (`runner_test.go`, `coverage_replay_test.go`, `minimize_operation_test.go`,
`retention_characterization_test.go`, `replay_operation_test.go`, `replay_io_integration_test.go`,
`watchdog_replay_test.go`, `guided_selection_test.go`, `completion_characterization_test.go`,
`diagnostic_identity_test.go`, `diagnostics_test.go`, `retention_test.go`).
`cmd/gomad/internal/cli` tests construct detached `artifact.Artifact` literals only and need no
change. No module outside `tools/gomad3` imports `artifact`.

### Implementation record (task 12)

Implemented as inventoried; no further exported declaration changed.

- `Opened` methods: `Close`, `Path`, `Manifest`, `StoredBytes`, `Snapshot`, `OpenPayload`,
  `ReadPayload`, `CopyPayload`, `TargetSharing`. Payload methods keep the old bodies, so the
  check order and error texts are unchanged: `artifact is not open`, `artifact payload %q is
  not listed`, `... exceeds its bound`, `<name> metadata does not match its manifest`,
  `... identity mismatch`, `<name> is a symbolic link`, link-count and `os.Root` escape errors.
  A nil `*Opened` closes as a no-op; its `OpenPayload`, `CopyPayload` and `TargetSharing` fail
  with `artifact is not open`, and its `ReadPayload` reports the payload unlisted, as a zero
  `Artifact` did before.
- Path, Manifest, StoredBytes and Snapshot remain usable after Close; they hold no resource.
- `Manifest()` and `Snapshot()` copy through the private `cloneManifest`, a reflective deep
  copy that keeps nil pointers, slices and maps nil and panics on interface, func or chan
  fields. `TestCloneManifestSharesNoMemory` fills every field of `record.ExecutionRecord` and
  proves the copy is equal and shares no pointer, slice or map.
- The store's validated-reuse path returns `existing.Snapshot()` with the store's sharing, in
  place of the field-by-field copy.
- Migration notes: `runner.preflight` used a named result for the handle, so on a validation
  error the deferred close saw the zeroed result and never closed the root. Its handle is now
  a local, so a rejected artifact is closed and a close error joins the validation error.
  `minimizationSession.acceptedInput` passed the handle's manifest into publication, which
  writes through pointer fields; it now passes its own copy (same values, same bytes).
- Fixed-input publication is byte-identical: a scratch test (not committed) published 12
  store configurations (failure, reuse, success collisions, execution and record keys, pool
  shared, pool reuse, reuse without pool) on base `48c95c0c97` and after. Directory
  names, `manifest.json` SHA-256, record hashes, stored bytes and sharing all matched.
