---
satisfies: [R4]
---
# fn-114-gomad-correct-search-path-defects-and.4 Commit a diverging exploration candidate as a retained candidate result

## Description
C3 (R4): a forced prefix that diverges becomes a typed candidate result that the round commits, instead of a `HostError` that discards the round. Depends on task 3 only because both edit `runner_test.go`.

**Size:** M
**Files:** `tools/gomad3/runner/choice_exploration_campaign.go`, `tools/gomad3/runner/internal/exploration/choice/engine.go`, `engine_test.go`, `tools/gomad3/runner/internal/campaign/choice_exploration_journal.go` and its test, `tools/gomad3/runner/runner_test.go`, `tools/gomad3/runner/resume.go`, `tools/gomad3/cmd/gomad/internal/cli/explore_output.go`
**Touches:** [tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/internal/exploration/choice/**, tools/gomad3/runner/internal/campaign/choice_exploration_journal.go, tools/gomad3/runner/internal/campaign/choice_exploration_journal_test.go, tools/gomad3/runner/runner_test.go, tools/gomad3/runner/resume.go, tools/gomad3/cmd/gomad/internal/cli/**, .flow/specs/fn-114-gomad-correct-search-path-defects-and.md]

### Approach
- Start from task 1's `TestRunChoiceExplorationDivergingPrefixDiscardsCompletedRound` and invert its loss assertions. A real executor returns `ChoiceReplayDivergenceError`; `runSeed` preserves that error and `executeExplorationRound` currently returns `HostError{Reason: "target_supervision"}` before outcome processing. Preserve the typed divergence through collection and commit it as a candidate result. The runner-domain branch is not the reproduced trigger; retain its host-error fallback for other runner outcomes where reachable.
- Add a divergence form to the engine's candidate result: candidate identity, divergence ordinal and reason, and the expected and observed records from the terminal frame. `CommitRound` accepts it without an outcome digest.
- A divergence result adds nothing to the outcome set or the failure-signature set, has no trace, and yields no children.
- In the campaign, only a typed choice replay divergence of a forced-prefix candidate takes the new path, including the executor-error completion path. Keep all other execution errors and runner-domain fallback outcomes (host failure, cancellation, trace capacity, missing identity) as `HostError`. List which reasons take which path and test each side.
- The other candidates of the round have already completed when the divergence is classified; their results commit with the round.
- Report the divergence under the existing `replay_divergence` classification in the campaign result, `runs` output, and `inspect`. The campaign must not exit as a success and must not count the divergence as a target failure.
- Failure policy: a divergence stops the campaign under `first`, and under `budget` and `all` exploration continues. It does not consume the distinct-failure-signature budget. Record this choice in the spec's Decision Context.
- Raise the round-segment schema because the segment bytes change. Resume rejects a journal of the other schema visibly and replays a segment holding a divergence result to the same state identity.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/choice_exploration_campaign.go:113-176`, `:250-326` — round loop, completion collection, runner-domain branch
- `tools/gomad3/runner/internal/exploration/choice/engine.go:59-105`, `:202-290` — candidate, result, segment types and `CommitRound`
- `tools/gomad3/runner/internal/campaign/choice_exploration_journal.go:55`, `:175-200` — segment schema and plan check on resume
- `tools/gomad3/runner/internal/execution/choicetrace.go:19-42` — divergence error and its fields
- `tools/gomad3/runner/internal/execution/process_unix.go:635-644`, `tools/gomad3/runner/runner.go:1462-1467`, `:1714-1725` — executor-error trigger, propagation, and current host-error reason
- `tools/gomad3/runner/internal/execution/outcome.go:23`, `:85-86` — runner-domain classification and reason names

**Optional** (reference as needed):
- `tools/gomad3/runner/runner_test.go:1064`, `:1135` — resume and failure tests to mirror
- `tools/gomad3/runner/choice_exploration_campaign.go:428-429`, `tools/gomad3/runner/resume.go:195` — where divergence reasons are counted
- `tools/gomad3/cmd/gomad/internal/cli/explore_output.go:138` — summary rendering

### Key context
- Task 1 re-anchor (2026-10-02): C3 is changed in mechanism, with its loss symptom confirmed on HEAD `1d7272e654f268f9a45f3fe965918fe2522827c6` plus source hashes in `reanchor.md`. Completed siblings remain as raw `partial.json`, `stdout.head`, `stderr.head`, and workspace scaffolding, but no candidate execution record, round segment, artifact, or typed divergence evidence is retained. Narrow the fix to the actual typed executor-error path without weakening runner-domain failure handling; no production correction was made in task 1.
- A diverging prefix taken from a deterministic parent is a Gomad confidence failure. It is never a target outcome.
- fn-109 tasks 2 and 3 edit the Runner options and completion paths; rebase onto whichever landed.
## Acceptance
- [ ] A campaign with one diverging candidate commits the round, including the results of its sibling candidates
- [ ] The retained evidence holds the candidate identity, divergence ordinal, reason, and expected and observed records
- [ ] The campaign reports `replay_divergence`, exits neither as success nor as target failure, and adds nothing to outcomes or failure signatures
- [ ] The diverged candidate produces no children
- [ ] A campaign killed after the round with the divergence resumes to the same final state as an uninterrupted run
- [ ] Runner-domain outcomes other than forced-prefix divergence still end the campaign as `HostError`, each with a test
- [ ] A journal with the previous segment schema is rejected visibly on resume
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/... ./cmd/gomad/...` and `make -C tools/gomad3 validate` pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
