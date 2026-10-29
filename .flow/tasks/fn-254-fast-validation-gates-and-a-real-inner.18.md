---
satisfies: [R4, R9]
---
# fn-254-fast-validation-gates-and-a-real-inner.18 Use lighter dev debuginfo in both Rust workspaces

## Description
Complete the dev-profile portion left optional by .4. Both independent workspace roots still use Cargo's default dev debuginfo; their test profiles already use line tables.

**Size:** S
**Files:** `model/Cargo.toml`, `model/umpire/Cargo.toml`, `model/design/gate-speed.md`.
**Touches:** [model/Cargo.toml, model/umpire/Cargo.toml, model/design/gate-speed.md]

### Approach
- Set `[profile.dev] debug = "line-tables-only"` in both roots, following their existing `[profile.test]` setting. Retain checked overflow and existing test settings. Do not change release/guest optimization, stripping, LTO or panic contracts.
- Verify actual effective compiler arguments with profile/flag overrides unset, including both dev-profile xtask aliases. The guest and native parity helpers use the custom guest profile inheriting release; preserve that distinction.
- Record focused matched compile/relink observations and artifact sizes where available. Separate compilation from runtime; a no-op Cargo Fresh run is not compilation evidence. Use the owner's focused-check cadence; final full gates remain .8.

### Investigation targets
**Required:**
- `model/Cargo.toml:108` and `model/umpire/Cargo.toml:123` — workspace-owned profiles.
- `model/.cargo/config.toml` and `model/umpire/.cargo/config.toml` — dev-profile xtask aliases.
- `model/design/gate-speed.md` — evidence and comparison limits.

### Key context
Cargo reads profiles from the workspace root, and config/environment overrides can mask the manifests. Source: https://doc.rust-lang.org/cargo/reference/profiles.html .

## Acceptance
- [ ] Both workspace dev builds demonstrably use line-table debuginfo with checked overflow; existing test and optimized release/guest settings are preserved.
- [ ] A focused native/xtask build in each workspace passes; guest/parity profile selection remains unchanged under the inspected effective commands.
- [ ] Matched build observations, effective flags, host/toolchain and comparison limits are recorded without an invented speedup threshold or a full-gate pass claim.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
