---
satisfies: [R7, R8]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.5 Retain bounded immutable measurements without overwriting or redispatch

## Description
Retain bounded immutable measurements without overwriting or redispatch.

**Size:** M
**Files:** tools/canary/store.go; tools/canary/store_test.go; tools/canary/result.go; tools/canary/result_test.go
**Touches:** [tools/canary/store.go, tools/canary/store_test.go, tools/canary/result.go, tools/canary/result_test.go]

### Approach
- Reuse artifactio.ImmutableDirectory for atomic manifest-addressed publication and validated reads. Include immutable measurement envelope, actual Run/Verdict and checksum/provenance; do not use replace-capable Publish for prior measurements.
- Enforce summary including provenance <=16KiB, measurement<=8MiB, count<=128 and aggregate<=256MiB. Reserve count/byte capacity before target dispatch; include staging/inflight data in accounting, bounded encoding and recovery scans. Reject exhaustion without deleting prior measurements or triggering execution.
- Use owned path-safe layout and process concurrency reservation; serialize same sink ownership with existing local locking patterns or reject second writer, without a lease service. Record a durable local measurement claim before invoking Run; duplicate or unresolved claimed identities cannot dispatch again. Crash leftovers count against capacity until explicitly reconciled; reclaiming space never authorizes redispatch.
- Late/duplicate identical publication is idempotent; conflicting publication rejects. Late durable evidence never overwrites failed orchestration status or another measurement. Publication error is observable and never requests redispatch. Test traversal/symlink, permission/disk errors, overflow, concurrent reservation, crash-stage and late publication. Storage management remains operator-owned; do not add a prune command.

### Investigation targets
**Required:**
- tools/common/artifactio/immutable.go:23,43,185 — bounded immutable publication/read.
- tools/common/artifactio/artifact.go:10 — replacement primitive to avoid for immutable records.
- tools/common/artifactio/set.go:649 — path-safety conventions.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./tools/canary ./tools/common/artifactio`
`mise exec -- go test -race -count=1 -tags test_dep ./tools/canary`
Add and run TestCanaryMeasurementRetention and TestCanaryLatePublicationIsolation explicitly.

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] All four finite caps apply to completed/inflight/staged storage and bounded Workflow summaries; capacity fails before dispatch.
- [ ] Publication is atomic, immutable and path-safe; failures/conflicts preserve earlier measurements.
- [ ] Concurrent/late/duplicate and crash-stage tests preserve identity/status isolation without redispatch.
- [ ] Returned summaries keep disposition, Verdict, cleanup and reporting status separate; full Run/capabilities never enter history.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
