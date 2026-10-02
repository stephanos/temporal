---
satisfies: [R1]
---
# fn-114-gomad-correct-search-path-defects-and.1 Re-anchor the ten findings and reproduce C3 on the unmodified tree

## Description
Gate for every other task (R1): re-anchor C1 to C4 and E1 to E6 at the start commit, and reproduce C3 with a Runner test. This is the spec's early proof point. The runtime reproductions (C2, E3) are task 2 because they need a built toolchain and fixture programs.

**Size:** M
**Files:** `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/reanchor.md` (new), `tools/gomad3/runner/runner_test.go`, the Status column of the spec's findings table (through `flowctl spec set-plan`)
**Touches:** [.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/reanchor.md, tools/gomad3/runner/runner_test.go, .flow/specs/fn-114-gomad-correct-search-path-defects-and.md, .flow/tasks/fn-114-gomad-correct-search-path-defects-and.*.md]

### Approach
- Record the commit the re-anchoring was taken at. For each finding write one row: current file and line, verdict (`confirmed`, `changed`, `refuted`), and the evidence read.
- The planning pass on 2026-10-01 read all ten as confirmed (targets below). Treat that as a starting point and re-check each against the start commit; other specs edit the same files.
- Resolve the three evidence references the planning pass could not confirm, and record the path or the correction:
  - E2: the 155 to 179 MB binary sizes are in `.plans/GOMAD_CMP.md:85-86`; the README states only the total at `tools/gomad3/README.md:334`.
  - E3 and E4: the retained D14 and D21 reports are cited only by `docs/research/gomad/2026-10-01-feasibility-schedule-search.md:424` and `:739`. Locate the report files and re-derive 26,865 of 57,801 and the 26 branching decisions, or mark the counts unverifiable.
  - C2: no identity scheme version constant exists. Versioning is the `/v1` label strings and `gomadChoiceWireVersion`.
- C3 reproduction: a characterization test in the choice-exploration group of `runner_test.go` whose injected executor returns a choice replay divergence for one forced-prefix candidate of a round. Assert today's behavior: the campaign ends in a `HostError`, the round is not committed, the results of the sibling candidates, which had already completed, are discarded with it, and no candidate evidence is retained. Siblings are not cancelled: the round collects every completion before the divergence is classified. Task 4 inverts the assertions.
- Closure rule for a refuted or changed finding: write the evidence in `reanchor.md`, set the spec Status cell, and note in the owning task (through `flowctl task set-spec`) that it closes with no code change or with a narrowed scope. Do not delete tasks.
- No production code changes.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/internal/corpus/model.go:37-47`, `:98-147` — C1 identity and projection
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:750-803` — C2 identity assignment, E4 run-queue choice
- `tools/gomad3/runner/choice_exploration_campaign.go:250-326` — C3 completion loop and runner-domain branch
- `tools/gomad3/target/target.go:828-851` — C4 provenance checks
- `tools/gomad3/runner/seeds.go:27-84` — E1 selection
- `tools/gomad3/runner/internal/exploration/choice/engine.go:349-384` — E5 depth rule, E3 expansion
- `tools/gomad3/runner/minimize_operation.go:73-136`, `:189-196`, `:487-498` — E6 session lifetime

**Optional** (reference as needed):
- `tools/gomad3/artifact/publication.go:44-53` — E2 target payload
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:338-342`, `:580-592`, `:689-697` — C2, E4, E3 call sites
- `tools/gomad3/runner/runner_test.go:1064-1135` — resume and failure tests to mirror for the C3 test
- `tools/gomad3/runner/internal/execution/choicetrace.go:19-42` — the divergence error the C3 test injects

### Key context
- `tools/gomad3/runner/retention_characterization_test.go` has uncommitted edits from another session. Do not touch it.
- The C3 trigger is inferred in the spec. If no injected divergence can reach the runner-domain branch, C3 is `changed` or `refuted`; record which path it takes instead.
## Acceptance
- [ ] `reanchor.md` holds the start commit and one row per finding C1 to C4 and E1 to E6 with file, line, verdict, and evidence
- [ ] The D14 and D21 report paths are recorded and the E3 and E4 counts re-derived, or the counts are marked unverifiable with the reason
- [ ] The three evidence corrections (E2 sizes, C2 version constant, report locations) are recorded
- [ ] A Runner test reproduces C3 on the unmodified tree and asserts the `HostError`, the uncommitted round, the discarded results of completed siblings, and the missing candidate evidence
- [ ] The spec findings table Status column matches the verdicts
- [ ] Each refuted or changed finding has its owning task annotated with the closure or the narrowed scope
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/...` passes with no production file changed
## Done summary
Re-anchored all ten search findings against HEAD `1d7272e654f268f9a45f3fe965918fe2522827c6` and the bound uncommitted production files. Eight findings are confirmed. C3 is changed in trigger location with its lost-round symptom reproduced; E4 is changed in the historical two-user premise with its unfiltered alternatives and reported 26 branching decisions confirmed. No finding is refuted and no downstream owner closes on this evidence.

`TestRunChoiceExplorationDivergingPrefixDiscardsCompletedRound` characterizes a typed forced-prefix executor divergence returning `HostError{Reason: "target_supervision"}` after all siblings finish. Only the root commits; the failed round has no segment or candidate execution records and publishes no typed divergence evidence. Raw partial output heads, state, candidate/prefix staging data, and work directories survive for recovery. Production code and the protected retention characterization test are unchanged.

Corrected E2's binary-size reference to `.plans/GOMAD_CMP.md:85-86`, distinguished identity `/v1` domains from choice-wire version 2, and located D14/D21 reports. Re-derived D14's 30,936 runnable + 26,865 select-poll = 57,801 decisions and D21 seed 11's 26 branching Runnable decisions. These are historical report counters, not current-toolchain measurements; D21 seed 17 has 29 decisions and its control source starts no goroutine explicitly. Source and reference hashes are retained with `reanchor.md`.

The pre-edit choice baseline, ten C3 repetitions, isolated existing retention case, final complete `./runner/...` gate, focused vet, formatting, and diff checks pass on darwin/arm64. The initial full Runner run hit the existing 100-job retention case's 10-second deadline at 97 executions; its isolated retry passed in 5.27 seconds and the final Runner package passed in 94.549 seconds. Both failure and recovery evidence remain. Root lint fails at default unavailable `main`, then at nested-module discovery with `GOLANGCI_LINT_BASE_REV=HEAD`; it is not reported clean. No runtime rebuild was performed and Linux remains unverified.

The spec and task 4/task 13 annotations were applied through Flow. Implementation review returned SHIP with no findings and task-scoped R1 met; review-round1.json and review-round1.md retain the verdict, and parent-review-bindings.json binds the reviewed files. Runtime reproductions remain task 2. No commit, staging, push, worktree, or unrelated source modification was performed.

stage: impl-review - ran (model: gpt-6-astra at high) - SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep ./runner -run '^TestRunChoiceExploration' -count=1, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep ./runner -run '^TestRunChoiceExplorationDivergingPrefixDiscardsCompletedRound$' -count=10 -v, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep ./runner -run '^TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs/100_jobs/discard$' -count=1 -v, env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep ./runner/..., env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT tools/gomad3/.toolchain/bin/go -C tools/gomad3 vet -tags test_dep ./runner/..., git diff --check -- tools/gomad3/runner/runner_test.go, env -u GOROOT /Users/stephan/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin/gofmt -l tools/gomad3/runner/runner_test.go
- PRs: