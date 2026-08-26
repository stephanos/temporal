---
satisfies: [R2, R3]
---
# fn-22-deterministic-replay-semantic.2 Classify isolated repeated baseline replays

## Description
Add the bounded replay controller behind the small replay module API. Adapt the fn-19 and fn-20 library entry points rather than their CLIs, creating a fresh run identity, destination, and isolated local authority for each attempt while retaining seed zero, attempt one, candidate input identity, phase bounds, and cleanup. Require exactly two baseline attempts and classify them as reproducible, not-reproduced, or indeterminate from the full operational/qualification/semantic gates plus the recomputed ViolationSignature. Add fake executor/conformance matrices that independently cover status precedence, signature disagreement, cleanup, cancellation, fresh identities, and fact isolation.

**Size:** M
**Files:** `tools/umpire/replay/controller.go`, `tools/umpire/replay/controller_test.go`
**Touches:** [tools/umpire/replay/controller.go, tools/umpire/replay/controller_test.go]

## Acceptance
Two succeeded/qualified/violated attempts with the baseline signature are reproducible. Qualified satisfied or different-signature resolved attempts are not-reproduced. Operational failed/incomplete, non-qualified, semantic incomplete, timeout, cancellation, cleanup uncertainty, or unavailable authority is indeterminate. Attempts use fresh run/output identities, never share facts, invoke existing runtime/conformance semantics exactly once each, and leave no environment active.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
