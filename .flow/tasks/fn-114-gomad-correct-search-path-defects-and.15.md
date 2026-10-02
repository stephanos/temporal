# fn-114-gomad-correct-search-path-defects-and.15 Scope minimizer workspace state per parent artifact
## Description
Follow-up to task 8 (R11). Task 8 keeps minimizer state under `OUTPUT/.minimize`, and the default output root `ARTIFACTS/minimized` is shared by every parent artifact. A run that ends in an error, or is killed, therefore blocks minimizing any other artifact under that root until it is resumed or the directory is removed.

**Size:** S
**Files:** `tools/gomad3/runner/internal/minimizer/workspace.go`, `workspace_unix_test.go`, `tools/gomad3/runner/minimize_operation.go`, `minimize_operation_test.go`
**Touches:** [tools/gomad3/runner/minimize_operation.go, tools/gomad3/runner/minimize_operation_test.go, tools/gomad3/runner/internal/minimizer/**, tools/gomad3/cmd/gomad/internal/cli/**]

### Approach
- Key the workspace by the parent artifact's record hash (for example `OUTPUT/.minimize/<parent-record-sha256>/`), so state, lock, and last accepted artifact for one parent never collide with another parent's.
- Keep every task 8 guarantee per parent: exclusive lock, refusal without `--resume` when that parent's state exists, clear error for `--resume` with no state for that parent, no repeated attempt, and no second publication.
- Two different parents can minimize into the same output root concurrently; two runs for the same parent still exclude each other.
- State written by task 8's layout was never released, so no migration is needed; say so in the done summary.
- While here, add the test the task 8 review asked for: a kill between the final publication and the checkpoint that records it, resumed without a second publication.

## Acceptance
- [ ] A failed or killed run for parent A leaves state that does not block a fresh `minimize` of parent B into the same output root
- [ ] `minimize --resume` for parent A continues from A's state and repeats no evaluated attempt
- [ ] Two concurrent runs for the same parent are still mutually excluded by the lock
- [ ] A kill between final publication and its checkpoint resumes without a second publication, shown by a test
- [ ] `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` and `make -C tools/gomad3 validate` pass

## Done summary
Minimizer state is now kept per parent artifact. State, lock, and the last accepted artifact for a parent live under `OUTPUT/.minimize/sha256-<parent record hash>/`, with the lock beside it as `sha256-<parent record hash>.lock`. An interrupted or failed run for one parent no longer blocks `gomad minimize` of another parent into the same output root, and two different parents can minimize into one root at the same time.

What a user sees:

- A plain `minimize` is refused only when state exists for that same parent (`ErrCheckpointExists`; the message names the per-parent directory to remove).
- `minimize --resume` for a parent with no state fails with `ErrNoCheckpoint`, also when the root holds another parent's state. Before this task that case reported "different parent artifact".
- A second run or resume for the same parent while one is active still fails with `hostfs.ErrContended`.
- Every task 8 guarantee holds per parent: no repeated attempt, budget and implementation checks, fail-closed on a damaged accepted artifact, one publication.

No migration: task 8's layout (`OUTPUT/.minimize/state.json`, `OUTPUT/.minimize.lock`) was never released. State left in that layout is ignored, not reinterpreted. The checkpoint schema and bytes are unchanged.

How it holds together:

- `minimizer.OpenWorkspace` derives the directory from the binding's parent record hash and takes that parent's lock before reading or writing state. `load` still compares the stored parent binding, so a state directory moved under another parent's name fails closed.
- `.minimize` must be a real directory; a symbolic link is rejected before the lock file is created.
- `Minimize` is split into `reduce` (the attempt loop) and `publishAccepted` (the final publication without its checkpoint). The split is the seam the task 8 review asked for; behavior is unchanged.

Tests, one per acceptance item:

- A's leftover state does not block B, and A then resumes without repeating an attempt: `TestMinimizeKeepsStatePerParentArtifactInOneOutputRoot`, `TestWorkspaceKeepsStatePerParentArtifact`.
- Same parent excluded, another parent runs to completion meanwhile: `TestMinimizeExcludesConcurrentRunsOfOneParentOnOneOutputRoot` (initial runs, resumes).
- Kill between final publication and its checkpoint: `TestMinimizeResumeAfterUnrecordedFinalPublicationDoesNotPublishAgain`. It passes on task 8's code too, because the record-keyed store already returned the existing artifact; mutation M6 turns it red.
- `TestWorkspaceResumeRejectsStateMovedFromAnotherParent`, `TestWorkspaceRejectsSymbolicLinkStateRoot`.

Two existing assertions changed by intent: the "changed parent artifact" rows of `TestMinimizeResumeRejectsStateOfAnotherRun` and `TestWorkspaceResumeRejectsStateOfAnotherRun` now expect `ErrNoCheckpoint`, because another parent's state is no longer visible to the resume. The binding check they used to reach is covered by the moved-state test.

Gates on darwin/arm64: `test-host` green at cd3a6fd34 (45 packages, 194 s, run once), `validate` green, architecture test, vet, and `-race` on the Minimize and Workspace tests green. Six mutations each turned the matching tests red.
GATE_SKIPPED:unittest:green-receipt cd3a6fd3 - Verify at 6299870b0 reused the post-commit pass; only .flow evidence files changed after it
Not run: linux/amd64 (no native host); root `make lint-code-fast`; the spec's literal Quick command with the default PATH go; the qualification sets (task 14).

Review: SHIP on the first round from claude-fable-5-1 at high through the `claude` backend (same family as the writer; the reviewer had no shell and relied on the committed gate evidence). Two P3 notes are left open:

- One empty lock file per parent stays under `.minimize/` after completion. Unlinking a `flock` file is racy, so they are kept. A `--resume` with no state also creates one when `.minimize` exists.
- No test overlaps two parents in the final publication itself. That path relies on the artifact store's unique staging directory and rename-without-replace.

For task 14's docs: the per-parent layout above, the lock files that remain, and README line 163, which still says minimizer resume is not implemented.

Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-15/`.

stage: impl-review - ran (model: claude-fable-5-1, verdict SHIP, 2026-10-02T17:45Z)
## Evidence
- Commits: cd3a6fd34ded809eb44963f6260165f5b36c7d45, 6299870b04f2c98f22c3c279334818a4143a6994
- Tests: baseline: green via handoff (verified at 46083d3d by fn-112.14; only .flow evidence commits since), GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host (cd3a6fd34, exit 0, 45 packages ok, 194 s, pinned stock go1.27.1 first on PATH, run once), GATE_SKIPPED:unittest:green-receipt cd3a6fd3 - Verify at 6299870b0 reused the post-commit pass; only .flow evidence files changed after it, make -C tools/gomad3 validate (cd3a6fd34 and 6299870b0, exit 0), .toolchain/bin/go test -tags test_dep -count=1 ./runner/internal/minimizer/ ./cmd/gomad/... (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -run TestMinimize ./runner/ (exit 0; exit 1 before the change, see red-before-change.txt), .toolchain/bin/go test -tags test_dep -count=1 -run TestPackageArchitecture . (exit 0), .toolchain/bin/go vet -tags test_dep ./runner/ ./runner/internal/minimizer/ ./cmd/gomad/... (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -race -run TestMinimize|TestWorkspace ./runner/ ./runner/internal/minimizer/ (exit 0), six mutation checks, each red on the named tests (final-gates.json), NOT RUN: linux/amd64 (no native host), NOT RUN: root make lint-code-fast (cannot typecheck the nested module; inconclusive in task 8), NOT RUN: go -C tools/gomad3 test ... (literal spec Quick command with the default PATH go; test-host covers the same packages)
- PRs: