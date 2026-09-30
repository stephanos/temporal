---
satisfies: [R1, R11]
---
# fn-107-gomad-finish-downstream-cell.6 Implement bounded CDS auxiliary stores and in-process WAL

## Description
Implement profile-scoped bounded auxiliary storage in ../saas-temporal, independently of Walker profile wiring. Use new test_dep support packages cds/storage/memory and cds/stream/memory; preserve existing scrap harness defaults. Metadata and shard stores must implement actual CAS, locking, immutable row copies, conditional deletion, context cancellation and bounded capacity. Watermark storage retains deterministic ordered history and exact LT/GTE/latest-N/range-delete semantics. The in-process stream.Dialer must supply real record write/read/paging, fencing metadata, delete/recreate, reopen lifetime, cancellation and resource limits; exercise it through the existing GRPC stream provider, not a success-only RPC fake. Files/Touches: cds/storage/memory/**, cds/stream/memory/** only. Quick: focused native mise run test (test_dep, timeout 3m), gomad-tag native checks and focused lint. No commits or staging. Work in an isolated worktree authorized by the user; return source patch/files and evidence for conductor integration/review.

## Acceptance
Stateful auxiliary stores and WAL preserve ownership/CAS, cloned records, real stream fencing/readback/reopen and deletion. Isolated tests cover stale/concurrent CAS, cancellation, capacity failures without partial state, watermark ordering/boundaries, record mutation isolation, close/re-dial, fencing and unsupported operations. Existing scrap fixtures and native defaults remain unchanged.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
