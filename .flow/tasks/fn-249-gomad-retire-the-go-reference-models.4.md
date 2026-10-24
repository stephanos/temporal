---
satisfies: [R3]
---
# fn-249-gomad-retire-the-go-reference-models.4 Delete the stdio helper protocol path; in-process linked API only

## Description
Delete `read_frame`/`send_frame`/callback frames, `cluster/execution/manual.rs`, the `transport*` modules and corpora, broker and storage frame I/O, `main.rs` (`reference-helper`), `conformance.rs` and both Cargo features. Replace `protocol::Execute`/`Outcome` in the `linked` API with in-process types. Re-point the helper corpora at `linked` with the same inputs and expectations.

## Acceptance
- [ ] No `read_frame`/`send_frame` call or `protocol::` wire type remains in the `linked` API; helper-corpus cases pass through `linked`; `cargo test --workspace` green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
