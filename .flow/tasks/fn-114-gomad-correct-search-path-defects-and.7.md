---
satisfies: [R6]
---
# fn-114-gomad-correct-search-path-defects-and.7 Stop guided selection from re-running answered corpus seeds and add the regression mode

## Description
E1 (R6): a guided campaign schedules no corpus seed whose retained record already answers the execution, and reports how many executions were new. Depends on task 3 for corpus identity and `guidance.go`, and task 4 to serialize shared Runner and CLI edits. The approved 2026-10-02 order moves this task before task 6; start-ordinal support is not a functional prerequisite.

**Size:** M
**Files:** `tools/gomad3/runner/seeds.go`, `seeds_test.go`, `runner.go`, `guidance.go`, `runner_test.go`, `portable_plan.go`, `tools/gomad3/runner/internal/corpus/model.go`, `tools/gomad3/runner/campaign_plan.go`, `tools/gomad3/runner/internal/campaign/resume_plan.go`, `tools/gomad3/cmd/gomad/internal/cli/cli.go`, `explore_output.go`, `cli_test.go`
**Touches:** [tools/gomad3/runner/seeds.go, tools/gomad3/runner/seeds_test.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/guidance.go, tools/gomad3/runner/runner_test.go, tools/gomad3/runner/portable_plan.go, tools/gomad3/runner/campaign_plan.go, tools/gomad3/runner/resume.go, tools/gomad3/runner/coordinator.go, tools/gomad3/runner/internal/corpus/**, tools/gomad3/runner/internal/campaign/**, tools/gomad3/cmd/gomad/internal/cli/**]

### Approach
- First step: confirm the inferred consequence from task 1. Show with a test that a guided seed drawn from a corpus bound to the same identity reproduces the record the corpus holds. If it does not, close E1 per the closure rule.
- A corpus entry answers an execution when its identity equals the campaign's corpus identity and its replay result is verified and matching. After task 3 that identity includes the environment and tick policy.
- Default mode: answered seeds are left out of the guided share and out of the unguided pool, so they are not executed at all. Keep the reserved unguided quarter and the selection's dependence on the snapshot digest only, never on completions.
- When the corpus offers no unanswered seed to prioritize, guidance selects none and the result says so. The campaign then runs its requested selection minus the answered seeds. It never substitutes other seeds to keep the requested count.
- The reduced selection is the one recorded in the Campaign plan, so resume and shards see the same seeds. The result reports how many requested seeds were left out as answered.
- If every requested seed is answered, the campaign executes nothing, completes without a failure, and reports that all requested seeds are answered and that the regression mode re-runs them. State the exit status in the CLI guide text and test it.
- Regression mode: an explicit flag (proposed `--guide-regression`) restores today's selection, re-running corpus cases. The mode enters the Campaign plan and its identity, so resume and shards compute the same selection, and resume with a different mode is rejected.
- Report: a count of new executions separate from the selection count, in the campaign result, the JSON output, and the text summary.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/seeds.go:27-95` — selection mix and the unguided reservation
- `tools/gomad3/runner/runner.go:551-568` — caller, snapshot digest, journal selection
- `tools/gomad3/runner/runner.go:82-107`, `:180-200` — campaign event and result counts
- `tools/gomad3/runner/internal/corpus/model.go:60-65`, `:160-185` — replay result and prioritized seeds
- `tools/gomad3/runner/internal/campaign/resume_plan.go:55` — plan fields restored on resume
- `tools/gomad3/runner/resume.go:101` — resume takes the selection from the plan

**Optional** (reference as needed):
- `tools/gomad3/runner/seeds_test.go:9`, `:32` — selection tests to extend
- `tools/gomad3/runner/runner_test.go:297`, `:1436` — guided run and guided resume tests
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:404-460` — explore flags

### Key context
- With the corpus bound to the exact target, default guidance may select no seed at all for an unchanged target. That is the intended result of this finding; the report line makes it visible.
- CLI usage text changes here; README and CLI guide prose is written in task 14.
## Acceptance
- [ ] A test shows the E1 consequence on the tree before the change, or E1 is closed as refuted with the evidence
- [ ] A guided campaign over a corpus whose cases all bind the current identity executes no seed the corpus already answers
- [ ] The unguided share is at least one quarter of the selection, rounded up
- [ ] A resumed guided campaign and each shard select the same seeds as the uninterrupted campaign
- [ ] With no unanswered corpus seed, the result states that guidance selected none, and the requested selection minus the answered seeds runs
- [ ] A requested selection that partly overlaps answered seeds runs only the unanswered ones and reports the number left out
- [ ] A requested selection that is entirely answered executes nothing and reports that, with a tested exit status
- [ ] The regression mode re-runs corpus cases; resuming with a different mode is rejected
- [ ] The result reports the number of new executions as its own field in JSON and text
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/... ./cmd/gomad/...` and `make -C tools/gomad3 validate` pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
