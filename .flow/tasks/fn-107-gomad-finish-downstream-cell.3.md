---
satisfies: [R1, R2, R6, R11]
---
# fn-107-gomad-finish-downstream-cell.3 Compose CDS persistence from injected auxiliary stores and remove host-only source dependencies

## Description
Integrate and verify the completed fn-107.2 profile, fn-107.6 bounded auxiliary resources and fn-107.7 injectable CDS factory/source seams in ../saas-temporal. Reuse their implementations without duplicate construction or rewrites. Supply real Walker per-shard stores and walkabout client, bounded in-process CDS metadata/shard/watermark/WAL resources, and the same borrowed SQL base seeded by testcore. Exercise paired OSS shard/execution stores and actual Walker acquisition/recovery; verify config errors, initialization failure cleanup, close/reacquire and borrowed-versus-owned resource lifetime. Confirm combined gomad/!gomad source selection excludes native eager Cassandra/SMS/BOSS/cloud/process/signal paths. Coordinate any integration-only changes across these surfaces after task2/6/7 are independently green and reviewed. Files/Touches: necessary integration tests and small composition/lifecycle adjustments across CDS/Walker boundaries; no duplicate auxiliary implementation. Quick: focused native mise tests with test_dep and timeout3m, gomad-tag compile/source checks and focused lint. No commits/staging; conductor-owned Flow review/done.
## Acceptance
A tested injectable CDS factory constructs real Walker backing without Cassandra, BOSS or etcd; auxiliary stores/WAL preserve state and CAS/record/recovery semantics. Existing production constructor defaults behave identically. Forbidden downstream source imports are eliminated at build time; invalid provider configuration and lifecycle errors propagate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
