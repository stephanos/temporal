---
satisfies: [R1]
---
# fn-247-gomad-protobuf-host-side-evidence.1 Inventory custom binary formats and set up protobuf generation

## Description
Classify every custom binary format (GMWSNAP1/GMWORLD2, GMNETEV1, GMVOLIMG, storage evidence streams, heap profile, choicewire, timewire, net_stream_status, overlay frames) as host-only or guest-boundary, naming writers, readers, identity uses and goldens; record in `.flow/artifacts/fn-247/formats.md`. Set up `tools/gomad/internal/model/proto/`, `prost` + `protox` (pinned `=`) generation into checked-in Rust under `make generate`, Go generation with the repo's buf tooling, and `make validate` staleness checks. No format moves in this task.

Complexity amendment (owner, 2026-10-24): fn-249 retires the test-only Go reference models first, so this task generates and updates no Go protobuf code. Conformance against the Go model is replaced by the Rust corpus tests fn-249 captures.
## Acceptance
- [ ] Every format has a host-only/guest-boundary row with writers, readers, identities and goldens.
- [ ] `make -C tools/gomad generate validate` renders and checks protobuf outputs; no `protoc` binary and no `build.rs` codegen required.
- [ ] No Go protobuf generation is set up.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
