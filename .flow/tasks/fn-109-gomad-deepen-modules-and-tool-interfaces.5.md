---
satisfies: [R6]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.5 Share plan and explore parsing directly and move semantic normalization to Runner

## Description
Stage 3, second half of R6. `gomad plan` re-enters `runExplore` through a hidden flag, and application rules live in both the CLI and Runner validation. Parse once, invoke each operation directly, and leave semantic normalization with the options owner from task 2.

**Size:** M
**Files:** `tools/gomad3/cmd/gomad/internal/cli/cli.go`, `explore_output.go`, `cli_test.go`, `tools/gomad3/runner/` options owner (task 2) for any normalization that moves.
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/runner/*.go]

### Approach
- Hidden route: `runPlan` (`cli.go:689-691`) calls `runExplore(append([]string{"--__plan", "--on-failure=all"}, arguments...))`; `runExplore` (`:405-687`) branches on `*planOnly` at `:633`. Split `runExplore` into one shared parse step returning a typed parsed request and two thin operations (`runner.Explore`, `runner.CreateCampaignPlan`). The `--__plan` flag disappears; `gomad explore --__plan` must become an unknown-flag error, and `gomad plan` keeps `--on-failure=all` as its fixed policy with the same messages (`"gomad plan requires --output FILE"`, `:635`).
- Division of rules: the CLI keeps flag-presence validation and presentation (`exploreStrategyOptions` `:693-716`, `resolveExploreStrategy` `:718-800`, `resolveExploreGuidance` `:806`, `resolveExploreSeeds` `:825`, `resolveExploreCoverage` `:841`, `resolveChoiceTrace` `:858`, and the `--coverage` needs `--choices` check `:585-594`). Semantic rules that do not depend on flag presence move to, or are deleted in favour of, Runner's normalization (`validateConfig`, `runner.go:1194`). Keep first-error precedence: for each input in the characterization table the same message wins.
- Reuse the application value from task 4 for identity and child commands.
- Pin behaviour with task 4's characterization tests plus plan-specific cases: `plan` with and without `--output`, JSON and text output (`:647-659`), rejected explore-only flags, and argv after the target.

### Investigation targets
**Required:**
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:405-870`
- `tools/gomad3/runner/runner.go:1194-1404` and the task 2 options owner
- `tools/gomad3/runner/portable_plan.go:1-140` (`CampaignPlanSpec`, `CreateCampaignPlan`)
- `tools/gomad3/CLI.md` (plan and explore sections)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'ValidateConfig|CampaignPlan|PortablePlan'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./cmd/... ./runner/... ./qualification/...
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
- [x] Plan and explore share one parse step and each calls its own Runner operation; no hidden plan-only argument route remains, and `--__plan` is rejected as an unknown flag.
- [x] Semantic normalization has one owner in Runner; the CLI keeps only presence-sensitive validation and reporting.
- [x] The characterization tests from task 4 plus the plan cases pass with identical messages, classifications, stdout/stderr routing and exit statuses.
- [x] No default changes; explicit zero and irrelevant flags are rejected exactly as before.
## Done summary
Plan and explore share one typed parse step and call their respective Runner operations directly. The hidden --__plan route is removed and rejected. Runner owns shared strategy/coverage defaults, seed cardinality, coverage/probe and choice-trace validation; the CLI retains presence-sensitive defaults and error presentation at its original validation points. Direct Runner requests still reject absent guided coverage. CLI grammar, explicit-zero/irrelevant flag errors, text/JSON output, argv/environment/build tags, first-error order and canonical characterization are preserved.

Captured pre-edit CLI/Runner Quick checks and plan characterization pass; hidden-route and guided-default regressions retain actual failing evidence. The frozen Darwin full host gate passes in 157.019s, including cmd/Runner/qualification/root architecture, with source hashes unchanged throughout. After the bounded diagnostic-writer and typed-error fixes, affected CLI checks, Runner's 76-row canonical characterization, plan text/JSON e2e, vet, architecture, formatting/diff checks, task-only patch lint and CLI rebuild pass. Parent independently verifies final characterization, original bytes/comments, final hashes and patch binding. Handover and actual logs identify each command and its source scope.

Independent codex:gpt-6-sol:high review returned SHIP at 2026-10-03T12:37:13.028379Z after replacing two mutable error globals with typed errors; there are zero introduced findings. The same-family review records the unchanged pre-existing doctor JSON stdout-write P2. Task-only patch lint is clean; the earlier unfiltered run retains 494 historical findings and root lint's known nested-module loading failure was not rerun. This is task-level R6 acceptance, not full-spec R18/R19 completion. Native linux/amd64 qualification remains incomplete in task 21.

Nothing was staged, committed or pushed; the user owns commits.

stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/...; exit 0; baseline-cli.json; baseline-cli.log, .toolchain/bin/go test -count=1 -tags test_dep ./runner -run ValidateConfig|CampaignPlan|PortablePlan; exit 0; baseline-runner.json; baseline-runner.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/internal/cli -run TestPlanValidationOrderAndJSONRouting -v; exit 0; pre-refactor-characterization.json; pre-refactor-characterization.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad -run TestPlanTextAndJSONPreserveCampaignIdentityAndTargetArguments -v; exit 0; pre-refactor-plan-output.json; pre-refactor-plan-output.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/internal/cli -run TestPlanValidationOrderAndJSONRouting|TestCampaignOptionNormalizationAndValidation; exit 0; before-semantic-cli-precedence.json; before-semantic-cli-precedence.log, .toolchain/bin/go test -count=1 -tags test_dep ./runner -run TestParseSingleBaseSeedSharesRunnerStrategyRule|TestCampaignOptionNormalizationAndValidation|ValidateConfig|CampaignPlan|PortablePlan; exit 0; semantic-runner.json; semantic-runner.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/internal/cli -run TestPlanAndExploreRejectHiddenPlanRoute|TestPlanValidationOrderAndJSONRouting|TestResolveExplore|TestResolveChoiceTrace|TestPublicCommandExplicitZeroAndIrrelevantFlagClassification; exit 0; semantic-cli.json; semantic-cli.log, make -C tools/gomad3 test-host; exit 0; full-host-result.json; full-host.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/internal/cli -run TestExploreErrorReportsClassificationAfterChoiceDiagnosticWriterFailure|TestPlanAndExploreRejectHiddenPlanRoute|TestPlanValidationOrderAndJSONRouting|TestPublicCommandWriterFailuresPreserveStatus; exit 0; writer-focused.json; writer-focused.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/...; exit 0; post-host-cli.json; post-host-cli.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad/internal/cli ./runner -run TestResolveExploreTypedErrorPresentation|TestCampaignOptionErrorsPreserveKindThroughWrapping|TestCampaignOptionNormalizationAndValidation|TestParseSingleBaseSeedSharesRunnerStrategyRule|TestPlanAndExploreRejectHiddenPlanRoute|TestPlanValidationOrderAndJSONRouting|TestExploreErrorReportsClassificationAfterChoiceDiagnosticWriterFailure|ValidateConfig|CampaignPlan|PortablePlan|CampaignOptionsCharacterization; exit 0; review-fix-focused.json; review-fix-focused.log, .toolchain/bin/go test -count=1 -tags test_dep ./cmd/gomad -run TestPlanTextAndJSONPreserveCampaignIdentityAndTargetArguments; exit 0; review-fix-plan-output.json; review-fix-plan-output.log, .toolchain/bin/go test -count=1 -tags test_dep . -run TestPackageArchitecture; exit 0; review-fix-architecture.json; review-fix-architecture.log, .toolchain/bin/go vet -tags test_dep ./cmd/gomad/... ./runner/...; exit 0; review-fix-vet.json; review-fix-vet.log, /Users/stephan/Workspace/temporal/gomad/.bin/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --new-from-patch=/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-5/task-only.patch ./cmd/gomad/... ./runner/...; exit 0; review-fix-lint.json; review-fix-lint.log, make -C tools/gomad3 runner; exit 0; review-fix-runner-build.json; review-fix-runner-build.log, Parent final CLI/Runner characterization exit0; parent-final-characterization.log (before typed-error fix), parent-review-fix-characterization.log (final source), gofmt/diff checks and source/comment/patch validation; final-source.json, parent-source-verification.json; exit0, Expected behavioral red regressions: pre-refactor-red.log (--__plan), guided-default-red.log (Runner default), writer-behavior-red.log (premature return); final affected checks exit0
- PRs: