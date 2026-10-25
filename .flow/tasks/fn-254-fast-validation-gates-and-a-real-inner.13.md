---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.13 Separate framework workspace, tools and input roots

## Description
Implement .9's standalone build boundary before .10 composes the gates. This is the workspace/tool-root portion split from the former oversized .10.

**Size:** M
**Files:** Cargo workspaces/lockfiles, framework-local configuration and pins, xtask location/root resolver, backend cache roots and path-sensitive tests/guest identity primitives.
**Touches:** [Makefile, model/Cargo.toml, model/Cargo.lock, model/umpire/**/Cargo.toml, model/umpire/Cargo.lock, model/umpire/.cargo/**, model/umpire/mise.toml, model/umpire/rust-toolchain.toml, model/umpire/rustfmt.toml, model/umpire/clippy.toml, model/umpire/deny.toml, model/umpire/dev/**, model/umpire/backends/process/**, model/umpire/backends/apalache/**, model/umpire/backends/tlc/**, model/umpire/backends/tla/tests/**, model/umpire/protocol/**, model/dev/xtask/**, model/scripts/xtask.sh, model/scripts/guest*.sh, model/scripts/proto-go.sh, model/temporal/**/Cargo.toml, model/temporal/hosts/go/umpire_temporal_guest.wasm, mise.toml, model/README.md, model/GUEST_BUILDS.md]

### Approach
- Establish the selected independent framework Cargo workspace and lockfile; explicitly exclude it from the Temporal workspace, whose crates use framework path dependencies. Preserve resolved dependency versions, lints, checked arithmetic and build/test/guest profiles. Record any unavoidable lockfile resolution change explicitly.
- Keep the intermediate checkpoint fully checked: update root Rust stages to invoke both workspaces until .10 installs delegation, and bridge existing tool/protobuf/guest entry points to the moved package. Removing framework members from the parent workspace must not silently remove their fmt/clippy/docs/tests from the current gate. Temporary bridges stay Temporal-owned.
- Move reusable xtask modules and pin data under the framework root; leave a thin Temporal-owned extension for Temporal selectors/API inputs. Framework tools/configuration/pinned generators and caches resolve inside that root or its explicitly owned target. Do not read parent configuration to admit a framework build.
- Reanchor existing crate-root-relative fixtures and explicit targets, including tests outside `tests/`. Capture and preserve the exact pre-move logical inventory and snapshots. This correctness check remains required after the owner waived fresh timing baselines. Coordinate .2 and .4 writers at a committed integration boundary; no concurrent edits of cache/process/manifests/generated assets.
- Preserve source remapping and identity admission when xtask binary and workspace lockfile inputs move (`model/dev/xtask/src/guest.rs:243-255,353-357`). Refresh affected current guest bytes through the existing builder and verify source freshness; .10 owns the final split-stage/host parity integration.

### Investigation targets
- `model/design/framework-gate.md` (.9's decision/inventory).
- `model/Cargo.toml:5-149` and root `mise.toml`.
- `model/dev/xtask/src/guest.rs:79-81,243-255,353-357` and `apalache_pins.rs`.
- `model/umpire/backends/process/src/support.rs:40-60`, path-sensitive framework test helpers and explicit test manifests.

## Acceptance
- [ ] Framework and Temporal workspace metadata/build/test closures resolve with their own lockfiles; the framework build needs no absent parent workspace/configuration. Policies, versions and exact logical tests remain accounted for.
- [ ] Reusable tools/pins/cache/root resolution live under the framework and no framework source input reaches above it. Pinned generator and jar SHA admission remain enforced; missing tools fail explicitly.
- [ ] Current guests reproduce with valid identity/freshness after root/lockfile changes; no stale bytes or cross-checkout Cargo target are used as evidence. The committed handoff names .10's remaining orchestration/proto/Host obligations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
