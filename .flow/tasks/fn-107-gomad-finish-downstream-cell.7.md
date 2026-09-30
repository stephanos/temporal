---
satisfies: [R1, R2, R6, R11]
---
# fn-107-gomad-finish-downstream-cell.7 Add injectable CDS factory and host source seams

## Description
Implement the injectable CDS persistence factory and host source seams in ../saas-temporal independently of Walker profile wiring. Reuse shard.NewController, HistoryAppController, NewExecutionStoreWrapper and existing Walker provider conversion/interceptors. Constructor accepts typed CDS shard/metadata/watermark stores, stream.Dialer for both configured MS/HE WALs, validated Walker shardspace/count/config used to construct the real per-shard Walker provider and a borrowed reliable walkabout client, namespace registry and borrowed seeded SQL base factory. Supply CDS ShardStore and ExecutionStore as a pair; delegate SQL auxiliary stores. Validate Walker mode, positive HEWAL generation, matching shard count/space, required providers/config and lifecycle failures. Owned controller/registry/metadata handle cleanup is idempotent and never closes borrowed SQL. Add optional WAL dialer injection preserving native defaults. Separate native eager Cassandra/SMS/BOSS, full server FX, cloud blob/provider construction and metering type imports from Gomad closure with paired sources; preserve comments and ordinary API behavior. Files/Touches: cds/export/cds/**, cds/shard/wedge_wals.go and dialer seam files, cds/stream native dialer seams, cds/config metering seams, root blobstore/**. common/testx and cds/daemon signal seams are owned by fn107.2: do not edit those. New cds/storage/memory and cds/stream/memory support packages are owned by fn107.6: use existing interfaces and mocks for isolated constructor tests. Quick: focused native mise run test with test_dep and timeout 3m, gomad-tag compile/source checks and focused lint. No commits/staging. Isolated worktree authorized; conductor integrates and reviews.


Prefer constructing MultiWalkerStoreProvider from validated identity over adding getters to its private fields. A supplied namespace registry is borrowed and must never be started/stopped/closed by the factory; an absent one is constructed from a distinct seeded SQL metadata handle and owned together with that handle.

Fresh source inspection extends narrow seam ownership to cloudmetricshandler/events (native Chronicle encoder alias versus typed in-process logging), common/common.go role-assumer construction (unchanged native extraction), and cds/persistence/pendingbranch plus the existing pending_branch_delete.go delegating public wrappers. Preserve pending-delete sentinel/context-key identity rather than disabling its semantics. common/testx and cds/daemon remain fn107.2-owned. The injected profile rejects enabled metering; native storage-calculator behavior is preserved in paired source.

The same narrow cloud metrics source seam also covers cloudmetricshandler Chronicle config/recorder and shared interfaces/mocks: preserve native Chronicle construction and exported native type aliases; retain real Prometheus/in-memory metrics and logging under gomad, with explicit Unimplemented for native cloud construction. This removes the test_dep history utility closure without changing its assertions or the CDS app.
## Acceptance
A tested typed injectable factory uses real CDS controller/wrapper and the supplied Walker provider, without constructing external auxiliary services. Missing resources, incompatible config and constructor/lifecycle failures propagate. Borrowed SQL lifecycle and native default constructor behavior are preserved. Source-selected Gomad paths reject unavailable native construction without retaining forbidden downstream cloud/signal/process imports.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
