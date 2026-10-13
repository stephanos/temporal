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

**Shared classification/evidence Touches:** [tools/gomad3/record/softassert/softassert.go, tools/gomad3/record/softassert/softassert_test.go, tools/gomad3/backends/execution.go, tools/gomad3/backends/outcome.go, tools/gomad3/backends/outcome_test.go, tools/gomad3/record/types.go, tools/gomad3/record/identity.go, tools/gomad3/record/validation.go, tools/gomad3/backends/wasm/backend/lifecycle.go, tools/gomad3/backends/wasm/backend/lifecycle_test.go, tools/gomad3/backends/wasm/simulation/replay.go, tools/gomad3sim/oracle.go, tools/gomad3sim/oracle_test.go]
One worker owns this coherent implementation. The detector belongs below evidence/record in the final six-group tree. Preserve complete authenticated raw output and all existing non-softassert failure signatures while projecting static assertion messages/instance for softassert deduplication. Optout is an explicit recorded typed run option, not a guest environment mutation or test-argument spelling heuristic. Bind exact configuration/identity/CLI support surfaces after discovery; provider and retained paths stay with fn153.21 until its source joins.

**Typed-option propagation Touches:** [tools/gomad3/target/target.go, tools/gomad3/runner/runner.go, tools/gomad3/runner/evidence.go, tools/gomad3/runner/replay.go, tools/gomad3/backends/native/provider.go]
The explicit report-only option is supported for Go-test targets; unsupported native/exec/go-run configuration is refused before execution. Cluster support uses explicit per-boot typed workload kind sealed in configuration identity and retained evidence, never guest-argument inference. Exact CLI/configuration owners still require binding before edits. Backend provenance/preparation propagation remains a serialized follow-up after fn153.21 releases provider.go; no overlapping edits are admitted yet.
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
