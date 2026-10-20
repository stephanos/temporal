---
satisfies: [R4]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.5 inspect in Rust

Touches: tools/gomad/rust/gomad/src/cli/inspect.rs

## Description
Implement `inspect` over Rust-lineage artifacts with the same facts the Go inspect reports (record, identities, streams, choice summary and sites, world and simulation summaries, minimization block) and a documented `--json` shape; Go-lineage artifacts refused. Add inspect to the harness with field-semantic comparison against Go inspect on equivalent fresh executions.

## Acceptance
Harness inspect lane passes on the corpus; corrupt manifest and wrong lineage produce the existing errors and statuses; CLI.md diff lists every `--json` shape change.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
