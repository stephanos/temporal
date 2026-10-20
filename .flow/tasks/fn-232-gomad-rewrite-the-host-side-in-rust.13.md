---
satisfies: [R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.13 Qualification cycle on the Rust binary and Go host retirement

Touches: tools/gomad/cmd/gomad, internal/runner, internal/evidence, internal/cli, Makefile, CI, archcheck, ARCHITECTURE.md, CLI.md, MILESTONES.md

## Description
Run one full qualification cycle with the Rust binary alone under the one-hour driver and retain the reports; then remove the Go host-side packages and binary from the default build, keep the Go overlay, models, generator and corpora, update archcheck to forbid default-set imports of removed packages, switch Make and CI to the Rust binary, and rewrite ARCHITECTURE.md's module map and CLI.md to the Rust host. Close out with the identity ledger.

## Acceptance
Qualification reports retained for every cell on the Rust binary; default build contains no Go host packages (archcheck test); CI green on the Rust binary; docs updated; harness job removed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
