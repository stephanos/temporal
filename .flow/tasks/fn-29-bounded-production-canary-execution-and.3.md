---
satisfies: [R3]
---
# fn-29-bounded-production-canary-execution-and.3 Implement protected canary authority and exact-scope preflight

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, conformance, and qualification interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Build R3's production-canary authority/scope boundary on the remote transport seam and prove the dedicated target and run-owned identity closure before mutation.

**Size:** M
**Files:** `tools/umpire/temporal/remote/**`, `tools/umpire/temporal/canary/authority.go`, `tools/umpire/temporal/canary/authority_test.go`, `tools/umpire/temporal/canary/target.go`, `tools/umpire/temporal/canary/target_test.go`, `tools/umpire/temporal/canary/testdata/**`
**Touches:** [tools/umpire/temporal/remote/**, tools/umpire/temporal/canary/authority.go, tools/umpire/temporal/canary/authority_test.go, tools/umpire/temporal/canary/target.go, tools/umpire/temporal/canary/target_test.go, tools/umpire/temporal/canary/testdata/**]

### Approach
- Reuse the fn-28 public TLS/client/target primitives; extract only environment-neutral seams needed to avoid staging dependencies or lifecycle duplication.
- Parse the fixed protected `ProtectedCanaryAuthority/v1` under the 1-MiB closed-field contract, retain secrets only in memory, and accept no target/credential from arguments or artifacts.
- Validate the credential-free guard's protected-default-ref and admitted-SHA context, credential lifetime, TLS/server identity, production-canary environment, registered dedicated namespace, exact Nexus route to the dedicated task queue, public capabilities, isolation/ownership attestation, and deterministic prospective run-owned identities.
- Construct only secret-free checked digests and closed trust/omission values; explicitly represent the protected assertion as operational provenance rather than independently audited fact.
- Fail before worker/start/mutation on collision, mismatch, stale authority, broadened capability, or inability to establish exact scope.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.3.md` — remote authority and target preflight contract
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.4.md` — lease/fence and target-owned delivery behavior
- `common/sdk/factory.go` — server-owned fatal/retry behavior not suitable for this bounded adapter
- `common/testing/umpire/canary/canary.go` — safety/redaction concepts only; it remains non-authoritative
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — secret-free provenance and strict limits

### Key context
The adapter can verify exact dedicated routing and run-owned identities, but it cannot prove global production inactivity. Preserve that limitation in trust/omission data and never weaken the pre-mutation gate.

### Acceptance
- [ ] Production canary authority comes only from the fixed protected environment after exact admitted-ref/SHA validation and cannot select another target.
- [ ] Every identity/routing/capability/attestation/collision failure performs no mutation or publication.
- [ ] Valid preflight returns only opaque handles and checked secret-free digests.
- [ ] Race, cancellation, TLS, path, disclosure, stale/redirect, and N/N+1 matrices pass.
## Acceptance
- [ ] R3 protected authority and exact-scope preflight are complete without duplicating the remote adapter.
- [ ] Independent tests prove no mutation and no secret disclosure on every failure row.
- [ ] Existing transport and safety comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
