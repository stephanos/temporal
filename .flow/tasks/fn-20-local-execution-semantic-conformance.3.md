---
satisfies: [R1, R2, R6]
---
# fn-20-local-execution-semantic-conformance.3 Implement the bounded Go checker process bridge

## Description
Implement the transport-only Go checker adapter and fixed sibling-process lifecycle (R1/R2/R6), independently testable with a fixture child.

**Size:** M
**Files:** `tools/umpire/conformance/checker.go`, `tools/umpire/conformance/protocol.go`, `tools/umpire/conformance/checker_test.go`, `tools/umpire/conformance/testdata/**`
**Touches:** [tools/umpire/conformance/checker.go, tools/umpire/conformance/protocol.go, tools/umpire/conformance/checker_test.go, tools/umpire/conformance/testdata/**]

### Approach
- Keep the checker transport interface and fixture injection package-private. The eventual exported production `Check(admittedSet)` resolves the fixed sibling internally and exposes no response-producing callback or implementation parameter.
- Encode exact non-path admitted bindings, separate Run/RawEvidence omissions, and the remaining direct request projection; strictly decode the exact response with bounded N+1 readers, canonical byte checks, closed enums, and complete echoed-binding validation.
- Resolve only a regular realpath-contained sibling checker, verify the handshake, and expose no path/env/PATH/plugin override.
- Run under one 30-second child context; on cancellation, timeout, malformed output, nonzero exit, or stderr bytes, terminate and reap before returning one sanitized error.
- Use a fixture child with independent literal responses to test every lifecycle/protocol failure without invoking semantic code.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.2.md` — bounded strict JSON kernel pattern
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.3.md` — canonical mutation/limit fixtures
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` §API Contracts — sanitized error and cancellation precedent
- `tools/umpire/regression/projection.go:201-225` — existing strict trailing/duplicate JSON precedent
- `tools/common/artifactio/set.go:475-645` — path/identity and interruption patterns
## Acceptance
- [ ] Valid fixture child round-trips exact request/response bytes, four artifact bindings, omission arrays, and all semantic identities.
- [ ] Missing, symlinked/misdirected/non-regular, wrong-handshake, timed-out, canceled, nonzero, stderr-writing, malformed, noncanonical, oversized, trailing, or stale-response children fail closed and are reaped.
- [ ] N and N+1 byte tests allocate only within the stated ceiling.
- [ ] No exported production API exposes a checker implementation, executable location, or semantic callback; injection remains package-private and test-only.
- [ ] Go race/fuzz tests cover protocol decoding and concurrent independent invocations.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
