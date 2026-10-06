---
satisfies: [R21]
---
# fn-126-read-each-feature-top-to-bottom-one.12 Cover composition stuck states and construct only needed witnesses

## Description
Close the original R21 coverage gap documented in task .9. The default stuck-state kind must inspect reachable states of machines and compositions, with the existing hole/unknown-behavior interpretation, and report every non-end state with no enabled action class and a shortest witness. A prior receipt's machine-only implementation was not an amendment of R21. Inspect available composition tables without silently excluding all compositions or using a failed refinement/property verdict to hide an otherwise constructible transition table. Keep genuine structural/build failures explicit through the existing lint/model diagnostics.

The standards audit also found that stuckStates eagerly constructs spellPath(table.PathTo(state)) for passing states. PathTo performs a fresh BFS. Count passing states without constructing witness arguments; compute the path only for a finding, preserving populations, messages, positions and shortest witnesses. Reuse the reader's existing composition/table/path machinery. If a minimal reader API is needed, keep it generic; do not implement the pending fn-124 package split here.

**Touches:** tools/umpire/lint/**, tools/umpire/model/**, model/README.md, model/ir/*.lint.json

Reader changes are limited to directly required composition/table/path helpers and tests. Lint acceptance changes require a justified finding on an existing deliberate control. Do not change a Model to silence a new finding. Report an actual Model bug to the conductor. Add a small composed deadlock regression and healthy/terminal/hole coverage. Measure the witness optimization on equivalent healthy inputs before claiming a speedup; no new profiling/caching framework.

Run focused red/green tests, current-IR lint and task-base read-only Go lint before per-task review. The expensive full model/Go/smoke gates run once after dependent task .13 at the next batch boundary. Reuse applicable .11 evidence for unchanged inputs and record this deferral explicitly.
## Acceptance
- [ ] Default stuck-state checks cover machine and composition reachable states. A composed non-end deadlock is reported even when no member is individually stuck; its shortest witness and declaration position are correct.
- [ ] Terminal, enabled and genuinely unmodeled/hole states do not produce false findings. Composition construction/refinement failures remain visible and do not become a blanket exemption for otherwise inspectable composition tables.
- [ ] PathTo and witness formatting run only for findings. Existing machine messages, populations and shortest-witness assertions remain intact; equivalent-input timing evidence supports any performance claim.
- [ ] Each newly exposed finding on current Models is accepted with a specific existing-design reason or reported as a Model bug; no Model behavior is fitted to silence the lint.
- [ ] Focused lint/reader tests, current-IR lint and read-only task-base Go lint pass. Full required batch gates are explicitly deferred to .13, and per-task implementation review reaches SHIP before verified Flow completion.


## Done summary
Stuck-state now checks reachable machine and composition tables, including constructible compositions whose replacement refinement is rejected or incomplete. Passing states are counted without PathTo or witness formatting; findings retain shortest paths, declaration positions and the existing machine message.

baseline: green — pre-edit focused lint/model command exited 0; `.flow/tmp/fn-126/task12/baseline.md` records the command and observations. No full baseline handoff was claimed. Task base: `4e0c4c5b3783e5cfcd4e038d113d615d9b259474`.

R21 evidence:

- `TestCompositionStuckState` pins a composed non-end deadlock with individually terminal members, the shortest replayable witness, declaration position and population; its terminal variant and enabled start have no finding. The original `TestStuckState` machine populations and exact message remain pinned, with the composition population added.
- `TestCompositionStuckStateWithUnknownReplacement`, `TestCompositionStuckStateWithRejectedReplacement` and `TestStuckStateWithRefinementMapHole` retain available transition tables without hiding incomplete/rejected refinement receipts. `TestCompositionStuckStateConstructionFailure` retains explicit malformed-end failure. Reader tests preserve fingerprints, unknown pairs, ceilings and name errors.
- Red reproductions are committed at `d6126d1a92` and `2be6da9f7f`. The initial composition omission failed on absent populations; the review regression failed for both disk and detailedPair on the refinement-map hole. Their logs are `composition-red.log` and `map-hole-red.log` under `.flow/tmp/fn-126/task12/`.
- The final affected checks all exited 0: focused lint/model readers (`focused-review-fix.log`, `focused-review-fix.exit`), all lint-package tests (`lint-package-review-fix.log`, `.exit`), current-IR lint (`current-ir-lint-review-fix.log`, `.exit`) and read-only task-base Go lint (`lint-code-fast-review-fix.log`, `.exit`). Go lint uses `GOLANGCI_LINT_BASE_REV=4e0c4c5b3783e5cfcd4e038d113d615d9b259474 GOLANGCI_LINT_FIX=false`. The selected tests were inspected to exclude an empty selection; git diff --check passed.

Current-IR lint exposed no new unaccepted finding. Existing machine findings remain, including the accepted ackByOriginal stuck-state control. No Model behavior or acceptance sidecar was changed. No speedup claim is made: the optimization is structural, and equivalent-input performance was not measured.

Full model, full Umpire Go and smoke gates are DEFERRED to dependent task .13 under the owner-approved .12/.13 batch boundary. The cumulative diff classifies full, not docs-only; no focused command minted a full-gate receipt. Unchanged Scala/generation evidence remains in `.flow/tmp/handovers/fn-126.11-summary.md` and task11 logs. The early inconclusive observations and test-source typo are retained in baseline.md; they were not counted as green.

stage: impl-review - ran [2026-10-06 07:13:30 UTC..2026-10-06 07:19:34 UTC] — SHIP after one NEEDS_WORK fix, explicit codex:gpt-6.1-sol:high. This is the same Codex family as the implementer, not independent-family review. Receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-126-read-each-feature-top-to-bottom-one.12.json`. First-round axes, merge and resumed output are retained under `.flow/review-fanout/e7d50430c1b5432295edd5ba2a16c183/` and `.flow/tmp/fn-126/task12/review-*`. Reviewer-side test execution was blocked by its read-only sandbox; the worker supplied the observed scoped exits.
stage: memory-capture - ran — `bug/integration/transition-table-reads-must-omit-2026-10-06`, after the non-trivial refinement-metadata fix reached SHIP.
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(policy: bridge inactive in `.flow/tmp/fn-126/task12-sync-active.json`)

Tier: session (jev-unavailable(no_key))

Prior-fix search: local history contains machine-only stuck-state and hole handling; no known-good composition implementation was found. GitHub issue/PR searches were inconclusive (401 credentials), so bisect was skipped without a known-good revision. No feature-map route changed. Parent spec completion, task13 full gates and MILESTONES update remain with the conductor.
## Evidence
- Commits: d6126d1a924f71cf7963ad6daae4d15192fb5ea3, 285fbbc519e290c527384a92ca572facc9170dca, 2be6da9f7f3ff0f9edda17ba7d18742bce05bccf, d8224bfc1148107e2e216112bbc2f76354be56c2, fdf4c10b61805e8fcf3a4fcb4e490ab5cb2c43d9
- Tests: go test -count=1 -tags test_dep -p 2 -timeout 10m ./tools/umpire/lint ./tools/umpire/model -run 'TestCompositionStuckState|TestStuckState|TestAComposedReading|TestAMembersHoleIsAnUnknownPairOfTheComposition|TestAComposition|TestChecking|TestADeclarationHoleStaysWithWhatDependsOnIt', go test -count=1 -tags test_dep -p 2 -timeout 10m ./tools/umpire/lint, make umpire-check-lint, GOLANGCI_LINT_BASE_REV=4e0c4c5b3783e5cfcd4e038d113d615d9b259474 GOLANGCI_LINT_FIX=false make lint-code-fast, git diff --check, DEFERRED: full model/Go/smoke gates to dependent task .13 under owner-approved batching; no full-gate receipt minted
- PRs: