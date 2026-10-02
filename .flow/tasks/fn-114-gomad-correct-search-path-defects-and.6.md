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
Added the choice-exploration start ordinal (R10/E5). `gomad explore --strategy=choice-exploration --choice-start-ordinal N` never expands a replay-plan decision before N, counts `--max-choice-depth` from N (the decision at N has depth 1), and leaves earlier decisions out of the omitted-by-depth count. `newCandidate` rejects any forced prefix that would alter a decision before the start. The start travels from the CLI through `CampaignSpec`, the coordinator wire, the Campaign plan (`choice_start_ordinal`), resume restore, and the engine `Config` (`start_ordinal`). The whole-config comparison on resume rejects a different start. A start at or past the root trace's decision count ends the run with `stop=choice_start_unreached`, which is not bounded-complete. The flag is rejected for the seed and simulation-exploration strategies by both the CLI and Runner validation.

The zero value is omitted from every encoding. Retained pre-change values pin this: the initial and after-root state identities, the exploration plan bytes, and the canonical choice-exploration Campaign plan digest, all in `runner/internal/campaign/choice_exploration_start_test.go`. A journal written by the pre-change Runner (`runner/internal/campaign/testdata/pre-start-ordinal-journal`) resumes to the retained state identity. Removing `omitempty` makes both pin tests fail.

`inspect --choices` now lists replay-plan decisions (`choice-decision: ordinal=… kind=… site-offset=… alternatives=… selected=…`; JSON `replay_decisions`, schema `gomad3.choice-inspection/v3`). Their numbering is the exploration numbering, because the list is projected with `choice.ProjectReplayPlan` and checked against the recorded tape SHA-256. `TestInspectChoicesListsReplayPlanDecisionOrdinals` shows that observation and single-alternative records are skipped. Finding the first test body (for task 14's docs): record a seed run with `--choices --keep-successes=all`, run `inspect --choices` on it, and map select-poll `site-offset` values (text offsets) to functions with the retained target binary. In the real fixture, runnable decisions report `site-offset=missing`, so only select sites locate user code. That limitation bears on spec Open Question 3.

Tests: engine (`TestExplorationStartOrdinalExpandsOnlyFromTheStart`, `TestExplorationReportsStartAtOrPastRootTrace`, `TestExplorationRejectsCandidateAlteringADecisionBeforeTheStart`), each red before the change; campaign pins and resume (`TestDefaultChoiceStartKeepsRetainedIdentityAndPlanBytes`, `TestResumeExplorationJournalWrittenBeforeTheStartOrdinal`, `TestResumeExplorationJournalRequiresItsStartOrdinal`, `TestSeedCampaignPlanRejectsChoiceStartOrdinal`); Runner (`TestRunChoiceExplorationResumesWithItsStartAndReportsItUnreached`, which fails when resume.go drops the start, and new validation rows); CLI strategy rows. A real-runtime run on the conformance `choice_exploration` fixture (seed 7, depth 2) gave: start 3 explores only forced depths 4–5 (6 executions); start 6 reports `choice_start_unreached` after 1 execution. Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-6/`.

Gates on darwin/arm64 at baef81f6e: full `test-host` green (45 packages, 146 s), `validate` green, vet green. The first full gate caught a pinned validation message in `coordinator_transport_test.go`; baef81f6e restored the message and gave the start its own error. Not run: linux/amd64 (no native host); root `make lint-code-fast` (`GOLANGCI_LINT_BASE_REV=main` unknown). Baseline: green via `test-host` at 49bada28e (45 packages). The literal spec Quick command run with Homebrew go1.27.0 is red pre-edit, because it lacks the patched toolchain. Human output lines gained `start=` / `exploration-start=`. README/CLI prose stays for task 14.


Review: SHIP from claude-fable-5-1 at high through the `claude` backend (same family as the writer; the reviewer had no shell and relied on the committed gate evidence). Its first note was applied: the seed-plan test now asserts `validateCampaignPlan` rejects a start ordinal. Two notes pass to task 14: the result line reports an absolute `exploration-depth` beside a start-relative `exploration-max-depth`, and the `choice-decision:` printer line has no unit test.

stage: impl-review - ran (model: claude-fable-5-1)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 07f7cb69fae9133408a5bd4ea93a956ab332525b, baef81f6e049b317e5234f58f6469937c371dc95, b3aebefea692498c6669fdab8547de26cbd4f0f9, aca268fe83731335ec0ac1a91a7e86fbf070abcb
- Tests: GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host (baef81f6e, exit 0, 45 packages, 146s), make -C tools/gomad3 validate (exit 0), go vet -tags test_dep ./runner/... ./cmd/... (exit 0), .toolchain/bin/go test -tags test_dep -count=1 ./runner (exit 0, 118s), real-runtime e2e: gomad explore --strategy=choice-exploration --choice-start-ordinal {0,3,6} on conformance choice_exploration (.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-6/e2e-start-ordinal.txt), make lint-code-fast: unavailable (GOLANGCI_LINT_BASE_REV=main unknown), linux/amd64: not run (no native host), .toolchain/bin/go test -tags test_dep -count=1 ./runner/internal/campaign/ (review follow-up, exit 0)
- PRs: