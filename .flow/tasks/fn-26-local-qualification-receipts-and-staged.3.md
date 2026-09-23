---
satisfies: [R5, R7]
---

# fn-26-local-qualification-receipts-and-staged.3 Render canonical Evaluation Receipts and publish them immutably

## Description
Add `tools/umpire/evaluation/receipt.go`: the receipt as one Go structure rendered in a fixed key order with one trailing LF, binding the Profile name and identity, the Case identity and its Case, Program and Contract IDs, the recorded `DriverIdentity`, the Run ID, disposition and cleanup status, the Verdict status and each rule's ID, status, terminal state and supporting sequences, the decision and every reason in order, the Known Gaps by kind and code, and the Limits; its identity is the SHA-256 of the rendered bytes. `DecodeReceipt` reads one back strictly (unknown or duplicate keys, a trailing document, an oversized or version-incompatible receipt reject). Add `cli.Publish(root, name string, bytes []byte) (Publication, error)` to `tools/umpire/internal/cli`: the root resolved and contained as the proposal writer's is, the file created exclusively, an existing file with identical bytes `already-published`, one with other bytes a conflict that is reported and never overwritten. Goldens pin the receipt bytes for an accepted, a rejected and an incomplete subject.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ ./tools/umpire/internal/cli/`

**Size:** L
**Files:** `tools/umpire/evaluation/receipt.go`, `tools/umpire/evaluation/receipt_test.go`, `tools/umpire/evaluation/testdata/receipts/**`, `tools/umpire/internal/cli/publish.go`, `tools/umpire/internal/cli/publish_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] The Profile's rendered bytes and identity agree between Lean and Go, and the receipt's bytes and identity are pinned by goldens.
- [ ] Unknown, duplicate, oversized, crossed, secret-bearing, incompatible, or invalid closure rejects.
- [ ] Same subject/Profile content publishes idempotently; different Profiles yield distinct receipts.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
