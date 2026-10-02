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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
