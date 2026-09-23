---
satisfies: [R5, R6, R7]
---

# fn-26-local-qualification-receipts-and-staged.4 Render canonical Evaluation Receipts and publish them atomically

## Description
Add `tools/umpire/evaluation/receipt.go`: `Render(subject, profile, decision) ([]byte, error)`, the receipt as one Go structure in a fixed key order with one trailing LF, binding the Profile name and identity, the Case identity and its Case, Program and Contract IDs, the Run identity, the recorded `DriverIdentity`, the Run ID, disposition and cleanup status, the Verdict status and each rule's ID, status, terminal state and supporting sequences (its evidence links), the decision and every reason in order, the trust basis and the admission caps the subject was admitted under, the Known Gaps by kind and code, and the receipt format version (`1`); its identity is the SHA-256 of the rendered bytes, and a receipt over 1 MiB is refused as the named tooling failure `receipt-oversized` (never a decision, never published), tested at N and N+1. `DecodeReceipt` reads one back strictly (unknown or duplicate keys, a trailing document, an oversized or version-incompatible receipt reject). Add `cli.Publish(ctx context.Context, root, name string, bytes []byte) (Publication, error)` to `tools/umpire/internal/cli`: the root must already exist and is resolved through its symlinks and contained as the proposal writer's is; `name` must equal `filepath.Base(name)` and be neither `.` nor `..`; the bytes written to a temporary file in the target directory, dot-prefixed and not ending in `.json` so a leftover never looks like a receipt, synced, and hard-linked to the final name with `os.Link`, which is atomic and exclusive; on an existing name, `os.Lstat` it and require a regular file (a symlink, FIFO, device or directory is a conflict), then open it with `O_NOFOLLOW|O_NONBLOCK` and re-check it is a regular file before reading it through a reader capped one byte past the receipt cap: identical bytes are `already-published`, anything else a reported conflict, never overwritten; the temporary file removed. The context is checked immediately before `os.Link`: cancelled before it, nothing is published and the temporary file is removed; after it, the receipt stands. The receipt is chmod'ed 0644 before the link. The directory is not synced after the link: a crash may lose the name but never exposes a partial receipt, and publishing again restores it. Goldens pin the receipt bytes for an accepted, a rejected and an incomplete Decision produced by `Assess`.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ ./tools/umpire/internal/cli/`

**Size:** L
**Files:** `tools/umpire/evaluation/receipt.go`, `tools/umpire/evaluation/receipt_test.go`, `tools/umpire/evaluation/testdata/receipts/**`, `tools/umpire/internal/cli/publish.go`, `tools/umpire/internal/cli/publish_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] The receipt's bytes and identity are pinned by goldens produced by `Assess`.
- [x] `DecodeReceipt` rejects an unknown or duplicate key, a trailing document, a receipt over 1 MiB, and another format version.
- [x] A context cancelled before the link publishes nothing and leaves no temporary file; one cancelled after it leaves the receipt published.
- [x] A missing root, a name with a separator, `.` or `..`, and a receipt at the cap and one byte over are handled as stated.
- [x] An existing final name that is a symlink, FIFO or directory, or a regular file over the cap or with other bytes, is a conflict and is left as it was.
- [x] Same subject/Profile content publishes idempotently; different Profiles yield distinct receipts; an interrupted publication leaves no partial receipt under its final name.

## Done summary
`tools/umpire/evaluation/receipt.go`: `Render(subject, profile, decision)` writes one receipt in a fixed key order (format version 1; the Profile's name, identity, claim and trust; the Case identity and IDs; the Run identity, Run ID, recorded Driver identity, disposition and cleanup; the Verdict with each rule's status, terminal state and supporting sequences; the decision, its reasons and unsupported rules; the Known Gaps by kind and code; the admission caps), refusing a Decision made under another Profile or on another subject; over 1 MiB it is the tooling failure `receipt-oversized`. `DecodeReceipt` reads only the canonical rendering of format version 1 back. `ReceiptIdentity` is the bare hex SHA-256. Goldens for an accepted, a rejected and an incomplete Decision produced by `Assess` are pinned under `testdata/receipts/` (`UMPIRE_RECEIPT_GOLDENS=write` rewrites them). `cli.Publish(ctx, root, name, bytes)` writes a synced 0644 dot-prefixed temporary file and hard-links it to the final name; the same bytes are `already-published`, and anything else under the name (other bytes, a symlink, FIFO or directory) is a `ConflictError`, never overwritten; the context is checked immediately before the link. Implementation review: SHIP in one round; its P3 and two FYIs applied.
## Evidence
- Commits: 941f2a73e1fee017c996c0574deb59f73e9a8c89, 3b5c8ca93b6839bbea73c124d69330ebd46a076b
- Tests: go test -count=1 -tags test_dep ./tools/umpire/evaluation/ ./tools/umpire/internal/cli/, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: