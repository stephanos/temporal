---
satisfies: [R8, R6]
---
# fn-166-gomad-readable-package-structure-and.8 Delete dead helper profile, network v1 and choice v1 (rotation batch)

## Description
Delete the Rust helper diagnostic-profile feature, WASM network v1 (Go and Rust) and the legacy choice-trace v1 decoder as specified in R8. Land in the same integrated candidate as fn-153, fn-154 and fn-166 tasks 6-7.

## Acceptance
No diagnostic-profile feature, network v1 model or choice v1 decoder remains; README model table drops Network v1; v1 traces are refused by name; Rust fmt/clippy/tests and the full WASM gate set pass on the integrated candidate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
