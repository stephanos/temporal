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
- [ ] Plan and explore share one parse step and each calls its own Runner operation; no hidden plan-only argument route remains, and `--__plan` is rejected as an unknown flag.
- [ ] Semantic normalization has one owner in Runner; the CLI keeps only presence-sensitive validation and reporting.
- [ ] The characterization tests from task 4 plus the plan cases pass with identical messages, classifications, stdout/stderr routing and exit statuses.
- [ ] No default changes; explicit zero and irrelevant flags are rejected exactly as before.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
