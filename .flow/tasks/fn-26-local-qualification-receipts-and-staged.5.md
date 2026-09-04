---
satisfies: [R5, R6, R7]
---

# fn-26-local-qualification-receipts-and-staged.5 Expose the exact local assessment command
## Description
Expose one thin offline local-assessment command and root Make target accepting only the canonical subject, fixed Profile name, and output root. Validate the complete receipt before one atomic immutable publication and report claim, tooling, and post-publication ambiguity separately.

**Size:** M
**Touches:** `tools/umpire/cmd/umpire-assess-local/**`, `Makefile`

## Acceptance
- [ ] Arguments, summary/error schema, exit statuses, cancellation, and reporting are closed and deterministic.
- [ ] No execution, Host, endpoint, credential, arbitrary checker, policy definition, retry, or network option exists.
- [ ] Publication is contained, lock-guarded, idempotent for identical bytes, and never partial.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
