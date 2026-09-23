---
satisfies: [R7, R10]
---

# fn-29-bounded-production-canary-execution-and.6 Bind secret-free canary provenance to the fn-26 receipt

## Description
Add `tools/canary/assessment/provenance.go`: one `Provenance` document in a fixed key order with version `1`: the fn-26 receipt identity it accompanies, the Evaluation Profile identity, the authority class (`protected-workflow`), the workflow ref and run ID, the coordinate digests, the lease ID digest and fence, the invocation ID and iteration number, the Limits, the iteration's recorded cleanup, the invocation's cleanup outcome (`released`, or `uncertain` with the lease held; known because provenance is published after the cleanup attempt), the isolation statement, and `releaseEligibility: false`. A lost iteration has no provenance; reconcile reports it in its own report. `Render` writes it canonically with one trailing newline and its identity is the SHA-256 of the bytes; `DecodeProvenance` reads only its canonical rendering back and rejects another version, an unknown, repeated or case-folded key, a trailing document and any `releaseEligibility` other than `false`; a document over 64 KiB is refused.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/assessment/ -run Provenance`

**Files:** `tools/canary/assessment/provenance.go`, `tools/canary/assessment/provenance_test.go`, `tools/canary/assessment/testdata/**`
**Touches:** `tools/canary/assessment/provenance.go`, `tools/canary/assessment/provenance_test.go`, `tools/canary/assessment/testdata/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Its bytes and identity are pinned by goldens; it carries no credential, raw coordinate or payload.
- [ ] Every version, identity, status, relation, key and N+1 mutation is rejected by the decoder.
- [ ] `releaseEligibility` is `false` by construction and cannot be decoded or rendered otherwise.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
