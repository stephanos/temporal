---
satisfies: [R5, R6, R7]
---

# fn-26-local-qualification-receipts-and-staged.4 Render canonical Evaluation Receipts and publish them atomically

## Description
Add `tools/umpire/evaluation/receipt.go`: `Render(subject, profile, decision) ([]byte, error)`, the receipt as one Go structure in a fixed key order with one trailing LF, binding the Profile name and identity, the Case identity and its Case, Program and Contract IDs, the Run identity, the recorded `DriverIdentity`, the Run ID, disposition and cleanup status, the Verdict status and each rule's ID, status, terminal state and supporting sequences (its evidence links), the decision and every reason in order, the trust basis and the admission caps the subject was admitted under, the Known Gaps by kind and code, and the receipt format version (`1`); its identity is the SHA-256 of the rendered bytes, and a receipt over 1 MiB is refused as the named tooling failure `receipt-oversized` (never a decision, never published), tested at N and N+1. `DecodeReceipt` reads one back strictly (unknown or duplicate keys, a trailing document, an oversized or version-incompatible receipt reject). Add `cli.Publish(root, name string, bytes []byte) (Publication, error)` to `tools/umpire/internal/cli`: the root must already exist and is resolved through its symlinks and contained as the proposal writer's is; `name` must equal `filepath.Base(name)` and be neither `.` nor `..`; the bytes written to a temporary file in the target directory, dot-prefixed and not ending in `.json` so a leftover never looks like a receipt, synced, and hard-linked to the final name with `os.Link`, which is atomic and exclusive; on an existing name, `os.Lstat` it and require a regular file (a symlink, FIFO, device or directory is a conflict), then open it with `O_NOFOLLOW|O_NONBLOCK` and re-check it is a regular file before reading it through a reader capped one byte past the receipt cap: identical bytes are `already-published`, anything else a reported conflict, never overwritten; the temporary file removed. The directory is not synced after the link: a crash may lose the name but never exposes a partial receipt, and publishing again restores it. Goldens pin the receipt bytes for an accepted, a rejected and an incomplete Decision produced by `Assess`.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ ./tools/umpire/internal/cli/`

**Size:** L
**Files:** `tools/umpire/evaluation/receipt.go`, `tools/umpire/evaluation/receipt_test.go`, `tools/umpire/evaluation/testdata/receipts/**`, `tools/umpire/internal/cli/publish.go`, `tools/umpire/internal/cli/publish_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] The receipt's bytes and identity are pinned by goldens produced by `Assess`.
- [ ] `DecodeReceipt` rejects an unknown or duplicate key, a trailing document, a receipt over 1 MiB, and another format version.
- [ ] A missing root, a name with a separator, `.` or `..`, and a receipt at the cap and one byte over are handled as stated.
- [ ] An existing final name that is a symlink, FIFO or directory, or a regular file over the cap or with other bytes, is a conflict and is left as it was.
- [ ] Same subject/Profile content publishes idempotently; different Profiles yield distinct receipts; an interrupted publication leaves no partial receipt under its final name.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
