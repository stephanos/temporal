---
satisfies: [R8]
---
# fn-114-gomad-correct-search-path-defects-and.12 Check the select-poll reduction for soundness and stop expanding no-op decisions

## Description
E3 (R8), explorer half: establish that poll order is unobservable when fewer than two cases are ready, then stop expanding those decisions for the shapes the check covers. Depends on task 11 for the ready count and on task 10 so the engine, campaign, and Runner test files it shares with tasks 4 to 10 are edited serially.

**Size:** M
**Files:** `tools/gomad3/runner/internal/exploration/choice/engine.go`, `engine_test.go`, `tools/gomad3/runner/choice_exploration_campaign.go` (omitted-count reporting), `tools/gomad3/runner/runner_test.go`, the E3 fixture and `runtime_scheduling.go`, measurement record under `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/select-reduction/`
**Touches:** [tools/gomad3/runner/internal/exploration/choice/**, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/runner_test.go, tools/gomad3/internal/gomadtool/conformance/**, .flow/artifacts/fn-114-gomad-correct-search-path-defects-and/select-reduction/**, .flow/specs/fn-114-gomad-correct-search-path-defects-and.md]

### Approach
- Soundness first. Explore the task 2 fixture to exhaustion twice on the task 11 toolchain: unreduced, and with select-poll decisions of fewer than two ready cases left unexpanded. Compare the sets of outcomes and deadlocks per select shape.
- The unreduced set on the new toolchain must equal the set task 2 retained, compared on behavior and not on identities.
- A shape passes when the two sets are equal. A shape that fails, or a decision whose ready count is unknown, stays expanded, and the result is recorded with the shape.
- Eligibility predicate: a pure function of the evidence task 11 puts on the decision (ready count, case count, default, and the shape flags). A decision is eligible only when its ready count is known and below two and its evidence matches a shape that passed. Write the predicate as an explicit list of passing shapes, so anything unlisted is ineligible.
- Then the rule: expansion skips eligible select-poll decisions only. Count the skipped alternatives in their own omitted counter, reported beside the depth and execution-bound counters.
- The rule is part of the engine configuration identity. A campaign journal written without it resumes under its recorded configuration or is rejected visibly; it is never reinterpreted.
- Measure the Signal suite (`TestSignalWorkflowTestSuiteChasm`, seed 11): trace bytes and branching-decision counts before and after. Trace bytes do not change in this task; report the count of decisions the explorer no longer expands.
- Decide whether no-op decisions should also be dropped from the Choice Trace, from the measured share of trace bytes they occupy and the replay-validation cost of losing them. Readiness is unknown when the decision is recorded, so dropping needs a different recording point. Record the decision in the spec's Decision Context. If the decision is to drop, add a follow-up task; do not implement it here.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/internal/exploration/choice/engine.go:24-57`, `:349-384` — configuration identity and expansion
- `tools/gomad3/runner/internal/exploration/choice/engine.go:97-105`, `:232-290` — segment results and omitted counters
- `tools/gomad3/choice/tape.go:145` — the projected plan the explorer reads, with the count from task 11
- `tools/gomad3/internal/gomadtool/conformance/runtime_scheduling.go:12` — fixture checks
- `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/` — the unreduced baseline retained by task 2

**Optional** (reference as needed):
- `docs/research/gomad/2026-10-01-feasibility-schedule-search.md:62-63`, `:424` — the Signal suite counts and their source report
- `tools/gomad3/runner/runner_test.go:940`, `:1155` — exhaustive and equal-budget exploration tests
- `tools/gomad3integration/qualification/temporal.json` — the manifest entry for the Signal suite

### Key context
- The reduction is sound only for the shapes the fixture covers. Any select shape outside them stays expanded.
- The Signal suite measurement needs the representative target built with the candidate toolchain on darwin/arm64.
## Acceptance
- [ ] Reduced and unreduced exploration of the fixture agree on outcomes and deadlocks for every shape that the rule covers
- [ ] The unreduced result on the new toolchain equals the baseline retained by task 2
- [ ] The explorer expands no select-poll decision with a known ready count below two for covered shapes, shown by a test
- [ ] A failing shape and a decision with an unknown count stay expanded, each with a test and a recorded result
- [ ] Two decisions with the same ready count, one of a passing shape and one of an excluded or unlisted shape, are treated differently, shown by a test
- [ ] Skipped alternatives are reported in their own omitted counter
- [ ] A resumed campaign keeps the rule it started with
- [ ] Signal suite trace bytes and branching-decision counts before and after are retained under `select-reduction/` in the spec's artifacts directory
- [ ] The decision on dropping no-op decisions from the Choice Trace is recorded with its numbers
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/...` and `make -C tools/gomad3 validate test-runtime` pass
## Done summary
The choice explorer no longer expands a select-poll decision whose select polled two cases and recorded one of the seven fixture shapes with fewer than two ready cases (R8/E3, explorer half). The reduction was shown sound before it was applied: `TestRuntimeSearchFixtures` now explores every select-readiness shape to exhaustion twice, in full and with ready-below-two poll decisions left unexpanded, and the two agree on outcomes and deadlocks for all nine shapes (reduced 96/32/68/32/96/32/32/96/68 executions against full 196/68/68/68/196/68/68/200/68; the full counts equal task 2's and task 11's baselines). The list of proven shapes is `choice.NoOpSelectShapes`, keyed on two polled cases plus the recorded readiness; the fixture holds its `noOp` rows and that list to the same set before exploring, so neither side can gain a shape alone. Anything else stays expanded: an unknown readiness, two or more ready cases, a flag combination off the list, or a select polling three or more cases, each with an engine test, and the ready-two shapes are recorded as "equal, nothing reduced".

The rule is the v3 controller identity (`deterministic-rounds/breadth-first-rank-prefix/v3`), so a journal written under v2 is refused on resume by the existing identity checks rather than reinterpreted; a journal written under v3 resumes and replays its rounds under the rule (`TestRunChoiceExplorationLeavesNoOpSelectPollsUnexpandedAcrossResume`, `TestExplorationRoundSegmentReplaysByteIdentically`). Skipped alternatives are counted in `omitted_by_select_readiness` beside the depth, execution-bound, and capacity counters in the engine summary, the campaign record, `runner.ChoiceExplorationSummary`, and the `gomad explore` and `gomad inspect` exploration lines; it makes no stop reason because it leaves nothing unexplored.

Signal suite (`TestSignalWorkflowTestSuiteChasm` seed 11, build 2008ea81, run under `gomad explore --choices --choice-bytes=64MiB`): the trace is 86,070 records and 8,262,720 bytes before and after (recording is task 11's); of 57,696 branching decisions the explorer now expands 42,918. The 14,778 skipped are 55% of the 26,797 select-poll decisions, 26% of all decisions, and 3.5% of the alternatives a full expansion would run, because runnable decisions average 12.8 alternatives. Another 6,574 select-poll decisions have a known ready count below two but an unproven shape (2,919 two-case nil-and-closed, 81 timer-and-closed, 2 nil-and-timer; 3,572 steps of three- to six-case selects) and stay expanded. Decision, recorded in the spec's Decision Context with its numbers: no-op decisions stay in the Choice Trace. Their records are 17% of the trace (25% if every ready-below-two shape were proven), which does not evidently bring any D15 overflow suite under the 64 MiB cap, while dropping them needs a new recording point and replay rule (readiness is known only after the poll draws are taken) and loses the record that pins a forced prefix or divergence to the exact poll step. No follow-up task.

Outside the declared Touches, listed here: `runner/runner.go` (the public summary field and its projection), `cmd/gomad/internal/cli/explore_output.go` and `cli.go` (the counter on the two exploration lines), `choice/no_op_select.go` (the shared list the fixture reads, since conformance cannot import `runner/internal`), and `runner/internal/campaign/choice_exploration_start_test.go` with its `testdata/pre-start-ordinal-journal` re-encoded through `CommitRound` under the v3 controller identity (candidate and state identities changed, decisions unchanged; the three pinned constants repinned). The first three came from the review's P1 and P2 findings; the last is the mechanical consequence of the identity change. No runtime, overlay, or wire source changed; the toolchain build key is unchanged.

Not met or not run: linux/amd64 (no native host); `core-qualification` and `gomad3-smoke-qualification` (toolchain identity unchanged, task 14 qualifies the combined candidate); the spec's literal `go -C tools/gomad3 test -tags test_dep ./runner/...` with the stock go (fails helper-target tests regardless; `test-host` covers those packages with the patched toolchain). The fixture evidence's execution directories and the 163 MB Signal artifact stayed in session scratch. Follow-ups: new fixture shapes for the two-case nil-and-closed combination (2,919 decisions in the Signal trace) and for three-case selects would extend the list; for task 14, README and CLI docs gain the `omitted-select-readiness` field and the explorer's no-op rule, and the task 11 P3 follow-ups (`runChoiceAccepting`, `ready_at_poll`) are still open.

Gates on darwin/arm64 (`.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-12/gates.md`): first round `validate` 0:04, `test-host` 3:17 (45 packages), `test-runtime` 8:03, `test-toolchain overlay-test` 0:32, all exit 0 on their first run; after the review fixes `validate` 0:05, `test-host` 2:42, `test-runtime` 7:12, `go test ./cmd/...`, all exit 0. Baseline: green via handoff (all tiers at 7c93c665 on the same build key; only .flow commits since). Red-first for every new test and for the fixture's two checks in `select-reduction/red-first.txt`. Evidence: `select-reduction/` (README, `search-reproduction.json`, `signal-seed11.json`, `signal-seed11-explore-result.json`, `red-first.txt`, `toolchain-build-key.txt`) and `task-12/` (`gates.md`, `impl-review-receipt.json`).

Tier: not stated by the conductor; executed on claude-fable-5-1 (session model).

stage: impl-review - ran (backend claude, model claude-fable-5-1 at high, same family as the writer; round 1 NEEDS_WORK with two findings, counter not carried to the public summary and CLI lines, and two hand-maintained shape lists, both fixed; round 2 SHIP)
## Evidence
- Commits: 95e48ed08d48aff5bb625cf23c98e29ee7def353, 779a1744037c2cf3aee37f2794992b1d405117dc, 92cd50f86c9ae6bedab018bb20895bf181ed0aef
- Tests: go test -tags test_dep -count=1 ./runner/internal/exploration/choice/ ./choice/ . (tools/gomad3), go test -tags test_dep -count=1 -run ChoiceExploration ./runner/ (tools/gomad3), go test -tags test_dep -count=1 ./runner/internal/campaign/ ./cmd/... (tools/gomad3), env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT GOMAD3_RUNTIME_REPRODUCTION_DIR=<scratch> go test -tags test_dep -count=1 -run TestRuntimeSearchFixtures$ ./internal/gomadtool/conformance (tools/gomad3), make -C tools/gomad3 validate, GOFLAGS=-tags=test_dep -count=1 make -C tools/gomad3 test-host, make -C tools/gomad3 test-runtime, make -C tools/gomad3 test-toolchain overlay-test, baseline: green via handoff (all tiers at 7c93c665 by fn-114.11 on build key 2008ea81; only .flow commits since), gomad explore --seeds 11 --choices --choice-bytes=64MiB ... go-test ./tests -- -test.run=^TestSignalWorkflowTestSuiteChasm$ (measurement, exit 0)
- PRs: