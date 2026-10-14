---
satisfies: [R5, R6, R7]
---

# fn-26-local-qualification-receipts-and-staged.5 Expose the exact local assessment command

## Description
Add `umpire-assess run --case <case.json> --run <recorded-run.json> --profile <name> --receipt-root <dir> [--model-root <dir>]` (`tools/umpire/cmd/umpire-assess`): the Profile is one of the embedded Profiles by exact name; the model root defaults to `model` and is resolved like `umpire-replay`'s, and the receipt root must lie outside it (both resolved through symlinks). It refuses the command line before reading anything, reads `--case` and `--run` each through a reader capped one byte past its admission cap (an oversized input is refused before it is held), admits, assesses, renders, decodes the rendered receipt back as a check, publishes once, and writes one JSON summary to stdout (the receipt identity, the decision, the reasons, the publication status and path). Exit codes: 0 accepted and published (or already published with identical bytes), 1 rejected, 2 incomplete, 3 everything else, each with its own named summary status: a rejected subject (`rejected-subject`, with its reason), an unknown Profile, an unreadable input, a catalog that does not build, `receipt-oversized`, a rendered receipt that does not decode back, a publication conflict or failure, or a publication that succeeded but could not be reported, which names the ambiguity and is never retried automatically. `make umpire-assess` and `make umpire-assess-run` wrap it. It runs under `cli.Interruptible` with a fixed one-minute timeout (a named constant, no flag), whose expiry reports as an interrupt does; admission and assessment are pure and quick, so an interrupt matters only around publication: `Publish` checks the context immediately before the hard link, so before it nothing is published (exit 3, `interrupted`); after it the receipt stands and is reported, or its report failure is the ambiguity above. A summary stdout cannot take is written to stderr, naming the ambiguity. No Driver, deployment, endpoint, credential, checker, policy or retry flag.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-assess/`

**Size:** M
**Files:** `tools/umpire/cmd/umpire-assess/main.go`, `tools/umpire/cmd/umpire-assess/run.go`, `tools/umpire/cmd/umpire-assess/run_test.go`, `Makefile`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Arguments, summary/error schema, exit statuses, cancellation, and reporting are closed and deterministic: every tooling failure listed exits 3 with its own summary status, each pinned by a test.
- [x] No execution, Driver, endpoint, credential, arbitrary checker, policy definition, retry, or network option exists.
- [x] Publication is contained, atomic by hard link, idempotent for identical bytes, and never partial.

## Done summary
`umpire-assess run --case --run --profile --receipt-root [--model-root]` (`tools/umpire/cmd/umpire-assess`): refuses the command line before reading anything (a receipt root that is missing or under the model, a positional, Driver or policy flag; an unknown Profile name with its own `unknown-profile` status), reads each input through a reader capped one byte past its admission cap, admits against the tree's static catalog, assesses, renders, re-reads the receipt, and publishes it once under `<identity>.json` inside a one-minute `cli.Interruptible` context. Exit 0 accepted, 1 rejected, 2 incomplete, 3 otherwise, each with a named summary status (`rejected-subject` with its class, `unknown-profile`, `profile-unreadable`, `unreadable-input`, `catalog-unavailable`, `receipt-oversized`, `receipt-unreadable`, `publication-conflict`, `publication-failed`, `interrupted`, `publication-unreported`, `internal-error`), each pinned by a test. `make umpire-assess` and `make umpire-assess-run` wrap it; run end to end on the control record it rejects the control against the tree's catalog. Implementation review: round one NEEDS_WORK (one P2, three P3), all applied; round two SHIP (after a transport timeout was re-dispatched), its three P3 notes applied.
## Evidence
- Commits: 013a921571a845d951108c708e53e9f3baddb3ea, a6cd1d5b84c8e8991554bf18918b75fa2749b724, 9dae4c2884f99ad74e771209baa2dc20dc72b2ac
- Tests: go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-assess/ ./tools/umpire/evaluation/, make umpire-assess-run CASE=... RUN=... PROFILE=local-ephemeral RECEIPT_ROOT=..., GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: