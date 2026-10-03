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
