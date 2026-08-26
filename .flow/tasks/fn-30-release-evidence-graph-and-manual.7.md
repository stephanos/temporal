---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8, R9]
---
# fn-30-release-evidence-graph-and-manual.7 Prove release qualification and authorization end to end

## Description
### Umpire4 reconciliation (normative)

Release evidence policy, retention/signing, human roles, revocation, workflows, and authorization are owned by a named downstream release-policy component under the standalone canary/release boundary or an existing external release platform—not by `tools/umpire`. Umpire receipts are immutable generic inputs only. The release owner consumes retained standalone-canary evidence plus external build/deployment attestations, preserves each trust class, and acquires no semantic reinterpretation or deployment authority. Replace legacy `tools/umpire/release` paths and reusable Umpire release-policy types accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Complete R1-R9 with one controlled end-to-end proof, adversarial matrices, compatibility/aggregate gates, and operator-facing documentation. Tests exercise fixed external-attestation fixtures and protected signer seams without minting a retained production authorization.

**Size:** M
**Files:** focused Lean/Go/workflow integration tests and fixtures under the new release modules, root `Makefile`, `model/Umpire/ARCHITECTURE.md`, `model/GLOSSARY.md`, `.plans/UMPIRE4_COMPONENTS.md`, operator documentation
**Touches:** [model/Umpire/Qualification/Release/**, model/Temporal/System/Qualification/**, tools/umpire/release/**, tools/umpire/cmd/umpire-release/**, tools/umpire/cmd/umpire-release-sign/**, .github/workflows/umpire-release-authorization.yml, Makefile, model/Umpire/ARCHITECTURE.md, model/GLOSSARY.md, .plans/UMPIRE4_COMPONENTS.md, docs/**]

### Approach
- Run a deterministic four-profile fixture through signed retention, fixed-lookup ReleaseEvidenceIndex signing/admission, externally issued build/deployment attestation admission, graph qualification, both protected role decisions, authorization-set publication, inspection, expiry, and single-role revocation.
- Use ephemeral test roots/keys and an explicitly non-production authorization marker; assert no fixture signer/output is accepted by the protected production trust snapshot or retained channel.
- Add table-driven invalid/held/rejected matrices for trust, candidate, evidence-index signature/role/candidate/invocation/expiry/order/cardinality/tampering, seven slots, source diversity, freshness, omissions, policy-process protocol, role ordering, gate refusal, concurrency, paths, versions, limits, secrets, and capability reachability.
- Prove canonical Lean/Go protocol agreement and unchanged v1-v5 source bytes/readers; wire focused checks plus the repository's aggregate regression gate.
- Document candidate versus ExperimentSpec, external authority attestations, evidence gaps versus invalid input, evidence versus trust, qualification versus authorization, two-role veto/gate semantics, expiry/revocation, inspection commands, and the non-deployment boundary.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:1-29` — public module-documentation convention
- `model/GLOSSARY.md` — canonical vocabulary/index convention
- `tools/common/artifactio/set_test.go` — adversarial filesystem/publication coverage
- `.plans/UMPIRE4_COMPONENTS.md` — roadmap status and deferred-boundary source
- Flow specs `fn-26` through `fn-29` — exact source proof points and aggregate gates

### Acceptance
- [ ] The controlled proof yields an inspectable qualified graph and authorization history, then an immutable revocation, with all identities/expiries independently revalidated.
- [ ] Signed explicit-gap fixtures yield held sets; absent/tampered/expired indexes, caller omissions, and malformed/untrusted members yield no set; contradictions/revocations yield rejected/non-authorizing sets, and the policy-process error matrix maps to status 1.
- [ ] No test retains or signs an accepted production authorization; capability/secret scans show no deployment path or protected material.
- [ ] Cross-language canonical protocol fixtures and prior-version compatibility fixtures pass byte-for-byte.
- [ ] Focused Lean, Go, workflow, race, and root aggregate commands pass; operator docs and glossary use the final canonical vocabulary.
- [ ] The component roadmap records the reviewed/implemented state without broadening deferred work.
## Acceptance
- [ ] The end-to-end proof and every evidence-index/invalid/held/rejected/signing/policy-process matrix pass without producing a retained production authorization.
- [ ] Prior artifact versions remain byte-compatible and closed to release-set inputs.
- [ ] Aggregate checks, docs, glossary, and roadmap are synchronized.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
