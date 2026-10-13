---
satisfies: [R1, R2, R3]
---
# fn-179-gomad-temporal-server-consistency.1 Soft-assertion oracle: soft assertions fail the Gomad run

## Description
Add a soft-assertion oracle that fails any Gomad-run Temporal workload in which `softassert.That`/`Fail` fired, independently of the guest's test logger.

**Size:** M
**Files:** WASM simulation oracle plan and single-process result classification in `tools/gomad3/backends/wasm` (exact owners to be confirmed against the current oracle plan and Runner failure classification), oracle tests and fixtures.

### Approach
- Read only the replayed evidence: each guest's captured stdout/stderr. Detect the `softassert` marker (`failed assertion:` prefix / `FailedAssertion` tag) and extract the static message.
- Classify the run as a soft-assertion failure with signature = static message (+ instance for clusters), even if the guest test passed, the suite set its own logger, or the assertion fired after the test ended.
- Truncated or unreadable log evidence → inconclusive for this oracle, never a pass.
- Opt-out: an explicit run option for suites that use `WithDisableTestloggerFailure`, recorded in the run identity; an opted-out run reports assertions without failing. Refuse the option where unsupported.

### Investigation targets
- `common/softassert/softassert.go` (marker and tag)
- `tests/testcore/functional_test_base.go` (current test-logger expectation), `tests/testcore/test_env.go` (`WithDisableTestloggerFailure`)
- `tools/gomad3/backends/wasm/simulation/oracles.go` (`OraclePlan`, stdout observation decoding)

### Parallel oracle source lane (2026-10-13)

Revival: MILESTONES authorizes fn179 now independently of the performance queue. Implement R1–R3 against current owners without modifying server logging/test behavior. Root integrates and reviews; task is not complete until actual acceptance/replay evidence exists.

**Initial Touches:** [tools/gomad3/backends/wasm/simulation/oracles.go, tools/gomad3/backends/wasm/simulation/oracles_test.go, tools/gomad3/backends/wasm/simulation/soft_assertion.go, tools/gomad3/backends/wasm/simulation/soft_assertion_test.go, tools/gomad3/backends/wasm/backend/runtime.go, tools/gomad3/backends/wasm/backend/soft_assertion.go, tools/gomad3/backends/wasm/backend/soft_assertion_test.go]

Start with owner/identity/option/result discovery and report exact additional paths required before implementation outside this declaration. Do not modify backend/provider.go, provider_test.go, simulation/retained.go or retained admission/canonical tests (fn153.21 owns them), or archcheck checker files (fn166 retirement owns them). Root admits shared consumer/identity paths in serialized integration order. Keep one semantic detector shared by single-process and cluster owners rather than duplicate parsing. No changes to native deferrals, server assertions, checksum work, CLI defaults or unrelated oracle semantics.
## Acceptance
- [ ] A fixture whose guest logs a soft assertion and then passes its test fails the Gomad run with a soft-assertion classification and the static message as signature (single-process and cluster).
- [ ] An assertion logged after the guest test reported success still fails the run.
- [ ] The same assertion under two seeds deduplicates to one signature; replaying the artifact reproduces it.
- [ ] Truncated log evidence yields an inconclusive oracle result, not a pass.
- [ ] The explicit opt-out records itself in the run identity and reports without failing; it is refused for unsupported workloads.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
