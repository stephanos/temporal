---
satisfies: [R2, R3]
---
# fn-157-bound-native-verification-memory-and.2 Repair the measured private native owner with isolation proof

## Description
Correct the private native owner demonstrated by task .1 (R2-R3). Split from consumer work because callers must adopt one measured ownership contract.

**Size:** M
**Files:** interpreter, checker, private engine and focused ownership/replay tests, only where .1 identifies the cause.
**Touches:** [tools/umpire/interp/**, tools/umpire/check/**, tools/umpire/internal/engine/**]

### Approach

- Read task .1's pins, causal profiles and frozen oracle before choosing a lifetime or representation correction. Existing Interpreter behavior sharing already keys types and Steps, shallow-clones outer slices, bypasses tracing and leaves failed evaluation uncached. Treat those as compatibility inputs, not a new optimization.
- Prefer private representation or bounded lifetime correction for the demonstrated owner. Preserve full finite domains and admission-before-allocation ceilings, ordered rows/results, per-owner declaration/error metadata and independent fresh witness interpretation. Do not have replay compare an object with itself or reuse the optimized helper as its oracle.
- Define ownership of supplied Model/declarations and returned nested values/tables, immutability or defensive copying, supported concurrent access, and invalidation or refusal after mutation. Bound functions' existing nonconcurrent contract stays explicit. Sharing cannot couple owners, traced/failed evaluation, separate interpreters or repeated calls.
- Extend focused tests for input and returned nested mutation, fresh instances, repeated runs, changed same-name type/function/Steps content, different Models, failed evaluations, tracing, replay after rebind and concurrent supported readers. Use the task .1 `native-preservation` harness against its frozen full outputs for every affected machine and receipt; establish full equivalence before consumer adoption.
- Measure the corrected phase on equivalent full inputs/settings and report before/after heap/allocation/RSS by scope. A capacity-only diagnosis warrants no speculative cache. If .1 points outside this declared owner seam, stop and re-anchor tasks with the conductor.

### Investigation targets

**Required:**
- `tools/umpire/interp/machine.go:279`, `tools/umpire/interp/eval.go:242` - sharing and exposed pointers.
- `tools/umpire/check/checking.go:250` - original and replay bindings.
- `tools/umpire/check/claims.go:156`, `tools/umpire/check/claims.go:294` - field augmentation and fingerprint memo.
- `tools/umpire/internal/engine/table.go:171` - mutable nested ownership.
- `tools/umpire/check/checking_test.go:1484` - identity and independent replay rejection.
- `tools/umpire/check/realizer_test.go:86` - pointer-specific memo tests do not prove in-place invalidation.

## Acceptance
- [ ] The measured owner has a causal private correction or documented capacity-only no-source disposition; no general cache framework or acceptance population change.
- [ ] Ownership, deep caller-mutation isolation, lifetime and supported concurrency/invalidation tests pass, including traced/failed paths and independent replay refusal.
- [ ] Task .1's frozen full oracle compares all affected ordered native values, metadata, receipts, errors and witnesses exactly. Before/after resource evidence uses equivalent full pins and unmodified runtime settings.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
