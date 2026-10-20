---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.13 Qualification cycle on the Rust binary and Go host retirement

Touches: tools/gomad/cmd/gomad, internal/runner, internal/evidence, internal/cli, Makefile, CI, archcheck, ARCHITECTURE.md, CLI.md, MILESTONES.md

## Description
Frozen Go baseline owner/input index: /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a/README.md. Preserve its independently copied launcher/helper, source archive and receipts/tool pins for fresh equivalence; it supplies no qualification claim.
Run one full qualification cycle with the Rust binary alone under the one-hour driver and retain the reports; then remove the Go host-side packages and binary from the default build, keep the Go overlay, models, generator and corpora, update archcheck to forbid default-set imports of removed packages, switch Make and CI to the Rust binary, and rewrite ARCHITECTURE.md's module map and CLI.md to the Rust host. Close out with the identity ledger. Before fn-228 closes, record this task's acceptance of its pending R7 final runtime/workload/replay qualification, and inventory the frozen pre-fn-228 Go executable/helper/source/preparation/configuration identities under fn-232 ownership. This task also receives pending fn-219 R5, fn-227 R4 and fn-188 identity-triggered qualification gates. Preserve their failed/interrupted/unrun Go evidence as history; no transferred gate is reported passed before this frozen Rust candidate supplies its actual reports and exits.

## Acceptance
Qualification reports retained for every cell on the Rust binary; default build contains no Go host packages (archcheck test); CI source wiring validated and mandatory local commands pass, with no CI run dispatched; docs updated; every public verb's local parity/golden lane passes before harness job removal, and goldens remain as regressions; runtime/workload/retained-replay exits and final identities recorded under the unchanged one-hour qualification limit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
