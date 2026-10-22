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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
