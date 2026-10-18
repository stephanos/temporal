---
satisfies: [R3]
---
# fn-211-apalache-backend-for-umpire-in-rust.3 ITF counterexamples decoded through Spell and replayed through the Model

## Description
**Placement (owner 2026-10-18):** `model/umpire/backends/apalache/`.

Decode Apalache's ITF JSON trace into states and actions via the canonical `Spell` form (the same decoder rules as the TLC dump), then replay the path step by step through the Model's `step`; a mismatch is witness-rejected. Write `expected/paths/apalache_<invariant>.txt` for violations.

**Files:** model/umpire/backends/apalache/src/itf.rs, replay helper in the checker or testing crate.

**Quick commands:** `cd model && UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache`
## Acceptance
- Kettle's `!beeping` negative control yields a replayed violation path golden.
- A tampered trace is witness-rejected.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
