---
satisfies: [R2, R6]
---
# fn-166-gomad-readable-package-structure-and.2 Consolidate runner tests by behavior

## Description
Split runner_test.go by Runner sub-responsibility, merge one-test and per-bug files into the file covering their behavior, and retire legacy characterization only where current behavioral tests cover the same cases.

## Acceptance
runner has at most 25 test files following the R2 convention; the set of test names and assertions is preserved or each removal is mapped to its covering test in the task notes; Make/CI -run selectors are unchanged in effect.

## Done summary
Runner tests organized by behavior: 54 to 25 test files, every test and assertion kept (line-by-line source check), six characterization files merged into characterization_test.go; Make and CI selectors unchanged; Runner failure set unchanged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2424c71bd9
- Tests: go test ./runner failure-set comparison, CI gate routing test, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: