---
satisfies: [R2]
---
# fn-30-release-evidence-graph-and-manual.2 Implement the signed release trust and attestation channel

## Description
### Umpire4 reconciliation (normative)

Release evidence policy, retention/signing, human roles, revocation, workflows, and authorization are owned by a named downstream release-policy component under the standalone canary/release boundary or an existing external release platform—not by `tools/umpire`. Umpire receipts are immutable generic inputs only. The release owner consumes retained standalone-canary evidence plus external build/deployment attestations, preserves each trust class, and acquires no semantic reinterpretation or deployment authority. Replace legacy `tools/umpire/release` paths and reusable Umpire release-policy types accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Build the strict operational trust boundary for R2: canonical retention, the signed seven-slot release evidence index, externally issued build/deployment attestations, trust snapshots, evaluation contexts, and the fixed protected-signer core. Keep cryptographic verification, key acquisition, and bounded byte admission outside Lean.

**Size:** M
**Files:** focused schema/codec/signature/trust files and tests under `tools/umpire/release/`, protected signer command skeleton under `tools/umpire/cmd/umpire-release-sign/`
**Touches:** [tools/umpire/release/**, tools/umpire/cmd/umpire-release-sign/**]

### Approach
- Use Go's standard Ed25519 implementation behind small verifier/signer interfaces; sign canonical payload bytes under an explicit record/version domain and exclude only the signature field.
- Apply strict, size-bounded, one-value decoding; duplicate/unknown fields, trailing bytes, noncanonical encodings, unsupported algorithms/roles/versions, bad signatures, invalid validity order, crossed identity, and key-role ambiguity are invalid.
- Define strict `ReleaseEvidenceIndex/v1` bytes: candidate, fixed retention collection, invocation, issued/15-minute expiry, exactly seven ordered present/gap slots, closed present bindings, three closed gap reasons, signature domain, `release-evidence-index` role, and size limits.
- Distinguish checked authentic-but-stale records (held) and current-snapshot revocation facts (rejected/non-authorizing) from malformed/untrusted bytes or indexes (invalid, no graph).
- Resolve public keys and append-only revocations from one current signed trust snapshot. The retention/evidence-index/role-decision signer modes read one fixed `UMPIRE_RELEASE_SIGNING_KEY_B64` secret, derives its unique active key/role, accepts no key/role/algorithm selector, enforces lifetime caps, publishes mode-0600 output, redacts errors, and clears decoded key buffers.
- Model exact candidate, source set/receipt, target fingerprint, occupancy/run interval, workflow ref/SHA, retention identity, issuance, and expiry bindings without changing source artifacts or minting build/deployment authority claims.

### Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/artifact.go:10-40` — durable single-artifact publication primitive
- `tools/common/artifactio/set.go:16-103` — validated atomic set transaction pattern
- `tools/common/github/artifacts.go:11-61` — provider artifact transport that must remain non-authenticating
- `tools/umpire/cmd/umpire-gen-lean-dynamic-config-catalog/publish.go` — small artifactio caller pattern

**Optional** (reference as needed):
- `tools/common/artifactio/set_test.go` — interruption, symlink, and concurrent-publication test style

### Acceptance
- [ ] Canonical round trips and domain-separated verification succeed for every v1 record and reject all malformed, malleable, crossed, wrong-role/version/domain, and limit+1 fixtures.
- [ ] Known authentic expiry becomes a checked stale value, current-snapshot revocation becomes a checked rejection/non-authorizing fact, and trust-snapshot expiry or untrusted bytes yield invalid/no graph.
- [ ] Trust resolution proves role/key uniqueness and fixed root authority without accepting arbitrary algorithms, clocks, repositories, URLs, key IDs, or role selectors.
- [ ] Evidence-index codec/signer tests cover seven-slot order/cardinality, present bindings, closed gaps, candidate/invocation/retention binding, 15-minute expiry, fixed role, and tamper/omission/limit+1 rejection.
- [ ] Protected signer tests cover absent/malformed/wrong-role secrets, lifetime caps, mode-0600 publication, redaction, injected test signers, and production-trust rejection of test artifacts.
- [ ] Focused Go tests pass with race coverage for verifier/signer reuse.
## Acceptance
- [ ] Every signed-channel record has strict canonical admission and exact role/binding checks.
- [ ] Invalid, stale, and revoked states have one unambiguous graph/status partition.
- [ ] ReleaseEvidenceIndex v1 is the canonical authenticated completeness boundary.
- [ ] The fixed protected signer cannot select or disclose signing authority.
- [ ] Focused negative, race, permission, and secret tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
