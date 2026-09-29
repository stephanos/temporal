---
satisfies: [R2]
---
# fn-101-gomad-f7-any-functional-test-and-ci.2 Fix the ./tests divergences and verify on the affected suites

## Description
Fix the divergences found in ./tests suites (signal-chasm, signal-legacy, nexus-otel replay, completion callbacks — the shared GC-scan-order channel; versioning-functional's in-memory FS bound) in Gomad, with regression fixtures, and verify on those suites individually (both seeds, repeat, replay) plus the core set and the representative set. Demonstrated test bugs keep named, owned exclusions. Do NOT run the full ./tests set (2026-09-29 user decision: not scalable); the sharding/merge and caches that exist stay as tooling.
## Acceptance
- darwin report meets counts

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
