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

### Deferred regrouping wish (for the executor-injection task, R5)

`CampaignSpec` is still a flat list of 48 fields that mixes intent with wiring. The private
`campaignOptions` groups show a possible public shape: separate target intent, search settings,
resource limits, observation and retention from process wiring (`SupervisorCommand`,
`CoordinatorCommand`, `RunnerBuild`, `Progress`) and injected dependencies (`Preparer`,
`Executor`, `Replayer`). Regrouping the exported fields is a source-incompatible change for
`cmd/gomad/internal/cli` and every other `CampaignSpec` literal, so it needs the consumer inventory
that the executor-injection task owns. fn-109.2 did not do it.
