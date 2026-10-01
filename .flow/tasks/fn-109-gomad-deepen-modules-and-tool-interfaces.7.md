---
satisfies: [R4]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.7 Introduce the complete preparation owner and migrate explore and portable planning

## Description
Stage 3, first half of R4 (F4). Runner and portable planning each run the same four-step protocol: prepare build adapters, call the preparer, attach adapter identity, validate the prepared target. Create one composition module above `target` and `deterministicio` and move these two callers onto it. Analysis and compatibility review follow in the next task.

**Size:** M
**Files:** new `tools/gomad3/internal/preparation/` (name is a proposal), `tools/gomad3/runner/runner.go`, `runner/portable_plan.go`, `runner/campaign_shard_execution.go`, `runner/deterministicio.go`, `tools/gomad3/architecture_test.go` (owner registration), tests.
**Touches:** [tools/gomad3/internal/preparation/**, tools/gomad3/runner/*.go, tools/gomad3/architecture_test.go]

### Approach
- Repeated protocol: `runner.go:517-547` (`PrepareTargetBuildAdapters`, `preparer.Prepare`, `prepared.Adapters = executionAdapters(...)`, `ValidatePreparedTarget`) and `portable_plan.go:119-136` (same sequence with the bundle as preparation root). Implementations beneath the seam stay where they are: `deterministicio.Spec.PrepareTargetBuildAdapters` (`deterministicio/adapter_registry.go:274`), `ValidatePreparedTarget` (`deterministicio/profile.go:279`), `target.Prepare` (`target/target.go:209`), `executionAdapters` (`runner/deterministicio.go:9`).
- Placement: `deterministicio` imports `target`, so the composition cannot live in `target`. A new root-internal package needs an architectural owner or `TestPackageArchitecture` fails with "has no architectural owner": add it to `packageOwner` and `ownerMayImport` (`architecture_test.go:398-466`), allowing `runner`, `qualification` and `cli` to import it and it to import `target`, `deterministicio`, `record`, `hostfs`. The broader fitness rules belong to the architecture-check task.
- Interface (two operations, per spec "Preparation and host commands"): complete target preparation returning a validated prepared target with adapter identities already attached, and capability inspection (next task). The caller supplies the durable destination where one exists (`journal.PreparedPath()` for campaigns, the bundle for portable plans); the module owns only implementation workspace such as `.io-adapter` (`adapter_registry.go:219`).
- Custom preparers stay supported: `CampaignSpec.Preparer` skips adapter preparation today (`runner.go:523-530`) and `campaignPlanPreparer` (`campaign_shard_execution.go:135-177`) restores a planned target. The new owner must express "caller-supplied prepared target" without the caller re-implementing validation ordering.
- Journal transitions stay with the campaign owner: `BeginPreparation`, `FailPreparation` and the `target_preparation` / context-reason classification (`runner.go:517-540`) keep their order and reasons.
- Equivalence harness: for a fixed fixture target with fixed toolchain, compare `target.Prepared` (including `Adapters`) and the resulting record target projection before and after, for a fresh build and a cache hit.

### Investigation targets
**Required:**
- `tools/gomad3/runner/runner.go:498-560`
- `tools/gomad3/runner/portable_plan.go:100-170`
- `tools/gomad3/deterministicio/adapter_registry.go:130-150,200-300`, `deterministicio/profile.go:270-330`
- `tools/gomad3/runner/campaign_shard_execution.go:80-177`
- `tools/gomad3/architecture_test.go:24-56,398-490`
**Optional:**
- `tools/gomad3/runner/runner_test.go:1507-1570` (preparation failure, cancellation, timeout classification)
- `tools/gomad3/qualification/workload/workload.go` (independent preparation per repetition)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./internal/preparation/... ./runner/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'Preparation|PortablePlan|CampaignPlan|Shard'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./qualification/...
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Explore and portable planning obtain a validated prepared target with adapter identities attached from one preparation owner; neither attaches adapters nor orders validation itself.
- [ ] Implementation-only workspace cleanup is owned by the module, and cleanup failure is reported; campaign and bundle owners keep their durable destinations and journal transitions.
- [ ] Tests through the new interface cover fresh and cache builds, an external module with a local replacement, a custom preparer and two independent preparations of the same target.
- [ ] Invalid sums, replacement conflicts, changed binaries and cleanup failures keep their existing classifications and `HostError` reasons, including cancellation and overall-timeout reasons.
- [ ] The new package has a registered architectural owner and introduces no import cycle; fixed-input `target.Prepared` values match the pre-change values.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
