---
satisfies: [R9]
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
- Changing dev debuginfo changes the guest-driver executable bytes and therefore its guest-cache namespace. Record that invalidation explicitly. Task .8 owns fresh-driver/fresh-namespace guest reproduction against committed guest bytes, including old/new driver hashes; this task does not renew the guest freshness proof by inspecting flags alone.

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
- [ ] The driver-byte/cache-namespace invalidation and .8's guest-reproduction obligation are recorded; prior guest freshness evidence is not claimed to cover the changed driver.

## Done summary
Both workspace dev profiles now use line-table debug info with checked overflow. Parsed profile comparisons show no other test/release/guest changes. Real compiler invocations and Cargo artifact profiles establish the effective settings; both xtask aliases and native parity helper profile selection are preserved.

Before/after focused native builds and alias smoke checks pass. Matched warm leaf rebuilds record actual compilation/linking and executable sizes in the committed task18 proof and model/design/gate-speed.md. Timing attribution remains inconclusive under concurrent work. No end-to-end speedup or full-gate success is claimed.

Final integrated verification on bc26d214f598a87fd03ceed51943632588288ad5 passes both native dev tool builds, 130 selected tests and strict Clippy for the five packages affected by the concurrent countdown task. The broad model gate was attempted and fails on the unchanged paused gate-budget tooling's 94 Clippy errors.

stage: impl-review - ran; SHIP from the configured claude:claude-fable-5-1:high correctness reviewer, with no blocking findings.
Tier: implementer - AGENTS.md gpt-6.1-sol at medium effort; actual worker model not reported.

The driver hash changed, selecting a new guest-cache namespace. Prior driver-bound guest freshness proof is invalidated. Task .8 owns fresh integrated-driver/guest reproduction against committed bytes and final combined/standalone proof. No guest reproduction is claimed here. Failed exploratory alias attempts remain explicitly recorded in the proof; corrected smoke checks passed. The task-owned target and raw logs are retained; no shared cache was cleaned.

The sccache, mold and residual consolidation tasks remain open. Budget admission and migration were not resumed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2c5350e3bc2964755e3dd81863217f46514fcf1d, aad703bc63cdacac1468e137a06ceb721b57c287
- Tests: (model/umpire) sanitized CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn25418-dev-debug mise exec -- cargo build --locked --profile dev -p xtask --bin xtask -vv (before prepare/matched and after prepare/matched: exit 0; actual compilation/link), (model) sanitized CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn25418-dev-debug mise exec -- cargo build --locked --profile dev -p temporal-xtask --bin temporal-xtask -vv (before prepare/matched and after prepare/matched: exit 0; actual compilation/link), (model/umpire) sanitized CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn25418-dev-debug mise exec -- cargo build --locked --profile dev -p umpire-guest-driver --bin umpire-guest-driver -vv (before/after: exit 0), (model/umpire) sanitized CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn25418-dev-debug mise exec -- cargo -vv xtask retired-names '^fn25418-proof$' < /dev/null (exit 0; alias compile/run), (model) sanitized CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn25418-dev-debug mise exec -- cargo -vv xtask gate --list (exit 0; alias compile/run; no gate execution), Cargo --message-format=json focused builds both roots (exit 0; bin profile line-tables-only, overflow_checks=true, debug_assertions=true, opt_level=0; metadata only, not compilation evidence), Parsed git HEAD vs edited workspace profile tables: exactly dev.debug added in each root; all test/release/guest settings identical (passed before commit), git diff --check (exit 0), flowctl gate classify --base 93a13e47a7097bcb7ae77b824e3d6490edda0320 (exit 1 FULL: .toml executable config; focused build verification applied), FAILED OBSERVATION: (model/umpire) cargo -vv xtask gate --list (exit 1 after successful dev compilation; inherited mise inventory undefined env.UMPIRE_TEMPORAL_ROOT), FAILED CALLER ATTEMPT: (model/umpire) cargo -vv xtask retired-names (exit 2; missing regex; corrected command above passed), Integrated bc26d214: native framework and Temporal xtask dev builds exit 0, Integrated bc26d214: 114 front/DSL/macros/consumer tests plus 16 TLA values/phased/optional tests pass, Integrated bc26d214: strict Clippy for front, DSL, macros, facade consumer and TLA passes
- PRs: