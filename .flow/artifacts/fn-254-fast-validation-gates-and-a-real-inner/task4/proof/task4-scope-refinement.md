---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.4 One test binary per crate and a cheaper test build profile (umpire crates)


## Description
Cut compile and link time (spec R4): fewer integration-test binaries, less debuginfo, and a faster linker. Limited to `model/umpire/**`; `model/temporal/**` follows in .7 after fn-246's committed closure. Consume .9's selected profile/configuration ownership and serialize workspace-manifest changes with .13.

**Size:** M
**Files:** `model/umpire/**/tests/*.rs` (moved under one `tests/main.rs` per crate), `model/Cargo.toml` (profiles), `model/.cargo/config.toml` (linker, if adopted).
**Touches:** [model/README.md, model/design/second-opinion.md, model/temporal/testing/snapshot_ownership.rs, model/umpire/reference/kettle/README.md, model/umpire/**/tests/**, model/umpire/**/Cargo.toml, model/umpire/dev/fixtures/*.rs, model/umpire/Cargo.toml, model/umpire/.cargo/config.toml, model/Cargo.toml, model/.cargo/config.toml]

### Approach
- Per crate: move `tests/<name>.rs` to `tests/main/<name>.rs` modules under one `tests/main.rs`. Keep trybuild and test files that rely on process-global state (env vars, `set_hook`, statics) as separate binaries; list each exception and why.
- Snapshot paths (insta) and fixture-relative paths must keep resolving; fix `include!`/`#[path]` users. Keep the Kettle README test-target command and the snapshot-ownership diagnostic command accurate after their target migrations; the latter is a string-only change in `model/temporal/testing/snapshot_ownership.rs`. Update the active oracle-source link in `model/design/second-opinion.md` and the DSL README test command in `model/README.md` too. These are admitted command/path-reference consequences of consolidation; historical inventories remain unchanged.
- Inventory explicit `[[test]]` targets and `autotests=false` crate manifests, including root-level `dev/fixtures/fixtures.rs` and `interpreted.rs`; do not assume every integration test is under `tests/`. Preserve dedicated gate filters and process-global isolation. Check a durable bijection of old/new package/target/name/ignored identities, including unchanged snapshots and UI-case mappings.
- Capture the exact pre-change inventory in this task; .1 now supplies historical timing references rather than new executable inventory. The timing-baseline waiver does not waive these correctness controls. Report historical compile/link attribution as unknown where unavailable, alongside actual post-change measurements.
- Profile: `[profile.test] debug = "line-tables-only"` (and `[profile.dev]` if it helps the build), keeping `overflow-checks = true` (fn-210 R11).
- Linker: try `lld` (via `-C link-arg=-fuse-ld=lld`) or `mold` if available through mise. Adopt only with a measured gain; no new system dependency without a pin.
- **Equivalence pin:** full test list, with ignored tests, identical before and after (names change only by module path; map old to new in the PR).

### Investigation targets
**Required:**
- `model/design/gate-speed.md` — compile vs run split per binary
- `model/Cargo.toml:133-148` — profiles
- `model/umpire/backends/native/tests/` — many small binaries plus shared `support/`

## Owner validation cadence (2026-10-25)

Keep affected crate execution, exact identity mapping, snapshots and scoped fmt/clippy now; unrelated unfiltered model gates wait for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] Binary count and compile/link time recorded before and after.
- [ ] Test list identical modulo the recorded path mapping; snapshots unchanged.
- [ ] Affected crate tests and relevant fmt/clippy checks pass with the checked identity mapping; unrelated full `model-check` execution is deferred to .8 under the owner cadence amendment.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
