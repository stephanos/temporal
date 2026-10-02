---
satisfies: [R11]
---
# fn-114-gomad-correct-search-path-defects-and.8 Persist minimizer state and add minimize resume

## Description
E6 (R11): `minimize` writes its sealed state and last accepted artifact under the output directory after each commit, and a resume option continues from them. Depends on task 7 only for the shared CLI files.

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
