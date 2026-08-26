---
satisfies: [R3, R4, R5, R6, R7, R8, R9]
---
# fn-28-authorized-remote-staging-black-box.10 Build the public-boundary harness and adversarial qualification matrix

## Description
Complete R3-R9's independent protocol proof and adversarial cross-layer verification after the production workflow boundary exists.

**Size:** M
**Files:** `model/Temporal/System/Execution/RemoteStagingTests.lean`, `model/Temporal/System/Qualification/RemoteStagingTests.lean`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/temporal/remote/**`, `tools/umpire/conformance/**`, `tools/umpire/artifact/**`, `tools/umpire/staging/**`, `tools/umpire/cmd/umpire-qualify-remote-staging/**`
**Touches:** [model/Temporal/System/Execution/RemoteStagingTests.lean, model/Temporal/System/Qualification/RemoteStagingTests.lean, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/temporal/remote/**, tools/umpire/conformance/**, tools/umpire/artifact/**, tools/umpire/staging/**, tools/umpire/cmd/umpire-qualify-remote-staging/**]

### Approach
- Build a controlled mTLS/public-gRPC integration harness that exercises the production authority, target, lease, participant, cleanup/postflight, conformance, qualification, and publication protocols without server-internal telemetry or a retained staging claim.
- Use independent literal/oracle fixtures for profile/configuration, TLS/authority/target, lease/fence, ambiguous start, target redelivery/idempotency, evidence closure, cleanup/recovery, progress, conformance, receipt/set, command, and publication behavior.
- Mutate every cross-layer identity/version/status/order/nullability/limit edge, including RawEvidence v1's 16-MiB ceiling, recovery-record tampering, stale fences, target drift, duplicate delivery, reporting-after-publication, and concurrent writers.
- Run race/fuzz/secret scans over logs, progress, recovery handling, summaries, artifacts, and diagnostics; prove the recovery record is never uploaded or admitted.
- Run focused and aggregate Lean/Go/model/regression gates and prove local/CI fixtures, prior readers, source-member bytes, and generated projections remain unchanged.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-28-authorized-remote-staging-black-box.md` — exact negative-case and version contract
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.4.md` — delivery/idempotency/cleanup matrix
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.9.md` — recovery/progress/workflow boundary
- `tools/common/artifactio/set_test.go:13-215` — publication failure/recovery matrix pattern
- `temporaltest/server.go:42-171` — controlled public client/worker harness lifecycle
- `tests/nexus_workflow_update_test.go:182-220` — public history and operation observation pattern

### Key context
The harness proves production protocol behavior against a controlled public boundary; it must label every retained result synthetic and cannot mint the protected staging environment identity or an accepted staging receipt.

### Acceptance
- [ ] The harness exercises every production seam through public TLS/gRPC interfaces with independent expected values.
- [ ] Every authority/lease/delivery/evidence/cleanup/recovery/progress/receipt/set/command/version/limit mutation has one deterministic rejection or non-success outcome.
- [ ] Secret scans and race/fuzz suites prove no raw authority/target/payload/recovery data crosses forbidden boundaries.
- [ ] Aggregate checks pass while all earlier artifacts/readers and generated regressions remain unchanged.
- [ ] No synthetic test output can be mistaken for or retained as an accepted protected-staging claim.

## Acceptance
- [ ] R3-R9 independent public-boundary and mutation verification is complete.
- [ ] Focused, race/fuzz, secret-scan, aggregate, and unchanged-regression gates pass.
- [ ] Existing test comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
