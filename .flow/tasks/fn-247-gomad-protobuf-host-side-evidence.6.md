---
satisfies: [R5]
---
# fn-247-gomad-protobuf-host-side-evidence.6 Build the heap profile from pprof protobuf messages

## Description
Vendor the pprof `profile.proto` (license retained) and build the heap diagnostic profile in `linked_inspection.rs` from `prost` messages instead of `proto_uint`/`proto_bytes`/`varint`. Compress with the task 2 helper.

## Acceptance
- [ ] `go tool pprof -raw` reads a generated profile in a test or recorded check; no hand-written protobuf field encoding remains in `linked_inspection.rs`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
