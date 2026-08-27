---
satisfies: [R3]
---
# fn-28-authorized-remote-staging-black-box.3 Implement protected authority acquisition and target preflight

## Description
Build R3's closed secret-bearing adapter boundary and prove exact target identity before any remote mutation.

**Size:** M
**Files:** `tools/umpire/temporal/remote/authority.go`, `tools/umpire/temporal/remote/authority_test.go`, `tools/umpire/temporal/remote/target.go`, `tools/umpire/temporal/remote/target_test.go`, `tools/umpire/temporal/remote/testdata/**`
**Touches:** [tools/umpire/temporal/remote/authority.go, tools/umpire/temporal/remote/authority_test.go, tools/umpire/temporal/remote/target.go, tools/umpire/temporal/remote/target_test.go, tools/umpire/temporal/remote/testdata/**]

### Approach
- Parse the single fixed protected-environment `ProtectedRemoteAuthority/v2` bundle under the 1-MiB/closed-field contract; keep raw coordinates and key material in memory and scrub or drop references on close.
- Dial a bounded public SDK/gRPC client with explicit mTLS, hostname/certificate validation, no server-internal client factory, no ambient retry, and deterministic sanitized error classes.
- Read only public system, namespace, and Nexus endpoint APIs to construct the preflight fingerprint and confirm the exact registered/isolated target and capability closure.
- Expose a narrow package API returning opaque authority/client/target handles plus secret-free digests; package-private injection supports independent TLS/public-API tests.
- Test every malformed, stale, mismatched, oversized, redirect/hostname/IP, capability, disclosure, cancellation, and no-side-effect preflight row.

### Investigation targets
**Required** (read before coding):
- `common/sdk/factory.go:24-145` — server-owned/fatal/retry behavior this adapter must not reuse
- `common/testing/umpire/canary/canary.go:19-175` — bounded authority, redaction, and cleanup concepts
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.5.md` — hermetic runner and Run Evaluation composition boundary
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — secret-free provenance and strict limit conventions
- `go.mod` — pinned public SDK/API versions

### Key context
Target selection is structurally absent from the command. The protected environment is the selector; a valid credential for a different target still fails the exact environment/fingerprint contract.

### Acceptance
- [ ] Production accepts authority only from the fixed environment bundle and no public argument or artifact.
- [ ] Every preflight mismatch fails before StartWorkflow, worker startup, or other remote mutation.
- [ ] Valid preflight returns only checked opaque handles and secret-free canonical digests.
- [ ] Race, cancellation, leak/redaction, byte/token N+1, TLS, and public-API mutation tests pass.

## Acceptance
- [ ] R3 protected authority and target preflight are complete.
- [ ] Independent tests prove no mutation and no secret disclosure on every failure row.
- [ ] Existing transport comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
