---
satisfies: [R2]
---
# fn-216-mini-activity-vertical-slice-the-model.2 Model protocol rename and core Any messages with Go encoding

## Description
Land the protocol and host encoding portion of R2. This is the wire compatibility checkpoint;
task .5 adds generic stateful dispatch and .6 supplies the system binding.

**Size:** M
**Files:** core schema/generation, existing Rust generated imports, generic Go module and wrappers.
**Touches:** [model/umpire/proto/**, model/umpire/wasm/build.rs, model/umpire/wasm/src/**, model/umpire/wasm/tests/**, model/umpire/wasm/Cargo.toml, model/umpire/hosts/go/**, model/scripts/proto-go.sh, model/umpire/tests/fixtures/shared/**, model/Cargo.lock]

## Approach
- Before renaming, pin literal existing Explore/Rows requests, successful responses and refusals as exact encoded bytes, independent of regenerated descriptors. Keep existing method numbers, tags, refusal codes and ABI exports unchanged.
- Rename schema/package/modules from oracle to model, including the generic Go module `umpire.local/model` and generated Go import paths. Update every actual consumer found by rg; ask for a precise scope extension if the declaration above misses one. Do not retain symlink aliases.
- Add the new core Reset/Judge/Answer request/response shapes over Any, with zero-step explanations representable and distinct new method IDs. Core Answer carries opaque system reply data, not Temporal-specific completion/failure types.
- Use admitted prost/prost-build to generate well-known types with one shared owner. Do not silently add a prost-types runtime dependency. Update the existing pinned Go generator and freshness check.
- Extend generic Go typed encoding/decoding and method/response-oneof validation. Reject malformed data and mismatched response variants. Until .5, unsupported new operation dispatch remains an honest typed refusal; do not claim stateful behavior is complete here.
- Run existing Kettle native/stock-WASM/Go parity, module contract, allocation/trap/refusal and generator-freshness controls. Only descriptor metadata/import identity changes; old request/response byte controls remain identical.

## Investigation targets
**Required:**
- `model/umpire/proto/umpire/oracle/v1/oracle.proto:3` - current wire owner.
- `model/umpire/wasm/build.rs:4` - Rust generation input.
- `model/scripts/proto-go.sh:18` - pinned Go generation/freshness.
- `model/umpire/wasm/tests/dispatch.rs:70` - encoded dispatch controls.
- `model/umpire/hosts/go/host.go:273` - response validation.
- `model/umpire/hosts/go/host_test.go:117` - module/host controls.

## Quick commands
From model/, run cargo test -p umpire-wasm through mise with the shared CARGO_TARGET_DIR. Run CGO_ENABLED=0 mise exec -- go test ./... in the generic host module. From the root run make model-check and make model-time.

## Acceptance
- [ ] Existing literal wire-byte controls survive the namespace/module rename unchanged; old ABI signatures, method IDs, field tags and refusal codes are pinned.
- [ ] New generic core messages and Go wrappers round-trip valid payloads and reject malformed or method/oneof-mismatched responses; zero-step explanation is representable.
- [ ] Generated Rust and Go are fresh and agree, with one WKT owner and no new runtime library; no Temporal types enter the generic tree.
- [ ] Existing native/WASM/CGO-disabled host parity and full model gate pass; stateful behavior remains explicitly assigned to .5.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
