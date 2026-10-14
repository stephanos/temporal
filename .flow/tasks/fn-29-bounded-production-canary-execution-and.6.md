---
satisfies: [R7, R10]
---

# fn-29-bounded-production-canary-execution-and.6 Bind secret-free canary provenance to the fn-26 receipt

## Description
Add `tools/canary/assessment/provenance.go`: one `Provenance` document in a fixed key order with version `1`: the fn-26 receipt identity it accompanies, the Evaluation Profile identity, the authority class, copied from the policy (the decoder accepts exactly `protected-workflow` and `harness`), the workflow ref and run ID, the coordinate digests, the lease ID digest and fence, the invocation ID and iteration number, the Limits, the iteration's recorded cleanup, the invocation's cleanup outcome (`released`, or `uncertain` with the lease held; known because provenance is published after the cleanup attempt), the isolation statement, the workflow IDs the lease fenced (so they outlive the namespace's retention), and `releaseEligibility: false`. A lost iteration has no provenance; reconcile reports it in its own report. `Render` writes it canonically with one trailing newline and its identity is the SHA-256 of the bytes; `DecodeProvenance` reads only its canonical rendering back and rejects another version, an unknown, repeated or case-folded key, a trailing document and any `releaseEligibility` other than `false`; a document over 64 KiB is refused.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/assessment/ -run Provenance`

**Files:** `tools/canary/assessment/provenance.go`, `tools/canary/assessment/provenance_test.go`, `tools/canary/assessment/testdata/**`
**Touches:** `tools/canary/assessment/provenance.go`, `tools/canary/assessment/provenance_test.go`, `tools/canary/assessment/testdata/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Its bytes and identity are pinned by goldens; it carries no credential, raw coordinate or payload.
- [x] Every version, identity, status, relation, key and N+1 mutation is rejected by the decoder.
- [x] `releaseEligibility` is `false` by construction and cannot be decoded or rendered otherwise.

## Done summary
`tools/canary/assessment/provenance.go`: `Provenance` (version 1, fixed key order) names the fn-26 receipt identity, the Evaluation Profile identity, the authority class, the workflow ref and Actions run, the coordinate digests, the lease ID digest and fence, the invocation, iteration and Run, the Limits, the iteration's recorded cleanup and the invocation's cleanup outcome, the isolation statement, the fenced workflow IDs, and `releaseEligibility`, whose type renders and decodes only `false`. `RenderProvenance` writes one compact document with a trailing newline, `ProvenanceIdentity` is its SHA-256, and `DecodeProvenance` reads only the canonical rendering up to 64 KiB, refusing every version, identity, status, relation, key and N+1 mutation. Goldens pin the released and uncertain documents. Outside the task's files: `policy.Limits.Validate` and `policy.IsDigest` are exported and shared, Testpilot exports `RunIDPrefix`/`IsRunID`, the fenced Driver and the controller's fence reader refuse non-Testpilot IDs, and preflight and the policy accept only what provenance renders. Implementation review: NEEDS_WORK four times, then SHIP; P3 notes applied.
## Evidence
- Commits: 25121d10262568bbe9df5cd6ca15960fb719a74e, c6a1238716a461a2a8ce21b18fc8538986313114, 7920df968ce34c0e89c79868627f5fb0310a718e, 02b145c10f594fe2f4f5c95851157c0070faa4ae, a4300438d535c969b954849cce5cd275fa1211e3, 0c67ff61fcc53e8d8e78053a88415e811439c688, 70b97b900ca447546603b7ad6221544058e3a00e
- Tests: go test -count=1 -tags test_dep ./tools/canary/... ./common/testing/testpilot/..., go test -count=1 -tags 'test_dep integration' ./tests/ -run '^TestTestpilotCanaryLifecycle$', GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: