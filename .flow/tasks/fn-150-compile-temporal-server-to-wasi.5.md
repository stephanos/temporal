---
satisfies: [R2, R4]
---
# fn-150-compile-temporal-server-to-wasi.5 Align nested mixedbrain dependencies after SQLite driver replacement

## Description
Fix the mixedbrain CI startup failure caused by older nested-module x/sys and x/text selections after replacing the root SQLite driver. Keep the correction in the second PR and merge it into the third without rewriting open PR history.

## Acceptance
- Update the nested module and sums using go mod tidy.
- Compile the mixedbrain test binary with test_dep, race detection, coverage and read-only module resolution.
- Publish the correction on the existing stacked draft branches using ordinary pushes.

## Done summary
Reproduced mixedbrain nested-module tidiness failure caused by root x/sys and x/text upgrades. Updated nested requirements and sums in commit601f430d86, pushed PR9 normally and merged into PR10 preserving history. Race/coverage test binary compilation with test_dep and -mod=readonly passed; nested go mod tidy -diff and git diff --check passed. Full PostgreSQL mixedbrain execution was not performed locally. Gomad compatibility packs remain unreconciled and qualification unverified.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 601f430d86, 792dedb958
- Tests: CGO_ENABLED=1 go test -mod=readonly -tags disable_grpc_modules,test_dep -race -cover -c -o ../../.tmp/mixedbrain.test . (tests/mixedbrain), go -C tests/mixedbrain mod tidy -diff, git diff --check
- PRs: https://github.com/stephanos/temporal/pull/9, https://github.com/stephanos/temporal/pull/10