---
satisfies: [R10]
---
# fn-114-gomad-correct-search-path-defects-and.6 Add the choice-exploration start ordinal

## Description
E5 (R10): choice exploration takes a start ordinal so `--max-choice-depth` is spent after bootstrap. Depends on task 7 to serialize the shared CLI and campaign edits; task 4 is a transitive dependency. Guided seed deduplication comes first under the approved 2026-10-02 delivery order.

**Size:** M
**Files:** `tools/gomad3/cmd/gomad/internal/cli/cli.go`, `cli_test.go`, `tools/gomad3/runner/coordinator.go`, `choice_exploration_campaign.go`, `runner.go` (config validation), `tools/gomad3/runner/internal/exploration/choice/engine.go`, `engine_test.go`, `tools/gomad3/runner/campaign_plan.go`, `resume.go`, `tools/gomad3/runner/internal/campaign/resume_plan.go`, `choice_exploration_journal.go`, `tools/gomad3/runner/runner_test.go`
**Touches:** [tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/runner/coordinator.go, tools/gomad3/runner/campaign_plan.go, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/internal/exploration/choice/**, tools/gomad3/runner/internal/campaign/**, tools/gomad3/runner/resume.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/runner_test.go, tools/gomad3/runner/inspect.go]

### Approach
- Add a start ordinal to the engine configuration, following the path `--max-choice-depth` takes from the flag to the engine, the Campaign plan, and resume validation. Proposed flag: `--choice-start-ordinal N`, accepted only with `--strategy=choice-exploration`.
- In expansion, a decision before the start is never expanded and adds nothing to the omitted-by-depth count. Depth counts from the start: the decision at the start ordinal has depth 1.
- The start ordinal counts decisions of the projected replay plan, which leaves out observations and single-alternative records. It is not a trace record ordinal; `inspect --choices` must show the same numbering.
- Decisions before the start are already forced as recorded by the child prefix; assert no candidate has a forced depth that alters a decision before the start.
- Default 0 must be byte-identical to today for a fixed identity. The engine configuration is hashed into the state identity and compared whole on resume, so the zero value must encode exactly as the configuration encodes today. Pin this with a retained state identity and plan bytes taken before the change.
- A start at or past the end of the root trace is reported explicitly in the campaign result. It must not look like an exhausted search.
- Document how a developer finds the ordinal of the first test body: check whether `inspect --choices` prints decision ordinals, and add the ordinal to that output if it does not. No marker API.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/internal/exploration/choice/engine.go:24-57`, `:349-384` — configuration and expansion
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:412`, `:504`, `:528-534`, `:620`, `:753-774` — flag, parsing, and strategy validation
- `tools/gomad3/runner/coordinator.go:41`, `:102`, `:121` — option plumbing
- `tools/gomad3/runner/campaign_plan.go:47`, `tools/gomad3/runner/resume.go:103` — plan construction and restore
- `tools/gomad3/runner/internal/campaign/resume_plan.go:55`, `:229` — plan field and resume validation
- `tools/gomad3/runner/internal/campaign/choice_exploration_journal.go:194` — whole-configuration comparison on resume

**Optional** (reference as needed):
- `tools/gomad3/runner/runner_test.go:454`, `:940` — validation and exhaustive exploration tests
- `tools/gomad3/runner/inspect.go:683` — choice listing
- `tools/gomad3/runner/choice_exploration_campaign.go:56` — engine configuration from the campaign
- `tools/gomad3/choice/tape.go:140-150` — which trace records become plan decisions

### Key context
- fn-109 tasks 2 and 5 edit the options owner and the CLI parser; rebase onto whichever landed.
- CLI usage text changes here; README and CLI guide prose is written in task 14.

## Acceptance
- [ ] With a start ordinal N, no candidate alters a decision before N, and depth is counted from N
- [ ] The omitted-by-depth count excludes decisions before the start
- [ ] A resumed campaign keeps the same start; resuming with a different start is rejected
- [ ] With the default, the state identity and plan bytes for a fixed configuration equal the values retained before the change, and a journal written before the change resumes
- [ ] A start at or past the root trace length is reported explicitly
- [ ] The flag is rejected for the seed and simulation-exploration strategies
- [ ] A developer can read decision ordinals from `inspect --choices`, shown by a test
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/... ./cmd/gomad/...` and `make -C tools/gomad3 validate` pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
