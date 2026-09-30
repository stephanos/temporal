---
satisfies: [R1, R2, R6, R11]
---
# fn-107-gomad-finish-downstream-cell.3 Compose CDS persistence from injected auxiliary stores and remove host-only source dependencies

## Description
Work in ../saas-temporal. Add a bounded CDS composition seam that reuses shard.NewController, HistoryAppController, NewExecutionStoreWrapper and the existing Walker provider wrapper. Inject actual in-memory CDS shard/metadata/watermark/WAL implementations (existing scrap harness semantics) and delegate OSS namespace/cluster/matching/queue services to the same SQLite base factory. Override OSS ShardStore and ExecutionStore as a pair so shard acquisition/recovery still reaches Walker. Keep eager native Cassandra/SMS/BOSS construction separate. Isolate signal registration, repository-root subprocesses, CLI/cloud-provider closure imports and host collectors using gomad/!gomad paired sources as needed. Files/Touches: cds/export/cds/**, cds/shard/**, cds/daemon/**, common/testx/**, blobstore provider construction seams; coordinate any extra files. Quick: focused mise run test on changed CDS packages plus native factory tests/lint. Do not introduce fake-success Walker execution/history implementations.

## Acceptance
A tested injectable CDS factory constructs real Walker backing without Cassandra, BOSS or etcd; auxiliary stores/WAL preserve state and CAS/record/recovery semantics. Existing production constructor defaults behave identically. Forbidden downstream source imports are eliminated at build time; invalid provider configuration and lifecycle errors propagate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
