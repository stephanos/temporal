---
satisfies: [R5, R6, R7]
---

# fn-26-local-qualification-receipts-and-staged.5 Expose the exact local assessment command

## Description
Add `umpire-assess run --case <case.json> --run <recorded-run.json> --profile <name> --receipt-root <dir> [--model-root <dir>]` (`tools/umpire/cmd/umpire-assess`): the Profile is one of the embedded Profiles by exact name; the model root defaults to `model` and is resolved like `umpire-replay`'s, and the receipt root must lie outside it (both resolved through symlinks). It refuses the command line before reading anything, admits, assesses, renders, decodes the rendered receipt back as a check, publishes once, and writes one JSON summary to stdout (the receipt identity, the decision, the reasons, the publication status and path). Exit codes: 0 accepted and published (or already published with identical bytes), 1 rejected, 2 incomplete, 3 a rejected subject, an unknown Profile, a publication conflict or failure, or a publication that succeeded but could not be reported, which names the ambiguity and is never retried automatically. `make umpire-assess` and `make umpire-assess-run` wrap it. Admission and assessment are pure and quick, so an interrupt matters only around publication: before the hard link nothing is published; after it the receipt stands and is reported, or its report failure is the ambiguity above. No Driver, deployment, endpoint, credential, checker, policy or retry flag.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-assess/`

**Size:** M
**Files:** `tools/umpire/cmd/umpire-assess/main.go`, `tools/umpire/cmd/umpire-assess/run.go`, `tools/umpire/cmd/umpire-assess/run_test.go`, `Makefile`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Arguments, summary/error schema, exit statuses, cancellation, and reporting are closed and deterministic.
- [ ] No execution, Driver, endpoint, credential, arbitrary checker, policy definition, retry, or network option exists.
- [ ] Publication is contained, atomic by hard link, idempotent for identical bytes, and never partial.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
