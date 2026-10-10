---
satisfies: [R1, R6, R7, R8]
---
# fn-152-gomad-runner-storage-on-one-append-only.6 Replay corpus admissions and evictions into the live hash index

## Description
Implement R6 and the corpus-specific R1/R7/R8 controls. This owner is a parallel candidate with campaign conversion after shared contracts land.

**Size:** M
**Files:** Core files (4-5): corpus.go, admission.go, model.go, lock.go if superseded, and corpus replay/fault tests. Wider touches: guidance tests and fixture helpers if snapshot projection changes; shared target integration reuses task 2.
**Touches:** [tools/gomad3/runner/internal/corpus/**, tools/gomad3/runner/guidance.go, tools/gomad3/runner/guided*_test.go]

### Approach

Replace corpus.json with the same log primitive. One admission transaction records the replay-verified entry plus the committed eviction delta and new immutable snapshot identity. Rebuild only the bounded live hash index while streaming historical records; validate final live artifacts after replay. A live hash is a no-op before publication, and a hash released by committed eviction may be replay-verified and admitted again. Preserve selection/ranking, 1,024 live entries, 1 GiB live payload accounting, shared target charges and campaign frozen snapshots. Historical log size is a separate diagnostic and grows with churn until normal host storage failure. Do not charge it against the live payload budget, invent a cap/flag or retain evicted hashes forever. Bound each decoded frame with checked arithmetic. Recovery reclaims only final-state unreachable owned case files.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/corpus/corpus.go:138`
- `tools/gomad3/runner/internal/corpus/admission.go:21`
- `tools/gomad3/runner/internal/corpus/model.go:34`
- `tools/gomad3/runner/internal/corpus/guide_test.go`
- `tools/gomad3/runner/guidance.go`
- `tools/gomad3/artifact/target_pool.go:103`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/corpus.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R6: replay rebuilds the live in-memory hash index, duplicate live admission is a no-op, and eviction followed by replay-verified readmission survives reopen with the same snapshot semantics.
- [ ] R6/R7: entry plus eviction membership is one durable transaction; missing final live payloads are corruption while committed eviction permits cleanup of retired payloads, including shared-target accounting.
- [ ] R6: count/ranking/selection and 1 GiB live payload capacity remain unchanged; history replay streams records and log bytes are reported separately with no history cap, new flag or compactor.
- [ ] R1/R6: append/sync failure, reopen, parallel writer contention, identity mismatch and artifact-publication crash controls preserve the prior live snapshot and cleanup reachability.
- [ ] R8: old corpus layouts and malformed/unknown new lineage fail clearly before mutation; parked largest-corpus memory/replay/log-growth measurement stays empirical unless actual retained input exists.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
