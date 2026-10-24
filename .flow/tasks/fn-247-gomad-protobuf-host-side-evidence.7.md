---
satisfies: [R6]
---
# fn-247-gomad-protobuf-host-side-evidence.7 Regenerate changed identities and goldens once

## Description
Regenerate every golden and identity whose bytes changed in tasks 2–6, once, recording before/after identity lists and reasons in `.flow/artifacts/fn-247/identities.md` with exact revisions. This must finish before fn-246 binds its candidate.

## Acceptance
- [ ] Every regenerated golden has a recorded reason and revision; full `make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism lint-rust` and `cargo test --workspace` green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
