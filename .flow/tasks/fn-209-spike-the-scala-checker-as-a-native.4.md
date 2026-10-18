---
satisfies: [R4]
---
# fn-209-spike-the-scala-checker-as-a-native.4 Go/no-go, recommendation, and the fn-203.3 / fn-204 recut on go

## Description
**Owner 2026-10-18:** the decision rule changes: go means the wasm module loads and snapshots in wasmtime (R3b); the sidecar targets are reported but do not decide. On no-go for wasm, record the exact host or toolchain blocker and whether it is on wasmtime's or the Scala toolchain's roadmap, since the simulation's host choice binds the Model's.

---
satisfies: [R4]
---
# fn-209-spike-the-scala-checker-as-a-native.4 Go/no-go, recommendation, and the fn-203.3 / fn-204 recut on go

### Description
Write the per-target verdict and one recommendation; on go, close fn-203.3 as unnecessary, recut fn-204 to the oracle-sidecar route with the descriptor as the rejected alternative, and update MILESTONES' Phase 4 row; on no-go, close this spec with the evidence and leave fn-203/fn-204 as filed.

**Files:** model/design/native-checker-spike.md, .flow/specs/fn-203-*.md, .flow/specs/fn-204-*.md, MILESTONES.md.

**Quick commands:** none

### Acceptance
- Decision recorded; dependent specs recut or left as filed accordingly.

### Done summary
TBD

### Evidence
- Commits:
- Tests:
- PRs:
## Acceptance
- Decision recorded; dependent specs recut or left as filed accordingly.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
