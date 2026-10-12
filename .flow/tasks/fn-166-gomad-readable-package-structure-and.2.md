---
satisfies: [R2, R6]
---
# fn-166-gomad-readable-package-structure-and.2 Consolidate runner tests by behavior

## Description
Split runner_test.go by Runner sub-responsibility, merge one-test and per-bug files into the file covering their behavior, and retire legacy characterization only where current behavioral tests cover the same cases.

## Acceptance
runner has at most 25 test files following the R2 convention; the set of test names and assertions is preserved or each removal is mapped to its covering test in the task notes; Make/CI -run selectors are unchanged in effect.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
