---
satisfies: [R16]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.3 Give the seed controller one atomic completion transition

## Description
Stage 2 (S4). R2/R3 need no task here: fn-108 R6/R7 own shared assessment and retention, and the final evidence task links them. This task covers the remaining stage-2 work: `SeedController` requires callers to pair `FinishAttempt` with a later `RecordSuccess`/`RecordCancelled`/`RecordFailure`, so attempted, active and classified counters can disagree between the two calls.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`. The call sites below sit inside the seed completion loop that fn-108 restructures.

**Size:** S/M
**Files:** `tools/gomad3/runner/internal/campaign/controller.go`, `controller_test.go`, `tools/gomad3/runner/runner.go` (call sites).
**Touches:** [tools/gomad3/runner/internal/campaign/controller.go, tools/gomad3/runner/internal/campaign/controller_test.go, tools/gomad3/runner/runner.go]

### Approach
- Current protocol: `FinishAttempt` (`controller.go:98`, panics when `active == 0`), `RecordSuccess` (`:106`), `RecordCancelled` (`:110`), `RecordFailure(domain, reason, distinct)` (`:114-138`). Callers: `runner.go:778` (finish), `:818` (cancelled), `:1035` (success), `:1100` (failure, returns cancel-active).
- Replace them with one pure completion transition taking a completion value (success, cancelled, or failure with domain, reason and the explicit distinct-failure count) and returning whether active work must be cancelled. Attempted, active, the classified counter and the failure-policy stop change in that one call.
- Completion without active work is an invariant violation, as today. Decide error-return versus panic by what `runner.go` can handle without changing its error precedence, and state the choice in the summary.
- Between `runner.go:778` and the classification sites the loop performs evidence work that can return early. Characterize first what `Statistics()` reports on each early-return path, then keep those observable counters identical; the atomic transition must not turn an early return into a lost attempt.
- Keep `Next`, `Stop`, `Stopped`, `Active`, `Done`, `Finalize`, `Statistics` and ordinal scheduling untouched. The controller stays pure: no host calls, no publication.

### Investigation targets
**Required:**
- `tools/gomad3/runner/internal/campaign/controller.go` (whole file, 170 lines)
- `tools/gomad3/runner/internal/campaign/controller_test.go`
- `tools/gomad3/runner/runner.go:760-830,1025-1110` (re-anchor after fn-108)
**Optional:**
- `tools/gomad3/runner/runner_test.go:671-712` (first-failure and budget policies), `:1324-1440` (cancellation, resume counters)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/campaign -run 'SeedController|Controller'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'TestRun'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/...
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
- [ ] `SeedController` exposes one completion transition; the `FinishAttempt` plus `Record*` pairing no longer exists and `runner.go` calls the transition once per completed job.
- [ ] Controller tests cover success, cancellation, watchdog, replay divergence, distinct and duplicate failures, resume-seeded counters and each failure policy (`all`, `first`, `budget`), comparing whole `CampaignStatistics` values.
- [ ] Completion without active work is rejected as an invariant violation; no path leaves attempted and classified counters partially updated.
- [ ] Ordinal scheduling, stop reasons and every counter reported by existing runner tests are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
