---
satisfies: [R7]
---
# fn-28-authorized-remote-staging-black-box.6 Add remote provenance and EvaluationReceipt v4 codecs

## Description
Implement R7's secret-free remote provenance and v4 receipt family without changing earlier readers.

**Size:** M
**Files:** `model/Umpire/Evaluation/**`, `model/Umpire/Artifact/Evaluation.lean`, `model/Umpire/Artifact/Tests/Evaluation.lean`, `tools/umpire/artifact/evaluation.go`, `tools/umpire/artifact/qualification_test.go`
**Touches:** [model/Umpire/Evaluation/**, model/Umpire/Artifact/Evaluation.lean, model/Umpire/Artifact/Tests/Evaluation.lean, tools/umpire/artifact/evaluation.go, tools/umpire/artifact/qualification_test.go]

### Approach
- Add exact reusable RemoteClaimAssessmentProvenance v2 fields, checked constructors, canonical Generated View, digests, limits, and cross-language codecs; concrete profile meanings remain Temporal-owned.
- Bind target delivery-attempt closure, exactly-one semantic mutation, cleanup/postflight state, and recovery/reconciliation outcome without retaining the ephemeral recovery record or progress stream.
- Add EvaluationReceipt v4 with the exact remote environment/profile/provenance binding and closed additional reason set while retaining all prior independent status, evidence, cleanup, formal, decision, and Known Gap fields.
- Keep receipt v2/v3 bytes, token/cardinality ceilings, reason tables, and readers unchanged; every reader rejects other versions.
- Implement accumulating remote reasons with rejected-over-incomplete precedence, pre-dispatch no-receipt boundaries, and exact Known Gap/secret exclusions.
- Pin equality fixtures and mutate every field, nullability, order, enum, identity preimage, digest, reason/status combination, Known Gap, and N/N+1 boundary independently.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v2 receipt, reason, identity, and limit contract
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.4.md` — byte-identical Artifact Checksum and Behavior Fingerprint parity pattern
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — canonical cross-language codec rules
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.1.md` — v4 checked vocabulary and profile values
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.3.md` — secret-free authority/target outputs

### Key context
Raw endpoint, namespace, task queue, certificate, key, header, payload, recovery record, progress event, and arbitrary remote error values are forbidden; retain only exact checked digests, closures, counts, and closed error classes.

### Acceptance
- [ ] Remote provenance and receipt v4 round-trip byte-for-byte across Lean/Go at exact limits.
- [ ] Delivery, idempotency, cleanup, postflight, and recovery outcomes are cross-checked against the admitted Run/RawEvidence closure.
- [ ] Every secret-bearing, crossed, stale, malformed, reason/status, Generated View, and N+1 mutation rejects.
- [ ] Compound failures accumulate deterministically and rejected dominates incomplete.
- [ ] V2/v3 receipt fixtures and readers remain byte-identical and reject v3.

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
