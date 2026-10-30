---
satisfies: [R9, R7]
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
- Distinguish WASI target units from host build scripts/proc macros compiled during that same guest build: Cargo may apply the host target's linker to those host units. Prove they link under the pinned driver/mold environment, and record any resulting host-variant sidecar identity/cache invalidation. A clean WASI-unit command alone does not prove guest-build isolation.
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
Adopted mise-pinned mold3.0.0 for supported Linux GNU host targets in both workspaces, preserving guest linking and ordinary profiles. Matched forced links on Linux/aarch64 measured median GNU ld0.230219s versus adapter0.081408s, including mise resolution. No full-build or x86_64/Darwin speedup claim.

39 guest-driver tests, five Temporal Make integration tests, standalone config execution, real WASI host units, unchanged stock guest bytes/sidecars, current driver/bootstrap and native/WASM Go parity pass. Root Make config discovery was fixed with explicit --config while preserving caller-relative paths: actual Make/nested/Make steps recompiled43 units each before and zero after. Final reviewer-requested canonical test root passes a Linux symlink-TMPDIR red/green regression plus focused fmt/strict Clippy. Original proof remains historical; make-entry-proof, symlink-fix-proof and source-envelope bind final corrections.

stage: impl-review - ran (model: claude-fable-5-1 at high); first correctness/contracts/integration round NEEDS_WORK with one retained P1, then single follow-up SHIP. P1 fixed and second-review P2 test-path correction applied exactly as suggested with focused red/green evidence, no third review under fix-pass policy. Optional diagnostic FYIs not required for behavior remain in raw reviews.
stage: completion-review - skipped(policy: unfinished spec tasks).
Gates: focused controls and changed-target strict lint/format pass. Broader Temporal all-targets Clippy still fails unchanged gate.rs too_many_lines; full model-check/model-validate/current-input standalone and Temporal guest renewal remain fn-254.8. Budgets .12 remain paused and .19 cache reuse remains blocked.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 75b4ee542c, e6919f360a, 827b76300d
- Tests: model/umpire: cargo test --locked -p umpire-guest-driver (5 unit +34 integration passed); strict focused Clippy and fmt pass, Actual WASI stock baseline/restored same-namespace build:39 WASI rustc and63 host-linker invocations checked;20 host outputs distinguish linkers, unchanged guest SHA and five sidecar/host variants, Standalone copied configuration executable and both native bootstraps pass; two CGO-disabled Go export/current-contract parity tests pass, Matched captured-object forced relinks:6 observations per arm plus warmups, median0.230219s GNU ld and0.081408s pinned adapter, Actual Make/nested/Make config regression:43/43/43 rustc/config-dirty before,0/0/0 after; five Temporal integration tests pass, Linux private symlink TMPDIR exact new control: exit101 before canonicalization,1/1 pass after; changed-target Clippy/fmt pass, Known unrelated failure retained: Temporal --all-targets Clippy rejects unchanged model/dev/xtask/src/gate.rs too_many_lines
- PRs: