---
satisfies: [R7]
---
# fn-29-bounded-production-canary-execution-and.6 Add canary provenance and EvaluationReceipt v5 codecs

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R7's secret-free production-canary provenance and v5 receipt family without changing earlier readers.

**Size:** M
**Files:** `model/Umpire/Evaluation/**`, `model/Umpire/Artifact/Evaluation.lean`, `model/Umpire/Artifact/Tests/Evaluation.lean`, `tools/umpire/artifact/evaluation.go`, `tools/umpire/artifact/qualification_test.go`
**Touches:** [model/Umpire/Evaluation/**, model/Umpire/Artifact/Evaluation.lean, model/Umpire/Artifact/Tests/Evaluation.lean, tools/umpire/artifact/evaluation.go, tools/umpire/artifact/qualification_test.go]

### Approach
- Add exact reusable ProductionCanaryClaimAssessmentProvenance v2 checked fields, canonical Generated View, identities, limits, and cross-language codecs; concrete profile meanings remain Temporal-owned.
- Bind protected authority/workflow-context class, target/routing pre/post digests, lease/fence, exact action/fault/resource limits, delivery/idempotency, isolation attestation/trust, public evidence, cleanup/reconciliation, Known Gaps, and mandatory non-release eligibility.
- Add EvaluationReceipt v5 with exact canary environment/profile/provenance and closed reason set while preserving independent operational/evidence/semantic/cleanup/formal statuses.
- Preserve receipt v2-v4 bytes, limits, reason tables, and readers; each reader rejects other versions.
- Accumulate canary reasons with rejected-over-incomplete precedence and pin pre-dispatch no-receipt versus post-dispatch truthful-publication boundaries.
- Mutate every field, order, enum, Definition ID/Behavior Fingerprint, reason/status, Known Gap, secret exclusion, release-eligibility, and N/N+1 edge independently.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.6.md` — remote provenance/v4 receipt pattern
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — base receipt/reason contract
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — canonical cross-language codec rules
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.1.md` — v5 policy values
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.3.md` — secret-free authority/scope output

### Key context
No raw endpoint, namespace, task queue, credential, workflow actor, payload, customer/tenant data, recovery record, progress event, or arbitrary remote error may be retained. `releaseEligibility` can only be false in v4. The receipt records trust and Known Gaps but is not a self-authenticating proof of protected production origin.

### Acceptance
- [ ] Canary provenance and receipt v5 round-trip byte-for-byte across Lean/Go at exact limits.
- [ ] Isolation, delivery, mutation, cleanup, postflight, and recovery facts cross-check against admitted run/evidence closure.
- [ ] Every secret, crossed/stale binding, status/reason, release-eligibility, Generated View, and N+1 mutation rejects.
- [ ] V2-v4 fixtures/readers remain byte-identical and reject v4.
- [ ] Tests and docs never claim that schema-valid receipt bytes alone authenticate the producing workflow or target.
## Acceptance
- [ ] R7 canary provenance/receipt schemas, identities, limits, decisions, and strict codecs are complete.
- [ ] Cross-language equality and exhaustive version/status/secret matrices pass.
- [ ] Existing artifact comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
