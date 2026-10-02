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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
