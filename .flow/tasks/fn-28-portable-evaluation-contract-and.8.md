---
satisfies: [R5, R6]
---

# fn-28-portable-evaluation-contract-and.8 Expose the resident executor through bounded HTTP protobuf transport
## Description

Add a thin long-lived HTTP adapter that accepts one protobuf execution request and returns one protobuf result while delegating all behavior to the executor module.

**Size:** M
**Files:** `tools/umpire/executorhttp/**`
**Touches:** [`tools/umpire/executorhttp/**`]

### Approach
- Use bounded protobuf request/response bodies, fixed routes and methods, request contexts, and explicit deadlines; expose no checker, model, executable, endpoint, credential, or retry selectors.
- Keep transport errors distinct from operational and semantic statuses; a client disconnect cancels owned work and cannot publish partial success.
- Prove one handler instance serves multiple sequential requests and rejects overlapping requests through the executor's atomic pre-I/O `busy` path without starting Go, Lean, Make, or shell subprocesses.

### Investigation targets

**Required** (read before coding):
- Parent executor and generated protobuf messages.
- Repository HTTP handler limits, error encoding, graceful shutdown, and `httptest` patterns.
- Existing Umpire structured error and bounded-capture conventions.

## Acceptance
- [ ] Exact bounded protobuf requests reach the executor and return canonical protobuf results with distinct transport/tooling statuses.
- [ ] Wrong method/path/content, unknown fields, oversized bodies, cancellation, deadlines, and malformed protobuf fail closed without semantic success.
- [ ] Sequential requests reuse one handler/executor instance and focused race/lint tests pass.
- [ ] Concurrent HTTP requests cannot bypass single-flight admission; the loser receives typed `busy` plus local `inconclusive` before runtime I/O.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
