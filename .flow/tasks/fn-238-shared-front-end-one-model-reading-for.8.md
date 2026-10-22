---
satisfies: [R3]
---
# fn-238-shared-front-end-one-model-reading-for.8 Guest build identity hashes only the guest's dependency closure, not the whole workspace lockfile

## Description
Conductor follow-up 2026-10-22. `model/scripts/guest-rustc.py` hashes the entire `Cargo.lock` into every guest crate identity, so adding or changing any unrelated workspace crate (umpire-tlc dev-dep in fn-227.8, umpire-front in fn-238.2) changes the embedded Temporal guest bytes and forces a refresh plus slice/gaps gates. Hash only the lockfile entries in the guest's resolved dependency closure (e.g. from `cargo metadata --filter-platform wasm32-wasip1` for the guest package), preserving fn-236's host-independent reproducibility and fn-232's Linux/macOS byte identity.

Run before fn-238.3 if possible, since .3 and .4 change workspace crates again.

### Verification
The guest's focused script tests (`model/scripts/tests`), `make umpire-guest` reproducibility on two paths, a control showing that adding an unrelated workspace crate no longer changes guest bytes, and Full gates including slice/gaps (the guest bytes change once with this task). Record whether fn-232's cross-OS proof needs renewal; if so, note it rather than re-running the four-checkout certification here.

## Acceptance
- [ ] Guest identity covers only the guest's dependency closure; an unrelated workspace crate change leaves guest bytes identical (control test).
- [ ] Host-independent reproducibility controls from fn-236 still pass; guest refreshed once with slice/gaps green.
- [ ] Whether fn-232's certification needs renewal is recorded.

## Done summary
Guest crate identities hash only each crate's own lockfile dependency closure (`model/scripts/guest-lock.py` over `cargo metadata --locked`, all platforms, dev-dependencies excluded, path packages without location) instead of the whole Cargo.lock. The closure map is written once under its content hash; the guest cache key includes `guest-lock.py`; missing, ambiguous, mixed path/registry and checksum-less registry entries refuse; std crates need Cargo's std marker plus a sysroot location or std's own lockfile. Controls: two clean paths byte-identical; an unrelated workspace crate leaves both guests unchanged (the same edit at the old base changed the Temporal guest); optional-feature activation over-hashes (documented and pinned). 27 script tests. Guest refreshed once (3,332,314 bytes, sha256 0248382e...). fn-232's cross-OS certification needs renewal: filed as standalone fn-239 (blocked on a native macOS run).

Built in worktree `work/fn2388-guest-identity`, merged as cbe2c6fcb7. Review: NEEDS_WORK (cache key missed guest-lock.py; docs overstated), then SHIP (.flow/tmp/fn2388-review.md, fn2388-review2.md). Full gates on the merged tree: `UMPIRE_BACKENDS=require make model-check` 406s, `make model-time` 50.199s gate / 6.128s inner loop, slice 20s, gaps 371s.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 107285ac64, 62d210388e, 9df904d985, a9351de8cd, ec72646fa7, cbe2c6fcb7
- Tests: python3 -m unittest discover -s model/scripts/tests, UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps
- PRs: