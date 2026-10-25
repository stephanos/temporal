---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.13 Separate framework workspace, tools and input roots

## Description
Implement .9's standalone build boundary before .10 composes the gates. This is the workspace/tool-root portion split from the former oversized .10.

**Size:** M
**Files:** Cargo workspaces/lockfiles, framework-local configuration and pins, xtask location/root resolver, backend cache roots and path-sensitive tests/guest identity primitives.
**Touches:** [model/design/framework-gate.md, model/umpire/scripts/**, model/scripts/**, model/umpire/author/dsl/tests/readme.rs, model/umpire/hosts/go/**, model/umpire/reference/kettle/**, model/umpire/**/README.md, AGENTS.md, .plans/UMPIRE_MODULES.md, Makefile, model/Cargo.toml, model/Cargo.lock, model/umpire/**/Cargo.toml, model/umpire/Cargo.lock, model/umpire/.cargo/**, model/umpire/mise.toml, model/umpire/rust-toolchain.toml, model/umpire/rustfmt.toml, model/umpire/clippy.toml, model/umpire/deny.toml, model/umpire/dev/**, model/umpire/backends/process/**, model/umpire/backends/apalache/**, model/umpire/backends/tlc/**, model/umpire/backends/tla/tests/**, model/umpire/protocol/**, model/dev/xtask/**, model/scripts/xtask.sh, model/scripts/guest*.sh, model/scripts/proto-go.sh, model/temporal/**/Cargo.toml, model/temporal/hosts/go/umpire_temporal_guest.wasm, mise.toml, model/README.md, model/GUEST_BUILDS.md]

### Approach
- Establish the selected independent framework Cargo workspace and lockfile; explicitly exclude it from the Temporal workspace, whose crates use framework path dependencies. Preserve resolved dependency versions, lints, checked arithmetic and build/test/guest profiles. Record any unavoidable lockfile resolution change explicitly.
- Keep the intermediate checkpoint fully checked: update root Rust stages to invoke both workspaces until .10 installs delegation, and bridge existing tool/protobuf/guest entry points to the moved package. Removing framework members from the parent workspace must not silently remove their fmt/clippy/docs/tests from the current gate. Temporary bridges stay Temporal-owned.
- Move reusable xtask modules and pin data under the framework root; leave a thin Temporal-owned extension for Temporal selectors/API inputs. Framework tools/configuration/pinned generators and caches resolve inside that root or its explicitly owned target. Do not read parent configuration to admit a framework build.
- Fix the three confirmed native-tool couplings while moving them. Put guest orchestration, compiler wrapping, lock admission and their support in the dedicated `umpire-guest-driver` Cargo package/executable at `model/umpire/dev/guest-driver`. Keep installer, retired-name and future gate commands in `model/umpire/dev/xtask`. The driver has no dependency on the general CLI/gate implementation. The general CLI may use the driver library's admitted context utilities, but guest operations execute the dedicated driver, so `current_exe`/wrapper publication and guest cache identity refer to that driver, never the general CLI. Merely moving Rust modules inside the same executable does not satisfy this contract.
- One shared admitted context in the guest-driver library owns framework tool/pin roots, caller compilation workspace, caller-relative output/target paths, sysroot/Cargo roots and canonical longest-root physical-to-logical remaps. Its raw-OS environment encoder/decoder serves orchestration and compiler wrapping; flags and identity normalization consume the same remap list. Resolve roots once and pass context into commands, including installer callers, rather than repeating ancestor discovery or parent-relative configuration lookup. Keep compiler-mode dispatch private to the native driver and preserve existing wrapper/loader/jobserver behavior; add no shell compiler shim or generic launch framework.
- Preserve the exact guest-driver executable hash, current full caller lockfile hash and compiler input in guest cache invalidation, plus existing per-package portable sidecar/lock identities. Add focused controls showing a changed driver/lock/compiler invalidates and an unrelated general-CLI executable change leaves the driver/key stable. .10 renews independence when gate code is added. Reuse required guest reproduction runs for these controls where inputs match, without a new baseline campaign.
- Reanchor existing crate-root-relative fixtures and explicit targets, including tests outside `tests/`. Capture and preserve the exact pre-move logical inventory and snapshots, mapping moved package/target/name identities exactly once and listing the newly required controls separately from that preserved baseline. This correctness check remains required after the owner waived fresh timing baselines. Coordinate .2 and .4 writers at a committed integration boundary; no concurrent edits of cache/process/manifests/generated assets.
- Implement the selected layout and complete coupling inventory in `model/design/framework-gate.md`. Framework scripts, the DSL README consumer, Go default-target paths and moved-tool documentation are admitted owners as well as the workspace/backend paths. Keep root AGENTS/module-map guidance accurate at this integration boundary, using writing-for-agents for agent instructions; root alone owns MILESTONES. The exact gate-policy data exception is selected, not a broad permission to move Temporal knowledge into framework source.
- Preserve source remapping and identity admission when xtask binary and workspace lockfile inputs move (`model/dev/xtask/src/guest.rs:243-255,353-357`). Refresh affected current guest bytes through the existing builder and verify source freshness; .10 owns the final split-stage/host parity integration.

### Investigation targets
- `model/design/framework-gate.md` (.9's decision/inventory).
- `model/Cargo.toml:5-149` and root `mise.toml`.
- `model/dev/xtask/src/guest.rs:79-81,243-255,353-357` and `apalache_pins.rs`.
- `model/umpire/backends/process/src/support.rs:40-60`, path-sensitive framework test helpers and explicit test manifests.

## Owner validation cadence (2026-10-25)

Keep both locked metadata/build closures, test compilation and affected root/tool/generator/guest identity/freshness/parity controls now. Unrelated full workspace execution and combined model gates wait for .8; the intermediate root gate must still account for both workspaces. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] The guest driver builds independently of general CLI/gate implementation; guest caches hash that driver. One admitted tool/workspace context and one remap/environment codec serve all consumers, with focused independence/invalidation/raw-OS/root-refusal controls.
- [ ] Framework and Temporal workspace metadata/build/test closures resolve with their own lockfiles; the framework build needs no absent parent workspace/configuration. Policies, versions and exact logical tests remain accounted for.
- [ ] Reusable tools/pins/cache/root resolution live under the framework and no framework source input reaches above it. Pinned generator and jar SHA admission remain enforced; missing tools fail explicitly.
- [ ] Current guests reproduce with valid identity/freshness after root/lockfile changes; no stale bytes or cross-checkout Cargo target are used as evidence. The committed handoff names .10's remaining orchestration/proto/Host obligations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
