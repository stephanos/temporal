---
satisfies: [R2, R3, R8]
---
# fn-30-release-evidence-graph-and-manual.3 Bind retained environment evidence to one release candidate

## Description
Add the four source adapters and CI/staging/canary retention hooks that supply checked objects to the signed seven-slot evidence index for R2, R3, and R8. Preserve every source byte and require externally signed build/deployment authority records rather than inferring candidate occupancy.

**Size:** M
**Files:** source admission/retention files and tests under `tools/umpire/release/`; exact producer workflows delivered by fn-27 through fn-29
**Touches:** [tools/umpire/release/source*.go, tools/umpire/release/retention*.go, .github/workflows/umpire-ci-qualification.yml, .github/workflows/umpire-remote-staging-qualification.yml, .github/workflows/umpire-production-canary-qualification.yml]

### Approach
- Implement four closed adapters for the exact receipt/set versions; delegate their byte validation to the canonical artifact/conformance authorities supplied by the prerequisite specs.
- Strictly admit one current `ReleaseEvidenceIndex/v1` signed by the unique release-evidence-index role. Require its candidate/invocation/retention bindings, exact seven-slot order/cardinality, present content identities/digests/roles, and closed gap reasons before any member lookup.
- Resolve only present slots at the index's fixed retained identities. A signed gap remains held; caller omission, an absent/invalid index, tampering, digest mismatch, extra object, malformed/unsupported/untrusted member, or an unlisted object is invalid.
- Require the external build attestation to bind digest-addressed registry/build-archive facts and each deployment attestation to bind the exact candidate, target fingerprint, and occupancy interval enclosing the corresponding run and cleanup/postflight.
- Prove common semantic scope/outcome and pairwise-distinct run/set/receipt identities before graph construction; authentic absence/staleness is held, contradictions/revocations are rejected, and structural/trust failures are invalid.
- Add protected CI/staging/canary hooks that use the fixed retention signer only after immutable source publication. Local retention signing belongs to task .6; provider artifact download remains transport.

### Investigation targets
**Required** (read before coding):
- Flow spec `fn-26-local-qualification-receipts-and-staged` — local receipt/set and staged-profile contracts
- Flow spec `fn-27-hermetic-ci-execution-and-qualification` — CI receipt/set and provenance contract
- Flow spec `fn-28-authorized-remote-staging-black-box` — staging trust/target/run contract
- Flow spec `fn-29-bounded-production-canary-execution-and` — canary trust/cleanup/non-eligibility contract
- `tools/common/github/artifacts.go:19-61` — transport-only boundary

### Acceptance
- [ ] Exactly four source slots and three attestation slots come from one valid signed index; caller omission/index tampering is invalid, signed gaps/stale evidence hold, and contradictions/revocations reject.
- [ ] Staging and canary require externally signed occupancy intervals enclosing their run and cleanup/postflight; no candidate inference from a green receipt is possible.
- [ ] Local remains untrusted execution, CI remains source-bound rather than image-bound, and canary remains non-release-eligible in all projections.
- [ ] The three exact producer hooks are protected, deterministic, secret-safe, and leave original receipt/set bytes unchanged.
## Acceptance
- [ ] One signed ReleaseEvidenceIndex authenticates completeness for four source adapters and seven fixed slots.
- [ ] The three exact producer workflows sign retention only; local signing is owned by task .6.
- [ ] Crossed, duplicate, missing, stale, revoked, unsupported, and external-attestation tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
