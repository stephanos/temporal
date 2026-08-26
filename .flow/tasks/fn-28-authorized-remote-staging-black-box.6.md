---
satisfies: [R7]
---
# fn-28-authorized-remote-staging-black-box.6 Add remote provenance and QualificationReceipt v3 codecs

## Description
Implement R7's secret-free remote provenance and v3 receipt family without changing earlier readers.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Umpire/Artifact/Qualification.lean`, `model/Umpire/Artifact/Tests/Qualification.lean`, `tools/umpire/artifact/qualification.go`, `tools/umpire/artifact/qualification_test.go`
**Touches:** [model/Umpire/Qualification/**, model/Umpire/Artifact/Qualification.lean, model/Umpire/Artifact/Tests/Qualification.lean, tools/umpire/artifact/qualification.go, tools/umpire/artifact/qualification_test.go]

### Approach
- Add exact reusable RemoteQualificationProvenance v1 fields, checked constructors, canonical projection, digests, limits, and cross-language codecs; concrete profile meanings remain Temporal-owned.
- Bind target delivery-attempt closure, exactly-one semantic mutation, cleanup/postflight state, and recovery/reconciliation outcome without retaining the ephemeral recovery record or progress stream.
- Add QualificationReceipt v3 with the exact remote environment/profile/provenance binding and closed additional reason set while retaining all prior independent status, evidence, cleanup, formal, decision, and omission fields.
- Keep receipt v1/v2 bytes, token/cardinality ceilings, reason tables, and readers unchanged; every reader rejects other versions.
- Implement accumulating remote reasons with rejected-over-incomplete precedence, pre-dispatch no-receipt boundaries, and exact omission/secret exclusions.
- Pin equality fixtures and mutate every field, nullability, order, enum, identity projection, digest, reason/status combination, omission, and N/N+1 boundary independently.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v1 receipt, reason, identity, and limit contract
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.4.md` — v2 provenance/receipt evolution pattern
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — canonical cross-language codec rules
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.1.md` — v3 checked vocabulary and profile values
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.3.md` — secret-free authority/target outputs

### Key context
Raw endpoint, namespace, task queue, certificate, key, header, payload, recovery record, progress event, and arbitrary remote error values are forbidden; retain only exact checked digests, closures, counts, and closed error classes.

### Acceptance
- [ ] Remote provenance and receipt v3 round-trip byte-for-byte across Lean/Go at exact limits.
- [ ] Delivery, idempotency, cleanup, postflight, and recovery outcomes are cross-checked against the admitted Run/RawEvidence closure.
- [ ] Every secret-bearing, crossed, stale, malformed, reason/status, projection, and N+1 mutation rejects.
- [ ] Compound failures accumulate deterministically and rejected dominates incomplete.
- [ ] V1/v2 receipt fixtures and readers remain byte-identical and reject v3.

## Acceptance
- [ ] R7 remote provenance/receipt schemas, decisions, identities, limits, and strict codecs are complete.
- [ ] Cross-language equality and exhaustive version/status/secret mutation matrices pass.
- [ ] Existing artifact comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
