---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.4 Generated Rust World snapshot writer from the shared schema, Go envelope round trip

## Description
Generate the Rust writer for the World snapshot wire from the same checked-in schema that generates the Go codec (R2). The Rust World can then emit the expanded `GMWSNAP1` stream that Go decodes, wraps in the `GMWORLD2` envelope and validates.

**Size:** M
**Files:** `tools/gomad3/backends/native/generation/protocol/protocol.go` (Rust emission for the World snapshot schema, `-check` drift), `tools/gomad3/world/schema/snapshotwire.rs.tmpl` (new template), `tools/gomad3/backends/wasm/wasmhost/src/world/wire_generated.rs` (generated), `tools/gomad3/backends/native/generation/protocol/protocol_test.go`, `tools/gomad3/backends/wasm/simulation/world_codec_test.go` (Go-decode round trip of Rust output)
**Touches:** [tools/gomad3/backends/native/generation/protocol/**, tools/gomad3/world/schema/**, tools/gomad3/backends/wasm/wasmhost/src/world/wire_generated.rs, tools/gomad3/backends/wasm/simulation/world_codec_test.go]

### Approach
- The generator `gomadtool protocol-generate` (protocol.go:541-559) already emits `world/compressed_wire_generated.go` from `world/schema/snapshotwire.json` plus a Go template. Add a Rust template that emits a writer only (Rust never decodes World evidence):
  - magic `GMWSNAP1`, big-endian integers, length-prefixed strings, digests
  - transitions tagged register 1, ready 2, cancel 3, quiesce 4
  - the profile bounds (depth 4, enum bytes 32)
- Extend `protocol-generate -check` (tools/gomad3/Makefile:203) so a stale Rust output fails like a stale Go output.
- Round trip: a Go test feeds Rust-produced expanded streams from the conformance driver's `snapshot` record into the existing generated decoder, wraps them in the `GMWORLD2` envelope with the Go encoder, and requires byte equality with the Go-produced envelope for the same World history.
- Do not change the schema's meaning; World identity rotation is not required by this task.

### Investigation targets
**Required:**
- `tools/gomad3/backends/native/generation/protocol/protocol.go:520-600`
- `tools/gomad3/world/schema/` (snapshotwire.json and the Go template)
- `tools/gomad3/world/compressed.go`, `tools/gomad3/world/compressed_decode.go`
- `tools/gomad3/backends/wasm/simulation/world_codec_test.go`, `tools/gomad3/backends/wasm/simulation/world_profile_test.go`

### Key context
- fn-153.19 owns the format (done). fn-153.18 and fn-153.20 are still in progress; if they change the World profile, regenerate.

## Acceptance
- [ ] `make -C tools/gomad3 generate-check` (or the existing `protocol-generate -check` target) fails on a stale Rust World writer and passes when regenerated.
- [ ] For the World conformance corpus histories, a Go-wrapped Rust snapshot is byte-identical to the Go-produced compressed World evidence and passes `world.Restore` validation.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
