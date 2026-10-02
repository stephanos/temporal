---
satisfies: [R11]
---
# fn-114-gomad-correct-search-path-defects-and.8 Persist minimizer state and add minimize resume

## Description
E6 (R11): `minimize` writes its sealed state and last accepted artifact under the output directory after each commit, and a resume option continues from them. Depends on task 6 only for the shared CLI files; task 7 is a transitive dependency.

**Size:** M
**Files:** `tools/gomad3/runner/minimize_operation.go`, `minimize_operation_test.go`, `tools/gomad3/runner/internal/minimizer/minimizer.go`, `minimizer_test.go`, `tools/gomad3/cmd/gomad/internal/cli/cli.go`, `cli_test.go`
**Touches:** [tools/gomad3/runner/minimize_operation.go, tools/gomad3/runner/minimize_operation_test.go, tools/gomad3/runner/internal/minimizer/**, tools/gomad3/internal/hostfs/**, tools/gomad3/cmd/gomad/internal/cli/**]

### Approach
- The minimizer state is already sealed, self-validating, and JSON-encoded; persist it as is. Do not add a second state format.
- After each committed attempt, write the accepted artifact first and the state second, each by temporary file and rename, so the state never references an artifact that is not on disk.
- The state file binds the parent artifact's record hash, the attempt budget, the minimizer implementation identity, and the toolchain build key. Resume rejects a difference in any of them.
- Proposed option: `gomad minimize --resume`. Without it, an output directory that already holds minimizer state is refused. With it and no state present, fail with a clear error.
- Take an exclusive lock on the output workspace before checking for or loading state, and hold it through every checkpoint, the final publication, and its validation. Use the host lock the campaign resume path uses. A second `minimize` or `minimize --resume` on the same workspace is rejected while the lock is held, and the lock is released when the holder dies.
- Resume repeats no evaluated attempt: the attempt counter and the accepted list continue from the persisted state.
- Handle a kill between the final publication and its replay validation: resume detects the already published result and validates it, and does not publish a second time into a store-key collision.
- The scratch workspace for candidate targets can stay temporary. Only the state and the accepted artifact need to survive.
- Mirror the campaign journal's stage and commit discipline for write ordering and corruption checks.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/runner/minimize_operation.go:48-136` — session, loop, final publication and replay validation
- `tools/gomad3/runner/minimize_operation.go:189-196`, `:487-498` — temporary workspace and `close`
- `tools/gomad3/runner/internal/minimizer/minimizer.go:49-60`, `:121` — state, seal, and validation
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:42`, `:922-949` — usage and flags
- `tools/gomad3/runner/internal/campaign/choice_exploration_journal.go:55`, `:280`, `:364` — stage, commit, and resume pattern
- `tools/gomad3/runner/internal/campaign/resume_journal.go:31-37` — exclusive resume lock to reuse

**Optional** (reference as needed):
- `tools/gomad3/runner/minimize_operation_test.go:18`, `:67` — operation tests to extend
- `tools/gomad3/runner/internal/minimizer/minimizer_test.go:51` — state round-trip test
- `tools/gomad3/runner/resume.go:33-60` — campaign resume entry

### Key context
- fn-109 task 6 moves executor injection in `minimize_operation.go` and task 12 changes how it opens artifacts; rebase onto whichever landed.
- Typed scenario shrinking is out of scope (BUG-5 keeps it).
- CLI usage text changes here; README and CLI guide prose is written in task 14.
## Acceptance
- [ ] A run killed after an accepted reduction resumes, repeats no evaluated attempt, and publishes a result equal to the uninterrupted run's
- [ ] Resume with zero further acceptances still reports the result as changed
- [ ] Resume rejects a changed parent artifact, a changed attempt budget, a changed minimizer implementation identity, and a changed toolchain build key, each with a test
- [ ] A state file that references a missing or corrupt accepted artifact fails closed
- [ ] An output directory with existing state is refused without the resume option
- [ ] Two concurrent initial runs and two concurrent resumes on one workspace: exactly one proceeds and the other is rejected, each with a test
- [ ] A lock left by a killed process does not block the next resume
- [ ] A kill after the final publication and before its validation resumes without a second publication
- [ ] `go -C tools/gomad3 test -tags test_dep ./runner/... ./cmd/gomad/...` and `make -C tools/gomad3 validate` pass
## Done summary
Added minimizer resume (R11/E6). `gomad minimize` now checkpoints its sealed state after every committed attempt under `OUTPUT/.minimize/state.json`, and `gomad minimize --resume` continues from it. `minimizer.Workspace` (new, `runner/internal/minimizer/workspace.go`) owns the lock, the checkpoint, and the accepted-artifact store; `Minimize` drives it.

What a user sees:

- An interrupted run leaves state. A plain `minimize` on that output root is refused (`ErrCheckpointExists`: resume it or remove `.minimize`). `--resume` with no state fails with `ErrNoCheckpoint`.
- Resume rejects a different parent artifact, attempt budget, minimizer implementation, or toolchain build key. It also rejects a state whose starting candidate or exploration config differs from the parent's.
- A completed run removes its state, so the next `minimize` into the same root starts fresh.
- A second `minimize` or `minimize --resume` on a locked root fails with `hostfs.ErrContended`. The lock is the `flock` campaign resume uses (`OUTPUT/.minimize.lock`), taken before the state check and held through final publication and its replay validation.

Behavior change to know about: any run that ends in an error, not only a killed one, leaves state. The default output root `ARTIFACTS/minimized` is shared by all parents, so a failed run blocks minimizing other artifacts into that root until it is resumed or `.minimize` is removed.

How it holds together:

- Write order: the accepted artifact is published into `.minimize/accepted` first, then the state is written by temporary file and rename. `Workspace.Commit` refuses a reference whose directory is absent. Superseded accepted artifacts are pruned after the state commit and on resume.
- The checkpoint wraps the unchanged sealed `State` with the binding, the accepted-artifact reference, and the publication reference. It is canonical JSON with its own seal; load rejects a symlink, a non-canonical or edited file, and a missing accepted artifact. `Minimize` then opens the accepted artifact and checks its record hash and payloads before evaluating anything.
- The final artifact is always published from the retained accepted artifact, so an uninterrupted and a resumed run publish from the same bytes. Their record hashes are equal in the tests.
- The publication is recorded in the checkpoint. A run killed before validating it reopens the published artifact by record hash and only replays it.
- `Changed` comes from the persisted accepted reference, so a resume with no further acceptance still reports a change.
- `cloneState` now keeps an empty `Evaluated` list empty. It used to turn `[]` into `null`, which changed the state seal on the first clone of an initial state.

Tests, one per acceptance item:

- `TestMinimizeResumeContinuesAfterAcceptedReductionWithoutRepeatingAttempts`: resume after an accepted reduction evaluates one candidate, equals the uninterrupted outcome, reports `Changed`, refuses a plain run over the state, and allows a fresh run after completion.
- `TestMinimizeResumeRejectsStateOfAnotherRun` (parent, budget, no state) and `TestWorkspaceResumeRejectsStateOfAnotherRun` (parent, implementation, build key, budget, starting candidate).
- `TestMinimizeResumeFailsClosedOnDamagedAcceptedArtifact` (missing, corrupt payload) and `TestWorkspaceResumeFailsClosedOnDamagedState`.
- `TestMinimizeExcludesConcurrentRunsOnOneOutputRoot` (initial runs, resumes).
- `TestWorkspaceLockOfKilledProcessDoesNotBlockResume` kills a real lock-holding process.
- `TestMinimizeResumeAfterFinalPublicationValidatesWithoutPublishingAgain`.
- `TestRunMinimizeResumesOnlyOnRequest` (CLI flag).

Five mutations of the implementation each turned the matching test red; `final-gates.json` lists them.

Gates on darwin/arm64 at 0b5eb20f8: `test-host` green (45 packages, 169 s), `validate` green, architecture test, vet, and `-race` on the Minimize tests green.
GATE_SKIPPED:unittest:green-receipt 0b5eb20f - Verify at fd24d8264 reused the post-commit pass; only .flow evidence files changed after it
Inconclusive, not counted as passes: the first `test-host` attempt stopped in 2 s because the default `go` on PATH was 1.26.5 (rerun with the pinned go1.27.1 first on PATH); root `make lint-code-fast` cannot typecheck the nested module. Not run: linux/amd64 (no native host); the literal spec Quick command with the Homebrew go.

Review: SHIP on the first round from claude-fable-5-1 at high through the `claude` backend (same family as the writer; the reviewer had no shell and relied on the committed gate evidence). Two P3 notes are left open as follow-ups:

- The build-key binding is taken from the parent manifest, which the parent record hash already covers and preflight already checks against the pinned toolchain. Through `Minimize` that branch is unreachable; only the workspace test exercises it.
- No test covers a kill between the final publication and the checkpoint that records it. That window relies on the republished record being byte-identical, so the record-keyed store returns the existing artifact. A test needs a seam in `RecordPublication`.

For task 14's docs: the `--resume` flag, the `.minimize` state directory and lock file in the minimized output root, and the leftover-state behavior above. For task 9: each accepted reduction keeps one extra copy of the target under `.minimize/accepted` until the run completes.

Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-8/`.

stage: impl-review - ran (model: claude-fable-5-1)
## Evidence
- Commits: 0b5eb20f8fbf9d2ef36b367e3fa3afd2d505bcb7, fd24d8264e951f3594697275fc097dc0d97f7be1
- Tests: baseline: green via handoff (verified at baef81f6 by fn-114.6; only a test assertion in runner/internal/campaign and .flow files changed since), GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host (0b5eb20f8, exit 0, 45 packages ok, 169 s, pinned go1.27.1 first on PATH), GATE_SKIPPED:unittest:green-receipt 0b5eb20f - Verify at fd24d8264 reused the post-commit pass; only .flow evidence files changed after it, make -C tools/gomad3 validate (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -run TestPackageArchitecture . (exit 0), .toolchain/bin/go vet -tags test_dep ./runner/ ./runner/internal/minimizer/ ./cmd/gomad/... (exit 0), .toolchain/bin/go test -tags test_dep -count=1 -race -run TestMinimize ./runner/ (exit 0), INCONCLUSIVE: first test-host attempt at 0b5eb20f8 exited 2 after 2 s before any test ran (default PATH go resolved to go1.26.5 under GOTOOLCHAIN=local), INCONCLUSIVE: GOLANGCI_LINT_BASE_REV=619891e5c make lint-code-fast exited 2 - 0 issues reported but golangci-lint cannot typecheck the nested tools/gomad3 module, NOT RUN: go -C tools/gomad3 test -tags test_dep ./runner/... ./cmd/gomad/... (literal spec command; red before this task with the Homebrew go, covered by test-host), NOT RUN: linux/amd64 (no native host)
- PRs: