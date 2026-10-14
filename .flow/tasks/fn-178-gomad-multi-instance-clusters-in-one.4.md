---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.4 Generated Rust World snapshot writer from the shared schema, Go envelope round trip

## Description
Generate the Rust writer for the World snapshot wire from the same checked-in schema that generates the Go codec (R2). The Rust World can then emit the expanded `GMWSNAP1` stream that Go decodes, wraps in the `GMWORLD2` envelope and validates.

**Size:** M
**Files:** `tools/gomad/internal/backends/native/generation/protocol/protocol.go` (Rust emission for the World snapshot schema, `-check` drift), `tools/gomad/internal/model/world/schema/snapshotwire.rs.tmpl` (new template), `tools/gomad/internal/backends/wasm/wasmhost/src/world/wire_generated.rs` (generated), `tools/gomad/internal/backends/native/generation/protocol/protocol_test.go`, `tools/gomad/internal/backends/wasm/cluster/world_codec_test.go` (Go-decode round trip of Rust output)
**Touches:** [tools/gomad/internal/backends/native/generation/protocol/**, tools/gomad/internal/model/world/schema/**, tools/gomad/internal/backends/wasm/wasmhost/src/world/wire_generated.rs, tools/gomad/internal/backends/wasm/cluster/world_codec_test.go]

### Approach
- The generator `gomadtool protocol-generate` (protocol.go:541-559) already emits `internal/model/world/compressed_wire_generated.go` from `internal/model/world/schema/snapshotwire.json` plus a Go template. Add a Rust template that emits a writer only (Rust never decodes World evidence):
  - magic `GMWSNAP1`, big-endian integers, length-prefixed strings, digests
  - transitions tagged register 1, ready 2, cancel 3, quiesce 4
  - the profile bounds (depth 4, enum bytes 32)
- Extend `protocol-generate -check` (tools/gomad/Makefile:203) so a stale Rust output fails like a stale Go output.
- Round trip (corpus pattern, design record item 9): the Rust replay requires the generated writer's expanded stream for every World corpus history to byte-match Go's expanded `GMWSNAP1` wire stored in the corpus. A Go test wraps each stored expanded wire with `world.WrapExpandedSnapshot` (generated decoder, canonical check, Go encoder), requires byte equality with the Go-produced `GMWORLD2` envelope and validates it with `world.Restore`. Byte equality of the Rust stream and the stored wire makes the Go round trip hold for Rust output.
- Do not change the schema's meaning; World identity rotation is not required by this task.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/native/generation/protocol/protocol.go:520-600`
- `tools/gomad/internal/model/world/schema/` (snapshotwire.json and the Go template)
- `tools/gomad/internal/model/world/compressed.go`, `tools/gomad/internal/model/world/compressed_decode.go`
- `tools/gomad/internal/backends/wasm/cluster/world_codec_test.go`, `tools/gomad/internal/backends/wasm/cluster/world_profile_test.go`

### Key context
- fn-153.19 owns the format (done). fn-153.18 and fn-153.20 are still in progress; if they change the World profile, regenerate.

## Acceptance
- [ ] `make -C tools/gomad generate-check` (or the existing `protocol-generate -check` target) fails on a stale Rust World writer and passes when regenerated.
- [ ] For every World conformance corpus history, the Rust writer's expanded stream equals the stored Go wire (`cargo test`), and the Go-wrapped stored wire is byte-identical to the Go-produced compressed World evidence and passes `world.Restore` (`go test -run TestWorldConformanceFinalWireRoundTrips`).


## Done summary
`gomadtool protocol-generate` now also renders `internal/model/world/schema/snapshotwire.rs.tmpl` (writer only: GMWSNAP1 magic as hex literals, big-endian integers, u32-prefixed strings, u64-prefixed bytes, schema-ordered layouts and transition tags 1-4, profile depth/enum constants) into `wasmhost/src/world/wire_generated.rs` (`#[rustfmt::skip]`, consumes the fn-178.3 model accessors); `-check` fails on a stale Rust writer like the Go codec (generator test covers generation, staleness and schema-ordered field/tag emission). Shared `render` helper for Go and Rust templates. The Rust replay requires the generated writer's expanded stream to byte-match (and SHA-256 match) Go's stored wire for every World corpus history with a final snapshot (>=50, incl. the 4096-step recorded history); `TestWorldConformanceFinalWireRoundTrips` wraps each stored wire with `world.WrapExpandedSnapshot` into a byte-identical GMWORLD2 envelope and validates it with `world.Restore`. Schema meaning unchanged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c43303abc0, 331526f960
- Tests: go run ./internal/backends/native/cmd/gomadtool protocol-generate -check (exit 0), make -C tools/gomad validate (exit 0, after generation-input change), go test -tags test_dep -count=1 ./internal/backends/native/generation/protocol (exit 0), cargo +1.94.1 test --release --locked --test world_conformance (exit 0; expanded wire byte-equal for every final), go test -run TestWorldConformanceFinalWireRoundTrips ./internal/backends/wasm/cluster (exit 0), flowctl claude impl-review fn-178.4 --spec claude:claude-opus-5-5:high: SHIP (2 P3 follow-ups applied)
- PRs: