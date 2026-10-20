---
satisfies: [R4, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.5 inspect in Rust

Touches: tools/gomad/rust/gomad/src/cli/inspect.rs

## Description
Implement `inspect` over Rust-lineage artifacts with the same facts the Go inspect reports (record, identities, streams, choice summary and sites, world and simulation summaries, minimization block) and a documented `--json` shape; Go-lineage artifacts refused. Use task .3's shared authenticated pinned-root opener for every manifest/payload read, without path-based duplicate readers. Add inspect to the harness with field-semantic comparison against Go inspect on equivalent fresh executions, using task .4's declared golden view and separate raw-integrity checks.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness inspect lane passes on the corpus; corrupt manifest, wrong lineage, root substitution/symlinks, traversal, wrong permissions/links, oversized or altered payloads and digest mismatches produce the existing open refusals; legitimate target-pool module hard links remain accepted; CLI.md diff lists every `--json` shape change.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
