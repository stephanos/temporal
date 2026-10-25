---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.4 One test binary per crate and a cheaper test build profile (umpire crates)


## Description
Cut compile and link time (spec R4): fewer integration-test binaries, less debuginfo, and a faster linker. Limited to `model/umpire/**`; `model/temporal/**` follows in .7 after fn-246's committed closure. Consume .9's selected profile/configuration ownership and serialize workspace-manifest changes with .13.

**Size:** M
**Files:** `model/umpire/**/tests/*.rs` (moved under one `tests/main.rs` per crate), `model/Cargo.toml` (profiles), `model/.cargo/config.toml` (linker, if adopted).
**Touches:** [model/umpire/**/tests/**, model/umpire/**/Cargo.toml, model/umpire/dev/fixtures/*.rs, model/umpire/Cargo.toml, model/umpire/.cargo/config.toml, model/Cargo.toml, model/.cargo/config.toml]

### Approach
- Per crate: move `tests/<name>.rs` to `tests/main/<name>.rs` modules under one `tests/main.rs`. Keep trybuild and test files that rely on process-global state (env vars, `set_hook`, statics) as separate binaries; list each exception and why.
- Snapshot paths (insta) and fixture-relative paths must keep resolving; fix `include!`/`#[path]` users.
- Inventory explicit `[[test]]` targets and `autotests=false` crate manifests, including root-level `dev/fixtures/fixtures.rs` and `interpreted.rs`; do not assume every integration test is under `tests/`. Preserve dedicated gate filters and process-global isolation. Check a durable bijection of old/new package/target/name/ignored identities, including unchanged snapshots and UI-case mappings.
- Profile: `[profile.test] debug = "line-tables-only"` (and `[profile.dev]` if it helps the build), keeping `overflow-checks = true` (fn-210 R11).
- Linker: try `lld` (via `-C link-arg=-fuse-ld=lld`) or `mold` if available through mise. Adopt only with a measured gain; no new system dependency without a pin.
- **Equivalence pin:** full test list, with ignored tests, identical before and after (names change only by module path; map old to new in the PR).

### Investigation targets
**Required:**
- `model/design/gate-speed.md` — compile vs run split per binary
- `model/Cargo.toml:133-148` — profiles
- `model/umpire/backends/native/tests/` — many small binaries plus shared `support/`

## Acceptance
- [ ] Binary count and compile/link time recorded before and after.
- [ ] Test list identical modulo the recorded path mapping; snapshots unchanged.
- [ ] `make model-check` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
