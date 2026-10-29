---
satisfies: [R4, R9, R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.20 Measure and adopt a mise-pinned Linux host mold linker

## Description
Close .4's unmeasured alternate-linker follow-up using mold pinned through mise. Limit adoption to supported Linux host targets and retain it only when matched measurements show a gain.

**Size:** M
**Files:** both mise manifests/locks, `model/.cargo/config.toml`, `model/umpire/.cargo/config.toml`, existing tool/context controls and build guidance.
**Touches:** [mise.toml, mise.lock, model/umpire/mise.toml, model/umpire/mise.lock, model/.cargo/config.toml, model/umpire/.cargo/config.toml, model/umpire/dev/guest-driver/src/**, model/umpire/dev/guest-driver/tests/**, model/umpire/dev/xtask/tests/**, model/GUEST_BUILDS.md, model/design/gate-speed.md]

### Approach
- Resolve a supported pinned Linux mold asset via mise in each owning tool context. Preserve no-auto-install and fail-hard conventions; do not require an unpinned system linker installation or make non-Linux hosts provision mold. Serialize shared manifests/configuration with .19 and existing pin owners.
- Follow target-scoped Cargo linker configuration rather than global RUSTFLAGS. Prove Linux host links actually invoke the pinned mold, including any compiler-driver requirement. Existing quick/bootstrap sanitation and driver-owned encoded flags/remaps must still apply.
- Inspect `model/umpire/dev/guest-driver/src/guest.rs:159`: wasm32-wasip1/build-std must keep its own linker path. Verify host flags do not leak to WASI, native parity helper guest-profile builds, unsupported targets or non-Linux hosts. Keep framework-only execution independent of parent config/tools.
- Compare the current linker with mold on identical source/profile/features/target/toolchain and equivalent forced-relink work; report link/build wall time and host conditions, with sccache held fixed or disabled for both arms. A no-op warm Cargo run is not a linker measurement. Adopt only with measured benefit; otherwise retain the control/configuration decision and evidence without declaring a speedup.
- Keep affected execution and isolation controls here; full current-input/standalone gate renewal and combined timings remain .8. Do not resume fn-258 or .12.

### Investigation targets
**Required:**
- `model/.cargo/config.toml` and `model/umpire/.cargo/config.toml` — target-scoped settings.
- Both mise manifests/locks — platform-aware pins.
- `model/umpire/dev/guest-driver/src/guest.rs:159` — explicit WASI build.
- `model/umpire/dev/guest-driver/src/compiler.rs` — compiler and flag admission.
- `model/design/gate-speed.md` — matched evidence.

### Key context
Mold's Rust configuration uses Linux target-specific Cargo settings and a suitable compiler linker driver. Source: https://github.com/rui314/mold#how-to-use . Verify the pinned release/platform asset and current driver compatibility during implementation.

## Acceptance
- [ ] A supported Linux host build demonstrates use of the mise-pinned mold; matched baseline/mold relink measurements and adoption decision are retained. If no gain is measured, report that result and leave the existing default.
- [ ] Target/config controls and representative affected execution prove WASI guest builds and native parity/profile behavior remain correct, with no mold flags on unsupported/non-Linux target commands.
- [ ] Framework-local pins/configuration and bootstrap/quick sanitation remain valid; no global Rust flags, new unpinned system dependency or fabricated performance threshold is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
