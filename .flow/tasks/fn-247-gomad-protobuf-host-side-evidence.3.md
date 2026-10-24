---
satisfies: [R3]
---
# fn-247-gomad-protobuf-host-side-evidence.3 Move World snapshot evidence to protobuf

## Description
Define the World snapshot as length-delimited protobuf records (header, requests, events, transitions) inside the compressed envelope. Rust writes and validates with `prost`; the Go reference model (`internal/model/world`) decodes with Go protobuf. Conformance compares decoded messages instead of bytes. Keep the streaming write bound (384 MiB expanded) and per-field limits. Delete the `protocolgen` World family (`snapshotwire.*`, `wire_generated.rs`, `compressed_wire_generated.go`) and the hand-written reader in `world/retained.rs`.

Complexity amendment (owner, 2026-10-24): fn-249 retires the test-only Go reference models first, so this task generates and updates no Go protobuf code. Conformance against the Go model is replaced by the Rust corpus tests fn-249 captures.
## Acceptance
- [ ] Encoding the same model twice yields identical bytes; mutated, truncated, oversized and trailing record streams are refused by tests.
- [ ] No byte-level Go/Rust World conformance check remains; the `protocolgen` World family is deleted.
- [ ] No Go World decoder remains.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
