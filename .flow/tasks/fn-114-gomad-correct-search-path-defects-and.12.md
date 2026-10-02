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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
