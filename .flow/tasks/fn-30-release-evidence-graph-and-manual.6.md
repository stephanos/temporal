---
satisfies: [R2, R6, R7, R8]
---
# fn-30-release-evidence-graph-and-manual.6 Wire protected manual authorization workflow and root UX

## Description
Expose the reviewed release boundary through root commands and one isolated, manually dispatched, multi-environment workflow for R2, R6-R8. Own local retention, signed evidence-index, and role-decision production while consuming externally signed build/deployment attestations by fixed content identity.

**Size:** M
**Files:** repository-root `Makefile`, `.github/workflows/umpire-release-authorization.yml`, workflow/static tests
**Touches:** [Makefile, .github/workflows/umpire-release-authorization.yml, tools/umpire/release/*workflow*_test.go]

### Approach
- Add root-only targets for qualify, authorize, deny, and revoke that pass only fixed artifact paths, output root, and invocation identity to the verifier/controller.
- Accept one closed operation plus candidate/evidence-index/current-set content identities; gate to the protected default ref and verify its exact SHA in a credential-free job before protected execution.
- In the release-evidence environment, sign local retention, perform all seven lookups through the repository-owned fixed retention manifest, construct present/gap slots, and invoke the evidence-index signer. It derives identities/digests/reasons/times itself, uses the unique release-evidence-index role, publishes immutably, and accepts no caller-authored slots or arbitrary URLs/repositories.
- For authorize, run release-owner and production-owner jobs under distinct gates/keys and invoke the role-decision signer after each succeeds; for role-specific deny/revoke run only the named protected role job. Gate refusal/timeout yields no decision.
- Finish in a keyless verification/publication job with pinned actions, read-only repository permissions, fixed artifacts, hard timeouts, and candidate-scoped concurrency.
- Add structural tests proving absent push/PR/schedule/release triggers, absent reusable release/deploy calls, absent registry/deployment credentials, and disjoint protected environments/keys.

### Investigation targets
**Required** (read before coding):
- `Makefile` — repository-root target and help conventions
- `.github/workflows/release.yml:1-25` — existing release capability that this workflow must not call
- `.github/workflows/promote-server-image.yml:1-25` — existing promotion capability that must remain disconnected
- `.github/workflows/umpire-model-verification.yml` — current Umpire workflow conventions

**Optional** (reference as needed):
- `tools/common/github/artifacts.go:19-61` — fixed artifact retrieval as transport only

### Acceptance
- [ ] Root Make targets invoke only closed verifier/controller modes and no model-local Makefile changes are introduced.
- [ ] The workflow's closed operations/content identities, exact protected default-ref SHA, local retention signer, fixed seven-lookups evidence-index signer, external attestation retrieval, and separate role signer jobs are structurally test-proven.
- [ ] Gate refusal/timeout produces no role decision or authorization; an explicit deny/revoke requires the selected protected role job.
- [ ] Actions, permissions, artifacts, secrets, timeouts, and concurrency are pinned/least-privilege/bounded; untrusted and final keyless jobs never receive signing material.
- [ ] Static graph/capability tests prove no trigger or call edge to release, promotion, deployment, registry-write, traffic, configuration, or rollback machinery.
## Acceptance
- [ ] Root commands and the manual workflow expose qualification/authorization without deployment authority.
- [ ] Local retention, the signed evidence index, and role decisions have explicit protected signer ownership; build/deployment attestations are fixed external inputs.
- [ ] Ref/SHA binding, gate semantics, permissions, timeouts, concurrency, and capability scans pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
