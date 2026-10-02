---
satisfies: [R2, R12, R18, R19]
---
# fn-115-make-the-scala-model-the-model-and.7 Separate reader admission concerns and trim live checker and lowering surfaces

## Description
Separate reader admission concerns and trim live checker and lowering surfaces. Implements R2, R12, R18, R19 using the reviewed parent contracts.

**Size:** M
**Files:** tools/umpire/model loader/validator and internal checker; tools/umpire/lower; focused tests
**Touches:** [tools/umpire/model/**, tools/umpire/lower/**, tools/umpire/conformance/**, tools/umpire/export/**, tools/umpire/explore/**]

### Approach
- The conductor owns shared module-map and migration-manifest updates after this task returns; return any required changes in the task-specific handover instead of editing those shared files.
- Split loading from admission/validation along the existing concerns identified in load.go; preserve diagnostics and error ordering rather than adding validation. Give each concern focused tests through the intended interface.
- Audit exports and actual production/test callers in the moved checker and lowering. Remove unreachable implementation, unexport package-only declarations and merge forwarding-only helpers; retain functionality with live consumers.
- Record live tooling line counts and the exported-surface inventory before/after. Keep aliases only where they implement the reviewed reader surface, not as historical package compatibility wrappers.
- Run mutation-sensitive goldens and all reader/lowering/conformance/export/exploration callers after each coherent change.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/goir/load.go:18`
- `model/scalav2/goir/load.go:41`
- `model/scalav2/goir/load.go:1552`
- `model/scalav2/goir/load.go:1891`
- `model/go/umpire`
- `model/go/caseproducer`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/...; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.

## Acceptance
- [ ] Loading and validation have separate concern files and focused tests without changed admission/diagnostic behavior.
- [ ] Every remaining exported checker/lowering declaration has a real outside-package caller or is removed/unexported; no forwarding-only package remains.
- [ ] Before/after line counts and public interface inventory are recorded, and all goldens and affected callers pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
