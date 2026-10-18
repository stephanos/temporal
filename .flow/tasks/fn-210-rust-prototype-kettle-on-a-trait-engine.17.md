---
satisfies: [R43]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.17 Move the generic Go wasm host inside model/umpire

## Description
Owner-approved mechanical relocation: the generic Go wasm host belongs beside Umpire's wasm guest and protobuf contract. Move `model/hosts/go/` to `model/umpire/hosts/go/` after .15, before .6/.7, running alone because paths, generated-output destinations and shared build/doc files change.

**Touches:** `model/hosts/**`, `model/umpire/hosts/**`, `model/scripts/proto-go.sh`, `Makefile`, `AGENTS.md`, `model/README.md`, `model/DESIGN.md`, `model/umpire/proto/README.md`, `MIGRATE.md`

Use a history-preserving move, then update Makefile Go-host gate paths, proto-go.sh generation/freshness output paths, Go fixture/artifact relative paths and current docs/agent instructions. Keep UMPIRE_WASM/CARGO_TARGET_DIR overrides and stock direct guest builds working. Preserve Go module identity umpire.local/oracle, schema, generated Go bytes, method IDs and ABI ownership semantics. Run generator freshness; this task changes location, not schema. The public module-path change belongs to the later standalone extraction plan, not here.

Read MIGRATE.md; it is a plan only, not authority to extract, publish or push. Correct its current-path statements if needed while retaining its historical survey as dated source evidence. Do not alter archived Scala, unrelated Go tooling, protocol messages, fixtures or goldens. MILESTONES.md is conductor-owned.

**Quick commands:** `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check`; `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time`; from the new host module, `UMPIRE_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_wasm.wasm CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./...`.

R30: paste actual warm timing output; inner <=3s and full gate <=30s, fixing any breach or obtaining an owner exception. Use task-unique logs and return a clean checkpoint; no reviews, Flow lifecycle writes, pushes or unrelated history edits from the isolated implementation lane.

## Acceptance
- The host and its tracked history reside at model/umpire/hosts/go; the old model/hosts/go is absent.
- Protocol/schema/generated Go bytes, module identity, goldens and guest exports are unchanged.
- Build/generation commands and default/overridden wasm and fixture paths work at the new location; current docs and AGENTS agree.
- The actual full Rust gate, direct host tests and protobuf freshness pass; actual model-time satisfies R30.
- No standalone extraction, remote changes or Scala/cache changes occur.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
